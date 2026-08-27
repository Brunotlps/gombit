package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gombit-dev/gombit/benchmarks/internal/microbench"
)

func TestRunAccumulatesStacks(t *testing.T) {
	out := filepath.Join(t.TempDir(), "microbench.json")

	feed := func(stack, ns string) int {
		in := strings.NewReader("BenchmarkFrameworkTax/json-16   100   " + ns + " ns/op   512 B/op   8 allocs/op\n")
		var so, se bytes.Buffer
		return run([]string{"-stack", stack, "-out", out}, in, &so, &se)
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
	if len(rows) != 2 {
		t.Fatalf("rows = %d, want 2 (both stacks accumulated)", len(rows))
	}
	byStack := map[string]float64{}
	for _, r := range rows {
		byStack[r.Stack] = r.NsPerOp
	}
	if byStack["nethttp"] != 900 || byStack["gombit"] != 3100 {
		t.Errorf("stacks not accumulated: %+v", byStack)
	}
}

func TestRunRejectsEmptyStackOrOutput(t *testing.T) {
	out := filepath.Join(t.TempDir(), "microbench.json")
	var so, se bytes.Buffer
	// missing -stack
	if code := run([]string{"-out", out}, strings.NewReader(""), &so, &se); code == 0 {
		t.Error("missing -stack should be non-zero")
	}
	// stack given but no benchmark rows on stdin
	if code := run([]string{"-stack", "gin", "-out", out}, strings.NewReader("noise\n"), &so, &se); code == 0 {
		t.Error("no rows should be non-zero (a broken bench run must not silently write nothing)")
	}
}
