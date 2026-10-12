package resolvers

import (
	"context"
	"fmt"
	"net"
	"net/netip"
	"strconv"
	"strings"
	"time"
)

// parseSourceAddr parses a --source value into a local IP. The value may be
// a bare IP ("192.0.2.1", "2001:db8::1", "fe80::1%en0"), a bracketed IPv6
// literal without a port ("[2001:db8::1]"), or an IP with a port
// ("192.0.2.1:5300", "[2001:db8::1]:5300"). Only port 0 (or an omitted port)
// is accepted: doggo issues several queries concurrently and concurrent
// sockets cannot share one fixed local port. An empty string is not an error
// and yields a zero address, signalling that no source binding was requested.
func parseSourceAddr(source string) (netip.Addr, error) {
	source = strings.TrimSpace(source)
	if source == "" {
		return netip.Addr{}, nil
	}

	var (
		host, portStr = source, ""
		hasPort       bool
	)
	if h, p, err := net.SplitHostPort(source); err == nil {
		host, portStr, hasPort = h, p, true
	} else if strings.HasPrefix(source, "[") && strings.HasSuffix(source, "]") {
		// Bracketed IPv6 literal without a port.
		host = source[1 : len(source)-1]
	}

	addr, err := netip.ParseAddr(host)
	if err != nil {
		return netip.Addr{}, fmt.Errorf("invalid source address %q: not an IP address", source)
	}

	if hasPort {
		if portStr == "" {
			return netip.Addr{}, fmt.Errorf("invalid source address %q: missing port number", source)
		}
		port, err := strconv.Atoi(portStr)
		if err != nil || port < 0 || port > 65535 {
			return netip.Addr{}, fmt.Errorf("invalid source port in %q: must be between 0 and 65535", source)
		}
		if port != 0 {
			return netip.Addr{}, fmt.Errorf("invalid source address %q: a fixed source port is not supported because doggo issues queries concurrently (use a bare IP and let the OS pick a source port)", source)
		}
	}
	return addr, nil
}

// sourceUDPAddr parses the source address and returns the UDP network
// matching its family ("udp4" or "udp6") along with the local UDP address.
// A nil address is returned when no source address is configured.
func sourceUDPAddr(source string) (network string, laddr *net.UDPAddr, err error) {
	addr, err := parseSourceAddr(source)
	if err != nil || !addr.IsValid() {
		return "", nil, err
	}
	return udpNetwork(addr), &net.UDPAddr{IP: net.IP(addr.AsSlice()), Zone: addr.Zone()}, nil
}

// udpNetwork returns the UDP network matching the address family, so that
// remote addresses are resolved and sockets are bound with a consistent
// family (an IPv6 source must not dial an IPv4 remote).
func udpNetwork(addr netip.Addr) string {
	if addr.Is4() || addr.Is4In6() {
		return "udp4"
	}
	return "udp6"
}

// resolveUDPAddrCompat resolves addr restricted to network's address family,
// failing with a clear error when the server has no address of that family.
func resolveUDPAddrCompat(ctx context.Context, network, addr string) (*net.UDPAddr, error) {
	host, service, err := net.SplitHostPort(addr)
	if err != nil {
		return nil, err
	}
	port, err := net.DefaultResolver.LookupPort(ctx, network, service)
	if err != nil {
		return nil, fmt.Errorf("resolving port for %s: %w", addr, err)
	}
	ipNetwork := "ip4"
	if network == "udp6" {
		ipNetwork = "ip6"
	}
	zone := ""
	if ip, err := netip.ParseAddr(host); err == nil {
		zone = ip.Zone()
		host = ip.WithZone("").String()
	}
	ips, err := net.DefaultResolver.LookupIP(ctx, ipNetwork, host)
	if err != nil {
		return nil, fmt.Errorf("resolving %s over %s: %w", addr, network, err)
	}
	for _, ip := range ips {
		return &net.UDPAddr{IP: ip, Port: port, Zone: zone}, nil
	}
	return nil, fmt.Errorf("resolving %s over %s: no address of the requested IP family", addr, network)
}

// sourceLocalAddr builds a net.Addr of the concrete type the given network
// requires (a *net.TCPAddr for "tcp"/"tcp4"/"tcp6"/"tcp*-tls", a *net.UDPAddr
// otherwise) so it can be assigned to net.Dialer.LocalAddr. It returns a nil
// address when source is empty.
func sourceLocalAddr(network, source string) (net.Addr, error) {
	addr, err := parseSourceAddr(source)
	if err != nil || !addr.IsValid() {
		return nil, err
	}
	ip := net.IP(addr.AsSlice())
	if strings.HasPrefix(network, "tcp") {
		return &net.TCPAddr{IP: ip, Zone: addr.Zone()}, nil
	}
	return &net.UDPAddr{IP: ip, Zone: addr.Zone()}, nil
}

// sourceDialer returns a *net.Dialer bound to the configured source address
// for the given network, or a nil dialer when no source address is
// configured. An explicit dial timeout (and TCP keepalive) is set because
// callers that install a dialer on a dns.Client or http.Transport otherwise
// lose the default dial timeout.
func sourceDialer(network, source string, timeout time.Duration) (*net.Dialer, error) {
	la, err := sourceLocalAddr(network, source)
	if err != nil || la == nil {
		return nil, err
	}
	return &net.Dialer{
		Timeout:   timeout,
		KeepAlive: 30 * time.Second,
		LocalAddr: la,
	}, nil
}

// NewSourceDialer returns a net.Dialer bound to the given source address for
// the given network, for callers outside this package (such as the
// --authoritative bootstrap lookups). A nil dialer is returned when no
// source address is configured.
func NewSourceDialer(network, source string, timeout time.Duration) (*net.Dialer, error) {
	return sourceDialer(network, source, timeout)
}
