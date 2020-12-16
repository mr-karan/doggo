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
	DefaultTCPPort = "53"
	UDPResolver    = "udp"
	DOHResolver    = "doh"
	TCPResolver    = "tcp"
	DOTResolver    = "dot"
)

// initResolver checks for various flags and initialises
// the correct resolver based on the config.
func (hub *Hub) initResolver() error {
	// for each nameserver, initialise the correct resolver
	for _, ns := range hub.Nameservers {
		if ns.Type == DOHResolver {
			hub.Logger.Debug("initiating DOH resolver")
			rslvr, err := resolvers.NewDOHResolver(ns.Address, resolvers.DOHResolverOpts{
				Timeout: hub.QueryFlags.Timeout * time.Second,
			})
			if err != nil {
				return err
			}
			hub.Resolver = append(hub.Resolver, rslvr)
		}
		if ns.Type == DOTResolver {
			hub.Logger.Debug("initiating DOT resolver")
			rslvr, err := resolvers.NewClassicResolver(ns.Address, resolvers.ClassicResolverOpts{
				IPv4Only: hub.QueryFlags.UseIPv4,
				IPv6Only: hub.QueryFlags.UseIPv6,
				Timeout:  hub.QueryFlags.Timeout * time.Second,
				UseTLS:   true,
				UseTCP:   true,
			})
			if err != nil {
				return err
			}
			hub.Resolver = append(hub.Resolver, rslvr)
		}
		if ns.Type == TCPResolver {
			hub.Logger.Debug("initiating TCP resolver")
			rslvr, err := resolvers.NewClassicResolver(ns.Address, resolvers.ClassicResolverOpts{
				IPv4Only: hub.QueryFlags.UseIPv4,
				IPv6Only: hub.QueryFlags.UseIPv6,
				Timeout:  hub.QueryFlags.Timeout * time.Second,
				UseTLS:   false,
				UseTCP:   true,
			})
			if err != nil {
				return err
			}
			hub.Resolver = append(hub.Resolver, rslvr)
		}
		if ns.Type == UDPResolver {
			hub.Logger.Debug("initiating UDP resolver")
			rslvr, err := resolvers.NewClassicResolver(ns.Address, resolvers.ClassicResolverOpts{
				IPv4Only: hub.QueryFlags.UseIPv4,
				IPv6Only: hub.QueryFlags.UseIPv6,
				Timeout:  hub.QueryFlags.Timeout * time.Second,
				UseTLS:   false,
				UseTCP:   false,
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
				Type:    UDPResolver,
				Address: fmt.Sprintf("%s:%s", s, cfg.Port),
			}
			servers = append(servers, ns)
		} else {
			ns := Nameserver{
				Type:    UDPResolver,
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
		Type:    UDPResolver,
		Address: n,
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
