package models

import (
	"strings"
	"time"
)

const (
	// DefaultTLSPort specifies the default port for a DNS server connecting over TCP over TLS.
	DefaultTLSPort = "853"
	// DefaultUDPPort specifies the default port for a DNS server connecting over UDP.
	DefaultUDPPort = "53"
	// DefaultTCPPort specifies the default port for a DNS server connecting over TCP.
	DefaultTCPPort = "53"
	// DefaultDOQPort specifies the default port for a DNS server connecting over DNS over QUIC.
	DefaultDOQPort   = "853"
	UDPResolver      = "udp"
	DOHResolver      = "doh"
	TCPResolver      = "tcp"
	DOTResolver      = "dot"
	DNSCryptResolver = "dnscrypt"
	DOQResolver      = "doq"
)

// QueryFlags is used store the query params
// supplied by the user.
type QueryFlags struct {
	QNames             []string      `koanf:"query" json:"query"`
	QTypes             []string      `koanf:"type" json:"type"`
	QClasses           []string      `koanf:"class" json:"class"`
	Nameservers        []string      `koanf:"nameservers" json:"nameservers"`
	UseIPv4            bool          `koanf:"ipv4" json:"ipv4"`
	UseIPv6            bool          `koanf:"ipv6" json:"ipv6"`
	Ndots              int           `koanf:"ndots" json:"ndots"`
	Timeout            time.Duration `koanf:"timeout" json:"timeout"`
	Color              bool          `koanf:"color" json:"-"`
	DisplayTimeTaken   bool          `koanf:"time" json:"-"`
	ShowJSON           bool          `koanf:"json" json:"-"`
	ShortOutput        bool          `koanf:"short" short:"-"`
	UseSearchList      bool          `koanf:"search" json:"-"`
	ReverseLookup      bool          `koanf:"reverse" reverse:"-"`
	Strategy           string        `koanf:"strategy" strategy:"-"`
	InsecureSkipVerify bool          `koanf:"skip-hostname-verification" skip-hostname-verification:"-"`
	TLSHostname        string        `koanf:"tls-hostname" tls-hostname:"-"`
	QueryAny           bool          `koanf:"any" json:"any"`
}

// Nameserver represents the type of Nameserver
// along with the server address.
type Nameserver struct {
	Address string
	Type    string
}

// CommonRecordTypes is a string containing all common DNS record types
const CommonRecordTypes = "A AAAA CNAME MX NS PTR SOA SRV TXT CAA"

// GetCommonRecordTypes returns a slice of common DNS record types
func GetCommonRecordTypes() []string {
	return strings.Fields(CommonRecordTypes)
}
