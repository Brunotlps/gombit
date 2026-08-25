// Package nethttp is the plain net/http row of the BENCH-1 framework-tax
// microbenchmark matrix (issue #141): no router, no framework, hand-written
// JSON encode/decode and manual validation. This is the floor every other
// stack in benchmarks/micro/{gin,huma,gombit} is measured against.
package nethttp

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/gombit-dev/gombit/benchmarks/micro/scenario"
)

// NewHandler returns the net/http baseline carrying the same four scenarios
// (plaintext, JSON, path parameter, validated POST) as the other rows.
func NewHandler() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /plaintext", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = w.Write([]byte("Hello, World!"))
	})

	mux.HandleFunc("GET /json", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"message": "Hello, World!"})
	})

	mux.HandleFunc("GET /users/{id}", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, scenario.BenchUser{ID: r.PathValue("id"), Name: "Ada Lovelace"})
	})

	mux.HandleFunc("POST /users", func(w http.ResponseWriter, r *http.Request) {
		var body scenario.CreateBenchUserBody
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON"})
			return
		}
		if err := validate(body); err != "" {
			writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": err})
			return
		}
		writeJSON(w, http.StatusCreated, scenario.BenchUser{ID: "user-1", Name: body.Name, Email: body.Email})
	})

	return mux
}

func validate(body scenario.CreateBenchUserBody) string {
	if strings.TrimSpace(body.Name) == "" {
		return "name is required"
	}
	if strings.TrimSpace(body.Email) == "" || !strings.Contains(body.Email, "@") {
		return "email must be a valid address"
	}
	return ""
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
