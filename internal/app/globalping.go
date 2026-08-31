package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"

	"github.com/fatih/color"
	"github.com/jsdelivr/globalping-go"
	"github.com/mr-karan/doggo/pkg/resolvers"
	"github.com/olekukonko/tablewriter"
	"github.com/olekukonko/tablewriter/tw"
)

var (
	ErrTargetIPVersionNotAllowed   = errors.New("ipVersion is not allowed when target is not a domain")
	ErrResolverIPVersionNotAllowed = errors.New("ipVersion is not allowed when resolver is not a domain")
	ErrGlobalpingTargetRequired    = errors.New("a target is required for globalping")
)

func (app *App) GlobalpingMeasurement() (*globalping.Measurement, error) {
	ctx := context.Background()

	if len(app.QueryFlags.QNames) == 0 {
		return nil, ErrGlobalpingTargetRequired
	}
	if len(app.QueryFlags.QNames) > 1 {
		return nil, errors.New("only one target is allowed for globalping")
	}
	if len(app.QueryFlags.QTypes) > 1 {
		return nil, errors.New("only one query type is allowed for globalping")
	}

	target := app.QueryFlags.QNames[0]
	resolver, port, protocol, err := parseGlobalpingResolver(app.QueryFlags.Nameservers)
	if err != nil {
		return nil, err
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
		Type:      globalping.MeasurementTypeDNS,
		Target:    target,
		Limit:     app.QueryFlags.GPLimit,
		Locations: parseGlobalpingLocations(app.QueryFlags.GPFrom),
		Options: &globalping.MeasurementOptions{
			Protocol: protocol,
			Port:     uint16(port),
		},
	}
	if app.QueryFlags.UseIPv4 {
		o.Options.IPVersion = globalping.IPVersion4
	} else if app.QueryFlags.UseIPv6 {
		o.Options.IPVersion = globalping.IPVersion6
	}
	if resolver != "" {
		o.Options.Resolver = resolver
	}
	if len(app.QueryFlags.QTypes) > 0 {
		o.Options.Query = &globalping.QueryOptions{
			Type: app.QueryFlags.QTypes[0],
		}
	}
	res, err := app.globalping.CreateMeasurement(ctx, o)
	if err != nil {
		return nil, err
	}
	measurement, err := app.globalping.AwaitMeasurement(ctx, res.ID)
	if err != nil {
		return nil, err
	}

	if measurement.Status != globalping.MeasurementStatusFinished {
		return nil, fmt.Errorf("globalping measurement did not complete successfully (status: %s)", measurement.Status)
	}
	return measurement, nil
}

func (app *App) OutputGlobalping(m *globalping.Measurement) error {
	// Disables colorized output if user specified.
	if !app.QueryFlags.Color {
		color.NoColor = true
	}

	table := tablewriter.NewWriter(color.Output)
	table.Options(
		tablewriter.WithRendition(tw.Rendition{
			Borders: tw.Border{
				Left:   tw.Off,
				Right:  tw.Off,
				Top:    tw.Off,
				Bottom: tw.Off,
			},
			Settings: tw.Settings{
				Separators: tw.Separators{
					ShowHeader:     tw.Off,
					ShowFooter:     tw.Off,
					BetweenRows:    tw.Off,
					BetweenColumns: tw.Off,
				},
				Lines: tw.Lines{
					ShowTop:        tw.Off,
					ShowBottom:     tw.Off,
					ShowHeaderLine: tw.Off,
					ShowFooterLine: tw.Off,
				},
			},
			Symbols: tw.NewSymbols(tw.StyleLight),
		}),
		tablewriter.WithPadding(tw.Padding{Left: "", Right: "  ", Overwrite: true}),
		tablewriter.WithRowAutoWrap(tw.WrapNormal),
		tablewriter.WithRowMaxWidth(30),
		tablewriter.WithHeaderAlignment(tw.AlignLeft),
	)

	// Formatting options for the table.
	table.Header("Location", "Name", "Type", "Class", "TTL", "Address", "Nameserver")

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

func (app *App) OutputGlobalpingShort(m *globalping.Measurement) error {
	for i := range m.Results {
		fmt.Printf("%s\n", getGlobalPingLocationText(&m.Results[i]))
		answers, err := globalping.DecodeDNSAnswers(m.Results[i].Result.AnswersRaw)
		if err != nil {
			return err
		}
		for _, ans := range answers {
			fmt.Printf("%s\n", ans.Value)
		}
	}
	return nil
}

type GlobalpingOutputResponse struct {
	Location string             `json:"location"`
	Answers  []resolvers.Answer `json:"answers"`
}

func (app *App) OutputGlobalpingJSON(m *globalping.Measurement) error {
	jsonOutput := struct {
		Responses []GlobalpingOutputResponse `json:"responses"`
	}{
		Responses: make([]GlobalpingOutputResponse, 0, len(m.Results)),
	}
	for i := range m.Results {
		jsonOutput.Responses = append(jsonOutput.Responses, GlobalpingOutputResponse{})
		jsonOutput.Responses[i].Location = getGlobalPingLocationText(&m.Results[i])
		answers, err := globalping.DecodeDNSAnswers(m.Results[i].Result.AnswersRaw)
		if err != nil {
			return err
		}
		resolver := m.Results[i].Result.Resolver
		for _, ans := range answers {
			jsonOutput.Responses[i].Answers = append(jsonOutput.Responses[i].Answers, resolvers.Answer{
				Name:       ans.Name,
				Type:       ans.Type,
				Class:      ans.Class,
				TTL:        fmt.Sprintf("%ds", ans.TTL),
				Address:    ans.Value,
				Nameserver: resolver,
			})
		}
	}

	// Pretty print with 4 spaces.
	res, err := json.MarshalIndent(jsonOutput, "", "    ")
	if err != nil {
		return err
	}
	fmt.Printf("%s\n", res)
	return nil
}

func parseGlobalpingLocations(from string) globalping.LocationOptions {
	if from == "" {
		return globalping.LocationOptions{
			{
				Magic: "world",
			},
		}
	}
	fromArr := strings.Split(from, ",")
	locations := make(globalping.LocationOptions, len(fromArr))
	for i, v := range fromArr {
		locations[i] = globalping.Locations{
			Magic: strings.TrimSpace(v),
		}
	}
	return locations
}

func getGlobalPingLocationText(m *globalping.ProbeMeasurement) string {
	state := ""
	// State became an optional *string in globalping-go v0.3.0 (nil when the
	// probe has no US state).
	if m.Probe.State != nil && *m.Probe.State != "" {
		state = " (" + *m.Probe.State + ")"
	}
	return m.Probe.City + state + ", " +
		m.Probe.Country + ", " +
		m.Probe.Continent + ", " +
		m.Probe.Network + " " +
		"(AS" + fmt.Sprint(m.Probe.ASN) + ")"
}

// parses the resolver string and returns the hostname, port, and protocol.
func parseGlobalpingResolver(nameservers []string) (string, int, string, error) {
	port := 53
	protocol := "udp"
	if len(nameservers) == 0 {
		return "", port, protocol, nil
	}

	if len(nameservers) > 1 {
		return "", 0, "", errors.New("only one resolver is allowed for globalping")
	}

	u, err := url.Parse(nameservers[0])
	if err != nil {
		return "", 0, "", err
	}
	if u.Port() != "" {
		port, err = strconv.Atoi(u.Port())
		if err != nil {
			return "", 0, "", err
		}
	}
	switch u.Scheme {
	case "tcp":
		protocol = "tcp"
	}

	return u.Hostname(), port, protocol, nil
}
