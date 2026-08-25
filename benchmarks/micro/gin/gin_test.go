package gin

import (
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
	scenario.RunBenchmark(b, stack())
}
