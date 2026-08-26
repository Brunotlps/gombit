package result

import (
	"bytes"
	"strings"
	"testing"
)

func sampleResults() []Result {
	return []Result{
		{
			SchemaVersion: SchemaVersion, Framework: "gombit", FrameworkVersion: "v0.1.0",
			Runtime: "go", RuntimeVersion: "go1.25.7", Benchmark: "crud-list",
			Database: "postgresql", Concurrency: 100, Trial: 2, DurationSeconds: 30,
			Requests: 123456, RequestsPerSecond: 4115.2,
			LatencyMs: Latency{P50: 10.1, P95: 21.8, P99: 32.4},
			Errors:    0, CPUPercent: 55.5, RSSBytes: 41943040,
		},
		{
			SchemaVersion: SchemaVersion, Framework: "gin-gorm", FrameworkVersion: "v1.11.0",
			Runtime: "go", RuntimeVersion: "go1.25.7", Benchmark: "crud-list",
			Database: "postgresql", Concurrency: 100, Trial: 1, DurationSeconds: 30,
			Requests: 200000, RequestsPerSecond: 6666.6,
			LatencyMs: Latency{P50: 5, P95: 12, P99: 20},
		},
	}
}

func TestJSONRoundTrip(t *testing.T) {
	var buf bytes.Buffer
	if err := WriteJSON(&buf, sampleResults()); err != nil {
		t.Fatalf("WriteJSON: %v", err)
	}

	got, err := ReadJSON(&buf)
	if err != nil {
		t.Fatalf("ReadJSON: %v", err)
	}
	want := sampleResults()
	if len(got) != len(want) {
		t.Fatalf("round trip len = %d, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("row %d round trip = %+v, want %+v", i, got[i], want[i])
		}
	}
}

func TestJSONUsesSnakeCaseAndNestedLatency(t *testing.T) {
	var buf bytes.Buffer
	if err := WriteJSON(&buf, sampleResults()[:1]); err != nil {
		t.Fatalf("WriteJSON: %v", err)
	}
	s := buf.String()
	for _, want := range []string{
		`"requests_per_second": 4115.2`,
		`"latency_ms": {`,
		`"p95": 21.8`,
		`"schema_version": 1`,
	} {
		if !strings.Contains(s, want) {
			t.Errorf("json missing %q\n%s", want, s)
		}
	}
}

func TestWriteJSONEmptyIsArrayNotNull(t *testing.T) {
	var buf bytes.Buffer
	if err := WriteJSON(&buf, nil); err != nil {
		t.Fatalf("WriteJSON(nil): %v", err)
	}
	if got := strings.TrimSpace(buf.String()); got != "[]" {
		t.Errorf("WriteJSON(nil) = %q, want []", got)
	}
}

func TestWriteCSVHeaderAndSortOrder(t *testing.T) {
	var buf bytes.Buffer
	if err := WriteCSV(&buf, sampleResults()); err != nil {
		t.Fatalf("WriteCSV: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) != 3 {
		t.Fatalf("csv has %d lines, want 3 (header + 2 rows)", len(lines))
	}
	if !strings.HasPrefix(lines[0], "schema_version,framework,") {
		t.Errorf("csv header = %q", lines[0])
	}
	if !strings.Contains(lines[0], "latency_p50_ms,latency_p95_ms,latency_p99_ms") {
		t.Errorf("csv header missing flattened latency columns: %q", lines[0])
	}
	// Sorted by framework: gin-gorm before gombit, regardless of input order.
	if !strings.HasPrefix(lines[1], "1,gin-gorm,") {
		t.Errorf("first data row = %q, want gin-gorm first (deterministic sort)", lines[1])
	}
	if !strings.HasPrefix(lines[2], "1,gombit,") {
		t.Errorf("second data row = %q, want gombit second", lines[2])
	}
}

func TestFormatFloatNoTrailingZeroOrScientific(t *testing.T) {
	cases := map[float64]string{
		30:      "30",
		4115.2:  "4115.2",
		0:       "0",
		0.00001: "0.00001",
	}
	for in, want := range cases {
		if got := formatFloat(in); got != want {
			t.Errorf("formatFloat(%v) = %q, want %q", in, got, want)
		}
	}
}
