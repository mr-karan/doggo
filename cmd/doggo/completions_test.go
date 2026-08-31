package main

import (
	"strings"
	"testing"
)

// TestFlagsAreIncludedInEveryCompletion asserts that flags added after the
// completion scripts were first written are wired up in every shell, using
// the shell-specific syntax so a bare mention in a description cannot pass.
func TestFlagsAreIncludedInEveryCompletion(t *testing.T) {
	tests := map[string]struct {
		completion string
		expect     []string
	}{
		"bash": {
			completion: bashCompletion,
			expect: []string{
				"-A --authoritative", // in the opts word list
				"-b --source",
			},
		},
		"zsh": {
			completion: zshCompletion,
			expect: []string{
				"'(-A --authoritative)'{-A,--authoritative}",
				"'(-b --source)'{-b,--source}",
				"--gp-limit[Limit the number of probes to use from Globalping]:limit",
			},
		},
		"fish": {
			completion: fishCompletion,
			expect: []string{
				"-s 'A' -l 'authoritative'",
				"-s 'b' -l 'source'",
				"-l 'gp-limit' -d \"Limit the number of probes to use from Globalping\" -x",
				"-l 'ndots'     -d \"Specify ndots parameter\" -x",
			},
		},
	}
	for shell, tc := range tests {
		for _, want := range tc.expect {
			t.Run(shell+"/"+want, func(t *testing.T) {
				if !strings.Contains(tc.completion, want) {
					t.Fatalf("%s completion is missing %q", shell, want)
				}
			})
		}
	}
}

// TestBooleanFlagsDoNotTakeValues asserts pflag booleans are not completed
// with a separate value argument (--search false is invalid; --search=false
// is the explicit form).
func TestBooleanFlagsDoNotTakeValues(t *testing.T) {
	if strings.Contains(zshCompletion, "--search[Use the search list defined in resolv.conf]:") {
		t.Fatal("zsh completion offers a separate value for boolean --search")
	}
	if strings.Contains(fishCompletion, `-l 'search'    -d "Use the search list defined in resolv.conf" -x`) {
		t.Fatal("fish completion requires a value for boolean --search")
	}
	if strings.Contains(bashCompletion, `--search|--color|--http3)
            COMPREPLY=( $(compgen -W "true false"`) {
		t.Fatal("bash completion offers a separate value for boolean flags")
	}
}
