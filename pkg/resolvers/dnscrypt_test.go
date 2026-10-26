package resolvers

import (
	"context"
	"net"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ameshkov/dnscrypt/v2"
	"github.com/miekg/dns"
)

// largeAnswerHandler replies with more A records than the DNSCrypt UDP
// response size limit allows, so the server sets TC on UDP.
type largeAnswerHandler struct {
	answers  int
	tcpCalls *atomic.Int32
}

func (h largeAnswerHandler) ServeDNS(rw dnscrypt.ResponseWriter, r *dns.Msg) error {
	if _, ok := rw.RemoteAddr().(*net.TCPAddr); ok {
		h.tcpCalls.Add(1)
	}
	m := new(dns.Msg)
	m.SetReply(r)
	for i := 0; i < h.answers; i++ {
		m.Answer = append(m.Answer, &dns.A{
			Hdr: dns.RR_Header{Name: r.Question[0].Name, Rrtype: dns.TypeA, Class: dns.ClassINET, Ttl: 60},
			A:   net.IPv4(127, 0, 0, byte(i)),
		})
	}
	return rw.WriteMsg(m)
}

// TestDNSCryptResolverTruncatedRetry ensures a DNSCrypt reply truncated over
// UDP is retried over TCP instead of returning an empty answer. See issue #262.
func TestDNSCryptResolverTruncatedRetry(t *testing.T) {
	rc, err := dnscrypt.GenerateResolverConfig("2.dnscrypt-cert.doggo.test", nil)
	if err != nil {
		t.Fatal(err)
	}
	cert, err := rc.CreateCert()
	if err != nil {
		t.Fatal(err)
	}

	const answers = 64
	var tcpCalls atomic.Int32
	srv := &dnscrypt.Server{
		ProviderName: rc.ProviderName,
		ResolverCert: cert,
		Handler:      largeAnswerHandler{answers: answers, tcpCalls: &tcpCalls},
	}

	// The stamp carries one address, so UDP and TCP must share the port.
	udpConn, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1)})
	if err != nil {
		t.Fatal(err)
	}
	addr := udpConn.LocalAddr().(*net.UDPAddr)
	tcpLn, err := net.ListenTCP("tcp", &net.TCPAddr{IP: addr.IP, Port: addr.Port})
	if err != nil {
		_ = udpConn.Close()
		t.Fatal(err)
	}
	go func() { _ = srv.ServeUDP(udpConn) }()
	go func() { _ = srv.ServeTCP(tcpLn) }()
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		if err := srv.Shutdown(ctx); err != nil {
			t.Errorf("DNSCrypt server shutdown: %v", err)
		}
		// Shutdown does not close caller-owned listeners.
		_ = udpConn.Close()
		_ = tcpLn.Close()
	})

	stamp, err := rc.CreateStamp(addr.String())
	if err != nil {
		t.Fatal(err)
	}
	rslvr, err := NewDNSCryptResolver(stamp.String(), DNSCryptResolverOpts{}, Options{
		Timeout: 2 * time.Second,
		Logger:  discardLogger(),
	})
	if err != nil {
		t.Fatal(err)
	}

	rsp, err := rslvr.Lookup(context.Background(), []dns.Question{
		{Name: "large.test.", Qtype: dns.TypeA, Qclass: dns.ClassINET},
	}, QueryFlags{})
	if err != nil {
		t.Fatalf("lookup failed: %v", err)
	}
	if len(rsp) == 0 || len(rsp[0].Answers) != answers {
		t.Fatalf("expected %d answers, got %+v", answers, rsp)
	}
	if tcpCalls.Load() != 1 {
		t.Fatalf("expected 1 TCP retry, got %d", tcpCalls.Load())
	}
}
