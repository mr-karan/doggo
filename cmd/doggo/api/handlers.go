package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
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
	sendResponse(w, http.StatusOK, "Welcome to Doggo API.")
	return
}

func handleHealthCheck(w http.ResponseWriter, r *http.Request) {
	sendResponse(w, http.StatusOK, "PONG")
	return
}

func handleLookup(w http.ResponseWriter, r *http.Request) {
	var (
		app = r.Context().Value("app").(app.App)
	)

	// Read body.
	b, err := ioutil.ReadAll(r.Body)
	defer r.Body.Close()
	if err != nil {
		app.Logger.WithError(err).Error("error reading request body")
		sendErrorResponse(w, fmt.Sprintf("Invalid JSON payload"), http.StatusBadRequest, nil)
		return
	}
	// Prepare query flags.
	var qFlags models.QueryFlags
	if err := json.Unmarshal(b, &qFlags); err != nil {
		app.Logger.WithError(err).Error("error unmarshalling payload")
		sendErrorResponse(w, fmt.Sprintf("Invalid JSON payload"), http.StatusBadRequest, nil)
		return
	}

	app.QueryFlags = qFlags
	// Load fallbacks.
	app.LoadFallbacks()

	// Load Questions.
	app.PrepareQuestions()

	if len(app.Questions) == 0 {
		sendErrorResponse(w, fmt.Sprintf("Missing field `query`."), http.StatusBadRequest, nil)
		return
	}

	// Load Nameservers.
	err = app.LoadNameservers()
	if err != nil {
		app.Logger.WithError(err).Error("error loading nameservers")
		sendErrorResponse(w, fmt.Sprintf("Error lookuping up for records."), http.StatusInternalServerError, nil)
		return
	}

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
		app.Logger.WithError(err).Error("error loading resolver")
		sendErrorResponse(w, fmt.Sprintf("Error lookuping up for records."), http.StatusInternalServerError, nil)
		return
	}
	app.Resolvers = rslvrs

	var responses []resolvers.Response
	for _, q := range app.Questions {
		for _, rslv := range app.Resolvers {
			resp, err := rslv.Lookup(q)
			if err != nil {
				app.Logger.WithError(err).Error("error looking up DNS records")
				sendErrorResponse(w, fmt.Sprintf("Error lookuping up for records."), http.StatusInternalServerError, nil)
				return
			}
			responses = append(responses, resp)
		}
	}
	sendResponse(w, http.StatusOK, responses)
	return
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
