package resolvers

import (
	"net"
	"testing"
)

func TestParseSourceAddr(t *testing.T) {
	tests := []struct {
		name     string
		source   string
		wantIP   string
		wantPort int
		wantNil  bool
		wantErr  bool
	}{
		{name: "empty yields nil", source: "", wantNil: true},
		{name: "whitespace yields nil", source: "   ", wantNil: true},
		{name: "bare ipv4", source: "192.0.2.1", wantIP: "192.0.2.1", wantPort: 0},
		{name: "ipv4 with port", source: "192.0.2.1:5300", wantIP: "192.0.2.1", wantPort: 5300},
		{name: "bare ipv6", source: "2001:db8::1", wantIP: "2001:db8::1", wantPort: 0},
		{name: "bracketed ipv6 with port", source: "[2001:db8::1]:5300", wantIP: "2001:db8::1", wantPort: 5300},
		{name: "loopback", source: "127.0.0.1", wantIP: "127.0.0.1", wantPort: 0},
		{name: "not an ip", source: "example.com", wantErr: true},
		{name: "garbage", source: "not-an-ip", wantErr: true},
		{name: "port out of range", source: "192.0.2.1:70000", wantErr: true},
		{name: "negative port unparseable", source: "192.0.2.1:-1", wantErr: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ip, port, err := parseSourceAddr(tc.source)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error for %q, got ip=%v port=%d", tc.source, ip, port)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error for %q: %v", tc.source, err)
			}
			if tc.wantNil {
				if ip != nil {
					t.Fatalf("expected nil IP for %q, got %v", tc.source, ip)
				}
				return
			}
			if !ip.Equal(net.ParseIP(tc.wantIP)) {
				t.Fatalf("ip mismatch for %q: got %v, want %s", tc.source, ip, tc.wantIP)
			}
			if port != tc.wantPort {
				t.Fatalf("port mismatch for %q: got %d, want %d", tc.source, port, tc.wantPort)
			}
		})
	}
}

func TestSourceLocalAddrType(t *testing.T) {
	udpAddr, err := sourceLocalAddr("udp", "192.0.2.1:5300")
	if err != nil {
		t.Fatalf("udp: unexpected error: %v", err)
	}
	if _, ok := udpAddr.(*net.UDPAddr); !ok {
		t.Fatalf("udp: expected *net.UDPAddr, got %T", udpAddr)
	}

	// TCP and DoT ("tcp-tls") networks must both yield a *net.TCPAddr so the
	// address type matches the dialed network.
	for _, network := range []string{"tcp", "tcp4", "tcp-tls"} {
		tcpAddr, err := sourceLocalAddr(network, "192.0.2.1")
		if err != nil {
			t.Fatalf("%s: unexpected error: %v", network, err)
		}
		if _, ok := tcpAddr.(*net.TCPAddr); !ok {
			t.Fatalf("%s: expected *net.TCPAddr, got %T", network, tcpAddr)
		}
	}

	// An empty source is not an error and must not produce an address.
	if addr, err := sourceLocalAddr("udp", ""); err != nil || addr != nil {
		t.Fatalf("empty source: got addr=%v err=%v, want nil,nil", addr, err)
	}
}

func TestSourceDialerAndUDPAddr(t *testing.T) {
	dialer, err := sourceDialer("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("sourceDialer: unexpected error: %v", err)
	}
	if dialer == nil || dialer.LocalAddr == nil {
		t.Fatal("sourceDialer: expected a dialer with LocalAddr set")
	}

	if d, err := sourceDialer("tcp", ""); err != nil || d != nil {
		t.Fatalf("sourceDialer empty: got %v, %v; want nil,nil", d, err)
	}

	udp, err := sourceUDPAddr("127.0.0.1:0")
	if err != nil {
		t.Fatalf("sourceUDPAddr: unexpected error: %v", err)
	}
	if udp == nil || !udp.IP.Equal(net.IPv4(127, 0, 0, 1)) {
		t.Fatalf("sourceUDPAddr: unexpected result %v", udp)
	}

	if _, err := sourceUDPAddr("nope"); err == nil {
		t.Fatal("sourceUDPAddr: expected error for invalid address")
	}
}
