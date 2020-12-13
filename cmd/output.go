package main

import (
	"fmt"
	"os"
	"strconv"

	"github.com/fatih/color"
	"github.com/miekg/dns"
	"github.com/mr-karan/doggo/pkg/resolvers"
	"github.com/olekukonko/tablewriter"
)

// Output takes a list of `dns.Answers` and based
// on the output format specified displays the information.
func (hub *Hub) Output(responses []resolvers.Response) {
	// Create SprintXxx functions to mix strings with other non-colorized strings:
	green := color.New(color.FgGreen).SprintFunc()
	blue := color.New(color.FgBlue).SprintFunc()

	table := tablewriter.NewWriter(os.Stdout)
	header := []string{"Name", "Type", "Class", "TTL", "Address"}
	if hub.QueryFlags.DisplayTimeTaken {
		header = append(header, "Time Taken")
	}
	table.SetHeader(header)

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
			output := []string{name, qtype, qclass, ttl, res}
			// Print how long it took
			if hub.QueryFlags.DisplayTimeTaken {
				output = append(output, fmt.Sprintf("%dms", r.RTT.Milliseconds()))
			}
			table.Append(output)
		}
	}
	table.Render()
}
