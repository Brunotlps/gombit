package summary

import (
	"bytes"
	"math"
	"strings"
	"testing"

	"github.com/gombit-dev/gombit/benchmarks/internal/result"
)

func row(framework string, concurrency, trial int, rps, p50, p95, p99 float64, errors int64) result.Result {
	return rowB(framework, "crud-list", concurrency, trial, rps, p50, p95, p99, errors)
}

func rowB(framework, benchmark string, concurrency, trial int, rps, p50, p95, p99 float64, errors int64) result.Result {
	return result.Result{
		SchemaVersion: result.SchemaVersion, Framework: framework, FrameworkVersion: "v1",
		Runtime: "go", Benchmark: benchmark, Database: "postgresql",
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
	// Both are crud-list, so within that benchmark gin-gorm sorts before gombit.
	if groups[0].Framework != "gin-gorm" || groups[1].Framework != "gombit" {
		t.Fatalf("group order = %s, %s; want gin-gorm, gombit", groups[0].Framework, groups[1].Framework)
	}

	single := groups[0]
	if single.Trials != 1 || single.RequestsPerSecond.Mean != 500 ||
		single.RequestsPerSecond.Median != 500 ||
		single.RequestsPerSecond.StdDev != 0 || single.RequestsPerSecond.CoV != 0 {
		t.Errorf("single-trial group = %+v; want mean/median 500, zero dispersion", single.RequestsPerSecond)
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

// TestSummarizeMedianDistinctFromMean pins the issue's published statistic: the
// median, not the mean, so a single noisy trial cannot become the headline. A
// skewed [100,200,900] group has median 200 but mean 400.
func TestSummarizeMedianDistinctFromMean(t *testing.T) {
	results := []result.Result{
		row("x", 100, 1, 100, 10, 20, 30, 0),
		row("x", 100, 2, 200, 20, 40, 60, 0),
		row("x", 100, 3, 900, 90, 180, 270, 0),
	}
	g := Summarize(results, DefaultCoVThreshold)[0]
	if math.Abs(g.RequestsPerSecond.Median-200) > 1e-9 {
		t.Errorf("median rps = %v, want 200 (not the mean 400)", g.RequestsPerSecond.Median)
	}
	if math.Abs(g.RequestsPerSecond.Mean-400) > 1e-9 {
		t.Errorf("mean rps = %v, want 400", g.RequestsPerSecond.Mean)
	}
	if math.Abs(g.LatencyP50.Median-20) > 1e-9 {
		t.Errorf("median p50 = %v, want 20", g.LatencyP50.Median)
	}

	var buf bytes.Buffer
	if err := WriteMarkdown(&buf, []Group{g}); err != nil {
		t.Fatalf("WriteMarkdown: %v", err)
	}
	s := buf.String()
	// The headline column is the median (200), never the mean (400).
	if !strings.Contains(s, "| x | 100 | 3 | 200 |") {
		t.Errorf("row should publish median rps 200:\n%s", s)
	}
	if strings.Contains(s, "| x | 100 | 3 | 400 |") {
		t.Errorf("row must not publish mean rps 400:\n%s", s)
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

// TestWriteMarkdownOneTablePerBenchmark is the finding-1 regression: with two
// workloads and two frameworks, each workload must be a single table that holds
// every framework. The old framework-primary sort produced one heading per
// (framework, benchmark) pair — four tables for two workloads — so this fails
// on that HEAD.
func TestWriteMarkdownOneTablePerBenchmark(t *testing.T) {
	results := []result.Result{
		rowB("gin-gorm", "crud-list", 10, 1, 500, 1, 2, 3, 0),
		rowB("gin-gorm", "techempower", 10, 1, 800, 1, 2, 3, 0),
		rowB("gombit", "crud-list", 10, 1, 480, 1, 2, 3, 0),
		rowB("gombit", "techempower", 10, 1, 790, 1, 2, 3, 0),
	}
	var buf bytes.Buffer
	if err := WriteMarkdown(&buf, Summarize(results, DefaultCoVThreshold)); err != nil {
		t.Fatalf("WriteMarkdown: %v", err)
	}
	s := buf.String()

	// Exactly one heading per workload — not one per (framework, benchmark).
	if n := strings.Count(s, "\n## "); n != 2 {
		t.Fatalf("want 2 benchmark headings (one per workload), got %d:\n%s", n, s)
	}
	if c := strings.Count(s, "## crud-list"); c != 1 {
		t.Errorf("## crud-list appears %d times, want 1:\n%s", c, s)
	}
	if c := strings.Count(s, "## techempower"); c != 1 {
		t.Errorf("## techempower appears %d times, want 1:\n%s", c, s)
	}

	// Each workload's section must contain both frameworks.
	for _, bench := range []string{"crud-list", "techempower"} {
		sec := section(s, bench)
		if !strings.Contains(sec, "| gin-gorm |") || !strings.Contains(sec, "| gombit |") {
			t.Errorf("%q table must hold every framework, got:\n%s", bench, sec)
		}
	}
}

// TestWriteMarkdownFlagPlacement pins ⚠ to the high-variance row and its absence
// from the low-variance one — an implementation that flags every row (or none)
// must fail.
func TestWriteMarkdownFlagPlacement(t *testing.T) {
	results := []result.Result{
		row("gombit", 100, 1, 100, 5, 10, 15, 0),
		row("gombit", 100, 2, 200, 7, 12, 18, 0), // huge variance -> flagged
		row("gin-gorm", 10, 1, 500, 1, 2, 3, 0),  // single trial -> not flagged
	}
	var buf bytes.Buffer
	if err := WriteMarkdown(&buf, Summarize(results, DefaultCoVThreshold)); err != nil {
		t.Fatalf("WriteMarkdown: %v", err)
	}
	s := buf.String()

	for _, want := range []string{
		"# Benchmark summary", "coordinated omission", "## crud-list",
		"| framework | concurrency | trials |",
	} {
		if !strings.Contains(s, want) {
			t.Errorf("markdown missing %q\n%s", want, s)
		}
	}

	hi := line(s, "| gombit | 100 |")
	if !strings.Contains(hi, "⚠") {
		t.Errorf("high-variance gombit row must be flagged, got %q", hi)
	}
	lo := line(s, "| gin-gorm | 10 |")
	if strings.Contains(lo, "⚠") {
		t.Errorf("low-variance gin-gorm row must not be flagged, got %q", lo)
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

// section returns the text of the "## <bench>" heading up to the next "## ".
func section(s, bench string) string {
	start := strings.Index(s, "## "+bench)
	if start < 0 {
		return ""
	}
	rest := s[start+len("## "+bench):]
	if next := strings.Index(rest, "\n## "); next >= 0 {
		return rest[:next]
	}
	return rest
}

// line returns the first line containing sub (without trailing newline).
func line(s, sub string) string {
	for _, l := range strings.Split(s, "\n") {
		if strings.Contains(l, sub) {
			return l
		}
	}
	return ""
}
