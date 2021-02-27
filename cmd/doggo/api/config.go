package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/knadh/koanf/parsers/toml"
	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/providers/posflag"
	"github.com/sirupsen/logrus"
	flag "github.com/spf13/pflag"
)

// Config is the config given by the user
type Config struct {
	HTTPAddr string `koanf:"listen_addr"`
}

func initConfig() {
	f := flag.NewFlagSet("api", flag.ContinueOnError)
	f.Usage = func() {
		fmt.Println(f.FlagUsages())
		os.Exit(0)
	}

	// Register --config flag.
	f.StringSlice("config", []string{"config.toml"},
		"Path to one or more TOML config files to load in order")

	// Register --version flag.
	f.Bool("version", false, "Show build version")
	f.Parse(os.Args[1:])
	// Display version.
	if ok, _ := f.GetBool("version"); ok {
		fmt.Println(buildVersion, buildDate)
		os.Exit(0)
	}

	// Read the config files.
	cFiles, _ := f.GetStringSlice("config")
	for _, f := range cFiles {
		logger.WithFields(logrus.Fields{
			"file": f,
		}).Info("reading config")
		if err := ko.Load(file.Provider(f), toml.Parser()); err != nil {
			logger.Fatalf("error reading config: %v", err)
		}
	}
	// Load environment variables and merge into the loaded config.
	if err := ko.Load(env.Provider("DOGGO_API_", ".", func(s string) string {
		return strings.Replace(strings.ToLower(
			strings.TrimPrefix(s, "DOGGO_API_")), "__", ".", -1)
	}), nil); err != nil {
		logger.Fatalf("error loading env config: %v", err)
	}

	ko.Load(posflag.Provider(f, ".", ko), nil)
}
