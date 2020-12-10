package main

import (
	"runtime"
	"strings"

	"github.com/miekg/dns"
	"github.com/mr-karan/doggo/pkg/resolve"
	"github.com/urfave/cli/v2"
)

func (hub *Hub) loadQueryArgs(c *cli.Context) error {
	err := hub.parseFreeArgs(c)
	if err != nil {
		cli.Exit("Error parsing arguments", -1)
	}
	err = hub.loadResolver(c)
	if err != nil {
		cli.Exit("Error parsing nameservers", -1)
	}
	hub.loadFallbacks(c)
	return err
}

// parseFreeArgs tries to parse all the arguments
// given to the CLI. These arguments don't have any specific
// order so we have to deduce based on the pattern of argument.
// For eg, a nameserver must always begin with `@`. In this
// pattern we deduce the arguments and map it to internal query
// options. In case an argument isn't able to fit in any of the existing
// pattern it is considered to be a "query name".
func (hub *Hub) parseFreeArgs(c *cli.Context) error {
	for _, arg := range c.Args().Slice() {
		if strings.HasPrefix(arg, "@") {
			hub.QueryFlags.Nameservers.Set(arg)
		} else if _, ok := dns.StringToType[strings.ToUpper(arg)]; ok {
			hub.QueryFlags.QTypes.Set(arg)
		} else if _, ok := dns.StringToClass[strings.ToUpper(arg)]; ok {
			hub.QueryFlags.QClasses.Set(arg)
		} else {
			// if nothing matches, consider it's a query name.
			hub.QueryFlags.QNames.Set(arg)
		}
	}
	return nil
}

// loadFallbacks sets fallbacks for options
// that are not specified by the user.
func (hub *Hub) loadFallbacks(c *cli.Context) {
	if len(hub.QueryFlags.QTypes.Value()) == 0 {
		hub.QueryFlags.QTypes.Set("A")
	}
	if len(hub.QueryFlags.QClasses.Value()) == 0 {
		hub.QueryFlags.QClasses.Set("IN")
	}
}

// loadResolver checks
func (hub *Hub) loadResolver(c *cli.Context) error {
	if len(hub.QueryFlags.Nameservers.Value()) == 0 {
		if runtime.GOOS == "windows" {
			// TODO: Add a method for reading system default nameserver in windows.
		} else {
			rslvr, err := resolve.NewResolverFromResolvFile(resolve.DefaultResolvConfPath)
			if err != nil {
				return err
			}
			hub.Resolver = rslvr
		}
	} else {
		rslvr := resolve.NewResolver(hub.QueryFlags.Nameservers.Value())
		hub.Resolver = rslvr
	}
	return nil
}
