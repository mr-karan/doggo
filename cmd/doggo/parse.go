package main

import (
	"strings"

	"github.com/miekg/dns"
)

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
// Returns a list of nameserver, queryTypes, queryClasses, queryNames.
func loadUnparsedArgs(args []string) ([]string, []string, []string, []string) {
	var nameservers, queryTypes, queryClasses, queryNames []string
	for _, arg := range args {
		if strings.HasPrefix(arg, "@") {
			nameservers = append(nameservers, strings.TrimPrefix(arg, "@"))
		} else if qt, ok := dns.StringToType[strings.ToUpper(arg)]; ok {
			queryTypes = append(queryTypes, dns.TypeToString[qt])
		} else if qc, ok := dns.StringToClass[strings.ToUpper(arg)]; ok {
			queryClasses = append(queryClasses, dns.ClassToString[qc])
		} else {
			queryNames = append(queryNames, arg)
		}
	}
	return nameservers, queryTypes, queryClasses, queryNames
}
