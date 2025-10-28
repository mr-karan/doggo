package app

import (
	"fmt"
	"math/rand"
	"net"
	"net/url"
	"strings"
	"time"

	"github.com/ameshkov/dnsstamps"
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
	}

	// If no nameservers were successfully loaded, fall back to system nameservers
	if len(app.Nameservers) == 0 {
		return app.loadSystemNameservers()
	}

	app.Logger.Debug("LoadNameservers: Final nameservers", "nameservers", app.Nameservers)
	return nil
}

func (app *App) loadSystemNameservers() error {
	app.Logger.Debug("No user specified nameservers, falling back to system nameservers")
	ns, ndots, search, err := getDefaultServers(app.QueryFlags.Strategy)
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

func getDefaultServers(strategy string) ([]models.Nameserver, int, []string, error) {
	// Load nameservers from `/etc/resolv.conf`.
	dnsServers, ndots, search, err := config.GetDefaultServers()
	if err != nil {
		return nil, 0, nil, err
	}
	servers := make([]models.Nameserver, 0, len(dnsServers))

	switch strategy {
	case "random":
		// Create a new local random source and generator.
		src := rand.NewSource(time.Now().UnixNano())
		rnd := rand.New(src)

		// Choose a random server from the list.
		srv := dnsServers[rnd.Intn(len(dnsServers))]
		ns := models.Nameserver{
			Type:    models.UDPResolver,
			Address: net.JoinHostPort(srv, models.DefaultUDPPort),
		}
		servers = append(servers, ns)

	case "first":
		// Choose the first from the list, always.
		srv := dnsServers[0]
		ns := models.Nameserver{
			Type:    models.UDPResolver,
			Address: net.JoinHostPort(srv, models.DefaultUDPPort),
		}
		servers = append(servers, ns)

	default:
		// Default behaviour is to load all nameservers.
		for _, s := range dnsServers {
			ns := models.Nameserver{
				Type:    models.UDPResolver,
				Address: net.JoinHostPort(s, models.DefaultUDPPort),
			}
			servers = append(servers, ns)
		}
	}

	return servers, ndots, search, nil
}
