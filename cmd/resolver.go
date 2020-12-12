package main

import (
	"runtime"

	"github.com/mr-karan/doggo/pkg/resolvers"
	"github.com/urfave/cli/v2"
)

// loadResolver checks
func (hub *Hub) loadResolver(c *cli.Context) error {
	// check if DOH flag is set.
	if hub.QueryFlags.IsDOH {
		rslvr, err := resolvers.NewDOHResolver(hub.QueryFlags.Nameservers.Value())
		if err != nil {
			return err
		}
		hub.Resolver = rslvr
		return nil
	}
	// check if DOT flag is set.

	// check if TCP flag is set.

	// fallback to good ol UDP.
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
		rslvr, err := resolvers.NewClassicResolver(hub.QueryFlags.Nameservers.Value())
		if err != nil {
			return err
		}
		hub.Resolver = rslvr
		return nil
	}
	return nil
}
