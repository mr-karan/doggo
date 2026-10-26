package resolvers

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/ameshkov/dnscrypt/v2"
	"github.com/miekg/dns"
	"github.com/mr-karan/doggo/pkg/models"
)

// DNSCryptResolver represents the config options for setting up a Resolver.
type DNSCryptResolver struct {
	client          *dnscrypt.Client
	server          string
	resolverInfo    *dnscrypt.ResolverInfo
	resolverOptions Options
}

// NewDNSCryptResolver accepts a list of nameservers and configures a DNS resolver.
func NewDNSCryptResolver(server string, resolverOpts Options) (Resolver, error) {
	if resolverOpts.SourceAddr != "" {
		return nil, fmt.Errorf("binding to a source address (--source) is not supported for DNSCrypt resolvers")
	}

	client := &dnscrypt.Client{Net: "udp", Timeout: resolverOpts.Timeout, UDPSize: 1232}
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

// Address implements the Resolver interface.
func (r *DNSCryptResolver) Address() string {
	return r.server
}

// Lookup implements the Resolver interface
func (r *DNSCryptResolver) Lookup(ctx context.Context, questions []dns.Question, flags QueryFlags) ([]Response, error) {
	return ConcurrentLookup(ctx, questions, flags, r.query, r.resolverOptions.Logger)
}

// exchange sends msg to the server over network ("udp" or "tcp"). The
// library's Exchange dials without a timeout or context, so dial here and
// close the connection when ctx ends to stop an in-flight exchange.
func (r *DNSCryptResolver) exchange(ctx context.Context, network string, msg *dns.Msg) (*dns.Msg, error) {
	dialer := net.Dialer{Timeout: r.resolverOptions.Timeout}
	conn, err := dialer.DialContext(ctx, network, r.server)
	if err != nil {
		return nil, err
	}
	defer func() { _ = conn.Close() }()
	stop := context.AfterFunc(ctx, func() { _ = conn.Close() })
	defer stop()

	resp, err := r.client.ExchangeConn(conn, msg, r.resolverInfo)
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return nil, ctxErr
		}
		return nil, err
	}
	return resp, nil
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

		in, err := r.exchange(ctx, "udp", &msg)
		if err != nil {
			return rsp, err
		}
		// DNSCrypt servers set TC when the reply exceeds the UDP response
		// size limit; retry over TCP like dig and the classic resolver do.
		if in.Truncated {
			r.resolverOptions.Logger.Debug("Response truncated; retrying now", "protocol", "tcp")
			in, err = r.exchange(ctx, "tcp", &msg)
			if err != nil {
				return rsp, err
			}
		}
		rtt := time.Since(now)

		// pack questions in output.
		for _, q := range msg.Question {
			ques := Question{
				Name:  q.Name,
				Class: dns.ClassToString[q.Qclass],
				Type:  models.RecordTypeString(q.Qtype),
			}
			rsp.Questions = append(rsp.Questions, ques)
		}
		// get the authorities and answers.
		output := parseMessage(in, rtt, r.server)
		rsp.Authorities = output.Authorities
		rsp.Answers = output.Answers
		rsp.Additional = output.Additional
		rsp.Edns = mergeEdnsInfo(rsp.Edns, output.Edns)

		if len(output.Answers) > 0 || in.Rcode == dns.RcodeSuccess {
			// stop iterating the searchlist.
			return rsp, nil
		}
	}
	return rsp, nil
}
