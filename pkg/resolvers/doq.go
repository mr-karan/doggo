package resolvers

import (
	"context"
	"crypto/tls"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/miekg/dns"
	"github.com/quic-go/quic-go"
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
			NextProtos:         []string{"doq"},
			ServerName:         resolverOpts.TLSHostname,
			InsecureSkipVerify: resolverOpts.InsecureSkipVerify,
		},
		server:          server,
		resolverOptions: resolverOpts,
	}, nil
}

// Lookup implements the Resolver interface
func (r *DOQResolver) Lookup(questions []dns.Question, flags QueryFlags) ([]Response, error) {
	return ConcurrentLookup(questions, flags, r.query, r.resolverOptions.Logger)
}

// Lookup takes a dns.Question and sends them to DNS Server.
// It parses the Response from the server in a custom output format.
func (r *DOQResolver) query(question dns.Question, flags QueryFlags) (Response, error) {
	var (
		rsp      Response
		messages = prepareMessages(question, flags, r.resolverOptions.Ndots, r.resolverOptions.SearchList)
	)

	session, err := quic.DialAddr(context.TODO(), r.server, r.tls, nil)
	if err != nil {
		return rsp, err
	}
	defer session.CloseWithError(quic.ApplicationErrorCode(quic.NoError), "")

	for _, msg := range messages {
		r.resolverOptions.Logger.Debug("Attempting to resolve",
			"domain", msg.Question[0].Name,
			"ndots", r.resolverOptions.Ndots,
			"nameserver", r.server,
		)

		// ref: https://www.rfc-editor.org/rfc/rfc9250.html#name-dns-message-ids
		msg.Id = 0

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

		var msgLen = uint16(len(b))
		var msgLenBytes = []byte{byte(msgLen >> 8), byte(msgLen & 0xFF)}
		_, err = stream.Write(msgLenBytes)
		if err != nil {
			return rsp, err
		}
		// Make a QUIC request to the DNS server with the DNS message as wire format bytes in the body.
		_, err = stream.Write(b)
		if err != nil {
			return rsp, err
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

		_ = stream.Close()

		packetLen := binary.BigEndian.Uint16(buf[:2])
		if packetLen != uint16(len(buf[2:])) {
			return rsp, fmt.Errorf("packet length mismatch")
		}
		err = msg.Unpack(buf[2:])
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
