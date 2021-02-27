package main

import (
	"net/http"
	"time"

	"github.com/mr-karan/doggo/internal/app"
	"github.com/mr-karan/doggo/pkg/utils"
	"github.com/sirupsen/logrus"

	"github.com/go-chi/chi"
	"github.com/knadh/koanf"
)

var (
	logger = utils.InitLogger()
	ko     = koanf.New(".")
	// Version and date of the build. This is injected at build-time.
	buildVersion = "unknown"
	buildDate    = "unknown"
)

func main() {
	initConfig()

	// Initialize app.
	app := app.New(logger, buildVersion)

	// Register handles.
	r := chi.NewRouter()
	r.Get("/", wrap(app, handleIndex))
	r.Get("/ping/", wrap(app, handleHealthCheck))
	r.Post("/lookup/", wrap(app, handleLookup))

	// HTTP Server.
	srv := &http.Server{
		Addr:         ko.String("server.address"),
		Handler:      r,
		ReadTimeout:  ko.Duration("server.read_timeout") * time.Millisecond,
		WriteTimeout: ko.Duration("server.write_timeout") * time.Millisecond,
		IdleTimeout:  ko.Duration("server.keepalive_timeout") * time.Millisecond,
	}

	logger.WithFields(logrus.Fields{
		"address": srv.Addr,
	}).Info("starting server")

	if err := srv.ListenAndServe(); err != nil {
		logger.Fatalf("couldn't start server: %v", err)
	}
}
