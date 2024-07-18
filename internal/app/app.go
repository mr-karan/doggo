package app

import (
	"log/slog"

	"github.com/jsdelivr/globalping-cli/globalping"
	"github.com/miekg/dns"
	"github.com/mr-karan/doggo/pkg/models"
	"github.com/mr-karan/doggo/pkg/resolvers"
)

// App represents the structure for all app wide configuration.
type App struct {
	Logger       *slog.Logger
	Version      string
	QueryFlags   models.QueryFlags
	Questions    []dns.Question
	Resolvers    []resolvers.Resolver
	ResolverOpts resolvers.Options
	Nameservers  []models.Nameserver

	globalping globalping.Client
}

// NewApp initializes an instance of App which holds app wide configuration.
func New(
	logger *slog.Logger,
	globalping globalping.Client,
	buildVersion string,
) App {
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
		globalping:  globalping,
	}
	return app
}
