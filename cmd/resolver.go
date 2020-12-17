package main

import (
	"time"

	"github.com/mr-karan/doggo/pkg/resolvers"
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
	SystemResolver = "system"
)

// loadResolvers loads differently configured
// resolvers based on a list of nameserver.
func (hub *Hub) loadResolvers() error {
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
		if ns.Type == UDPResolver || ns.Type == SystemResolver {
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
