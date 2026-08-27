package microbench

import (
	"bytes"
	"strings"
	"testing"
)

// fullOutput returns `go test -bench` text covering all five scenarios once.
func fullOutput() string {
	var b strings.Builder
	b.WriteString("goos: linux\npkg: .../gin\n")
	for _, s := range Scenarios {
		b.WriteString("BenchmarkFrameworkTax/" + s + "-16   100   2740 ns/op   512 B/op   8 allocs/op\n")
	}
	b.WriteString("PASS\nok\t.../gin\t3.2s\n")
	return b.String()
}

func TestParseBenchOutputAllScenarios(t *testing.T) {
	rows, err := ParseBenchOutput("gin", strings.NewReader(fullOutput()))
	if err != nil {
		t.Fatalf("ParseBenchOutput: %v", err)
	}
	if len(rows) != len(Scenarios) {
		t.Fatalf("rows = %d, want %d", len(rows), len(Scenarios))
	}
	for _, r := range rows {
		if r.Stack != "gin" || r.SchemaVersion != SchemaVersion {
			t.Errorf("bad row identity: %+v", r)
		}
	}
}

func TestParseBenchOutputKeepsEverySample(t *testing.T) {
	// One scenario with three ns/op samples + the other four once each.
	var b strings.Builder
	for _, ns := range []string{"3000", "2600", "2800"} {
		b.WriteString("BenchmarkFrameworkTax/valid-post-16   100   " + ns + " ns/op   500 B/op   8 allocs/op\n")
	}
	for _, s := range []string{"plaintext", "json", "path-param", "invalid-post"} {
		b.WriteString("BenchmarkFrameworkTax/" + s + "-16   100   1000 ns/op   100 B/op   2 allocs/op\n")
	}
	rows, err := ParseBenchOutput("huma", strings.NewReader(b.String()))
	if err != nil {
		t.Fatal(err)
	}
	var vp Row
	for _, r := range rows {
		if r.Scenario == "valid-post" {
			vp = r
		}
	}
	if len(vp.NsPerOp) != 3 {
		t.Fatalf("valid-post kept %d samples, want 3 (nothing thrown away)", len(vp.NsPerOp))
	}
	if vp.MedianNsPerOp() != 2800 { // median of 2600,2800,3000
		t.Errorf("median = %v, want 2800", vp.MedianNsPerOp())
	}
}

// At GOMAXPROCS=1 (go test -cpu=1 / a 1-CPU host) Go omits the -<N> suffix, so
// the hyphenated scenario names arrive bare. LastIndex("-") would truncate
// valid-post -> valid; stripProcs must keep them.
func TestParseBenchOutputUnsuffixedHyphenatedNames(t *testing.T) {
	var b strings.Builder
	for _, s := range Scenarios {
		b.WriteString("BenchmarkFrameworkTax/" + s + "   100   1500 ns/op   256 B/op   4 allocs/op\n")
	}
	rows, err := ParseBenchOutput("gombit", strings.NewReader(b.String()))
	if err != nil {
		t.Fatalf("unsuffixed names must parse: %v", err)
	}
	seen := map[string]bool{}
	for _, r := range rows {
		seen[r.Scenario] = true
	}
	for _, want := range []string{"path-param", "valid-post", "invalid-post"} {
		if !seen[want] {
			t.Errorf("hyphenated scenario %q was mangled by the suffix strip; got %v", want, seen)
		}
	}
}

func TestStripProcs(t *testing.T) {
	// A trailing -<digits> is dropped (the GOMAXPROCS suffix); a hyphen followed
	// by non-digits is part of the scenario name and kept.
	cases := map[string]string{
		"valid-post-16": "valid-post",
		"valid-post":    "valid-post",
		"path-param-8":  "path-param",
		"json-16":       "json",
		"json":          "json",
		"plaintext":     "plaintext",
	}
	for in, want := range cases {
		if got := stripProcs(in); got != want {
			t.Errorf("stripProcs(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestParseBenchOutputRejectsIncompleteRun(t *testing.T) {
	// A panic after the GET scenarios leaves valid-post/invalid-post missing.
	partial := "BenchmarkFrameworkTax/plaintext-16   1   1000 ns/op   1 B/op   1 allocs/op\n" +
		"BenchmarkFrameworkTax/json-16   1   2000 ns/op   2 B/op   2 allocs/op\npanic: boom\n"
	if _, err := ParseBenchOutput("gin", strings.NewReader(partial)); err == nil {
		t.Error("an incomplete run (missing scenarios) must be rejected, not published partial")
	}
}

func TestParseBenchOutputRejectsEmptyStack(t *testing.T) {
	if _, err := ParseBenchOutput("", strings.NewReader(fullOutput())); err == nil {
		t.Error("empty stack should error")
	}
}

func TestMergeReplacesWholeStack(t *testing.T) {
	existing := []Row{
		{Stack: "gin", Scenario: "valid-post", NsPerOp: []float64{2740}},
		{Stack: "gin", Scenario: "json", NsPerOp: []float64{900}}, // stale scenario from an old run
		{Stack: "huma", Scenario: "valid-post", NsPerOp: []float64{5000}},
	}
	// A fresh gin run reports only valid-post this time; the stale gin/json must go.
	incoming := []Row{{Stack: "gin", Scenario: "valid-post", NsPerOp: []float64{9999}}}
	merged := Merge(existing, incoming)

	stacks := map[string]int{}
	for _, r := range merged {
		stacks[r.Stack]++
	}
	if stacks["gin"] != 1 {
		t.Errorf("gin should be replaced wholesale (1 row), got %d", stacks["gin"])
	}
	if stacks["huma"] != 1 {
		t.Errorf("huma should be kept, got %d", stacks["huma"])
	}
}

func TestRelative(t *testing.T) {
	if got := Relative(5217, 804); got != "6.5×" {
		t.Errorf("Relative(5217,804) = %q, want 6.5×", got)
	}
	if got := Relative(804, 804); got != "1.0×" {
		t.Errorf("baseline should be 1.0×, got %q", got)
	}
}

func TestJSONRoundTrip(t *testing.T) {
	rows := []Row{
		{Stack: "huma", Scenario: "valid-post", NsPerOp: []float64{5000, 5100}},
		{Stack: "gin", Scenario: "plaintext", NsPerOp: []float64{1050}},
		{Stack: "gin", Scenario: "valid-post", NsPerOp: []float64{2740}},
	}
	var buf bytes.Buffer
	if err := WriteJSON(&buf, rows); err != nil {
		t.Fatal(err)
	}
	got, err := ReadJSON(&buf)
	if err != nil {
		t.Fatal(err)
	}
	// Sorted by (stack, scenario-in-Scenarios-order): gin/plaintext, gin/valid-post, huma/valid-post.
	if len(got) != 3 || got[0].Stack != "gin" || got[0].Scenario != "plaintext" {
		t.Fatalf("unexpected order: %+v", got)
	}
	if len(got[2].NsPerOp) != 2 {
		t.Errorf("samples not preserved through round-trip: %+v", got[2])
	}
}
