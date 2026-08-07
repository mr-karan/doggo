package main_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/miekg/dns"
	"github.com/quic-go/quic-go/http3"
)

// integrationBuild lazily compiles the doggo binary for the integration suite
// so the tests exercise the real exit codes and stdout/stderr emitted by the
// CLI, not just the library APIs.
var (
	integrationBuildOnce sync.Once
	integrationBinPath   string
	integrationBuildErr  error
)

func doggoBin(t *testing.T) string {
	t.Helper()
	integrationBuildOnce.Do(func() {
		if testing.Short() {
			integrationBuildErr = errors.New("skipped in -short mode")
			return
		}
		dir, err := os.MkdirTemp("", "doggo-bin-")
		if err != nil {
			integrationBuildErr = err
			return
		}
		bin := filepath.Join(dir, "doggo")
		if runtime.GOOS == "windows" {
			bin += ".exe"
		}
		// Build from the cmd/doggo package; integration_test.go lives there so
		// using `.` would pick up tests-only deps. Use the module path
		// explicitly to avoid that.
		cmd := exec.Command("go", "build", "-o", bin, "github.com/mr-karan/doggo/cmd/doggo")
		out, err := cmd.CombinedOutput()
		if err != nil {
			integrationBuildErr = fmt.Errorf("go build failed: %v\n%s", err, out)
			return
		}
		integrationBinPath = bin
	})
	if integrationBuildErr != nil {
		t.Skipf("doggo binary unavailable: %v", integrationBuildErr)
	}
	return integrationBinPath
}

// startDNSServer starts a UDP DNS test server bound to 127.0.0.1 on a random
// port. The handler answers A queries for the supplied domain with the given
// IP. Returns the address as "host:port" and a shutdown function the test
// must call.
func startDNSServer(t *testing.T, domain, answer string) (string, func()) {
	t.Helper()
	mux := dns.NewServeMux()
	mux.HandleFunc(dns.Fqdn(domain), func(w dns.ResponseWriter, req *dns.Msg) {
		m := new(dns.Msg)
		m.SetReply(req)
		m.Authoritative = true
		for _, q := range req.Question {
			if q.Qtype == dns.TypeA {
				rr, err := dns.NewRR(fmt.Sprintf("%s 60 IN A %s", q.Name, answer))
				if err != nil {
					continue
				}
				m.Answer = append(m.Answer, rr)
			}
		}
		_ = w.WriteMsg(m)
	})

	conn, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0})
	if err != nil {
		t.Fatalf("ListenUDP: %v", err)
	}
	srv := &dns.Server{PacketConn: conn, Handler: mux}
	ready := make(chan struct{})
	srv.NotifyStartedFunc = func() { close(ready) }

	go func() {
		_ = srv.ActivateAndServe()
	}()

	select {
	case <-ready:
	case <-time.After(2 * time.Second):
		_ = srv.Shutdown()
		t.Fatal("DNS test server did not start within 2s")
	}

	return conn.LocalAddr().String(), func() {
		_ = srv.Shutdown()
	}
}

// startDOHHTTP3Server starts a local HTTP/3-only DoH server. A successful CLI
// lookup against it proves that --http3 selected QUIC rather than ordinary
// HTTPS; there is no TCP listener for the returned URL.
func startDOHHTTP3Server(t *testing.T, domain, answer string) string {
	t.Helper()
	bootstrap := httptest.NewTLSServer(http.NotFoundHandler())
	tlsConfig := bootstrap.TLS.Clone()
	bootstrap.Close()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.ProtoMajor != 3 || r.Method != http.MethodPost {
			http.Error(w, "HTTP/3 POST required", http.StatusBadRequest)
			return
		}
		wire, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "invalid body", http.StatusBadRequest)
			return
		}
		var request dns.Msg
		if err := request.Unpack(wire); err != nil {
			http.Error(w, "invalid DNS message", http.StatusBadRequest)
			return
		}
		response := new(dns.Msg)
		response.SetReply(&request)
		for _, question := range request.Question {
			if question.Name != dns.Fqdn(domain) || question.Qtype != dns.TypeA {
				continue
			}
			rr, err := dns.NewRR(fmt.Sprintf("%s 60 IN A %s", question.Name, answer))
			if err == nil {
				response.Answer = append(response.Answer, rr)
			}
		}
		packed, err := response.Pack()
		if err != nil {
			http.Error(w, "failed to pack DNS message", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/dns-message")
		_, _ = w.Write(packed)
	})

	conn, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0})
	if err != nil {
		t.Fatalf("ListenUDP: %v", err)
	}
	server := &http3.Server{Handler: handler, TLSConfig: tlsConfig}
	serveErr := make(chan error, 1)
	go func() { serveErr <- server.Serve(conn) }()
	t.Cleanup(func() {
		_ = server.Close()
		_ = conn.Close()
		select {
		case err := <-serveErr:
			if err != nil && !errors.Is(err, http.ErrServerClosed) && !errors.Is(err, net.ErrClosed) {
				t.Errorf("HTTP/3 server stopped with error: %v", err)
			}
		case <-time.After(2 * time.Second):
			t.Error("timed out waiting for HTTP/3 server shutdown")
		}
	})
	return "https://" + conn.LocalAddr().String() + "/dns-query"
}

// reservedClosedPort returns a TCP/UDP port that almost certainly has nothing
// listening: we bind, capture the port, then close. There is a TOCTOU window
// but it is large enough for these tests and the port lives on loopback only.
func reservedClosedPort(t *testing.T) int {
	t.Helper()
	conn, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0})
	if err != nil {
		t.Fatalf("ListenUDP: %v", err)
	}
	port := conn.LocalAddr().(*net.UDPAddr).Port
	_ = conn.Close()
	return port
}

func runDoggo(t *testing.T, args ...string) (stdout, stderr string, exit int) {
	t.Helper()
	return runDoggoEnv(t, nil, args...)
}

// runDoggoEnv runs the doggo binary with extra environment variables. The
// host's HOME, XDG_CONFIG_HOME and DOGGO_* variables are stripped and the
// config search paths are pointed at an empty temp dir so tests stay hermetic
// regardless of the developer's own doggo configuration. Values in extraEnv
// override the hermetic defaults (Go keeps the last duplicate env entry).
func runDoggoEnv(t *testing.T, extraEnv []string, args ...string) (stdout, stderr string, exit int) {
	t.Helper()
	bin := doggoBin(t)
	cmd := exec.Command(bin, args...)

	clean := t.TempDir()
	env := make([]string, 0, len(os.Environ()))
	for _, e := range os.Environ() {
		if strings.HasPrefix(e, "HOME=") ||
			strings.HasPrefix(e, "XDG_CONFIG_HOME=") ||
			strings.HasPrefix(e, "DOGGO_") {
			continue
		}
		env = append(env, e)
	}
	env = append(env, "NO_COLOR=1", "HOME="+clean, "XDG_CONFIG_HOME="+clean)
	cmd.Env = append(env, extraEnv...)

	var outBuf, errBuf strings.Builder
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf
	err := cmd.Run()
	exit = 0
	if err != nil {
		var ee *exec.ExitError
		if errors.As(err, &ee) {
			exit = ee.ExitCode()
		} else {
			t.Fatalf("cmd.Run: %v\nstderr: %s", err, errBuf.String())
		}
	}
	return outBuf.String(), errBuf.String(), exit
}

// writeConfigFile writes a doggo config.toml under dir and returns its path.
func writeConfigFile(t *testing.T, dir, contents string) string {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	path := filepath.Join(dir, "config.toml")
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	return path
}

func TestHTTP3WithoutDOHNameserverFailsClearly(t *testing.T) {
	stdout, stderr, exit := runDoggo(t,
		"--http3",
		"--nameserver=127.0.0.1",
		"example.test",
	)
	if exit == 0 {
		t.Fatalf("exit = 0, want failure\nstdout:\n%s\nstderr:\n%s", stdout, stderr)
	}
	if !strings.Contains(stderr, "HTTP/3 requires at least one HTTPS (DoH) nameserver") {
		t.Fatalf("stderr missing clear HTTP/3 requirement\nstderr:\n%s", stderr)
	}
}

func TestCLIUsesHTTP3ForDOH(t *testing.T) {
	serverURL := startDOHHTTP3Server(t, "example.test", "192.0.2.188")
	stdout, stderr, exit := runDoggo(t,
		"--http3",
		"--skip-hostname-verification",
		"--timeout=2s",
		"--short",
		"@"+serverURL,
		"example.test",
	)
	if exit != 0 {
		t.Fatalf("exit = %d, want 0\nstdout:\n%s\nstderr:\n%s", exit, stdout, stderr)
	}
	if !strings.Contains(stdout, "192.0.2.188") {
		t.Fatalf("stdout missing local HTTP/3 answer\nstdout:\n%s", stdout)
	}
}

func TestPartialFailureExitsTwoAndPrintsResponse(t *testing.T) {
	serverAddr, stop := startDNSServer(t, "example.test", "192.0.2.10")
	defer stop()

	deadPort := reservedClosedPort(t)
	deadAddr := fmt.Sprintf("127.0.0.1:%d", deadPort)

	stdout, stderr, exit := runDoggo(t,
		"--timeout=2s",
		"@"+serverAddr,
		"@"+deadAddr,
		"A",
		"example.test",
	)

	if exit != 2 {
		t.Fatalf("exit = %d, want 2 (partial failure)\nstdout:\n%s\nstderr:\n%s", exit, stdout, stderr)
	}
	if !strings.Contains(stdout, "192.0.2.10") {
		t.Fatalf("stdout missing successful answer\nstdout:\n%s", stdout)
	}
	if !strings.Contains(stderr, "lookup failed") {
		t.Fatalf("stderr missing per-resolver warning\nstderr:\n%s", stderr)
	}
	if !strings.Contains(stderr, deadAddr) {
		t.Fatalf("stderr missing dead nameserver identity\nstderr:\n%s", stderr)
	}
}

func TestFullFailureExitsNine(t *testing.T) {
	deadPort := reservedClosedPort(t)
	deadAddr := fmt.Sprintf("127.0.0.1:%d", deadPort)

	stdout, stderr, exit := runDoggo(t,
		"--timeout=2s",
		"@"+deadAddr,
		"A",
		"example.test",
	)

	if exit != 9 {
		t.Fatalf("exit = %d, want 9 (full failure)\nstdout:\n%s\nstderr:\n%s", exit, stdout, stderr)
	}
	if !strings.Contains(stderr, "Error looking up DNS records") {
		t.Fatalf("stderr missing top-level failure message\nstderr:\n%s", stderr)
	}
	if !strings.Contains(stderr, deadAddr) {
		t.Fatalf("stderr missing dead nameserver identity\nstderr:\n%s", stderr)
	}
}

func TestCleanSuccessExitsZero(t *testing.T) {
	serverAddr, stop := startDNSServer(t, "clean.test", "192.0.2.20")
	defer stop()

	stdout, _, exit := runDoggo(t,
		"--timeout=2s",
		"@"+serverAddr,
		"A",
		"clean.test",
	)

	if exit != 0 {
		t.Fatalf("exit = %d, want 0", exit)
	}
	if !strings.Contains(stdout, "192.0.2.20") {
		t.Fatalf("stdout missing answer\nstdout:\n%s", stdout)
	}
}

func TestRecordTypeFormsReachTheWire(t *testing.T) {
	serverAddr, stop := startDNSServer(t, "types.test", "192.0.2.25")
	defer stop()

	tests := []struct {
		name     string
		extraEnv []string
		args     []string
		want     string
	}{
		{name: "positional HTTPS", args: []string{"HTTPS", "types.test"}, want: "HTTPS"},
		{name: "positional SVCB", args: []string{"SVCB", "types.test"}, want: "SVCB"},
		{name: "positional TYPE65", args: []string{"TYPE65", "types.test"}, want: "HTTPS"},
		{name: "positional decimal", args: []string{"65", "types.test"}, want: "HTTPS"},
		{name: "type flag HTTPS", args: []string{"--type", "HTTPS", "types.test"}, want: "HTTPS"},
		{name: "type flag SVCB", args: []string{"--type", "SVCB", "types.test"}, want: "SVCB"},
		{name: "type flag TYPE65", args: []string{"--type", "TYPE65", "types.test"}, want: "HTTPS"},
		{name: "type flag decimal", args: []string{"--type", "65", "types.test"}, want: "HTTPS"},
		{name: "reserved upper boundary decimal", args: []string{"--type", "65535", "types.test"}, want: "TYPE65535"},
		{name: "reserved upper boundary TYPE", args: []string{"TYPE65535", "types.test"}, want: "TYPE65535"},
		{name: "environment", extraEnv: []string{"DOGGO_TYPE=TYPE64"}, args: []string{"types.test"}, want: "SVCB"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			args := append([]string{"--json", "--timeout=2s", "@" + serverAddr}, test.args...)
			stdout, stderr, exit := runDoggoEnv(t, test.extraEnv, args...)
			if exit != 0 {
				t.Fatalf("exit = %d, want 0\nstdout:\n%s\nstderr:\n%s", exit, stdout, stderr)
			}
			var payload struct {
				Responses []struct {
					Questions []struct {
						Type string `json:"type"`
					} `json:"questions"`
				} `json:"responses"`
			}
			if err := json.Unmarshal([]byte(stdout), &payload); err != nil {
				t.Fatalf("invalid JSON: %v\nstdout:\n%s", err, stdout)
			}
			if len(payload.Responses) != 1 || len(payload.Responses[0].Questions) != 1 {
				t.Fatalf("unexpected response questions\nstdout:\n%s", stdout)
			}
			if got := payload.Responses[0].Questions[0].Type; got != test.want {
				t.Fatalf("question type = %q, want %q", got, test.want)
			}
		})
	}

	t.Run("config", func(t *testing.T) {
		xdg := t.TempDir()
		writeConfigFile(t, filepath.Join(xdg, "doggo"), `type = "HTTPS"`)
		stdout, stderr, exit := runDoggoEnv(t,
			[]string{"XDG_CONFIG_HOME=" + xdg},
			"--json", "--timeout=2s", "@"+serverAddr, "types.test",
		)
		if exit != 0 {
			t.Fatalf("exit = %d, want 0\nstdout:\n%s\nstderr:\n%s", exit, stdout, stderr)
		}
		if !strings.Contains(stdout, `"type": "HTTPS"`) {
			t.Fatalf("configured HTTPS type did not reach the wire\nstdout:\n%s", stdout)
		}
	})
}

func TestPositionalNoneClassReachesTheWire(t *testing.T) {
	serverAddr, stop := startDNSServer(t, "class.test", "192.0.2.26")
	defer stop()

	stdout, stderr, exit := runDoggo(t,
		"--json", "--timeout=2s", "@"+serverAddr, "NONE", "class.test",
	)
	if exit != 0 {
		t.Fatalf("exit = %d, want 0\nstdout:\n%s\nstderr:\n%s", exit, stdout, stderr)
	}
	var payload struct {
		Responses []struct {
			Questions []struct {
				Type  string `json:"type"`
				Class string `json:"class"`
			} `json:"questions"`
		} `json:"responses"`
	}
	if err := json.Unmarshal([]byte(stdout), &payload); err != nil {
		t.Fatalf("invalid JSON: %v\nstdout:\n%s", err, stdout)
	}
	seen := map[string]bool{}
	for _, response := range payload.Responses {
		for _, question := range response.Questions {
			if question.Class != "NONE" {
				t.Fatalf("question class = %q, want NONE", question.Class)
			}
			if question.Type == "TYPE0" || question.Type == "None" {
				t.Fatalf("positional NONE became QTYPE 0: %+v", question)
			}
			seen[question.Type] = true
		}
	}
	if !seen["A"] || !seen["AAAA"] {
		t.Fatalf("question types = %v, want default A and AAAA", seen)
	}
}

func TestInvalidRecordTypesFailClearly(t *testing.T) {
	assertFailure := func(t *testing.T, stdout, stderr string, exit int, input, wantMessage string) {
		t.Helper()
		if exit != 1 {
			t.Fatalf("exit = %d, want 1\nstdout:\n%s\nstderr:\n%s", exit, stdout, stderr)
		}
		output := stdout + stderr
		if !strings.Contains(output, wantMessage) {
			t.Fatalf("output does not contain %q\n%s", wantMessage, output)
		}
		if !strings.Contains(output, input) {
			t.Fatalf("output does not identify invalid value %q\n%s", input, output)
		}
	}

	tests := []struct {
		name     string
		extraEnv []string
		args     []string
		input    string
		message  string
	}{
		{name: "invalid name", args: []string{"--type", "NOTATYPE", "example.test"}, input: "NOTATYPE", message: "invalid DNS record type"},
		{name: "invalid TYPE notation", args: []string{"--type", "TYPEfoo", "example.test"}, input: "TYPEfoo", message: "invalid DNS record type"},
		{name: "reserved zero", args: []string{"--type", "0", "example.test"}, input: "0", message: "reserved and cannot be used in a question"},
		{name: "reserved TYPE0", args: []string{"TYPE0", "example.test"}, input: "TYPE0", message: "reserved and cannot be used in a question"},
		{name: "OPT meta type", args: []string{"--type", "OPT", "example.test"}, input: "OPT", message: "cannot be used in a question"},
		{name: "TKEY meta type", args: []string{"--type", "249", "example.test"}, input: "249", message: "cannot be used in a question"},
		{name: "TSIG meta type", args: []string{"TYPE250", "example.test"}, input: "TYPE250", message: "cannot be used in a question"},
		{name: "decimal out of range", args: []string{"--type", "65536", "example.test"}, input: "65536", message: "out of range"},
		{name: "TYPE out of range", args: []string{"TYPE65536", "example.test"}, input: "TYPE65536", message: "out of range"},
		{name: "reverse with invalid type", args: []string{"--reverse", "--type", "NOTATYPE", "192.0.2.1"}, input: "NOTATYPE", message: "invalid DNS record type"},
		{name: "environment", extraEnv: []string{"DOGGO_TYPE=NOTATYPE"}, args: []string{"example.test"}, input: "NOTATYPE", message: "invalid DNS record type"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			stdout, stderr, exit := runDoggoEnv(t, test.extraEnv, test.args...)
			assertFailure(t, stdout, stderr, exit, test.input, test.message)
		})
	}

	t.Run("config", func(t *testing.T) {
		xdg := t.TempDir()
		writeConfigFile(t, filepath.Join(xdg, "doggo"), `type = "NOTATYPE"`)
		stdout, stderr, exit := runDoggoEnv(t,
			[]string{"XDG_CONFIG_HOME=" + xdg},
			"example.test",
		)
		assertFailure(t, stdout, stderr, exit, "NOTATYPE", "invalid DNS record type")
	})
}

func TestInvalidIDNExitsGenericFailure(t *testing.T) {
	stdout, stderr, exit := runDoggo(t, "xn--example-.com")
	if exit != 1 {
		t.Fatalf("exit = %d, want 1\nstdout:\n%s\nstderr:\n%s", exit, stdout, stderr)
	}
	if !strings.Contains(stderr, "Error preparing DNS questions") {
		t.Fatalf("stderr missing question-preparation error\nstderr:\n%s", stderr)
	}
}

func TestPartialFailureJSONOutputIncludesErrorsArray(t *testing.T) {
	serverAddr, stop := startDNSServer(t, "json.test", "192.0.2.30")
	defer stop()

	deadPort := reservedClosedPort(t)
	deadAddr := fmt.Sprintf("127.0.0.1:%d", deadPort)

	stdout, stderr, exit := runDoggo(t,
		"--timeout=2s",
		"--json",
		"@"+serverAddr,
		"@"+deadAddr,
		"A",
		"json.test",
	)

	if exit != 2 {
		t.Fatalf("exit = %d, want 2\nstdout:\n%s\nstderr:\n%s", exit, stdout, stderr)
	}

	var payload struct {
		Responses []map[string]any `json:"responses"`
		Errors    []struct {
			Nameserver string `json:"nameserver"`
			Error      string `json:"error"`
		} `json:"errors"`
		Error string `json:"error"`
	}
	if err := json.Unmarshal([]byte(stdout), &payload); err != nil {
		t.Fatalf("invalid JSON: %v\nstdout:\n%s", err, stdout)
	}
	if len(payload.Responses) == 0 {
		t.Fatalf("expected at least one response, got %d\nstdout:\n%s", len(payload.Responses), stdout)
	}
	if len(payload.Errors) == 0 {
		t.Fatalf("expected populated errors[], got 0\nstdout:\n%s", stdout)
	}
	if payload.Errors[0].Nameserver != deadAddr {
		t.Fatalf("errors[0].nameserver = %q, want %q", payload.Errors[0].Nameserver, deadAddr)
	}
	if payload.Error != "" {
		t.Fatalf(`legacy "error" field should be empty on partial failure, got %q`, payload.Error)
	}
}

func TestFullFailureJSONOutputPopulatesLegacyErrorField(t *testing.T) {
	deadPort := reservedClosedPort(t)
	deadAddr := fmt.Sprintf("127.0.0.1:%d", deadPort)

	stdout, _, exit := runDoggo(t,
		"--timeout=2s",
		"--json",
		"@"+deadAddr,
		"A",
		"json.test",
	)

	if exit != 9 {
		t.Fatalf("exit = %d, want 9\nstdout:\n%s", exit, stdout)
	}

	var payload struct {
		Errors []struct {
			Nameserver string `json:"nameserver"`
			Error      string `json:"error"`
		} `json:"errors"`
		Error string `json:"error"`
	}
	if err := json.Unmarshal([]byte(stdout), &payload); err != nil {
		t.Fatalf("invalid JSON: %v\nstdout:\n%s", err, stdout)
	}
	if payload.Error == "" {
		t.Fatalf(`legacy "error" should be populated on full failure\nstdout:\n%s`, stdout)
	}
	if len(payload.Errors) == 0 {
		t.Fatalf("errors[] should still be populated for new clients\nstdout:\n%s", stdout)
	}
}

func TestDebugLogsStrategyApplication(t *testing.T) {
	serverAddr, stop := startDNSServer(t, "debug.test", "192.0.2.40")
	defer stop()

	deadPort := reservedClosedPort(t)
	deadAddr := fmt.Sprintf("127.0.0.1:%d", deadPort)

	_, stderr, exit := runDoggo(t,
		"--timeout=2s",
		"--debug",
		"--strategy=first",
		"@"+serverAddr,
		"@"+deadAddr,
		"A",
		"debug.test",
	)
	if exit != 0 && exit != 2 {
		t.Fatalf("exit = %d, want 0 or 2\nstderr:\n%s", exit, stderr)
	}

	// Debug log should describe the strategy decision so users no longer have
	// to guess why their second @host was silently dropped.
	if !strings.Contains(stderr, "Applying nameserver strategy") {
		t.Fatalf("missing strategy-application debug log\nstderr:\n%s", stderr)
	}
	if !strings.Contains(stderr, "Applied nameserver strategy") {
		t.Fatalf("missing strategy-applied debug log\nstderr:\n%s", stderr)
	}
	if !strings.Contains(stderr, `source=explicit`) {
		t.Fatalf(`missing source="explicit" label\nstderr:\n%s`, stderr)
	}
	if !regexp.MustCompile(`dropped_count=1`).MatchString(stderr) {
		t.Fatalf("missing dropped_count=1 indicating @deadAddr was filtered\nstderr:\n%s", stderr)
	}
}

// TestConfigFileSetsDefaults verifies a config file at the default XDG path
// changes CLI behavior without any flags: strategy=first must drop the second
// nameserver and the query must succeed via the first.
func TestConfigFileSetsDefaults(t *testing.T) {
	serverAddr, stop := startDNSServer(t, "cfg.test", "192.0.2.50")
	defer stop()

	deadPort := reservedClosedPort(t)
	deadAddr := fmt.Sprintf("127.0.0.1:%d", deadPort)

	xdg := t.TempDir()
	writeConfigFile(t, filepath.Join(xdg, "doggo"), "strategy = \"first\"\n")

	stdout, stderr, exit := runDoggoEnv(t,
		[]string{"XDG_CONFIG_HOME=" + xdg},
		"--timeout=2s", "--debug",
		"@"+serverAddr, "@"+deadAddr,
		"A", "cfg.test",
	)

	if exit != 0 {
		t.Fatalf("exit = %d, want 0 (strategy=first should only use the live server)\nstdout:\n%s\nstderr:\n%s", exit, stdout, stderr)
	}
	if !strings.Contains(stdout, "192.0.2.50") {
		t.Fatalf("stdout missing answer\nstdout:\n%s", stdout)
	}
	if !regexp.MustCompile(`strategy=first`).MatchString(stderr) {
		t.Fatalf("missing strategy=first from config file in debug log\nstderr:\n%s", stderr)
	}
	if !regexp.MustCompile(`dropped_count=1`).MatchString(stderr) {
		t.Fatalf("missing dropped_count=1 indicating second nameserver was filtered\nstderr:\n%s", stderr)
	}
}

// TestConfigFileJSONOutput verifies a boolean output option from the config
// file (json = true) is honored end to end.
func TestConfigFileJSONOutput(t *testing.T) {
	serverAddr, stop := startDNSServer(t, "cfgjson.test", "192.0.2.60")
	defer stop()

	xdg := t.TempDir()
	writeConfigFile(t, filepath.Join(xdg, "doggo"), "json = true\n")

	stdout, stderr, exit := runDoggoEnv(t,
		[]string{"XDG_CONFIG_HOME=" + xdg},
		"--timeout=2s", "@"+serverAddr, "A", "cfgjson.test",
	)

	if exit != 0 {
		t.Fatalf("exit = %d, want 0\nstderr:\n%s", exit, stderr)
	}
	var payload struct {
		Responses []map[string]any `json:"responses"`
	}
	if err := json.Unmarshal([]byte(stdout), &payload); err != nil {
		t.Fatalf("stdout is not JSON despite json=true in config: %v\nstdout:\n%s", err, stdout)
	}
	if len(payload.Responses) == 0 {
		t.Fatalf("expected responses in JSON output\nstdout:\n%s", stdout)
	}
}

// TestCLIFlagOverridesConfigFile verifies explicit CLI flags beat the config
// file: json = true in the file, but --json=false on the command line.
func TestCLIFlagOverridesConfigFile(t *testing.T) {
	serverAddr, stop := startDNSServer(t, "cfgoverride.test", "192.0.2.70")
	defer stop()

	xdg := t.TempDir()
	writeConfigFile(t, filepath.Join(xdg, "doggo"), "json = true\n")

	stdout, stderr, exit := runDoggoEnv(t,
		[]string{"XDG_CONFIG_HOME=" + xdg},
		"--timeout=2s", "--json=false", "@"+serverAddr, "A", "cfgoverride.test",
	)

	if exit != 0 {
		t.Fatalf("exit = %d, want 0\nstderr:\n%s", exit, stderr)
	}
	if json.Valid([]byte(stdout)) {
		t.Fatalf("stdout is JSON despite --json=false overriding config\nstdout:\n%s", stdout)
	}
	if !strings.Contains(stdout, "192.0.2.70") {
		t.Fatalf("stdout missing answer\nstdout:\n%s", stdout)
	}
}

// TestEnvVarSetsStrategy verifies DOGGO_* env vars map onto flags:
// DOGGO_STRATEGY=first must narrow the nameserver set to the first entry.
func TestEnvVarSetsStrategy(t *testing.T) {
	serverAddr, stop := startDNSServer(t, "env.test", "192.0.2.80")
	defer stop()

	deadPort := reservedClosedPort(t)
	deadAddr := fmt.Sprintf("127.0.0.1:%d", deadPort)

	stdout, stderr, exit := runDoggoEnv(t,
		[]string{"DOGGO_STRATEGY=first"},
		"--timeout=2s", "--debug",
		"@"+serverAddr, "@"+deadAddr,
		"A", "env.test",
	)

	if exit != 0 {
		t.Fatalf("exit = %d, want 0\nstdout:\n%s\nstderr:\n%s", exit, stdout, stderr)
	}
	if !regexp.MustCompile(`strategy=first`).MatchString(stderr) {
		t.Fatalf("missing strategy=first from env var in debug log\nstderr:\n%s", stderr)
	}
	if !regexp.MustCompile(`dropped_count=1`).MatchString(stderr) {
		t.Fatalf("missing dropped_count=1 indicating second nameserver was filtered\nstderr:\n%s", stderr)
	}
}

// TestExplicitConfigFlagLoadsFile verifies --config loads a file from an
// arbitrary location outside the default search paths.
func TestExplicitConfigFlagLoadsFile(t *testing.T) {
	serverAddr, stop := startDNSServer(t, "explicit.test", "192.0.2.90")
	defer stop()

	deadPort := reservedClosedPort(t)
	deadAddr := fmt.Sprintf("127.0.0.1:%d", deadPort)

	cfgPath := writeConfigFile(t, t.TempDir(), "strategy = \"first\"\n")

	stdout, stderr, exit := runDoggo(t,
		"--timeout=2s", "--debug", "--config", cfgPath,
		"@"+serverAddr, "@"+deadAddr,
		"A", "explicit.test",
	)

	if exit != 0 {
		t.Fatalf("exit = %d, want 0\nstdout:\n%s\nstderr:\n%s", exit, stdout, stderr)
	}
	if !regexp.MustCompile(`strategy=first`).MatchString(stderr) {
		t.Fatalf("missing strategy=first from --config file in debug log\nstderr:\n%s", stderr)
	}
}

// TestMissingExplicitConfigFails verifies that requesting a nonexistent
// config file is a hard error, not silently ignored.
func TestMissingExplicitConfigFails(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "nope.toml")

	stdout, _, exit := runDoggo(t, "--config", missing, "example.test")

	if exit != 1 {
		t.Fatalf("exit = %d, want 1 for missing --config file\nstdout:\n%s", exit, stdout)
	}
	if !strings.Contains(stdout, "config") {
		t.Fatalf("error output should mention the config file\nstdout:\n%s", stdout)
	}
}

// TestMalformedConfigFails verifies a config file with invalid TOML at a
// default search path is a hard error.
func TestMalformedConfigFails(t *testing.T) {
	xdg := t.TempDir()
	writeConfigFile(t, filepath.Join(xdg, "doggo"), "strategy = [unclosed")

	stdout, _, exit := runDoggoEnv(t,
		[]string{"XDG_CONFIG_HOME=" + xdg},
		"example.test",
	)

	if exit != 1 {
		t.Fatalf("exit = %d, want 1 for malformed config\nstdout:\n%s", exit, stdout)
	}
	if !strings.Contains(stdout, "config") {
		t.Fatalf("error output should mention the config file\nstdout:\n%s", stdout)
	}
}

// TestPositionalNameserverBeatsConfigFile guards against issue M1 from code
// review: a nameserver set in the config file must not override a positional
// @ns argument, otherwise an ad-hoc query is silently sent to the wrong
// resolver.
func TestPositionalNameserverBeatsConfigFile(t *testing.T) {
	serverAddr, stop := startDNSServer(t, "pos.test", "192.0.2.120")
	defer stop()

	xdg := t.TempDir()
	// Config points at a public resolver that does not serve pos.test; the
	// positional @<serverAddr> must win for the query to succeed.
	writeConfigFile(t, filepath.Join(xdg, "doggo"), "nameserver = [\"1.1.1.1\"]\n")

	stdout, stderr, exit := runDoggoEnv(t,
		[]string{"XDG_CONFIG_HOME=" + xdg},
		"--timeout=2s", "@"+serverAddr, "A", "pos.test",
	)

	if exit != 0 {
		t.Fatalf("exit = %d, want 0 (positional @ns must beat config)\nstdout:\n%s\nstderr:\n%s", exit, stdout, stderr)
	}
	if !strings.Contains(stdout, "192.0.2.120") {
		t.Fatalf("stdout missing answer from positional @ns; config resolver leaked in?\nstdout:\n%s", stdout)
	}
}

// TestPositionalTypeBeatsConfigFile guards against issue M2: a type set in
// the config file must not widen an ad-hoc positional query. With config
// type=["A"] and a positional MX, the effective query must be MX only. The
// test server answers A and nothing else; an MX query returns an empty
// NOERROR response, so:
//   - with the M2 bug (A unioned in), the A answer 192.0.2.130 would appear;
//   - with the fix (MX replaces A), the table is empty.
//
// Asserting the A answer is absent proves the positional type replaced the
// config default end-to-end.
func TestPositionalTypeBeatsConfigFile(t *testing.T) {
	serverAddr, stop := startDNSServer(t, "ptype.test", "192.0.2.130")
	defer stop()

	xdg := t.TempDir()
	writeConfigFile(t, filepath.Join(xdg, "doggo"), "type = [\"A\"]\n")

	stdout, stderr, exit := runDoggoEnv(t,
		[]string{"XDG_CONFIG_HOME=" + xdg},
		"--timeout=2s", "@"+serverAddr, "MX", "ptype.test",
	)

	if exit != 0 {
		t.Fatalf("exit = %d, want 0 (empty MX response is still success)\nstdout:\n%s\nstderr:\n%s", exit, stdout, stderr)
	}
	if strings.Contains(stdout, "192.0.2.130") {
		t.Fatalf("A answer leaked in; config type was unioned with positional MX instead of replaced\nstdout:\n%s", stdout)
	}
}

func TestExplicitTypeBeatsConfiguredAny(t *testing.T) {
	serverAddr, stop := startDNSServer(t, "any.test", "192.0.2.140")
	defer stop()

	xdg := t.TempDir()
	writeConfigFile(t, filepath.Join(xdg, "doggo"), "any = true\n")

	stdout, stderr, exit := runDoggoEnv(t,
		[]string{"XDG_CONFIG_HOME=" + xdg},
		"--json", "--timeout=2s", "@"+serverAddr, "MX", "any.test",
	)
	if exit != 0 {
		t.Fatalf("exit = %d, want 0\nstdout:\n%s\nstderr:\n%s", exit, stdout, stderr)
	}

	var payload struct {
		Responses []struct {
			Questions []struct {
				Type string `json:"type"`
			} `json:"questions"`
		} `json:"responses"`
	}
	if err := json.Unmarshal([]byte(stdout), &payload); err != nil {
		t.Fatalf("invalid JSON: %v\nstdout:\n%s", err, stdout)
	}
	if len(payload.Responses) == 0 {
		t.Fatalf("expected a DNS response\nstdout:\n%s", stdout)
	}
	for _, response := range payload.Responses {
		for _, question := range response.Questions {
			if question.Type != "MX" {
				t.Fatalf("question type = %q, want MX; configured any=true widened the query", question.Type)
			}
		}
	}
}

func TestGlobalpingDefaultWithoutQueryShowsHelp(t *testing.T) {
	stdout, stderr, exit := runDoggoEnv(t, []string{"DOGGO_GP_FROM=Germany"})

	if exit != 0 {
		t.Fatalf("exit = %d, want 0\nstdout:\n%s\nstderr:\n%s", exit, stdout, stderr)
	}
	if !strings.Contains(stdout, "USAGE:") {
		t.Fatalf("stdout missing help text\nstdout:\n%s", stdout)
	}
	if strings.Contains(stderr, "panic:") {
		t.Fatalf("stderr contains panic\nstderr:\n%s", stderr)
	}
}

// TestUnknownConfigKeyFails verifies a typo'd config key is rejected rather
// than silently ignored.
func TestUnknownConfigKeyFails(t *testing.T) {
	xdg := t.TempDir()
	writeConfigFile(t, filepath.Join(xdg, "doggo"), "strategiy = \"first\"\n")

	stdout, _, exit := runDoggoEnv(t,
		[]string{"XDG_CONFIG_HOME=" + xdg},
		"example.test",
	)

	if exit != 1 {
		t.Fatalf("exit = %d, want 1 for unknown config key\nstdout:\n%s", exit, stdout)
	}
	if !strings.Contains(stdout, "unknown config key") {
		t.Fatalf("error output should mention unknown config key\nstdout:\n%s", stdout)
	}
}
