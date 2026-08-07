package resolvers

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/miekg/dns"
	"github.com/mr-karan/doggo/pkg/models"
	"github.com/quic-go/quic-go/http3"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}

type closeErrorBody struct {
	io.Reader
	closed bool
}

func (b *closeErrorBody) Close() error {
	b.closed = true
	return errors.New("close failed")
}

type countingBody struct {
	read   int
	closed bool
}

func (b *countingBody) Read(p []byte) (int, error) {
	for i := range p {
		p[i] = 'x'
	}
	b.read += len(p)
	return len(p), nil
}

func (b *countingBody) Close() error {
	b.closed = true
	return nil
}

func startHTTP3TestServer(t *testing.T, handler http.Handler) string {
	t.Helper()

	// Reuse Go's test certificate generation, then serve it on a UDP socket.
	bootstrap := httptest.NewTLSServer(http.NotFoundHandler())
	tlsConfig := bootstrap.TLS.Clone()
	bootstrap.Close()

	conn, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0})
	if err != nil {
		t.Fatalf("ListenUDP: %v", err)
	}
	server := &http3.Server{Handler: handler, TLSConfig: tlsConfig}
	serveErr := make(chan error, 1)
	go func() { serveErr <- server.Serve(conn) }()

	t.Cleanup(func() {
		_ = server.Close()
		_ = conn.Close()
		select {
		case err := <-serveErr:
			if err != nil && !errors.Is(err, http.ErrServerClosed) && !errors.Is(err, net.ErrClosed) {
				t.Errorf("HTTP/3 server stopped with error: %v", err)
			}
		case <-time.After(2 * time.Second):
			t.Error("timed out waiting for HTTP/3 server shutdown")
		}
	})

	return "https://" + conn.LocalAddr().String() + "/dns-query"
}

func dnsResponse(w http.ResponseWriter, wire []byte) error {
	var request dns.Msg
	if err := request.Unpack(wire); err != nil {
		return err
	}

	response := new(dns.Msg)
	response.SetReply(&request)
	for _, question := range request.Question {
		if question.Qtype != dns.TypeA {
			continue
		}
		rr, err := dns.NewRR(fmt.Sprintf("%s 60 IN A 192.0.2.188", question.Name))
		if err != nil {
			return err
		}
		response.Answer = append(response.Answer, rr)
	}

	packed, err := response.Pack()
	if err != nil {
		return err
	}
	w.Header().Set("Content-Type", "application/dns-message")
	w.WriteHeader(http.StatusOK)
	_, err = w.Write(packed)
	return err
}

func testDOHLookup(t *testing.T, resolver Resolver) []Response {
	t.Helper()
	question := dns.Question{Name: "example.test.", Qtype: dns.TypeA, Qclass: dns.ClassINET}
	responses, err := resolver.Lookup(context.Background(), []dns.Question{question}, QueryFlags{RD: true})
	if err != nil {
		t.Fatalf("Lookup: %v", err)
	}
	if len(responses) != 1 || len(responses[0].Answers) != 1 {
		t.Fatalf("responses = %#v, want one A answer", responses)
	}
	if got := responses[0].Answers[0].Address; got != "192.0.2.188" {
		t.Fatalf("answer = %q, want 192.0.2.188", got)
	}
	return responses
}

func TestDOHHTTP3POSTUsesHTTP3(t *testing.T) {
	var protocol, method atomic.Int32
	serverURL := startHTTP3TestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		protocol.Store(int32(r.ProtoMajor))
		if r.Method == http.MethodPost {
			method.Store(1)
		}
		if r.Header.Get("Content-Type") != "application/dns-message" {
			http.Error(w, "missing DNS content type", http.StatusBadRequest)
			return
		}
		wire, err := io.ReadAll(r.Body)
		if err != nil || dnsResponse(w, wire) != nil {
			http.Error(w, "invalid DNS request", http.StatusBadRequest)
		}
	}))

	resolver, err := NewDOHResolver(serverURL, Options{
		Logger:             discardLogger(),
		Timeout:            2 * time.Second,
		UseHTTP3:           true,
		InsecureSkipVerify: true,
		TLSHostname:        "resolver.test",
	})
	if err != nil {
		t.Fatalf("NewDOHResolver: %v", err)
	}
	doh := resolver.(*DOHResolver)
	t.Cleanup(func() { _ = doh.Close() })

	transport, ok := doh.client.Transport.(*http3.Transport)
	if !ok {
		t.Fatalf("transport = %T, want *http3.Transport", doh.client.Transport)
	}
	if doh.client.Timeout != 2*time.Second {
		t.Fatalf("client timeout = %v, want 2s", doh.client.Timeout)
	}
	if !transport.TLSClientConfig.InsecureSkipVerify || transport.TLSClientConfig.ServerName != "resolver.test" {
		t.Fatalf("TLS config = %#v, want skip verification and resolver.test", transport.TLSClientConfig)
	}

	testDOHLookup(t, resolver)
	if protocol.Load() != 3 {
		t.Fatalf("HTTP protocol major = %d, want 3", protocol.Load())
	}
	if method.Load() != 1 {
		t.Fatal("HTTP/3 server did not receive a POST request")
	}
}

func TestDOHHTTP3GETFallback(t *testing.T) {
	var (
		methodsMu sync.Mutex
		methods   []string
	)
	serverURL := startHTTP3TestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		methodsMu.Lock()
		methods = append(methods, r.Method)
		methodsMu.Unlock()
		if r.ProtoMajor != 3 {
			http.Error(w, "HTTP/3 required", http.StatusHTTPVersionNotSupported)
			return
		}
		if r.Method == http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		encoded := r.URL.Query().Get("dns")
		wire, err := base64.RawURLEncoding.DecodeString(encoded)
		if err != nil || dnsResponse(w, wire) != nil {
			http.Error(w, "invalid DNS query", http.StatusBadRequest)
		}
	}))

	resolver, err := NewDOHResolver(serverURL, Options{
		Logger:             discardLogger(),
		Timeout:            2 * time.Second,
		UseHTTP3:           true,
		InsecureSkipVerify: true,
	})
	if err != nil {
		t.Fatalf("NewDOHResolver: %v", err)
	}
	t.Cleanup(func() { _ = resolver.(*DOHResolver).Close() })

	testDOHLookup(t, resolver)
	methodsMu.Lock()
	got := strings.Join(methods, ",")
	methodsMu.Unlock()
	if got != "POST,GET" {
		t.Fatalf("methods = %q, want POST,GET", got)
	}
}

func TestDOHHTTP3HonorsRequestContext(t *testing.T) {
	var startedOnce sync.Once
	started := make(chan struct{})
	serverURL := startHTTP3TestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		startedOnce.Do(func() { close(started) })
		<-r.Context().Done()
	}))

	resolver, err := NewDOHResolver(serverURL, Options{
		Logger:             discardLogger(),
		Timeout:            2 * time.Second,
		UseHTTP3:           true,
		InsecureSkipVerify: true,
	})
	if err != nil {
		t.Fatalf("NewDOHResolver: %v", err)
	}
	t.Cleanup(func() { _ = resolver.(*DOHResolver).Close() })

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	question := dns.Question{Name: "example.test.", Qtype: dns.TypeA, Qclass: dns.ClassINET}
	_, err = resolver.Lookup(ctx, []dns.Question{question}, QueryFlags{RD: true})
	if err == nil || !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("Lookup error = %v, want context deadline exceeded", err)
	}
	select {
	case <-started:
	default:
		t.Fatal("HTTP/3 request never reached the server")
	}
}

func TestDOHHTTP3DoesNotFallbackToHTTP2(t *testing.T) {
	var requests atomic.Int32
	tcpServer := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		wire, err := io.ReadAll(r.Body)
		if err != nil || dnsResponse(w, wire) != nil {
			http.Error(w, "invalid DNS request", http.StatusBadRequest)
		}
	}))
	defer tcpServer.Close()
	serverURL, err := url.JoinPath(tcpServer.URL, "dns-query")
	if err != nil {
		t.Fatalf("JoinPath: %v", err)
	}

	h3Resolver, err := NewDOHResolver(serverURL, Options{
		Logger:             discardLogger(),
		Timeout:            150 * time.Millisecond,
		UseHTTP3:           true,
		InsecureSkipVerify: true,
	})
	if err != nil {
		t.Fatalf("NewDOHResolver(HTTP/3): %v", err)
	}
	question := dns.Question{Name: "example.test.", Qtype: dns.TypeA, Qclass: dns.ClassINET}
	started := time.Now()
	_, err = h3Resolver.Lookup(context.Background(), []dns.Question{question}, QueryFlags{RD: true})
	elapsed := time.Since(started)
	_ = h3Resolver.(*DOHResolver).Close()
	if err == nil {
		t.Fatal("HTTP/3 lookup unexpectedly succeeded against an HTTP/1.1 server")
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("HTTP/3 lookup error = %v, want client timeout", err)
	}
	if elapsed > time.Second {
		t.Fatalf("HTTP/3 client timeout took %v, want approximately 150ms", elapsed)
	}
	if requests.Load() != 0 {
		t.Fatalf("HTTP server received %d requests, want 0 (no protocol fallback)", requests.Load())
	}

	normalResolver, err := NewDOHResolver(serverURL, Options{
		Logger:             discardLogger(),
		Timeout:            2 * time.Second,
		InsecureSkipVerify: true,
	})
	if err != nil {
		t.Fatalf("NewDOHResolver(normal): %v", err)
	}
	t.Cleanup(func() { _ = normalResolver.(*DOHResolver).Close() })
	testDOHLookup(t, normalResolver)
	if requests.Load() != 1 {
		t.Fatalf("normal DoH server requests = %d, want 1", requests.Load())
	}
}

func TestHTTP3RequiresDOHNameserver(t *testing.T) {
	_, err := LoadResolvers(Options{
		Logger:   discardLogger(),
		UseHTTP3: true,
		Nameservers: []models.Nameserver{{
			Type:    models.UDPResolver,
			Address: "127.0.0.1:53",
		}},
	})
	if err == nil || !strings.Contains(err.Error(), "HTTP/3 requires at least one HTTPS (DoH) nameserver") {
		t.Fatalf("LoadResolvers error = %v, want clear HTTP/3/DoH error", err)
	}
}

func TestLoadResolversClosesHTTP3TransportOnPartialFailure(t *testing.T) {
	serverURL := startHTTP3TestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		wire, err := io.ReadAll(r.Body)
		if err != nil || dnsResponse(w, wire) != nil {
			http.Error(w, "invalid DNS request", http.StatusBadRequest)
		}
	}))

	resolver, err := NewDOHResolver(serverURL, Options{
		Logger:             discardLogger(),
		Timeout:            2 * time.Second,
		UseHTTP3:           true,
		InsecureSkipVerify: true,
	})
	if err != nil {
		t.Fatalf("NewDOHResolver: %v", err)
	}
	doh := resolver.(*DOHResolver)
	t.Cleanup(func() { _ = doh.Close() })
	testDOHLookup(t, doh) // Ensure the transport owns a live UDP socket and QUIC connection.

	var closed atomic.Bool
	closeTransport := doh.closeTransport
	doh.closeTransport = func() error {
		closed.Store(true)
		return closeTransport()
	}

	errConstruction := errors.New("later resolver failed")
	call := 0
	factory := func(models.Nameserver, Options) (Resolver, error) {
		call++
		if call == 1 {
			return doh, nil
		}
		return nil, errConstruction
	}

	loaded, err := loadResolvers(Options{
		Logger:   discardLogger(),
		UseHTTP3: true,
		Nameservers: []models.Nameserver{
			{Type: models.DOHResolver, Address: serverURL},
			{Type: models.DOHResolver, Address: "https://invalid.test/dns-query"},
		},
	}, factory)
	if !errors.Is(err, errConstruction) {
		t.Fatalf("loadResolvers error = %v, want construction error", err)
	}
	if loaded != nil {
		t.Fatalf("loadResolvers returned partial resolvers: %#v", loaded)
	}
	if !closed.Load() {
		t.Fatal("HTTP/3 transport was not closed after partial construction failure")
	}
}

func TestDOHKeepsValidResponseWhenBodyCloseFails(t *testing.T) {
	var responseBody *closeErrorBody
	transport := roundTripFunc(func(request *http.Request) (*http.Response, error) {
		wire, err := io.ReadAll(request.Body)
		if err != nil {
			return nil, err
		}
		var query dns.Msg
		if err := query.Unpack(wire); err != nil {
			return nil, err
		}
		response := new(dns.Msg)
		response.SetReply(&query)
		rr, err := dns.NewRR(fmt.Sprintf("%s 60 IN A 192.0.2.188", query.Question[0].Name))
		if err != nil {
			return nil, err
		}
		response.Answer = append(response.Answer, rr)
		packed, err := response.Pack()
		if err != nil {
			return nil, err
		}
		responseBody = &closeErrorBody{Reader: bytes.NewReader(packed)}
		return &http.Response{
			StatusCode: http.StatusOK,
			Status:     "200 OK",
			Header:     make(http.Header),
			Body:       responseBody,
		}, nil
	})
	resolver := &DOHResolver{
		client:          &http.Client{Transport: transport},
		server:          "https://resolver.test/dns-query",
		resolverOptions: Options{Logger: discardLogger(), Ndots: 1},
	}

	testDOHLookup(t, resolver)
	if responseBody == nil || !responseBody.closed {
		t.Fatal("successful response body was not closed")
	}
}

func TestDOHBoundsErrorBodyDrain(t *testing.T) {
	body := &countingBody{}
	resolver := &DOHResolver{
		client: &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusBadGateway,
				Status:     "502 Bad Gateway",
				Header:     make(http.Header),
				Body:       body,
			}, nil
		})},
		server:          "https://resolver.test/dns-query",
		resolverOptions: Options{Logger: discardLogger(), Ndots: 1},
	}
	question := dns.Question{Name: "example.test.", Qtype: dns.TypeA, Qclass: dns.ClassINET}
	_, err := resolver.Lookup(context.Background(), []dns.Question{question}, QueryFlags{RD: true})
	if err == nil {
		t.Fatal("Lookup unexpectedly succeeded")
	}
	if body.read != maxErrorBodyDrain {
		t.Fatalf("drained %d bytes, want bounded %d", body.read, maxErrorBodyDrain)
	}
	if !body.closed {
		t.Fatal("error response body was not closed")
	}
}

func TestDOHResolverCloseIsConcurrentIdempotentAndRetryable(t *testing.T) {
	errFirstClose := errors.New("first close failed")
	var attempts atomic.Int32
	resolver := &DOHResolver{
		closeTransport: func() error {
			if attempts.Add(1) == 1 {
				return errFirstClose
			}
			return nil
		},
	}

	const callers = 16
	start := make(chan struct{})
	errorsSeen := make(chan error, callers)
	var wg sync.WaitGroup
	for range callers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			errorsSeen <- resolver.Close()
		}()
	}
	close(start)
	wg.Wait()
	close(errorsSeen)

	firstCloseErrors := 0
	for err := range errorsSeen {
		if errors.Is(err, errFirstClose) {
			firstCloseErrors++
			continue
		}
		if err != nil {
			t.Fatalf("unexpected Close error: %v", err)
		}
	}
	if firstCloseErrors != 1 {
		t.Fatalf("first-close errors = %d, want 1", firstCloseErrors)
	}
	if attempts.Load() != 2 {
		t.Fatalf("underlying close attempts = %d, want one failure and one retry", attempts.Load())
	}
	if err := resolver.Close(); err != nil {
		t.Fatalf("idempotent Close: %v", err)
	}
	if attempts.Load() != 2 {
		t.Fatalf("successful Close was retried; attempts = %d", attempts.Load())
	}
}

func TestDOHResolverCloseWithoutTransportIsNoOp(t *testing.T) {
	resolver := &DOHResolver{}
	if err := resolver.Close(); err != nil {
		t.Fatalf("Close() = %v, want nil", err)
	}
	if err := resolver.Close(); err != nil {
		t.Fatalf("second Close() = %v, want nil", err)
	}
}
