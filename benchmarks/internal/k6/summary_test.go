package k6

import (
	"strings"
	"testing"

	"github.com/gombit-dev/gombit/benchmarks/internal/result"
)

// A real crud-list.js summary (captured shape).
const sampleSummary = `{
  "requests": 123456,
  "requests_per_second": 4115.2,
  "latency_ms": {"p50": 10.1, "p95": 21.8, "p99": 32.4},
  "errors": 0
}`

func TestParseSummary(t *testing.T) {
	s, err := ParseSummary(strings.NewReader(sampleSummary))
	if err != nil {
		t.Fatalf("ParseSummary: %v", err)
	}
	if s.Requests != 123456 || s.RequestsPerSecond != 4115.2 {
		t.Errorf("throughput = %d / %v", s.Requests, s.RequestsPerSecond)
	}
	if s.LatencyMs != (result.Latency{P50: 10.1, P95: 21.8, P99: 32.4}) {
		t.Errorf("latency = %+v", s.LatencyMs)
	}
	if s.Errors != 0 {
		t.Errorf("errors = %d", s.Errors)
	}
}

func TestParseSummaryRejectsGarbage(t *testing.T) {
	if _, err := ParseSummary(strings.NewReader("not json")); err == nil {
		t.Error("ParseSummary(garbage) = nil error; want a decode error")
	}
}

func TestValidate(t *testing.T) {
	valid := Summary{Requests: 1000, RequestsPerSecond: 100, Errors: 0, ChecksFailed: 0}
	if err := valid.Validate(); err != nil {
		t.Errorf("Validate(clean run) = %v, want nil", err)
	}

	cases := map[string]Summary{
		"no traffic (unreachable)":                   {Requests: 0},
		"all requests failed":                        {Requests: 11379, Errors: 11379},
		"some requests failed":                       {Requests: 1000, Errors: 3},
		"content checks failed (200 but wrong page)": {Requests: 1000, Errors: 0, ChecksFailed: 5},
	}
	for name, s := range cases {
		if err := s.Validate(); err == nil {
			t.Errorf("Validate(%s) = nil error; want a rejection", name)
		}
	}
}

func TestMergeKeepsIdentityFieldsAndSetsMeasured(t *testing.T) {
	base := result.Result{
		SchemaVersion: result.SchemaVersion, Framework: "gin-gorm", FrameworkVersion: "v1.11.0",
		Runtime: "go", Benchmark: "crud-list", Database: "postgresql",
		Concurrency: 100, Trial: 2, DurationSeconds: 30,
	}
	s := Summary{Requests: 999, RequestsPerSecond: 33.3, LatencyMs: result.Latency{P50: 1, P95: 2, P99: 3}, Errors: 4}

	got := s.Merge(base)

	// Identity/config fields survive.
	if got.Framework != "gin-gorm" || got.Concurrency != 100 || got.Trial != 2 || got.DurationSeconds != 30 {
		t.Errorf("Merge clobbered identity fields: %+v", got)
	}
	// Measured fields come from the summary.
	if got.Requests != 999 || got.RequestsPerSecond != 33.3 || got.Errors != 4 ||
		got.LatencyMs != (result.Latency{P50: 1, P95: 2, P99: 3}) {
		t.Errorf("Merge did not set measured fields: %+v", got)
	}
}
