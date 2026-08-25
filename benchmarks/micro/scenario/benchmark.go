package scenario

import (
	"net/http"
	"testing"
)

// RunBenchmark runs the five framework-tax benchmark scenarios (plaintext,
// JSON, path parameter, valid POST, invalid POST) against stack.Handler as
// b.Run sub-benchmarks, reporting ns/op, B/op, and allocs/op for each. It is
// shared by every row (benchmarks/micro/{nethttp,gin,huma,gombit}) so the
// sub-benchmark set and the request-timing loop can't drift between
// packages — each row's own BenchmarkFrameworkTax is just
// `scenario.RunBenchmark(b, stack())`.
func RunBenchmark(b *testing.B, stack Stack) {
	b.Helper()

	b.Run("plaintext", func(b *testing.B) {
		runScenario(b, stack.Handler, http.MethodGet, "/plaintext", "")
	})
	b.Run("json", func(b *testing.B) {
		runScenario(b, stack.Handler, http.MethodGet, "/json", "")
	})
	b.Run("path-param", func(b *testing.B) {
		runScenario(b, stack.Handler, http.MethodGet, "/users/user-42", "")
	})
	b.Run("valid-post", func(b *testing.B) {
		runScenario(b, stack.Handler, http.MethodPost, "/users", ValidCreateUserBody)
	})
	b.Run("invalid-post", func(b *testing.B) {
		runScenario(b, stack.Handler, http.MethodPost, "/users", InvalidCreateUserBody)
	})
}

func runScenario(b *testing.B, handler http.Handler, method, path, body string) {
	b.Helper()
	b.ReportAllocs()

	for range b.N {
		response := Do(handler, method, path, body)
		if response.Code >= 500 {
			b.Fatalf("%s %s status = %d, want < 500; body: %s", method, path, response.Code, response.Body.String())
		}
	}
}
