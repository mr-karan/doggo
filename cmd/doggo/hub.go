package main

import (
	"time"

	"github.com/miekg/dns"
	"github.com/mr-karan/doggo/pkg/resolvers"
	"github.com/sirupsen/logrus"
)

// Hub represents the structure for all app wide configuration.
type Hub struct {
	Logger       *logrus.Logger
	Version      string
	QueryFlags   QueryFlags
	UnparsedArgs []string
	Questions    []dns.Question
	Resolver     []resolvers.Resolver
	ResolverOpts resolvers.Options
	Nameservers  []Nameserver
}

// QueryFlags is used store the query params
// supplied by the user.
type QueryFlags struct {
	QNames           []string      `koanf:"query"`
	QTypes           []string      `koanf:"type"`
	QClasses         []string      `koanf:"class"`
	Nameservers      []string      `koanf:"nameserver"`
	UseIPv4          bool          `koanf:"ipv4"`
	UseIPv6          bool          `koanf:"ipv6"`
	DisplayTimeTaken bool          `koanf:"time"`
	ShowJSON         bool          `koanf:"json"`
	UseSearchList    bool          `koanf:"search"`
	Ndots            int           `koanf:"ndots"`
	Color            bool          `koanf:"color"`
	Timeout          time.Duration `koanf:"timeout"`
}

// Nameserver represents the type of Nameserver
// along with the server address.
type Nameserver struct {
	Address string
	Type    string
}

// NewHub initializes an instance of Hub which holds app wide configuration.
func NewHub(logger *logrus.Logger, buildVersion string) *Hub {
	hub := &Hub{
		Logger:  logger,
		Version: buildVersion,
		QueryFlags: QueryFlags{
			QNames:      []string{},
			QTypes:      []string{},
			QClasses:    []string{},
			Nameservers: []string{},
		},
		Nameservers: []Nameserver{},
	}
	return hub
}

// initLogger initializes logger
func initLogger() *logrus.Logger {
	logger := logrus.New()
	logger.SetFormatter(&logrus.TextFormatter{
		FullTimestamp:          true,
		DisableLevelTruncation: true,
	})
	return logger
}
