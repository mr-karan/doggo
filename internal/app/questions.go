package app

import (
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
