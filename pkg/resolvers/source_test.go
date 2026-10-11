package resolvers

import (
	"context"
	"net"
	"strings"
	"testing"
	"time"
)

func TestParseSourceAddr(t *testing.T) {
	tests := []struct {
		name    string
		source  string
		wantIP  string
		wantNil bool
		wantErr string
	}{
		{name: "empty yields nil", source: "", wantNil: true},
		{name: "whitespace yields nil", source: "   ", wantNil: true},
		{name: "bare ipv4", source: "192.0.2.1", wantIP: "192.0.2.1"},
		{name: "ipv4 port 0 is ephemeral", source: "192.0.2.1:0", wantIP: "192.0.2.1"},
		{name: "bare ipv6", source: "2001:db8::1", wantIP: "2001:db8::1"},
		{name: "bracketed ipv6 with port 0", source: "[2001:db8::1]:0", wantIP: "2001:db8::1"},
		{name: "bracketed ipv6 without port", source: "[2001:db8::1]", wantIP: "2001:db8::1"},
		{name: "ipv6 with zone id", source: "fe80::1%en0", wantIP: "fe80::1"},
		{name: "loopback", source: "127.0.0.1", wantIP: "127.0.0.1"},
		{name: "not an ip", source: "example.com", wantErr: "not an IP address"},
		{name: "garbage", source: "not-an-ip", wantErr: "not an IP address"},
		{name: "port out of range", source: "192.0.2.1:70000", wantErr: "between 0 and 65535"},
		{name: "negative port unparseable", source: "192.0.2.1:-1", wantErr: "between 0 and 65535"},
		{name: "empty port", source: "192.0.2.1:", wantErr: "missing port number"},
		{name: "fixed port rejected", source: "192.0.2.1:5300", wantErr: "fixed source port is not supported"},
		{name: "fixed port rejected ipv6", source: "[2001:db8::1]:5300", wantErr: "fixed source port is not supported"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			addr, err := parseSourceAddr(tc.source)
			if tc.wantErr != "" {
				if err == nil {
					t.Fatalf("expected error for %q, got addr=%v", tc.source, addr)
				}
				if !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("error for %q should mention %q, got: %v", tc.source, tc.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error for %q: %v", tc.source, err)
			}
			if tc.wantNil {
				if addr.IsValid() {
					t.Fatalf("expected zero address for %q, got %v", tc.source, addr)
				}
				return
			}
			if addr.String() != tc.wantIP && !strings.HasPrefix(addr.String(), tc.wantIP+"%") {
				t.Fatalf("ip mismatch for %q: got %v, want %s", tc.source, addr, tc.wantIP)
			}
		})
	}
}

func TestSourceLocalAddrType(t *testing.T) {
	udpAddr, err := sourceLocalAddr("udp", "192.0.2.1")
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
	dialer, err := sourceDialer("tcp", "127.0.0.1", 5*time.Second)
	if err != nil {
		t.Fatalf("sourceDialer: unexpected error: %v", err)
	}
	if dialer == nil || dialer.LocalAddr == nil {
		t.Fatal("sourceDialer: expected a dialer with LocalAddr set")
	}
	if dialer.Timeout != 5*time.Second {
		t.Fatalf("sourceDialer: expected dial timeout to be set, got %v", dialer.Timeout)
	}

	if d, err := sourceDialer("tcp", "", time.Second); err != nil || d != nil {
		t.Fatalf("sourceDialer empty: got %v, %v; want nil,nil", d, err)
	}

	network, udp, err := sourceUDPAddr("127.0.0.1")
	if err != nil {
		t.Fatalf("sourceUDPAddr: unexpected error: %v", err)
	}
	if network != "udp4" || udp == nil || !udp.IP.Equal(net.IPv4(127, 0, 0, 1)) {
		t.Fatalf("sourceUDPAddr: unexpected result network=%s addr=%v", network, udp)
	}

	network, udp, err = sourceUDPAddr("2001:db8::1")
	if err != nil {
		t.Fatalf("sourceUDPAddr ipv6: unexpected error: %v", err)
	}
	if network != "udp6" || udp == nil {
		t.Fatalf("sourceUDPAddr ipv6: unexpected result network=%s addr=%v", network, udp)
	}

	if _, _, err := sourceUDPAddr("nope"); err == nil {
		t.Fatal("sourceUDPAddr: expected error for invalid address")
	}
}

func TestResolveUDPAddrCompat(t *testing.T) {
	for _, tc := range []struct {
		name     string
		network  string
		address  string
		wantIP   string
		wantZone string
		wantErr  bool
	}{
		{name: "ipv4", network: "udp4", address: "127.0.0.1:53", wantIP: "127.0.0.1"},
		{name: "ipv6", network: "udp6", address: "[::1]:53", wantIP: "::1"},
		{name: "ipv6 zone", network: "udp6", address: "[fe80::1%en0]:53", wantIP: "fe80::1", wantZone: "en0"},
		{name: "mapped ipv4", network: "udp4", address: "[::ffff:127.0.0.1]:53", wantIP: "127.0.0.1"},
		{name: "named port", network: "udp4", address: "127.0.0.1:domain", wantIP: "127.0.0.1"},
		{name: "ipv4 rejects ipv6", network: "udp4", address: "[::1]:53", wantErr: true},
		{name: "ipv6 rejects ipv4", network: "udp6", address: "127.0.0.1:53", wantErr: true},
		{name: "ipv6 rejects mapped ipv4", network: "udp6", address: "[::ffff:127.0.0.1]:53", wantErr: true},
		{name: "missing port", network: "udp4", address: "127.0.0.1", wantErr: true},
		{name: "invalid port", network: "udp4", address: "127.0.0.1:65536", wantErr: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			remote, err := resolveUDPAddrCompat(context.Background(), tc.network, tc.address)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("resolved %s over %s, want error", tc.address, tc.network)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if !remote.IP.Equal(net.ParseIP(tc.wantIP)) || remote.Port != 53 || remote.Zone != tc.wantZone {
				t.Fatalf("resolved address = %v, want IP %s, port 53, zone %q", remote, tc.wantIP, tc.wantZone)
			}
		})
	}
}
