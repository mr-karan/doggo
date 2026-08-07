package main

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	appcore "github.com/mr-karan/doggo/internal/app"
)

func TestHandleLookupRejectsInvalidRecordType(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	application := appcore.New(logger, nil, "test")
	handler := wrap(application, handleLookup)

	req := httptest.NewRequest(http.MethodPost, "/api/lookup/", strings.NewReader(`{
		"query": ["example.test"],
		"type": ["TYPE0"]
	}`))
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d; body: %s", recorder.Code, http.StatusBadRequest, recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), "reserved and cannot be used in a question") {
		t.Fatalf("body missing record-type validation error: %s", recorder.Body.String())
	}
}
