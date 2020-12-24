package resolvers

import (
	"time"

	"github.com/miekg/dns"
	"github.com/sirupsen/logrus"
)

// Options represent a set of common options
// to configure a Resolver.
type Options struct {
	SearchList []string
	Ndots      int
	Timeout    time.Duration
	Logger     *logrus.Logger
}

// Resolver implements the configuration for a DNS
// Client. Different types of providers can load
// a DNS Resolver satisfying this interface.
type Resolver interface {
	Lookup(dns.Question) (Response, error)
}

// Response represents a custom output format
// for DNS queries. It wraps metadata about the DNS query
// and the DNS Answer as well.
type Response struct {
	Answers     []Answer    `json:"answers"`
	Authorities []Authority `json:"authorities"`
	Questions   []Question  `json:"questions"`
}

type Question struct {
	Name  string `json:"name"`
	Type  string `json:"type"`
	Class string `json:"class"`
}

type Answer struct {
	Name       string `json:"name"`
	Type       string `json:"type"`
	Class      string `json:"class"`
	TTL        string `json:"ttl"`
	Address    string `json:"address"`
	Status     string `json:"status"`
	RTT        string `json:"rtt"`
	Nameserver string `json:"nameserver"`
}

type Authority struct {
	Name       string `json:"name"`
	Type       string `json:"type"`
	Class      string `json:"class"`
	TTL        string `json:"ttl"`
	MName      string `json:"mname"`
	Status     string `json:"status"`
	RTT        string `json:"rtt"`
	Nameserver string `json:"nameserver"`
}
