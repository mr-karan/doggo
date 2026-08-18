package resolvers

import (
	"fmt"
	"net"
	"strconv"
	"strings"
)

// parseSourceAddr parses a --source value into a local IP and port. The value
// may be a bare IP ("192.0.2.1", "2001:db8::1") or an IP with a port
// ("192.0.2.1:5300", "[2001:db8::1]:5300"). A missing or zero port lets the OS
// pick an ephemeral source port. An empty string is not an error and yields a
// nil IP, signalling that no source binding was requested.
func parseSourceAddr(source string) (net.IP, int, error) {
	source = strings.TrimSpace(source)
	if source == "" {
		return nil, 0, nil
	}

	host, portStr := source, ""
	if h, p, err := net.SplitHostPort(source); err == nil {
		host, portStr = h, p
	}

	ip := net.ParseIP(host)
	if ip == nil {
		return nil, 0, fmt.Errorf("invalid source address %q: not an IP address", source)
	}

	port := 0
	if portStr != "" {
		p, err := strconv.Atoi(portStr)
		if err != nil || p < 0 || p > 65535 {
			return nil, 0, fmt.Errorf("invalid source port in %q: must be between 0 and 65535", source)
		}
		port = p
	}
	return ip, port, nil
}

// sourceLocalAddr builds a net.Addr of the concrete type the given network
// requires (a *net.TCPAddr for "tcp"/"tcp4"/"tcp6"/"tcp*-tls", a *net.UDPAddr
// otherwise) so it can be assigned to net.Dialer.LocalAddr. It returns a nil
// address when source is empty.
func sourceLocalAddr(network, source string) (net.Addr, error) {
	ip, port, err := parseSourceAddr(source)
	if err != nil || ip == nil {
		return nil, err
	}
	if strings.HasPrefix(network, "tcp") {
		return &net.TCPAddr{IP: ip, Port: port}, nil
	}
	return &net.UDPAddr{IP: ip, Port: port}, nil
}

// sourceDialer returns a *net.Dialer bound to the configured source address for
// the given network, or a nil dialer when no source address is configured.
func sourceDialer(network, source string) (*net.Dialer, error) {
	la, err := sourceLocalAddr(network, source)
	if err != nil || la == nil {
		return nil, err
	}
	return &net.Dialer{LocalAddr: la}, nil
}

// sourceUDPAddr returns a *net.UDPAddr for binding a UDP PacketConn (used by
// the QUIC-based DoQ and HTTP/3 DoH transports), or nil when no source address
// is configured.
func sourceUDPAddr(source string) (*net.UDPAddr, error) {
	ip, port, err := parseSourceAddr(source)
	if err != nil || ip == nil {
		return nil, err
	}
	return &net.UDPAddr{IP: ip, Port: port}, nil
}
