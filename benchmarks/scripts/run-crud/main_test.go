package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/gombit-dev/gombit/benchmarks/internal/result"
)

func TestParseIntListRejectsInvalidToken(t *testing.T) {
	for _, in := range []string{"1,10,abc,100", "1,,10", "100o"} {
		if got, err := parseIntList(in); err == nil {
			t.Errorf("parseIntList(%q) = %v, nil; want an error", in, got)
		}
	}
}

func TestMergeRowsReplacesSameFrameworkKeepsOthers(t *testing.T) {
	existing := []result.Result{
		{Framework: "gin-gorm", Concurrency: 10, Trial: 1},
		{Framework: "gombit", Concurrency: 10, Trial: 1},    // other framework, kept
		{Framework: "gin-gorm", Concurrency: 100, Trial: 1}, // old gin-gorm rows, replaced
	}
	newRows := []result.Result{
		{Framework: "gin-gorm", Concurrency: 10, Trial: 1, Requests: 42},
	}

	merged := mergeRows(existing, newRows, "gin-gorm")

	var ginGorm, gombit int
	for _, r := range merged {
		switch r.Framework {
		case "gin-gorm":
			ginGorm++
			if r.Requests != 42 {
				t.Errorf("gin-gorm row not the new one: %+v", r)
			}
		case "gombit":
			gombit++
		}
	}
	if ginGorm != 1 {
		t.Errorf("gin-gorm rows = %d, want 1 (old ones replaced)", ginGorm)
	}
	if gombit != 1 {
		t.Errorf("gombit rows = %d, want 1 (other framework kept)", gombit)
	}
}

// A trial that fails Validate (here the real all-failed k6 golden) must make
// run() return an error AND leave no results.json — a failed implementation
// must not write a partial or bogus snapshot (issue #141 §10).
func TestRunFailsAndWritesNothingOnValidateFailure(t *testing.T) {
	dir := t.TempDir()
	allFailed, err := os.ReadFile(filepath.Join("..", "..", "internal", "k6", "testdata", "summary_all_failed.json")) //nolint:gosec // fixed testdata golden path
	if err != nil {
		t.Fatalf("read golden: %v", err)
	}

	// Injected k6: warm-up (empty summaryPath) is a no-op; the measured run
	// writes the all-failed golden.
	k6run := func(_ int, _ string, summaryPath string) error {
		if summaryPath == "" {
			return nil
		}
		return os.WriteFile(summaryPath, allFailed, 0o600) //nolint:gosec // summaryPath is under t.TempDir()
	}

	cfg := runConfig{
		targetURL: "http://unused", framework: "x",
		concurrency: []int{1}, duration: "1s", warmup: "1s", trials: 1,
		outDir: dir,
	}

	if err := run(cfg, k6run); err == nil {
		t.Fatal("run() = nil, want an error for an all-failed trial")
	}
	if _, statErr := os.Stat(filepath.Join(dir, "results.json")); !os.IsNotExist(statErr) {
		t.Errorf("results.json exists after a failed run; want nothing written (stat err: %v)", statErr)
	}
}

// A clean injected run writes the snapshot, and a second framework's run merges
// rather than truncating.
func TestRunWritesAndAccumulatesAcrossFrameworks(t *testing.T) {
	dir := t.TempDir()
	ok, err := os.ReadFile(filepath.Join("..", "..", "internal", "k6", "testdata", "summary_ok.json")) //nolint:gosec // fixed testdata golden path
	if err != nil {
		t.Fatalf("read golden: %v", err)
	}
	k6run := func(_ int, _ string, summaryPath string) error {
		if summaryPath == "" {
			return nil
		}
		return os.WriteFile(summaryPath, ok, 0o600) //nolint:gosec // summaryPath is under t.TempDir()
	}

	for _, fw := range []string{"gin-gorm", "gombit"} {
		cfg := runConfig{
			targetURL: "http://unused", framework: fw, concurrency: []int{10},
			duration: "1s", warmup: "1s", trials: 1, outDir: dir,
		}
		if err := run(cfg, k6run); err != nil {
			t.Fatalf("run(%s) = %v", fw, err)
		}
	}

	f, err := os.Open(filepath.Join(dir, "results.json")) //nolint:gosec // dir is t.TempDir()
	if err != nil {
		t.Fatalf("open results.json: %v", err)
	}
	defer func() { _ = f.Close() }()
	rows, err := result.ReadJSON(f)
	if err != nil {
		t.Fatalf("ReadJSON: %v", err)
	}
	frameworks := map[string]bool{}
	for _, r := range rows {
		frameworks[r.Framework] = true
	}
	if !frameworks["gin-gorm"] || !frameworks["gombit"] {
		t.Errorf("second run truncated the first: frameworks = %v", frameworks)
	}
}
