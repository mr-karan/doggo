package resolvers

import (
	"time"

	"github.com/miekg/dns"
)

// UDPResolver represents the config options for setting up a Resolver.
type UDPResolver struct {
	client *dns.Client
	server string
}

// UDPResolverOpts holds options for setting up a Classic resolver.
type UDPResolverOpts struct {
	IPv4Only bool
	IPv6Only bool
	Timeout  time.Duration
}

// NewUDPResolver accepts a list of nameservers and configures a DNS resolver.
func NewUDPResolver(server string, opts UDPResolverOpts) (Resolver, error) {
	client := &dns.Client{
		Timeout: opts.Timeout,
	}

	client.Net = "udp"
	if opts.IPv4Only {
		client.Net = "udp4"
	}
	if opts.IPv6Only {
		client.Net = "udp6"
	}
	return &UDPResolver{
		client: client,
		server: server,
	}, nil
}

// Lookup prepare a list of DNS messages to be sent to the server.
// It's possible to send multiple question in one message
// but some nameservers are not able to
func (r *UDPResolver) Lookup(questions []dns.Question) ([]Response, error) {
	var (
		messages  = prepareMessages(questions)
		responses []Response
	)

	for _, msg := range messages {
		in, rtt, err := r.client.Exchange(&msg, r.server)
		if err != nil {
			return nil, err
		}
		rsp := Response{
			Message:    *in,
			RTT:        rtt,
			Nameserver: r.server,
		}
		responses = append(responses, rsp)
	}
	return responses, nil
}
