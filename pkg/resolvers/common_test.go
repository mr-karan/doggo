package resolvers

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"sync/atomic"
	"testing"
	"time"

	"github.com/miekg/dns"
)

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

// TestConcurrentLookupReturnsCompletedWorkAfterContextExpiry guards against
// the prior race where ctx.Done() in the parent select caused already-finished
// goroutine results to be dropped on the floor. The first goroutine completes
// successfully and cancels the parent context; the second blocks until
// cancellation and then returns ctx.Err() — both must reach the caller.
func TestConcurrentLookupReturnsCompletedWorkAfterContextExpiry(t *testing.T) {
	good := dns.Question{Name: "good.example.com.", Qtype: dns.TypeA, Qclass: dns.ClassINET}
	slow := dns.Question{Name: "slow.example.com.", Qtype: dns.TypeA, Qclass: dns.ClassINET}

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	qf := func(ctx context.Context, q dns.Question, _ QueryFlags) (Response, error) {
		if q.Name == good.Name {
			cancel()
			return Response{Questions: []Question{{Name: q.Name, Type: "A", Class: "IN"}}}, nil
		}
		// Block deterministically until the parent context is cancelled so
		// this goroutine cannot finish before the cancel signal lands.
		<-ctx.Done()
		return Response{}, ctx.Err()
	}

	responses, err := ConcurrentLookup(ctx, []dns.Question{good, slow}, QueryFlags{}, qf, discardLogger())

	if len(responses) != 1 {
		t.Fatalf("len(responses) = %d, want 1 (the completed work)", len(responses))
	}
	if responses[0].Questions[0].Name != good.Name {
		t.Fatalf("responses[0].Questions[0].Name = %q, want %q", responses[0].Questions[0].Name, good.Name)
	}
	if err == nil {
		t.Fatal("err = nil, want joined error describing the cancelled question")
	}
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v, want errors.Is(err, context.Canceled)", err)
	}
}

// TestConcurrentLookupJoinsPerQuestionErrors ensures partial per-question
// failures are surfaced via errors.Join rather than discarded.
func TestConcurrentLookupJoinsPerQuestionErrors(t *testing.T) {
	a := dns.Question{Name: "a.example.com.", Qtype: dns.TypeA, Qclass: dns.ClassINET}
	b := dns.Question{Name: "b.example.com.", Qtype: dns.TypeA, Qclass: dns.ClassINET}

	errBoom := errors.New("boom")
	errKaboom := errors.New("kaboom")

	qf := func(_ context.Context, q dns.Question, _ QueryFlags) (Response, error) {
		switch q.Name {
		case a.Name:
			return Response{}, errBoom
		case b.Name:
			return Response{}, errKaboom
		}
		return Response{}, errors.New("unexpected question")
	}

	responses, err := ConcurrentLookup(context.Background(), []dns.Question{a, b}, QueryFlags{}, qf, discardLogger())
	if len(responses) != 0 {
		t.Fatalf("len(responses) = %d, want 0", len(responses))
	}
	if err == nil {
		t.Fatal("err = nil, want joined error")
	}
	if !errors.Is(err, errBoom) || !errors.Is(err, errKaboom) {
		t.Fatalf("err = %v, want both errBoom and errKaboom to be unwrappable", err)
	}
}

// TestConcurrentLookupSurfacesPartialErrorAlongsideResponse verifies that a
// successful question and a failed question in the same call both reach the
// caller — the response in the slice, the error via errors.Join.
func TestConcurrentLookupSurfacesPartialErrorAlongsideResponse(t *testing.T) {
	good := dns.Question{Name: "good.example.com.", Qtype: dns.TypeA, Qclass: dns.ClassINET}
	bad := dns.Question{Name: "bad.example.com.", Qtype: dns.TypeA, Qclass: dns.ClassINET}

	errBoom := errors.New("boom")

	qf := func(_ context.Context, q dns.Question, _ QueryFlags) (Response, error) {
		if q.Name == good.Name {
			return Response{Questions: []Question{{Name: q.Name, Type: "A", Class: "IN"}}}, nil
		}
		return Response{}, errBoom
	}

	responses, err := ConcurrentLookup(context.Background(), []dns.Question{good, bad}, QueryFlags{}, qf, discardLogger())
	if len(responses) != 1 {
		t.Fatalf("len(responses) = %d, want 1", len(responses))
	}
	if err == nil || !errors.Is(err, errBoom) {
		t.Fatalf("err = %v, want errors.Is(err, errBoom)", err)
	}
}

// TestConcurrentLookupWaitsForAllGoroutines makes sure we never short-circuit
// the wait — a slow goroutine still gets to record its result.
func TestConcurrentLookupWaitsForAllGoroutines(t *testing.T) {
	q1 := dns.Question{Name: "fast.example.com.", Qtype: dns.TypeA, Qclass: dns.ClassINET}
	q2 := dns.Question{Name: "slow.example.com.", Qtype: dns.TypeA, Qclass: dns.ClassINET}

	var slowFinished atomic.Bool
	qf := func(_ context.Context, q dns.Question, _ QueryFlags) (Response, error) {
		if q.Name == q2.Name {
			time.Sleep(50 * time.Millisecond)
			slowFinished.Store(true)
		}
		return Response{Questions: []Question{{Name: q.Name, Type: "A", Class: "IN"}}}, nil
	}

	responses, err := ConcurrentLookup(context.Background(), []dns.Question{q1, q2}, QueryFlags{}, qf, discardLogger())
	if err != nil {
		t.Fatalf("err = %v, want nil", err)
	}
	if len(responses) != 2 {
		t.Fatalf("len(responses) = %d, want 2", len(responses))
	}
	if !slowFinished.Load() {
		t.Fatal("ConcurrentLookup returned before slow goroutine finished")
	}
}

