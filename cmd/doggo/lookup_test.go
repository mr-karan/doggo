package main

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/miekg/dns"
	"github.com/mr-karan/doggo/internal/app"
	"github.com/mr-karan/doggo/pkg/resolvers"
)

type orderedTestResolver struct {
	address   string
	delay     time.Duration
	responses []resolvers.Response
	err       error
}

func (r orderedTestResolver) Address() string { return r.address }

func (r orderedTestResolver) Lookup(ctx context.Context, _ []dns.Question, _ resolvers.QueryFlags) ([]resolvers.Response, error) {
	timer := time.NewTimer(r.delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-timer.C:
		return r.responses, r.err
	}
}

func TestPerformLookupPreservesConfiguredResolverOrder(t *testing.T) {
	errFirst := errors.New("first resolver warning")
	errThird := errors.New("third resolver failure")
	response := func(name string) resolvers.Response {
		return resolvers.Response{Questions: []resolvers.Question{{Name: name}}}
	}

	application := &app.App{
		Resolvers: []resolvers.Resolver{
			orderedTestResolver{
				address:   "resolver-1",
				delay:     8 * time.Millisecond,
				responses: []resolvers.Response{response("resolver-1-question-1"), response("resolver-1-question-2")},
				err:       errFirst,
			},
			orderedTestResolver{
				address:   "resolver-2",
				responses: []resolvers.Response{response("resolver-2-question-1"), response("resolver-2-question-2")},
			},
			orderedTestResolver{
				address: "resolver-3",
				delay:   2 * time.Millisecond,
				err:     errThird,
			},
		},
	}
	cfg := &config{timeout: time.Second}

	wantResponses := []string{
		"resolver-1-question-1",
		"resolver-1-question-2",
		"resolver-2-question-1",
		"resolver-2-question-2",
	}
	wantErrorServers := []string{"resolver-1", "resolver-3"}

	for run := 0; run < 20; run++ {
		responses, responseErrors := performLookup(application, cfg)

		gotResponses := make([]string, 0, len(responses))
		for _, response := range responses {
			gotResponses = append(gotResponses, response.Questions[0].Name)
		}
		if !reflect.DeepEqual(gotResponses, wantResponses) {
			t.Fatalf("run %d response order = %v, want %v", run, gotResponses, wantResponses)
		}

		gotErrorServers := make([]string, 0, len(responseErrors))
		for _, responseErr := range responseErrors {
			var lookupErr *resolvers.LookupError
			if !errors.As(responseErr, &lookupErr) {
				t.Fatalf("run %d error %v is not a LookupError", run, responseErr)
			}
			gotErrorServers = append(gotErrorServers, lookupErr.Nameserver)
		}
		if !reflect.DeepEqual(gotErrorServers, wantErrorServers) {
			t.Fatalf("run %d error order = %v, want %v", run, gotErrorServers, wantErrorServers)
		}
	}
}
