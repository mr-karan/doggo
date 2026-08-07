package main

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/miekg/dns"
	"github.com/mr-karan/doggo/internal/app"
	"github.com/quic-go/quic-go"
	"github.com/quic-go/quic-go/http3"
)

type webHTTP3Server struct {
	url         string
	protocol    atomic.Int32
	serverName  atomic.Value
	connections chan *quic.Conn
}

func startWebHTTP3Server(t *testing.T, responseStatus int) *webHTTP3Server {
	t.Helper()
	result := &webHTTP3Server{connections: make(chan *quic.Conn, 1)}
	result.serverName.Store("")

	bootstrap := httptest.NewTLSServer(http.NotFoundHandler())
	tlsConfig := bootstrap.TLS.Clone()
	bootstrap.Close()
	tlsConfig.GetConfigForClient = func(hello *tls.ClientHelloInfo) (*tls.Config, error) {
		result.serverName.Store(hello.ServerName)
		return nil, nil
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		result.protocol.Store(int32(r.ProtoMajor))
		if responseStatus != http.StatusOK {
			http.Error(w, "resolver failure", responseStatus)
			return
		}
		wire, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}
		var request dns.Msg
		if err := request.Unpack(wire); err != nil {
			http.Error(w, "invalid DNS message", http.StatusBadRequest)
			return
		}
		response := new(dns.Msg)
		response.SetReply(&request)
		for _, question := range request.Question {
			if question.Qtype != dns.TypeA {
				continue
			}
			rr, err := dns.NewRR(fmt.Sprintf("%s 60 IN A 192.0.2.188", question.Name))
			if err == nil {
				response.Answer = append(response.Answer, rr)
			}
		}
		packed, err := response.Pack()
		if err != nil {
			http.Error(w, "failed to pack DNS response", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/dns-message")
		_, _ = w.Write(packed)
	})

	conn, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0})
	if err != nil {
		t.Fatalf("ListenUDP: %v", err)
	}
	server := &http3.Server{
		Handler:   handler,
		TLSConfig: tlsConfig,
		ConnContext: func(ctx context.Context, conn *quic.Conn) context.Context {
			select {
			case result.connections <- conn:
			default:
			}
			return ctx
		},
	}
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
	result.url = "https://" + conn.LocalAddr().String() + "/dns-query"
	return result
}

func TestHandleLookupUsesAndClosesHTTP3(t *testing.T) {
	tests := []struct {
		name           string
		resolverStatus int
		wantAPIStatus  int
	}{
		{name: "success", resolverStatus: http.StatusOK, wantAPIStatus: http.StatusOK},
		{name: "lookup error", resolverStatus: http.StatusServiceUnavailable, wantAPIStatus: http.StatusInternalServerError},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h3Server := startWebHTTP3Server(t, tt.resolverStatus)
			payload, err := json.Marshal(map[string]any{
				"query":                      []string{"example.test"},
				"type":                       []string{"A"},
				"nameservers":                []string{h3Server.url},
				"http3":                      true,
				"timeout":                    2,
				"skip_hostname_verification": true,
				"tls_hostname":               "resolver.test",
			})
			if err != nil {
				t.Fatalf("Marshal: %v", err)
			}

			logger := slog.New(slog.NewTextHandler(io.Discard, nil))
			testApp := app.New(logger, nil, "test")
			req := httptest.NewRequest(http.MethodPost, "/api/lookup/", bytes.NewReader(payload))
			recorder := httptest.NewRecorder()
			wrap(testApp, handleLookup).ServeHTTP(recorder, req)

			if recorder.Code != tt.wantAPIStatus {
				t.Fatalf("API status = %d, want %d; body: %s", recorder.Code, tt.wantAPIStatus, recorder.Body.String())
			}
			if tt.wantAPIStatus == http.StatusOK && !strings.Contains(recorder.Body.String(), "192.0.2.188") {
				t.Fatalf("API response missing HTTP/3 DNS answer: %s", recorder.Body.String())
			}
			if h3Server.protocol.Load() != 3 {
				t.Fatalf("resolver protocol = HTTP/%d, want HTTP/3", h3Server.protocol.Load())
			}
			if got := h3Server.serverName.Load().(string); got != "resolver.test" {
				t.Fatalf("TLS server name = %q, want resolver.test", got)
			}

			var connection *quic.Conn
			select {
			case connection = <-h3Server.connections:
			case <-time.After(2 * time.Second):
				t.Fatal("HTTP/3 server did not observe a QUIC connection")
			}
			select {
			case <-connection.Context().Done():
			case <-time.After(2 * time.Second):
				t.Fatal("web handler returned without closing its HTTP/3 transport")
			}
		})
	}
}

func TestHandleLookupRejectsInvalidResolverConfiguration(t *testing.T) {
	tests := []struct {
		name       string
		payload    map[string]any
		wantReason string
	}{
		{
			name: "HTTP3 without DoH nameserver",
			payload: map[string]any{
				"query":       []string{"example.test"},
				"nameservers": []string{"127.0.0.1"},
				"http3":       true,
			},
			wantReason: "HTTP/3 requires at least one HTTPS (DoH) nameserver",
		},
		{
			name: "unsupported nameserver scheme",
			payload: map[string]any{
				"query":       []string{"example.test"},
				"nameservers": []string{"ftp://resolver.test"},
			},
			wantReason: "error parsing nameserver: ftp://resolver.test",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			payload, err := json.Marshal(tt.payload)
			if err != nil {
				t.Fatalf("Marshal: %v", err)
			}
			logger := slog.New(slog.NewTextHandler(io.Discard, nil))
			testApp := app.New(logger, nil, "test")
			req := httptest.NewRequest(http.MethodPost, "/api/lookup/", bytes.NewReader(payload))
			recorder := httptest.NewRecorder()
			wrap(testApp, handleLookup).ServeHTTP(recorder, req)

			if recorder.Code != http.StatusBadRequest {
				t.Fatalf("API status = %d, want 400; body: %s", recorder.Code, recorder.Body.String())
			}
			if !strings.Contains(recorder.Body.String(), tt.wantReason) {
				t.Fatalf("API response missing %q: %s", tt.wantReason, recorder.Body.String())
			}
		})
	}
}

func TestSendNameserverLoadErrorClassification(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantBody   string
		hiddenBody string
	}{
		{
			name:       "system resolver failure",
			err:        fmt.Errorf("%w: permission denied reading /private/system/resolvers", app.ErrSystemNameservers),
			wantStatus: http.StatusInternalServerError,
			wantBody:   systemNameserverErrorMessage,
			hiddenBody: "/private/system/resolvers",
		},
		{
			name:       "client validation failure",
			err:        errors.New("error parsing nameserver: ftp://resolver.test"),
			wantStatus: http.StatusBadRequest,
			wantBody:   "error parsing nameserver: ftp://resolver.test",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			sendNameserverLoadError(recorder, tt.err)

			if recorder.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d; body: %s", recorder.Code, tt.wantStatus, recorder.Body.String())
			}
			if !strings.Contains(recorder.Body.String(), tt.wantBody) {
				t.Fatalf("body missing %q: %s", tt.wantBody, recorder.Body.String())
			}
			if tt.hiddenBody != "" && strings.Contains(recorder.Body.String(), tt.hiddenBody) {
				t.Fatalf("body leaks system error detail %q: %s", tt.hiddenBody, recorder.Body.String())
			}
		})
	}
}

func TestHandleLookupRejectsInvalidRecordType(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	application := app.New(logger, nil, "test")
	handler := wrap(application, handleLookup)

	req := httptest.NewRequest(http.MethodPost, "/api/lookup/", strings.NewReader(`{
		"query": ["example.test"],
		"type": ["TYPE0"]
	}`))
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d; body: %s", recorder.Code, http.StatusBadRequest, recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), "reserved and cannot be used in a question") {
		t.Fatalf("body missing record-type validation error: %s", recorder.Body.String())
	}
}
