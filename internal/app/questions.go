package app

import (
	"os"
	"strings"

	"github.com/miekg/dns"
)

// LoadFallbacks sets fallbacks for options
// that are not specified by the user but necessary
// for the resolver.
func (app *App) LoadFallbacks() {
	if len(app.QueryFlags.QTypes) == 0 {
		app.QueryFlags.QTypes = append(app.QueryFlags.QTypes, "A")
	}
	if len(app.QueryFlags.QClasses) == 0 {
		app.QueryFlags.QClasses = append(app.QueryFlags.QClasses, "IN")
	}
}

// PrepareQuestions takes a list of query names, query types and query classes
// and prepare a question for each combination of the above.
func (app *App) PrepareQuestions() {
	for _, n := range app.QueryFlags.QNames {
		for _, t := range app.QueryFlags.QTypes {
			for _, c := range app.QueryFlags.QClasses {
				app.Questions = append(app.Questions, dns.Question{
					Name:   n,
					Qtype:  dns.StringToType[strings.ToUpper(t)],
					Qclass: dns.StringToClass[strings.ToUpper(c)],
				})
			}
		}
	}
}

// ReverseLookup is used to perform a reverse DNS Lookup
// using an IPv4 or IPv6 address.
// Query Type is set to PTR, Query Class is set to IN.
// Query Names must be formatted in in-addr.arpa. or ip6.arpa format.
func (app *App) ReverseLookup() {
	app.QueryFlags.QTypes = []string{"PTR"}
	app.QueryFlags.QClasses = []string{"IN"}
	formattedNames := make([]string, 0, len(app.QueryFlags.QNames))

	for _, n := range app.QueryFlags.QNames {
		addr, err := dns.ReverseAddr(n)
		if err != nil {
			app.Logger.Error("error formatting address", "error", err)
			os.Exit(2)
		}
		formattedNames = append(formattedNames, addr)
	}
	app.QueryFlags.QNames = formattedNames
}
