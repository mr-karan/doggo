package resolve

import (
	"fmt"
	"net"

	"github.com/miekg/dns"
)

// Resolver holds the configuration for a dns.Client
type Resolver struct {
	client  *dns.Client
	servers []string
}

//DefaultResolvConfPath specifies path to default resolv config file on UNIX.
const DefaultResolvConfPath = "/etc/resolv.conf"

// NewResolver accepts a list of nameservers and configures a DNS resolver.
func NewResolver(servers []string) *Resolver {
	client := &dns.Client{}
	return &Resolver{
		client:  client,
		servers: servers,
	}
}

// NewResolverFromResolvFile loads the configuration from resolv config file
// and initialises a DNS resolver.
func NewResolverFromResolvFile(resolvFilePath string) (*Resolver, error) {
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
	return &Resolver{
		client:  client,
		servers: servers,
	}, nil
}

func (r *Resolver) Lookup(domains []string) []error {
	// Prepare a list of DNS messages to be sent to the server.
	// It's possible to send multiple question in one message
	// but some nameservers are not able to
	var messages = make([]dns.Msg, 0, len(domains))

	for _, d := range domains {
		msg := dns.Msg{}
		msg.Id = dns.Id()
		msg.RecursionDesired = true
		// It's recommended to only send 1 question for 1 DNS message.
		msg.Question = []dns.Question{(dns.Question{
			Name:   dns.Fqdn(d),
			Qtype:  dns.TypeA,
			Qclass: dns.ClassINET,
		})}
		messages = append(messages, msg)
	}
	var errors []error
	for _, msg := range messages {
		for _, srv := range r.servers {
			in, rtt, err := r.client.Exchange(&msg, srv)
			if err != nil {
				errors = append(errors, err)
			}
			for _, ans := range in.Answer {
				if t, ok := ans.(*dns.A); ok {
					fmt.Println(t.String())
				}
			}
			fmt.Println("rtt is", rtt, msg.Question)
		}
	}
	return errors
}
