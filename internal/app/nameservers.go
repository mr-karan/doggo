package app

import (
	"fmt"
	"math/rand"
	"net"
	"net/url"
	"time"

	"github.com/ameshkov/dnsstamps"
	"github.com/mr-karan/doggo/pkg/config"
	"github.com/mr-karan/doggo/pkg/models"
)

// LoadNameservers reads all the user given
// nameservers and loads to App.
func (app *App) LoadNameservers() error {
	for _, srv := range app.QueryFlags.Nameservers {
		ns, err := initNameserver(srv)
		if err != nil {
			return fmt.Errorf("error parsing nameserver: %s", srv)
		}
		// check if properly initialised.
		if ns.Address != "" && ns.Type != "" {
			app.Nameservers = append(app.Nameservers, ns)
		}
	}

	// Set `ndots` to the user specified value.
	app.ResolverOpts.Ndots = app.QueryFlags.Ndots
	// fallback to system nameserver
	// in case no nameserver is specified by user.
	if len(app.Nameservers) == 0 {
		ns, ndots, search, err := getDefaultServers(app.QueryFlags.Strategy)
		if err != nil {
			return fmt.Errorf("error fetching system default nameserver")
		}
		// `-1` indicates the flag is not set.
		// use from config if user hasn't specified any value.
		if app.ResolverOpts.Ndots == -1 {
			app.ResolverOpts.Ndots = ndots
		}
		if len(search) > 0 && app.QueryFlags.UseSearchList {
			app.ResolverOpts.SearchList = search
		}
		app.Nameservers = append(app.Nameservers, ns...)
	}
	// if the user hasn't given any override of `ndots` AND has
	// given a custom nameserver. Set `ndots` to 1 as the fallback value
	if app.ResolverOpts.Ndots == -1 {
		app.ResolverOpts.Ndots = 0
	}
	return nil
}

func initNameserver(n string) (models.Nameserver, error) {
	// Instantiate a UDP resolver with default port as a fallback.
	ns := models.Nameserver{
		Type:    models.UDPResolver,
		Address: net.JoinHostPort(n, models.DefaultUDPPort),
	}
	u, err := url.Parse(n)
	if err != nil {
		return ns, err
	}
	switch u.Scheme {
	case "sdns":
		stamp, err := dnsstamps.NewServerStampFromString(n)
		if err != nil {
			return ns, err
		}
		switch stamp.Proto {
		case dnsstamps.StampProtoTypeDoH:
			ns.Type = models.DOHResolver
			address := url.URL{Scheme: "https", Host: stamp.ProviderName, Path: stamp.Path}
			ns.Address = address.String()
		case dnsstamps.StampProtoTypeDNSCrypt:
			ns.Type = models.DNSCryptResolver
			ns.Address = n
		default:
			return ns, fmt.Errorf("unsupported protocol: %v", stamp.Proto.String())
		}

	case "https":
		ns.Type = models.DOHResolver
		ns.Address = u.String()

	case "tls":
		ns.Type = models.DOTResolver
		if u.Port() == "" {
			ns.Address = net.JoinHostPort(u.Hostname(), models.DefaultTLSPort)
		} else {
			ns.Address = net.JoinHostPort(u.Hostname(), u.Port())
		}

	case "tcp":
		ns.Type = models.TCPResolver
		if u.Port() == "" {
			ns.Address = net.JoinHostPort(u.Hostname(), models.DefaultTCPPort)
		} else {
			ns.Address = net.JoinHostPort(u.Hostname(), u.Port())
		}

	case "udp":
		ns.Type = models.UDPResolver
		if u.Port() == "" {
			ns.Address = net.JoinHostPort(u.Hostname(), models.DefaultUDPPort)
		} else {
			ns.Address = net.JoinHostPort(u.Hostname(), u.Port())
		}
	case "quic":
		ns.Type = models.DOQResolver
		if u.Port() == "" {
			ns.Address = net.JoinHostPort(u.Hostname(), models.DefaultDOQPort)
		} else {
			ns.Address = net.JoinHostPort(u.Hostname(), u.Port())
		}
	}
	return ns, nil
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
		// Choose a random server from the list.
		rand.Seed(time.Now().Unix())
		srv := dnsServers[rand.Intn(len(dnsServers))]
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
