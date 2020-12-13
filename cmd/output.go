package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"

	"github.com/fatih/color"
	"github.com/miekg/dns"
	"github.com/mr-karan/doggo/pkg/resolvers"
	"github.com/olekukonko/tablewriter"
)

// Output has a list of fields which are produced for the output
type Output struct {
	Name       string `json:"name"`
	Type       string `json:"type"`
	Class      string `json:"class"`
	TTL        string `json:"ttl"`
	Address    string `json:"address"`
	TimeTaken  string `json:"rtt"`
	Nameserver string `json:"nameserver"`
}

type Query struct {
	Name  string `json:"name"`
	Type  string `json:"type"`
	Class string `json:"class"`
}
type Response struct {
	Output  []Output `json:"answers"`
	Queries []Query  `json:"queries"`
}

type JSONResponse struct {
	Response `json:"responses"`
}

func (hub *Hub) outputJSON(out []Output, msgs []resolvers.Response) {
	// get the questions
	queries := make([]Query, 0, len(msgs))
	for _, ques := range hub.Questions {
		q := Query{
			Name:  ques.Name,
			Type:  dns.ClassToString[ques.Qtype],
			Class: dns.ClassToString[ques.Qclass],
		}
		queries = append(queries, q)
	}

	resp := JSONResponse{
		Response{
			Output:  out,
			Queries: queries,
		},
	}
	res, err := json.Marshal(resp)
	if err != nil {
		hub.Logger.WithError(err).Error("unable to output data in JSON")
		hub.Logger.Exit(-1)
	}
	fmt.Printf("%s", res)
}

func (hub *Hub) outputTerminal(out []Output) {
	green := color.New(color.FgGreen).SprintFunc()
	blue := color.New(color.FgBlue).SprintFunc()

	table := tablewriter.NewWriter(os.Stdout)
	header := []string{"Name", "Type", "Class", "TTL", "Address"}
	if hub.QueryFlags.DisplayTimeTaken {
		header = append(header, "Time Taken")
	}
	table.SetAutoWrapText(false)
	table.SetAutoFormatHeaders(true)
	table.SetHeader(header)

	for _, o := range out {
		output := []string{green(o.Name), blue(o.Type), o.Class, o.TTL, o.Address}
		// Print how long it took
		if hub.QueryFlags.DisplayTimeTaken {
			output = append(output, o.TimeTaken)
		}
		table.Append(output)
	}
	table.Render()
}

// Output takes a list of `dns.Answers` and based
// on the output format specified displays the information.
func (hub *Hub) Output(responses []resolvers.Response) {
	out := collectOutput(responses)
	if len(out) == 0 {
		hub.Logger.Info("No records found")
		hub.Logger.Exit(0)
	}
	if hub.QueryFlags.ShowJSON {
		hub.outputJSON(out, responses)
	} else {
		hub.outputTerminal(out)
	}
}

func collectOutput(responses []resolvers.Response) []Output {
	var out []Output
	// gather Output from the DNS Messages
	for _, r := range responses {
		var addr string
		for _, a := range r.Message.Answer {
			switch t := a.(type) {
			case *dns.A:
				addr = t.A.String()
			case *dns.AAAA:
				addr = t.AAAA.String()
			case *dns.CNAME:
				addr = t.Target
			case *dns.MX:
				addr = strconv.Itoa(int(t.Preference)) + " " + t.Mx
			case *dns.SOA:
				addr = t.String()
			}

			h := a.Header()
			name := h.Name
			qclass := dns.Class(h.Class).String()
			ttl := strconv.FormatInt(int64(h.Ttl), 10) + "s"
			qtype := dns.Type(h.Rrtype).String()
			rtt := fmt.Sprintf("%dms", r.RTT.Milliseconds())
			o := Output{
				Name:       name,
				Type:       qtype,
				TTL:        ttl,
				Class:      qclass,
				Address:    addr,
				TimeTaken:  rtt,
				Nameserver: r.Nameserver,
			}
			out = append(out, o)
		}
	}
	return out
}
