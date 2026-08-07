package main

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/knadh/koanf/parsers/toml"
	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/providers/posflag"
	"github.com/knadh/koanf/v2"
	flag "github.com/spf13/pflag"
)

// envPrefix is the prefix for environment variables that map to CLI flags,
// e.g. DOGGO_STRATEGY=first is equivalent to --strategy=first.
const envPrefix = "DOGGO_"

// sliceFlags are flags whose values are lists. Environment variables for
// these are split on commas (e.g. DOGGO_NAMESERVER="1.1.1.1, 9.9.9.9").
var sliceFlags = map[string]bool{
	"query":      true,
	"type":       true,
	"class":      true,
	"nameserver": true,
}

// cliOnlyFlags are registered flags that make no sense as persistent
// defaults: version is a one-shot action (a config file containing it would
// make every invocation print the version and exit) and config names the
// file itself. They are rejected as config keys and ignored as DOGGO_* env
// vars (DOGGO_CONFIG is read directly by resolveConfigPath).
var cliOnlyFlags = map[string]bool{
	"version": true,
	"config":  true,
}

// parseAndLoadFlags parses CLI args into the flagset and merges configuration
// into k. Precedence, lowest to highest:
//
//	flag defaults < config file < DOGGO_* environment variables < CLI flags
//
// posflag.Provider with a non-nil koanf instance merges flag defaults only for
// keys not already present, while always merging explicitly set flags, which
// is what yields the precedence above.
func parseAndLoadFlags(k *koanf.Koanf, f *flag.FlagSet, args []string) error {
	if err := f.Parse(args); err != nil {
		return fmt.Errorf("error parsing flags: %w", err)
	}

	if err := loadConfigFile(k, f); err != nil {
		return err
	}

	if err := k.Load(envProvider(f), nil); err != nil {
		return fmt.Errorf("error loading environment config: %w", err)
	}

	if err := k.Load(posflag.Provider(f, ".", k), nil); err != nil {
		return fmt.Errorf("error loading flags: %w", err)
	}
	return nil
}

// loadConfigFile loads a TOML config file into k. The path is resolved from
// --config, then DOGGO_CONFIG, then the default search paths. An explicitly
// requested file that is missing or invalid is an error; a missing file at a
// default path is silently skipped.
func loadConfigFile(k *koanf.Koanf, f *flag.FlagSet) error {
	path, explicit := resolveConfigPath(f)
	if path == "" {
		return nil
	}
	if err := k.Load(file.Provider(path), toml.Parser()); err != nil {
		// Tolerate a default-path file vanishing between discovery and load.
		if !explicit && errors.Is(err, fs.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("error loading config file %q: %w", path, err)
	}
	normalizeSliceScalars(k)
	if err := validateConfigKeys(k, f); err != nil {
		return fmt.Errorf("error loading config file %q: %w", path, err)
	}
	return nil
}

// resolveConfigPath returns the config file to load and whether it was
// explicitly requested by the user (via --config or DOGGO_CONFIG).
func resolveConfigPath(f *flag.FlagSet) (string, bool) {
	if f.Changed("config") {
		path, _ := f.GetString("config")
		return path, true
	}
	if path := os.Getenv("DOGGO_CONFIG"); path != "" {
		return path, true
	}
	for _, path := range defaultConfigPaths() {
		if info, err := os.Stat(path); err == nil && !info.IsDir() {
			return path, false
		}
	}
	return "", false
}

// defaultConfigPaths lists config file locations in search order:
// $XDG_CONFIG_HOME/doggo/config.toml, the OS user config dir
// (~/.config/doggo/config.toml on Linux, ~/Library/Application Support on
// macOS, %AppData% on Windows), and ~/.doggo.toml.
func defaultConfigPaths() []string {
	var paths []string
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		paths = append(paths, filepath.Join(xdg, "doggo", "config.toml"))
	}
	if dir, err := os.UserConfigDir(); err == nil {
		paths = append(paths, filepath.Join(dir, "doggo", "config.toml"))
	}
	if home, err := os.UserHomeDir(); err == nil {
		paths = append(paths, filepath.Join(home, ".doggo.toml"))
	}
	return paths
}

// envProvider maps DOGGO_* environment variables onto flag names:
// DOGGO_SKIP_HOSTNAME_VERIFICATION becomes skip-hostname-verification.
// Variables that don't correspond to a registered flag (e.g. DOGGO_API_*
// used by the API server) are ignored. List flags accept comma-separated
// values.
func envProvider(f *flag.FlagSet) *env.Env {
	return env.ProviderWithValue(envPrefix, ".", func(key, value string) (string, interface{}) {
		name := strings.ReplaceAll(strings.ToLower(strings.TrimPrefix(key, envPrefix)), "_", "-")
		if cliOnlyFlags[name] || f.Lookup(name) == nil {
			return "", nil
		}
		if sliceFlags[name] {
			return name, splitCSV(value)
		}
		return name, value
	})
}

// splitCSV splits a comma-separated string into trimmed, non-empty fields.
func splitCSV(s string) []string {
	var parts []string
	for _, p := range strings.Split(s, ",") {
		if p = strings.TrimSpace(p); p != "" {
			parts = append(parts, p)
		}
	}
	return parts
}

// normalizeSliceScalars coerces scalar TOML values for list flags into slices.
// Without this, `nameserver = "1.1.1.1"` (scalar) is silently ignored because
// koanf.Strings only handles []any/[]string, and the flag default is an empty
// slice. Env vars are already split by envProvider and flags are already
// slices, so this only affects config-file values.
func normalizeSliceScalars(k *koanf.Koanf) {
	for key := range sliceFlags {
		switch v := k.Get(key).(type) {
		case nil, []string, []any:
			// already absent or a slice; leave as-is
		case string:
			k.Set(key, splitCSV(v))
		default:
			// numbers, bools, etc. — wrap the scalar as a single-element list
			k.Set(key, []string{fmt.Sprintf("%v", v)})
		}
	}
}

// validateConfigKeys rejects config keys that don't map to a registered flag,
// catching typos like `strategiy = "first"` early instead of silently ignoring
// them, as well as cliOnlyFlags, which are valid flags but meaningless as
// persistent defaults.
func validateConfigKeys(k *koanf.Koanf, f *flag.FlagSet) error {
	for _, key := range k.Keys() {
		if f.Lookup(key) == nil {
			return fmt.Errorf("unknown config key %q", key)
		}
		if cliOnlyFlags[key] {
			return fmt.Errorf("config key %q is only usable as a command-line flag", key)
		}
	}
	return nil
}
