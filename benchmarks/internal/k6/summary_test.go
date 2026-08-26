package k6

import (
	"os"
	"strings"
	"testing"

	"github.com/gombit-dev/gombit/benchmarks/internal/result"
)

func parseGolden(t *testing.T, name string) Summary {
	t.Helper()
	f, err := os.Open("testdata/" + name) //nolint:gosec // fixed testdata golden name
	if err != nil {
		t.Fatalf("open golden: %v", err)
	}
	defer func() { _ = f.Close() }()
	s, err := ParseSummary(f)
	if err != nil {
		t.Fatalf("ParseSummary(%s): %v", name, err)
	}
	return s
}

// summary_ok.json is a real k6 dump from a clean 3s/10-VU run against
// gin-gorm: 1253 requests, http_req_failed passes:0 (none failed), checks
// fails:0, state.testRunDurationMs ~3001.
func TestParseCleanGolden(t *testing.T) {
	s := parseGolden(t, "summary_ok.json")

	if s.Requests != 1253 {
		t.Errorf("Requests = %d, want 1253", s.Requests)
	}
	if s.Errors != 0 {
		t.Errorf("Errors = %d, want 0 (an all-200 run has http_req_failed passes:0)", s.Errors)
	}
	if s.ChecksFailed != 0 {
		t.Errorf("ChecksFailed = %d, want 0", s.ChecksFailed)
	}
	if s.LatencyMs.P50 <= 0 || s.LatencyMs.P95 < s.LatencyMs.P50 || s.LatencyMs.P99 < s.LatencyMs.P95 {
		t.Errorf("latency percentiles not monotone/positive: %+v", s.LatencyMs)
	}
	// gracefulStop:0s -> elapsed is the ~3s window, not 0 and not the drain.
	if s.DurationSeconds < 2.9 || s.DurationSeconds > 3.2 {
		t.Errorf("DurationSeconds = %v, want ~3 (actual elapsed from state)", s.DurationSeconds)
	}
	if err := s.Validate(); err != nil {
		t.Errorf("Validate(clean golden) = %v, want nil", err)
	}
}

// summary_all_failed.json is a real k6 dump against an unreachable port: every
// request failed. This is the golden that pins the http_req_failed inversion —
// the FAILED count is `passes` (12279), not `fails` (0). A one-token change to
// read `fails` would make Errors 0 here and this test would catch it.
func TestParseAllFailedGolden(t *testing.T) {
	s := parseGolden(t, "summary_all_failed.json")

	if s.Requests <= 0 {
		t.Fatalf("Requests = %d, want > 0 (failed requests still count)", s.Requests)
	}
	if s.Errors != s.Requests {
		t.Errorf("Errors = %d, want %d (every request failed -> http_req_failed passes == count)", s.Errors, s.Requests)
	}
	if err := s.Validate(); err == nil {
		t.Error("Validate(all-failed golden) = nil; want a rejection")
	}
}

func TestParseSummaryRejectsGarbage(t *testing.T) {
	if _, err := ParseSummary(strings.NewReader("not json")); err == nil {
		t.Error("ParseSummary(garbage) = nil error; want a decode error")
	}
}

func TestValidate(t *testing.T) {
	valid := Summary{Requests: 1000, RequestsPerSecond: 100}
	if err := valid.Validate(); err != nil {
		t.Errorf("Validate(clean) = %v, want nil", err)
	}
	cases := map[string]Summary{
		"no traffic (unreachable)": {Requests: 0},
		"all requests failed":      {Requests: 11379, Errors: 11379},
		"some requests failed":     {Requests: 1000, Errors: 3},
		"content checks failed":    {Requests: 1000, ChecksFailed: 5},
	}
	for name, s := range cases {
		if err := s.Validate(); err == nil {
			t.Errorf("Validate(%s) = nil; want a rejection", name)
		}
	}
}

func TestMergeKeepsIdentitySetsMeasuredAndElapsedDuration(t *testing.T) {
	base := result.Result{
		SchemaVersion: result.SchemaVersion, Framework: "gin-gorm", FrameworkVersion: "v1.11.0",
		Runtime: "go", Benchmark: "crud-list", Database: "postgresql",
		Concurrency: 100, Trial: 2, DurationSeconds: 30, // the flag; Merge overwrites with elapsed
	}
	s := Summary{Requests: 999, RequestsPerSecond: 33.3, LatencyMs: result.Latency{P50: 1, P95: 2, P99: 3}, Errors: 0, DurationSeconds: 30.05}

	got := s.Merge(base)

	if got.Framework != "gin-gorm" || got.Concurrency != 100 || got.Trial != 2 {
		t.Errorf("Merge clobbered identity fields: %+v", got)
	}
	if got.Requests != 999 || got.RequestsPerSecond != 33.3 || got.LatencyMs.P99 != 3 {
		t.Errorf("Merge did not set measured fields: %+v", got)
	}
	// duration_seconds becomes the actual elapsed window, not the 30 flag.
	if got.DurationSeconds != 30.05 {
		t.Errorf("DurationSeconds = %v, want the measured 30.05, not the flag", got.DurationSeconds)
	}
}
