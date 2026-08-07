package resolvers

import (
	"context"
	"errors"
	"fmt"
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
	UseHTTP3           bool
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
	// Address returns the nameserver identity used when reporting errors.
	Address() string
	Lookup(ctx context.Context, questions []dns.Question, flags QueryFlags) ([]Response, error)
}

// CloseResolvers closes resolver-owned resources and joins any close errors.
// Resolvers without a Close method don't own persistent resources.
func CloseResolvers(resolvers []Resolver) error {
	var closeErrors []error
	for _, resolver := range resolvers {
		closer, ok := resolver.(interface{ Close() error })
		if !ok {
			continue
		}
		if err := closer.Close(); err != nil {
			closeErrors = append(closeErrors, fmt.Errorf("closing resolver %s: %w", resolver.Address(), err))
		}
	}
	return errors.Join(closeErrors...)
}

// LookupError tags a resolver failure with the nameserver that produced it
// so partial failures can be reported per-resolver instead of as an opaque
// top-level error.
type LookupError struct {
	Nameserver string
	Err        error
}

func (e *LookupError) Error() string {
	if e.Nameserver == "" {
		return e.Err.Error()
	}
	return fmt.Sprintf("%s: %v", e.Nameserver, e.Err)
}

func (e *LookupError) Unwrap() error {
	return e.Err
}

// Response represents a custom output format
// for DNS queries. It wraps metadata about the DNS query
// and the DNS Answer as well.
type Response struct {
	Answers     []Answer    `json:"answers"`
	Authorities []Authority `json:"authorities"`
	Questions   []Question  `json:"questions"`
	Additional  []Answer    `json:"additional,omitempty"`
	Edns        *EdnsInfo   `json:"edns,omitempty"`
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

type EdnsInfo struct {
	Nameserver  string `json:"nameserver,omitempty"`
	NSID        string `json:"nsid,omitempty"`
	Cookie      string `json:"cookie,omitempty"`
	Subnet      string `json:"subnet,omitempty"`
	SubnetScope uint8  `json:"subnet_scope,omitempty"`
	// ExtendedErr is retained for JSON compatibility and contains the first
	// EDE option. ExtendedErrors is the authoritative ordered representation.
	ExtendedErr    string          `json:"extended_error,omitempty"`
	ExtendedErrors []ExtendedError `json:"extended_errors,omitempty"`
	UDPSize        uint16          `json:"udp_size,omitempty"`
	DNSSECOk       bool            `json:"dnssec_ok,omitempty"`
}

type ExtendedError struct {
	Code        uint16 `json:"code"`
	Description string `json:"description,omitempty"`
	ExtraText   string `json:"extra_text,omitempty"`
}

// LoadResolvers loads differently configured
// resolvers based on a list of nameserver.
func LoadResolvers(opts Options) ([]Resolver, error) {
	return loadResolvers(opts, newResolver)
}

type resolverFactory func(models.Nameserver, Options) (Resolver, error)

func loadResolvers(opts Options, factory resolverFactory) ([]Resolver, error) {
	if opts.UseHTTP3 {
		hasDOH := false
		for _, ns := range opts.Nameservers {
			if ns.Type == models.DOHResolver {
				hasDOH = true
				break
			}
		}
		if !hasDOH {
			return nil, fmt.Errorf("HTTP/3 requires at least one HTTPS (DoH) nameserver")
		}
	}

	// For each nameserver, initialise the correct resolver.
	rslvrs := make([]Resolver, 0, len(opts.Nameservers))

	for _, ns := range opts.Nameservers {
		rslvr, err := factory(ns, opts)
		if err != nil {
			if closeErr := CloseResolvers(rslvrs); closeErr != nil {
				err = errors.Join(err, closeErr)
			}
			return nil, err
		}
		if rslvr != nil {
			rslvrs = append(rslvrs, rslvr)
		}
	}
	return rslvrs, nil
}

func newResolver(ns models.Nameserver, opts Options) (Resolver, error) {
	switch ns.Type {
	case models.DOHResolver:
		opts.Logger.Debug("initiating DOH resolver")
		return NewDOHResolver(ns.Address, opts)
	case models.DOTResolver:
		opts.Logger.Debug("initiating DOT resolver")
		return NewClassicResolver(ns.Address, ClassicResolverOpts{UseTLS: true, UseTCP: true}, opts)
	case models.TCPResolver:
		opts.Logger.Debug("initiating TCP resolver")
		return NewClassicResolver(ns.Address, ClassicResolverOpts{UseTCP: true}, opts)
	case models.UDPResolver:
		opts.Logger.Debug("initiating UDP resolver")
		return NewClassicResolver(ns.Address, ClassicResolverOpts{}, opts)
	case models.DNSCryptResolver:
		opts.Logger.Debug("initiating DNSCrypt resolver")
		return NewDNSCryptResolver(ns.Address, DNSCryptResolverOpts{}, opts)
	case models.DOQResolver:
		opts.Logger.Debug("initiating DOQ resolver")
		return NewDOQResolver(ns.Address, opts)
	default:
		return nil, nil
	}
}
