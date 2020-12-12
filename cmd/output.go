package main

import (
	"fmt"
	"strconv"

	"github.com/fatih/color"
	"github.com/miekg/dns"
	"github.com/mr-karan/doggo/pkg/resolvers"
)

// Output takes a list of `dns.Answers` and based
// on the output format specified displays the information.
func (hub *Hub) Output(responses []resolvers.Response) {
	// Create SprintXxx functions to mix strings with other non-colorized strings:
	green := color.New(color.FgGreen).SprintFunc()
	blue := color.New(color.FgBlue).SprintFunc()
	for _, r := range responses {
		var res string
		for _, a := range r.Message.Answer {
			switch t := a.(type) {
			case *dns.A:
				res = t.A.String()
			}
			h := a.Header()
			name := green(h.Name)
			qclass := dns.Class(h.Class).String()
			ttl := strconv.FormatInt(int64(h.Ttl), 10) + "s"
			qtype := blue(dns.Type(h.Rrtype).String())
			fmt.Printf("%s \t %s \t %s \t %s \t %s\n", qtype, name, qclass, ttl, res)
		}
	}
}
