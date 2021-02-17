package main

import (
	"time"

	"github.com/mr-karan/doggo/pkg/resolvers"
)

// loadResolverOptions loads the common options
// to configure a resolver from the query args.
func (hub *Hub) loadResolverOptions() {
	hub.ResolverOpts.Timeout = hub.QueryFlags.Timeout
}

// loadResolvers loads differently configured
// resolvers based on a list of nameserver.
func (hub *Hub) loadResolvers() error {
	var resolverOpts = resolvers.Options{
		Timeout:    hub.QueryFlags.Timeout * time.Second,
		Ndots:      hub.ResolverOpts.Ndots,
		SearchList: hub.ResolverOpts.SearchList,
		Logger:     hub.Logger,
	}
	// for each nameserver, initialise the correct resolver
	for _, ns := range hub.Nameservers {
		if ns.Type == DOHResolver {
			hub.Logger.Debug("initiating DOH resolver")
			rslvr, err := resolvers.NewDOHResolver(ns.Address, resolverOpts)
			if err != nil {
				return err
			}
			hub.Resolver = append(hub.Resolver, rslvr)
		}
		if ns.Type == DOTResolver {
			hub.Logger.Debug("initiating DOT resolver")
			rslvr, err := resolvers.NewClassicResolver(ns.Address,
				resolvers.ClassicResolverOpts{
					IPv4Only: hub.QueryFlags.UseIPv4,
					IPv6Only: hub.QueryFlags.UseIPv6,
					UseTLS:   true,
					UseTCP:   true,
				}, resolverOpts)

			if err != nil {
				return err
			}
			hub.Resolver = append(hub.Resolver, rslvr)
		}
		if ns.Type == TCPResolver {
			hub.Logger.Debug("initiating TCP resolver")
			rslvr, err := resolvers.NewClassicResolver(ns.Address,
				resolvers.ClassicResolverOpts{
					IPv4Only: hub.QueryFlags.UseIPv4,
					IPv6Only: hub.QueryFlags.UseIPv6,
					UseTLS:   false,
					UseTCP:   true,
				}, resolverOpts)
			if err != nil {
				return err
			}
			hub.Resolver = append(hub.Resolver, rslvr)
		}
		if ns.Type == UDPResolver {
			hub.Logger.Debug("initiating UDP resolver")
			rslvr, err := resolvers.NewClassicResolver(ns.Address,
				resolvers.ClassicResolverOpts{
					IPv4Only: hub.QueryFlags.UseIPv4,
					IPv6Only: hub.QueryFlags.UseIPv6,
					UseTLS:   false,
					UseTCP:   false,
				}, resolverOpts)
			if err != nil {
				return err
			}
			hub.Resolver = append(hub.Resolver, rslvr)
		}
	}
	return nil
}
