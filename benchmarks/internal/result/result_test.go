package result

import (
	"bytes"
	"strconv"
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
	// WriteJSON emits rows in the canonical sorted order, so the round trip
	// comes back sorted regardless of input order.
	want := sortedCopy(sampleResults())
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

func TestWriteCSVHeader(t *testing.T) {
	var buf bytes.Buffer
	if err := WriteCSV(&buf, sampleResults()); err != nil {
		t.Fatalf("WriteCSV: %v", err)
	}
	header := strings.SplitN(strings.TrimSpace(buf.String()), "\n", 2)[0]
	if !strings.HasPrefix(header, "schema_version,framework,") {
		t.Errorf("csv header = %q", header)
	}
	if !strings.Contains(header, "latency_p50_ms,latency_p95_ms,latency_p99_ms") {
		t.Errorf("csv header missing flattened latency columns: %q", header)
	}
}

// sortFixture shares a framework and benchmark across rows and differs on the
// later sort keys (benchmark, concurrency, trial), so the full comparator is
// exercised — a comparator that only compared Framework would pass the old
// two-row test but fail here.
func sortFixture() []Result {
	r := func(framework, benchmark string, concurrency, trial int) Result {
		return Result{SchemaVersion: SchemaVersion, Framework: framework, Benchmark: benchmark, Concurrency: concurrency, Trial: trial}
	}
	// Deliberately unsorted input.
	return []Result{
		r("gombit", "crud-list", 100, 2),
		r("gombit", "crud-list", 100, 1),
		r("gombit", "crud-list", 10, 1),
		r("gombit", "auth", 100, 1),
		r("gin-gorm", "crud-list", 10, 1),
	}
}

func wantSortedKeys() [][4]any {
	return [][4]any{
		{"gin-gorm", "crud-list", 10, 1},
		{"gombit", "auth", 100, 1},
		{"gombit", "crud-list", 10, 1},
		{"gombit", "crud-list", 100, 1},
		{"gombit", "crud-list", 100, 2},
	}
}

func TestSortOrderIsFullKeyForBothEncoders(t *testing.T) {
	want := wantSortedKeys()

	// CSV rows, in order (skip the header).
	var csvBuf bytes.Buffer
	if err := WriteCSV(&csvBuf, sortFixture()); err != nil {
		t.Fatalf("WriteCSV: %v", err)
	}
	rows := strings.Split(strings.TrimSpace(csvBuf.String()), "\n")[1:]
	if len(rows) != len(want) {
		t.Fatalf("csv rows = %d, want %d", len(rows), len(want))
	}
	for i, w := range want {
		cols := strings.Split(rows[i], ",")
		// framework(col 1), benchmark(col 5), concurrency(col 7), trial(col 8)
		if cols[1] != w[0] || cols[5] != w[1] ||
			cols[7] != strconv.Itoa(w[2].(int)) || cols[8] != strconv.Itoa(w[3].(int)) {
			t.Errorf("csv row %d = %q, want key %v", i, rows[i], w)
		}
	}

	// The canonical JSON must be sorted the same way, not collection order.
	var jsonBuf bytes.Buffer
	if err := WriteJSON(&jsonBuf, sortFixture()); err != nil {
		t.Fatalf("WriteJSON: %v", err)
	}
	got, err := ReadJSON(&jsonBuf)
	if err != nil {
		t.Fatalf("ReadJSON: %v", err)
	}
	if len(got) != len(want) {
		t.Fatalf("json rows = %d, want %d", len(got), len(want))
	}
	for i, w := range want {
		if got[i].Framework != w[0] || got[i].Benchmark != w[1] ||
			got[i].Concurrency != w[2] || got[i].Trial != w[3] {
			t.Errorf("json row %d = %+v, want key %v", i, got[i], w)
		}
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
