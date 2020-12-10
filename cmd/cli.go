package main

import (
	"os"
	"strings"

	resolver "github.com/mr-karan/doggo/pkg/resolve"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
)

var (
	// Version and date of the build. This is injected at build-time.
	buildVersion = "unknown"
	buildDate    = "unknown"
)

// initLogger initializes logger
func initLogger(verbose bool) *logrus.Logger {
	logger := logrus.New()
	logger.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
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
	// Intialize new CLI app
	app := cli.NewApp()
	app.Name = "doggo"
	app.Usage = "Command-line DNS Client"
	app.Version = buildVersion
	var (
		logger = initLogger(true)
	)
	// Initialize hub.
	hub := NewHub(logger, buildVersion)
	// Register command line args.
	app.Flags = []cli.Flag{
		&cli.StringSliceFlag{
			Name:        "query",
			Usage:       "Domain name to query",
			Destination: hub.Domains,
		},
		&cli.BoolFlag{
			Name:  "verbose",
			Usage: "Enable verbose logging",
		},
	}
	app.Action = func(c *cli.Context) error {

		// parse arguments
		var domains cli.StringSlice
		for _, arg := range c.Args().Slice() {
			if strings.HasPrefix(arg, "@") {
				hub.Nameservers = append(hub.Nameservers, arg)
			} else if isUpper(arg) {
				if parseQueryType(arg) {
					hub.QTypes = append(hub.QTypes, arg)
				} else if parseQueryClass(arg) {
					hub.QClass = append(hub.QClass, arg)
				}
			} else {
				domains.Set(arg)
				hub.Domains = &domains
			}
		}
		// load defaults
		if len(hub.QTypes) == 0 {
			hub.QTypes = append(hub.QTypes, "A")
		}
		if len(hub.Nameservers) == 0 {
			ns, err := resolver.GetDefaultNameserver()
			if err != nil {
				panic(err)
			}
			hub.Nameservers = append(hub.Nameservers, ns)
		}
		// resolve query
		hub.Resolve()
		return nil
	}

	// Run the app.
	hub.Logger.Info("Starting doggo...")
	err := app.Run(os.Args)
	if err != nil {
		logger.Errorf("Something terrbily went wrong: %s", err)
	}
}
