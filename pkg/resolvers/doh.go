package resolvers

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/miekg/dns"
)

// DOHResolver represents the config options for setting up a DOH based resolver.
type DOHResolver struct {
	client  *http.Client
	servers []string
}

// NewDOHResolver accepts a list of nameservers and configures a DOH based resolver.
func NewDOHResolver(servers []string) (Resolver, error) {
	httpClient := &http.Client{
		Timeout: 10 * time.Second,
	}
	return &DOHResolver{
		client:  httpClient,
		servers: servers,
	}, nil
}

func (r *DOHResolver) Lookup(questions []dns.Question) ([]Response, error) {
	var (
		messages  = prepareMessages(questions)
		responses []Response
	)

	for _, msg := range messages {
		// get the DNS Message in wire format.
		b, err := msg.Pack()
		if err != nil {
			return nil, err
		}
		for _, srv := range r.servers {
			now := time.Now()
			// Make an HTTP POST request to the DNS server with the DNS message as wire format bytes in the body.
			resp, err := r.client.Post(srv, "application/dns-message", bytes.NewBuffer(b))
			if err != nil {
				return nil, err
			}
			if resp.StatusCode != http.StatusOK {
				return nil, err
			}
			rtt := time.Since(now)
			// extract the binary response in DNS Message.
			body, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				return nil, err
			}

			err = msg.Unpack(body)
			if err != nil {
				return nil, err
			}
			rsp := Response{
				Message:    msg,
				RTT:        rtt,
				Nameserver: srv,
			}
			responses = append(responses, rsp)
		}
	}
	return responses, nil
}
