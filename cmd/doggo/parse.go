package main

import (
	"strings"

	"github.com/miekg/dns"
	"github.com/mr-karan/doggo/pkg/models"
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
func loadUnparsedArgs(args []string) ([]string, []string, []string, []string, error) {
	var nameservers, queryTypes, queryClasses, queryNames []string
	for _, arg := range args {
		if strings.HasPrefix(arg, "@") {
			nameservers = append(nameservers, strings.TrimPrefix(arg, "@"))
		} else if qt, err := models.ParseRecordType(arg); err == nil {
			queryTypes = append(queryTypes, models.RecordTypeString(qt))
		} else if _, knownType := dns.StringToType[strings.ToUpper(arg)]; knownType || looksLikeNumericRecordType(arg) {
			return nil, nil, nil, nil, err
		} else if qc, ok := dns.StringToClass[strings.ToUpper(arg)]; ok {
			queryClasses = append(queryClasses, dns.ClassToString[qc])
		} else {
			queryNames = append(queryNames, arg)
		}
	}
	return nameservers, queryTypes, queryClasses, queryNames, nil
}

// Unknown free-form arguments are hostnames, but values that explicitly use
// one of the numeric record-type forms should surface parse errors instead of
// being mistaken for query names.
func looksLikeNumericRecordType(value string) bool {
	if value == "" {
		return false
	}
	upper := strings.ToUpper(value)
	if strings.HasPrefix(upper, "TYPE") {
		value = value[len("TYPE"):]
		if value == "" {
			return false
		}
		if value[0] == '-' || value[0] == '+' {
			value = value[1:]
		}
		if value == "" {
			return false
		}
	}
	for _, r := range value {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}
