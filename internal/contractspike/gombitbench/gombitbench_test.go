package gombitbench

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

const (
	validCreateUserBody   = `{"name":"Ada Lovelace","email":"ada@example.com"}`
	invalidCreateUserBody = `{"name":"","email":"not-an-email"}`
)

// TestGombitBenchScenarios checks the same five scenarios asserted for the
// net/http, Gin, and Huma+Gin stacks in internal/contractspike, applied to
// the Gombit runtime stack, before it's used for benchmarking.
func TestGombitBenchScenarios(t *testing.T) {
	handler := NewBenchGombitApp().Router()

	t.Run("plaintext", func(t *testing.T) {
		response := doRequest(handler, http.MethodGet, "/plaintext", "")
		if response.Code != http.StatusOK {
			t.Fatalf("GET /plaintext status = %d, want %d", response.Code, http.StatusOK)
		}
		if got := response.Body.String(); got != "Hello, World!" {
			t.Fatalf("GET /plaintext body = %q, want %q", got, "Hello, World!")
		}
	})

	t.Run("json", func(t *testing.T) {
		response := doRequest(handler, http.MethodGet, "/json", "")
		if response.Code != http.StatusOK {
			t.Fatalf("GET /json status = %d, want %d; body: %s", response.Code, http.StatusOK, response.Body.String())
		}
		if !strings.Contains(response.Body.String(), "Hello, World!") {
			t.Fatalf("GET /json body = %s, want it to contain %q", response.Body.String(), "Hello, World!")
		}
	})

	t.Run("path-param", func(t *testing.T) {
		response := doRequest(handler, http.MethodGet, "/users/user-42", "")
		if response.Code != http.StatusOK {
			t.Fatalf("GET /users/user-42 status = %d, want %d; body: %s", response.Code, http.StatusOK, response.Body.String())
		}
		if !strings.Contains(response.Body.String(), "user-42") {
			t.Fatalf("GET /users/user-42 body = %s, want it to echo the path parameter", response.Body.String())
		}
	})

	t.Run("valid-post", func(t *testing.T) {
		response := doRequest(handler, http.MethodPost, "/users", validCreateUserBody)
		if response.Code != http.StatusOK && response.Code != http.StatusCreated {
			t.Fatalf("POST /users (valid) status = %d, want 200 or 201; body: %s", response.Code, response.Body.String())
		}
		if !strings.Contains(response.Body.String(), "Ada Lovelace") {
			t.Fatalf("POST /users (valid) body = %s, want it to echo the created user", response.Body.String())
		}
	})

	t.Run("invalid-post", func(t *testing.T) {
		response := doRequest(handler, http.MethodPost, "/users", invalidCreateUserBody)
		if response.Code < 400 || response.Code >= 500 {
			t.Fatalf("POST /users (invalid) status = %d, want a 4xx validation error; body: %s", response.Code, response.Body.String())
		}
	})
}

func doRequest(handler http.Handler, method, path, body string) *httptest.ResponseRecorder {
	var request *http.Request
	if body == "" {
		request = httptest.NewRequest(method, path, nil)
	} else {
		request = httptest.NewRequest(method, path, strings.NewReader(body))
		request.Header.Set("Content-Type", "application/json")
	}

	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	return response
}

// BenchmarkFrameworkTax reports the Gombit row of the framework-tax matrix.
// Run alongside internal/contractspike's net-http/gin/huma-gin rows with:
//
//	go test ./internal/contractspike/... -bench=BenchmarkFrameworkTax -benchmem -count=10
func BenchmarkFrameworkTax(b *testing.B) {
	handler := NewBenchGombitApp().Router()

	b.Run("gombit/plaintext", func(b *testing.B) {
		runScenario(b, handler, http.MethodGet, "/plaintext", "")
	})
	b.Run("gombit/json", func(b *testing.B) {
		runScenario(b, handler, http.MethodGet, "/json", "")
	})
	b.Run("gombit/path-param", func(b *testing.B) {
		runScenario(b, handler, http.MethodGet, "/users/user-42", "")
	})
	b.Run("gombit/valid-post", func(b *testing.B) {
		runScenario(b, handler, http.MethodPost, "/users", validCreateUserBody)
	})
	b.Run("gombit/invalid-post", func(b *testing.B) {
		runScenario(b, handler, http.MethodPost, "/users", invalidCreateUserBody)
	})
}

func runScenario(b *testing.B, handler http.Handler, method, path, body string) {
	b.Helper()
	b.ReportAllocs()

	for range b.N {
		response := doRequest(handler, method, path, body)
		if response.Code >= 500 {
			b.Fatalf("%s %s status = %d, want < 500; body: %s", method, path, response.Code, response.Body.String())
		}
	}
}
