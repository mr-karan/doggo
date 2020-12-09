package main

import (
	"os"
	"strings"

	"github.com/mr-karan/doggo/pkg/resolver"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"
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
	app.Author = "Karan Sharma @mrkaran"
	// Register command line args.
	app.Flags = []cli.Flag{
		cli.BoolFlag{
			Name:  "verbose",
			Usage: "Enable verbose logging",
		},
	}
	var (
		logger = initLogger(true)
	)
	// Initialize hub.
	hub := NewHub(logger, buildVersion)

	app.Action = func(c *cli.Context) error {
		// parse arguments
		for _, arg := range c.Args() {
			if strings.HasPrefix(arg, "@") {
				hub.Nameservers = append(hub.Nameservers, arg)
			} else if isUpper(arg) {
				if parseQueryType(arg) {
					hub.QTypes = append(hub.QTypes, arg)
				} else if parseQueryClass(arg) {
					hub.QClass = append(hub.QClass, arg)
				}
			} else {
				hub.Domains = append(hub.Domains, arg)
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
