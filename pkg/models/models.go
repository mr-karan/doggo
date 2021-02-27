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
	QNames           []string      `koanf:"query" json:"query"`
	QTypes           []string      `koanf:"type" json:"type"`
	QClasses         []string      `koanf:"class" json:"class"`
	Nameservers      []string      `koanf:"nameservers" json:"nameservers"`
	UseIPv4          bool          `koanf:"ipv4" json:"ipv4"`
	UseIPv6          bool          `koanf:"ipv6" json:"ipv6"`
	Ndots            int           `koanf:"ndots" json:"ndots"`
	Color            bool          `koanf:"color" json:"color"`
	Timeout          time.Duration `koanf:"timeout" json:"timeout"`
	DisplayTimeTaken bool          `koanf:"time" json:"-"`
	ShowJSON         bool          `koanf:"json" json:"-"`
	UseSearchList    bool          `koanf:"search" json:"-"`
}

// Nameserver represents the type of Nameserver
// along with the server address.
type Nameserver struct {
	Address string
	Type    string
}
