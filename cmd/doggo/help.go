package main

import (
	"os"
	"text/template"

	"github.com/fatih/color"
)

// appHelpTextTemplate is the text/template to customise the Help output.
var appHelpTextTemplate = `{{ "NAME" | color "" "heading" }}:
  {{ .Name | color "green" "bold" }} 🐶 {{ .Description }}

{{ "USAGE" | color "" "heading" }}:
  {{ .Name | color "green" "bold" }} [--] {{ "[query options]" | color "yellow" "" }} {{ "[arguments...]" | color "cyan" "" }}

{{ "VERSION" | color "" "heading" }}:
  {{ .Version | color "red" "" }} - {{ .Date | color "red" "" }}

{{ "EXAMPLES" | color "" "heading" }}:
  {{- range $example := .Examples }}
  {{ $.Name | color "green" "bold" }} {{ printf "%-40s" $example.Command | color "cyan" "" }}{{ $example.Description }}
  {{- end }}

{{ "FREE FORM ARGUMENTS" | color "" "heading" }}:
  Supply hostnames, query types, and classes without flags. Example:
  {{ .Name | color "green" "bold" }} {{ "mrkaran.dev A @1.1.1.1" | color "cyan" "" }}

{{ "TRANSPORT OPTIONS" | color "" "heading" }}:
  Specify the protocol with a URL-type scheme.
  UDP is used if no scheme is specified.

  {{- range $opt := .TransportOptions }}
  {{ printf "%-12s" $opt.Scheme | color "yellow" "" }}{{ printf "%-68s" $opt.Example }}{{ $opt.Description | color "cyan" "" }}
  {{- end }}

{{ "SUBCOMMANDS" | color "" "heading" }}:
  {{- range $opt := .Subcommands }}
  {{ printf "%-30s" $opt.Flag | color "yellow" "" }}{{ $opt.Description }}
  {{- end }}

{{ "QUERY OPTIONS" | color "" "heading" }}:
  {{- range $opt := .QueryOptions }}
  {{ printf "%-30s" $opt.Flag | color "yellow" "" }}{{ $opt.Description }}
  {{- end }}

{{ "RESOLVER OPTIONS" | color "" "heading" }}:
  {{- range $opt := .ResolverOptions }}
  {{ printf "%-30s" $opt.Flag | color "yellow" "" }}{{ $opt.Description }}
  {{- end }}

{{ "QUERY FLAGS" | color "" "heading" }}:
  {{- range $flag := .QueryFlags }}
  {{ printf "%-30s" $flag.Flag | color "yellow" "" }}{{ $flag.Description }}
  {{- end }}

{{ "OUTPUT OPTIONS" | color "" "heading" }}:
  {{- range $opt := .OutputOptions }}
  {{ printf "%-30s" $opt.Flag | color "yellow" "" }}{{ $opt.Description }}
  {{- end }}

{{ "GLOBALPING OPTIONS" | color "" "heading" }}:
  {{- range $opt := .GlobalPingOptions }}
  {{ printf "%-30s" $opt.Flag | color "yellow" "" }}{{ $opt.Description }}
  {{- end }}
`

func renderCustomHelp() {
	type Option struct {
		Flag        string
		Description string
	}

	type Example struct {
		Command     string
		Description string
	}

	type TransportOption struct {
		Scheme      string
		Example     string
		Description string
	}

	helpTmplVars := map[string]interface{}{
		"Name":        "doggo",
		"Description": "DNS Client for Humans",
		"Version":     buildVersion,
		"Date":        buildDate,
		"Examples": []Example{
			{"mrkaran.dev", "Query a domain using defaults."},
			{"mrkaran.dev CNAME", "Query for a CNAME record."},
			{"mrkaran.dev MX @9.9.9.9", "Uses a custom DNS resolver."},
			{"-q mrkaran.dev -t MX -n 1.1.1.1", "Using named arguments."},
			{"mrkaran.dev --aa --ad", "Query with Authoritative Answer and Authenticated Data flags set."},
			{"mrkaran.dev --cd --do", "Query with Checking Disabled and DNSSEC OK flags set."},
			{"mrkaran.dev --gp-from Germany", "Query using Globalping API from a specific location."},
		},
		"TransportOptions": []TransportOption{
			{"@udp://", "eg: @1.1.1.1", "initiates a UDP query to 1.1.1.1:53."},
			{"@tcp://", "eg: @tcp://1.1.1.1", "initiates a TCP query to 1.1.1.1:53."},
			{"@https://", "eg: @https://cloudflare-dns.com/dns-query", "initiates a DOH query to Cloudflare via DoH."},
			{"@tls://", "eg: @tls://1.1.1.1", "initiates a DoT query to 1.1.1.1:853."},
			{"@sdns://", "initiates a DNSCrypt or DoH query using a DNS stamp.", ""},
			{"@quic://", "initiates a DOQ query.", ""},
		},
		"Subcommands": []Option{
			{"completions [bash|zsh|fish]", "Generate the shell completion script for the specified shell."},
		},
		"QueryOptions": []Option{
			{"-q, --query=HOSTNAME", "Hostname to query the DNS records for (eg mrkaran.dev)."},
			{"-t, --type=TYPE", "Type of the DNS Record (A, MX, NS etc)."},
			{"-n, --nameserver=ADDR", "Address of a specific nameserver to send queries to (9.9.9.9, 8.8.8.8 etc)."},
			{"-c, --class=CLASS", "Network class of the DNS record (IN, CH, HS etc)."},
			{"-x, --reverse", "Performs a DNS Lookup for an IPv4 or IPv6 address. Sets the query type and class to PTR and IN respectively."},
			{"--any", "Query all supported DNS record types (A, AAAA, CNAME, MX, NS, PTR, SOA, SRV, TXT, CAA)."},
		},
		"ResolverOptions": []Option{
			{"--strategy=STRATEGY", "Specify strategy to query nameserver listed in etc/resolv.conf. (all, random, first)."},
			{"--ndots=INT", "Specify ndots parameter. Takes value from /etc/resolv.conf if using the system namesever or 1 otherwise."},
			{"--search", "Use the search list defined in resolv.conf. Defaults to true. Set --search=false to disable search list."},
			{"--timeout=DURATION", "Specify timeout for the resolver to return a response (e.g., 5s, 400ms, 1m)."},
			{"-4, --ipv4", "Use IPv4 only."},
			{"-6, --ipv6", "Use IPv6 only."},
			{"--tls-hostname=HOSTNAME", "Provide a hostname for verification of the certificate if the provided DoT nameserver is an IP."},
			{"--skip-hostname-verification", "Skip TLS Hostname Verification in case of DOT Lookups."},
		},
		"QueryFlags": []Option{
			{"--aa", "Set Authoritative Answer flag."},
			{"--ad", "Set Authenticated Data flag."},
			{"--cd", "Set Checking Disabled flag."},
			{"--rd", "Set Recursion Desired flag (default: true)."},
			{"--z", "Set Z flag (reserved for future use)."},
			{"--do", "Set DNSSEC OK flag."},
		},
		"OutputOptions": []Option{
			{"-J, --json", "Format the output as JSON."},
			{"--short", "Short output format. Shows only the response section."},
			{"--color", "Defaults to true. Set --color=false to disable colored output."},
			{"--debug", "Enable debug logging."},
			{"--time", "Shows how long the response took from the server."},
		},
		"GlobalPingOptions": []Option{
			{"--gp-from=Germany", "Query using Globalping API from a specific location."},
			{"--gp-limit=INT", "Limit the number of probes to use from Globalping."},
		},
	}

	tmpl, err := template.New("help").Funcs(template.FuncMap{
		"color": func(clr string, format string, str string) string {
			formatter := color.New()
			switch clr {
			case "yellow":
				formatter = formatter.Add(color.FgYellow)
			case "red":
				formatter = formatter.Add(color.FgRed)
			case "cyan":
				formatter = formatter.Add(color.FgCyan)
			case "green":
				formatter = formatter.Add(color.FgGreen)
			}
			switch format {
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
		panic(err)
	}
	err = tmpl.Execute(color.Output, helpTmplVars)
	if err != nil {
		panic(err)
	}
	os.Exit(0)
}
