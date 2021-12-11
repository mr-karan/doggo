package resolvers

import (
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/lucas-clemente/quic-go"
	"github.com/miekg/dns"
	"github.com/sirupsen/logrus"
)

// DOQResolver represents the config options for setting up a DOQ based resolver.
type DOQResolver struct {
	tls             *tls.Config
	server          string
	resolverOptions Options
}

// NewDOQResolver accepts a nameserver address and configures a DOQ based resolver.
func NewDOQResolver(server string, resolverOpts Options) (Resolver, error) {
	return &DOQResolver{
		tls: &tls.Config{
			NextProtos: []string{"doq-i02", "doq-i00", "dq", "doq"},
		},
		server:          server,
		resolverOptions: resolverOpts,
	}, nil
}

// Lookup takes a dns.Question and sends them to DNS Server.
// It parses the Response from the server in a custom output format.
func (r *DOQResolver) Lookup(question dns.Question) (Response, error) {
	var (
		rsp      Response
		messages = prepareMessages(question, r.resolverOptions.Ndots, r.resolverOptions.SearchList)
	)

	session, err := quic.DialAddr(r.server, r.tls, nil)
	if err != nil {
		return rsp, err
	}
	defer session.CloseWithError(quic.ApplicationErrorCode(quic.NoError), "")

	for _, msg := range messages {
		r.resolverOptions.Logger.WithFields(logrus.Fields{
			"domain":     msg.Question[0].Name,
			"ndots":      r.resolverOptions.Ndots,
			"nameserver": r.server,
		}).Debug("Attempting to resolve")

		// get the DNS Message in wire format.
		var b []byte
		b, err = msg.Pack()
		if err != nil {
			return rsp, err
		}
		now := time.Now()

		var stream quic.Stream
		stream, err = session.OpenStream()
		if err != nil {
			return rsp, err
		}

		// Make a QUIC request to the DNS server with the DNS message as wire format bytes in the body.
		_, err = stream.Write(b)
		_ = stream.Close()
		if err != nil {
			return rsp, fmt.Errorf("send query error: %w", err)
		}
		err = stream.SetDeadline(time.Now().Add(r.resolverOptions.Timeout))
		if err != nil {
			return rsp, err
		}

		var buf []byte
		buf, err = io.ReadAll(stream)
		if err != nil {
			if errors.Is(err, os.ErrDeadlineExceeded) {
				return rsp, fmt.Errorf("timeout")
			}
			return rsp, err
		}
		rtt := time.Since(now)

		err = msg.Unpack(buf)
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
		output := parseMessage(&msg, rtt, r.server)
		rsp.Authorities = output.Authorities
		rsp.Answers = output.Answers

		if len(output.Answers) > 0 {
			// stop iterating the searchlist.
			break
		}
	}
	return rsp, nil
}
