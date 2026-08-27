package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeFixture writes docker-inspect-shaped JSON to a temp file and returns it.
func writeFixture(t *testing.T, body string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "inspect.json")
	if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	return p
}

const (
	enforcedJSON  = `[{"HostConfig":{"NanoCpus":2000000000,"Memory":1073741824}}]`
	unlimitedJSON = `[{"HostConfig":{"NanoCpus":0,"Memory":0}}]`
)

func TestRunEnforcedExitsZero(t *testing.T) {
	f := writeFixture(t, enforcedJSON)
	var out, errBuf bytes.Buffer
	code := run([]string{"-container", "c", "-cpus", "2", "-memory", "1g", "-inspect-file", f}, &out, &errBuf)
	if code != 0 {
		t.Fatalf("exit = %d, want 0 (stderr: %s)", code, errBuf.String())
	}
	if !strings.HasPrefix(out.String(), "enforced:") {
		t.Errorf("stdout should lead with enforced: %q", out.String())
	}
}

// The reason -strict exists: a container running under the wrong (or no) limit
// must make the command fail-closed. An always-exit-0 main would pass without
// this test.
func TestRunStrictNotAppliedExitsNonZero(t *testing.T) {
	f := writeFixture(t, unlimitedJSON)
	var out, errBuf bytes.Buffer
	code := run([]string{"-container", "c", "-cpus", "2", "-memory", "1g", "-inspect-file", f, "-strict"}, &out, &errBuf)
	if code != 1 {
		t.Fatalf("exit = %d, want 1 for -strict on an unenforced budget", code)
	}
	if !strings.Contains(errBuf.String(), "not enforced") {
		t.Errorf("stderr should explain the non-zero exit: %q", errBuf.String())
	}
}

func TestRunNotAppliedWithoutStrictExitsZero(t *testing.T) {
	f := writeFixture(t, unlimitedJSON)
	var out, errBuf bytes.Buffer
	code := run([]string{"-container", "c", "-cpus", "2", "-memory", "1g", "-inspect-file", f}, &out, &errBuf)
	if code != 0 {
		t.Fatalf("exit = %d, want 0 without -strict", code)
	}
	if !strings.HasPrefix(out.String(), "not applied:") {
		t.Errorf("stdout should report 'not applied:' %q", out.String())
	}
}

func TestRunJSONFormat(t *testing.T) {
	f := writeFixture(t, enforcedJSON)
	var out, errBuf bytes.Buffer
	if code := run([]string{"-container", "c", "-cpus", "2", "-memory", "1g", "-inspect-file", f, "-format", "json"}, &out, &errBuf); code != 0 {
		t.Fatalf("exit = %d, want 0", code)
	}
	var report struct {
		Status string `json:"status"`
	}
	if err := json.Unmarshal(out.Bytes(), &report); err != nil {
		t.Fatalf("stdout is not valid JSON: %v\n%s", err, out.String())
	}
	if report.Status != "enforced" {
		t.Errorf("json status = %q, want enforced", report.Status)
	}
}

func TestRunUsageErrors(t *testing.T) {
	cases := map[string][]string{
		"missing container": {"-cpus", "2", "-memory", "1g"},
		"missing budget":    {"-container", "c"},
		"bad memory":        {"-container", "c", "-cpus", "2", "-memory", "12x"},
		"bad format":        {"-container", "c", "-cpus", "2", "-memory", "1g", "-format", "yaml", "-inspect-file", writeFixture(t, enforcedJSON)},
	}
	for name, args := range cases {
		var out, errBuf bytes.Buffer
		if code := run(args, &out, &errBuf); code != 2 {
			t.Errorf("%s: exit = %d, want 2", name, code)
		}
	}
}
