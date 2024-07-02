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
	"time"

	"github.com/miekg/dns"
)

// DOHResolver represents the config options for setting up a DOH based resolver.
type DOHResolver struct {
	client          *http.Client
	server          string
	resolverOptions Options
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
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.TLSClientConfig = &tls.Config{
		ServerName:         resolverOpts.TLSHostname,
		InsecureSkipVerify: resolverOpts.InsecureSkipVerify,
	}
	httpClient := &http.Client{
		Timeout:   resolverOpts.Timeout,
		Transport: transport,
	}
	return &DOHResolver{
		client:          httpClient,
		server:          server,
		resolverOptions: resolverOpts,
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
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusMethodNotAllowed {
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
			defer resp.Body.Close()
		}
		if resp.StatusCode != http.StatusOK {
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
			return rsp, err
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
				Type:  dns.TypeToString[q.Qtype],
			}
			rsp.Questions = append(rsp.Questions, ques)
		}
		// get the authorities and answers.
		output := parseMessage(&msg, rtt, r.server)
		rsp.Authorities = output.Authorities
		rsp.Answers = output.Answers

		if len(output.Answers) > 0 {
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

// Lookup implements the Resolver interface
func (r *DOHResolver) Lookup(ctx context.Context, questions []dns.Question, flags QueryFlags) ([]Response, error) {
	return ConcurrentLookup(ctx, questions, flags, r.query, r.resolverOptions.Logger)
}
