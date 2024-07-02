package resolvers

import (
	"context"
	"log/slog"
	"sync"

	"github.com/miekg/dns"
)

// QueryFunc represents the signature of a query function
type QueryFunc func(ctx context.Context, question dns.Question, flags QueryFlags) (Response, error)

// ConcurrentLookup performs concurrent DNS lookups
func ConcurrentLookup(ctx context.Context, questions []dns.Question, flags QueryFlags, queryFunc QueryFunc, logger *slog.Logger) ([]Response, error) {
	var wg sync.WaitGroup
	responses := make([]Response, len(questions))
	errors := make([]error, len(questions))
	done := make(chan struct{})

	for i, q := range questions {
		wg.Add(1)
		go func(i int, q dns.Question) {
			defer wg.Done()
			select {
			case <-ctx.Done():
				errors[i] = ctx.Err()
			default:
				resp, err := queryFunc(ctx, q, flags)
				responses[i] = resp
				errors[i] = err
			}
		}(i, q)
	}

	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-done:
		// All goroutines have finished
	}

	// Collect non-nil responses and handle errors
	var validResponses []Response
	for i, resp := range responses {
		if errors[i] != nil {
			if errors[i] != context.Canceled && errors[i] != context.DeadlineExceeded {
				logger.Error("error in lookup", "error", errors[i])
			}
		} else {
			validResponses = append(validResponses, resp)
		}
	}

	if len(validResponses) == 0 && ctx.Err() != nil {
		return nil, ctx.Err()
	}

	return validResponses, nil
}
