package resolvers

import (
	"time"

	"github.com/miekg/dns"
)

// ClassicResolver represents the config options for setting up a Resolver.
type ClassicResolver struct {
	client *dns.Client
	server string
}

// ClassicResolverOpts holds options for setting up a Classic resolver.
type ClassicResolverOpts struct {
	IPv4Only bool
	IPv6Only bool
	Timeout  time.Duration
	UseTLS   bool
	UseTCP   bool
}

// NewClassicResolver accepts a list of nameservers and configures a DNS resolver.
func NewClassicResolver(server string, opts ClassicResolverOpts) (Resolver, error) {
	net := "udp"
	client := &dns.Client{
		Timeout: opts.Timeout,
		Net:     "udp",
	}

	if opts.UseTCP {
		net = "tcp"
	}

	if opts.IPv4Only {
		net = net + "4"
	}
	if opts.IPv6Only {
		net = net + "6"
	}

	if opts.UseTLS {
		net = net + "-tls"
	}

	client.Net = net

	return &ClassicResolver{
		client: client,
		server: server,
	}, nil
}

// Lookup prepare a list of DNS messages to be sent to the server.
// It's possible to send multiple question in one message
// but some nameservers are not able to
func (r *ClassicResolver) Lookup(questions []dns.Question) ([]Response, error) {
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
