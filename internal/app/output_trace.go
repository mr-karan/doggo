package app

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/fatih/color"
	"github.com/mr-karan/doggo/pkg/resolvers"
)

// maxDelegationTargetsShown caps how many delegated nameserver targets are
// rendered inline in the human trace output before collapsing the remainder
// into a "+ N more" suffix.
const maxDelegationTargetsShown = 4

// traceJSONBody mirrors resolvers.TraceResult without its SchemaVersion field
// so that the rendered JSON never repeats schema_version inside the nested
// "trace" object.
type traceJSONBody struct {
	Query   resolvers.TraceQuestion `json:"query"`
	Status  resolvers.TraceStatus   `json:"status"`
	Verdict resolvers.TraceVerdict  `json:"verdict"`
	Hops    []resolvers.TraceHop    `json:"hops"`
	Summary resolvers.TraceSummary  `json:"summary"`
	Error   *resolvers.TraceError   `json:"error,omitempty"`
}

// traceJSONOutput is the exact top-level shape documented in
// docs/trace-design.md: schema_version plus a single "trace" object.
type traceJSONOutput struct {
	SchemaVersion int           `json:"schema_version"`
	Trace         traceJSONBody `json:"trace"`
}

// OutputTrace renders a resolvers.TraceResult in the format selected by the
// current QueryFlags (JSON, short, or the default human terminal renderer).
// It never terminates the process; callers decide how to react to errors and
// exit codes.
func (app *App) OutputTrace(result resolvers.TraceResult) error {
	switch {
	case app.QueryFlags.ShowJSON:
		return app.outputTraceJSON(result)
	case app.QueryFlags.ShortOutput:
		return app.outputTraceShort(result)
	default:
		return app.outputTraceTerminal(result)
	}
}

func (app *App) outputTraceJSON(result resolvers.TraceResult) error {
	hops := result.Hops
	if hops == nil {
		hops = []resolvers.TraceHop{}
	}
	out := traceJSONOutput{
		SchemaVersion: result.SchemaVersion,
		Trace: traceJSONBody{
			Query:   result.Query,
			Status:  result.Status,
			Verdict: result.Verdict,
			Hops:    hops,
			Summary: result.Summary,
			Error:   result.Error,
		},
	}

	res, err := json.MarshalIndent(out, "", "    ")
	if err != nil {
		return fmt.Errorf("unable to marshal trace output as JSON: %w", err)
	}
	_, err = fmt.Printf("%s\n", res)
	return err
}

// outputTraceShort prints one tab-separated line per successful hop
// (referral/answer/nxdomain/nodata/cname), followed by the final answer
// record data, matching regular --short semantics.
func (app *App) outputTraceShort(result resolvers.TraceResult) error {
	for _, hop := range result.Hops {
		if hop.Outcome == resolvers.TraceOutcomeError || hop.Outcome == "" {
			continue
		}
		attempt := lastSuccessfulTraceAttempt(hop.Attempts)
		var ns, ip, protocol, rtt, rcode string
		if attempt != nil {
			ns = attempt.Nameserver
			ip = attempt.IP
			protocol = attempt.Protocol
			rtt = fmt.Sprintf("%dms", attempt.RTTMS)
			rcode = attempt.RCode
		}
		if _, err := fmt.Printf("%s\t%s\t%s\t%s\t%s\t%s\t%s\n", hop.Zone, string(hop.Outcome), ns, ip, protocol, rtt, rcode); err != nil {
			return err
		}
	}

	if len(result.Hops) == 0 {
		return nil
	}
	last := result.Hops[len(result.Hops)-1]
	if last.Outcome != resolvers.TraceOutcomeAnswer {
		return nil
	}
	for _, rec := range last.Answers {
		if _, err := fmt.Printf("%s\n", rec.Data); err != nil {
			return err
		}
	}
	return nil
}

// outputTraceTerminal renders a human-readable, vertically structured trace:
// one block per hop showing the contacted server(s), every attempt (success
// or failure), delegation details, and the terminal result, followed by a
// concise summary line.
func (app *App) outputTraceTerminal(result resolvers.TraceResult) error {
	if !app.QueryFlags.Color {
		color.NoColor = true
	}
	var output strings.Builder
	w := &output

	_, _ = fmt.Fprintf(w, "TRACE  %s  %s  %s\n", result.Query.Name, result.Query.Type, result.Query.Class)

	for _, hop := range result.Hops {
		_, _ = fmt.Fprintln(w)
		_, _ = fmt.Fprintf(w, "%2d  %-38s %s\n", hop.Number, hop.Zone, formatTraceHopRole(hop))

		for _, attempt := range hop.Attempts {
			_, _ = fmt.Fprintf(w, "    %s\n", formatTraceAttempt(attempt))
		}

		if hop.Delegation != nil {
			_, _ = fmt.Fprintf(w, "    %s\n", formatTraceDelegation(hop.Delegation))
		}

		switch hop.Outcome {
		case resolvers.TraceOutcomeAnswer:
			_, _ = fmt.Fprintf(w, "\n    %s\n", TerminalColorGreen("ANSWER"))
			for _, rec := range hop.Answers {
				_, _ = fmt.Fprintf(w, "    %s\n", formatTraceRecord(rec))
			}
		case resolvers.TraceOutcomeCNAME:
			_, _ = fmt.Fprintf(w, "\n    %s\n", TerminalColorYellow("ALIAS"))
			for _, rec := range hop.Answers {
				_, _ = fmt.Fprintf(w, "    %s\n", formatTraceRecord(rec))
			}
		case resolvers.TraceOutcomeNXDOMAIN:
			_, _ = fmt.Fprintf(w, "\n    %s\n", TerminalColorRed("NXDOMAIN"))
			for _, rec := range hop.Authorities {
				_, _ = fmt.Fprintf(w, "    %s\n", formatTraceRecord(rec))
			}
		case resolvers.TraceOutcomeNODATA:
			_, _ = fmt.Fprintf(w, "\n    %s\n", TerminalColorYellow("NODATA"))
			for _, rec := range hop.Authorities {
				_, _ = fmt.Fprintf(w, "    %s\n", formatTraceRecord(rec))
			}
		case resolvers.TraceOutcomeError:
			if len(hop.Attempts) == 0 && result.Error != nil {
				_, _ = fmt.Fprintf(w, "    %s\n", TerminalColorRed(fmt.Sprintf("ERROR: %s", result.Error.Error())))
			}
		}
	}

	_, _ = fmt.Fprintln(w)
	_, _ = fmt.Fprintf(w, "%s\n", formatTraceSummary(result))
	_, err := io.WriteString(color.Output, output.String())
	return err
}

// formatTraceHopRole renders the role column of a hop header, e.g.
// "root", "delegation", "authoritative [ok]", "delegation [error]".
func formatTraceHopRole(hop resolvers.TraceHop) string {
	label := string(hop.Role)
	tag := traceOutcomeTag(hop.Outcome)
	if tag != "" {
		label = fmt.Sprintf("%s [%s]", label, tag)
	}

	switch hop.Outcome {
	case resolvers.TraceOutcomeError:
		return TerminalColorRed(label)
	case resolvers.TraceOutcomeNXDOMAIN, resolvers.TraceOutcomeNODATA:
		return TerminalColorYellow(label)
	default:
		return TerminalColorCyan(label)
	}
}

func traceOutcomeTag(outcome resolvers.TraceOutcome) string {
	switch outcome {
	case resolvers.TraceOutcomeAnswer:
		return "ok"
	case resolvers.TraceOutcomeCNAME:
		return "alias"
	case resolvers.TraceOutcomeNXDOMAIN:
		return "nxdomain"
	case resolvers.TraceOutcomeNODATA:
		return "nodata"
	case resolvers.TraceOutcomeReferral:
		return "referral"
	case resolvers.TraceOutcomeError:
		return "error"
	default:
		return ""
	}
}

// formatTraceAttempt renders a single dialed attempt, whether it succeeded
// or failed. Failed attempts remain visible so operational retries are
// never hidden from the human renderer.
func formatTraceAttempt(attempt resolvers.TraceAttempt) string {
	target := attempt.Nameserver
	if attempt.IP != "" {
		target = fmt.Sprintf("%s (%s)", attempt.Nameserver, attempt.IP)
	}
	if target == "" {
		target = "(no address)"
	}

	if attempt.Error != nil {
		detail := sanitizeEDEExtraText(attempt.Error.Error())
		return fmt.Sprintf("-> %s  %s  %dms  %s", target, attempt.Protocol, attempt.RTTMS, TerminalColorRed(fmt.Sprintf("FAILED %s", detail)))
	}
	suffix := ""
	if attempt.Truncated {
		suffix = "  " + TerminalColorYellow("truncated; retrying over TCP")
	}
	return fmt.Sprintf("-> %s  %s  %dms  %s%s", target, attempt.Protocol, attempt.RTTMS, attempt.RCode, suffix)
}

// formatTraceDelegation renders the child zone and the sorted nameserver
// targets (with any discovered address hints) that a referral delegated to.
func formatTraceDelegation(d *resolvers.TraceDelegation) string {
	entries := make([]string, 0, len(d.Nameservers))
	for _, ns := range d.Nameservers {
		entry := ns.Name
		if len(ns.Addresses) > 0 {
			entry = fmt.Sprintf("%s (%s)", ns.Name, strings.Join(ns.Addresses, ", "))
		}
		entries = append(entries, entry)
	}

	shown := entries
	suffix := ""
	if len(entries) > maxDelegationTargetsShown {
		shown = entries[:maxDelegationTargetsShown]
		suffix = fmt.Sprintf(" + %d more", len(entries)-maxDelegationTargetsShown)
	}

	return fmt.Sprintf("delegates %s to %s%s", d.Child, strings.Join(shown, ", "), suffix)
}

// formatTraceRecord renders a single answer/authority record line.
func formatTraceRecord(rec resolvers.TraceRecord) string {
	return fmt.Sprintf("%s  %s  %ds  %s", rec.Name, getColoredType(rec.Type), rec.TTL, sanitizeEDEExtraText(rec.Data))
}

// formatTraceSummary renders the concise closing line describing hop count,
// total RTT, and the terminal verdict or operational error.
func formatTraceSummary(result resolvers.TraceResult) string {
	var desc string
	switch result.Status {
	case resolvers.TraceStatusComplete:
		ns := lastSuccessfulTraceNameserver(result.Hops)
		switch result.Verdict {
		case resolvers.TraceVerdictAnswer:
			desc = TerminalColorGreen(fmt.Sprintf("answer from %s", ns))
		case resolvers.TraceVerdictNXDOMAIN:
			desc = TerminalColorYellow(fmt.Sprintf("nxdomain from %s", ns))
		case resolvers.TraceVerdictNODATA:
			desc = TerminalColorYellow(fmt.Sprintf("nodata from %s", ns))
		default:
			desc = TerminalColorGreen("complete")
		}
	case resolvers.TraceStatusPartial:
		desc = TerminalColorRed(fmt.Sprintf("partial: %s", traceErrorText(result.Error)))
	default:
		desc = TerminalColorRed(fmt.Sprintf("failed: %s", traceErrorText(result.Error)))
	}

	return fmt.Sprintf("-- %d hops · %dms total · %s --", result.Summary.HopCount, result.Summary.TotalRTTMS, desc)
}

func traceErrorText(err *resolvers.TraceError) string {
	if err == nil {
		return "unknown error"
	}
	return sanitizeEDEExtraText(err.Error())
}

func lastSuccessfulTraceAttempt(attempts []resolvers.TraceAttempt) *resolvers.TraceAttempt {
	for i := len(attempts) - 1; i >= 0; i-- {
		if attempts[i].Error == nil {
			return &attempts[i]
		}
	}
	return nil
}

func lastSuccessfulTraceNameserver(hops []resolvers.TraceHop) string {
	if len(hops) == 0 {
		return ""
	}
	last := hops[len(hops)-1]
	if attempt := lastSuccessfulTraceAttempt(last.Attempts); attempt != nil {
		return attempt.Nameserver
	}
	return ""
}
