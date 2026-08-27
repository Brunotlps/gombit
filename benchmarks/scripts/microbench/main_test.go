package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gombit-dev/gombit/benchmarks/internal/microbench"
)

// benchAll returns full `go test -bench` output (all five scenarios) with the
// given ns/op for valid-post.
func benchAll(validPostNs string) string {
	var b strings.Builder
	for _, s := range microbench.Scenarios {
		ns := "1000"
		if s == "valid-post" {
			ns = validPostNs
		}
		b.WriteString("BenchmarkFrameworkTax/" + s + "-16   100   " + ns + " ns/op   512 B/op   8 allocs/op\n")
	}
	return b.String()
}

func TestRunAccumulatesStacks(t *testing.T) {
	out := filepath.Join(t.TempDir(), "microbench.json")
	feed := func(stack, ns string) int {
		var so, se bytes.Buffer
		return run([]string{"-stack", stack, "-out", out}, strings.NewReader(benchAll(ns)), &so, &se)
	}
	if code := feed("nethttp", "900"); code != 0 {
		t.Fatalf("nethttp exit=%d", code)
	}
	if code := feed("gombit", "3100"); code != 0 {
		t.Fatalf("gombit exit=%d", code)
	}

	f, err := os.Open(out) //nolint:gosec // test reads a temp file it just wrote
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()
	rows, err := microbench.ReadJSON(f)
	if err != nil {
		t.Fatal(err)
	}
	// Two stacks × five scenarios.
	if len(rows) != 2*len(microbench.Scenarios) {
		t.Fatalf("rows = %d, want %d", len(rows), 2*len(microbench.Scenarios))
	}
	got := map[string]float64{}
	for _, r := range rows {
		if r.Scenario == "valid-post" {
			got[r.Stack] = r.MedianNsPerOp()
		}
	}
	if got["nethttp"] != 900 || got["gombit"] != 3100 {
		t.Errorf("stacks not accumulated: %+v", got)
	}
}

func TestRunRejectsEmptyStackOrIncompleteOutput(t *testing.T) {
	out := filepath.Join(t.TempDir(), "microbench.json")
	var so, se bytes.Buffer
	if code := run([]string{"-out", out}, strings.NewReader(""), &so, &se); code == 0 {
		t.Error("missing -stack should be non-zero")
	}
	// Only one scenario present -> incomplete run -> must fail (not write a partial file).
	partial := "BenchmarkFrameworkTax/json-16   1   1000 ns/op   1 B/op   1 allocs/op\n"
	if code := run([]string{"-stack", "gin", "-out", out}, strings.NewReader(partial), &so, &se); code == 0 {
		t.Error("an incomplete bench run must fail, not write a partial stack")
	}
	if _, err := os.Stat(out); err == nil {
		t.Error("a rejected run must not have written the output file")
	}
}
