package main

import (
	"os"

	"github.com/knadh/koanf"
	"github.com/knadh/koanf/providers/posflag"
	"github.com/sirupsen/logrus"
	flag "github.com/spf13/pflag"
)

var (
	// Version and date of the build. This is injected at build-time.
	buildVersion = "unknown"
	buildDate    = "unknown"
)

func main() {
	var (
		logger = initLogger()
		k      = koanf.New(".")
	)

	// Initialize hub.
	hub := NewHub(logger, buildVersion)

	// Configure Flags.
	f := flag.NewFlagSet("config", flag.ContinueOnError)
	hub.flag = f

	// Custom Help Text.
	f.Usage = renderCustomHelp

	// Query Options.
	f.StringSliceP("query", "q", []string{}, "Domain name to query")
	f.StringSliceP("type", "t", []string{}, "Type of DNS record to be queried (A, AAAA, MX etc)")
	f.StringSliceP("class", "c", []string{}, "Network class of the DNS record to be queried (IN, CH, HS etc)")
	f.StringSliceP("nameserver", "n", []string{}, "Address of the nameserver to send packets to")

	// Resolver Options
	f.Int("timeout", 5, "Sets the timeout for a query to T seconds. The default timeout is 5 seconds.")
	f.Bool("search", true, "Use the search list provided in resolv.conf. It sets the `ndots` parameter as well unless overriden by `ndots` flag.")
	f.Int("ndots", 1, "Specify the ndots paramter. Default value is taken from resolv.conf and fallbacks to 1 if ndots statement is missing in resolv.conf")
	f.BoolP("ipv4", "4", false, "Use IPv4 only")
	f.BoolP("ipv6", "6", false, "Use IPv6 only")

	// Output Options
	f.BoolP("json", "J", false, "Set the output format as JSON")
	f.Bool("time", false, "Display how long it took for the response to arrive")
	f.Bool("color", true, "Show colored output")
	f.Bool("debug", false, "Enable debug mode")

	// Parse and Load Flags
	err := f.Parse(os.Args[1:])
	if err != nil {
		hub.Logger.WithError(err).Error("error parsing flags")
		hub.Logger.Exit(2)
	}
	if err = k.Load(posflag.Provider(f, ".", k), nil); err != nil {
		hub.Logger.WithError(err).Error("error loading flags")
		f.Usage()
		hub.Logger.Exit(2)
	}

	// Set log level.
	if k.Bool("debug") {
		// Set logger level
		hub.Logger.SetLevel(logrus.DebugLevel)
	} else {
		hub.Logger.SetLevel(logrus.InfoLevel)
	}

	// Unmarshall flags to the hub.
	err = k.Unmarshal("", &hub.QueryFlags)
	if err != nil {
		hub.Logger.WithError(err).Error("error loading args")
		hub.Logger.Exit(2)
	}

	// Load all `non-flag` arguments
	// which will be parsed separately.
	hub.UnparsedArgs = f.Args()

	// Parse Query Args
	err = hub.loadQueryArgs()
	if err != nil {
		hub.Logger.WithError(err).Error("error parsing flags/arguments")
		hub.Logger.Exit(2)
	}

	// Load Nameservers
	err = hub.loadNameservers()
	if err != nil {
		hub.Logger.WithError(err).Error("error loading nameservers")
		hub.Logger.Exit(2)
	}

	// Load Resolvers
	err = hub.loadResolvers()
	if err != nil {
		hub.Logger.WithError(err).Error("error loading resolver")
		hub.Logger.Exit(2)
	}

	// Start App
	// Run the app.
	hub.Logger.Debug("Starting doggo 🐶")

	if len(hub.QueryFlags.QNames) == 0 {
		f.Usage()
		hub.Logger.Exit(0)
	}

	// Resolve Queries.
	err = hub.Lookup()
	if err != nil {
		hub.Logger.WithError(err).Error("error looking up DNS records")
		hub.Logger.Exit(2)
	}
}
