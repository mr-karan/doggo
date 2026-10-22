package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/knadh/koanf/v2"
)

func TestInitConfigErrors(t *testing.T) {
	previousArgs, previousConfig := os.Args, ko
	t.Cleanup(func() { os.Args, ko = previousArgs, previousConfig })
	for _, test := range []struct {
		name string
		args []string
		want string
	}{
		{"unknown flag", []string{"--invalid-flag"}, "error parsing flags"},
		{"missing file", []string{"--config", filepath.Join(t.TempDir(), "missing.toml")}, "error reading file"},
	} {
		t.Run(test.name, func(t *testing.T) {
			os.Args = append([]string{"doggo-web"}, test.args...)
			ko = koanf.New(".")
			if err := initConfig(); err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("initConfig error = %v, want %q", err, test.want)
			}
		})
	}
}

func TestInitConfigEnvironmentOverridesFile(t *testing.T) {
	previousArgs, previousConfig := os.Args, ko
	t.Cleanup(func() { os.Args, ko = previousArgs, previousConfig })
	path := filepath.Join(t.TempDir(), "config.toml")
	if err := os.WriteFile(path, []byte("[server]\naddress = ':8080'\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("DOGGO_API_SERVER__ADDRESS", "127.0.0.1:9090")
	os.Args = []string{"doggo-web", "--config", path}
	ko = koanf.New(".")
	if err := initConfig(); err != nil {
		t.Fatal(err)
	}
	if got := ko.String("server.address"); got != "127.0.0.1:9090" {
		t.Fatalf("server address = %q, want environment override", got)
	}
}
