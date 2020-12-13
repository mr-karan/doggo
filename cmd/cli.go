package main

import (
	"fmt"
	"os"

	"github.com/knadh/koanf"
	"github.com/knadh/koanf/providers/posflag"
	flag "github.com/spf13/pflag"
)

var (
	// Version and date of the build. This is injected at build-time.
	buildVersion = "unknown"
	buildDate    = "unknown"
	k            = koanf.New(".")
)

func main() {
	var (
		logger = initLogger()
	)

	// Initialize hub.
	hub := NewHub(logger, buildVersion)

	// Configure Flags
	// Use the POSIX compliant pflag lib instead of Go's flag lib.
	f := flag.NewFlagSet("config", flag.ContinueOnError)
	f.Usage = func() {
		fmt.Println(f.FlagUsages())
		os.Exit(0)
	}
	// Path to one or more config files to load into koanf along with some config params.
	f.StringSliceP("query", "q", []string{}, "Domain name to query")
	f.StringSliceP("type", "t", []string{}, "Type of DNS record to be queried (A, AAAA, MX etc)")
	f.StringSliceP("class", "c", []string{}, "Network class of the DNS record to be queried (IN, CH, HS etc)")
	f.StringSliceP("nameservers", "n", []string{}, "Address of the nameserver to send packets to")

	// Protocol Options
	f.BoolP("udp", "U", false, "Use the DNS protocol over UDP")
	f.BoolP("tcp", "T", false, "Use the DNS protocol over TCP")
	f.BoolP("doh", "H", false, "Use the DNS-over-HTTPS protocol")
	f.BoolP("dot", "S", false, "Use the DNS-over-TLS")

	// Resolver Options
	f.Bool("search", false, "Use the search list provided in resolv.conf. It sets the `ndots` parameter as well unless overriden by `ndots` flag.")
	f.Int("ndots", 1, "Specify the ndots paramter")

	// Output Options
	f.BoolP("json", "J", false, "Set the output format as JSON")
	f.Bool("time", false, "Display how long it took for the response to arrive")
	f.Bool("debug", false, "Enable debug mode")

	// Parse and Load Flags
	f.Parse(os.Args[1:])
	if err := k.Load(posflag.Provider(f, ".", k), nil); err != nil {
		hub.Logger.Fatalf("error loading flags: %v", err)
		fmt.Println(f.FlagUsages())
		os.Exit(0)
	}

	// Run the app.
	hub.Logger.Debug("Starting doggo...")

	// Parse Query Args
	hub.loadQueryArgs()

	// Start App
	if len(hub.QueryFlags.QNames) == 0 {
		fmt.Println(f.FlagUsages())
		os.Exit(0)
	}
	hub.Lookup()

}
