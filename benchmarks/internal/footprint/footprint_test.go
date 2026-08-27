package footprint

import (
	"bytes"
	"math"
	"strings"
	"testing"
)

func TestComputeColdStart(t *testing.T) {
	// 1..9 ms: median 5, p95 = interpolate at rank 0.95*8=7.6 -> 8 + .6*(9-8)=8.6.
	cs := ComputeColdStart([]float64{5, 1, 9, 3, 7, 2, 8, 4, 6})
	if cs.Runs != 9 {
		t.Errorf("Runs = %d, want 9", cs.Runs)
	}
	if math.Abs(cs.MedianMs-5) > 1e-9 {
		t.Errorf("median = %v, want 5", cs.MedianMs)
	}
	if math.Abs(cs.P95Ms-8.6) > 1e-9 {
		t.Errorf("p95 = %v, want 8.6", cs.P95Ms)
	}
}

func TestComputeColdStartEdge(t *testing.T) {
	if got := ComputeColdStart(nil); got != (ColdStart{}) {
		t.Errorf("empty = %+v, want zero", got)
	}
	one := ComputeColdStart([]float64{42})
	if one.MedianMs != 42 || one.P95Ms != 42 || one.Runs != 1 {
		t.Errorf("single = %+v, want 42/42/1", one)
	}
}

func TestMergeReplacesByFrameworkAndVariant(t *testing.T) {
	existing := []Footprint{
		{Framework: "gin-gorm", Variant: VariantContainer, IdleRSSBytes: 10},
		{Framework: "gombit", Variant: VariantContainer, IdleRSSBytes: 20},
		{Framework: "gombit", Variant: VariantEmbedded, IdleRSSBytes: 30},
	}
	// Re-measure gombit's container row only.
	incoming := []Footprint{{Framework: "gombit", Variant: VariantContainer, IdleRSSBytes: 99}}
	merged := Merge(existing, incoming)

	if len(merged) != 3 {
		t.Fatalf("len = %d, want 3", len(merged))
	}
	byKey := map[string]int64{}
	for _, f := range merged {
		byKey[f.Key()] = f.IdleRSSBytes
	}
	if byKey["gombit\x00container"] != 99 {
		t.Errorf("gombit/container not replaced: %d", byKey["gombit\x00container"])
	}
	if byKey["gombit\x00embedded"] != 30 {
		t.Errorf("gombit/embedded should be kept: %d", byKey["gombit\x00embedded"])
	}
	if byKey["gin-gorm\x00container"] != 10 {
		t.Errorf("gin-gorm/container should be kept: %d", byKey["gin-gorm\x00container"])
	}
}

func TestJSONRoundTripAndDeterministicOrder(t *testing.T) {
	rows := []Footprint{
		{SchemaVersion: SchemaVersion, Framework: "gombit", Variant: VariantEmbedded, BinarySizeBytes: 25_000_000},
		{SchemaVersion: SchemaVersion, Framework: "gin-gorm", Variant: VariantContainer, IdleRSSBytes: 8_000_000},
		{SchemaVersion: SchemaVersion, Framework: "gombit", Variant: VariantContainer, IdleRSSBytes: 12_000_000},
	}
	var buf bytes.Buffer
	if err := WriteJSON(&buf, rows); err != nil {
		t.Fatalf("WriteJSON: %v", err)
	}
	encoded := buf.String() // ReadJSON drains buf, so keep the text for assertions
	got, err := ReadJSON(&buf)
	if err != nil {
		t.Fatalf("ReadJSON: %v", err)
	}
	// Sorted by (framework, variant): gin-gorm/container, gombit/container, gombit/embedded.
	wantOrder := []string{"gin-gorm\x00container", "gombit\x00container", "gombit\x00embedded"}
	if len(got) != len(wantOrder) {
		t.Fatalf("len = %d, want %d", len(got), len(wantOrder))
	}
	for i, k := range wantOrder {
		if got[i].Key() != k {
			t.Errorf("row %d = %q, want %q", i, got[i].Key(), k)
		}
	}
	// binary_size_bytes is omitted when zero, so it appears only in the one
	// embedded row (the two container rows omit it).
	if n := strings.Count(encoded, "binary_size_bytes"); n != 1 {
		t.Errorf("binary_size_bytes appears %d times, want 1 (only the embedded row):\n%s", n, encoded)
	}
}

func TestWriteCSV(t *testing.T) {
	var buf bytes.Buffer
	rows := []Footprint{{
		Framework: "gombit", FrameworkVersion: "v0.1.3", Runtime: "go", RuntimeVersion: "go1.25.7",
		Variant: VariantEmbedded, ColdStart: ColdStart{MedianMs: 12.5, P95Ms: 18, Runs: 20},
		IdleRSSBytes: 15_000_000, ImageSizeBytes: 30_000_000, BinarySizeBytes: 25_000_000,
	}}
	if err := WriteCSV(&buf, rows); err != nil {
		t.Fatalf("WriteCSV: %v", err)
	}
	out := buf.String()
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) != 2 {
		t.Fatalf("want header + 1 row, got %d lines:\n%s", len(lines), out)
	}
	if !strings.HasPrefix(lines[0], "framework,framework_version,") {
		t.Errorf("bad header: %s", lines[0])
	}
	for _, want := range []string{"gombit", "v0.1.3", "embedded", "12.5", "18", "20", "25000000"} {
		if !strings.Contains(lines[1], want) {
			t.Errorf("row missing %q: %s", want, lines[1])
		}
	}
}
