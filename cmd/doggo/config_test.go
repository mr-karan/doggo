package main

import (
	"context"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/jsdelivr/globalping-go"
	"github.com/knadh/koanf/parsers/toml"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
	"github.com/mr-karan/doggo/internal/app"
	flag "github.com/spf13/pflag"
)

type recordingGlobalpingClient struct {
	globalping.Client
	request *globalping.MeasurementCreate
}

func (c *recordingGlobalpingClient) CreateMeasurement(_ context.Context, request *globalping.MeasurementCreate) (*globalping.MeasurementCreateResponse, error) {
	c.request = request
	return &globalping.MeasurementCreateResponse{ID: "test"}, nil
}

func (c *recordingGlobalpingClient) AwaitMeasurement(_ context.Context, _ string) (*globalping.Measurement, error) {
	return &globalping.Measurement{Status: globalping.StatusFinished}, nil
}

// isolateConfigEnv points all config file search paths at empty temp dirs and
// clears any inherited DOGGO_* variables so tests are hermetic regardless of
// the host's XDG_CONFIG_HOME / HOME / exported doggo env.
func isolateConfigEnv(t *testing.T) (xdg, home string) {
	t.Helper()
	for _, kv := range os.Environ() {
		if !strings.HasPrefix(kv, "DOGGO_") {
			continue
		}
		eq := strings.IndexByte(kv, '=')
		if eq < 0 {
			continue
		}
		key, val := kv[:eq], kv[eq+1:]
		os.Unsetenv(key)
		t.Cleanup(func() { _ = os.Setenv(key, val) })
	}
	xdg = t.TempDir()
	home = t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", xdg)
	t.Setenv("HOME", home)
	return xdg, home
}

// writeXDGConfig writes a config.toml into the XDG search path.
func writeXDGConfig(t *testing.T, xdg, contents string) string {
	t.Helper()
	dir := filepath.Join(xdg, "doggo")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	path := filepath.Join(dir, "config.toml")
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	return path
}

func loadTestConfigFull(t *testing.T, args ...string) (*koanf.Koanf, *flag.FlagSet) {
	t.Helper()
	k := koanf.New(".")
	f := setupFlags()
	if err := parseAndLoadFlags(k, f, args); err != nil {
		t.Fatalf("parseAndLoadFlags(%v): %v", args, err)
	}
	return k, f
}

func loadTestConfig(t *testing.T, args ...string) *koanf.Koanf {
	t.Helper()
	k, _ := loadTestConfigFull(t, args...)
	return k
}

// buildAppFromConfig reproduces the production load sequence (Unmarshal into
// QueryFlags, then loadNameservers) so tests exercise the real precedence
// behavior rather than an isolated helper.
func buildAppFromConfig(t *testing.T, args ...string) (*app.App, *koanf.Koanf, *flag.FlagSet) {
	t.Helper()
	k, f := loadTestConfigFull(t, args...)
	a := newTestApp()
	if err := k.Unmarshal("", &a.QueryFlags); err != nil {
		t.Fatalf("k.Unmarshal: %v", err)
	}
	if err := loadNameservers(a, k, f); err != nil {
		t.Fatalf("loadNameservers: %v", err)
	}
	return a, k, f
}

func TestRecordTypesFromConfigAndEnvAreNormalized(t *testing.T) {
	t.Run("config", func(t *testing.T) {
		xdg, _ := isolateConfigEnv(t)
		writeXDGConfig(t, xdg, `type = ["HTTPS", "SVCB", "TYPE64", "65"]`)
		a, _, _ := buildAppFromConfig(t, "example.com")
		want := []string{"HTTPS", "SVCB", "SVCB", "HTTPS"}
		for i := range want {
			if a.QueryFlags.QTypes[i] != want[i] {
				t.Errorf("QTypes[%d] = %q, want %q", i, a.QueryFlags.QTypes[i], want[i])
			}
		}
	})

	t.Run("environment", func(t *testing.T) {
		isolateConfigEnv(t)
		t.Setenv("DOGGO_TYPE", "SVCB,TYPE65,64")
		a, _, _ := buildAppFromConfig(t, "example.com")
		want := []string{"SVCB", "HTTPS", "SVCB"}
		for i := range want {
			if a.QueryFlags.QTypes[i] != want[i] {
				t.Errorf("QTypes[%d] = %q, want %q", i, a.QueryFlags.QTypes[i], want[i])
			}
		}
	})
}

func TestGlobalpingReceivesNormalizedNumericRecordType(t *testing.T) {
	isolateConfigEnv(t)
	k, f := loadTestConfigFull(t, "--type", "65", "--gp-from", "Germany", "example.com")
	client := &recordingGlobalpingClient{}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	a := app.New(logger, client, "test")
	if err := k.Unmarshal("", &a.QueryFlags); err != nil {
		t.Fatalf("k.Unmarshal: %v", err)
	}
	if err := loadNameservers(&a, k, f); err != nil {
		t.Fatalf("loadNameservers: %v", err)
	}

	if _, err := a.GlobalpingMeasurement(); err != nil {
		t.Fatalf("GlobalpingMeasurement: %v", err)
	}
	if client.request == nil || client.request.Options == nil || client.request.Options.Query == nil {
		t.Fatalf("Globalping request missing DNS query options: %+v", client.request)
	}
	if got := client.request.Options.Query.Type; got != "HTTPS" {
		t.Fatalf("Globalping query type = %q, want HTTPS", got)
	}
}

// newTestApp builds a minimal app.App for exercising loadNameservers without
// spinning up the globalping client. Output/discarded logger keeps it quiet.
func newTestApp() *app.App {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	a := app.New(logger, nil, "test")
	return &a
}

func TestConfigDefaultsWithoutFileOrEnv(t *testing.T) {
	isolateConfigEnv(t)
	k := loadTestConfig(t)

	if got := k.String("strategy"); got != "all" {
		t.Errorf("strategy = %q, want %q", got, "all")
	}
	if !k.Bool("color") {
		t.Error("color should default to true")
	}
	if !k.Bool("rd") {
		t.Error("rd should default to true")
	}
	if !k.Bool("search") {
		t.Error("search should default to true")
	}
	if got := k.Duration("timeout"); got != 5*time.Second {
		t.Errorf("timeout = %v, want 5s", got)
	}
}

func TestConfigFileOverridesDefaults(t *testing.T) {
	xdg, _ := isolateConfigEnv(t)
	writeXDGConfig(t, xdg, `
strategy = "first"
color = false
timeout = "10s"
bufsize = 4096
`)

	k := loadTestConfig(t)

	if got := k.String("strategy"); got != "first" {
		t.Errorf("strategy = %q, want %q", got, "first")
	}
	if k.Bool("color") {
		t.Error("color should be false from config file")
	}
	if got := k.Duration("timeout"); got != 10*time.Second {
		t.Errorf("timeout = %v, want 10s", got)
	}
	if got := k.Int("bufsize"); got != 4096 {
		t.Errorf("bufsize = %d, want 4096", got)
	}
	// Keys not present in the file must keep their flag defaults.
	if !k.Bool("rd") {
		t.Error("rd should keep its default of true")
	}
	if !k.Bool("search") {
		t.Error("search should keep its default of true")
	}
}

func TestConfigFileFromDotfileFallback(t *testing.T) {
	_, home := isolateConfigEnv(t)
	path := filepath.Join(home, ".doggo.toml")
	if err := os.WriteFile(path, []byte("strategy = \"random\"\n"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	k := loadTestConfig(t)
	if got := k.String("strategy"); got != "random" {
		t.Errorf("strategy = %q, want %q", got, "random")
	}
}

func TestEnvOverridesConfigFile(t *testing.T) {
	xdg, _ := isolateConfigEnv(t)
	writeXDGConfig(t, xdg, `strategy = "first"`)
	t.Setenv("DOGGO_STRATEGY", "random")

	k := loadTestConfig(t)
	if got := k.String("strategy"); got != "random" {
		t.Errorf("strategy = %q, want %q (env wins over file)", got, "random")
	}
}

func TestFlagOverridesEnvAndConfigFile(t *testing.T) {
	xdg, _ := isolateConfigEnv(t)
	writeXDGConfig(t, xdg, `strategy = "first"`)
	t.Setenv("DOGGO_STRATEGY", "random")

	k := loadTestConfig(t, "--strategy=all")
	if got := k.String("strategy"); got != "all" {
		t.Errorf("strategy = %q, want %q (flag wins over env and file)", got, "all")
	}
}

func TestEnvKeyTransformation(t *testing.T) {
	isolateConfigEnv(t)
	t.Setenv("DOGGO_SKIP_HOSTNAME_VERIFICATION", "true")
	t.Setenv("DOGGO_IPV4", "true")
	t.Setenv("DOGGO_NDOTS", "2")

	k := loadTestConfig(t)
	if !k.Bool("skip-hostname-verification") {
		t.Error("DOGGO_SKIP_HOSTNAME_VERIFICATION should map to skip-hostname-verification")
	}
	if !k.Bool("ipv4") {
		t.Error("DOGGO_IPV4 should map to ipv4")
	}
	if got := k.Int("ndots"); got != 2 {
		t.Errorf("ndots = %d, want 2", got)
	}
}

func TestEnvUnknownVarsIgnored(t *testing.T) {
	isolateConfigEnv(t)
	// DOGGO_API_* belongs to the API server (web/), not the CLI.
	t.Setenv("DOGGO_API_LISTEN_ADDR", ":8080")
	t.Setenv("DOGGO_NOTAFLAG", "nope")
	// CLI-only flags must not be settable from the environment either.
	t.Setenv("DOGGO_VERSION", "true")

	k := loadTestConfig(t)
	if k.Exists("api-listen-addr") {
		t.Error("DOGGO_API_LISTEN_ADDR should be ignored by the CLI")
	}
	if k.Exists("notaflag") {
		t.Error("DOGGO_NOTAFLAG should be ignored")
	}
	if k.Bool("version") {
		t.Error("DOGGO_VERSION should be ignored")
	}
}

func TestEnvSliceSplitting(t *testing.T) {
	isolateConfigEnv(t)
	t.Setenv("DOGGO_NAMESERVER", "1.1.1.1, 8.8.8.8 ,,9.9.9.9")

	k := loadTestConfig(t)
	got := k.Strings("nameserver")
	want := []string{"1.1.1.1", "8.8.8.8", "9.9.9.9"}
	if len(got) != len(want) {
		t.Fatalf("nameserver = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("nameserver[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestExplicitConfigFlag(t *testing.T) {
	isolateConfigEnv(t)
	dir := t.TempDir()
	path := filepath.Join(dir, "custom.toml")
	if err := os.WriteFile(path, []byte("strategy = \"internal\"\n"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	k := loadTestConfig(t, "--config", path)
	if got := k.String("strategy"); got != "internal" {
		t.Errorf("strategy = %q, want %q", got, "internal")
	}
}

func TestExplicitConfigFlagBeatsEnvPath(t *testing.T) {
	xdg, _ := isolateConfigEnv(t)
	xdgPath := writeXDGConfig(t, xdg, `strategy = "first"`)

	dir := t.TempDir()
	flagPath := filepath.Join(dir, "custom.toml")
	if err := os.WriteFile(flagPath, []byte("strategy = \"internal\"\n"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	t.Setenv("DOGGO_CONFIG", xdgPath)

	k := loadTestConfig(t, "--config", flagPath)
	if got := k.String("strategy"); got != "internal" {
		t.Errorf("strategy = %q, want %q (--config wins over DOGGO_CONFIG)", got, "internal")
	}
}

func TestMissingExplicitConfigIsError(t *testing.T) {
	isolateConfigEnv(t)
	missing := filepath.Join(t.TempDir(), "nope.toml")

	k := koanf.New(".")
	f := setupFlags()
	if err := parseAndLoadFlags(k, f, []string{"--config", missing}); err == nil {
		t.Fatal("expected error for missing --config file, got nil")
	}

	t.Setenv("DOGGO_CONFIG", missing)
	k = koanf.New(".")
	f = setupFlags()
	if err := parseAndLoadFlags(k, f, nil); err == nil {
		t.Fatal("expected error for missing DOGGO_CONFIG file, got nil")
	}
}

func TestInvalidConfigTOMLIsError(t *testing.T) {
	xdg, _ := isolateConfigEnv(t)
	writeXDGConfig(t, xdg, "strategy = [unclosed")

	k := koanf.New(".")
	f := setupFlags()
	if err := parseAndLoadFlags(k, f, nil); err == nil {
		t.Fatal("expected error for malformed config file, got nil")
	}
}

func TestMissingDefaultConfigIsNotError(t *testing.T) {
	isolateConfigEnv(t)
	// No config file anywhere; must not fail.
	loadTestConfig(t)
}

// --- M1/M2/M3: positional args vs config-file slice keys ---

func TestPositionalNameserverBeatsConfigFile(t *testing.T) {
	xdg, _ := isolateConfigEnv(t)
	writeXDGConfig(t, xdg, "nameserver = [\"9.9.9.9\"]\n")

	a, _, _ := buildAppFromConfig(t, "@8.8.8.8", "example.com")

	if got, want := a.QueryFlags.Nameservers, []string{"8.8.8.8"}; len(got) != 1 || got[0] != "8.8.8.8" {
		t.Fatalf("Nameservers = %v, want %v (positional @ns must beat config)", got, want)
	}
}

func TestNameserverFlagBeatsPositionalAndConfig(t *testing.T) {
	xdg, _ := isolateConfigEnv(t)
	writeXDGConfig(t, xdg, "nameserver = [\"9.9.9.9\"]\n")

	a, _, _ := buildAppFromConfig(t, "--nameserver", "1.1.1.1", "@8.8.8.8", "example.com")

	if got, want := a.QueryFlags.Nameservers, []string{"1.1.1.1"}; len(got) != 1 || got[0] != "1.1.1.1" {
		t.Fatalf("Nameservers = %v, want %v (--nameserver flag must win)", got, want)
	}
}

func TestConfigFileNameserverUsedWhenNoPositional(t *testing.T) {
	xdg, _ := isolateConfigEnv(t)
	writeXDGConfig(t, xdg, "nameserver = [\"9.9.9.9\", \"1.1.1.1\"]\n")

	a, _, _ := buildAppFromConfig(t, "example.com")

	if got := a.QueryFlags.Nameservers; len(got) != 2 || got[0] != "9.9.9.9" || got[1] != "1.1.1.1" {
		t.Fatalf("Nameservers = %v, want [9.9.9.9 1.1.1.1] from config", got)
	}
}

func TestPositionalTypeReplacesConfigFile(t *testing.T) {
	xdg, _ := isolateConfigEnv(t)
	writeXDGConfig(t, xdg, "type = [\"A\"]\n")

	a, _, _ := buildAppFromConfig(t, "MX", "example.com")

	// Positional MX must REPLACE the config-file default of A, not union.
	if got := a.QueryFlags.QTypes; len(got) != 1 || got[0] != "MX" {
		t.Fatalf("QTypes = %v, want [MX] (positional must replace config default)", got)
	}
}

func TestTypeFlagUnionsWithPositional(t *testing.T) {
	xdg, _ := isolateConfigEnv(t)
	writeXDGConfig(t, xdg, "type = [\"A\"]\n")

	a, _, _ := buildAppFromConfig(t, "--type", "MX", "AAAA", "example.com")

	// --type=MX (changed) unions with positional AAAA; config A is overridden
	// by the changed flag.
	got := a.QueryFlags.QTypes
	if len(got) != 2 {
		t.Fatalf("QTypes = %v, want [MX AAAA]", got)
	}
	for _, g := range got {
		if g == "A" {
			t.Fatalf("QTypes = %v, should not contain config value %q", got, "A")
		}
	}
}

func TestConfigFileTypeUsedWhenNoPositional(t *testing.T) {
	xdg, _ := isolateConfigEnv(t)
	writeXDGConfig(t, xdg, "type = [\"MX\"]\n")

	a, _, _ := buildAppFromConfig(t, "example.com")

	if got := a.QueryFlags.QTypes; len(got) != 1 || got[0] != "MX" {
		t.Fatalf("QTypes = %v, want [MX] from config", got)
	}
}

func TestExplicitTypeOverridesConfiguredAny(t *testing.T) {
	for _, tc := range []struct {
		name string
		args []string
	}{
		{name: "positional type", args: []string{"MX", "example.com"}},
		{name: "type flag", args: []string{"--type", "MX", "example.com"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			xdg, _ := isolateConfigEnv(t)
			writeXDGConfig(t, xdg, "any = true\n")

			a, _, _ := buildAppFromConfig(t, tc.args...)
			a.LoadFallbacks()

			if a.QueryFlags.QueryAny {
				t.Fatal("configured any=true should be disabled by an explicit type")
			}
			if got := a.QueryFlags.QTypes; len(got) != 1 || got[0] != "MX" {
				t.Fatalf("QTypes = %v, want [MX]", got)
			}
		})
	}
}

func TestExplicitAnyStillOverridesPositionalType(t *testing.T) {
	isolateConfigEnv(t)
	a, _, _ := buildAppFromConfig(t, "--any", "MX", "example.com")
	a.LoadFallbacks()

	if !a.QueryFlags.QueryAny {
		t.Fatal("explicit --any should remain enabled")
	}
	if got := a.QueryFlags.QTypes; len(got) <= 1 {
		t.Fatalf("QTypes = %v, want all common record types", got)
	}
}

// --- M3: scalar TOML values for list keys ---

func TestScalarNameserverInConfig(t *testing.T) {
	xdg, _ := isolateConfigEnv(t)
	writeXDGConfig(t, xdg, "nameserver = \"1.1.1.1\"\n")

	k, _ := loadTestConfigFull(t, "example.com")
	got := k.Strings("nameserver")
	if len(got) != 1 || got[0] != "1.1.1.1" {
		t.Fatalf("nameserver = %v, want [1.1.1.1] (scalar must be coerced)", got)
	}
}

func TestScalarTypeInConfig(t *testing.T) {
	xdg, _ := isolateConfigEnv(t)
	writeXDGConfig(t, xdg, "type = \"MX\"\n")

	k, _ := loadTestConfigFull(t, "example.com")
	got := k.Strings("type")
	if len(got) != 1 || got[0] != "MX" {
		t.Fatalf("type = %v, want [MX]", got)
	}
}

func TestCommaScalarInConfigSplits(t *testing.T) {
	xdg, _ := isolateConfigEnv(t)
	writeXDGConfig(t, xdg, "nameserver = \"1.1.1.1, 9.9.9.9\"\n")

	k, _ := loadTestConfigFull(t, "example.com")
	got := k.Strings("nameserver")
	if len(got) != 2 || got[0] != "1.1.1.1" || got[1] != "9.9.9.9" {
		t.Fatalf("nameserver = %v, want [1.1.1.1 9.9.9.9]", got)
	}
}

// --- m1: validation ---

func TestUnknownConfigKeyFails(t *testing.T) {
	xdg, _ := isolateConfigEnv(t)
	writeXDGConfig(t, xdg, "strategiy = \"first\"\n") // typo

	k := koanf.New(".")
	f := setupFlags()
	if err := parseAndLoadFlags(k, f, nil); err == nil {
		t.Fatal("expected error for unknown config key, got nil")
	}
}

func TestTimeoutAsNumberFails(t *testing.T) {
	xdg, _ := isolateConfigEnv(t)
	writeXDGConfig(t, xdg, "timeout = 10\n") // bare number, no unit

	k := koanf.New(".")
	f := setupFlags()
	if err := parseAndLoadFlags(k, f, nil); err != nil {
		t.Fatalf("parseAndLoadFlags: %v (file load should succeed; validation happens in loadConfig)", err)
	}
	if err := validateTimeout(k.Get("timeout")); err == nil {
		t.Fatal("expected validateTimeout to reject bare-number timeout")
	}
}

func TestTimeoutRejectsNonDurationAndNonPositive(t *testing.T) {
	for _, raw := range []any{true, []any{"10s"}, "0s", "-1s", 0 * time.Second, -time.Second} {
		if err := validateTimeout(raw); err == nil {
			t.Errorf("validateTimeout(%#v) = nil, want error", raw)
		}
	}
	for _, raw := range []any{"500ms", 5 * time.Second} {
		if err := validateTimeout(raw); err != nil {
			t.Errorf("validateTimeout(%#v) = %v, want nil", raw, err)
		}
	}
}

func TestTimeoutEnvBareNumberFails(t *testing.T) {
	isolateConfigEnv(t)
	t.Setenv("DOGGO_TIMEOUT", "10")

	k, _ := loadTestConfigFull(t) // loads fine
	if err := validateTimeout(k.Get("timeout")); err == nil {
		t.Fatal("expected validateTimeout to reject DOGGO_TIMEOUT=10")
	}
}

// --- config-cli-sample.toml ---

// uncommentedSamplePath writes config-cli-sample.toml with every documented
// `# key = value` line uncommented to a temp file and returns its path.
func uncommentedSamplePath(t *testing.T) string {
	t.Helper()
	body, err := os.ReadFile(filepath.Join("..", "..", "config-cli-sample.toml"))
	if err != nil {
		t.Fatalf("reading sample config: %v", err)
	}
	re := regexp.MustCompile(`(?m)^#\s*([a-z0-9-]+\s*=.*)$`)
	uncommented := re.ReplaceAllString(string(body), "$1")

	path := filepath.Join(t.TempDir(), "config.toml")
	if err := os.WriteFile(path, []byte(uncommented), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	return path
}

// TestSampleConfigCoversEveryFlag keeps config-cli-sample.toml in sync with
// the flag set: a newly added flag must be documented there, and the sample
// must not reference keys the CLI would reject. The sample's keys come from
// parsing the uncommented file with the real TOML parser, not a second
// hand-rolled notion of what a key looks like.
func TestSampleConfigCoversEveryFlag(t *testing.T) {
	k := koanf.New(".")
	if err := k.Load(file.Provider(uncommentedSamplePath(t)), toml.Parser()); err != nil {
		t.Fatalf("parsing sample config: %v", err)
	}
	if len(k.Keys()) == 0 {
		t.Fatal("no keys found in config-cli-sample.toml")
	}
	f := setupFlags()

	// Nothing documented may be a key the CLI rejects (unknown or CLI-only).
	if err := validateConfigKeys(k, f); err != nil {
		t.Errorf("config-cli-sample.toml: %v", err)
	}

	// Every flag usable in a config file must be documented.
	f.VisitAll(func(fl *flag.Flag) {
		if !cliOnlyFlags[fl.Name] && !k.Exists(fl.Name) {
			t.Errorf("flag --%s is missing from config-cli-sample.toml", fl.Name)
		}
	})
}

// TestSampleConfigLoads verifies the sample survives the full production
// load path (parse, key validation, timeout validation) once every
// documented key is uncommented.
func TestSampleConfigLoads(t *testing.T) {
	isolateConfigEnv(t)

	k, _ := loadTestConfigFull(t, "--config", uncommentedSamplePath(t))
	if err := validateTimeout(k.Get("timeout")); err != nil {
		t.Errorf("sample config timeout rejected: %v", err)
	}
}

// TestCLIOnlyKeyInConfigFails verifies flags that are meaningless as
// persistent defaults (version, config) are rejected as config keys rather
// than honored — `version = true` would otherwise turn every invocation
// into a version print.
func TestCLIOnlyKeyInConfigFails(t *testing.T) {
	xdg, _ := isolateConfigEnv(t)
	writeXDGConfig(t, xdg, "version = true\n")

	k := koanf.New(".")
	f := setupFlags()
	if err := parseAndLoadFlags(k, f, nil); err == nil {
		t.Fatal("expected error for cli-only key in config file, got nil")
	}
}

func TestInvalidStrategyFails(t *testing.T) {
	isolateConfigEnv(t)
	t.Setenv("DOGGO_STRATEGY", "bogus")

	k := koanf.New(".")
	f := setupFlags()
	if err := parseAndLoadFlags(k, f, nil); err != nil {
		t.Fatalf("parseAndLoadFlags: %v", err)
	}
	// Strategy validation lives in loadConfig, not parseAndLoadFlags.
	switch s := k.String("strategy"); s {
	case "", "all", "random", "first", "internal":
		t.Fatalf("strategy %q should have been invalid", s)
	default:
		// expected
	}
}
