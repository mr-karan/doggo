package app

import (
	"fmt"
	"math/rand"
	"net"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/ameshkov/dnsstamps"
	"github.com/miekg/dns"
	"github.com/mr-karan/doggo/pkg/config"
	"github.com/mr-karan/doggo/pkg/models"
)

func (app *App) LoadNameservers() error {
	app.Logger.Debug("LoadNameservers: Initial nameservers", "nameservers", app.QueryFlags.Nameservers)

	app.Nameservers = []models.Nameserver{} // Clear existing nameservers

	if len(app.QueryFlags.Nameservers) > 0 {
		for _, srv := range app.QueryFlags.Nameservers {
			ns, err := initNameserver(srv)
			if err != nil {
				app.Logger.Error("error parsing nameserver", "error", err)
				return fmt.Errorf("error parsing nameserver: %s", srv)
			}
			if ns.Address != "" && ns.Type != "" {
				app.Nameservers = append(app.Nameservers, ns)
				app.Logger.Debug("Added nameserver", "nameserver", ns)
			}
		}

		var err error
		app.Nameservers, err = app.applyNameserverStrategy(app.Nameservers, "explicit")
		if err != nil {
			app.Logger.Error("error applying nameserver strategy", "error", err)
			return err
		}
	}

	// If no nameservers were successfully loaded, check for authoritative flag
	if len(app.Nameservers) == 0 {
		if app.QueryFlags.UseAuthoritative && len(app.QueryFlags.QNames) > 0 {
			return app.loadAuthoritativeNameserver(app.QueryFlags.QNames[0])
		}
		return app.loadSystemNameservers()
	}

	app.Logger.Debug("LoadNameservers: Final nameservers", "nameservers", app.Nameservers)
	return nil
}

func (app *App) loadSystemNameservers() error {
	app.Logger.Debug("No user specified nameservers, falling back to system nameservers")
	ns, ndots, search, err := app.getDefaultServers()
	if err != nil {
		app.Logger.Error("error fetching system default nameserver", "error", err)
		return fmt.Errorf("error fetching system default nameserver: %v", err)
	}

	if app.ResolverOpts.Ndots == -1 {
		app.ResolverOpts.Ndots = ndots
	}

	if len(search) > 0 && app.QueryFlags.UseSearchList {
		app.ResolverOpts.SearchList = search
	}

	app.Nameservers = append(app.Nameservers, ns...)
	app.Logger.Debug("Loaded system nameservers", "nameservers", app.Nameservers)
	return nil
}

// wrapIPv6 wraps bare IPv6 addresses in brackets for URL parsing.
// This allows users to specify IPv6 addresses without brackets, like dig does.
// Examples:
//   - "2606:4700:4700::1111" -> "[2606:4700:4700::1111]"
//   - "fe80::1%eth0" -> "[fe80::1%eth0]"
//   - "[2606::1]" -> "[2606::1]" (already bracketed, no change)
//   - "8.8.8.8" -> "8.8.8.8" (IPv4, no change)
func wrapIPv6(addr string) string {
	// Already has brackets, no need to wrap
	if strings.HasPrefix(addr, "[") && strings.HasSuffix(addr, "]") {
		return addr
	}

	// Must contain colons to be IPv6
	if !strings.Contains(addr, ":") {
		return addr
	}

	// Extract host part (handle potential zone identifier like fe80::1%eth0)
	host := addr
	zoneIndex := strings.Index(addr, "%")

	// Try parsing the IP (without zone if present)
	var ipToParse string
	if zoneIndex != -1 {
		ipToParse = addr[:zoneIndex]
	} else {
		ipToParse = addr
	}

	// Parse as IP address
	ip := net.ParseIP(ipToParse)
	if ip == nil {
		// Not a valid IP, return as is
		return addr
	}

	// Check if it's IPv6 (not IPv4)
	if ip.To4() != nil {
		// It's IPv4, return as is
		return addr
	}

	// It's IPv6, wrap in brackets
	return "[" + host + "]"
}

// encodeZoneID URL-encodes the zone identifier in IPv6 addresses.
// Zone identifiers use % which must be percent-encoded as %25 for URL parsing.
// Example: "[fe80::1%eth0]" -> "[fe80::1%25eth0]"
func encodeZoneID(addr string) string {
	// Only process if we have brackets and a % inside them
	if !strings.Contains(addr, "[") || !strings.Contains(addr, "%") {
		return addr
	}

	// Find the zone identifier (% inside brackets)
	start := strings.Index(addr, "[")
	end := strings.Index(addr, "]")
	if start == -1 || end == -1 || start >= end {
		return addr
	}

	// Extract the bracketed part
	bracketed := addr[start+1 : end]
	if !strings.Contains(bracketed, "%") {
		return addr
	}

	// Replace % with %25 in the bracketed part
	encoded := strings.ReplaceAll(bracketed, "%", "%25")

	// Reconstruct the address
	return addr[:start+1] + encoded + addr[end:]
}

// wrapIPv6InURL wraps IPv6 addresses in URLs that already have a protocol.
// Example: "tcp://2606:4700:4700::1111" -> "tcp://[2606:4700:4700::1111]"
func wrapIPv6InURL(urlStr string) string {
	// Split by :// to separate protocol from host
	parts := strings.SplitN(urlStr, "://", 2)
	if len(parts) != 2 {
		return urlStr
	}

	protocol := parts[0]
	hostPart := parts[1]

	// For HTTPS URLs, don't try to wrap (they have domain names, not IPs usually)
	if protocol == "https" || protocol == "sdns" {
		return urlStr
	}

	// Wrap the host part if it's IPv6
	wrappedHost := wrapIPv6(hostPart)
	// Encode zone identifiers
	wrappedHost = encodeZoneID(wrappedHost)

	return protocol + "://" + wrappedHost
}

func initNameserver(n string) (models.Nameserver, error) {
	// If the nameserver doesn't have a protocol, assume it's UDP
	if !strings.Contains(n, "://") {
		// Wrap bare IPv6 addresses in brackets for proper URL parsing
		n = wrapIPv6(n)
		// URL-encode zone identifiers (%) for proper parsing
		n = encodeZoneID(n)
		n = "udp://" + n
	} else {
		// Protocol is present, but we still need to wrap IPv6 addresses in the host part
		n = wrapIPv6InURL(n)
	}

	u, err := url.Parse(n)
	if err != nil {
		return models.Nameserver{}, err
	}

	ns := models.Nameserver{
		Type:    models.UDPResolver,
		Address: getAddressWithDefaultPort(u, models.DefaultUDPPort),
	}

	switch u.Scheme {
	case "sdns":
		return handleSDNS(n)
	case "https":
		ns.Type = models.DOHResolver
		ns.Address = u.String()
	case "tls":
		ns.Type = models.DOTResolver
		ns.Address = getAddressWithDefaultPort(u, models.DefaultTLSPort)
	case "tcp":
		ns.Type = models.TCPResolver
		ns.Address = getAddressWithDefaultPort(u, models.DefaultTCPPort)
	case "udp":
		ns.Type = models.UDPResolver
		ns.Address = getAddressWithDefaultPort(u, models.DefaultUDPPort)
	case "quic":
		ns.Type = models.DOQResolver
		ns.Address = getAddressWithDefaultPort(u, models.DefaultDOQPort)
	default:
		return ns, fmt.Errorf("unsupported protocol: %s", u.Scheme)
	}

	return ns, nil
}

func getAddressWithDefaultPort(u *url.URL, defaultPort string) string {
	host := u.Hostname()
	port := u.Port()
	if port == "" {
		port = defaultPort
	}
	return net.JoinHostPort(host, port)
}

func handleSDNS(n string) (models.Nameserver, error) {
	stamp, err := dnsstamps.NewServerStampFromString(n)
	if err != nil {
		return models.Nameserver{}, err
	}

	switch stamp.Proto {
	case dnsstamps.StampProtoTypeDoH:
		address := url.URL{Scheme: "https", Host: stamp.ProviderName, Path: stamp.Path}
		return models.Nameserver{
			Type:    models.DOHResolver,
			Address: address.String(),
		}, nil
	case dnsstamps.StampProtoTypeDNSCrypt:
		return models.Nameserver{
			Type:    models.DNSCryptResolver,
			Address: n,
		}, nil
	default:
		return models.Nameserver{}, fmt.Errorf("unsupported protocol: %v", stamp.Proto.String())
	}
}

// isIPv4 checks if an IP address string is IPv4
func isIPv4(ipStr string) bool {
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return false
	}
	return ip.To4() != nil
}

// isIPv6 checks if an IP address string is IPv6
func isIPv6(ipStr string) bool {
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return false
	}
	return ip.To4() == nil
}

// isPrivateIP reports whether an IP address belongs to a private/internal
// range: RFC 1918 (IPv4), RFC 6598 Carrier-Grade NAT (IPv4, used by Tailscale's
// 100.100.100.100 MagicDNS resolver), or RFC 4193 Unique Local Addresses (IPv6).
func isPrivateIP(ipStr string) bool {
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return false
	}

	// IPv4 ranges
	if ipv4 := ip.To4(); ipv4 != nil {
		// 10.0.0.0/8
		if ipv4[0] == 10 {
			return true
		}
		// 172.16.0.0/12
		if ipv4[0] == 172 && ipv4[1] >= 16 && ipv4[1] <= 31 {
			return true
		}
		// 192.168.0.0/16
		if ipv4[0] == 192 && ipv4[1] == 168 {
			return true
		}
		// 100.64.0.0/10 (RFC 6598 CGNAT, e.g. Tailscale)
		if ipv4[0] == 100 && ipv4[1] >= 64 && ipv4[1] <= 127 {
			return true
		}
		return false
	}

	// IPv6 Unique Local Address (ULA) - RFC 4193
	// fd00::/8 range
	if len(ip) == 16 && ip[0] == 0xfd {
		return true
	}

	return false
}

// filterNameserversByIPVersion filters nameservers based on IPv4/IPv6 flags
func filterNameserversByIPVersion(servers []string, useIPv4, useIPv6 bool) []string {
	// If neither flag is set, return all servers
	if !useIPv4 && !useIPv6 {
		return servers
	}

	filtered := make([]string, 0, len(servers))
	for _, srv := range servers {
		if useIPv4 && isIPv4(srv) {
			filtered = append(filtered, srv)
		} else if useIPv6 && isIPv6(srv) {
			filtered = append(filtered, srv)
		}
	}

	return filtered
}

// applyNameserverStrategy narrows the supplied nameserver list according to
// app.QueryFlags.Strategy and emits debug logs describing what it did. The
// source argument labels the origin of the list (e.g. "explicit" for CLI
// overrides, "system" for resolv.conf) so the same log lines can describe
// both call sites without ambiguity.
func (app *App) applyNameserverStrategy(nameservers []models.Nameserver, source string) ([]models.Nameserver, error) {
	if len(nameservers) == 0 {
		return nameservers, nil
	}

	strategy := app.QueryFlags.Strategy
	app.Logger.Debug("Applying nameserver strategy",
		"source", source,
		"strategy", strategy,
		"before_count", len(nameservers),
		"before", nameservers,
	)

	switch strategy {
	case "random":
		src := rand.NewSource(time.Now().UnixNano())
		rnd := rand.New(src)
		selected := []models.Nameserver{nameservers[rnd.Intn(len(nameservers))]}
		app.logStrategyApplied(source, strategy, len(nameservers), selected)
		return selected, nil

	case "first":
		selected := []models.Nameserver{nameservers[0]}
		app.logStrategyApplied(source, strategy, len(nameservers), selected)
		return selected, nil

	case "internal":
		internalServers := make([]models.Nameserver, 0)
		for _, ns := range nameservers {
			if isPrivateIP(nameserverHost(ns)) {
				internalServers = append(internalServers, ns)
			}
		}

		if len(internalServers) == 0 {
			app.Logger.Debug("Nameserver strategy rejected all nameservers",
				"source", source,
				"strategy", strategy,
				"before_count", len(nameservers),
			)
			return nil, fmt.Errorf("no internal (private IP) nameservers found")
		}

		app.logStrategyApplied(source, strategy, len(nameservers), internalServers)
		return internalServers, nil

	default:
		app.Logger.Debug("Nameserver strategy left nameservers unchanged",
			"source", source,
			"strategy", strategy,
			"count", len(nameservers),
		)
		return nameservers, nil
	}
}

func (app *App) logStrategyApplied(source, strategy string, beforeCount int, after []models.Nameserver) {
	app.Logger.Debug("Applied nameserver strategy",
		"source", source,
		"strategy", strategy,
		"after_count", len(after),
		"dropped_count", beforeCount-len(after),
		"after", after,
	)
}

func nameserverHost(ns models.Nameserver) string {
	if u, err := url.Parse(ns.Address); err == nil && u.Hostname() != "" {
		return u.Hostname()
	}

	host, _, err := net.SplitHostPort(ns.Address)
	if err == nil {
		return host
	}

	return ns.Address
}

func (app *App) getDefaultServers() ([]models.Nameserver, int, []string, error) {
	// Load nameservers from the system resolver configuration. The "internal"
	// strategy needs to see Supplemental/domain-scoped resolvers (e.g. a VPN or
	// Tailscale split-DNS resolver), which GetDefaultServers hides, so it sources
	// the broader GetAllServers list instead.
	loadServers := config.GetDefaultServers
	if app.QueryFlags.Strategy == "internal" {
		loadServers = config.GetAllServers
	}

	dnsServers, ndots, search, err := loadServers()
	if err != nil {
		return nil, 0, nil, err
	}

	app.Logger.Debug("Loaded system resolver configuration",
		"nameservers", dnsServers,
		"ndots", ndots,
		"search", search,
	)

	// Filter nameservers based on IPv4/IPv6 flags
	beforeFilter := len(dnsServers)
	dnsServers = filterNameserversByIPVersion(dnsServers, app.QueryFlags.UseIPv4, app.QueryFlags.UseIPv6)
	if beforeFilter != len(dnsServers) {
		app.Logger.Debug("Filtered system nameservers by IP version",
			"use_ipv4", app.QueryFlags.UseIPv4,
			"use_ipv6", app.QueryFlags.UseIPv6,
			"before_count", beforeFilter,
			"after_count", len(dnsServers),
		)
	}

	// If after filtering we have no servers, return an error
	if len(dnsServers) == 0 {
		ipVersion := "IPv4"
		if app.QueryFlags.UseIPv6 {
			ipVersion = "IPv6"
		}
		return nil, ndots, search, fmt.Errorf("no %s nameservers found in system configuration", ipVersion)
	}

	servers := make([]models.Nameserver, 0, len(dnsServers))
	for _, s := range dnsServers {
		ns := models.Nameserver{
			Type:    models.UDPResolver,
			Address: net.JoinHostPort(s, models.DefaultUDPPort),
		}
		servers = append(servers, ns)
	}

	servers, err = app.applyNameserverStrategy(servers, "system")
	if err != nil {
		return nil, ndots, search, fmt.Errorf("%w in system configuration", err)
	}

	return servers, ndots, search, nil
}

// loadAuthoritativeNameserver finds the closest enclosing zone for the domain
// via SOA queries, then resolves that zone's delegated NS RRset and adds those
// nameservers to app.Nameservers.
//
// SOA is used only to locate the zone cut. The actual query targets come from
// the zone's NS records, which are the publicly delegated authoritative
// servers. We deliberately avoid SOA.Ns (the zone primary/MNAME): it is often
// an internal hostname that does not resolve publicly (e.g. amazon.com's MNAME
// is dns-external-route53.us-east-1.amazonaws.com), whereas the delegated NS
// set is what recursive resolvers actually query.
func (app *App) loadAuthoritativeNameserver(domain string) error {
	systemServers, _, _, err := config.GetDefaultServers()
	if err != nil || len(systemServers) == 0 {
		return fmt.Errorf("unable to load system nameservers for SOA lookup: %w", err)
	}
	resolver := net.JoinHostPort(systemServers[0], models.DefaultUDPPort)

	c := &dns.Client{Timeout: 5 * time.Second}

	// Step 1: use SOA to identify the closest enclosing zone (the zone cut).
	zone, err := app.closestZone(c, resolver, dns.Fqdn(domain))
	if err != nil {
		return err
	}

	// Step 2: fetch that zone's delegated NS RRset — the public authoritative servers.
	nsNames, err := app.zoneNameservers(c, resolver, zone)
	if err != nil {
		return err
	}

	servers := make([]models.Nameserver, 0, len(nsNames))
	for _, name := range nsNames {
		ns, err := initNameserver(strings.TrimSuffix(name, "."))
		if err != nil {
			app.Logger.Debug("Skipping invalid authoritative nameserver", "ns", name, "error", err)
			continue
		}
		servers = append(servers, ns)
	}
	if len(servers) == 0 {
		return fmt.Errorf("no usable authoritative nameservers found for zone %q", strings.TrimSuffix(zone, "."))
	}

	app.Logger.Debug("Resolved authoritative nameservers via NS RRset", "zone", zone, "nameservers", servers)

	// Step 3: let the nameserver strategy select the query target(s).
	servers, err = app.applyNameserverStrategy(servers, "authoritative")
	if err != nil {
		return err
	}

	app.Nameservers = append(app.Nameservers, servers...)
	return nil
}

// closestZone walks up the domain hierarchy issuing SOA queries until it finds
// the closest enclosing zone, returning that zone's apex as an FQDN.
func (app *App) closestZone(c *dns.Client, resolver, candidate string) (string, error) {
	domain := candidate
	for {
		m := new(dns.Msg)
		m.SetQuestion(candidate, dns.TypeSOA)
		m.RecursionDesired = true

		r, _, err := c.Exchange(m, resolver)
		if err == nil {
			if soa := firstSOA(r); soa != nil {
				// The SOA owner name is the zone apex regardless of the label
				// we queried: a recursive resolver returns the enclosing zone's
				// SOA in the authority section for sub-zone names.
				return soa.Hdr.Name, nil
			}
		}

		// No SOA found — walk up to the parent zone.
		labels := dns.SplitDomainName(candidate)
		if len(labels) <= 1 {
			return "", fmt.Errorf("no authoritative zone found for %q", strings.TrimSuffix(domain, "."))
		}
		candidate = dns.Fqdn(strings.Join(labels[1:], "."))
	}
}

// zoneNameservers queries the NS RRset for the given zone and returns the
// delegated nameserver hostnames, sorted for deterministic selection.
func (app *App) zoneNameservers(c *dns.Client, resolver, zone string) ([]string, error) {
	m := new(dns.Msg)
	m.SetQuestion(zone, dns.TypeNS)
	m.RecursionDesired = true

	r, _, err := c.Exchange(m, resolver)
	if err != nil {
		return nil, fmt.Errorf("failed to query NS records for zone %q: %w", strings.TrimSuffix(zone, "."), err)
	}

	var names []string
	for _, rr := range append(r.Answer, r.Ns...) {
		if ns, ok := rr.(*dns.NS); ok {
			names = append(names, ns.Ns)
		}
	}
	if len(names) == 0 {
		return nil, fmt.Errorf("no NS records found for zone %q", strings.TrimSuffix(zone, "."))
	}
	sort.Strings(names)

	return names, nil
}

// firstSOA returns the first SOA record from the answer or authority section.
func firstSOA(r *dns.Msg) *dns.SOA {
	for _, rr := range append(r.Answer, r.Ns...) {
		if soa, ok := rr.(*dns.SOA); ok {
			return soa
		}
	}
	return nil
}
