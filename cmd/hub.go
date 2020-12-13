package main

import (
	"github.com/miekg/dns"
	"github.com/mr-karan/doggo/pkg/resolvers"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
)

// Hub represents the structure for all app wide functions and structs.
type Hub struct {
	Logger     *logrus.Logger
	Version    string
	QueryFlags QueryFlags
	Questions  []dns.Question
	Resolver   resolvers.Resolver
}

// QueryFlags is used store the value of CLI flags.
type QueryFlags struct {
	QNames           *cli.StringSlice
	QTypes           *cli.StringSlice
	QClasses         *cli.StringSlice
	Nameservers      *cli.StringSlice
	IsDOH            bool
	IsDOT            bool
	IsUDP            bool
	UseTCP           bool
	UseIPv4          bool
	UseIPv6          bool
	DisplayTimeTaken bool
	ShowJSON         bool
}

// NewHub initializes an instance of Hub which holds app wide configuration.
func NewHub(logger *logrus.Logger, buildVersion string) *Hub {
	// Initialise Resolver
	hub := &Hub{
		Logger:  logger,
		Version: buildVersion,
		QueryFlags: QueryFlags{
			QNames:      cli.NewStringSlice(),
			QTypes:      cli.NewStringSlice(),
			QClasses:    cli.NewStringSlice(),
			Nameservers: cli.NewStringSlice(),
		},
	}
	return hub
}

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
