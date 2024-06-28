package resolvers

import (
	"time"

	"github.com/ameshkov/dnscrypt/v2"
	"github.com/miekg/dns"
	"github.com/sirupsen/logrus"
)

// DNSCryptResolver represents the config options for setting up a Resolver.
type DNSCryptResolver struct {
	client          *dnscrypt.Client
	server          string
	resolverInfo    *dnscrypt.ResolverInfo
	resolverOptions Options
}

// DNSCryptResolverOpts holds options for setting up a DNSCrypt resolver.
type DNSCryptResolverOpts struct {
	UseTCP bool
}

// NewDNSCryptResolver accepts a list of nameservers and configures a DNS resolver.
func NewDNSCryptResolver(server string, dnscryptOpts DNSCryptResolverOpts, resolverOpts Options) (Resolver, error) {
	net := "udp"
	if dnscryptOpts.UseTCP {
		net = "tcp"
	}

	client := &dnscrypt.Client{Net: net, Timeout: resolverOpts.Timeout, UDPSize: 4096}
	resolverInfo, err := client.Dial(server)
	if err != nil {
		return nil, err
	}
	return &DNSCryptResolver{
		client:          client,
		resolverInfo:    resolverInfo,
		server:          resolverInfo.ServerAddress,
		resolverOptions: resolverOpts,
	}, nil
}

// Lookup takes a dns.Question and sends them to DNS Server.
// It parses the Response from the server in a custom output format.
func (r *DNSCryptResolver) Lookup(question dns.Question, flags QueryFlags) (Response, error) {
	var (
		rsp      Response
		messages = prepareMessages(question, flags, r.resolverOptions.Ndots, r.resolverOptions.SearchList)
	)
	for _, msg := range messages {
		r.resolverOptions.Logger.WithFields(logrus.Fields{
			"domain":     msg.Question[0].Name,
			"ndots":      r.resolverOptions.Ndots,
			"nameserver": r.server,
		}).Debug("Attempting to resolve")
		now := time.Now()
		in, err := r.client.Exchange(&msg, r.resolverInfo)
		if err != nil {
			return rsp, err
		}
		rtt := time.Since(now)
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
