package main

import (
	"embed"
	"io/fs"
	"net/http"
	"os"
	"time"

	"github.com/mr-karan/doggo/internal/app"
	"github.com/mr-karan/doggo/pkg/utils"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/knadh/koanf/v2"
)

var (
	ko = koanf.New(".")
	// Version and date of the build. This is injected at build-time.
	buildVersion = "unknown"
	buildDate    = "unknown"
	//go:embed assets/*
	assetsDir embed.FS
	//go:embed index.html
	html []byte
)

func main() {
	initConfig()

	logger := utils.InitLogger(ko.Bool("app.debug"))

	// Initialize app.
	app := app.New(logger, nil, buildVersion)

	// Register router instance.
	r := chi.NewRouter()

	// Register middlewares
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// Frontend Handlers.
	assets, _ := fs.Sub(assetsDir, "assets")
	r.Get("/assets/*", func(w http.ResponseWriter, r *http.Request) {
		fs := http.StripPrefix("/assets/", http.FileServer(http.FS(assets)))
		fs.ServeHTTP(w, r)
	})
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "text/html")
		w.Write(html)
	})

	// API Handlers.
	r.Get("/api/", wrap(app, handleIndexAPI))
	r.Get("/api/ping/", wrap(app, handleHealthCheck))
	r.Post("/api/lookup/", wrap(app, handleLookup))

	// HTTP Server.
	srv := &http.Server{
		Addr:         ko.String("server.address"),
		Handler:      r,
		ReadTimeout:  ko.Duration("server.read_timeout") * time.Millisecond,
		WriteTimeout: ko.Duration("server.write_timeout") * time.Millisecond,
		IdleTimeout:  ko.Duration("server.keepalive_timeout") * time.Millisecond,
	}

	logger.Info("starting server", "address", srv.Addr, "version", buildVersion)

	if err := srv.ListenAndServe(); err != nil {
		logger.Error("couldn't start server", "error", err)
		os.Exit(1)
	}
}
