package resolvers

import (
	"fmt"
	"net"

	"github.com/miekg/dns"
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
}

//DefaultResolvConfPath specifies path to default resolv config file on UNIX.
const DefaultResolvConfPath = "/etc/resolv.conf"

// NewClassicResolver accepts a list of nameservers and configures a DNS resolver.
func NewClassicResolver(servers []string, opts ClassicResolverOpts) (Resolver, error) {
	client := &dns.Client{}
	var nameservers []string
	for _, srv := range servers {
		if i := net.ParseIP(srv); i != nil {
			nameservers = append(nameservers, net.JoinHostPort(srv, "53"))
		} else {
			host, port, err := net.SplitHostPort(srv)
			if err != nil {
				return nil, err
			}
			nameservers = append(nameservers, fmt.Sprintf("%s:%s", host, port))
		}
	}
	client.Net = "udp"
	if opts.UseIPv4 {
		client.Net = "udp4"
	}
	if opts.UseIPv6 {
		client.Net = "udp6"
	}
	return &ClassicResolver{
		client:  client,
		servers: nameservers,
	}, nil
}

// NewResolverFromResolvFile loads the configuration from resolv config file
// and initialises a DNS resolver.
func NewResolverFromResolvFile(resolvFilePath string) (Resolver, error) {
	if resolvFilePath == "" {
		resolvFilePath = DefaultResolvConfPath
	}
	cfg, err := dns.ClientConfigFromFile(resolvFilePath)
	if err != nil {
		return nil, err
	}

	servers := make([]string, 0, len(cfg.Servers))
	for _, s := range cfg.Servers {
		ip := net.ParseIP(s)
		// handle IPv6
		if ip != nil && ip.To4() != nil {
			servers = append(servers, fmt.Sprintf("%s:%s", s, cfg.Port))
		} else {
			servers = append(servers, fmt.Sprintf("[%s]:%s", s, cfg.Port))
		}
	}

	client := &dns.Client{}
	return &ClassicResolver{
		client:  client,
		servers: servers,
	}, nil
}

// Lookup prepare a list of DNS messages to be sent to the server.
// It's possible to send multiple question in one message
// but some nameservers are not able to
func (c *ClassicResolver) Lookup(questions []dns.Question) error {
	messages := prepareMessages(questions)

	for _, msg := range messages {
		for _, srv := range c.servers {
			in, rtt, err := c.client.Exchange(&msg, srv)
			if err != nil {
				return err
			}
			for _, ans := range in.Answer {
				if t, ok := ans.(*dns.A); ok {
					fmt.Println(t.String())
				}
			}
			fmt.Println("rtt is", rtt)
		}
	}
	return nil
}
