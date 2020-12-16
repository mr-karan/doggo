package resolvers

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"time"

	"github.com/miekg/dns"
)

// DOHResolver represents the config options for setting up a DOH based resolver.
type DOHResolver struct {
	client *http.Client
	server string
}

type DOHResolverOpts struct {
	Timeout time.Duration
}

// NewDOHResolver accepts a nameserver address and configures a DOH based resolver.
func NewDOHResolver(server string, opts DOHResolverOpts) (Resolver, error) {
	// do basic validation
	u, err := url.ParseRequestURI(server)
	if err != nil {
		return nil, fmt.Errorf("%s is not a valid HTTPS nameserver", server)
	}
	if u.Scheme != "https" {
		return nil, fmt.Errorf("missing https in %s", server)
	}
	httpClient := &http.Client{
		Timeout: opts.Timeout,
	}
	return &DOHResolver{
		client: httpClient,
		server: server,
	}, nil
}

func (d *DOHResolver) Lookup(questions []dns.Question) ([]Response, error) {
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
		now := time.Now()
		// Make an HTTP POST request to the DNS server with the DNS message as wire format bytes in the body.
		resp, err := d.client.Post(d.server, "application/dns-message", bytes.NewBuffer(b))
		if err != nil {
			return nil, err
		}
		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("error from nameserver %s", resp.Status)
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
			Nameserver: d.server,
		}
		responses = append(responses, rsp)
	}
	return responses, nil
}
