package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"math"
	"os"
	"sync"
	"time"

	"github.com/jsdelivr/globalping-cli/globalping"
	"github.com/knadh/koanf/providers/posflag"
	"github.com/knadh/koanf/v2"
	"github.com/mr-karan/doggo/internal/app"
	"github.com/mr-karan/doggo/pkg/resolvers"
	"github.com/mr-karan/doggo/pkg/utils"
	flag "github.com/spf13/pflag"
)

var (
	buildVersion = "unknown"
	buildDate    = "unknown"
	k            = koanf.New(".")
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "completions" {
		completionsCommand()
		return
	}

	cfg, err := loadConfig()
	if err != nil {
		fmt.Printf("Error loading configuration: %v\n", err)
		os.Exit(1)
	}

	if cfg.showVersion {
		fmt.Printf("%s - %s\n", buildVersion, buildDate)
		os.Exit(0)
	}

	logger := utils.InitLogger(cfg.debug)
	app := initializeApp(logger, cfg)

	if app.QueryFlags.GPFrom != "" {
		res, err := app.GlobalpingMeasurement()
		if err != nil {
			logger.Error("Error fetching globalping measurement", "error", err)
			os.Exit(2)
		}
		if app.QueryFlags.ShowJSON {
			err = app.OutputGlobalpingJSON(res)
		} else if app.QueryFlags.ShortOutput {
			err = app.OutputGlobalpingShort(res)
		} else {
			err = app.OutputGlobalping(res)
		}
		if err != nil {
			logger.Error("Error outputting globalping measurement", "error", err)
			os.Exit(2)
		}
		return
	}

	if cfg.reverseLookup {
		app.ReverseLookup()
	}

	app.LoadFallbacks()
	app.PrepareQuestions()

	if err := app.LoadNameservers(); err != nil {
		logger.Error("Error loading nameservers", "error", err)
		os.Exit(2)
	}

	resolvers, err := loadResolvers(app, cfg)
	if err != nil {
		logger.Error("Error loading resolvers", "error", err)
		os.Exit(2)
	}
	app.Resolvers = resolvers

	if len(app.QueryFlags.QNames) == 0 {
		cfg.flagSet.Usage()
		os.Exit(0)
	}

	responses, errors := performLookup(app, cfg)
	outputResults(app, responses, errors)
}

type config struct {
	flagSet       *flag.FlagSet
	showVersion   bool
	debug         bool
	reverseLookup bool
	timeout       time.Duration
	queryFlags    resolvers.QueryFlags
	outputJSON    bool
	showTime      bool
	useColor      bool
}

func loadConfig() (*config, error) {
	cfg := &config{}
	cfg.flagSet = setupFlags()

	if err := parseAndLoadFlags(cfg.flagSet); err != nil {
		return nil, fmt.Errorf("error parsing or loading flags: %w", err)
	}

	cfg.showVersion = k.Bool("version")
	cfg.debug = k.Bool("debug")
	cfg.reverseLookup = k.Bool("reverse")
	cfg.timeout = k.Duration("timeout")
	cfg.outputJSON = k.Bool("json")
	cfg.showTime = k.Bool("time")
	cfg.useColor = k.Bool("color")

	cfg.queryFlags = resolvers.QueryFlags{
		AA: k.Bool("aa"),
		AD: k.Bool("ad"),
		CD: k.Bool("cd"),
		RD: k.Bool("rd"),
		Z:  k.Bool("z"),
		DO: k.Bool("do"),
	}

	return cfg, nil
}

func setupFlags() *flag.FlagSet {
	f := flag.NewFlagSet("config", flag.ContinueOnError)
	f.Usage = renderCustomHelp

	f.StringSliceP("query", "q", []string{}, "Domain name to query")
	f.StringSliceP("type", "t", []string{}, "Type of DNS record to be queried (A, AAAA, MX etc)")
	f.StringSliceP("class", "c", []string{}, "Network class of the DNS record to be queried (IN, CH, HS etc)")
	f.StringSliceP("nameserver", "n", []string{}, "Address of the nameserver to send packets to")
	f.BoolP("reverse", "x", false, "Performs a DNS Lookup for an IPv4 or IPv6 address")

	f.String("gp-from", "", "Probe locations as a comma-separated list")
	f.Int("gp-limit", 1, "Limit the number of probes to use")

	f.DurationP("timeout", "T", 5*time.Second, "Sets the timeout for a query")
	f.Bool("search", true, "Use the search list provided in resolv.conf")
	f.Int("ndots", -1, "Specify the ndots parameter")
	f.BoolP("ipv4", "4", false, "Use IPv4 only")
	f.BoolP("ipv6", "6", false, "Use IPv6 only")
	f.String("strategy", "all", "Strategy to query nameservers in resolv.conf file")
	f.String("tls-hostname", "", "Hostname for certificate verification")
	f.Bool("skip-hostname-verification", false, "Skip TLS Hostname Verification")

	f.Bool("any", false, "Query all supported DNS record types")

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

func initializeApp(logger *slog.Logger, cfg *config) *app.App {
	gpConfig := globalping.Config{
		APIURL:    "https://api.globalping.io/v1",
		AuthURL:   "https://auth.globalping.io",
		UserAgent: fmt.Sprintf("doggo/%s (https://github.com/mr-karan/doggo)", buildVersion),
	}
	gpToken := os.Getenv("GLOBALPING_TOKEN")
	if gpToken != "" {
		gpConfig.AuthToken = &globalping.Token{
			AccessToken: gpToken,
			Expiry:      time.Now().Add(math.MaxInt64),
		}
	}
	globlpingClient := globalping.NewClient(gpConfig)

	app := app.New(logger, globlpingClient, buildVersion)

	if err := k.Unmarshal("", &app.QueryFlags); err != nil {
		logger.Error("Error loading args", "error", err)
		os.Exit(2)
	}

	loadNameservers(&app, cfg.flagSet.Args())
	return &app
}

func loadNameservers(app *app.App, args []string) {
	flagNameservers := k.Strings("nameserver")
	unparsedNameservers, qt, qc, qn := loadUnparsedArgs(args)

	if len(flagNameservers) > 0 {
		app.QueryFlags.Nameservers = flagNameservers
	} else {
		app.QueryFlags.Nameservers = unparsedNameservers
	}

	app.QueryFlags.QTypes = append(app.QueryFlags.QTypes, qt...)
	app.QueryFlags.QClasses = append(app.QueryFlags.QClasses, qc...)
	app.QueryFlags.QNames = append(app.QueryFlags.QNames, qn...)
}

func loadResolvers(app *app.App, cfg *config) ([]resolvers.Resolver, error) {
	return resolvers.LoadResolvers(resolvers.Options{
		Nameservers:        app.Nameservers,
		UseIPv4:            app.QueryFlags.UseIPv4,
		UseIPv6:            app.QueryFlags.UseIPv6,
		SearchList:         app.ResolverOpts.SearchList,
		Ndots:              app.ResolverOpts.Ndots,
		Timeout:            cfg.timeout,
		Logger:             app.Logger,
		Strategy:           app.QueryFlags.Strategy,
		InsecureSkipVerify: app.QueryFlags.InsecureSkipVerify,
		TLSHostname:        app.QueryFlags.TLSHostname,
	})
}

func performLookup(app *app.App, cfg *config) ([]resolvers.Response, []error) {
	ctx, cancel := context.WithTimeout(context.Background(), cfg.timeout)
	defer cancel()

	var (
		wg           sync.WaitGroup
		mu           sync.Mutex
		allResponses []resolvers.Response
		allErrors    []error
	)

	for _, resolver := range app.Resolvers {
		wg.Add(1)
		go func(r resolvers.Resolver) {
			defer wg.Done()
			responses, err := r.Lookup(ctx, app.Questions, cfg.queryFlags)
			mu.Lock()
			if err != nil {
				allErrors = append(allErrors, err)
			} else {
				allResponses = append(allResponses, responses...)
			}
			mu.Unlock()
		}(resolver)
	}

	wg.Wait()
	return allResponses, allErrors
}

func outputResults(app *app.App, responses []resolvers.Response, responseErrors []error) {
	if app.QueryFlags.ShowJSON {
		outputJSON(app.Logger, responses, responseErrors)
	} else {
		if len(responseErrors) > 0 {
			app.Logger.Error("Error looking up DNS records", "error", responseErrors[0])
			os.Exit(9)
		}
		app.Output(responses)
	}
}

func outputJSON(logger *slog.Logger, responses []resolvers.Response, responseErrors []error) {
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
		logger.Error("Error marshaling JSON")
		os.Exit(1)
	}
	fmt.Println(string(jsonData))
}
