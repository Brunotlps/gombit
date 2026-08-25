package gin

import (
	"net/http"
	"testing"

	"github.com/gombit-dev/gombit/benchmarks/micro/scenario"
)

func stack() scenario.Stack {
	return scenario.Stack{Name: "gin", Handler: NewRouter(), Envelope: false}
}

// TestScenarios checks the Gin row implements the framework-tax scenarios
// correctly before it's used for benchmarking. See scenario.Assert for what
// "correctly" means.
func TestScenarios(t *testing.T) {
	scenario.Assert(t, stack())
}

// BenchmarkFrameworkTax reports the Gin row of the framework-tax matrix. Run
// alongside the net-http, huma, and gombit rows with:
//
//	go test ./benchmarks/micro/... -bench=BenchmarkFrameworkTax -benchmem -count=10
func BenchmarkFrameworkTax(b *testing.B) {
	handler := NewRouter()

	b.Run("plaintext", func(b *testing.B) {
		run(b, handler, http.MethodGet, "/plaintext", "")
	})
	b.Run("json", func(b *testing.B) {
		run(b, handler, http.MethodGet, "/json", "")
	})
	b.Run("path-param", func(b *testing.B) {
		run(b, handler, http.MethodGet, "/users/user-42", "")
	})
	b.Run("valid-post", func(b *testing.B) {
		run(b, handler, http.MethodPost, "/users", scenario.ValidCreateUserBody)
	})
	b.Run("invalid-post", func(b *testing.B) {
		run(b, handler, http.MethodPost, "/users", scenario.InvalidCreateUserBody)
	})
}

func run(b *testing.B, handler http.Handler, method, path, body string) {
	b.Helper()
	b.ReportAllocs()

	for range b.N {
		response := scenario.Do(handler, method, path, body)
		if response.Code >= 500 {
			b.Fatalf("%s %s status = %d, want < 500; body: %s", method, path, response.Code, response.Body.String())
		}
	}
}
