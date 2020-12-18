package main

import (
	"time"

	"github.com/mr-karan/doggo/pkg/resolvers"
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
