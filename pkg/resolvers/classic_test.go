package resolvers

import (
	"context"
	"net"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/miekg/dns"
)

// TestClassicResolverOversizedUDPResponse ensures the classic UDP client's
// receive buffer is large enough for responses bigger than 512 bytes even
// when the query carries no EDNS0 OPT record advertising a buffer. Without
// a sized buffer the datagram is either OS-truncated and fails to unpack
// (Unix) or fails outright with WSAEMSGSIZE (Windows). See issue #251.
func TestClassicResolverOversizedUDPResponse(t *testing.T) {
	var sawOpt atomic.Bool
	handler := dns.HandlerFunc(func(w dns.ResponseWriter, r *dns.Msg) {
		sawOpt.Store(r.IsEdns0() != nil)
		m := new(dns.Msg)
		m.SetReply(r)
		// Pad the response well past 512 bytes while leaving TC clear.
		for i := 0; i < 4; i++ {
			m.Answer = append(m.Answer, &dns.TXT{
				Hdr: dns.RR_Header{Name: r.Question[0].Name, Rrtype: dns.TypeTXT, Class: dns.ClassINET, Ttl: 60},
				Txt: []string{strings.Repeat("x", 200)},
			})
		}
		if packed, err := m.Pack(); err != nil || len(packed) <= 512 {
			t.Errorf("test server response must exceed 512 bytes, got %d (err %v)", len(packed), err)
		}
		_ = w.WriteMsg(m)
	})

	pc, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	usrv := &dns.Server{PacketConn: pc, Handler: handler}
	go func() { _ = usrv.ActivateAndServe() }()
	t.Cleanup(func() {
		if err := usrv.Shutdown(); err != nil {
			t.Errorf("UDP server shutdown: %v", err)
		}
	})

	rslvr, err := NewClassicResolver(pc.LocalAddr().String(), ClassicResolverOpts{}, Options{
		Timeout: 2 * time.Second,
		Logger:  discardLogger(),
	})
	if err != nil {
		t.Fatal(err)
	}
	rsp, err := rslvr.Lookup(context.Background(), []dns.Question{
		{Name: "oversized.test.", Qtype: dns.TypeTXT, Qclass: dns.ClassINET},
	}, QueryFlags{})
	if err != nil {
		t.Fatalf("lookup against oversized UDP response failed: %v", err)
	}
	if sawOpt.Load() {
		t.Error("plain query must not gain an EDNS0 OPT record from the receive-buffer fix")
	}
	if len(rsp) == 0 || len(rsp[0].Answers) != 4 {
		t.Fatalf("expected all 4 answers from the oversized response, got %+v", rsp)
	}
}

// TestClassicResolverTruncatedRetryWithSource exercises the truncated-UDP to
// TCP fallback together with a bound source address: the retry must switch
// the client to TCP and rebuild the source dialer so its local address type
// matches the dialed network.
func TestClassicResolverTruncatedRetryWithSource(t *testing.T) {
	handler := dns.HandlerFunc(func(w dns.ResponseWriter, r *dns.Msg) {
		m := new(dns.Msg)
		m.SetReply(r)
		if _, ok := w.RemoteAddr().(*net.UDPAddr); ok {
			m.Truncated = true // force the TCP fallback
		} else {
			m.Answer = []dns.RR{&dns.A{
				Hdr: dns.RR_Header{Name: r.Question[0].Name, Rrtype: dns.TypeA, Class: dns.ClassINET, Ttl: 60},
				A:   net.IPv4(192, 0, 2, 99),
			}}
		}
		_ = w.WriteMsg(m)
	})

	// Serve UDP and TCP on the same loopback port.
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	server := l.Addr().String()
	_, port, err := net.SplitHostPort(server)
	if err != nil {
		t.Fatal(err)
	}
	pc, err := net.ListenPacket("udp", net.JoinHostPort("127.0.0.1", port))
	if err != nil {
		t.Fatal(err)
	}

	usrv := &dns.Server{PacketConn: pc, Handler: handler}
	tsrv := &dns.Server{Listener: l, Handler: handler}
	go func() { _ = usrv.ActivateAndServe() }()
	go func() { _ = tsrv.ActivateAndServe() }()
	t.Cleanup(func() {
		if err := usrv.Shutdown(); err != nil {
			t.Errorf("UDP server shutdown: %v", err)
		}
		if err := tsrv.Shutdown(); err != nil {
			t.Errorf("TCP server shutdown: %v", err)
		}
	})

	rslvr, err := NewClassicResolver(server, ClassicResolverOpts{}, Options{
		Timeout:    2 * time.Second,
		Logger:     discardLogger(),
		SourceAddr: "127.0.0.1",
	})
	if err != nil {
		t.Fatal(err)
	}
	rsp, err := rslvr.Lookup(context.Background(), []dns.Question{
		{Name: "retry.test.", Qtype: dns.TypeA, Qclass: dns.ClassINET},
	}, QueryFlags{})
	if err != nil {
		t.Fatalf("query with source + truncation retry failed: %v", err)
	}
	if len(rsp) == 0 || len(rsp[0].Answers) == 0 {
		t.Fatal("expected answers from the TCP retry")
	}
}
