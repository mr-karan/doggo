package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/jsdelivr/globalping-go"
	"github.com/knadh/koanf/v2"
	"github.com/mr-karan/doggo/internal/app"
	"github.com/mr-karan/doggo/pkg/models"
	"github.com/mr-karan/doggo/pkg/resolvers"
	"github.com/mr-karan/doggo/pkg/utils"
	flag "github.com/spf13/pflag"
)

// Exit codes used by the CLI. Exit 0 is implicit success; partial success
// (some resolvers answered, others failed) is 2; full lookup failure remains
// 9 to preserve compatibility with the pre-existing convention.
const (
	exitGenericFailure = 1
	exitPartialFailure = 2
	exitLookupFailure  = 9
)

var (
	buildVersion = "unknown"
	buildDate    = "unknown"
	k            = koanf.New(".")
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "completions" {
		completionsCommand()
		return
	}

	cfg, err := loadConfig()
	if err != nil {
		fmt.Printf("Error loading configuration: %v\n", err)
		os.Exit(1)
	}

	if cfg.showVersion {
		fmt.Printf("%s - %s\n", buildVersion, buildDate)
		os.Exit(0)
	}

	logger := utils.InitLogger(cfg.debug)
	app, err := initializeApp(logger, cfg)
	if err != nil {
		logger.Error("Error loading query arguments", "error", err)
		os.Exit(exitGenericFailure)
	}

	if len(app.QueryFlags.QNames) == 0 && !cfg.trace {
		cfg.flagSet.Usage()
		return
	}

	if cfg.trace {
		// Trace mode dispatches before Globalping and the normal lookup path;
		// --trace with --gp-from is a validation error (exit 1), so the
		// Globalping branch below never sees it.
		if cfg.reverseLookup {
			app.ReverseLookup()
		}
		os.Exit(runTrace(app, cfg, logger))
	}

	if app.QueryFlags.GPFrom != "" {
		// A local source bind cannot affect a probe executed by Globalping.
		if app.QueryFlags.SourceAddr != "" {
			logger.Error("--source cannot be combined with --gp-from: a local bind cannot affect a remote Globalping probe")
			os.Exit(exitGenericFailure)
		}
		res, err := app.GlobalpingMeasurement()
		if err != nil {
			logger.Error("Error fetching globalping measurement", "error", err)
			os.Exit(2)
		}
		if app.QueryFlags.ShowJSON {
			err = app.OutputGlobalpingJSON(res)
		} else if app.QueryFlags.ShortOutput {
			err = app.OutputGlobalpingShort(res)
		} else {
			err = app.OutputGlobalping(res)
		}
		if err != nil {
			logger.Error("Error outputting globalping measurement", "error", err)
			os.Exit(2)
		}
		return
	}

	if cfg.reverseLookup {
		app.ReverseLookup()
	}

	app.LoadFallbacks()
	if err := app.PrepareQuestions(); err != nil {
		logger.Error("Error preparing DNS questions", "error", err)
		os.Exit(exitGenericFailure)
	}

	if err := app.LoadNameservers(); err != nil {
		logger.Error("Error loading nameservers", "error", err)
		os.Exit(2)
	}

	loadedResolvers, err := loadResolvers(app, cfg)
	if err != nil {
		logger.Error("Error loading resolvers", "error", err)
		os.Exit(2)
	}
	app.Resolvers = loadedResolvers

	responses, lookupErrors := performLookup(app, cfg)
	if err := resolvers.CloseResolvers(app.Resolvers); err != nil {
		logger.Debug("Error closing resolvers", "error", err)
	}
	outputResults(app, responses, lookupErrors)
}

// runTrace executes --trace mode and returns the process exit code. It
// validates the invocation, traces the single effective question iteratively
// from the root, renders whatever was collected (including partial hops), and
// only then maps an operational failure onto an exit code: 0 for a completed
// trace (including authoritative NXDOMAIN/NODATA), 1 for an invalid
// invocation, 2 when at least one hop was collected before the error, and 9
// when no usable hop was produced.
func runTrace(a *app.App, cfg *config, logger *slog.Logger) int {
	applyTraceDefaults(&a.QueryFlags)
	if err := validateTraceQuery(a.QueryFlags); err != nil {
		logger.Error("Invalid --trace invocation", "error", err)
		return exitGenericFailure
	}

	if err := a.PrepareQuestions(); err != nil {
		logger.Error("Error preparing DNS questions", "error", err)
		return exitGenericFailure
	}

	opts := resolvers.TraceOptions{
		Flags:      cfg.queryFlags,
		UseIPv4:    a.QueryFlags.UseIPv4,
		UseIPv6:    a.QueryFlags.UseIPv6,
		SourceAddr: a.QueryFlags.SourceAddr,
		Timeout:    cfg.timeout,
	}

	// An explicit nameserver (flag, positional @server, or config/env) only
	// primes the root zone. Without one, the tracer starts from the compiled
	// IANA root hints and system resolvers are never loaded.
	if len(a.QueryFlags.Nameservers) > 0 {
		if err := a.LoadNameservers(); err != nil {
			logger.Error("Error loading nameservers", "error", err)
			return exitGenericFailure
		}
		bootstrap, err := loadResolvers(a, cfg)
		if err != nil {
			logger.Error("Error loading resolvers", "error", err)
			return exitGenericFailure
		}
		defer func() {
			if err := resolvers.CloseResolvers(bootstrap); err != nil {
				logger.Debug("Error closing bootstrap resolvers", "error", err)
			}
		}()
		if len(bootstrap) > 0 {
			opts.Bootstrap = bootstrap[0]
		}
	}

	result, err := resolvers.Trace(context.Background(), a.Questions[0], opts)

	// Render the trace -- partial results included -- before mapping an
	// operational failure onto an exit code.
	if outErr := a.OutputTrace(result); outErr != nil {
		logger.Error("Error rendering trace output", "error", outErr)
		return exitPartialFailure
	}
	if err != nil {
		logger.Error("Trace failed", "error", err)
		if errors.Is(err, resolvers.ErrInvalidTraceConfig) {
			return exitGenericFailure
		}
		if len(result.Hops) > 0 {
			return exitPartialFailure
		}
		return exitLookupFailure
	}
	return 0
}

// applyTraceDefaults narrows the free-form defaults for trace mode: a trace
// asks exactly one question, so an unspecified type/class becomes A/IN rather
// than the regular lookup's A+AAAA pair.
func applyTraceDefaults(qf *models.QueryFlags) {
	if len(qf.QTypes) == 0 {
		qf.QTypes = []string{"A"}
	}
	if len(qf.QClasses) == 0 {
		qf.QClasses = []string{"IN"}
	}
}

// validateTraceQuery enforces the --trace invocation contract: the mode is
// incompatible with fan-out features and accepts exactly one IN-class
// question. Errors here are usage mistakes (exit 1), never DNS failures.
func validateTraceQuery(qf models.QueryFlags) error {
	if qf.QueryAny {
		return errors.New("--trace cannot be combined with --any: a trace asks exactly one question")
	}
	if qf.UseAuthoritative {
		return errors.New("--trace cannot be combined with --authoritative: tracing already discovers the authoritative nameservers")
	}
	if qf.GPFrom != "" {
		return errors.New("--trace cannot be combined with --gp-from: tracing requires direct iterative queries and cannot run through Globalping")
	}
	if qf.UseIPv4 && qf.UseIPv6 {
		return errors.New("--trace cannot be combined with both --ipv4 and --ipv6: pick one address family")
	}
	if len(qf.QNames) != 1 {
		return fmt.Errorf("--trace requires exactly one query name, got %d", len(qf.QNames))
	}
	if len(qf.QTypes) != 1 {
		return fmt.Errorf("--trace requires exactly one query type, got %d", len(qf.QTypes))
	}
	if strings.EqualFold(qf.QTypes[0], "ANY") {
		return errors.New("--trace does not support query type ANY")
	}
	if len(qf.QClasses) != 1 {
		return fmt.Errorf("--trace requires exactly one query class, got %d", len(qf.QClasses))
	}
	if !strings.EqualFold(qf.QClasses[0], "IN") {
		return fmt.Errorf("--trace supports only class IN, got %q", qf.QClasses[0])
	}
	return nil
}

type config struct {
	flagSet       *flag.FlagSet
	showVersion   bool
	debug         bool
	reverseLookup bool
	trace         bool
	timeout       time.Duration
	queryFlags    resolvers.QueryFlags
	outputJSON    bool
	showTime      bool
	useColor      bool
}

func loadConfig() (*config, error) {
	cfg := &config{}
	cfg.flagSet = setupFlags()

	if err := parseAndLoadFlags(k, cfg.flagSet, os.Args[1:]); err != nil {
		return nil, fmt.Errorf("error parsing or loading flags: %w", err)
	}

	cfg.showVersion = k.Bool("version")
	cfg.debug = k.Bool("debug")
	cfg.reverseLookup = k.Bool("reverse")
	cfg.trace = k.Bool("trace")
	cfg.timeout = k.Duration("timeout")
	cfg.outputJSON = k.Bool("json")
	cfg.showTime = k.Bool("time")
	cfg.useColor = k.Bool("color")

	if err := validateTimeout(k.Get("timeout")); err != nil {
		return nil, err
	}
	switch s := k.String("strategy"); s {
	case "", "all", "random", "first", "internal":
	default:
		return nil, fmt.Errorf("invalid strategy %q: must be one of all, random, first, internal", s)
	}

	bufsize := k.Int("bufsize")
	if bufsize < 0 || bufsize > 65535 {
		return nil, fmt.Errorf("--bufsize must be between 0 and 65535, got %d", bufsize)
	}
	if bufsize > 0 && bufsize < 512 {
		return nil, fmt.Errorf("--bufsize must be 0 or at least 512 (RFC 6891), got %d", bufsize)
	}

	cfg.queryFlags = resolvers.QueryFlags{
		AA: k.Bool("aa"),
		AD: k.Bool("ad"),
		CD: k.Bool("cd"),
		RD: k.Bool("rd"),
		Z:  k.Bool("z"),
		DO: k.Bool("do"),

		// EDNS0 options
		NSID:    k.Bool("nsid"),
		Cookie:  k.Bool("cookie"),
		Padding: k.Bool("padding"),
		EDE:     k.Bool("ede"),
		ECS:     k.String("ecs"),
		Bufsize: uint16(bufsize),
	}

	return cfg, nil
}

// validateTimeout guards against the silent footgun where a bad timeout in a
// config file or env var (a bare number like `timeout = 10`, a bool, or a
// non-positive duration) is coerced to zero-or-nanoseconds by koanf's
// cast-based Duration getter, making every query time out instantly. Only a
// flag-sourced time.Duration or a string that time.ParseDuration accepts is
// valid, and the resulting duration must be positive.
func validateTimeout(raw any) error {
	var d time.Duration
	switch v := raw.(type) {
	case time.Duration:
		d = v
	case string:
		var err error
		if d, err = time.ParseDuration(v); err != nil {
			return fmt.Errorf("invalid timeout %q: %w", v, err)
		}
	default:
		return fmt.Errorf(`invalid timeout %v: use a duration string like "5s" or "500ms"`, v)
	}
	if d <= 0 {
		return fmt.Errorf("invalid timeout %v: must be positive", raw)
	}
	return nil
}

func setupFlags() *flag.FlagSet {
	f := flag.NewFlagSet("config", flag.ContinueOnError)
	f.Usage = renderCustomHelp

	f.StringSliceP("query", "q", []string{}, "Domain name to query")
	f.StringSliceP("type", "t", []string{}, "DNS record type by name, number, or TYPE<number> (for example HTTPS, 65, or TYPE65)")
	f.StringSliceP("class", "c", []string{}, "Network class of the DNS record to be queried (IN, CH, HS etc)")
	f.StringSliceP("nameserver", "n", []string{}, "Address of the nameserver to send packets to")
	f.BoolP("reverse", "x", false, "Performs a DNS Lookup for an IPv4 or IPv6 address")

	f.String("gp-from", "", "Probe locations as a comma-separated list")
	f.Int("gp-limit", 1, "Limit the number of probes to use")

	f.DurationP("timeout", "T", 5*time.Second, "Sets the timeout for a query")
	f.Bool("search", true, "Use the search list provided in resolv.conf")
	f.Int("ndots", -1, "Specify the ndots parameter")
	f.BoolP("ipv4", "4", false, "Use IPv4 only")
	f.BoolP("ipv6", "6", false, "Use IPv6 only")
	f.Bool("http3", false, "Use HTTP/3 for DNS-over-HTTPS nameservers")
	f.String("strategy", "all", "Strategy to query nameservers (all, random, first, internal)")
	f.String("tls-hostname", "", "Hostname for certificate verification")
	f.Bool("skip-hostname-verification", false, "Skip TLS Hostname Verification")
	f.StringP("source", "b", "", "Bind queries to a local source IP address, like dig -b. A fixed source port is not supported (queries run concurrently)")

	f.Bool("any", false, "Query all supported DNS record types")
	f.BoolP("authoritative", "A", false, "Automatically query the authoritative nameserver for the domain")
	f.Bool("trace", false, "Trace the delegation path from the root servers to the authoritative answer")

	f.BoolP("json", "J", false, "Set the output format as JSON")
	f.Bool("short", false, "Short output format")
	f.Bool("time", false, "Display how long the response took")
	f.Bool("color", true, "Show colored output")
	f.Bool("debug", false, "Enable debug mode")

	// Add flags for DNS query options
	f.Bool("aa", false, "Set Authoritative Answer flag")
	f.Bool("ad", false, "Set Authenticated Data flag")
	f.Bool("cd", false, "Set Checking Disabled flag")
	f.Bool("rd", true, "Set Recursion Desired flag (default: true)")
	f.Bool("z", false, "Set Z flag (reserved for future use)")
	f.Bool("do", false, "Set DNSSEC OK flag")

	// Add flags for EDNS0 options
	f.Bool("nsid", false, "Request Name Server Identifier (NSID)")
	f.Bool("cookie", false, "Request DNS Cookie")
	f.Bool("padding", false, "Request EDNS padding for privacy")
	f.Bool("ede", false, "Enable EDNS to receive Extended DNS Errors")
	f.String("ecs", "", "EDNS Client Subnet (e.g., '192.0.2.0/24' or '2001:db8::/32')")
	f.Int("bufsize", 0, "EDNS UDP buffer size in bytes (512-65535); setting this enables EDNS even without other EDNS options. Default is 1232 when EDNS is enabled.")

	f.Bool("version", false, "Show version of doggo")
	f.String("config", "", "Path to a TOML config file (default: $XDG_CONFIG_HOME/doggo/config.toml or ~/.doggo.toml)")

	return f
}

func initializeApp(logger *slog.Logger, cfg *config) (*app.App, error) {
	gpConfig := globalping.Config{
		UserAgent: fmt.Sprintf("doggo/%s (https://github.com/mr-karan/doggo)", buildVersion),
		AuthToken: os.Getenv("GLOBALPING_TOKEN"),
	}
	globalpingClient := globalping.NewClient(gpConfig)

	app := app.New(logger, globalpingClient, buildVersion)

	if err := k.Unmarshal("", &app.QueryFlags); err != nil {
		return nil, fmt.Errorf("loading args: %w", err)
	}

	if err := loadNameservers(&app, k, cfg.flagSet); err != nil {
		return nil, err
	}
	return &app, nil
}

// loadNameservers resolves the effective nameservers, query types, classes,
// and names from koanf (config file/env/flags) and the positional args.
//
// Precedence for nameservers: --nameserver flag > positional @ns > config/env
// > system fallback (empty). For type/class/query: positional args override
// config/env defaults, but union with the --type/--class/--query flags
// (preserving the pre-config behavior when those flags are combined with
// positional values). This keeps an ad-hoc query on the command line from
// being silently redirected or widened by a config file.
func loadNameservers(app *app.App, k *koanf.Koanf, f *flag.FlagSet) error {
	unparsedNameservers, qt, qc, qn, err := loadUnparsedArgs(f.Args())
	if err != nil {
		return err
	}

	switch {
	case f.Changed("nameserver"):
		app.QueryFlags.Nameservers = k.Strings("nameserver")
	case len(unparsedNameservers) > 0:
		app.QueryFlags.Nameservers = unparsedNameservers
	default:
		app.QueryFlags.Nameservers = k.Strings("nameserver")
	}

	// Positional type/class/name values override config-file/env defaults but
	// union with their --type/--class/--query flag equivalents.
	// A lower-precedence `any` default must not widen an explicitly requested
	// type; an explicit --any still keeps the pre-config behavior.
	if (len(qt) > 0 || f.Changed("type")) && !f.Changed("any") {
		app.QueryFlags.QueryAny = false
	}
	if len(qt) > 0 && !f.Changed("type") {
		app.QueryFlags.QTypes = qt
	} else {
		app.QueryFlags.QTypes = append(app.QueryFlags.QTypes, qt...)
	}
	if len(qc) > 0 && !f.Changed("class") {
		app.QueryFlags.QClasses = qc
	} else {
		app.QueryFlags.QClasses = append(app.QueryFlags.QClasses, qc...)
	}
	if len(qn) > 0 && !f.Changed("query") {
		app.QueryFlags.QNames = qn
	} else {
		app.QueryFlags.QNames = append(app.QueryFlags.QNames, qn...)
	}

	// Normalize here because Globalping consumes QTypes before the normal DNS
	// path reaches PrepareQuestions. PrepareQuestions revalidates for web and
	// direct callers that do not pass through the CLI loader.
	app.QueryFlags.QTypes, err = models.NormalizeRecordTypes(app.QueryFlags.QTypes)
	if err != nil {
		return err
	}
	return nil
}

func loadResolvers(app *app.App, cfg *config) ([]resolvers.Resolver, error) {
	return resolvers.LoadResolvers(resolvers.Options{
		Nameservers:        app.Nameservers,
		UseIPv4:            app.QueryFlags.UseIPv4,
		UseIPv6:            app.QueryFlags.UseIPv6,
		UseHTTP3:           app.QueryFlags.UseHTTP3,
		SearchList:         app.ResolverOpts.SearchList,
		Ndots:              app.ResolverOpts.Ndots,
		Timeout:            cfg.timeout,
		Logger:             app.Logger,
		Strategy:           app.QueryFlags.Strategy,
		InsecureSkipVerify: app.QueryFlags.InsecureSkipVerify,
		TLSHostname:        app.QueryFlags.TLSHostname,
		SourceAddr:         app.QueryFlags.SourceAddr,
	})
}

func performLookup(app *app.App, cfg *config) ([]resolvers.Response, []error) {
	ctx, cancel := context.WithTimeout(context.Background(), cfg.timeout)
	defer cancel()

	type lookupResult struct {
		responses  []resolvers.Response
		err        error
		nameserver string
	}

	var wg sync.WaitGroup
	results := make([]lookupResult, len(app.Resolvers))

	for i, resolver := range app.Resolvers {
		wg.Add(1)
		go func(i int, r resolvers.Resolver) {
			defer wg.Done()
			responses, err := r.Lookup(ctx, app.Questions, cfg.queryFlags)
			results[i] = lookupResult{responses: responses, err: err, nameserver: r.Address()}
		}(i, resolver)
	}

	wg.Wait()

	var (
		allResponses []resolvers.Response
		allErrors    []error
	)
	for _, result := range results {
		// Collect any responses the resolver produced even when err != nil
		// so partial successes within a single resolver still surface.
		allResponses = append(allResponses, result.responses...)
		if result.err != nil {
			allErrors = append(allErrors, &resolvers.LookupError{
				Nameserver: result.nameserver,
				Err:        result.err,
			})
		}
	}
	return allResponses, allErrors
}

func outputResults(app *app.App, responses []resolvers.Response, responseErrors []error) {
	if app.QueryFlags.ShowJSON {
		outputJSON(app.Logger, responses, responseErrors)
	} else {
		// Full failure: no resolver produced a usable response. Surface every
		// per-resolver error so the user can see which nameservers failed and
		// why, then exit with the legacy lookup-failure code.
		if len(responses) == 0 && len(responseErrors) > 0 {
			for _, err := range responseErrors {
				logResolverError(app.Logger, slog.LevelError, "Error looking up DNS records", err)
			}
			os.Exit(exitLookupFailure)
		}
		// Partial success: at least one resolver answered while another
		// failed. Demote the failure to a warning and print whatever we have.
		for _, err := range responseErrors {
			logResolverError(app.Logger, slog.LevelWarn, "lookup failed", err)
		}
		if err := app.Output(responses); err != nil {
			app.Logger.Error("error writing output", "error", err)
			os.Exit(exitLookupFailure)
		}
	}

	if len(responseErrors) > 0 && len(responses) > 0 {
		os.Exit(exitPartialFailure)
	}
	if len(responseErrors) > 0 {
		os.Exit(exitLookupFailure)
	}
}

// logResolverError emits a per-resolver lookup error at the given level,
// unwrapping LookupError so the nameserver shows up as its own structured
// field rather than embedded in the message.
func logResolverError(logger *slog.Logger, level slog.Level, msg string, err error) {
	var lookupErr *resolvers.LookupError
	if errors.As(err, &lookupErr) {
		logger.Log(context.Background(), level, msg,
			"nameserver", lookupErr.Nameserver,
			"error", lookupErr.Err,
		)
		return
	}
	logger.Log(context.Background(), level, msg, "error", err)
}

// resolverErrorJSON is the per-resolver error shape returned in JSON output.
type resolverErrorJSON struct {
	Nameserver string `json:"nameserver,omitempty"`
	Error      string `json:"error"`
}

func outputJSON(logger *slog.Logger, responses []resolvers.Response, responseErrors []error) {
	jsonOutput := struct {
		Responses []resolvers.Response `json:"responses,omitempty"`
		Errors    []resolverErrorJSON  `json:"errors,omitempty"`
		// Error is kept for backwards compatibility with scripts that parsed
		// the previous schema. It is populated only on full failure.
		Error string `json:"error,omitempty"`
	}{
		Responses: responses,
	}

	for _, err := range responseErrors {
		var lookupErr *resolvers.LookupError
		if errors.As(err, &lookupErr) {
			jsonOutput.Errors = append(jsonOutput.Errors, resolverErrorJSON{
				Nameserver: lookupErr.Nameserver,
				Error:      lookupErr.Err.Error(),
			})
			continue
		}
		jsonOutput.Errors = append(jsonOutput.Errors, resolverErrorJSON{Error: err.Error()})
	}

	if len(responses) == 0 && len(responseErrors) > 0 {
		jsonOutput.Error = responseErrors[0].Error()
	}

	jsonData, err := json.MarshalIndent(jsonOutput, "", "  ")
	if err != nil {
		logger.Error("Error marshaling JSON")
		os.Exit(exitGenericFailure)
	}
	fmt.Println(string(jsonData))
}
