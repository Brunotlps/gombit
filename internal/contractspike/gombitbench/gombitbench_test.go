package gombitbench

import (
	"net/http"
	"testing"

	"github.com/gombit-dev/gombit/internal/contractspike/benchassert"
)

func gombitStack() benchassert.Stack {
	return benchassert.Stack{
		Name:     "gombit",
		Handler:  NewBenchGombitApp().Router(),
		Envelope: true,
	}
}

// TestGombitBenchScenarios checks the same five scenarios asserted for the
// net/http, Gin, and Huma+Gin stacks in internal/contractspike, applied to
// the Gombit runtime stack, before it's used for benchmarking. See
// benchassert.Scenarios for what "correctly" means.
func TestGombitBenchScenarios(t *testing.T) {
	benchassert.Scenarios(t, gombitStack())
}

// BenchmarkFrameworkTax reports the Gombit row of the framework-tax matrix.
// Run alongside internal/contractspike's net-http/gin/huma-gin rows with:
//
//	go test ./internal/contractspike/... -bench=BenchmarkFrameworkTax -benchmem -count=10
func BenchmarkFrameworkTax(b *testing.B) {
	stack := gombitStack()

	b.Run(stack.Name+"/plaintext", func(b *testing.B) {
		runScenario(b, stack.Handler, http.MethodGet, "/plaintext", "")
	})
	b.Run(stack.Name+"/json", func(b *testing.B) {
		runScenario(b, stack.Handler, http.MethodGet, "/json", "")
	})
	b.Run(stack.Name+"/path-param", func(b *testing.B) {
		runScenario(b, stack.Handler, http.MethodGet, "/users/user-42", "")
	})
	b.Run(stack.Name+"/valid-post", func(b *testing.B) {
		runScenario(b, stack.Handler, http.MethodPost, "/users", benchassert.ValidCreateUserBody)
	})
	b.Run(stack.Name+"/invalid-post", func(b *testing.B) {
		runScenario(b, stack.Handler, http.MethodPost, "/users", benchassert.InvalidCreateUserBody)
	})
}

func runScenario(b *testing.B, handler http.Handler, method, path, body string) {
	b.Helper()
	b.ReportAllocs()

	for range b.N {
		response := benchassert.Do(handler, method, path, body)
		if response.Code >= 500 {
			b.Fatalf("%s %s status = %d, want < 500; body: %s", method, path, response.Code, response.Body.String())
		}
	}
}
