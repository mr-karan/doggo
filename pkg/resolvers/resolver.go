package resolvers

import "github.com/miekg/dns"

// Resolver implements the configuration for a DNS
// Client. Different types of client like (UDP/TCP/DOH/DOT)
// can be initialised.
type Resolver interface {
	Lookup([]dns.Question) error
}
