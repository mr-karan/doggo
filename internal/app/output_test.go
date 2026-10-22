package app

import (
	"errors"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/fatih/color"
	"github.com/jsdelivr/globalping-go"
	"github.com/mr-karan/doggo/pkg/models"
	"github.com/mr-karan/doggo/pkg/resolvers"
)

func TestOutputReportsWriteErrors(t *testing.T) {
	output, err := os.CreateTemp(t.TempDir(), "closed-output")
	if err != nil {
		t.Fatal(err)
	}
	if err := output.Close(); err != nil {
		t.Fatal(err)
	}
	previousStdout, previousColorOutput, previousNoColor := os.Stdout, color.Output, color.NoColor
	os.Stdout, color.Output = output, output
	t.Cleanup(func() {
		os.Stdout, color.Output, color.NoColor = previousStdout, previousColorOutput, previousNoColor
	})
	responses := []resolvers.Response{{Answers: []resolvers.Answer{{Name: "example.test.", Type: "A", Address: "192.0.2.1"}}}}
	trace := resolvers.TraceResult{Hops: []resolvers.TraceHop{{Outcome: resolvers.TraceOutcomeAnswer}}}
	for _, test := range []struct {
		name string
		run  func() error
	}{
		{"terminal", func() error { return (&App{}).Output(responses) }},
		{"short", func() error { return (&App{QueryFlags: models.QueryFlags{ShortOutput: true}}).Output(responses) }},
		{"json", func() error { return (&App{QueryFlags: models.QueryFlags{ShowJSON: true}}).Output(responses) }},
		{"trace", func() error { return (&App{}).OutputTrace(resolvers.TraceResult{}) }},
		{"trace short", func() error { return (&App{QueryFlags: models.QueryFlags{ShortOutput: true}}).OutputTrace(trace) }},
		{"trace json", func() error { return (&App{QueryFlags: models.QueryFlags{ShowJSON: true}}).OutputTrace(trace) }},
		{"globalping", func() error { return (&App{}).OutputGlobalping(&globalping.Measurement{}) }},
		{"globalping short", func() error {
			return (&App{}).OutputGlobalpingShort(&globalping.Measurement{Results: []globalping.ProbeMeasurement{{}}})
		}},
		{"globalping json", func() error { return (&App{}).OutputGlobalpingJSON(&globalping.Measurement{}) }},
	} {
		t.Run(test.name, func(t *testing.T) {
			if err := test.run(); !errors.Is(err, os.ErrClosed) {
				t.Fatalf("output error = %v, want closed output error", err)
			}
		})
	}
}

func TestFormatExtendedError(t *testing.T) {
	tests := []struct {
		name string
		ede  resolvers.ExtendedError
		want string
	}{
		{
			name: "known code with extra text",
			ede: resolvers.ExtendedError{
				Code:        22,
				Description: "No Reachable Authority",
				ExtraText:   "time limit exceeded",
			},
			want: "22 (No Reachable Authority): time limit exceeded",
		},
		{
			name: "known code without extra text",
			ede: resolvers.ExtendedError{
				Code:        3,
				Description: "Stale Answer",
			},
			want: "3 (Stale Answer)",
		},
		{
			name: "unknown code",
			ede: resolvers.ExtendedError{
				Code:      65000,
				ExtraText: "private error",
			},
			want: "65000: private error",
		},
		{
			name: "control sequences are escaped",
			ede: resolvers.ExtendedError{
				Code:        22,
				Description: "No Reachable Authority",
				ExtraText:   "line one\n\x1b[31mred\x1b]0;title\x07\u0085",
			},
			want: `22 (No Reachable Authority): line one\n\x1b[31mred\x1b]0;title\x07\x85`,
		},
		{
			name: "long extra text is truncated",
			ede: resolvers.ExtendedError{
				Code:      65000,
				ExtraText: strings.Repeat("x", maxEDEExtraTextDisplayLength+10),
			},
			want: "65000: " + strings.Repeat("x", maxEDEExtraTextDisplayLength-1) + "…",
		},
		{
			name: "bidi and zero-width format characters are escaped",
			ede: resolvers.ExtendedError{
				Code: 65000,
				ExtraText: "start\u00ad\u200b\u200c\u200d\u200e\u200f" +
					"\u202a\u202b\u202c\u202d\u202e" +
					"\u2060\u2066\u2067\u2068\u2069\ufeff\U000e0001end",
			},
			want: `65000: start\u00ad\u200b\u200c\u200d\u200e\u200f` +
				`\u202a\u202b\u202c\u202d\u202e` +
				`\u2060\u2066\u2067\u2068\u2069\ufeff\U000e0001end`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := formatExtendedError(tt.ede); got != tt.want {
				t.Errorf("formatExtendedError() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestOutputTerminalDisplaysAllExtendedErrors(t *testing.T) {
	readOut, writeOut, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe() error = %v", err)
	}

	previousStdout := os.Stdout
	previousColorOutput := color.Output
	previousNoColor := color.NoColor
	os.Stdout = writeOut
	color.Output = writeOut
	t.Cleanup(func() {
		os.Stdout = previousStdout
		color.Output = previousColorOutput
		color.NoColor = previousNoColor
		_ = readOut.Close()
		_ = writeOut.Close()
	})

	app := App{QueryFlags: models.QueryFlags{Color: false}}
	err = app.outputTerminal([]resolvers.Response{
		{
			Edns: &resolvers.EdnsInfo{
				Nameserver: "resolver-a:53",
				NSID:       "nsid-a",
				UDPSize:    1232,
			},
		},
		{
			Edns: &resolvers.EdnsInfo{
				Nameserver: "resolver-a:53",
				ExtendedErrors: []resolvers.ExtendedError{
					{Code: 3, Description: "Stale Answer"},
					{Code: 22, Description: "No Reachable Authority", ExtraText: "time limit exceeded"},
				},
			},
		},
		{
			Edns: &resolvers.EdnsInfo{
				Nameserver: "resolver-b:53",
				NSID:       "nsid-b",
				UDPSize:    4096,
				ExtendedErrors: []resolvers.ExtendedError{
					{Code: 15, Description: "Blocked", ExtraText: "local policy"},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("outputTerminal: %v", err)
	}

	if err := writeOut.Close(); err != nil {
		t.Fatalf("stdout close error = %v", err)
	}
	output, err := io.ReadAll(readOut)
	if err != nil {
		t.Fatalf("stdout read error = %v", err)
	}

	outputText := string(output)
	position := 0
	for _, want := range []string{
		"Nameserver: resolver-a:53",
		"NSID: nsid-a",
		"UDP Size: 1232",
		"Extended Error: 3 (Stale Answer)",
		"Extended Error: 22 (No Reachable Authority): time limit exceeded",
		"Nameserver: resolver-b:53",
		"NSID: nsid-b",
		"UDP Size: 4096",
		"Extended Error: 15 (Blocked): local policy",
	} {
		next := strings.Index(outputText[position:], want)
		if next < 0 {
			t.Fatalf("terminal output missing %q in order after byte %d:\n%s", want, position, outputText)
		}
		position += next + len(want)
	}
	if got := strings.Count(outputText, "Nameserver: resolver-a:53"); got != 1 {
		t.Errorf("terminal output has resolver-a metadata %d times, want once:\n%s", got, outputText)
	}
	if got := strings.Count(outputText, "Extended Error:"); got != 3 {
		t.Errorf("terminal output has %d EDE lines, want 3:\n%s", got, outputText)
	}
}
