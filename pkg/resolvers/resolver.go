package resolvers

import (
	"time"

	"github.com/miekg/dns"
)

// Resolver implements the configuration for a DNS
// Client. Different types of client like (UDP/TCP/DOH/DOT)
// can be initialised.
type Resolver interface {
	Lookup([]dns.Question) ([]Response, error)
}

// Response represents a custom output format
// which wraps certain metadata about the DNS query
// and the DNS Answer as well.
type Response struct {
	Message    dns.Msg
	RTT        time.Duration
	Nameserver string
}
