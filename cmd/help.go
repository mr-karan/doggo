package main

import (
	"os"
	"text/template"

	"github.com/fatih/color"
)

// AppHelpTemplate is the text template to customise the Help output.
// Uses text/template to render templates.
var AppHelpTemplate = `{{ "NAME" | color "" "heading" }}:
  {{ .Name | color "green" "bold" }} 🐶 {{.Description}}

{{ "USAGE" | color "" "heading" }}:
  {{ .Name | color "green" "" }} [--] {{ "[query options]" | color "yellow" "" }} {{ "[arguments...]" | color "cyan" "" }}

{{ "VERSION" | color "" "heading" }}:
  {{.Version | color "red" "" }} - {{.Date | color "red" ""}}

{{ "EXAMPLES" | color "" "heading" }}:
  {{ .Name | color "green" "" }} {{ "mrkaran.dev" | color "cyan" "" }}                            	Query a domain using defaults
  {{ .Name | color "green" "" }} {{ "mrkaran.dev CNAME" | color "cyan" "" }}                 	Looks up for a CNAME record
  {{ .Name | color "green" "" }} {{ "mrkaran.dev MX @9.9.9.9" | color "cyan" "" }}              	Uses a custom DNS resolver
  {{ .Name | color "green" "" }} {{"-q mrkaran.dev -t MX -n 1.1.1.1" | color "yellow" ""}}   	Using named arguments

{{ "Free Form Arguments" | color "" "heading" }}:
  Supply hostnames, query types, classes without any flag. For eg:
  {{ .Name | color "green" "" }} {{"mrkaran.dev A @1.1.1.1" | color "cyan" "" }}

{{ "Query Options" | color "" "heading" }}:
  {{"-q, --query=HOSTNAME" | color "yellow" ""}}        Hostname to query the DNS records for
  {{"-t, --type=TYPE" | color "yellow" ""}}             Type of the DNS Record (A, MX, NS etc)
  {{"-n, --nameserver=ADDR" | color "yellow" ""}}       Address of a specific nameserver to send queries to (9.9.9.9, 1.1.1.1 etc)
  {{"-c, --class=CLASS" | color "yellow" ""}}           Network class of the DNS record (IN, CH, HS etc)

{{ "Protocol Options" | color "" "heading" }}:
  {{"-U, --udp " | color "yellow" ""}}                  Send queries via DNS over UDP protocol
  {{"-T, --tcp " | color "yellow" ""}}                  Send queries via DNS over TCP protocol
  {{"-S, --dot " | color "yellow" ""}}                  Send queries via DNS over TLS (DoT) protocol
  {{"-H, --doh" | color "yellow" ""}}                   Send queries via DNS over HTTPS (DoH) protocol

{{ "Output Options" | color "" "heading" }}:
  {{"-J, --json " | color "yellow" ""}}                 Format the output as JSON
  {{"--color   " | color "yellow" ""}}                  Defaults to true. Set --color=false to disable colored output
  {{"--debug " | color "yellow" ""}}                    Enable debug logging
  {{"--time" | color "yellow" ""}}                      Shows how long the response took from the server
`

func renderCustomHelp() {
	helpTmplVars := map[string]string{
		"Name":        "doggo",
		"Description": "DNS Client for Humans",
		"Version":     buildVersion,
		"Date":        buildDate,
	}
	tmpl, err := template.New("test").Funcs(template.FuncMap{
		"color": func(clr string, format string, str string) string {
			formatter := color.New()
			switch c := clr; c {
			case "yellow":
				formatter = formatter.Add(color.FgYellow)
			case "red":
				formatter = formatter.Add(color.FgRed)
			case "cyan":
				formatter = formatter.Add(color.FgCyan)
			case "green":
				formatter = formatter.Add(color.FgGreen)
			}
			switch f := format; f {
			case "bold":
				formatter = formatter.Add(color.Bold)
			case "underline":
				formatter = formatter.Add(color.Underline)
			case "heading":
				formatter = formatter.Add(color.Bold)
				formatter = formatter.Add(color.Underline)
			}
			return formatter.SprintFunc()(str)
		},
	}).Parse(AppHelpTemplate)
	if err != nil {
		panic(err)
	}
	err = tmpl.Execute(os.Stdout, helpTmplVars)
	if err != nil {
		panic(err)
	}
	os.Exit(0)
}
