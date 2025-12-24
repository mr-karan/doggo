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
	domain        string
	searchDomains []string
	options       []string
	flags         []string
}

// GetDefaultServers retrieves DNS configuration from macOS SystemConfiguration
// by parsing the output of 'scutil --dns'. Falls back to /etc/resolv.conf on failure.
func GetDefaultServers() ([]string, int, []string, error) {
	// Try scutil first
	resolvers, ndots, search, err := getResolversFromScutil()
	if err != nil {
		// Fallback to /etc/resolv.conf
		return fallbackToResolvConf()
	}

	return resolvers, ndots, search, nil
}

// getResolversFromScutil executes scutil --dns and parses the output
func getResolversFromScutil() ([]string, int, []string, error) {
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

	// Filter out resolvers that shouldn't be used for general queries:
	// - mDNS resolvers (for .local domains)
	// - Supplemental resolvers (flagged as domain-specific)
	// - Domain-specific resolvers (have explicit domain field)
	validResolvers := make([]scutilResolver, 0)
	for _, r := range resolvers {
		if !isMDNS(r) && !isSupplemental(r) && !isDomainSpecific(r) && len(r.nameservers) > 0 {
			validResolvers = append(validResolvers, r)
		}
	}

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
