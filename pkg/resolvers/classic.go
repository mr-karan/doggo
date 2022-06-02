package resolvers

import (
	"crypto/tls"
	"time"

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
	UseTLS bool
	UseTCP bool
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

	if resolverOpts.UseIPv4 {
		net = net + "4"
	}
	if resolverOpts.UseIPv6 {
		net = net + "6"
	}

	if classicOpts.UseTLS {
		net = net + "-tls"
		// Provide extra TLS config for doing/skipping hostname verification.
		client.TLSConfig = &tls.Config{
			ServerName:         resolverOpts.TLSHostname,
			InsecureSkipVerify: resolverOpts.InsecureSkipVerify,
		}
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

		// Since the library doesn't include tcp.Dial time,
		// it's better to not rely on `rtt` provided here and calculate it ourselves.
		now := time.Now()
		in, _, err := r.client.Exchange(&msg, r.server)
		if err != nil {
			return rsp, err
		}

		// In case the response size exceeds 512 bytes (can happen with lot of TXT records),
		// fallback to TCP as with UDP the response is truncated. Fallback mechanism is in-line with `dig`.
		if in.Truncated {
			switch r.client.Net {
			case "udp":
				r.client.Net = "tcp"
			case "udp4":
				r.client.Net = "tcp4"
			case "udp6":
				r.client.Net = "tcp6"
			default:
				r.client.Net = "tcp"
			}
			r.resolverOptions.Logger.WithField("protocol", r.client.Net).Debug("Response truncated; retrying now")
			return r.Lookup(question)
		}

		// Pack questions in output.
		for _, q := range msg.Question {
			ques := Question{
				Name:  q.Name,
				Class: dns.ClassToString[q.Qclass],
				Type:  dns.TypeToString[q.Qtype],
			}
			rsp.Questions = append(rsp.Questions, ques)
		}
		rtt := time.Since(now)

		// Get the authorities and answers.
		output := parseMessage(in, rtt, r.server)
		rsp.Authorities = output.Authorities
		rsp.Answers = output.Answers

		if len(output.Answers) > 0 {
			// Stop iterating the searchlist.
			break
		}
	}
	return rsp, nil
}
