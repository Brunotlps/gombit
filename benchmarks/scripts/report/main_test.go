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

// The CLI must actually decode -results and render real numbers, not ignore the
// flag and always emit placeholders.
func TestWriteRendersRealResults(t *testing.T) {
	dir := t.TempDir()
	readme := filepath.Join(dir, "README.md")
	if err := os.WriteFile(readme, []byte("## Performance\n\n"+report.StartMarker+"\n\n"+report.EndMarker+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	resultsPath := filepath.Join(dir, "results.json")
	if err := os.WriteFile(resultsPath, []byte(`[
	  {"framework":"gin-gorm","benchmark":"crud-list","concurrency":100,"trial":1,"requests_per_second":24680,
	   "latency_ms":{"p50":1.2,"p95":3.4,"p99":5.6}}
	]`), 0o600); err != nil {
		t.Fatal(err)
	}
	miss := filepath.Join(dir, "nope.json")

	var so, se bytes.Buffer
	if code := run([]string{"-readme", readme, "-results", resultsPath, "-footprint", miss, "-metadata", miss}, &so, &se); code != 0 {
		t.Fatalf("exit=%d stderr=%s", code, se.String())
	}
	got, _ := os.ReadFile(readme) //nolint:gosec // test reads a temp file it just wrote
	if !strings.Contains(string(got), "24680") || !strings.Contains(string(got), "At **100 concurrent clients**") {
		t.Errorf("CLI did not render the real results.json numbers:\n%s", got)
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
