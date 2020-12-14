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
		fmt.Println(ques.Qtype)
		q := Query{
			Name:  ques.Name,
			Type:  dns.TypeToString[ques.Qtype],
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
	green := color.New(color.FgGreen, color.Bold).SprintFunc()
	blue := color.New(color.FgBlue, color.Bold).SprintFunc()
	yellow := color.New(color.FgYellow, color.Bold).SprintFunc()
	cyan := color.New(color.FgCyan, color.Bold).SprintFunc()
	red := color.New(color.FgRed, color.Bold).SprintFunc()

	if !hub.QueryFlags.Color {
		color.NoColor = true // disables colorized output
	}

	table := tablewriter.NewWriter(os.Stdout)
	header := []string{"Name", "Type", "Class", "TTL", "Address", "Nameserver"}
	if hub.QueryFlags.DisplayTimeTaken {
		header = append(header, "Time Taken")
	}
	table.SetHeader(header)
	table.SetAutoWrapText(false)
	table.SetAutoFormatHeaders(true)
	table.SetHeaderAlignment(tablewriter.ALIGN_LEFT)
	table.SetAlignment(tablewriter.ALIGN_LEFT)
	table.SetCenterSeparator("")
	table.SetColumnSeparator("")
	table.SetRowSeparator("")
	table.SetHeaderLine(false)
	table.SetBorder(false)
	table.SetTablePadding("\t") // pad with tabs
	table.SetNoWhiteSpace(true)

	for _, o := range out {
		var typOut string
		switch typ := o.Type; typ {
		case "A":
			typOut = blue(o.Type)
		case "AAAA":
			typOut = blue(o.Type)
		case "MX":
			typOut = red(o.Type)
		case "NS":
			typOut = cyan(o.Type)
		case "CNAME":
			typOut = yellow(o.Type)
		case "TXT":
			typOut = yellow(o.Type)
		default:
			typOut = blue(o.Type)
		}
		output := []string{green(o.Name), typOut, o.Class, o.TTL, o.Address, o.Nameserver}
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
			case *dns.CAA:
				addr = t.Tag + " " + t.Value
			case *dns.HINFO:
				addr = t.Cpu + " " + t.Os
			// case *dns.LOC:
			// 	addr = t.String()
			case *dns.PTR:
				addr = t.Ptr
			case *dns.SRV:
				addr = strconv.Itoa(int(t.Priority)) + " " +
					strconv.Itoa(int(t.Weight)) + " " +
					t.Target + ":" + strconv.Itoa(int(t.Port))
			case *dns.TXT:
				addr = t.String()
			case *dns.NS:
				addr = t.Ns
			case *dns.MX:
				addr = strconv.Itoa(int(t.Preference)) + " " + t.Mx
			case *dns.SOA:
				addr = t.String()
			case *dns.NAPTR:
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
