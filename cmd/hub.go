package main

import (
	"time"

	"github.com/miekg/dns"
	"github.com/mr-karan/doggo/pkg/resolvers"
	"github.com/sirupsen/logrus"
)

// Hub represents the structure for all app wide functions and structs.
type Hub struct {
	Logger      *logrus.Logger
	Version     string
	QueryFlags  QueryFlags
	FreeArgs    []string
	Questions   []dns.Question
	Resolver    []resolvers.Resolver
	Nameservers []Nameserver
}

// QueryFlags is used store the value of CLI flags.
type QueryFlags struct {
	QNames           []string      `koanf:"query"`
	QTypes           []string      `koanf:"type"`
	QClasses         []string      `koanf:"class"`
	Nameservers      []string      `koanf:"nameserver"`
	IsDOH            bool          `koanf:"doh"`
	IsDOT            bool          `koanf:"dot"`
	IsUDP            bool          `koanf:"udp"`
	IsTCP            bool          `koanf:"tcp"`
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
// along with it's address.
type Nameserver struct {
	Address string
	Type    string
}

// NewHub initializes an instance of Hub which holds app wide configuration.
func NewHub(logger *logrus.Logger, buildVersion string) *Hub {
	// Initialise Resolver
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
