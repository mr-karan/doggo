package main

import (
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
	// Version and date of the build. This is injected at build-time.
	buildVersion = "unknown"
	buildDate    = "unknown"
	logger       = utils.InitLogger()
	k            = koanf.New(".")
)

func main() {
	// Initialize app.
	app := app.New(logger, buildVersion)

	// Configure Flags.
	f := flag.NewFlagSet("config", flag.ContinueOnError)

	// Custom Help Text.
	f.Usage = renderCustomHelp

	// Query Options.
	f.StringSliceP("query", "q", []string{}, "Domain name to query")
	f.StringSliceP("type", "t", []string{}, "Type of DNS record to be queried (A, AAAA, MX etc)")
	f.StringSliceP("class", "c", []string{}, "Network class of the DNS record to be queried (IN, CH, HS etc)")
	f.StringSliceP("nameservers", "n", []string{}, "Address of the nameserver to send packets to")
	f.BoolP("reverse", "x", false, "Performs a DNS Lookup for an IPv4 or IPv6 address. Sets the query type and class to PTR and IN respectively.")

	// Resolver Options
	f.Int("timeout", 5, "Sets the timeout for a query to T seconds. The default timeout is 5 seconds.")
	f.Bool("search", true, "Use the search list provided in resolv.conf. It sets the `ndots` parameter as well unless overridden by `ndots` flag.")
	f.Int("ndots", -1, "Specify the ndots parameter. Default value is taken from resolv.conf and fallbacks to 1 if ndots statement is missing in resolv.conf")
	f.BoolP("ipv4", "4", false, "Use IPv4 only")
	f.BoolP("ipv6", "6", false, "Use IPv6 only")
	f.String("strategy", "all", "Strategy to query nameservers in resolv.conf file (`all`, `random`, `first`)")

	// Output Options
	f.BoolP("json", "J", false, "Set the output format as JSON")
	f.Bool("short", false, "Short output format")
	f.Bool("time", false, "Display how long it took for the response to arrive")
	f.Bool("color", true, "Show colored output")
	f.Bool("debug", false, "Enable debug mode")

	f.Bool("version", false, "Show version of doggo")

	// Parse and Load Flags.
	err := f.Parse(os.Args[1:])
	if err != nil {
		app.Logger.WithError(err).Error("error parsing flags")
		app.Logger.Exit(2)
	}
	if err = k.Load(posflag.Provider(f, ".", k), nil); err != nil {
		app.Logger.WithError(err).Error("error loading flags")
		f.Usage()
		app.Logger.Exit(2)
	}

	// If version flag is set, output version and quit.
	if k.Bool("version") {
		fmt.Printf("%s - %s\n", buildVersion, buildDate)
		app.Logger.Exit(0)
	}

	// Set log level.
	if k.Bool("debug") {
		// Set logger level
		app.Logger.SetLevel(logrus.DebugLevel)
	} else {
		app.Logger.SetLevel(logrus.InfoLevel)
	}

	// Unmarshall flags to the app.
	err = k.Unmarshal("", &app.QueryFlags)
	if err != nil {
		app.Logger.WithError(err).Error("error loading args")
		app.Logger.Exit(2)
	}

	// Load all `non-flag` arguments
	// which will be parsed separately.
	nsvrs, qt, qc, qn := loadUnparsedArgs(f.Args())
	app.QueryFlags.Nameservers = append(app.QueryFlags.Nameservers, nsvrs...)
	app.QueryFlags.QTypes = append(app.QueryFlags.QTypes, qt...)
	app.QueryFlags.QClasses = append(app.QueryFlags.QClasses, qc...)
	app.QueryFlags.QNames = append(app.QueryFlags.QNames, qn...)

	// Check if reverse flag is passed. If it is, then set
	// query type as PTR and query class as IN.
	// Modify query name like 94.2.0.192.in-addr.arpa if it's an IPv4 address.
	// Use IP6.ARPA nibble format otherwise.

	if app.QueryFlags.ReverseLookup {
		app.ReverseLookup()
	}

	// Load fallbacks.
	app.LoadFallbacks()

	// Load Questions.
	app.PrepareQuestions()

	// Load Nameservers.
	err = app.LoadNameservers()
	if err != nil {
		app.Logger.WithError(err).Error("error loading nameservers")
		app.Logger.Exit(2)
	}

	// Load Resolvers.
	rslvrs, err := resolvers.LoadResolvers(resolvers.Options{
		Nameservers: app.Nameservers,
		UseIPv4:     app.QueryFlags.UseIPv4,
		UseIPv6:     app.QueryFlags.UseIPv6,
		SearchList:  app.ResolverOpts.SearchList,
		Ndots:       app.ResolverOpts.Ndots,
		Timeout:     app.QueryFlags.Timeout * time.Second,
		Logger:      app.Logger,
		Strategy:    app.QueryFlags.Strategy,
	})
	if err != nil {
		app.Logger.WithError(err).Error("error loading resolver")
		app.Logger.Exit(2)
	}
	app.Resolvers = rslvrs

	// Run the app.
	app.Logger.Debug("Starting doggo 🐶")
	if len(app.QueryFlags.QNames) == 0 {
		f.Usage()
		app.Logger.Exit(0)
	}

	// Resolve Queries.
	var responses []resolvers.Response
	for _, q := range app.Questions {
		for _, rslv := range app.Resolvers {
			resp, err := rslv.Lookup(q)
			if err != nil {
				app.Logger.WithError(err).Error("error looking up DNS records")
			}
			responses = append(responses, resp)
		}
	}
	app.Output(responses)

	// Quitting.
	app.Logger.Exit(0)
}
