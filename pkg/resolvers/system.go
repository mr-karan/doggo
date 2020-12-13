package resolvers

import (
	"fmt"
	"net"

	"github.com/miekg/dns"
)

// SystemResolver represents the config options based on the
// resolvconf file.
type SystemResolver struct {
	client  *dns.Client
	config  *dns.ClientConfig
	servers []string
}

// NewSystemResolver loads the configuration from resolv config file
// and initialises a DNS resolver.
func NewSystemResolver(resolvFilePath string) (Resolver, error) {
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
	return &SystemResolver{
		client:  client,
		servers: servers,
		config:  cfg,
	}, nil
}

// Lookup prepare a list of DNS messages to be sent to the server.
// It's possible to send multiple question in one message
// but some nameservers are not able to
func (s *SystemResolver) Lookup(questions []dns.Question) ([]Response, error) {
	var (
		messages  = prepareMessages(questions)
		responses []Response
	)

	for _, msg := range messages {
		for _, srv := range s.servers {
			in, rtt, err := s.client.Exchange(&msg, srv)
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
