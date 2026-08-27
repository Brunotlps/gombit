package main

import (
	"io"
	"os"
	"path/filepath"
	"testing"
)

// fakeDocker writes a shell script standing in for `docker`: it finds the
// `-v <dir>:/out` mount in its args and, if goldenAbs is non-empty, copies that
// k6 summary golden into <dir>/summary.json — exactly what a real k6 run would
// leave behind. An empty goldenAbs simulates a run that produced no summary.
func fakeDocker(t *testing.T, goldenAbs string) string {
	t.Helper()
	script := "#!/bin/sh\nd=\nfor a in \"$@\"; do case \"$a\" in *:/out) d=\"${a%:/out}\";; esac; done\n"
	if goldenAbs != "" {
		script += "cp '" + goldenAbs + "' \"$d/summary.json\"\n"
	}
	script += "exit 0\n"
	p := filepath.Join(t.TempDir(), "fakedocker.sh")
	if err := os.WriteFile(p, []byte(script), 0o700); err != nil { //nolint:gosec // test helper script must be executable
		t.Fatal(err)
	}
	return p
}

func golden(t *testing.T, name string) string {
	t.Helper()
	abs, err := filepath.Abs(filepath.Join("..", "..", "internal", "k6", "testdata", name))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(abs); err != nil {
		t.Fatalf("golden %s missing: %v", name, err)
	}
	return abs
}

func TestRunCleanLoadExitsZero(t *testing.T) {
	code := run([]string{
		"-target-url", "http://127.0.0.1:9/x",
		"-docker", fakeDocker(t, golden(t, "summary_ok.json")),
	}, io.Discard, io.Discard)
	if code != 0 {
		t.Fatalf("clean load exit = %d, want 0", code)
	}
}

// The whole point of k6load: a run whose traffic all failed must NOT be blessed
// as a clean measurement. Deleting summary.Validate() would fail this test.
func TestRunFailedLoadExitsNonZero(t *testing.T) {
	code := run([]string{
		"-target-url", "http://127.0.0.1:9/x",
		"-docker", fakeDocker(t, golden(t, "summary_all_failed.json")),
	}, io.Discard, io.Discard)
	if code == 0 {
		t.Fatal("all-failed load exit = 0, want non-zero (Validate must reject error traffic)")
	}
}

func TestRunNoSummaryExitsNonZero(t *testing.T) {
	// fakeDocker with no golden writes no summary.json.
	code := run([]string{
		"-target-url", "http://127.0.0.1:9/x",
		"-docker", fakeDocker(t, ""),
	}, io.Discard, io.Discard)
	if code == 0 {
		t.Fatal("no-summary run exit = 0, want non-zero")
	}
}

func TestRunRequiresTarget(t *testing.T) {
	if code := run([]string{"-docker", fakeDocker(t, "")}, io.Discard, io.Discard); code != 2 {
		t.Fatalf("missing -target-url exit = %d, want 2", code)
	}
}
