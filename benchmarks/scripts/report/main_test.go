package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gombit-dev/gombit/benchmarks/internal/report"
)

func writeReadme(t *testing.T) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "README.md")
	body := "# App\n\n## Performance\n\n" + report.StartMarker + "\nstale\n" + report.EndMarker + "\n"
	if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestWriteThenCheckIsClean(t *testing.T) {
	readme := writeReadme(t)
	// Missing results/footprint/metadata files -> honest empty sections, still valid.
	miss := filepath.Join(t.TempDir(), "nope.json")

	var so, se bytes.Buffer
	if code := run([]string{"-readme", readme, "-results", miss, "-footprint", miss, "-metadata", miss}, &so, &se); code != 0 {
		t.Fatalf("write exit=%d stderr=%s", code, se.String())
	}
	got, _ := os.ReadFile(readme) //nolint:gosec // test reads a temp file it just wrote
	if strings.Contains(string(got), "stale") {
		t.Errorf("stale block not replaced:\n%s", got)
	}
	if !strings.Contains(string(got), "PostgreSQL CRUD read") {
		t.Errorf("regenerated block missing the CRUD section:\n%s", got)
	}

	// -check on the just-written README must pass.
	so.Reset()
	se.Reset()
	if code := run([]string{"-readme", readme, "-results", miss, "-footprint", miss, "-metadata", miss, "-check"}, &so, &se); code != 0 {
		t.Fatalf("check-after-write exit=%d stderr=%s", code, se.String())
	}
}

func TestCheckDetectsDrift(t *testing.T) {
	readme := writeReadme(t) // still holds "stale", never regenerated
	miss := filepath.Join(t.TempDir(), "nope.json")
	var so, se bytes.Buffer
	code := run([]string{"-readme", readme, "-results", miss, "-footprint", miss, "-metadata", miss, "-check"}, &so, &se)
	if code == 0 {
		t.Fatalf("check on a stale README should be non-zero; stderr=%s", se.String())
	}
}

func TestMissingMarkersFails(t *testing.T) {
	p := filepath.Join(t.TempDir(), "README.md")
	if err := os.WriteFile(p, []byte("# App\nno markers\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	miss := filepath.Join(t.TempDir(), "nope.json")
	var so, se bytes.Buffer
	if code := run([]string{"-readme", p, "-results", miss, "-footprint", miss, "-metadata", miss}, &so, &se); code == 0 {
		t.Error("README without markers should fail")
	}
}
