package app

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/jsdelivr/globalping-cli/globalping"
	"github.com/mr-karan/doggo/pkg/resolvers"
	"github.com/olekukonko/tablewriter"
)

var (
	ErrTargetIPVersionNotAllowed   = errors.New("ipVersion is not allowed when target is not a domain")
	ErrResolverIPVersionNotAllowed = errors.New("ipVersion is not allowed when resolver is not a domain")
)

func (app *App) GlobalpingMeasurement() (*globalping.Measurement, error) {
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
		Type:      "dns",
		Target:    target,
		Limit:     app.QueryFlags.GPLimit,
		Locations: parseGlobalpingLocations(app.QueryFlags.GPFrom),
		Options: &globalping.MeasurementOptions{
			Protocol: protocol,
			Port:     port,
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
