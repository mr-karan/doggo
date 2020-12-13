package resolvers

import (
	"net"

	"github.com/miekg/dns"
)

const (
	// DefaultUDPPort specifies the default port for a DNS server connecting over UDP
	DefaultUDPPort = "53"
	// DefaultTLSPort specifies the default port for a DNS server connecting over TCP over TLS
	DefaultTLSPort = "853"
	//DefaultResolvConfPath specifies path to default resolv config file on UNIX.
	DefaultResolvConfPath = "/etc/resolv.conf"
)

// ClassicResolver represents the config options for setting up a Resolver.
type ClassicResolver struct {
	client  *dns.Client
	servers []string
}

// ClassicResolverOpts holds options for setting up a Classic resolver.
type ClassicResolverOpts struct {
	UseIPv4 bool
	UseIPv6 bool
	UseTCP  bool
	UseTLS  bool
}

// NewClassicResolver accepts a list of nameservers and configures a DNS resolver.
func NewClassicResolver(servers []string, opts ClassicResolverOpts) (Resolver, error) {
	client := &dns.Client{}
	var nameservers []string
	for _, srv := range servers {
		if i := net.ParseIP(srv); i != nil {
			// if no port specified in nameserver, append defaults.
			if opts.UseTLS == true {
				nameservers = append(nameservers, net.JoinHostPort(srv, DefaultTLSPort))
			} else {
				nameservers = append(nameservers, net.JoinHostPort(srv, DefaultUDPPort))
			}
		} else {
			// use the port user specified.
			nameservers = append(nameservers, srv)
		}
	}

	client.Net = "udp"
	if opts.UseIPv4 {
		client.Net = "udp4"
	}
	if opts.UseIPv6 {
		client.Net = "udp6"
	}
	if opts.UseTCP {
		client.Net = "tcp"
	}
	if opts.UseTLS {
		client.Net = "tcp-tls"
	}
	return &ClassicResolver{
		client:  client,
		servers: nameservers,
	}, nil
}

// Lookup prepare a list of DNS messages to be sent to the server.
// It's possible to send multiple question in one message
// but some nameservers are not able to
func (c *ClassicResolver) Lookup(questions []dns.Question) ([]Response, error) {
	var (
		messages  = prepareMessages(questions)
		responses []Response
	)

	for _, msg := range messages {
		for _, srv := range c.servers {
			in, rtt, err := c.client.Exchange(&msg, srv)
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
