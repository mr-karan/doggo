package app

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/fatih/color"
	"github.com/miekg/dns"
	"github.com/mr-karan/doggo/pkg/resolvers"
	"github.com/olekukonko/tablewriter"
)

var (
	TerminalColorGreen   = color.New(color.FgGreen, color.Bold).SprintFunc()
	TerminalColorBlue    = color.New(color.FgBlue, color.Bold).SprintFunc()
	TerminalColorYellow  = color.New(color.FgYellow, color.Bold).SprintFunc()
	TerminalColorCyan    = color.New(color.FgCyan, color.Bold).SprintFunc()
	TerminalColorRed     = color.New(color.FgRed, color.Bold).SprintFunc()
	TerminalColorMagenta = color.New(color.FgMagenta, color.Bold).SprintFunc()
)

func (app *App) outputJSON(rsp []resolvers.Response) {
	jsonOutput := struct {
		Responses []resolvers.Response `json:"responses"`
	}{
		Responses: rsp,
	}

	// Pretty print with 4 spaces.
	res, err := json.MarshalIndent(jsonOutput, "", "    ")
	if err != nil {
		app.Logger.Error("unable to output data in JSON", "error", err)
		os.Exit(-1)
	}
	fmt.Printf("%s\n", res)
}

func (app *App) outputShort(rsp []resolvers.Response) {
	for _, r := range rsp {
		for _, a := range r.Answers {
			fmt.Printf("%s\n", a.Address)
		}
	}
}

func (app *App) outputSingle(rsp []resolvers.Response) {
	var addresses []string
	for _, r := range rsp {
		for _, a := range r.Answers {
			addresses = append(addresses, a.Address)
		}
	}
	if len(addresses) != 0 {
		fmt.Printf("%s\n", selectAddress(addresses))
	}
}

func (app *App) outputTerminal(rsp []resolvers.Response) {
	// Disables colorized output if user specified.
	if !app.QueryFlags.Color {
		color.NoColor = true
	}

	// Conditional Time column.
	table := tablewriter.NewWriter(color.Output)
	header := []string{"Name", "Type", "Class", "TTL", "Address", "Nameserver"}
	if app.QueryFlags.DisplayTimeTaken {
		header = append(header, "Time Taken")
	}

	// Show output in case if it's not
	// a NOERROR.
	outputStatus := false
	for _, r := range rsp {
		for _, a := range r.Authorities {
			if dns.StringToRcode[a.Status] != dns.RcodeSuccess {
				outputStatus = true
			}
		}
		for _, a := range r.Answers {
			if dns.StringToRcode[a.Status] != dns.RcodeSuccess {
				outputStatus = true
			}
		}
	}
	if outputStatus {
		header = append(header, "Status")
	}

	// Formatting options for the table.
	table.SetHeader(header)
	table.SetAutoWrapText(true)
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

	for _, r := range rsp {
		for _, ans := range r.Answers {
			typOut := getColoredType(ans.Type)
			output := []string{TerminalColorGreen(ans.Name), typOut, ans.Class, ans.TTL, ans.Address, ans.Nameserver}
			// Print how long it took
			if app.QueryFlags.DisplayTimeTaken {
				output = append(output, ans.RTT)
			}
			if outputStatus {
				output = append(output, TerminalColorRed(ans.Status))
			}
			table.Append(output)
		}
		for _, auth := range r.Authorities {
			var typOut string
			switch typ := auth.Type; typ {
			case "SOA":
				typOut = TerminalColorRed(auth.Type)
			default:
				typOut = TerminalColorBlue(auth.Type)
			}
			output := []string{TerminalColorGreen(auth.Name), typOut, auth.Class, auth.TTL, auth.MName, auth.Nameserver}
			// Print how long it took
			if app.QueryFlags.DisplayTimeTaken {
				output = append(output, auth.RTT)
			}
			if outputStatus {
				output = append(output, TerminalColorRed(auth.Status))
			}
			table.Append(output)
		}
	}
	table.Render()
}

func getColoredType(t string) string {
	switch t {
	case "A":
		return TerminalColorBlue(t)
	case "AAAA":
		return TerminalColorBlue(t)
	case "MX":
		return TerminalColorMagenta(t)
	case "NS":
		return TerminalColorCyan(t)
	case "CNAME":
		return TerminalColorYellow(t)
	case "TXT":
		return TerminalColorYellow(t)
	case "SOA":
		return TerminalColorRed(t)
	default:
		return TerminalColorBlue(t)
	}
}

// Output takes a list of `dns.Answers` and based
// on the output format specified displays the information.
func (app *App) Output(responses []resolvers.Response) {
	if app.QueryFlags.ShowJSON {
		app.outputJSON(responses)
	} else if app.QueryFlags.ShortOutput {
		app.outputShort(responses)
	} else if app.QueryFlags.SingleOutput {
		app.outputSingle(responses)
	} else {
		app.outputTerminal(responses)
	}
}
