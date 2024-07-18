package app

import (
	"errors"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/jsdelivr/globalping-cli/globalping"
	"github.com/olekukonko/tablewriter"
)

var (
	ErrTargetIPVersionNotAllowed   = errors.New("ipVersion is not allowed when target is not a domain")
	ErrResolverIPVersionNotAllowed = errors.New("ipVersion is not allowed when resolver is not a domain")
)

func (app *App) GlobalpingMeasurement() (*globalping.Measurement, error) {
	target := app.QueryFlags.QNames[0]
	resolver := ""
	if len(app.QueryFlags.Nameservers) > 0 {
		resolver = app.QueryFlags.Nameservers[0]
	}

	if app.QueryFlags.UseIPv4 || app.QueryFlags.UseIPv6 {
		if net.ParseIP(target) != nil {
			return nil, ErrTargetIPVersionNotAllowed
		}
		if resolver != "" && net.ParseIP(resolver) != nil {
			return nil, ErrResolverIPVersionNotAllowed
		}
	}

	o := &globalping.MeasurementCreate{
		Type:      "dns",
		Target:    target,
		Limit:     app.QueryFlags.Limit,
		Locations: parseGlobalpingLocations(app.QueryFlags.From),
		Options:   &globalping.MeasurementOptions{
			// TODO: Add support for these flags.
			// Protocol: opts.Protocol,
			// Port:     opts.Port,
		},
	}
	if app.QueryFlags.UseIPv4 {
		o.Options.IPVersion = globalping.IPVersion4
	} else if app.QueryFlags.UseIPv6 {
		o.Options.IPVersion = globalping.IPVersion6
	}
	if len(app.QueryFlags.Nameservers) > 0 {
		o.Options.Resolver = app.QueryFlags.Nameservers[0]
	}
	if len(app.QueryFlags.QTypes) > 0 {
		o.Options.Query = &globalping.QueryOptions{
			Type: app.QueryFlags.QTypes[0],
		}
	}
	res, err := app.globalping.CreateMeasurement(o)
	if err != nil {
		return nil, err
	}
	measurement, err := app.globalping.GetMeasurement(res.ID)
	if err != nil {
		return nil, err
	}
	for measurement.Status == globalping.StatusInProgress {
		time.Sleep(500 * time.Millisecond)
		measurement, err = app.globalping.GetMeasurement(res.ID)
		if err != nil {
			return nil, err
		}
	}

	if measurement.Status != globalping.StatusFinished {
		return nil, &globalping.MeasurementError{
			Message: "measurement did not complete successfully",
		}
	}
	return measurement, nil
}

// TODO: Add support for json output && short output
func (app *App) OutputGlobalping(m *globalping.Measurement) error {
	// Disables colorized output if user specified.
	if !app.QueryFlags.Color {
		color.NoColor = true
	}

	table := tablewriter.NewWriter(color.Output)
	header := []string{"Location", "Name", "Type", "Class", "TTL", "Address", "Nameserver"}

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

	for i := range m.Results {
		table.Append([]string{getGlobalPingLocationText(&m.Results[i]), "", "", "", "", "", ""})
		answers, err := globalping.DecodeDNSAnswers(m.Results[i].Result.AnswersRaw)
		if err != nil {
			return err
		}
		resolver := m.Results[i].Result.Resolver
		for _, ans := range answers {
			typOut := getColoredType(ans.Type)
			output := []string{"", TerminalColorGreen(ans.Name), typOut, ans.Class, fmt.Sprintf("%ds", ans.TTL), ans.Value, resolver}
			table.Append(output)
		}
	}
	table.Render()
	return nil
}

func parseGlobalpingLocations(from string) []globalping.Locations {
	if from == "" {
		return []globalping.Locations{
			{
				Magic: "world",
			},
		}
	}
	fromArr := strings.Split(from, ",")
	locations := make([]globalping.Locations, len(fromArr))
	for i, v := range fromArr {
		locations[i] = globalping.Locations{
			Magic: strings.TrimSpace(v),
		}
	}
	return locations
}

func getGlobalPingLocationText(m *globalping.ProbeMeasurement) string {
	state := ""
	if m.Probe.State != "" {
		state = " (" + m.Probe.State + ")"
	}
	return m.Probe.City + state + ", " +
		m.Probe.Country + ", " +
		m.Probe.Continent + ", " +
		m.Probe.Network + " " +
		"(AS" + fmt.Sprint(m.Probe.ASN) + ")"
}
