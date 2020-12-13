package main

import (
	"fmt"
	"io"
)

// Override Help Template
var helpTmpl = `NAME:
   {{.Name}}
   {{ range $key, $value := . }}
   <li><strong>{{ $key }}</strong>: {{ $value }}</li>
{{ end }}
`

func renderCustomHelp(w io.Writer, templ string, data interface{}) {
	var helpTmplVars = map[string]string{}

	helpTmplVars["Name"] = "doggo"
	fmt.Fprintf(w, helpTmpl, helpTmplVars)
}
