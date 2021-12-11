package resolvers

import (
	"time"

	"github.com/miekg/dns"
	"github.com/mr-karan/doggo/pkg/models"
	"github.com/sirupsen/logrus"
)

// Options represent a set of common options
// to configure a Resolver.
type Options struct {
	Nameservers []models.Nameserver
	UseIPv4     bool
	UseIPv6     bool
	SearchList  []string
	Ndots       int
	Timeout     time.Duration
	Logger      *logrus.Logger
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

// LoadResolvers loads differently configured
// resolvers based on a list of nameserver.
func LoadResolvers(opts Options) ([]Resolver, error) {
	var resolverOpts = Options{
		Timeout:    opts.Timeout,
		Ndots:      opts.Ndots,
		SearchList: opts.SearchList,
		Logger:     opts.Logger,
	}
	// for each nameserver, initialise the correct resolver
	rslvrs := make([]Resolver, 0, len(opts.Nameservers))
	for _, ns := range opts.Nameservers {
		if ns.Type == models.DOHResolver {
			opts.Logger.Debug("initiating DOH resolver")
			rslvr, err := NewDOHResolver(ns.Address, resolverOpts)
			if err != nil {
				return rslvrs, err
			}
			rslvrs = append(rslvrs, rslvr)
		}
		if ns.Type == models.DOTResolver {
			opts.Logger.Debug("initiating DOT resolver")
			rslvr, err := NewClassicResolver(ns.Address,
				ClassicResolverOpts{
					IPv4Only: opts.UseIPv4,
					IPv6Only: opts.UseIPv6,
					UseTLS:   true,
					UseTCP:   true,
				}, resolverOpts)

			if err != nil {
				return rslvrs, err
			}
			rslvrs = append(rslvrs, rslvr)
		}
		if ns.Type == models.TCPResolver {
			opts.Logger.Debug("initiating TCP resolver")
			rslvr, err := NewClassicResolver(ns.Address,
				ClassicResolverOpts{
					IPv4Only: opts.UseIPv4,
					IPv6Only: opts.UseIPv6,
					UseTLS:   false,
					UseTCP:   true,
				}, resolverOpts)
			if err != nil {
				return rslvrs, err
			}
			rslvrs = append(rslvrs, rslvr)
		}
		if ns.Type == models.UDPResolver {
			opts.Logger.Debug("initiating UDP resolver")
			rslvr, err := NewClassicResolver(ns.Address,
				ClassicResolverOpts{
					IPv4Only: opts.UseIPv4,
					IPv6Only: opts.UseIPv6,
					UseTLS:   false,
					UseTCP:   false,
				}, resolverOpts)
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
				}, resolverOpts)
			if err != nil {
				return rslvrs, err
			}
			rslvrs = append(rslvrs, rslvr)
		}
		if ns.Type == models.DOQResolver {
			opts.Logger.Debug("initiating DOQ resolver")
			rslvr, err := NewDOQResolver(ns.Address, resolverOpts)
			if err != nil {
				return rslvrs, err
			}
			rslvrs = append(rslvrs, rslvr)
		}
	}
	return rslvrs, nil
}
