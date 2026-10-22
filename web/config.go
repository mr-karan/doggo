package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/knadh/koanf/parsers/toml"
	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/providers/posflag"
	flag "github.com/spf13/pflag"
)

func initConfig() error {
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
	if err := f.Parse(os.Args[1:]); err != nil {
		return fmt.Errorf("error parsing flags: %w", err)
	}
	// Display version.
	if ok, _ := f.GetBool("version"); ok {
		fmt.Println(buildVersion, buildDate)
		os.Exit(0)
	}

	// Read the config files.
	cFiles, _ := f.GetStringSlice("config")
	for _, f := range cFiles {
		if err := ko.Load(file.Provider(f), toml.Parser()); err != nil {
			return fmt.Errorf("error reading file: %w", err)
		}
	}
	// Load environment variables and merge into the loaded config.
	if err := ko.Load(env.Provider("DOGGO_API_", ".", func(s string) string {
		return strings.ReplaceAll(strings.ToLower(
			strings.TrimPrefix(s, "DOGGO_API_")), "__", ".")
	}), nil); err != nil {
		return fmt.Errorf("error loading env config: %w", err)
	}

	if err := ko.Load(posflag.Provider(f, ".", ko), nil); err != nil {
		return fmt.Errorf("error loading flags: %w", err)
	}
	return nil
}
