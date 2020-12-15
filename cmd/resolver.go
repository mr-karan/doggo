package main

import (
	"time"

	"github.com/mr-karan/doggo/pkg/resolvers"
)

// initResolver checks for various flags and initialises
// the correct resolver based on the config.
func (hub *Hub) initResolver() error {
	// check if DOH flag is set.
	if hub.QueryFlags.IsDOH {
		hub.Logger.Debug("initiating DOH resolver")
		rslvr, err := resolvers.NewDOHResolver(hub.QueryFlags.Nameservers, resolvers.DOHResolverOpts{
			Timeout: hub.QueryFlags.Timeout * time.Second,
		})
		if err != nil {
			return err
		}
		hub.Resolver = append(hub.Resolver, rslvr)
	}
	if hub.QueryFlags.IsTCP {
		hub.Logger.Debug("initiating TCP resolver")
		rslvr, err := resolvers.NewTCPResolver(hub.QueryFlags.Nameservers, resolvers.TCPResolverOpts{
			IPv4Only: hub.QueryFlags.UseIPv4,
			IPv6Only: hub.QueryFlags.UseIPv6,
			Timeout:  hub.QueryFlags.Timeout * time.Second,
		})
		if err != nil {
			return err
		}
		hub.Resolver = append(hub.Resolver, rslvr)
	}
	// If so far no resolver has been set, then fallback to UDP.
	if hub.QueryFlags.IsUDP || len(hub.Resolver) == 0 {
		hub.Logger.Debug("initiating UDP resolver")
		rslvr, err := resolvers.NewUDPResolver(hub.QueryFlags.Nameservers, resolvers.UDPResolverOpts{
			IPv4Only: hub.QueryFlags.UseIPv4,
			IPv6Only: hub.QueryFlags.UseIPv6,
			Timeout:  hub.QueryFlags.Timeout * time.Second,
		})
		if err != nil {
			return err
		}
		hub.Resolver = append(hub.Resolver, rslvr)
	}
	return nil
}
