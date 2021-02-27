package app

import (
	"github.com/miekg/dns"
	"github.com/mr-karan/doggo/pkg/models"
	"github.com/mr-karan/doggo/pkg/resolvers"
	"github.com/sirupsen/logrus"
)

// App represents the structure for all app wide configuration.
type App struct {
	Logger       *logrus.Logger
	Version      string
	QueryFlags   models.QueryFlags
	Questions    []dns.Question
	Resolvers    []resolvers.Resolver
	ResolverOpts resolvers.Options
	Nameservers  []models.Nameserver
}

// NewApp initializes an instance of App which holds app wide configuration.
func New(logger *logrus.Logger, buildVersion string) App {
	app := App{
		Logger:  logger,
		Version: buildVersion,
		QueryFlags: models.QueryFlags{
			QNames:      []string{},
			QTypes:      []string{},
			QClasses:    []string{},
			Nameservers: []string{},
		},
		Nameservers: []models.Nameserver{},
	}
	return app
}
