package main

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"runtime"
	"time"

	"github.com/miekg/dns"
	"github.com/mr-karan/doggo/pkg/resolvers"
)

const (
	//DefaultResolvConfPath specifies path to default resolv config file on UNIX.
	DefaultResolvConfPath = "/etc/resolv.conf"
	// DefaultTLSPort specifies the default port for a DNS server connecting over TCP over TLS
	DefaultTLSPort = "853"
	// DefaultUDPPort specifies the default port for a DNS server connecting over UDP
	DefaultUDPPort = "53"
)

// initResolver checks for various flags and initialises
// the correct resolver based on the config.
func (hub *Hub) initResolver() error {
	// for each nameserver, initialise the correct resolver
	for _, ns := range hub.Nameservers {
		if ns.Type == "doh" {
			hub.Logger.Debug("initiating DOH resolver")
			rslvr, err := resolvers.NewDOHResolver(ns.Address, resolvers.DOHResolverOpts{
				Timeout: hub.QueryFlags.Timeout * time.Second,
			})
			if err != nil {
				return err
			}
			hub.Resolver = append(hub.Resolver, rslvr)
		}
		if ns.Type == "tcp" {
			hub.Logger.Debug("initiating TCP resolver")
			rslvr, err := resolvers.NewTCPResolver(ns.Address, resolvers.TCPResolverOpts{
				IPv4Only: hub.QueryFlags.UseIPv4,
				IPv6Only: hub.QueryFlags.UseIPv6,
				Timeout:  hub.QueryFlags.Timeout * time.Second,
			})
			if err != nil {
				return err
			}
			hub.Resolver = append(hub.Resolver, rslvr)
		}
		if ns.Type == "udp" {
			hub.Logger.Debug("initiating UDP resolver")
			rslvr, err := resolvers.NewUDPResolver(ns.Address, resolvers.UDPResolverOpts{
				IPv4Only: hub.QueryFlags.UseIPv4,
				IPv6Only: hub.QueryFlags.UseIPv6,
				Timeout:  hub.QueryFlags.Timeout * time.Second,
			})
			if err != nil {
				return err
			}
			hub.Resolver = append(hub.Resolver, rslvr)
		}
	}
	return nil
}

func getDefaultServers() ([]Nameserver, error) {
	if runtime.GOOS == "windows" {
		// TODO: Add a method for reading system default nameserver in windows.
		return nil, errors.New(`unable to read default nameservers in this machine`)
	}
	// if no nameserver is provided, take it from `resolv.conf`
	cfg, err := dns.ClientConfigFromFile(DefaultResolvConfPath)
	if err != nil {
		return nil, err
	}
	servers := make([]Nameserver, 0, len(cfg.Servers))
	for _, s := range cfg.Servers {
		ip := net.ParseIP(s)
		// handle IPv6
		if ip != nil && ip.To4() != nil {
			ns := Nameserver{
				Type:    "udp",
				Address: fmt.Sprintf("%s:%s", s, cfg.Port),
			}
			servers = append(servers, ns)
		} else {
			ns := Nameserver{
				Type:    "udp",
				Address: fmt.Sprintf("[%s]:%s", s, cfg.Port),
			}
			servers = append(servers, ns)
		}
	}
	return servers, nil
}

func initNameserver(n string) (Nameserver, error) {
	// Instantiate a dumb UDP resolver as a fallback.
	ns := Nameserver{
		Type:    "udp",
		Address: n,
	}
	u, err := url.Parse(n)
	if err != nil {
		return ns, err
	}
	if u.Scheme == "https" {
		ns.Address = u.String()
		ns.Type = "doh"
	}
	if u.Scheme == "tcp" {
		if i := net.ParseIP(n); i != nil {
			// if no port specified in nameserver, append defaults.
			n = net.JoinHostPort(n, DefaultTLSPort)
		}
		ns.Address = u.String()
		ns.Type = "tcp"
	}
	if u.Scheme == "udp" {
		ns.Type = "udp"
		if u.Port() == "" {
			ns.Address = net.JoinHostPort(u.Hostname(), DefaultUDPPort)
		} else {
			ns.Address = net.JoinHostPort(u.Hostname(), u.Port())
		}
	}
	return ns, nil
}
