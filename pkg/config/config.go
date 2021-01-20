package config

import "net"

// the whole `FEC0::/10` prefix is deprecated.
// [RFC 3879]: https://tools.ietf.org/html/rfc3879
func isUnicastLinkLocal(ip net.IP) bool {
	return len(ip) == net.IPv6len && ip[0] == 0xfe && ip[1] == 0xc0
}
