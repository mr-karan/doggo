package main

import (
	"strings"

	"github.com/miekg/dns"
)

func (hub *Hub) loadQueryArgs() error {
	err := hub.loadNamedArgs()
	if err != nil {
		return err
	}
	err = hub.loadFreeArgs()
	if err != nil {
		return err
	}
	hub.loadFallbacks()
	return nil
}

// loadFreeArgs tries to parse all the arguments
// given to the CLI. These arguments don't have any specific
// order so we have to deduce based on the pattern of argument.
// For eg, a nameserver must always begin with `@`. In this
// pattern we deduce the arguments and map it to internal query
// options. In case an argument isn't able to fit in any of the existing
// pattern it is considered to be a "query name".
func (hub *Hub) loadFreeArgs() error {
	for _, arg := range hub.FreeArgs {
		if strings.HasPrefix(arg, "@") {
			hub.QueryFlags.Nameservers = append(hub.QueryFlags.Nameservers, strings.Trim(arg, "@"))
		} else if _, ok := dns.StringToType[strings.ToUpper(arg)]; ok {
			hub.QueryFlags.QTypes = append(hub.QueryFlags.QTypes, arg)
		} else if _, ok := dns.StringToClass[strings.ToUpper(arg)]; ok {
			hub.QueryFlags.QClasses = append(hub.QueryFlags.QClasses, arg)
		} else {
			// if nothing matches, consider it's a query name.
			hub.QueryFlags.QNames = append(hub.QueryFlags.QNames, arg)
		}
	}
	return nil
}

// loadNamedArgs checks for all flags and loads their
// values inside the Hub.
func (hub *Hub) loadNamedArgs() error {
	// Unmarshall flags to the struct.
	err := k.Unmarshal("", &hub.QueryFlags)
	if err != nil {
		return err
	}
	return nil
}

// loadFallbacks sets fallbacks for options
// that are not specified by the user.
func (hub *Hub) loadFallbacks() {
	if len(hub.QueryFlags.QTypes) == 0 {
		hub.QueryFlags.QTypes = append(hub.QueryFlags.QTypes, "A")
	}
	if len(hub.QueryFlags.QClasses) == 0 {
		hub.QueryFlags.QClasses = append(hub.QueryFlags.QClasses, "IN")
	}
}
