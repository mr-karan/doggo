package resolvers

import (
	"net"
	"time"

	"github.com/miekg/dns"
)

const (
	// DefaultTLSPort specifies the default port for a DNS server connecting over TCP over TLS
	DefaultTLSPort = "853"
)

// TCPResolver represents the config options for setting up a Resolver.
type TCPResolver struct {
	client  *dns.Client
	servers []string
}

// TCPResolverOpts represents the config options for setting up a TCPResolver.
type TCPResolverOpts struct {
	IPv4Only bool
	IPv6Only bool
	Timeout  time.Duration
}

// NewTCPResolver accepts a list of nameservers and configures a DNS resolver.
func NewTCPResolver(servers []string, opts TCPResolverOpts) (Resolver, error) {
	client := &dns.Client{
		Timeout: opts.Timeout,
	}
	var nameservers []string

	// load list of nameservers to the config
	if len(servers) == 0 {
		ns, err := getDefaultServers()
		if err != nil {
			return nil, err
		}
		nameservers = ns
	} else {
		// load the list of servers that user specified.
		for _, srv := range servers {
			if i := net.ParseIP(srv); i != nil {
				// if no port specified in nameserver, append defaults.
				nameservers = append(nameservers, net.JoinHostPort(srv, DefaultTLSPort))
			} else {
				// use the port user specified.
				nameservers = append(nameservers, srv)
			}
		}
	}

	client.Net = "tcp"
	if opts.IPv4Only {
		client.Net = "tcp4"
	}
	if opts.IPv6Only {
		client.Net = "tcp6"
	}
	return &TCPResolver{
		client:  client,
		servers: nameservers,
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
		for _, srv := range r.servers {
			in, rtt, err := r.client.Exchange(&msg, srv)
			if err != nil {
				return nil, err
			}
			msg.Answer = in.Answer
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
