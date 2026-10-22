package config

import "net"

// the whole `FEC0::/10` prefix is deprecated.
// [RFC 3879]: https://tools.ietf.org/html/rfc3879
func isUnicastLinkLocal(ip net.IP) bool {
	return len(ip) == net.IPv6len && ip[0] == 0xfe && ip[1] == 0xc0
}

// DomainNameserver is a nameserver selected from a macOS
// Supplemental/domain-specific resolver. Port is 0 when the resolver uses
// the default DNS port.
type DomainNameserver struct {
	IP   string
	Port int
}
