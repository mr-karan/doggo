package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/mr-karan/doggo/internal/app"
	"github.com/mr-karan/doggo/pkg/models"
	"github.com/mr-karan/doggo/pkg/resolvers"
)

type httpResp struct {
	Status  string      `json:"status"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
}

func handleIndexAPI(w http.ResponseWriter, r *http.Request) {
	var (
		app = r.Context().Value("app").(app.App)
	)

	sendResponse(w, http.StatusOK, fmt.Sprintf("Welcome to Doggo API. Version: %s", app.Version))
}

func handleHealthCheck(w http.ResponseWriter, r *http.Request) {
	sendResponse(w, http.StatusOK, "PONG")
}

func handleLookup(w http.ResponseWriter, r *http.Request) {
	var (
		app = r.Context().Value("app").(app.App)
	)

	// Read body.
	b, err := io.ReadAll(r.Body)
	defer r.Body.Close()
	if err != nil {
		app.Logger.Error("error reading request body", "error", err)
		sendErrorResponse(w, "Invalid JSON payload", http.StatusBadRequest, nil)
		return
	}
	// Prepare query flags.
	var qFlags models.QueryFlags
	if err := json.Unmarshal(b, &qFlags); err != nil {
		app.Logger.Error("error unmarshalling payload", "error", err)
		sendErrorResponse(w, "Invalid JSON payload", http.StatusBadRequest, nil)
		return
	}

	app.QueryFlags = qFlags
	// Load fallbacks.
	app.LoadFallbacks()

	// Load Questions.
	app.PrepareQuestions()

	if len(app.Questions) == 0 {
		sendErrorResponse(w, "Missing field `query`.", http.StatusBadRequest, nil)
		return
	}

	// Load Nameservers.
	if err := app.LoadNameservers(); err != nil {
		app.Logger.Error("error loading nameservers", "error", err)
		sendErrorResponse(w, "Error looking up for records.", http.StatusInternalServerError, nil)
		return
	}

	app.Logger.Debug("Loaded nameservers", "nameservers", app.Nameservers)

	// Load Resolvers.
	rslvrs, err := resolvers.LoadResolvers(resolvers.Options{
		Nameservers: app.Nameservers,
		UseIPv4:     app.QueryFlags.UseIPv4,
		UseIPv6:     app.QueryFlags.UseIPv6,
		SearchList:  app.ResolverOpts.SearchList,
		Ndots:       app.ResolverOpts.Ndots,
		Timeout:     app.QueryFlags.Timeout * time.Second,
		Logger:      app.Logger,
	})
	if err != nil {
		app.Logger.Error("error loading resolver", "error", err)
		sendErrorResponse(w, "Error looking up for records.", http.StatusInternalServerError, nil)
		return
	}
	app.Resolvers = rslvrs

	app.Logger.Debug("Loaded resolvers", "resolvers", app.Resolvers)

	// Build query flags from the request payload
	queryFlags := resolvers.QueryFlags{
		AA: app.QueryFlags.AA,
		AD: app.QueryFlags.AD,
		CD: app.QueryFlags.CD,
		RD: app.QueryFlags.RD,
		Z:  app.QueryFlags.Z,
		DO: app.QueryFlags.DO,

		// EDNS0 options
		NSID:    app.QueryFlags.NSID,
		Cookie:  app.QueryFlags.Cookie,
		Padding: app.QueryFlags.Padding,
		EDE:     app.QueryFlags.EDE,
		ECS:     app.QueryFlags.ECS,
	}

	// Default to RD=true if not explicitly set
	if !app.QueryFlags.RD && !app.QueryFlags.AA && !app.QueryFlags.AD && !app.QueryFlags.CD && !app.QueryFlags.Z && !app.QueryFlags.DO {
		queryFlags.RD = true
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	var (
		wg           sync.WaitGroup
		mu           sync.Mutex
		allResponses []resolvers.Response
		allErrors    []error
	)

	for _, resolver := range app.Resolvers {
		wg.Add(1)
		go func(r resolvers.Resolver) {
			defer wg.Done()
			responses, err := r.Lookup(ctx, app.Questions, queryFlags)
			mu.Lock()
			if err != nil {
				allErrors = append(allErrors, err)
			} else {
				allResponses = append(allResponses, responses...)
			}
			mu.Unlock()
		}(resolver)
	}

	wg.Wait()

	if len(allErrors) > 0 {
		app.Logger.Error("errors looking up DNS records", "errors", allErrors)
		sendErrorResponse(w, "Error looking up for records.", http.StatusInternalServerError, nil)
		return
	}

	if len(allResponses) == 0 {
		sendErrorResponse(w, "No records found.", http.StatusNotFound, nil)
		return
	}

	sendResponse(w, http.StatusOK, allResponses)
}

// wrap is a middleware that wraps HTTP handlers and injects the "app" context.
func wrap(app app.App, next http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := context.WithValue(r.Context(), "app", app)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// sendResponse sends a JSON envelope to the HTTP response.
func sendResponse(w http.ResponseWriter, code int, data interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)

	out, err := json.Marshal(httpResp{Status: "success", Data: data})
	if err != nil {
		sendErrorResponse(w, "Internal Server Error", http.StatusInternalServerError, nil)
		return
	}

	w.Write(out)
}

// sendErrorResponse sends a JSON error envelope to the HTTP response.
func sendErrorResponse(w http.ResponseWriter, message string, code int, data interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)

	resp := httpResp{Status: "error",
		Message: message,
		Data:    data}
	out, _ := json.Marshal(resp)
	w.Write(out)
}
