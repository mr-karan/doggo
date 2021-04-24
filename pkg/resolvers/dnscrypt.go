package resolvers

import (
	"github.com/ameshkov/dnscrypt"
	"github.com/miekg/dns"
	"github.com/sirupsen/logrus"
)

// DNSCryptResolver represents the config options for setting up a Resolver.
type DNSCryptResolver struct {
	client          *dnscrypt.Client
	serverInfo      *dnscrypt.ServerInfo
	server          string
	resolverOptions Options
}

// DNSCryptResolverOpts holds options for setting up a DNSCrypt resolver.
type DNSCryptResolverOpts struct {
	IPv4Only bool
	IPv6Only bool
	UseTLS   bool
	UseTCP   bool
}

// NewDNSCryptResolver accepts a list of nameservers and configures a DNS resolver.
func NewDNSCryptResolver(server string, dnscryptOpts DNSCryptResolverOpts, resolverOpts Options) (Resolver, error) {
	net := "udp"
	if dnscryptOpts.UseTCP {
		net = "tcp"
	}
	client := &dnscrypt.Client{Proto: net, AdjustPayloadSize: true}
	serverInfo, _, err := client.Dial(server)
	if err != nil {
		return nil, err
	}
	return &DNSCryptResolver{
		client:          client,
		serverInfo:      serverInfo,
		server:          server,
		resolverOptions: resolverOpts,
	}, nil
}

// Lookup takes a dns.Question and sends them to DNS Server.
// It parses the Response from the server in a custom output format.
func (r *DNSCryptResolver) Lookup(question dns.Question) (Response, error) {
	var (
		rsp      Response
		messages = prepareMessages(question, r.resolverOptions.Ndots, r.resolverOptions.SearchList)
	)
	for _, msg := range messages {
		r.resolverOptions.Logger.WithFields(logrus.Fields{
			"domain":     msg.Question[0].Name,
			"ndots":      r.resolverOptions.Ndots,
			"nameserver": r.serverInfo.ProviderName,
		}).Debug("Attempting to resolve")
		in, rtt, err := r.client.Exchange(&msg, r.serverInfo)
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
