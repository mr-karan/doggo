package app

import (
	"errors"
	"io"
	"log/slog"
	"reflect"
	"strings"
	"testing"

	"github.com/mr-karan/doggo/pkg/models"
)

func TestLoadSystemNameserversWrapsSentinelAndCause(t *testing.T) {
	app := newTestApp()
	cause := errors.New("permission denied reading /private/system/resolvers")

	err := app.loadSystemNameserversWith(func() ([]models.Nameserver, int, []string, error) {
		return nil, 0, nil, cause
	})
	if !errors.Is(err, ErrSystemNameservers) {
		t.Fatalf("error = %v, want ErrSystemNameservers", err)
	}
	if !errors.Is(err, cause) {
		t.Fatalf("error = %v, want wrapped cause", err)
	}
	if !strings.Contains(err.Error(), cause.Error()) {
		t.Fatalf("error = %q, want detailed cause for CLI/logging", err)
	}
}

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

func TestInitNameserverClassifiesPort853AsDoT(t *testing.T) {
	tests := []struct {
		input    string
		wantType string
	}{
		{"9.9.9.9:853", models.DOTResolver},
		{"[2001:db8::1]:853", models.DOTResolver},
		{"9.9.9.9:53", models.UDPResolver},
		{"9.9.9.9", models.UDPResolver},
		{"2001:db8::1", models.UDPResolver},
		{"tls://9.9.9.9", models.DOTResolver},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			ns, err := initNameserver(tc.input)
			if err != nil {
				t.Fatalf("initNameserver(%q): %v", tc.input, err)
			}
			if ns.Type != tc.wantType {
				t.Fatalf("initNameserver(%q) type = %v, want %v", tc.input, ns.Type, tc.wantType)
			}
		})
	}
}

func TestPrimaryQueryNames(t *testing.T) {
	tests := []struct {
		name   string
		qnames []string
		ndots  int
		search []string
		want   []string
	}{
		{
			name:   "fqdn is queried as-is",
			qnames: []string{"host.foo.tld."},
			ndots:  1,
			search: []string{"bar.tld"},
			want:   []string{"host.foo.tld."},
		},
		{
			name:   "enough dots tries bare name first",
			qnames: []string{"host.example.com"},
			ndots:  1,
			search: []string{"foo.tld"},
			want:   []string{"host.example.com"},
		},
		{
			name:   "too few dots tries first search suffix first",
			qnames: []string{"host"},
			ndots:  1,
			search: []string{"foo.tld", "hq"},
			want:   []string{"host.foo.tld"},
		},
		{
			name:   "no search list falls back to bare name",
			qnames: []string{"host"},
			ndots:  1,
			search: nil,
			want:   []string{"host"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := primaryQueryNames(tc.qnames, tc.ndots, tc.search)
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("primaryQueryNames(%v, %d, %v) = %v, want %v", tc.qnames, tc.ndots, tc.search, got, tc.want)
			}
		})
	}
}

func TestEffectiveSearchSettingsMirrorsResolverOptions(t *testing.T) {
	app := newTestApp()

	// Defaults (ResolverOpts.Ndots = -1 as seeded by LoadNameservers):
	// system values apply.
	app.ResolverOpts.Ndots = -1
	app.QueryFlags.UseSearchList = true
	ndots, search := app.effectiveSearchSettings(2, []string{"foo.tld"})
	if ndots != 2 || !reflect.DeepEqual(search, []string{"foo.tld"}) {
		t.Fatalf("got ndots=%d search=%v, want 2 [foo.tld]", ndots, search)
	}

	// Configured ndots wins; --search=false suppresses the system search list.
	app.ResolverOpts.Ndots = 5
	app.QueryFlags.UseSearchList = false
	ndots, search = app.effectiveSearchSettings(2, []string{"foo.tld"})
	if ndots != 5 {
		t.Fatalf("ndots = %d, want 5", ndots)
	}
	if len(search) != 0 {
		t.Fatalf("search = %v, want none", search)
	}
}

func TestLoadSystemNameserversMergesNdotsAndSearch(t *testing.T) {
	// LoadNameservers seeds ResolverOpts.Ndots = -1 when --ndots is unset;
	// the system loader must then fill the system value (2) and apply the
	// system search list.
	app := newTestApp()
	app.ResolverOpts.Ndots = -1
	app.QueryFlags.UseSearchList = true
	err := app.loadSystemNameserversWith(func() ([]models.Nameserver, int, []string, error) {
		return nil, 2, []string{"foo.tld"}, nil
	})
	if err != nil {
		t.Fatalf("loadSystemNameserversWith: %v", err)
	}
	if app.ResolverOpts.Ndots != 2 {
		t.Fatalf("ResolverOpts.Ndots = %d, want system 2", app.ResolverOpts.Ndots)
	}
	if !reflect.DeepEqual(app.ResolverOpts.SearchList, []string{"foo.tld"}) {
		t.Fatalf("ResolverOpts.SearchList = %v, want [foo.tld]", app.ResolverOpts.SearchList)
	}

	// An already-configured ndots (5) must win over the system value, and
	// --search=false must suppress the system search list.
	app = newTestApp()
	app.ResolverOpts.Ndots = 5
	app.QueryFlags.UseSearchList = false
	err = app.loadSystemNameserversWith(func() ([]models.Nameserver, int, []string, error) {
		return nil, 2, []string{"foo.tld"}, nil
	})
	if err != nil {
		t.Fatalf("loadSystemNameserversWith: %v", err)
	}
	if app.ResolverOpts.Ndots != 5 {
		t.Fatalf("ResolverOpts.Ndots = %d, want 5", app.ResolverOpts.Ndots)
	}
	if len(app.ResolverOpts.SearchList) != 0 {
		t.Fatalf("ResolverOpts.SearchList = %v, want none with --search=false", app.ResolverOpts.SearchList)
	}
}

func TestLoadNameserversSeedsNdotsOnExplicitNameserverPath(t *testing.T) {
	// --ndots must reach the resolvers even when explicit @server
	// nameservers are given (the system loader is skipped on that path).
	app := newTestApp()
	app.QueryFlags.Ndots = 5
	app.QueryFlags.Nameservers = []string{"1.1.1.1"}
	if err := app.LoadNameservers(); err != nil {
		t.Fatalf("LoadNameservers: %v", err)
	}
	if app.ResolverOpts.Ndots != 5 {
		t.Fatalf("ResolverOpts.Ndots = %d, want 5 on explicit path", app.ResolverOpts.Ndots)
	}

	// Unset --ndots on the explicit path stays -1 (no system default to
	// apply; resolvers treat it as their own default).
	app = newTestApp()
	app.QueryFlags.Ndots = -1
	app.QueryFlags.Nameservers = []string{"1.1.1.1"}
	if err := app.LoadNameservers(); err != nil {
		t.Fatalf("LoadNameservers: %v", err)
	}
	if app.ResolverOpts.Ndots != -1 {
		t.Fatalf("ResolverOpts.Ndots = %d, want -1 when unset", app.ResolverOpts.Ndots)
	}
}

func TestPrimaryQueryNamesConvertsIDNA(t *testing.T) {
	got := primaryQueryNames([]string{"köln.example"}, 1, nil)
	want := []string{"xn--kln-sna.example"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("primaryQueryNames IDNA = %v, want %v", got, want)
	}
}
