package resolvers

import (
	"errors"
	"testing"
)

func TestLookupErrorTagsNameserver(t *testing.T) {
	inner := errors.New("connection refused")
	le := &LookupError{Nameserver: "127.0.0.1:53", Err: inner}

	got := le.Error()
	want := "127.0.0.1:53: connection refused"
	if got != want {
		t.Fatalf("Error() = %q, want %q", got, want)
	}

	if !errors.Is(le, inner) {
		t.Fatal("errors.Is should unwrap LookupError to the inner error")
	}

	var target *LookupError
	if !errors.As(le, &target) {
		t.Fatal("errors.As(&LookupError{}, &target) failed")
	}
	if target.Nameserver != "127.0.0.1:53" {
		t.Fatalf("Nameserver = %q, want 127.0.0.1:53", target.Nameserver)
	}
}

func TestLookupErrorWithoutNameserverFallsBackToInnerMessage(t *testing.T) {
	inner := errors.New("oops")
	le := &LookupError{Err: inner}
	if le.Error() != "oops" {
		t.Fatalf("Error() = %q, want %q", le.Error(), "oops")
	}
}
