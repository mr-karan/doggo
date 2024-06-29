package main

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/knadh/koanf"
	"github.com/knadh/koanf/providers/posflag"
	"github.com/mr-karan/doggo/internal/app"
	"github.com/mr-karan/doggo/pkg/resolvers"
	"github.com/mr-karan/doggo/pkg/utils"
	"github.com/sirupsen/logrus"
	flag "github.com/spf13/pflag"
)

var (
	buildVersion = "unknown"
	buildDate    = "unknown"
	logger       = utils.InitLogger()
	k            = koanf.New(".")
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "completions" {
		completionsCommand()
		return
	}

	app := app.New(logger, buildVersion)
	f := setupFlags()

	if err := parseAndLoadFlags(f); err != nil {
		app.Logger.WithError(err).Error("Error parsing or loading flags")
		app.Logger.Exit(2)
	}

	if k.Bool("version") {
		fmt.Printf("%s - %s\n", buildVersion, buildDate)
		app.Logger.Exit(0)
	}

	setupLogging(&app)

	if err := k.Unmarshal("", &app.QueryFlags); err != nil {
		app.Logger.WithError(err).Error("Error loading args")
		app.Logger.Exit(2)
	}

	loadNameservers(&app, f.Args())

	if app.QueryFlags.ReverseLookup {
		app.ReverseLookup()
	}

	app.LoadFallbacks()
	app.PrepareQuestions()

	if err := app.LoadNameservers(); err != nil {
		app.Logger.WithError(err).Error("Error loading nameservers")
		app.Logger.Exit(2)
	}

	rslvrs, err := resolvers.LoadResolvers(resolvers.Options{
		Nameservers:        app.Nameservers,
		UseIPv4:            app.QueryFlags.UseIPv4,
		UseIPv6:            app.QueryFlags.UseIPv6,
		SearchList:         app.ResolverOpts.SearchList,
		Ndots:              app.ResolverOpts.Ndots,
		Timeout:            app.QueryFlags.Timeout * time.Second,
		Logger:             app.Logger,
		Strategy:           app.QueryFlags.Strategy,
		InsecureSkipVerify: app.QueryFlags.InsecureSkipVerify,
		TLSHostname:        app.QueryFlags.TLSHostname,
	})
	if err != nil {
		app.Logger.WithError(err).Error("Error loading resolver")
		app.Logger.Exit(2)
	}
	app.Resolvers = rslvrs

	app.Logger.Debug("Starting doggo 🐶")
	if len(app.QueryFlags.QNames) == 0 {
		f.Usage()
		app.Logger.Exit(0)
	}

	queryFlags := resolvers.QueryFlags{
		AA: k.Bool("aa"),
		AD: k.Bool("ad"),
		CD: k.Bool("cd"),
		RD: k.Bool("rd"),
		Z:  k.Bool("z"),
		DO: k.Bool("do"),
	}

	responses, responseErrors := resolveQueries(&app, queryFlags)

	outputResults(&app, responses, responseErrors)

	app.Logger.Exit(0)
}

func setupFlags() *flag.FlagSet {
	f := flag.NewFlagSet("config", flag.ContinueOnError)
	f.Usage = renderCustomHelp

	f.StringSliceP("query", "q", []string{}, "Domain name to query")
	f.StringSliceP("type", "t", []string{}, "Type of DNS record to be queried (A, AAAA, MX etc)")
	f.StringSliceP("class", "c", []string{}, "Network class of the DNS record to be queried (IN, CH, HS etc)")
	f.StringSliceP("nameserver", "n", []string{}, "Address of the nameserver to send packets to")
	f.BoolP("reverse", "x", false, "Performs a DNS Lookup for an IPv4 or IPv6 address")

	f.Int("timeout", 5, "Sets the timeout for a query to T seconds")
	f.Bool("search", true, "Use the search list provided in resolv.conf")
	f.Int("ndots", -1, "Specify the ndots parameter")
	f.BoolP("ipv4", "4", false, "Use IPv4 only")
	f.BoolP("ipv6", "6", false, "Use IPv6 only")
	f.String("strategy", "all", "Strategy to query nameservers in resolv.conf file")
	f.String("tls-hostname", "", "Hostname for certificate verification")
	f.Bool("skip-hostname-verification", false, "Skip TLS Hostname Verification")

	f.BoolP("json", "J", false, "Set the output format as JSON")
	f.Bool("short", false, "Short output format")
	f.Bool("time", false, "Display how long the response took")
	f.Bool("color", true, "Show colored output")
	f.Bool("debug", false, "Enable debug mode")

	// Add flags for DNS query options
	f.Bool("aa", false, "Set Authoritative Answer flag")
	f.Bool("ad", false, "Set Authenticated Data flag")
	f.Bool("cd", false, "Set Checking Disabled flag")
	f.Bool("rd", true, "Set Recursion Desired flag (default: true)")
	f.Bool("z", false, "Set Z flag (reserved for future use)")
	f.Bool("do", false, "Set DNSSEC OK flag")

	f.Bool("version", false, "Show version of doggo")

	return f
}

func parseAndLoadFlags(f *flag.FlagSet) error {
	if err := f.Parse(os.Args[1:]); err != nil {
		return fmt.Errorf("error parsing flags: %w", err)
	}
	if err := k.Load(posflag.Provider(f, ".", k), nil); err != nil {
		return fmt.Errorf("error loading flags: %w", err)
	}
	return nil
}

func setupLogging(app *app.App) {
	if k.Bool("debug") {
		app.Logger.SetLevel(logrus.DebugLevel)
	} else {
		app.Logger.SetLevel(logrus.InfoLevel)
	}
}

func loadNameservers(app *app.App, args []string) {
	flagNameservers := k.Strings("nameserver")
	app.Logger.WithField("flagNameservers", flagNameservers).Debug("Nameservers from -n flag")

	unparsedNameservers, qt, qc, qn := loadUnparsedArgs(args)
	app.Logger.WithField("unparsedNameservers", unparsedNameservers).Debug("Nameservers from unparsed arguments")

	if len(flagNameservers) > 0 {
		app.QueryFlags.Nameservers = flagNameservers
	} else {
		app.QueryFlags.Nameservers = unparsedNameservers
	}

	app.QueryFlags.QTypes = append(app.QueryFlags.QTypes, qt...)
	app.QueryFlags.QClasses = append(app.QueryFlags.QClasses, qc...)
	app.QueryFlags.QNames = append(app.QueryFlags.QNames, qn...)

	app.Logger.WithField("finalNameservers", app.QueryFlags.Nameservers).Debug("Final nameservers")
}

func resolveQueries(app *app.App, flags resolvers.QueryFlags) ([]resolvers.Response, []error) {
	var responses []resolvers.Response
	var responseErrors []error

	for _, q := range app.Questions {
		for _, rslv := range app.Resolvers {
			resp, err := rslv.Lookup(q, flags)
			if err != nil {
				responseErrors = append(responseErrors, err)
			}
			responses = append(responses, resp)
		}
	}

	return responses, responseErrors
}

func outputResults(app *app.App, responses []resolvers.Response, responseErrors []error) {
	if app.QueryFlags.ShowJSON {
		outputJSON(responses, responseErrors)
	} else {
		if len(responseErrors) > 0 {
			app.Logger.WithError(responseErrors[0]).Error("Error looking up DNS records")
			app.Logger.Exit(9)
		}
		app.Output(responses)
	}
}

func outputJSON(responses []resolvers.Response, responseErrors []error) {
	jsonOutput := struct {
		Responses []resolvers.Response `json:"responses,omitempty"`
		Error     string               `json:"error,omitempty"`
	}{
		Responses: responses,
	}

	if len(responseErrors) > 0 {
		jsonOutput.Error = responseErrors[0].Error()
	}

	jsonData, err := json.MarshalIndent(jsonOutput, "", "  ")
	if err != nil {
		logger.WithError(err).Error("Error marshaling JSON")
		logger.Exit(1)
	}
	fmt.Println(string(jsonData))
}
