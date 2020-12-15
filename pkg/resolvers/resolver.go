package resolvers

import (
	"time"

	"github.com/miekg/dns"
)

// Resolver implements the configuration for a DNS
// Client. Different types of providers can load
// a DNS Resolver satisfying this interface.
type Resolver interface {
	Lookup([]dns.Question) ([]Response, error)
}

// Response represents a custom output format
// for DNS queries. It wraps metadata about the DNS query
// and the DNS Answer as well.
type Response struct {
	Message    dns.Msg
	RTT        time.Duration
	Nameserver string
}
