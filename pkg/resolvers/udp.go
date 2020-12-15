package resolvers

import (
	"net"
	"time"

	"github.com/miekg/dns"
)

const (
	// DefaultUDPPort specifies the default port for a DNS server connecting over UDP
	DefaultUDPPort = "53"
	//DefaultResolvConfPath specifies path to default resolv config file on UNIX.
	DefaultResolvConfPath = "/etc/resolv.conf"
)

// UDPResolver represents the config options for setting up a Resolver.
type UDPResolver struct {
	client  *dns.Client
	servers []string
}

// UDPResolverOpts holds options for setting up a Classic resolver.
type UDPResolverOpts struct {
	IPv4Only bool
	IPv6Only bool
	Timeout  time.Duration
}

// NewUDPResolver accepts a list of nameservers and configures a DNS resolver.
func NewUDPResolver(servers []string, opts UDPResolverOpts) (Resolver, error) {
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
				nameservers = append(nameservers, net.JoinHostPort(srv, DefaultUDPPort))
			} else {
				// use the port user specified.
				nameservers = append(nameservers, srv)
			}
		}
	}

	client.Net = "udp"
	if opts.IPv4Only {
		client.Net = "udp4"
	}
	if opts.IPv6Only {
		client.Net = "udp6"
	}
	return &UDPResolver{
		client:  client,
		servers: nameservers,
	}, nil
}

// Lookup prepare a list of DNS messages to be sent to the server.
// It's possible to send multiple question in one message
// but some nameservers are not able to
func (c *UDPResolver) Lookup(questions []dns.Question) ([]Response, error) {
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
