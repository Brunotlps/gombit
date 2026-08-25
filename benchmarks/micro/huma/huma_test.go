package huma

import (
	"testing"

	"github.com/gombit-dev/gombit/benchmarks/micro/scenario"
)

func stack() scenario.Stack {
	return scenario.Stack{Name: "huma-gin", Handler: NewServer(), Envelope: false}
}

// TestScenarios checks the bare Huma+Gin row implements the framework-tax
// scenarios correctly before it's used for benchmarking. See scenario.Assert
// for what "correctly" means.
func TestScenarios(t *testing.T) {
	scenario.Assert(t, stack())
}

// BenchmarkFrameworkTax reports the bare Huma+Gin row of the framework-tax
// matrix. Run alongside the net-http, gin, and gombit rows with:
//
//	go test ./benchmarks/micro/... -bench=BenchmarkFrameworkTax -benchmem -count=10
func BenchmarkFrameworkTax(b *testing.B) {
	scenario.RunBenchmark(b, stack())
}
