package resolvers

import (
	"time"

	"github.com/miekg/dns"
)

// TCPResolver represents the config options for setting up a Resolver.
type TCPResolver struct {
	client *dns.Client
	server string
}

// TCPResolverOpts represents the config options for setting up a TCPResolver.
type TCPResolverOpts struct {
	IPv4Only bool
	IPv6Only bool
	Timeout  time.Duration
}

// NewTCPResolver accepts a list of nameservers and configures a DNS resolver.
func NewTCPResolver(server string, opts TCPResolverOpts) (Resolver, error) {
	client := &dns.Client{
		Timeout: opts.Timeout,
	}

	client.Net = "tcp"
	if opts.IPv4Only {
		client.Net = "tcp4"
	}
	if opts.IPv6Only {
		client.Net = "tcp6"
	}
	return &TCPResolver{
		client: client,
		server: server,
	}, nil
}

// Lookup prepare a list of DNS messages to be sent to the server.
// It's possible to send multiple question in one message
// but some nameservers are not able to
func (r *TCPResolver) Lookup(questions []dns.Question) ([]Response, error) {
	var (
		messages  = prepareMessages(questions)
		responses []Response
	)

	for _, msg := range messages {
		in, rtt, err := r.client.Exchange(&msg, r.server)
		if err != nil {
			return nil, err
		}
		msg.Answer = in.Answer
		rsp := Response{
			Message:    msg,
			RTT:        rtt,
			Nameserver: r.server,
		}
		responses = append(responses, rsp)
	}
	return responses, nil
}
