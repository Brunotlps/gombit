package summary

import (
	"bytes"
	"math"
	"strings"
	"testing"

	"github.com/gombit-dev/gombit/benchmarks/internal/result"
)

func row(framework string, concurrency, trial int, rps, p50, p95, p99 float64, errors int64) result.Result {
	return result.Result{
		SchemaVersion: result.SchemaVersion, Framework: framework, FrameworkVersion: "v1",
		Runtime: "go", Benchmark: "crud-list", Database: "postgresql",
		Concurrency: concurrency, Trial: trial,
		RequestsPerSecond: rps,
		LatencyMs:         result.Latency{P50: p50, P95: p95, P99: p99},
		Errors:            errors,
	}
}

func TestSummarizeGroupsAndStats(t *testing.T) {
	results := []result.Result{
		// gombit @100: two trials, rps 100 and 120 -> mean 110, sample stddev
		// sqrt(((100-110)^2+(120-110)^2)/1)=sqrt(200)=14.142..., CoV ~0.1286.
		row("gombit", 100, 1, 100, 5, 10, 15, 0),
		row("gombit", 100, 2, 120, 7, 12, 18, 0),
		// gin-gorm @10: single trial -> no dispersion.
		row("gin-gorm", 10, 1, 500, 1, 2, 3, 0),
	}

	groups := Summarize(results, DefaultCoVThreshold)
	if len(groups) != 2 {
		t.Fatalf("groups = %d, want 2", len(groups))
	}
	// Sorted: gin-gorm before gombit.
	if groups[0].Framework != "gin-gorm" || groups[1].Framework != "gombit" {
		t.Fatalf("group order = %s, %s; want gin-gorm, gombit", groups[0].Framework, groups[1].Framework)
	}

	single := groups[0]
	if single.Trials != 1 || single.RequestsPerSecond.Mean != 500 ||
		single.RequestsPerSecond.StdDev != 0 || single.RequestsPerSecond.CoV != 0 {
		t.Errorf("single-trial group = %+v; want mean 500, zero dispersion", single.RequestsPerSecond)
	}

	multi := groups[1]
	if multi.Trials != 2 {
		t.Errorf("multi.Trials = %d, want 2", multi.Trials)
	}
	if math.Abs(multi.RequestsPerSecond.Mean-110) > 1e-9 {
		t.Errorf("mean rps = %v, want 110", multi.RequestsPerSecond.Mean)
	}
	if math.Abs(multi.RequestsPerSecond.StdDev-math.Sqrt(200)) > 1e-9 {
		t.Errorf("stddev = %v, want sqrt(200)=%v", multi.RequestsPerSecond.StdDev, math.Sqrt(200))
	}
	wantCoV := math.Sqrt(200) / 110
	if math.Abs(multi.RequestsPerSecond.CoV-wantCoV) > 1e-9 {
		t.Errorf("CoV = %v, want %v", multi.RequestsPerSecond.CoV, wantCoV)
	}
	// ~12.9% CoV > 5% default -> flagged.
	if !multi.HighVariance {
		t.Errorf("multi group CoV %.3f should exceed the 5%% threshold", multi.RequestsPerSecond.CoV)
	}
}

func TestSummarizeErrorsSummedAndLowVarianceNotFlagged(t *testing.T) {
	results := []result.Result{
		row("x", 10, 1, 1000, 1, 2, 3, 2),
		row("x", 10, 2, 1010, 1, 2, 3, 5), // ~0.5% CoV, well under 5%
	}
	g := Summarize(results, DefaultCoVThreshold)[0]
	if g.Errors != 7 {
		t.Errorf("Errors = %d, want 7 (summed across trials)", g.Errors)
	}
	if g.HighVariance {
		t.Errorf("CoV %.4f is under 5%%; should not be flagged", g.RequestsPerSecond.CoV)
	}
}

func TestWriteMarkdownTablesAndFlag(t *testing.T) {
	results := []result.Result{
		row("gombit", 100, 1, 100, 5, 10, 15, 0),
		row("gombit", 100, 2, 200, 7, 12, 18, 0), // huge variance -> flagged
		row("gin-gorm", 10, 1, 500, 1, 2, 3, 0),
	}
	groups := Summarize(results, DefaultCoVThreshold)

	var buf bytes.Buffer
	if err := WriteMarkdown(&buf, groups); err != nil {
		t.Fatalf("WriteMarkdown: %v", err)
	}
	s := buf.String()

	for _, want := range []string{
		"# Benchmark summary",
		"coordinated omission",
		"## crud-list",
		"| framework | concurrency | trials |",
		"| gin-gorm | 10 | 1 |",
		"| gombit | 100 | 2 |",
	} {
		if !strings.Contains(s, want) {
			t.Errorf("markdown missing %q\n%s", want, s)
		}
	}
	// The high-variance gombit row is flagged; the gin-gorm single-trial row is not.
	if !strings.Contains(s, "⚠") {
		t.Errorf("expected a ⚠ flag on the high-variance row:\n%s", s)
	}
}

func TestWriteMarkdownEmpty(t *testing.T) {
	var buf bytes.Buffer
	if err := WriteMarkdown(&buf, nil); err != nil {
		t.Fatalf("WriteMarkdown(nil): %v", err)
	}
	if !strings.Contains(buf.String(), "# Benchmark summary") {
		t.Error("empty summary should still render the header")
	}
}
