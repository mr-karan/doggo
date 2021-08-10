package resolvers

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"time"

	"github.com/miekg/dns"
	"github.com/sirupsen/logrus"
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
	httpClient := &http.Client{
		Timeout: resolverOpts.Timeout,
	}
	return &DOHResolver{
		client:          httpClient,
		server:          server,
		resolverOptions: resolverOpts,
	}, nil
}

// Lookup takes a dns.Question and sends them to DNS Server.
// It parses the Response from the server in a custom output format.
func (r *DOHResolver) Lookup(question dns.Question) (Response, error) {
	var (
		rsp      Response
		messages = prepareMessages(question, r.resolverOptions.Ndots, r.resolverOptions.SearchList)
	)

	for _, msg := range messages {
		r.resolverOptions.Logger.WithFields(logrus.Fields{
			"domain":     msg.Question[0].Name,
			"ndots":      r.resolverOptions.Ndots,
			"nameserver": r.server,
		}).Debug("Attempting to resolve")
		// get the DNS Message in wire format.
		b, err := msg.Pack()
		if err != nil {
			return rsp, err
		}
		now := time.Now()
		// Make an HTTP POST request to the DNS server with the DNS message as wire format bytes in the body.
		resp, err := r.client.Post(r.server, "application/dns-message", bytes.NewBuffer(b))
		if err != nil {
			return rsp, err
		}
		if resp.StatusCode == http.StatusMethodNotAllowed {
			url, err := url.Parse(r.server)
			if err != nil {
				return rsp, err
			}
			url.RawQuery = fmt.Sprintf("dns=%v", base64.RawURLEncoding.EncodeToString(b))
			resp, err = r.client.Get(url.String())
			if err != nil {
				return rsp, err
			}
		}
		if resp.StatusCode != http.StatusOK {
			return rsp, fmt.Errorf("error from nameserver %s", resp.Status)
		}
		rtt := time.Since(now)
		// if debug, extract the response headers
		if r.resolverOptions.Logger.IsLevelEnabled(logrus.DebugLevel) {
			for header, value := range resp.Header {
				r.resolverOptions.Logger.WithFields(logrus.Fields{
					header: value,
				}).Debug("DOH response header")
			}
		}
		// extract the binary response in DNS Message.
		body, err := ioutil.ReadAll(resp.Body)
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
	}
	return rsp, nil
}
