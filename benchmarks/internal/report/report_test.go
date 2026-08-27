package report

import (
	"strings"
	"testing"

	"github.com/gombit-dev/gombit/benchmarks/internal/footprint"
	"github.com/gombit-dev/gombit/benchmarks/internal/metadata"
	"github.com/gombit-dev/gombit/benchmarks/internal/microbench"
	"github.com/gombit-dev/gombit/benchmarks/internal/result"
)

func crudRow(fw string, conc, trial int, rps, p50, p95, p99 float64) result.Result {
	return result.Result{
		Framework: fw, Benchmark: "crud-list", Concurrency: conc, Trial: trial,
		RequestsPerSecond: rps,
		LatencyMs:         result.Latency{P50: p50, P95: p95, P99: p99},
	}
}

func TestRenderCRUDCarriesTailsAndCoVFlag(t *testing.T) {
	results := []result.Result{
		// Both frameworks at the headline concurrency (100). gin-gorm is stable;
		// gombit's trials (12000, 9000) vary ~20% -> must be flagged.
		crudRow("gin-gorm", 100, 1, 25000, 1, 2, 3), crudRow("gin-gorm", 100, 2, 25200, 1, 2, 3),
		crudRow("gombit", 100, 1, 12000, 2, 5, 8), crudRow("gombit", 100, 2, 9000, 2, 5, 8),
		// A lower-concurrency row that must NOT be the one published.
		crudRow("gin-gorm", 10, 1, 18000, 1, 1, 1),
	}
	prints := []footprint.Footprint{
		{Framework: "gin-gorm", Variant: footprint.VariantContainer, ColdStart: footprint.ColdStart{MedianMs: 117},
			IdleRSSBytes: 8 << 20, LoadedRSSBytes: 19 << 20, CPUPercentUnderLoad: 140, ImageSizeBytes: 22 << 20},
		{Framework: "gombit", Variant: footprint.VariantEmbedded, BinarySizeBytes: 25 << 20}, // must not appear
	}
	dirty := false
	meta := metadata.Metadata{
		GitCommit: "abcdef1234567890", GitDirty: &dirty, Timestamp: "2026-08-27T00:00:00Z",
		OS: "linux", Arch: "amd64", CPUModel: "Test CPU", LogicalCPUs: 8, RAMBytes: 16 << 30,
		Kernel: "5.15-wsl", PostgresVersion: "postgres:16.4-alpine", ResourceLimits: "enforced",
		BenchmarkTool: "grafana/k6:0.55.0", Concurrency: []int{1, 10, 100}, Trials: 3,
		DurationSeconds: 10, WarmupSeconds: 3,
	}

	// All four rungs at the headline (valid-post) scenario, with samples so the
	// median is exercised; a wrong-scenario row must be excluded.
	micro := []microbench.Row{
		{Stack: "nethttp", Scenario: "valid-post", NsPerOp: []float64{820, 800}, BytesPerOp: 1425, AllocsPerOp: 14},
		{Stack: "gin", Scenario: "valid-post", NsPerOp: []float64{900}, BytesPerOp: 1458, AllocsPerOp: 15},
		{Stack: "huma", Scenario: "valid-post", NsPerOp: []float64{1078}, BytesPerOp: 1467, AllocsPerOp: 17},
		{Stack: "gombit", Scenario: "valid-post", NsPerOp: []float64{3040, 3160}, BytesPerOp: 4901, AllocsPerOp: 51},
		{Stack: "gin", Scenario: "json", NsPerOp: []float64{500}, BytesPerOp: 64, AllocsPerOp: 1}, // wrong scenario -> excluded
	}
	out := Render(results, prints, micro, meta)

	// Framework-tax table: validated-POST ladder, median ns/op, relative column.
	// net/http median (820,800)=810 baseline; gombit median (3040,3160)=3100 -> 3.8×.
	for _, want := range []string{
		"### Framework tax", "validated typed POST",
		"| stack | ns/op | B/op | allocs/op | vs net/http |",
		"| net/http | 810 | 1425 | 14 | 1.0× |",
		"| Gombit | 3100 | 4901 | 51 | 3.8× |",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("framework-tax table missing %q\n%s", want, out)
		}
	}
	if strings.Index(out, "| net/http |") > strings.Index(out, "| Gombit |") {
		t.Errorf("framework-tax rows not in ladder order:\n%s", out)
	}
	if strings.Contains(out, "| 500 |") {
		t.Errorf("wrong-scenario (json) row leaked into the framework-tax table:\n%s", out)
	}
	// CRUD table: headline concurrency, req/s + tails, ⚠ on the noisy row only.
	for _, want := range []string{
		"At **100 concurrent clients**",
		"| framework | req/s | p50 ms | p95 ms | p99 ms |",
		"| gin-gorm | 25100 | 1.0 | 2.0 | 3.0 |",
		"| gombit | 10500 ⚠ | 2.0 | 5.0 | 8.0 |",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("CRUD table missing %q\n%s", want, out)
		}
	}
	// The stable gin-gorm row must not be flagged.
	if line := lineWith(out, "| gin-gorm | 25100"); strings.Contains(line, "⚠") {
		t.Errorf("stable row should not be flagged: %q", line)
	}
	// Footprint table: container only, MB conversions, embedded excluded.
	if !strings.Contains(out, "| gin-gorm | 117 | 8.0 | 19.0 | 140 | 22.0 |") {
		t.Errorf("footprint row wrong:\n%s", out)
	}
	if strings.Contains(out, "embedded") {
		t.Errorf("embedded variant leaked into the report:\n%s", out)
	}
	// CPU caption is not a "lower is better" quality claim.
	if !strings.Contains(out, "not* a quality score") {
		t.Errorf("footprint caption should not call CPU 'lower is better':\n%s", out)
	}
	for _, want := range []string{
		"Test CPU", "8 logical CPUs", "kernel 5.15-wsl", "abcdef123456", "postgres:16.4-alpine",
		"methodology.md",
		// The actual protocol must be printed so a reduced run can't wear the canonical sweep.
		"concurrency 1/10/100 VUs, 3 trials × 10s each (warm-up 3s)",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("methodology missing %q\n%s", want, out)
		}
	}
}

func TestFrameworkTaxFlagsNoisyRung(t *testing.T) {
	// A non-stationary Gin series (the Huma<Gin inversion class) must be flagged.
	micro := []microbench.Row{
		{Stack: "nethttp", Scenario: "valid-post", NsPerOp: []float64{800, 810, 805}},
		{Stack: "gin", Scenario: "valid-post", NsPerOp: []float64{5538, 2589, 4479, 3210}}, // ~30% CoV
		{Stack: "huma", Scenario: "valid-post", NsPerOp: []float64{3450, 3460, 3455}},
		{Stack: "gombit", Scenario: "valid-post", NsPerOp: []float64{9500, 9550, 9520}},
	}
	out := Render(nil, nil, micro, metadata.Metadata{})
	gin := lineWith(out, "| Gin |")
	if !strings.Contains(gin, "⚠") {
		t.Errorf("noisy Gin rung should be flagged: %q", gin)
	}
	if !strings.Contains(out, "varied by more than 5%") {
		t.Errorf("noise caption missing:\n%s", out)
	}
	// A stable rung must not be flagged.
	if hu := lineWith(out, "| Huma + Gin |"); strings.Contains(hu, "⚠") {
		t.Errorf("stable Huma rung should not be flagged: %q", hu)
	}
}

func TestFrameworkTaxIncompleteLadderNotPublished(t *testing.T) {
	// Only three of the four rungs -> must render "Incomplete", not a holey table.
	micro := []microbench.Row{
		{Stack: "nethttp", Scenario: "valid-post", NsPerOp: []float64{800}},
		{Stack: "gin", Scenario: "valid-post", NsPerOp: []float64{900}},
		{Stack: "huma", Scenario: "valid-post", NsPerOp: []float64{1000}},
		// gombit missing
	}
	out := Render(nil, nil, micro, metadata.Metadata{})
	if !strings.Contains(out, "Incomplete") {
		t.Errorf("a missing rung should render Incomplete, not a partial ladder:\n%s", out)
	}
	if strings.Contains(out, "| stack | ns/op") {
		t.Errorf("no table should be published for an incomplete ladder:\n%s", out)
	}
}

func TestPickConcurrencyFallsBackToMax(t *testing.T) {
	// No headline (100) level -> the highest present (500) is published.
	results := []result.Result{
		crudRow("x", 10, 1, 100, 1, 1, 1),
		crudRow("x", 500, 1, 90, 2, 2, 2),
	}
	out := Render(results, nil, nil, metadata.Metadata{})
	if !strings.Contains(out, "At **500 concurrent clients**") {
		t.Errorf("expected fallback to the max concurrency:\n%s", out)
	}
}

func TestRenderEmptyDataIsHonest(t *testing.T) {
	out := Render(nil, nil, nil, metadata.Metadata{})
	for _, want := range []string{
		"### Framework tax", "run `make benchmark-micro`", "run `make benchmark-crud-all`", "run `make benchmark-footprint`",
		"metadata not yet recorded",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("empty render missing %q\n%s", want, out)
		}
	}
}

func TestReplaceBlockAndInSync(t *testing.T) {
	readme := "# App\n\n## Performance\n\n" + StartMarker + "\nOLD CONTENT\n" + EndMarker + "\n\n## Next\n"
	block := "NEW CONTENT\n"

	updated, err := ReplaceBlock(readme, block)
	if err != nil {
		t.Fatalf("ReplaceBlock: %v", err)
	}
	if !strings.Contains(updated, "NEW CONTENT") || strings.Contains(updated, "OLD CONTENT") {
		t.Errorf("block not replaced:\n%s", updated)
	}
	if !strings.Contains(updated, "## Performance") || !strings.Contains(updated, "## Next") {
		t.Errorf("content outside markers lost:\n%s", updated)
	}
	// InSync: true means "matches / no drift".
	sync, err := InSync(updated, block)
	if err != nil {
		t.Fatalf("InSync: %v", err)
	}
	if !sync {
		t.Error("re-rendering the same block should be in sync")
	}
	if sync, _ := InSync(readme, block); sync {
		t.Error("stale README should not be in sync")
	}
}

func TestReplaceBlockMissingMarkers(t *testing.T) {
	if _, err := ReplaceBlock("no markers here", "x"); err == nil {
		t.Error("missing markers should error")
	}
}

func lineWith(s, sub string) string {
	for _, l := range strings.Split(s, "\n") {
		if strings.Contains(l, sub) {
			return l
		}
	}
	return ""
}
