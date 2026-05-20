package app

import (
	"io"
	"log/slog"
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

func newTestApp() App {
	return App{
		Logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
		QueryFlags: models.QueryFlags{
			Nameservers: []string{},
		},
		Nameservers: []models.Nameserver{},
	}
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
