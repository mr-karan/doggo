package main

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/knadh/koanf"
	"github.com/mr-karan/doggo/pkg/utils"
)

var (
	logger = utils.InitLogger()
	k      = koanf.New(".")
)

type resp struct {
	Status  string      `json:"status"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
}

func main() {

	r := chi.NewRouter()

	// Setup middlewares.
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		sendSuccessResponse("Welcome to Doggo DNS!", w)
		return
	})

	r.Get("/ping/", func(w http.ResponseWriter, r *http.Request) {
		sendSuccessResponse("PONG", w)
		return
	})

	r.Post("/lookup/", func(w http.ResponseWriter, r *http.Request) {
		return
	})

	http.ListenAndServe(":3000", r)
}

// sendResponse sends an HTTP success response.
func sendResponse(data interface{}, statusText string, status int, w http.ResponseWriter) {
	w.WriteHeader(status)
	w.Header().Set("Content-Type", "application/json; charset=utf-8")

	out, err := json.Marshal(resp{Status: statusText, Data: data})
	if err != nil {
		sendErrorResponse("Internal Server Error", http.StatusInternalServerError, nil, w)
		return
	}

	_, _ = w.Write(out)
}

// sendSuccessResponse sends an HTTP success (200 OK) response.
func sendSuccessResponse(data interface{}, w http.ResponseWriter) {
	sendResponse(data, "success", http.StatusOK, w)
}

// sendErrorResponse sends an HTTP error response.
func sendErrorResponse(message string, status int, data interface{}, w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)

	resp := resp{Status: "error",
		Message: message,
		Data:    data}

	out, _ := json.Marshal(resp)

	_, _ = w.Write(out)
}
