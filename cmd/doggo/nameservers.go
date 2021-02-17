package main

import (
	"fmt"
	"net"
	"net/url"

	"github.com/mr-karan/doggo/pkg/config"
)

const (
	// DefaultTLSPort specifies the default port for a DNS server connecting over TCP over TLS
	DefaultTLSPort = "853"
	// DefaultUDPPort specifies the default port for a DNS server connecting over UDP
	DefaultUDPPort = "53"
	// DefaultTCPPort specifies the default port for a DNS server connecting over TCP
	DefaultTCPPort = "53"
	UDPResolver    = "udp"
	DOHResolver    = "doh"
	TCPResolver    = "tcp"
	DOTResolver    = "dot"
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

func getDefaultServers() ([]Nameserver, int, []string, error) {
	dnsServers, ndots, search, err := config.GetDefaultServers()
	if err != nil {
		return nil, 0, nil, err
	}
	servers := make([]Nameserver, 0, len(dnsServers))
	for _, s := range dnsServers {
		ns := Nameserver{
			Type:    UDPResolver,
			Address: net.JoinHostPort(s, DefaultUDPPort),
		}
		servers = append(servers, ns)
	}
	return servers, ndots, search, nil
}

func initNameserver(n string) (Nameserver, error) {
	// Instantiate a UDP resolver with default port as a fallback.
	ns := Nameserver{
		Type:    UDPResolver,
		Address: net.JoinHostPort(n, DefaultUDPPort),
	}
	u, err := url.Parse(n)
	if err != nil {
		return ns, err
	}
	if u.Scheme == "https" {
		ns.Type = DOHResolver
		ns.Address = u.String()
	}
	if u.Scheme == "tls" {
		ns.Type = DOTResolver
		if u.Port() == "" {
			ns.Address = net.JoinHostPort(u.Hostname(), DefaultTLSPort)
		} else {
			ns.Address = net.JoinHostPort(u.Hostname(), u.Port())
		}
	}
	if u.Scheme == "tcp" {
		ns.Type = TCPResolver
		if u.Port() == "" {
			ns.Address = net.JoinHostPort(u.Hostname(), DefaultTCPPort)
		} else {
			ns.Address = net.JoinHostPort(u.Hostname(), u.Port())
		}
	}
	if u.Scheme == "udp" {
		ns.Type = UDPResolver
		if u.Port() == "" {
			ns.Address = net.JoinHostPort(u.Hostname(), DefaultUDPPort)
		} else {
			ns.Address = net.JoinHostPort(u.Hostname(), u.Port())
		}
	}
	return ns, nil
}
