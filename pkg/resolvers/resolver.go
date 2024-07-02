package resolvers

import (
	"context"
	"log/slog"
	"time"

	"github.com/miekg/dns"
	"github.com/mr-karan/doggo/pkg/models"
)

// Options represent a set of common options
// to configure a Resolver.
type Options struct {
	Logger *slog.Logger

	Nameservers        []models.Nameserver
	UseIPv4            bool
	UseIPv6            bool
	SearchList         []string
	Ndots              int
	Timeout            time.Duration
	Strategy           string
	InsecureSkipVerify bool
	TLSHostname        string
}

// Resolver implements the configuration for a DNS
// Client. Different types of providers can load
// a DNS Resolver satisfying this interface.
type Resolver interface {
	Lookup(ctx context.Context, questions []dns.Question, flags QueryFlags) ([]Response, error)
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

// LoadResolvers loads differently configured
// resolvers based on a list of nameserver.
func LoadResolvers(opts Options) ([]Resolver, error) {
	// For each nameserver, initialise the correct resolver.
	rslvrs := make([]Resolver, 0, len(opts.Nameservers))

	for _, ns := range opts.Nameservers {
		if ns.Type == models.DOHResolver {
			opts.Logger.Debug("initiating DOH resolver")
			rslvr, err := NewDOHResolver(ns.Address, opts)
			if err != nil {
				return rslvrs, err
			}
			rslvrs = append(rslvrs, rslvr)
		}
		if ns.Type == models.DOTResolver {
			opts.Logger.Debug("initiating DOT resolver")
			rslvr, err := NewClassicResolver(ns.Address,
				ClassicResolverOpts{
					UseTLS: true,
					UseTCP: true,
				}, opts)

			if err != nil {
				return rslvrs, err
			}
			rslvrs = append(rslvrs, rslvr)
		}
		if ns.Type == models.TCPResolver {
			opts.Logger.Debug("initiating TCP resolver")
			rslvr, err := NewClassicResolver(ns.Address,
				ClassicResolverOpts{
					UseTLS: false,
					UseTCP: true,
				}, opts)
			if err != nil {
				return rslvrs, err
			}
			rslvrs = append(rslvrs, rslvr)
		}
		if ns.Type == models.UDPResolver {
			opts.Logger.Debug("initiating UDP resolver")
			rslvr, err := NewClassicResolver(ns.Address,
				ClassicResolverOpts{
					UseTLS: false,
					UseTCP: false,
				}, opts)
			if err != nil {
				return rslvrs, err
			}
			rslvrs = append(rslvrs, rslvr)
		}
		if ns.Type == models.DNSCryptResolver {
			opts.Logger.Debug("initiating DNSCrypt resolver")
			rslvr, err := NewDNSCryptResolver(ns.Address,
				DNSCryptResolverOpts{
					UseTCP: false,
				}, opts)
			if err != nil {
				return rslvrs, err
			}
			rslvrs = append(rslvrs, rslvr)
		}
		if ns.Type == models.DOQResolver {
			opts.Logger.Debug("initiating DOQ resolver")
			rslvr, err := NewDOQResolver(ns.Address, opts)
			if err != nil {
				return rslvrs, err
			}
			rslvrs = append(rslvrs, rslvr)
		}
	}
	return rslvrs, nil
}
