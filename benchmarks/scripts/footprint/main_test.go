package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gombit-dev/gombit/benchmarks/internal/footprint"
)

func TestRunMergesContainerAndEmbeddedRows(t *testing.T) {
	out := filepath.Join(t.TempDir(), "footprint.json")
	var so, se bytes.Buffer

	// First a container row.
	code := run([]string{
		"-framework", "gombit", "-framework-version", "v0.1.3",
		"-runtime", "go", "-runtime-version", "go1.25.7",
		"-variant", "container", "-cold-start-ms", "200,220,240",
		"-idle-rss-bytes", "18000000", "-out", out,
	}, &so, &se)
	if code != 0 {
		t.Fatalf("container run exit=%d stderr=%s", code, se.String())
	}
	// Then an embedded row for the same framework — must accumulate, not replace.
	code = run([]string{
		"-framework", "gombit", "-variant", "embedded",
		"-cold-start-ms", "150", "-binary-size-bytes", "25000000", "-out", out,
	}, &so, &se)
	if code != 0 {
		t.Fatalf("embedded run exit=%d stderr=%s", code, se.String())
	}

	f, err := os.Open(out) //nolint:gosec // test reads a temp file it just wrote
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()
	rows, err := footprint.ReadJSON(f)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 2 {
		t.Fatalf("rows = %d, want 2 (container + embedded)", len(rows))
	}
	byVariant := map[string]footprint.Footprint{}
	for _, r := range rows {
		byVariant[r.Variant] = r
	}
	// median of 200,220,240 is 220.
	if byVariant["container"].ColdStart.MedianMs != 220 {
		t.Errorf("container cold-start median = %v, want 220", byVariant["container"].ColdStart.MedianMs)
	}
	if byVariant["embedded"].BinarySizeBytes != 25000000 {
		t.Errorf("embedded binary size = %d, want 25000000", byVariant["embedded"].BinarySizeBytes)
	}
	// The sibling CSV is written too.
	if _, err := os.Stat(strings.TrimSuffix(out, ".json") + ".csv"); err != nil {
		t.Errorf("sibling .csv not written: %v", err)
	}
}

func TestRunRejectsBadInput(t *testing.T) {
	out := filepath.Join(t.TempDir(), "footprint.json")
	cases := map[string][]string{
		"missing framework":   {"-variant", "container", "-out", out},
		"bad cold-start":      {"-framework", "x", "-variant", "container", "-cold-start-ms", "1,two,3", "-out", out},
		"negative cold-start": {"-framework", "x", "-variant", "container", "-cold-start-ms", "-5", "-out", out},
	}
	for name, args := range cases {
		var so, se bytes.Buffer
		if code := run(args, &so, &se); code == 0 {
			t.Errorf("%s: exit=0, want non-zero", name)
		}
	}
}
