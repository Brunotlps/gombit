package contractspike

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// benchStack is one row of the framework-tax matrix: net/http -> Gin ->
// Huma+Gin (issue #141 "1. Go abstraction-overhead microbenchmarks"). The
// fourth row, Gombit, lives in the sibling internal/contractspike/gombitbench
// package — see RegisterBenchRoutes for why it can't share this test binary.
type benchStack struct {
	name    string
	handler http.Handler
}

func benchStacks(tb testing.TB) []benchStack {
	tb.Helper()

	return []benchStack{
		{name: "net-http", handler: NewNetHTTPHandler()},
		{name: "gin", handler: NewBenchGinRouter()},
		{name: "huma-gin", handler: NewBenchHumaGinServer()},
	}
}

const (
	validCreateUserBody   = `{"name":"Ada Lovelace","email":"ada@example.com"}`
	invalidCreateUserBody = `{"name":"","email":"not-an-email"}`
)

// TestBenchStacksServeEquivalentScenarios checks that every stack in the
// framework-tax matrix implements the same five scenarios correctly before
// they're used for benchmarking: a benchmark over a broken handler is
// meaningless.
func TestBenchStacksServeEquivalentScenarios(t *testing.T) {
	for _, stack := range benchStacks(t) {
		t.Run(stack.name, func(t *testing.T) {
			assertBenchPlaintext(t, stack)
			assertBenchJSON(t, stack)
			assertBenchGetUser(t, stack)
			assertBenchValidPost(t, stack)
			assertBenchInvalidPost(t, stack)
		})
	}
}

func assertBenchPlaintext(t *testing.T, stack benchStack) {
	t.Helper()

	response := doBenchRequest(stack.handler, http.MethodGet, "/plaintext", "")
	if response.Code != http.StatusOK {
		t.Fatalf("%s: GET /plaintext status = %d, want %d", stack.name, response.Code, http.StatusOK)
	}
	if got := response.Body.String(); got != "Hello, World!" {
		t.Fatalf("%s: GET /plaintext body = %q, want %q", stack.name, got, "Hello, World!")
	}
}

func assertBenchJSON(t *testing.T, stack benchStack) {
	t.Helper()

	response := doBenchRequest(stack.handler, http.MethodGet, "/json", "")
	if response.Code != http.StatusOK {
		t.Fatalf("%s: GET /json status = %d, want %d; body: %s", stack.name, response.Code, http.StatusOK, response.Body.String())
	}
	if !strings.Contains(response.Body.String(), "Hello, World!") {
		t.Fatalf("%s: GET /json body = %s, want it to contain %q", stack.name, response.Body.String(), "Hello, World!")
	}
}

func assertBenchGetUser(t *testing.T, stack benchStack) {
	t.Helper()

	response := doBenchRequest(stack.handler, http.MethodGet, "/users/user-42", "")
	if response.Code != http.StatusOK {
		t.Fatalf("%s: GET /users/user-42 status = %d, want %d; body: %s", stack.name, response.Code, http.StatusOK, response.Body.String())
	}
	if !strings.Contains(response.Body.String(), "user-42") {
		t.Fatalf("%s: GET /users/user-42 body = %s, want it to echo the path parameter", stack.name, response.Body.String())
	}
}

func assertBenchValidPost(t *testing.T, stack benchStack) {
	t.Helper()

	response := doBenchRequest(stack.handler, http.MethodPost, "/users", validCreateUserBody)
	if response.Code != http.StatusOK && response.Code != http.StatusCreated {
		t.Fatalf("%s: POST /users (valid) status = %d, want 200 or 201; body: %s", stack.name, response.Code, response.Body.String())
	}
	if !strings.Contains(response.Body.String(), "Ada Lovelace") {
		t.Fatalf("%s: POST /users (valid) body = %s, want it to echo the created user", stack.name, response.Body.String())
	}
}

// assertBenchInvalidPost only checks that every stack rejects the payload
// with a client error. It deliberately does not assert a specific error
// envelope shape: contract.Install (M3-2) replaces Huma's package-level
// huma.NewError process-wide the first time any framework.App boots, so
// within one `go test` binary the bare Huma+Gin stack's error shape depends
// on whether a Gombit stack ran first. That's a real, documented Huma/Gombit
// interaction (see docs/spikes/M0-2_HUMA_GIN_SPIKE.md), not something this
// benchmark should paper over or make flaky assertions about.
func assertBenchInvalidPost(t *testing.T, stack benchStack) {
	t.Helper()

	response := doBenchRequest(stack.handler, http.MethodPost, "/users", invalidCreateUserBody)
	if response.Code < 400 || response.Code >= 500 {
		t.Fatalf("%s: POST /users (invalid) status = %d, want a 4xx validation error; body: %s", stack.name, response.Code, response.Body.String())
	}
}

func doBenchRequest(handler http.Handler, method, path, body string) *httptest.ResponseRecorder {
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

// -- Framework-tax microbenchmarks (issue #141 "1. Go abstraction-overhead
// microbenchmarks"). Run with:
//
//	go test ./internal/contractspike/... -bench=BenchmarkFrameworkTax -benchmem -count=10
//
// and summarize with benchstat. The Gombit row of the matrix is reported by
// the sibling gombitbench package's own BenchmarkFrameworkTax; the "..." glob
// runs both in one command. See docs/spikes/M0-2_HUMA_GIN_SPIKE.md for the
// historical two-stack (plain Gin vs Huma+Gin) M0 spike result this matrix
// supersedes as the canonical ongoing benchmark.

func BenchmarkFrameworkTax(b *testing.B) {
	for _, stack := range benchStacks(b) {
		stack := stack
		b.Run(stack.name+"/plaintext", func(b *testing.B) {
			runBenchScenario(b, stack.handler, http.MethodGet, "/plaintext", "")
		})
		b.Run(stack.name+"/json", func(b *testing.B) {
			runBenchScenario(b, stack.handler, http.MethodGet, "/json", "")
		})
		b.Run(stack.name+"/path-param", func(b *testing.B) {
			runBenchScenario(b, stack.handler, http.MethodGet, "/users/user-42", "")
		})
		b.Run(stack.name+"/valid-post", func(b *testing.B) {
			runBenchScenario(b, stack.handler, http.MethodPost, "/users", validCreateUserBody)
		})
		b.Run(stack.name+"/invalid-post", func(b *testing.B) {
			runBenchScenario(b, stack.handler, http.MethodPost, "/users", invalidCreateUserBody)
		})
	}
}

func runBenchScenario(b *testing.B, handler http.Handler, method, path, body string) {
	b.Helper()
	b.ReportAllocs()

	for range b.N {
		response := doBenchRequest(handler, method, path, body)
		if response.Code >= 500 {
			b.Fatalf("%s %s status = %d, want < 500; body: %s", method, path, response.Code, response.Body.String())
		}
	}
}
