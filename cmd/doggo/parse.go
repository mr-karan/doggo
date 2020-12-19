package main

import (
	"strings"

	"github.com/miekg/dns"
	flag "github.com/spf13/pflag"
)

func (hub *Hub) loadQueryArgs() error {
	// Appends a list of unparsed args to
	// internal query flags.
	err := hub.loadUnparsedArgs()
	if err != nil {
		return err
	}

	// Load all fallbacks in internal query flags.
	hub.loadFallbacks()
	return nil
}

// loadUnparsedArgs tries to parse all the arguments
// which are unparsed by `flag` library. These arguments don't have any specific
// order so we have to deduce based on the pattern of argument.
// For eg, a nameserver must always begin with `@`. In this
// pattern we deduce the arguments and append it to the
// list of internal query flags.
// In case an argument isn't able to fit in any of the existing
// pattern it is considered to be a "hostname".
// Eg of unparsed argument: `dig mrkaran.dev @1.1.1.1 AAAA`
// where `@1.1.1.1` and `AAAA` are "unparsed" args.
func (hub *Hub) loadUnparsedArgs() error {
	for _, arg := range hub.UnparsedArgs {
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

// loadFallbacks sets fallbacks for options
// that are not specified by the user but necessary
// for the resolver.
func (hub *Hub) loadFallbacks() {
	if len(hub.QueryFlags.QTypes) == 0 {
		hub.QueryFlags.QTypes = append(hub.QueryFlags.QTypes, "A")
	}
	if len(hub.QueryFlags.QClasses) == 0 {
		hub.QueryFlags.QClasses = append(hub.QueryFlags.QClasses, "IN")
	}
}

// isFlagPassed checks if the flag is supplied by
//user or not.
func isFlagPassed(name string, f *flag.FlagSet) bool {
	found := false
	f.Visit(func(f *flag.Flag) {
		if f.Name == name {
			found = true
		}
	})
	return found
}
