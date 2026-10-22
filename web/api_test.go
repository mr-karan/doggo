package main

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mr-karan/doggo/internal/app"
)

func TestResponseEncodingFailureReturnsServerError(t *testing.T) {
	for _, test := range []struct {
		name string
		send func(http.ResponseWriter)
	}{
		{"success envelope", func(w http.ResponseWriter) { sendResponse(w, http.StatusOK, make(chan int)) }},
		{"error envelope", func(w http.ResponseWriter) { sendErrorResponse(w, "invalid", http.StatusBadRequest, make(chan int)) }},
	} {
		t.Run(test.name, func(t *testing.T) {
			response := httptest.NewRecorder()
			test.send(response)
			if response.Code != http.StatusInternalServerError {
				t.Fatalf("status = %d, want 500", response.Code)
			}
			var envelope httpResp
			if err := json.Unmarshal(response.Body.Bytes(), &envelope); err != nil {
				t.Fatalf("invalid JSON response: %v", err)
			}
			if envelope.Status != "error" || envelope.Message != "Internal Server Error" {
				t.Fatalf("unexpected error envelope: %+v", envelope)
			}
		})
	}
}

func TestRouterPreservesPeerAddress(t *testing.T) {
	for _, header := range []string{"X-Forwarded-For", "X-Real-IP", "True-Client-IP"} {
		t.Run(header, func(t *testing.T) {
			application := app.New(slog.New(slog.NewTextHandler(io.Discard, nil)), nil, "test")
			router := newRouter(application)
			router.Get("/peer", func(w http.ResponseWriter, r *http.Request) {
				_, _ = io.WriteString(w, r.RemoteAddr)
			})
			request := httptest.NewRequest(http.MethodGet, "/peer", nil)
			request.RemoteAddr = "192.0.2.10:12345"
			request.Header.Set(header, "198.51.100.99")
			response := httptest.NewRecorder()
			router.ServeHTTP(response, request)
			if got := response.Body.String(); got != "192.0.2.10:12345" {
				t.Fatalf("peer address = %q, want direct peer 192.0.2.10:12345", got)
			}
		})
	}
}
