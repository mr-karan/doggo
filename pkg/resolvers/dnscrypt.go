package resolvers

import (
	"context"
	"time"

	"github.com/ameshkov/dnscrypt/v2"
	"github.com/miekg/dns"
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

// Lookup implements the Resolver interface
func (r *DNSCryptResolver) Lookup(ctx context.Context, questions []dns.Question, flags QueryFlags) ([]Response, error) {
	return ConcurrentLookup(ctx, questions, flags, r.query, r.resolverOptions.Logger)
}

// query performs a single DNS query
func (r *DNSCryptResolver) query(ctx context.Context, question dns.Question, flags QueryFlags) (Response, error) {
	var (
		rsp      Response
		messages = prepareMessages(question, flags, r.resolverOptions.Ndots, r.resolverOptions.SearchList)
	)
	for _, msg := range messages {
		r.resolverOptions.Logger.Debug("Attempting to resolve",
			"domain", msg.Question[0].Name,
			"ndots", r.resolverOptions.Ndots,
			"nameserver", r.server,
		)

		now := time.Now()

		// Use a channel to handle the result of the Exchange
		resultChan := make(chan struct {
			resp *dns.Msg
			err  error
		})

		go func() {
			resp, err := r.client.Exchange(&msg, r.resolverInfo)
			resultChan <- struct {
				resp *dns.Msg
				err  error
			}{resp, err}
		}()

		// Wait for either the query to complete or the context to be cancelled
		select {
		case result := <-resultChan:
			if result.err != nil {
				return rsp, result.err
			}
			in := result.resp
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
			rsp.Additional = output.Additional
			rsp.Edns = output.Edns

			if len(output.Answers) > 0 {
				// stop iterating the searchlist.
				return rsp, nil
			}
		case <-ctx.Done():
			return rsp, ctx.Err()
		}
	}
	return rsp, nil
}
