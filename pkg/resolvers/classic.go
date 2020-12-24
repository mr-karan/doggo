package resolvers

import (
	"github.com/miekg/dns"
	"github.com/sirupsen/logrus"
)

// ClassicResolver represents the config options for setting up a Resolver.
type ClassicResolver struct {
	client          *dns.Client
	server          string
	resolverOptions Options
}

// ClassicResolverOpts holds options for setting up a Classic resolver.
type ClassicResolverOpts struct {
	IPv4Only bool
	IPv6Only bool
	UseTLS   bool
	UseTCP   bool
}

// NewClassicResolver accepts a list of nameservers and configures a DNS resolver.
func NewClassicResolver(server string, classicOpts ClassicResolverOpts, resolverOpts Options) (Resolver, error) {
	net := "udp"
	client := &dns.Client{
		Timeout: resolverOpts.Timeout,
		Net:     "udp",
	}

	if classicOpts.UseTCP {
		net = "tcp"
	}

	if classicOpts.IPv4Only {
		net = net + "4"
	}
	if classicOpts.IPv6Only {
		net = net + "6"
	}

	if classicOpts.UseTLS {
		net = net + "-tls"
	}

	client.Net = net

	return &ClassicResolver{
		client:          client,
		server:          server,
		resolverOptions: resolverOpts,
	}, nil
}

// Lookup takes a dns.Question and sends them to DNS Server.
// It parses the Response from the server in a custom output format.
func (r *ClassicResolver) Lookup(question dns.Question) (Response, error) {
	var (
		rsp      Response
		messages = prepareMessages(question, r.resolverOptions.Ndots, r.resolverOptions.SearchList)
	)
	for _, msg := range messages {
		r.resolverOptions.Logger.WithFields(logrus.Fields{
			"domain":     msg.Question[0].Name,
			"ndots":      r.resolverOptions.Ndots,
			"nameserver": r.server,
		}).Debug("Attempting to resolve")
		in, rtt, err := r.client.Exchange(&msg, r.server)
		if err != nil {
			return rsp, err
		}
		// pack questions in output.
		for _, q := range msg.Question {
			ques := Question{
				Name:  q.Name,
				Class: dns.ClassToString[q.Qclass],
				Type:  dns.TypeToString[q.Qtype],
			}
			rsp.Questions = append(rsp.Questions, ques)
		}
		// get the authorities and answers.
		output := parseMessage(in, rtt, r.server)
		rsp.Authorities = output.Authorities
		rsp.Answers = output.Answers

		if len(output.Answers) > 0 {
			// stop iterating the searchlist.
			break
		}
	}
	return rsp, nil
}
