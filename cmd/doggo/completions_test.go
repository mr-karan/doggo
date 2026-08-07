package main

import (
	"strings"
	"testing"
)

func TestHTTP3IsIncludedInEveryCompletion(t *testing.T) {
	tests := map[string]string{
		"bash": bashCompletion,
		"zsh":  zshCompletion,
		"fish": fishCompletion,
	}
	for shell, completion := range tests {
		t.Run(shell, func(t *testing.T) {
			if !strings.Contains(completion, "http3") {
				t.Fatalf("%s completion is missing --http3", shell)
			}
		})
	}
}
