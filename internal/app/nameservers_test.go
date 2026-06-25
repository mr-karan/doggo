package app

import (
	"io"
	"log/slog"
	"strings"
	"testing"

	"github.com/mr-karan/doggo/pkg/models"
)

func TestLoadNameserversAppliesFirstStrategyToExplicitNameservers(t *testing.T) {
	app := newTestApp()
	app.QueryFlags.Nameservers = []string{"1.0.0.1", "1.1.1.1"}
	app.QueryFlags.Strategy = "first"

	if err := app.LoadNameservers(); err != nil {
		t.Fatalf("LoadNameservers() error = %v", err)
	}

	want := []models.Nameserver{
		{Address: "1.0.0.1:53", Type: models.UDPResolver},
	}
	assertNameservers(t, app.Nameservers, want)
}

func TestLoadNameserversAppliesInternalStrategyToExplicitNameservers(t *testing.T) {
	app := newTestApp()
	app.QueryFlags.Nameservers = []string{"1.1.1.1", "10.0.0.2", "tls://172.16.0.2"}
	app.QueryFlags.Strategy = "internal"

	if err := app.LoadNameservers(); err != nil {
		t.Fatalf("LoadNameservers() error = %v", err)
	}

	want := []models.Nameserver{
		{Address: "10.0.0.2:53", Type: models.UDPResolver},
		{Address: "172.16.0.2:853", Type: models.DOTResolver},
	}
	assertNameservers(t, app.Nameservers, want)
}

func TestLoadNameserversReturnsErrorWhenExplicitInternalStrategyHasNoPrivateNameservers(t *testing.T) {
	app := newTestApp()
	app.QueryFlags.Nameservers = []string{"1.1.1.1", "8.8.8.8"}
	app.QueryFlags.Strategy = "internal"

	if err := app.LoadNameservers(); err == nil {
		t.Fatal("LoadNameservers() error = nil, want error")
	}
}

func TestIsPrivateIP(t *testing.T) {
	cases := []struct {
		ip   string
		want bool
	}{
		// RFC 1918
		{"10.0.0.1", true},
		{"172.16.0.1", true},
		{"172.31.255.255", true},
		{"172.32.0.1", false},
		{"192.168.1.1", true},
		// RFC 6598 CGNAT (e.g. Tailscale MagicDNS)
		{"100.100.100.100", true},
		{"100.64.0.0", true},
		{"100.127.255.255", true},
		{"100.63.255.255", false}, // just below the range
		{"100.128.0.0", false},    // just above the range
		// Public
		{"8.8.8.8", false},
		{"1.1.1.1", false},
		// IPv6 ULA (RFC 4193): only the locally-assigned fd00::/8 half is matched
		{"fd7a:115c:a1e0::53", true},
		{"fc00::1", false}, // reserved/unused ULA half, not matched
		{"2606:4700:4700::1111", false},
		// Invalid
		{"not-an-ip", false},
	}

	for _, tc := range cases {
		if got := isPrivateIP(tc.ip); got != tc.want {
			t.Errorf("isPrivateIP(%q) = %v, want %v", tc.ip, got, tc.want)
		}
	}
}

func newTestApp() App {
	return App{
		Logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
		QueryFlags: models.QueryFlags{
			Nameservers: []string{},
		},
		Nameservers: []models.Nameserver{},
	}
}

func TestLoadNameserversExplicitNameserverTakesPrecedenceOverAuthoritative(t *testing.T) {
	app := newTestApp()
	app.QueryFlags.Nameservers = []string{"1.1.1.1"}
	app.QueryFlags.UseAuthoritative = true
	app.QueryFlags.QNames = []string{"github.com"}

	if err := app.LoadNameservers(); err != nil {
		t.Fatalf("LoadNameservers() error = %v", err)
	}

	want := []models.Nameserver{
		{Address: "1.1.1.1:53", Type: models.UDPResolver},
	}
	assertNameservers(t, app.Nameservers, want)
}

func TestLoadAuthoritativeNameserver(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test requiring network")
	}
	app := newTestApp()
	app.QueryFlags.UseAuthoritative = true
	app.QueryFlags.QNames = []string{"github.com"}

	if err := app.LoadNameservers(); err != nil {
		t.Fatalf("LoadNameservers() error = %v", err)
	}

	if len(app.Nameservers) == 0 {
		t.Fatal("expected at least one authoritative nameserver, got none")
	}
	t.Logf("resolved authoritative NS for github.com: %v", app.Nameservers[0].Address)
}

// TestLoadAuthoritativeNameserverUsesDelegatedNS verifies the resolver targets
// come from the zone's delegated NS RRset, not the SOA primary (MNAME). amazon.com
// is the canonical case: its MNAME (dns-external-route53.us-east-1.amazonaws.com)
// is not publicly queryable, while its delegated NS set lives under awsdns-*.
func TestLoadAuthoritativeNameserverUsesDelegatedNS(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test requiring network")
	}
	app := newTestApp()
	app.QueryFlags.UseAuthoritative = true
	app.QueryFlags.QNames = []string{"amazon.com"}

	if err := app.LoadNameservers(); err != nil {
		t.Fatalf("LoadNameservers() error = %v", err)
	}

	if len(app.Nameservers) == 0 {
		t.Fatal("expected at least one authoritative nameserver, got none")
	}

	for _, ns := range app.Nameservers {
		if strings.Contains(ns.Address, "dns-external-route53") {
			t.Fatalf("selected SOA primary (MNAME) instead of delegated NS: %v", ns.Address)
		}
		if !strings.Contains(ns.Address, "awsdns") {
			t.Errorf("expected a delegated awsdns nameserver, got %v", ns.Address)
		}
	}
	t.Logf("resolved authoritative NS for amazon.com: %v", app.Nameservers)
}

func assertNameservers(t *testing.T, got, want []models.Nameserver) {
	t.Helper()

	if len(got) != len(want) {
		t.Fatalf("len(nameservers) = %d, want %d: %#v", len(got), len(want), got)
	}

	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("nameservers[%d] = %#v, want %#v", i, got[i], want[i])
		}
	}
}
