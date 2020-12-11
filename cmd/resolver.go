package main

import (
	"runtime"

	"github.com/mr-karan/doggo/pkg/resolvers"
	"github.com/urfave/cli/v2"
)

// loadResolver checks
func (hub *Hub) loadResolver(c *cli.Context) error {
	if len(hub.QueryFlags.Nameservers.Value()) == 0 {
		if runtime.GOOS == "windows" {
			// TODO: Add a method for reading system default nameserver in windows.
		} else {
			rslvr, err := resolvers.NewResolverFromResolvFile(resolvers.DefaultResolvConfPath)
			if err != nil {
				return err
			}
			hub.Resolver = rslvr
		}
	} else {
		rslvr := resolvers.NewResolver(hub.QueryFlags.Nameservers.Value())
		hub.Resolver = rslvr
	}
	return nil
}
