//go:build darwin
// +build darwin

package config

import (
	"bytes"
	"fmt"
	"net"
	"os/exec"
	"regexp"
	"strconv"
	"strings"

	"github.com/miekg/dns"
)

// scutilResolver represents a parsed resolver from scutil --dns output
type scutilResolver struct {
	number        int
	nameservers   []string
	port          int
	domain        string
	searchDomains []string
	options       []string
	flags         []string
}

// GetDefaultServers retrieves DNS configuration from macOS SystemConfiguration
// by parsing the output of 'scutil --dns'. Falls back to /etc/resolv.conf on failure.
//
// Only general-purpose resolvers are returned: Supplemental and domain-scoped
// resolvers are excluded because they apply to specific domains, not arbitrary
// queries. See GetAllServers for the strategy-aware variant.
func GetDefaultServers() ([]string, int, []string, error) {
	return getSystemServers(false)
}

// GetAllServers is like GetDefaultServers but also includes Supplemental and
// domain-scoped resolvers (e.g. the resolvers a VPN or Tailscale installs for
// split-DNS). It exists so the "internal" nameserver strategy can discover
// private corporate/VPN resolvers that GetDefaultServers intentionally hides.
// mDNS resolvers (.local and reverse-DNS) are still excluded.
func GetAllServers() ([]string, int, []string, error) {
	return getSystemServers(true)
}

// MatchDomainNameservers selects nameservers for query names that fall under a
// macOS Supplemental/domain-specific scutil resolver. Longest domain match wins
// per query name. ok is true only when every query name falls under such a
// resolver — a mix of matching and non-matching names stays on the
// general-purpose list so unrelated names are not sent to a split-DNS
// resolver. The interface-scoped "DNS configuration (for scoped queries)"
// section is never consulted.
func MatchDomainNameservers(queryNames []string) (nameservers []DomainNameserver, matchedDomains []string, ok bool) {
	if len(queryNames) == 0 {
		return nil, nil, false
	}

	cmd := exec.Command("scutil", "--dns")
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	if err := cmd.Run(); err != nil {
		return nil, nil, false
	}

	resolvers, err := parseScutilOutput(stdout.String())
	if err != nil {
		return nil, nil, false
	}

	return matchDomainNameservers(queryNames, resolvers)
}

// getSystemServers tries scutil first and falls back to /etc/resolv.conf.
// includeSupplemental controls whether Supplemental/domain-scoped resolvers
// are kept (see GetAllServers).
func getSystemServers(includeSupplemental bool) ([]string, int, []string, error) {
	// Try scutil first
	resolvers, ndots, search, err := getResolversFromScutil(includeSupplemental)
	if err != nil {
		// Fallback to /etc/resolv.conf
		return fallbackToResolvConf()
	}

	return resolvers, ndots, search, nil
}

// getResolversFromScutil executes scutil --dns and parses the output
func getResolversFromScutil(includeSupplemental bool) ([]string, int, []string, error) {
	// Execute scutil --dns
	cmd := exec.Command("scutil", "--dns")
	var stdout bytes.Buffer
	cmd.Stdout = &stdout

	if err := cmd.Run(); err != nil {
		return nil, 0, nil, fmt.Errorf("scutil execution failed: %w", err)
	}

	output := stdout.String()
	if len(strings.TrimSpace(output)) == 0 {
		return nil, 0, nil, fmt.Errorf("scutil returned empty output")
	}

	// Parse the output
	resolvers, err := parseScutilOutput(output)
	if err != nil {
		return nil, 0, nil, fmt.Errorf("failed to parse scutil output: %w", err)
	}

	validResolvers := filterResolvers(resolvers, includeSupplemental)

	if len(validResolvers) == 0 {
		return nil, 0, nil, fmt.Errorf("no valid resolvers found")
	}

	// Aggregate nameservers from all valid resolvers
	// This allows the "internal" strategy to find domain-specific corporate DNS servers
	nameservers := make([]string, 0)
	seen := make(map[string]bool)

	for _, resolver := range validResolvers {
		for _, ns := range resolver.nameservers {
			ip := net.ParseIP(ns)
			// Skip link-local and duplicates
			if isUnicastLinkLocal(ip) || seen[ns] {
				continue
			}
			nameservers = append(nameservers, ns)
			seen[ns] = true
		}
	}

	// Aggregate search domains from all valid resolvers
	searchDomains := aggregateSearchDomains(validResolvers)

	// ndots: try to read from /etc/resolv.conf, default to 1
	ndots := 1
	if cfg, err := dns.ClientConfigFromFile("/etc/resolv.conf"); err == nil {
		ndots = cfg.Ndots
	}

	return nameservers, ndots, searchDomains, nil
}

// parseScutilOutput parses the output of scutil --dns
// It only parses the main "DNS configuration" section and stops at
// "DNS configuration (for scoped queries)" since scoped resolvers
// are interface-specific and shouldn't be used for general queries.
func parseScutilOutput(output string) ([]scutilResolver, error) {
	lines := strings.Split(output, "\n")
	resolvers := make([]scutilResolver, 0)

	var current *scutilResolver
	resolverRe := regexp.MustCompile(`^resolver #(\d+)`)
	nameserverRe := regexp.MustCompile(`^\s+nameserver\[\d+\]\s*:\s*(.+)`)
	domainRe := regexp.MustCompile(`^\s+domain\s*:\s*(.+)`)
	searchDomainRe := regexp.MustCompile(`^\s+search domain\[\d+\]\s*:\s*(.+)`)
	optionsRe := regexp.MustCompile(`^\s+options\s*:\s*(.+)`)
	flagsRe := regexp.MustCompile(`^\s+flags\s*:\s*(.+)`)
	portRe := regexp.MustCompile(`^\s+port\s*:\s*(\d+)`)

	for _, line := range lines {
		if strings.Contains(line, "DNS configuration (for scoped queries)") {
			break
		}

		// Check for resolver start
		if matches := resolverRe.FindStringSubmatch(line); matches != nil {
			if current != nil {
				resolvers = append(resolvers, *current)
			}
			num, _ := strconv.Atoi(matches[1])
			current = &scutilResolver{
				number:        num,
				nameservers:   make([]string, 0),
				searchDomains: make([]string, 0),
				options:       make([]string, 0),
				flags:         make([]string, 0),
			}
			continue
		}

		if current == nil {
			continue
		}

		// Parse nameserver
		if matches := nameserverRe.FindStringSubmatch(line); matches != nil {
			current.nameservers = append(current.nameservers, strings.TrimSpace(matches[1]))
			continue
		}

		// Parse domain
		if matches := domainRe.FindStringSubmatch(line); matches != nil {
			current.domain = strings.TrimSpace(matches[1])
			continue
		}

		// Parse port (kSCPropNetDNSServerPort); custom ports are rare but
		// valid for split-DNS resolvers.
		if matches := portRe.FindStringSubmatch(line); matches != nil {
			p, _ := strconv.Atoi(matches[1])
			current.port = p
			continue
		}

		// Parse search domain
		if matches := searchDomainRe.FindStringSubmatch(line); matches != nil {
			current.searchDomains = append(current.searchDomains, strings.TrimSpace(matches[1]))
			continue
		}

		// Parse options
		if matches := optionsRe.FindStringSubmatch(line); matches != nil {
			opts := strings.Fields(strings.TrimSpace(matches[1]))
			current.options = append(current.options, opts...)
			continue
		}

		// Parse flags (comma-separated, e.g., "Supplemental, Request A records")
		if matches := flagsRe.FindStringSubmatch(line); matches != nil {
			flagStr := strings.TrimSpace(matches[1])
			for _, f := range strings.Split(flagStr, ",") {
				current.flags = append(current.flags, strings.TrimSpace(f))
			}
			continue
		}
	}

	// Don't forget the last resolver
	if current != nil {
		resolvers = append(resolvers, *current)
	}

	return resolvers, nil
}

// filterResolvers selects resolvers usable for outbound DNS queries.
//
// mDNS resolvers (.local and reverse-DNS) and resolvers with no nameservers are
// always excluded. By default Supplemental and domain-scoped resolvers are also
// excluded, since they apply only to specific domains rather than general
// queries. When includeSupplemental is set they are kept, so the "internal"
// strategy can discover private VPN/Tailscale resolvers that would otherwise be
// hidden.
func filterResolvers(resolvers []scutilResolver, includeSupplemental bool) []scutilResolver {
	validResolvers := make([]scutilResolver, 0)
	for _, r := range resolvers {
		if isMDNS(r) || len(r.nameservers) == 0 {
			continue
		}
		if !includeSupplemental && (isSupplemental(r) || isDomainSpecific(r)) {
			continue
		}
		validResolvers = append(validResolvers, r)
	}
	return validResolvers
}

// isMDNS checks if a resolver is for mDNS (.local)
func isMDNS(r scutilResolver) bool {
	for _, opt := range r.options {
		if opt == "mdns" {
			return true
		}
	}
	return false
}

func isSupplemental(r scutilResolver) bool {
	for _, flag := range r.flags {
		if flag == "Supplemental" {
			return true
		}
	}
	return false
}

// isDomainSpecific checks if a resolver is configured for a specific domain only.
// Per scutil(8): "Those supplemental configurations containing a 'domain' name
// will be used for queries matching the specified domain."
// These should NOT be used for general DNS queries.
func isDomainSpecific(r scutilResolver) bool {
	return r.domain != ""
}

// matchDomainNameservers picks domain-specific resolvers whose domain is a
// suffix of a query name. Longest match wins per name; nameservers from all
// winning matches are unioned (order preserved, duplicates dropped). The
// match is all-or-nothing: if any query name matches no domain resolver, or
// different names match different domains (doggo routes all questions through
// one nameserver list and cannot partition per question), ok is false and the
// caller falls back to the general system resolvers.
func matchDomainNameservers(queryNames []string, resolvers []scutilResolver) ([]DomainNameserver, []string, bool) {
	domainResolvers := make([]scutilResolver, 0)
	for _, r := range resolvers {
		if isMDNS(r) || !isDomainSpecific(r) || len(r.nameservers) == 0 {
			continue
		}
		domainResolvers = append(domainResolvers, r)
	}
	if len(domainResolvers) == 0 {
		return nil, nil, false
	}

	seenNS := make(map[DomainNameserver]bool)
	seenDomain := make(map[string]bool)
	nameservers := make([]DomainNameserver, 0)
	matchedDomains := make([]string, 0)

	for _, qname := range queryNames {
		bestDomain := ""
		for _, r := range domainResolvers {
			domain := matchableDomain(r.domain)
			if domain == "" || !nameUnderDomain(qname, domain) {
				continue
			}
			if len(domain) > len(bestDomain) {
				bestDomain = domain
			}
		}
		if bestDomain == "" {
			// This name belongs to no split-DNS domain; forcing it through
			// another name's domain resolver could break public resolution.
			return nil, nil, false
		}

		if len(matchedDomains) > 0 && matchedDomains[0] != bestDomain {
			// Names from different split-DNS domains would need per-question
			// resolver routing, which doggo does not do; fall back to the
			// general list instead of leaking queries across domains.
			return nil, nil, false
		}
		if !seenDomain[bestDomain] {
			matchedDomains = append(matchedDomains, bestDomain)
			seenDomain[bestDomain] = true
		}
		// macOS can list several resolver records for the same domain (for
		// example one per VPN interface); union their nameservers in scutil
		// order so none of the domain's resolvers is silently dropped.
		for _, r := range domainResolvers {
			if matchableDomain(r.domain) != bestDomain {
				continue
			}
			for _, ns := range r.nameservers {
				ip := net.ParseIP(ns)
				entry := DomainNameserver{IP: ns, Port: r.port}
				// Deduplicate on IP and effective port (0 means the default
				// DNS port): the same IP on two ports is a distinct
				// endpoint, but an omitted port and an explicit "53" are the
				// same endpoint.
				dedupKey := entry
				if dedupKey.Port == 0 {
					dedupKey.Port = defaultDNSPort
				}
				if isUnicastLinkLocal(ip) || seenNS[dedupKey] {
					continue
				}
				nameservers = append(nameservers, entry)
				seenNS[dedupKey] = true
			}
		}
	}

	if len(nameservers) == 0 {
		return nil, nil, false
	}
	return nameservers, matchedDomains, true
}

func normalizeDNSName(name string) string {
	return strings.TrimSuffix(strings.ToLower(strings.TrimSpace(name)), ".")
}

// matchableDomain normalizes a scutil domain entry for matching. Apple DNS
// settings allow wildcard match domains like "*.example.com"; the leading
// wildcard label is stripped so subdomains and the apex both match, matching
// macOS resolver behavior.
func matchableDomain(domain string) string {
	return strings.TrimPrefix(normalizeDNSName(domain), "*.")
}

// nameUnderDomain reports whether name is equal to domain or a subdomain of it.
func nameUnderDomain(name, domain string) bool {
	name = normalizeDNSName(name)
	domain = normalizeDNSName(domain)
	if name == "" || domain == "" {
		return false
	}
	if name == domain {
		return true
	}
	return strings.HasSuffix(name, "."+domain)
}

// aggregateSearchDomains collects search domains from all resolvers
func aggregateSearchDomains(resolvers []scutilResolver) []string {
	seen := make(map[string]bool)
	result := make([]string, 0)

	for _, r := range resolvers {
		// Add domain if present
		if r.domain != "" && !seen[r.domain] {
			result = append(result, r.domain)
			seen[r.domain] = true
		}

		// Add search domains
		for _, sd := range r.searchDomains {
			if !seen[sd] {
				result = append(result, sd)
				seen[sd] = true
			}
		}
	}

	return result
}

// fallbackToResolvConf falls back to the traditional /etc/resolv.conf
func fallbackToResolvConf() ([]string, int, []string, error) {
	cfg, err := dns.ClientConfigFromFile("/etc/resolv.conf")
	if err != nil {
		return nil, 0, nil, err
	}

	servers := make([]string, 0)
	for _, server := range cfg.Servers {
		ip := net.ParseIP(server)
		if isUnicastLinkLocal(ip) {
			continue
		}
		servers = append(servers, server)
	}

	return servers, cfg.Ndots, cfg.Search, nil
}
