package main_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
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
	bin := doggoBin(t)
	cmd := exec.Command(bin, args...)
	cmd.Env = append(os.Environ(), "NO_COLOR=1")
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
