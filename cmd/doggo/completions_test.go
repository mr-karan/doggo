package main

import (
	"strings"
	"testing"
)

func TestFlagsAreIncludedInEveryCompletion(t *testing.T) {
	tests := map[string]string{
		"bash": bashCompletion,
		"zsh":  zshCompletion,
		"fish": fishCompletion,
	}
	flags := []string{"http3", "authoritative", "source"}
	for shell, completion := range tests {
		for _, flag := range flags {
			t.Run(shell+"/"+flag, func(t *testing.T) {
				if !strings.Contains(completion, flag) {
					t.Fatalf("%s completion is missing --%s", shell, flag)
				}
			})
		}
	}
}
