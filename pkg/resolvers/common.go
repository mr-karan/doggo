package resolvers

import (
	"context"
	"errors"
	"log/slog"
	"sync"

	"github.com/miekg/dns"
)

// QueryFunc represents the signature of a query function
type QueryFunc func(ctx context.Context, question dns.Question, flags QueryFlags) (Response, error)

// ConcurrentLookup performs concurrent DNS lookups across multiple questions
// against a single resolver. It always waits for all in-flight goroutines so
// completed work is never discarded when the context is cancelled or expires
// mid-flight; callers receive whatever responses finished plus a joined error
// describing any per-question failures.
func ConcurrentLookup(ctx context.Context, questions []dns.Question, flags QueryFlags, queryFunc QueryFunc, logger *slog.Logger) ([]Response, error) {
	var wg sync.WaitGroup
	responses := make([]Response, len(questions))
	errs := make([]error, len(questions))

	for i, q := range questions {
		wg.Add(1)
		go func(i int, q dns.Question) {
			defer wg.Done()
			if err := ctx.Err(); err != nil {
				errs[i] = err
				return
			}
			resp, err := queryFunc(ctx, q, flags)
			responses[i] = resp
			errs[i] = err
		}(i, q)
	}

	wg.Wait()

	var validResponses []Response
	var lookupErrs []error
	for i, resp := range responses {
		if errs[i] != nil {
			lookupErrs = append(lookupErrs, errs[i])
			if !errors.Is(errs[i], context.Canceled) && !errors.Is(errs[i], context.DeadlineExceeded) {
				logger.Error("error in lookup", "error", errs[i])
			}
			continue
		}
		validResponses = append(validResponses, resp)
	}

	if len(validResponses) == 0 && len(lookupErrs) > 0 {
		return nil, errors.Join(lookupErrs...)
	}

	return validResponses, errors.Join(lookupErrs...)
}
