package report

import (
	"strings"
	"testing"

	"github.com/gombit-dev/gombit/benchmarks/internal/footprint"
	"github.com/gombit-dev/gombit/benchmarks/internal/metadata"
	"github.com/gombit-dev/gombit/benchmarks/internal/result"
)

func crudRow(fw string, conc, trial int, rps float64) result.Result {
	return result.Result{
		Framework: fw, Benchmark: "crud-list", Concurrency: conc, Trial: trial,
		RequestsPerSecond: rps,
	}
}

func TestRenderCRUDPivotAndFootprint(t *testing.T) {
	results := []result.Result{
		crudRow("gin-gorm", 10, 1, 18000), crudRow("gin-gorm", 10, 2, 18200),
		crudRow("gin-gorm", 100, 1, 25000),
		crudRow("gombit", 10, 1, 12000), crudRow("gombit", 10, 2, 9000),
	}
	prints := []footprint.Footprint{
		{Framework: "gombit", Variant: footprint.VariantContainer, ColdStart: footprint.ColdStart{MedianMs: 120},
			IdleRSSBytes: 12 << 20, LoadedRSSBytes: 40 << 20, CPUPercentUnderLoad: 150, ImageSizeBytes: 30 << 20},
		{Framework: "gin-gorm", Variant: footprint.VariantContainer, ColdStart: footprint.ColdStart{MedianMs: 117},
			IdleRSSBytes: 8 << 20, LoadedRSSBytes: 19 << 20, CPUPercentUnderLoad: 140, ImageSizeBytes: 22 << 20},
		// An embedded row must not appear in the container footprint table.
		{Framework: "gombit", Variant: footprint.VariantEmbedded, BinarySizeBytes: 25 << 20},
	}
	dirty := false
	meta := metadata.Metadata{
		GitCommit: "abcdef1234567890", GitDirty: &dirty, Timestamp: "2026-08-27T00:00:00Z",
		OS: "linux", Arch: "amd64", CPUModel: "Test CPU", LogicalCPUs: 8, RAMBytes: 16 << 30,
		PostgresVersion: "postgres:16.4-alpine", ResourceLimits: "enforced: 2 vCPU / 1 GiB", BenchmarkTool: "grafana/k6:0.55.0",
	}

	out := Render(results, prints, meta)

	// CRUD table: median of gin-gorm@10 is (18000+18200)/2 = 18100; columns are
	// the union of concurrencies (10, 100), gombit missing @100 -> em dash.
	for _, want := range []string{
		"### PostgreSQL CRUD read", "| framework | 10 VUs | 100 VUs |",
		"| gin-gorm | 18100 | 25000 |", "| gombit | 10500 | — |",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("CRUD table missing %q\n%s", want, out)
		}
	}
	// Footprint table: sorted, embedded excluded, MB conversions.
	if !strings.Contains(out, "| gin-gorm | 117 | 8.0 | 19.0 | 140 | 22.0 |") {
		t.Errorf("footprint row wrong:\n%s", out)
	}
	if strings.Contains(out, "25.0 |") && strings.Contains(out, "embedded") {
		t.Errorf("embedded variant leaked into the report:\n%s", out)
	}
	// Methodology from metadata.
	for _, want := range []string{"Test CPU", "8 logical CPUs", "abcdef123456", "postgres:16.4-alpine", "methodology.md"} {
		if !strings.Contains(out, want) {
			t.Errorf("methodology missing %q\n%s", want, out)
		}
	}
}

func TestRenderEmptyDataIsHonest(t *testing.T) {
	out := Render(nil, nil, metadata.Metadata{})
	for _, want := range []string{
		"run `make benchmark-crud-all`", "run `make benchmark-footprint`", "metadata not yet recorded",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("empty render missing %q\n%s", want, out)
		}
	}
}

func TestReplaceBlockAndDrift(t *testing.T) {
	readme := "# App\n\n## Performance\n\n" + StartMarker + "\nOLD CONTENT\n" + EndMarker + "\n\n## Next\n"
	block := "NEW CONTENT\n"

	updated, err := ReplaceBlock(readme, block)
	if err != nil {
		t.Fatalf("ReplaceBlock: %v", err)
	}
	if !strings.Contains(updated, "NEW CONTENT") || strings.Contains(updated, "OLD CONTENT") {
		t.Errorf("block not replaced:\n%s", updated)
	}
	// Everything outside the markers is preserved.
	if !strings.Contains(updated, "## Performance") || !strings.Contains(updated, "## Next") {
		t.Errorf("content outside markers lost:\n%s", updated)
	}
	// Idempotent: replacing again yields the same, so drift is false against it.
	drift, err := CheckDrift(updated, block)
	if err != nil {
		t.Fatalf("CheckDrift: %v", err)
	}
	if !drift {
		t.Error("re-rendering the same block should report no drift")
	}
	// The original README (OLD CONTENT) does drift.
	drift, _ = CheckDrift(readme, block)
	if drift {
		t.Error("stale README should report drift")
	}
}

func TestReplaceBlockMissingMarkers(t *testing.T) {
	if _, err := ReplaceBlock("no markers here", "x"); err == nil {
		t.Error("missing markers should error")
	}
}
