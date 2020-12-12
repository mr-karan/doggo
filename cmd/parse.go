package main

import (
	"strings"

	"github.com/miekg/dns"
	"github.com/urfave/cli/v2"
)

func (hub *Hub) loadQueryArgs(c *cli.Context) error {
	err := hub.loadFreeArgs(c)
	if err != nil {
		cli.Exit("Error parsing arguments", -1)
	}
	err = hub.initResolver(c)
	if err != nil {
		cli.Exit("Error parsing nameservers", -1)
	}
	hub.loadFallbacks(c)
	return err
}

// loadFreeArgs tries to parse all the arguments
// given to the CLI. These arguments don't have any specific
// order so we have to deduce based on the pattern of argument.
// For eg, a nameserver must always begin with `@`. In this
// pattern we deduce the arguments and map it to internal query
// options. In case an argument isn't able to fit in any of the existing
// pattern it is considered to be a "query name".
func (hub *Hub) loadFreeArgs(c *cli.Context) error {
	for _, arg := range c.Args().Slice() {
		if strings.HasPrefix(arg, "@") {
			hub.QueryFlags.Nameservers.Set(strings.Trim(arg, "@"))
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
