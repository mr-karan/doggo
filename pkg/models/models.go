package models

import "time"

const (
	// DefaultTLSPort specifies the default port for a DNS server connecting over TCP over TLS
	DefaultTLSPort = "853"
	// DefaultUDPPort specifies the default port for a DNS server connecting over UDP
	DefaultUDPPort = "53"
	// DefaultTCPPort specifies the default port for a DNS server connecting over TCP
	DefaultTCPPort = "53"
	UDPResolver    = "udp"
	DOHResolver    = "doh"
	TCPResolver    = "tcp"
	DOTResolver    = "dot"
)

// QueryFlags is used store the query params
// supplied by the user.
type QueryFlags struct {
	QNames           []string      `koanf:"query"`
	QTypes           []string      `koanf:"type"`
	QClasses         []string      `koanf:"class"`
	Nameservers      []string      `koanf:"nameserver"`
	UseIPv4          bool          `koanf:"ipv4"`
	UseIPv6          bool          `koanf:"ipv6"`
	DisplayTimeTaken bool          `koanf:"time"`
	ShowJSON         bool          `koanf:"json"`
	UseSearchList    bool          `koanf:"search"`
	Ndots            int           `koanf:"ndots"`
	Color            bool          `koanf:"color"`
	Timeout          time.Duration `koanf:"timeout"`
}

// Nameserver represents the type of Nameserver
// along with the server address.
type Nameserver struct {
	Address string
	Type    string
}
