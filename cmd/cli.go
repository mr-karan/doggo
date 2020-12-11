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

	var qFlags QueryFlags
	// Register command line flags.
	app.Flags = []cli.Flag{
		&cli.StringSliceFlag{
			Name:        "query",
			Usage:       "Domain name to query",
			Destination: qFlags.QNames,
		},
		&cli.StringSliceFlag{
			Name:        "type",
			Usage:       "Type of DNS record to be queried (A, AAAA, MX etc)",
			Destination: qFlags.QTypes,
		},
		&cli.StringSliceFlag{
			Name:        "nameserver",
			Usage:       "Address of the nameserver to send packets to",
			Destination: qFlags.Nameservers,
		},
		&cli.StringSliceFlag{
			Name:        "class",
			Usage:       "Network class of the DNS record to be queried (IN, CH, HS etc)",
			Destination: qFlags.QClasses,
		},
		&cli.BoolFlag{
			Name:        "https",
			Usage:       "Use the DNS-over-HTTPS protocol",
			Destination: &qFlags.IsDOH,
			DefaultText: "udp",
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
