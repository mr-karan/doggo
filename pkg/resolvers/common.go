package resolvers

import (
	"log/slog"
	"sync"

	"github.com/miekg/dns"
)

// QueryFunc represents the signature of a query function
type QueryFunc func(question dns.Question, flags QueryFlags) (Response, error)

// ConcurrentLookup performs concurrent DNS lookups
func ConcurrentLookup(questions []dns.Question, flags QueryFlags, queryFunc QueryFunc, logger *slog.Logger) ([]Response, error) {
	var wg sync.WaitGroup
	responses := make([]Response, len(questions))
	errors := make([]error, len(questions))

	for i, q := range questions {
		wg.Add(1)
		go func(i int, q dns.Question) {
			defer wg.Done()
			resp, err := queryFunc(q, flags)
			responses[i] = resp
			errors[i] = err
		}(i, q)
	}

	wg.Wait()

	// Collect non-nil responses and handle errors
	var validResponses []Response
	for i, resp := range responses {
		if errors[i] != nil {
			logger.Error("error in lookup", "error", errors[i])
		} else {
			validResponses = append(validResponses, resp)
		}
	}

	return validResponses, nil
}
