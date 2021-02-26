package main

import (
	"fmt"
	"net"
	"net/url"

	"github.com/mr-karan/doggo/pkg/config"
	"github.com/mr-karan/doggo/pkg/models"
)

// loadNameservers reads all the user given
// nameservers and loads to Hub.
func (hub *Hub) loadNameservers() error {
	for _, srv := range hub.QueryFlags.Nameservers {
		ns, err := initNameserver(srv)
		if err != nil {
			return fmt.Errorf("error parsing nameserver: %s", srv)
		}
		// check if properly initialised.
		if ns.Address != "" && ns.Type != "" {
			hub.Nameservers = append(hub.Nameservers, ns)
		}
	}

	// Set `ndots` to the user specified value.
	hub.ResolverOpts.Ndots = hub.QueryFlags.Ndots
	// fallback to system nameserver
	// in case no nameserver is specified by user.
	if len(hub.Nameservers) == 0 {
		ns, ndots, search, err := getDefaultServers()
		if err != nil {
			return fmt.Errorf("error fetching system default nameserver")
		}
		// `-1` indicates the flag is not set.
		// use from config if user hasn't specified any value.
		if hub.ResolverOpts.Ndots == -1 {
			hub.ResolverOpts.Ndots = ndots
		}
		if len(search) > 0 && hub.QueryFlags.UseSearchList {
			hub.ResolverOpts.SearchList = search
		}
		hub.Nameservers = append(hub.Nameservers, ns...)
	}
	// if the user hasn't given any override of `ndots` AND has
	// given a custom nameserver. Set `ndots` to 1 as the fallback value
	if hub.ResolverOpts.Ndots == -1 {
		hub.ResolverOpts.Ndots = 0
	}
	return nil
}

func getDefaultServers() ([]models.Nameserver, int, []string, error) {
	dnsServers, ndots, search, err := config.GetDefaultServers()
	if err != nil {
		return nil, 0, nil, err
	}
	servers := make([]models.Nameserver, 0, len(dnsServers))
	for _, s := range dnsServers {
		ns := models.Nameserver{
			Type:    models.UDPResolver,
			Address: net.JoinHostPort(s, models.DefaultUDPPort),
		}
		servers = append(servers, ns)
	}
	return servers, ndots, search, nil
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
	if u.Scheme == "https" {
		ns.Type = models.DOHResolver
		ns.Address = u.String()
	}
	if u.Scheme == "tls" {
		ns.Type = models.DOTResolver
		if u.Port() == "" {
			ns.Address = net.JoinHostPort(u.Hostname(), models.DefaultTLSPort)
		} else {
			ns.Address = net.JoinHostPort(u.Hostname(), u.Port())
		}
	}
	if u.Scheme == "tcp" {
		ns.Type = models.TCPResolver
		if u.Port() == "" {
			ns.Address = net.JoinHostPort(u.Hostname(), models.DefaultTCPPort)
		} else {
			ns.Address = net.JoinHostPort(u.Hostname(), u.Port())
		}
	}
	if u.Scheme == "udp" {
		ns.Type = models.UDPResolver
		if u.Port() == "" {
			ns.Address = net.JoinHostPort(u.Hostname(), models.DefaultUDPPort)
		} else {
			ns.Address = net.JoinHostPort(u.Hostname(), u.Port())
		}
	}
	return ns, nil
}
