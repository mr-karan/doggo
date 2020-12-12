package main

import (
	"runtime"

	"github.com/mr-karan/doggo/pkg/resolvers"
	"github.com/urfave/cli/v2"
)

// initResolver checks for various flags and initialises
// the correct resolver based on the config.
func (hub *Hub) initResolver(c *cli.Context) error {
	// check if DOH flag is set.
	if hub.QueryFlags.IsDOH {
		rslvr, err := resolvers.NewDOHResolver(hub.QueryFlags.Nameservers.Value())
		if err != nil {
			return err
		}
		hub.Resolver = rslvr
		return nil
	}
	if len(hub.QueryFlags.Nameservers.Value()) == 0 {
		if runtime.GOOS == "windows" {
			// TODO: Add a method for reading system default nameserver in windows.
		} else {
			rslvr, err := resolvers.NewResolverFromResolvFile(resolvers.DefaultResolvConfPath)
			if err != nil {
				return err
			}
			hub.Resolver = rslvr
			return nil
		}
	} else {
		rslvr, err := resolvers.NewClassicResolver(hub.QueryFlags.Nameservers.Value(), resolvers.ClassicResolverOpts{
			UseIPv4: hub.QueryFlags.UseIPv4,
			UseIPv6: hub.QueryFlags.UseIPv6,
			UseTLS:  hub.QueryFlags.IsDOT,
			UseTCP:  hub.QueryFlags.UseTCP,
		})
		if err != nil {
			return err
		}
		hub.Resolver = rslvr
		return nil
	}
	return nil
}
