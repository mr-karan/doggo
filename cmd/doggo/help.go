package main

import (
	"os"
	"text/template"

	"github.com/fatih/color"
)

// appHelpTextTemplate is the text/template to customise the Help output.
// Uses text/template to render templates.
var appHelpTextTemplate = `{{ "NAME" | color "" "heading" }}:
  {{ .Name | color "green" "bold" }} 🐶 {{.Description}}

{{ "USAGE" | color "" "heading" }}:
  {{ .Name | color "green" "bold" }} [--] {{ "[query options]" | color "yellow" "" }} {{ "[arguments...]" | color "cyan" "" }}

{{ "VERSION" | color "" "heading" }}:
  {{.Version | color "red" "" }} - {{.Date | color "red" ""}}

{{ "EXAMPLES" | color "" "heading" }}:
  {{ .Name | color "green" "bold" }} {{ "mrkaran.dev" | color "cyan" "" }}                            	Query a domain using defaults.
  {{ .Name | color "green" "bold" }} {{ "mrkaran.dev CNAME" | color "cyan" "" }}                 	Looks up for a CNAME record.
  {{ .Name | color "green" "bold" }} {{ "mrkaran.dev MX @9.9.9.9" | color "cyan" "" }}              	Uses a custom DNS resolver.
  {{ .Name | color "green" "bold" }} {{"-q mrkaran.dev -t MX -n 1.1.1.1" | color "yellow" ""}}   	Using named arguments.

{{ "Free Form Arguments" | color "" "heading" }}:
  Supply hostnames, query types, classes without any flag. For eg:
  {{ .Name | color "green" "bold" }} {{"mrkaran.dev A @1.1.1.1" | color "cyan" "" }}

{{ "Transport Options" | color "" "heading" }}:
  Based on the URL scheme the correct resolver is chosen.
  Fallbacks to UDP resolver if no scheme is present.

  {{"@udp://" | color "yellow" ""}}        eg: @1.1.1.1 initiates a {{"UDP" | color "cyan" ""}} resolver for 1.1.1.1:53.
  {{"@tcp://" | color "yellow" ""}}        eg: @tcp://1.1.1.1 initiates a {{"TCP" | color "cyan" ""}} resolver for 1.1.1.1:53.
  {{"@https://" | color "yellow" ""}}      eg: @https://cloudflare-dns.com/dns-query initiates a {{"DOH" | color "cyan" ""}} resolver for Cloudflare DoH server.
  {{"@tls://" | color "yellow" ""}}        eg: @tls://1.1.1.1 initiates a {{"DoT" | color "cyan" ""}} resolver for 1.1.1.1:853.
  {{"@sdns://" | color "yellow" ""}}       initiates a {{"DNSCrypt" | color "cyan" ""}} or {{"DoH" | color "cyan" ""}} resolver using its DNS stamp.
  {{"@quic://" | color "yellow" ""}}       initiates a {{"DOQ" | color "cyan" ""}} resolver.

{{ "Query Options" | color "" "heading" }}:
  {{"-q, --query=HOSTNAME" | color "yellow" ""}}        Hostname to query the DNS records for (eg {{"mrkaran.dev" | color "cyan" ""}}).
  {{"-t, --type=TYPE" | color "yellow" ""}}             Type of the DNS Record ({{"A, MX, NS" | color "cyan" ""}} etc).
  {{"-n, --nameserver=ADDR" | color "yellow" ""}}       Address of a specific nameserver to send queries to ({{"9.9.9.9, 8.8.8.8" | color "cyan" ""}} etc).
  {{"-c, --class=CLASS" | color "yellow" ""}}           Network class of the DNS record ({{"IN, CH, HS" | color "cyan" ""}} etc).
  {{"-x, --reverse" | color "yellow" ""}}               Performs a DNS Lookup for an IPv4 or IPv6 address. Sets the query type and class to PTR and IN respectively.

{{ "Resolver Options" | color "" "heading" }}:
  {{"--ndots=INT" | color "yellow" ""}}        Specify ndots parameter. Takes value from /etc/resolv.conf if using the system namesever or 1 otherwise.
  {{"--search" | color "yellow" ""}}           Use the search list defined in resolv.conf. Defaults to true. Set --search=false to disable search list.
  {{"--timeout" | color "yellow" ""}}          Specify timeout (in seconds) for the resolver to return a response.
  {{"-4 --ipv4" | color "yellow" ""}}          Use IPv4 only.
  {{"-6 --ipv6" | color "yellow" ""}}          Use IPv6 only.

{{ "Output Options" | color "" "heading" }}:
  {{"-J, --json " | color "yellow" ""}}                 Format the output as JSON.
  {{"--color   " | color "yellow" ""}}                  Defaults to true. Set --color=false to disable colored output.
  {{"--debug " | color "yellow" ""}}                    Enable debug logging.
  {{"--time" | color "yellow" ""}}                      Shows how long the response took from the server.
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
				formatter = formatter.Add(color.Bold, color.Underline)
			}
			return formatter.SprintFunc()(str)
		},
	}).Parse(appHelpTextTemplate)
	if err != nil {
		// should ideally never happen.
		panic(err)
	}
	err = tmpl.Execute(os.Stdout, helpTmplVars)
	if err != nil {
		// should ideally never happen.
		panic(err)
	}
	os.Exit(0)
}
