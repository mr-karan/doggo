package resolvers

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/miekg/dns"
	"github.com/mr-karan/doggo/pkg/models"
	"github.com/quic-go/quic-go/http3"
)

const maxErrorBodyDrain = 64 << 10

// DOHResolver represents the config options for setting up a DOH based resolver.
type DOHResolver struct {
	client          *http.Client
	server          string
	resolverOptions Options
	closeTransport  func() error
	closeMu         sync.Mutex
	closed          bool
}

// NewDOHResolver accepts a nameserver address and configures a DOH based resolver.
func NewDOHResolver(server string, resolverOpts Options) (Resolver, error) {
	// do basic validation
	u, err := url.ParseRequestURI(server)
	if err != nil {
		return nil, fmt.Errorf("%s is not a valid HTTPS nameserver", server)
	}
	if u.Scheme != "https" {
		return nil, fmt.Errorf("missing https in %s", server)
	}
	tlsConfig := &tls.Config{
		ServerName:         resolverOpts.TLSHostname,
		InsecureSkipVerify: resolverOpts.InsecureSkipVerify,
	}

	var (
		transport      http.RoundTripper
		closeTransport func() error
	)
	if resolverOpts.UseHTTP3 {
		h3Transport := &http3.Transport{TLSClientConfig: tlsConfig}
		transport = h3Transport
		closeTransport = h3Transport.Close
	} else {
		httpsTransport := http.DefaultTransport.(*http.Transport).Clone()
		httpsTransport.TLSClientConfig = tlsConfig
		transport = httpsTransport
		closeTransport = func() error {
			httpsTransport.CloseIdleConnections()
			return nil
		}
	}
	httpClient := &http.Client{
		Timeout:   resolverOpts.Timeout,
		Transport: transport,
	}
	return &DOHResolver{
		client:          httpClient,
		server:          server,
		resolverOptions: resolverOpts,
		closeTransport:  closeTransport,
	}, nil
}

// query takes a dns.Question and sends them to DNS Server.
// It parses the Response from the server in a custom output format.
func (r *DOHResolver) query(ctx context.Context, question dns.Question, flags QueryFlags) (Response, error) {
	var (
		rsp      Response
		messages = prepareMessages(question, flags, r.resolverOptions.Ndots, r.resolverOptions.SearchList)
	)

	for _, msg := range messages {
		r.resolverOptions.Logger.Debug("Attempting to resolve",
			"domain", msg.Question[0].Name,
			"ndots", r.resolverOptions.Ndots,
			"nameserver", r.server,
		)
		// get the DNS Message in wire format.
		b, err := msg.Pack()
		if err != nil {
			return rsp, err
		}
		now := time.Now()

		// Create a new request with the context
		req, err := http.NewRequestWithContext(ctx, "POST", r.server, bytes.NewBuffer(b))
		if err != nil {
			return rsp, err
		}
		req.Header.Set("Content-Type", "application/dns-message")

		// Make an HTTP POST request to the DNS server with the DNS message as wire format bytes in the body.
		resp, err := r.client.Do(req)
		if err != nil {
			return rsp, err
		}

		if resp.StatusCode == http.StatusMethodNotAllowed {
			drainAndClose(resp.Body)

			url, err := url.Parse(r.server)
			if err != nil {
				return rsp, err
			}
			url.RawQuery = fmt.Sprintf("dns=%v", base64.RawURLEncoding.EncodeToString(b))

			req, err = http.NewRequestWithContext(ctx, "GET", url.String(), nil)
			if err != nil {
				return rsp, err
			}
			resp, err = r.client.Do(req)
			if err != nil {
				return rsp, err
			}
		}
		if resp.StatusCode != http.StatusOK {
			drainAndClose(resp.Body)
			return rsp, fmt.Errorf("error from nameserver %s", resp.Status)
		}
		rtt := time.Since(now)

		// if debug, extract the response headers
		for header, value := range resp.Header {
			r.resolverOptions.Logger.Debug("DOH response header", header, value)
		}

		// extract the binary response in DNS Message.
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			_ = resp.Body.Close()
			return rsp, err
		}
		if err := resp.Body.Close(); err != nil {
			r.resolverOptions.Logger.Debug("Error closing DOH response body", "error", err)
		}

		err = msg.Unpack(body)
		if err != nil {
			return rsp, err
		}
		// pack questions in output.
		for _, q := range msg.Question {
			ques := Question{
				Name:  q.Name,
				Class: dns.ClassToString[q.Qclass],
				Type:  models.RecordTypeString(q.Qtype),
			}
			rsp.Questions = append(rsp.Questions, ques)
		}
		// get the authorities and answers.
		output := parseMessage(&msg, rtt, r.server)
		rsp.Authorities = output.Authorities
		rsp.Answers = output.Answers
		rsp.Additional = output.Additional
		rsp.Edns = mergeEdnsInfo(rsp.Edns, output.Edns)

		if len(output.Answers) > 0 || msg.Rcode == dns.RcodeSuccess {
			// stop iterating the searchlist.
			break
		}

		// Check if context is done after each iteration
		select {
		case <-ctx.Done():
			return rsp, ctx.Err()
		default:
			// Continue to next iteration
		}
	}
	return rsp, nil
}

func drainAndClose(body io.ReadCloser) {
	_, _ = io.CopyN(io.Discard, body, maxErrorBodyDrain)
	_ = body.Close()
}

// Close releases connections and sockets owned by the HTTP transport.
func (r *DOHResolver) Close() error {
	r.closeMu.Lock()
	defer r.closeMu.Unlock()
	if r.closed || r.closeTransport == nil {
		return nil
	}
	if err := r.closeTransport(); err != nil {
		return err
	}
	r.closed = true
	return nil
}

// Address implements the Resolver interface.
func (r *DOHResolver) Address() string {
	return r.server
}

// Lookup implements the Resolver interface
func (r *DOHResolver) Lookup(ctx context.Context, questions []dns.Question, flags QueryFlags) ([]Response, error) {
	return ConcurrentLookup(ctx, questions, flags, r.query, r.resolverOptions.Logger)
}
