package reslimits

import (
	"strings"
	"testing"
)

// §7 budget for an app container.
var appBudget = Budget{CPUs: 2, MemoryBytes: 1 << 30} // 2 vCPU / 1 GiB

func TestClassifyEnforced(t *testing.T) {
	r := Classify("app", appBudget, Applied{NanoCPUs: 2_000_000_000, MemoryBytes: 1 << 30})
	if r.Status != Enforced {
		t.Fatalf("Status = %q, want %q\n%s", r.Status, Enforced, r)
	}
	if !strings.HasPrefix(r.String(), "enforced:") {
		t.Errorf("string should lead with enforced: %q", r.String())
	}
}

// The silent `docker compose up` case: both limits at Docker's 0 sentinel.
func TestClassifyNotApplied(t *testing.T) {
	r := Classify("app", appBudget, Applied{NanoCPUs: 0, MemoryBytes: 0})
	if r.Status != NotApplied {
		t.Fatalf("Status = %q, want %q", r.Status, NotApplied)
	}
	s := r.String()
	if !strings.HasPrefix(s, "not applied:") {
		t.Errorf("string should lead with 'not applied:': %q", s)
	}
	// The honest string must say the container is unlimited and point the
	// reader at what to check, without claiming a single prescribed fix.
	if !strings.Contains(s, "unlimited") {
		t.Errorf("string should say the container is unlimited: %q", s)
	}
	if !strings.Contains(s, "not enforced") {
		t.Errorf("explanation should state the limits were not enforced: %q", s)
	}
}

func TestClassifyPartialOneDimensionMissing(t *testing.T) {
	// CPU applied, memory left unlimited.
	r := Classify("app", appBudget, Applied{NanoCPUs: 2_000_000_000, MemoryBytes: 0})
	if r.Status != Partial {
		t.Fatalf("Status = %q, want %q", r.Status, Partial)
	}
	if !strings.Contains(r.String(), "partial:") {
		t.Errorf("string should lead with 'partial:': %q", r.String())
	}
}

func TestClassifyPartialWrongValueIsNotEnforced(t *testing.T) {
	// A limit applied but to the wrong value must NOT read as enforced — that
	// is the "silently pretending" failure the honest report exists to prevent.
	r := Classify("app", appBudget, Applied{NanoCPUs: 1_000_000_000, MemoryBytes: 512 << 20})
	if r.Status == Enforced {
		t.Fatalf("1 vCPU / 512 MiB against a 2 vCPU / 1 GiB budget must not be Enforced\n%s", r)
	}
	if r.Status != Partial {
		t.Errorf("Status = %q, want %q", r.Status, Partial)
	}
}

func TestClassifyCPUToleranceMatches(t *testing.T) {
	// NanoCpus rarely lands on an exact 2.0; a hair under must still match.
	r := Classify("app", appBudget, Applied{NanoCPUs: 1_999_999_500, MemoryBytes: 1 << 30})
	if r.Status != Enforced {
		t.Errorf("1.9999995 vCPU should match a 2 vCPU budget within tolerance: %q", r.Status)
	}
}

func TestParseInspect(t *testing.T) {
	// A trimmed but real-shaped `docker inspect` array.
	data := []byte(`[{"Id":"abc","HostConfig":{"NanoCpus":2000000000,"Memory":1073741824}}]`)
	got, err := ParseInspect(data)
	if err != nil {
		t.Fatalf("ParseInspect: %v", err)
	}
	if got.NanoCPUs != 2_000_000_000 || got.MemoryBytes != 1<<30 {
		t.Errorf("got %+v, want NanoCPUs=2e9, Memory=1GiB", got)
	}
}

func TestParseInspectUnlimited(t *testing.T) {
	// Fields present but 0 (the compose-ignored case) must parse, not error.
	data := []byte(`[{"HostConfig":{"NanoCpus":0,"Memory":0}}]`)
	got, err := ParseInspect(data)
	if err != nil {
		t.Fatalf("ParseInspect: %v", err)
	}
	if got != (Applied{}) {
		t.Errorf("got %+v, want zero (unlimited)", got)
	}
}

func TestParseInspectEmptyArrayErrors(t *testing.T) {
	if _, err := ParseInspect([]byte(`[]`)); err == nil {
		t.Error("empty inspect array should error, not report unlimited")
	}
}

func TestParseBytes(t *testing.T) {
	cases := map[string]int64{
		"1g":    1 << 30,
		"2g":    2 << 30,
		"512m":  512 << 20,
		"2048k": 2048 << 10,
		"1024":  1024,
		"1024b": 1024,
	}
	for in, want := range cases {
		got, err := ParseBytes(in)
		if err != nil {
			t.Errorf("ParseBytes(%q): %v", in, err)
			continue
		}
		if got != want {
			t.Errorf("ParseBytes(%q) = %d, want %d", in, got, want)
		}
	}
}

func TestParseBytesRejectsGarbage(t *testing.T) {
	for _, in := range []string{"", "abc", "1x", "-5m", "g"} {
		if _, err := ParseBytes(in); err == nil {
			t.Errorf("ParseBytes(%q) should error", in)
		}
	}
}

func TestFormatBytes(t *testing.T) {
	cases := map[int64]string{
		1 << 30:       "1 GiB",
		1 << 20:       "1 MiB",
		(3 << 30) / 2: "1.5 GiB",
		0:             "0",
	}
	for in, want := range cases {
		if got := FormatBytes(in); got != want {
			t.Errorf("FormatBytes(%d) = %q, want %q", in, got, want)
		}
	}
}
