package main

import (
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
)

// Hub represents the structure for all app wide functions and structs.
type Hub struct {
	Logger      *logrus.Logger
	Version     string
	Domains     *cli.StringSlice
	QTypes      []string
	QClass      []string
	Nameservers []string
}

// NewHub initializes an instance of Hub which holds app wide configuration.
func NewHub(logger *logrus.Logger, buildVersion string) *Hub {
	hub := &Hub{
		Logger:  logger,
		Version: buildVersion,
	}
	return hub
}

// initApp acts like a middleware to load app managers with Hub before running any command.
// Use this middleware to perform any action before the command is run.
func (hub *Hub) initApp(fn cli.ActionFunc) cli.ActionFunc {
	return func(c *cli.Context) error {
		return fn(c)
	}
}
