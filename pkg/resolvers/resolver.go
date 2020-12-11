package resolvers

import "github.com/miekg/dns"

type Resolver interface {
	Name() string
	Lookup([]dns.Question) error
}
