package main

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"runtime"

	"github.com/miekg/dns"
)

const (
	//DefaultResolvConfPath specifies path to default resolv config file on UNIX.
	DefaultResolvConfPath = "/etc/resolv.conf"
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

	// fallback to system nameserver
	// in case no nameserver is specified by user.
	if len(hub.Nameservers) == 0 {
		ns, ndots, err := getDefaultServers()
		if err != nil {
			return fmt.Errorf("error fetching system default nameserver")
		}
		if hub.QueryFlags.Ndots == 0 {
			hub.QueryFlags.Ndots = ndots
		}
		hub.Nameservers = append(hub.Nameservers, ns...)
	}
	return nil
}

// getDefaultServers reads the `resolv.conf`
// file and returns a list of nameservers.
func getDefaultServers() ([]Nameserver, int, error) {
	if runtime.GOOS == "windows" {
		// TODO: Add a method for reading system default nameserver in windows.
		return nil, 0, errors.New(`unable to read default nameservers in this machine`)
	}
	// if no nameserver is provided, take it from `resolv.conf`
	cfg, err := dns.ClientConfigFromFile(DefaultResolvConfPath)
	if err != nil {
		return nil, 0, err
	}
	servers := make([]Nameserver, 0, len(cfg.Servers))
	for _, s := range cfg.Servers {
		addr := net.JoinHostPort(s, cfg.Port)
		ns := Nameserver{
			Type:    UDPResolver,
			Address: addr,
		}
		servers = append(servers, ns)
	}
	return servers, cfg.Ndots, nil
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
