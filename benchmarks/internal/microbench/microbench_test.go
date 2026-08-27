package microbench

import (
	"bytes"
	"strings"
	"testing"
)

const sampleOutput = `goos: linux
goarch: amd64
pkg: github.com/gombit-dev/gombit/benchmarks/micro/gin
BenchmarkFrameworkTax/plaintext-16         	 1000000	      1050 ns/op	     128 B/op	       2 allocs/op
BenchmarkFrameworkTax/json-16              	  500000	      2740 ns/op	     512 B/op	       8 allocs/op
BenchmarkFrameworkTax/path-param-16        	  400000	      3100 ns/op	     640 B/op	      10 allocs/op
PASS
ok  	github.com/gombit-dev/gombit/benchmarks/micro/gin	3.2s
`

func TestParseBenchOutput(t *testing.T) {
	rows, err := ParseBenchOutput("gin", strings.NewReader(sampleOutput))
	if err != nil {
		t.Fatalf("ParseBenchOutput: %v", err)
	}
	if len(rows) != 3 {
		t.Fatalf("rows = %d, want 3 (plaintext/json/path-param)", len(rows))
	}
	byScen := map[string]Row{}
	for _, r := range rows {
		byScen[r.Scenario] = r
		if r.Stack != "gin" {
			t.Errorf("row stack = %q, want gin", r.Stack)
		}
		if r.SchemaVersion != SchemaVersion {
			t.Errorf("row schema = %d, want %d", r.SchemaVersion, SchemaVersion)
		}
	}
	json := byScen["json"]
	if json.NsPerOp != 2740 || json.BytesPerOp != 512 || json.AllocsPerOp != 8 {
		t.Errorf("json row = %+v, want 2740 ns / 512 B / 8 allocs", json)
	}
}

func TestParseBenchOutputCountKeepsLast(t *testing.T) {
	// -count=2 emits two lines per scenario; the last wins.
	out := `BenchmarkFrameworkTax/json-16   100   3000 ns/op   500 B/op   8 allocs/op
BenchmarkFrameworkTax/json-16   100   2800 ns/op   500 B/op   8 allocs/op
`
	rows, err := ParseBenchOutput("huma", strings.NewReader(out))
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0].NsPerOp != 2800 {
		t.Errorf("rows = %+v, want a single json row at 2800 ns/op", rows)
	}
}

func TestParseBenchOutputRejectsEmptyStack(t *testing.T) {
	if _, err := ParseBenchOutput("", strings.NewReader(sampleOutput)); err == nil {
		t.Error("empty stack should error")
	}
}

func TestParseBenchOutputIgnoresNoise(t *testing.T) {
	rows, err := ParseBenchOutput("nethttp", strings.NewReader("no benchmarks here\n--- FAIL: something\n"))
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 0 {
		t.Errorf("rows = %d, want 0 for non-benchmark output", len(rows))
	}
}

func TestMergeReplacesByStackScenario(t *testing.T) {
	existing := []Row{
		{Stack: "gin", Scenario: "json", NsPerOp: 2740},
		{Stack: "huma", Scenario: "json", NsPerOp: 5000},
	}
	incoming := []Row{{Stack: "gin", Scenario: "json", NsPerOp: 9999}}
	merged := Merge(existing, incoming)
	if len(merged) != 2 {
		t.Fatalf("len = %d, want 2", len(merged))
	}
	got := map[string]float64{}
	for _, r := range merged {
		got[r.Key()] = r.NsPerOp
	}
	if got["gin\x00json"] != 9999 {
		t.Errorf("gin/json not replaced: %v", got["gin\x00json"])
	}
	if got["huma\x00json"] != 5000 {
		t.Errorf("huma/json should be kept: %v", got["huma\x00json"])
	}
}

func TestJSONRoundTripDeterministicOrder(t *testing.T) {
	rows := []Row{
		{Stack: "huma", Scenario: "json", NsPerOp: 5000},
		{Stack: "gin", Scenario: "plaintext", NsPerOp: 1050},
		{Stack: "gin", Scenario: "json", NsPerOp: 2740},
	}
	var buf bytes.Buffer
	if err := WriteJSON(&buf, rows); err != nil {
		t.Fatal(err)
	}
	got, err := ReadJSON(&buf)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"gin\x00json", "gin\x00plaintext", "huma\x00json"}
	if len(got) != 3 {
		t.Fatalf("len = %d, want 3", len(got))
	}
	for i, k := range want {
		if got[i].Key() != k {
			t.Errorf("row %d = %q, want %q", i, got[i].Key(), k)
		}
	}
}
