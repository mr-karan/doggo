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

func (app *App) outputJSON(rsp []resolvers.Response) {
	// Pretty print with 4 spaces.
	res, err := json.MarshalIndent(rsp, "", "    ")
	if err != nil {
		app.Logger.WithError(err).Error("unable to output data in JSON")
		app.Logger.Exit(-1)
	}
	fmt.Printf("%s", res)
}

func (app *App) outputTerminal(rsp []resolvers.Response) {
	var (
		green   = color.New(color.FgGreen, color.Bold).SprintFunc()
		blue    = color.New(color.FgBlue, color.Bold).SprintFunc()
		yellow  = color.New(color.FgYellow, color.Bold).SprintFunc()
		cyan    = color.New(color.FgCyan, color.Bold).SprintFunc()
		red     = color.New(color.FgRed, color.Bold).SprintFunc()
		magenta = color.New(color.FgMagenta, color.Bold).SprintFunc()
	)

	// Disables colorized output if user specified.
	if !app.QueryFlags.Color {
		color.NoColor = true
	}

	// Conditional Time column.
	table := tablewriter.NewWriter(os.Stdout)
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
			var typOut string
			switch typ := ans.Type; typ {
			case "A":
				typOut = blue(ans.Type)
			case "AAAA":
				typOut = blue(ans.Type)
			case "MX":
				typOut = magenta(ans.Type)
			case "NS":
				typOut = cyan(ans.Type)
			case "CNAME":
				typOut = yellow(ans.Type)
			case "TXT":
				typOut = yellow(ans.Type)
			case "SOA":
				typOut = red(ans.Type)
			default:
				typOut = blue(ans.Type)
			}
			output := []string{green(ans.Name), typOut, ans.Class, ans.TTL, ans.Address, ans.Nameserver}
			// Print how long it took
			if app.QueryFlags.DisplayTimeTaken {
				output = append(output, ans.RTT)
			}
			if outputStatus {
				output = append(output, red(ans.Status))
			}
			table.Append(output)
		}
		for _, auth := range r.Authorities {
			var typOut string
			switch typ := auth.Type; typ {
			case "SOA":
				typOut = red(auth.Type)
			default:
				typOut = blue(auth.Type)
			}
			output := []string{green(auth.Name), typOut, auth.Class, auth.TTL, auth.MName, auth.Nameserver}
			// Print how long it took
			if app.QueryFlags.DisplayTimeTaken {
				output = append(output, auth.RTT)
			}
			if outputStatus {
				output = append(output, red(auth.Status))
			}
			table.Append(output)
		}
	}
	table.Render()
}

// Output takes a list of `dns.Answers` and based
// on the output format specified displays the information.
func (app *App) Output(responses []resolvers.Response) {
	if app.QueryFlags.ShowJSON {
		app.outputJSON(responses)
	} else {
		app.outputTerminal(responses)
	}
}
