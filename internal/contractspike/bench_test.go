// External test package: benchassert imports contractspike, so a test file
// declared as `package contractspike` (white-box) importing benchassert
// would create an import cycle for this test binary. Nothing here needs
// contractspike's unexported surface, so black-box works fine.
package contractspike_test

import (
	"net/http"
	"testing"

	"github.com/gombit-dev/gombit/internal/contractspike"
	"github.com/gombit-dev/gombit/internal/contractspike/benchassert"
)

// benchStacks returns the net/http -> Gin -> Huma+Gin rows of the
// framework-tax matrix (issue #141 "1. Go abstraction-overhead
// microbenchmarks"). The fourth row, Gombit, lives in the sibling
// internal/contractspike/gombitbench package — see
// contractspike.RegisterBenchRoutes for why it can't share this test binary.
func benchStacks(tb testing.TB) []benchassert.Stack {
	tb.Helper()

	return []benchassert.Stack{
		{Name: "net-http", Handler: contractspike.NewNetHTTPHandler(), Envelope: false},
		{Name: "gin", Handler: contractspike.NewBenchGinRouter(), Envelope: false},
		{Name: "huma-gin", Handler: contractspike.NewBenchHumaGinServer(), Envelope: true},
	}
}

// TestBenchStacksServeEquivalentScenarios checks that every stack in the
// framework-tax matrix implements the same five scenarios correctly before
// they're used for benchmarking: a benchmark over a broken handler is
// meaningless. See benchassert.Scenarios for what "correctly" means.
func TestBenchStacksServeEquivalentScenarios(t *testing.T) {
	for _, stack := range benchStacks(t) {
		t.Run(stack.Name, func(t *testing.T) {
			benchassert.Scenarios(t, stack)
		})
	}
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
		b.Run(stack.Name+"/plaintext", func(b *testing.B) {
			runBenchScenario(b, stack.Handler, http.MethodGet, "/plaintext", "")
		})
		b.Run(stack.Name+"/json", func(b *testing.B) {
			runBenchScenario(b, stack.Handler, http.MethodGet, "/json", "")
		})
		b.Run(stack.Name+"/path-param", func(b *testing.B) {
			runBenchScenario(b, stack.Handler, http.MethodGet, "/users/user-42", "")
		})
		b.Run(stack.Name+"/valid-post", func(b *testing.B) {
			runBenchScenario(b, stack.Handler, http.MethodPost, "/users", benchassert.ValidCreateUserBody)
		})
		b.Run(stack.Name+"/invalid-post", func(b *testing.B) {
			runBenchScenario(b, stack.Handler, http.MethodPost, "/users", benchassert.InvalidCreateUserBody)
		})
	}
}

func runBenchScenario(b *testing.B, handler http.Handler, method, path, body string) {
	b.Helper()
	b.ReportAllocs()

	for range b.N {
		response := benchassert.Do(handler, method, path, body)
		if response.Code >= 500 {
			b.Fatalf("%s %s status = %d, want < 500; body: %s", method, path, response.Code, response.Body.String())
		}
	}
}
