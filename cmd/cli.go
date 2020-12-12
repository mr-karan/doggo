package main

import (
	"os"

	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
)

var (
	// Version and date of the build. This is injected at build-time.
	buildVersion   = "unknown"
	buildDate      = "unknown"
	verboseEnabled = false
)

// initLogger initializes logger
func initLogger(verbose bool) *logrus.Logger {
	logger := logrus.New()
	logger.SetFormatter(&logrus.TextFormatter{
		DisableTimestamp:       true,
		DisableLevelTruncation: true,
	})
	// Set logger level
	if verbose {
		logger.SetLevel(logrus.DebugLevel)
		logger.Debug("verbose logging enabled")
	} else {
		logger.SetLevel(logrus.InfoLevel)
	}
	return logger
}

func main() {
	var (
		logger = initLogger(verboseEnabled)
		app    = cli.NewApp()
	)
	// Initialize hub.
	hub := NewHub(logger, buildVersion)

	// Configure CLI app.
	app.Name = "doggo"
	app.Usage = "Command-line DNS Client"
	app.Version = buildVersion

	// Register command line flags.
	app.Flags = []cli.Flag{
		&cli.StringSliceFlag{
			Name:        "query",
			Usage:       "Domain name to query",
			Destination: hub.QueryFlags.QNames,
		},
		&cli.StringSliceFlag{
			Name:        "type",
			Usage:       "Type of DNS record to be queried (A, AAAA, MX etc)",
			Destination: hub.QueryFlags.QTypes,
		},
		&cli.StringSliceFlag{
			Name:        "nameserver",
			Usage:       "Address of the nameserver to send packets to",
			Destination: hub.QueryFlags.Nameservers,
		},
		&cli.StringSliceFlag{
			Name:        "class",
			Usage:       "Network class of the DNS record to be queried (IN, CH, HS etc)",
			Destination: hub.QueryFlags.QClasses,
		},
		&cli.BoolFlag{
			Name:    "udp",
			Usage:   "Use the DNS protocol over UDP",
			Aliases: []string{"U"},
		},
		&cli.BoolFlag{
			Name:        "tcp",
			Usage:       "Use the DNS protocol over TCP",
			Aliases:     []string{"T"},
			Destination: &hub.QueryFlags.UseTCP,
		},
		&cli.BoolFlag{
			Name:        "https",
			Usage:       "Use the DNS-over-HTTPS protocol",
			Aliases:     []string{"H"},
			Destination: &hub.QueryFlags.IsDOH,
		},
		&cli.BoolFlag{
			Name:        "tls",
			Usage:       "Use the DNS-over-TLS",
			Aliases:     []string{"S"},
			Destination: &hub.QueryFlags.IsDOT,
		},
		&cli.BoolFlag{
			Name:        "ipv6",
			Aliases:     []string{"6"},
			Usage:       "Use IPv6 only",
			Destination: &hub.QueryFlags.UseIPv6,
		},
		&cli.BoolFlag{
			Name:        "ipv4",
			Aliases:     []string{"4"},
			Usage:       "Use IPv4 only",
			Destination: &hub.QueryFlags.UseIPv4,
		},
		&cli.BoolFlag{
			Name:        "verbose",
			Usage:       "Enable verbose logging",
			Destination: &verboseEnabled,
			DefaultText: "false",
		},
	}

	app.Before = hub.loadQueryArgs
	app.Action = func(c *cli.Context) error {
		if len(hub.QueryFlags.QNames.Value()) == 0 {
			cli.ShowAppHelpAndExit(c, 0)
		}
		hub.Lookup(c)
		return nil
	}
	// Run the app.
	hub.Logger.Debug("Starting doggo...")
	err := app.Run(os.Args)
	if err != nil {
		logger.Errorf("oops! we encountered an issue: %s", err)
	}
}
