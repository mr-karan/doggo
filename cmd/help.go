package main

import (
	"html/template"
	"os"
)

// AppHelpTemplate is the text template to customise the Help output.
// Uses text/template to render templates.
var AppHelpTemplate = `NAME:
  🐶 {{.Name}} - {{.Description}}

USAGE:
  {{.Name}} [OPTIONS] [--] <arguments>

VERSION:
  {{.Version}} Built at {{.Date}}

EXAMPLES:
  doggo mrkaran.dev                            	Query a domain using defaults
  doggo mrkaran.dev CNAME                      	Looks up for a CNAME record.
  doggo mrkaran.dev MX @9.9.9.9                	Uses a custom DNS resolver.
  doggo -q mrkaran.dev -t MX -n 1.1.1.1   	Using named arguments
`

func renderCustomHelp() {
	helpTmplVars := map[string]string{
		"Name":        "doggo",
		"Description": "DNS Client for Humans",
		"Version":     buildVersion,
		"Date":        buildDate,
	}
	tmpl, err := template.New("test").Parse(AppHelpTemplate)
	if err != nil {
		panic(err)
	}
	err = tmpl.Execute(os.Stdout, helpTmplVars)
	if err != nil {
		panic(err)
	}
	os.Exit(0)
}
