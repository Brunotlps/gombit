// Package result defines the machine-readable benchmark result schema
// (issue #141 §9) that every benchmark run writes, and the JSON/CSV encoders
// the summarizer and report generator read back. Markdown is never the
// canonical data source — it is generated from this.
package result

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strconv"
)

// SchemaVersion is bumped when the Result shape changes incompatibly, so a
// stored results.json can be recognized (or rejected) by a later summarizer.
const SchemaVersion = 1

// Latency holds the tail-latency percentiles for a single trial, in
// milliseconds.
type Latency struct {
	P50 float64 `json:"p50"`
	P95 float64 `json:"p95"`
	P99 float64 `json:"p99"`
}

// Result is one row: one implementation, one workload, one concurrency
// level, one trial. Issue #141 §9's recommended schema, plus schema_version.
// One trial per Result — variance across trials is computed by the
// summarizer from the set of rows sharing a (Framework, Benchmark,
// Concurrency) key, not stored here.
type Result struct {
	SchemaVersion     int     `json:"schema_version"`
	Framework         string  `json:"framework"`
	FrameworkVersion  string  `json:"framework_version"`
	Runtime           string  `json:"runtime"`
	RuntimeVersion    string  `json:"runtime_version"`
	Benchmark         string  `json:"benchmark"`
	Database          string  `json:"database"`
	Concurrency       int     `json:"concurrency"`
	Trial             int     `json:"trial"`
	DurationSeconds   float64 `json:"duration_seconds"`
	Requests          int64   `json:"requests"`
	RequestsPerSecond float64 `json:"requests_per_second"`
	LatencyMs         Latency `json:"latency_ms"`
	Errors            int64   `json:"errors"`
	CPUPercent        float64 `json:"cpu_percent"`
	RSSBytes          int64   `json:"rss_bytes"`
}

// WriteJSON encodes results as a pretty-printed JSON array. This is the
// canonical machine-readable output; results.csv and the Markdown report are
// derived from it.
func WriteJSON(w io.Writer, results []Result) error {
	// Never emit `null` for an empty run — an empty array is the honest,
	// still-parseable representation.
	if results == nil {
		results = []Result{}
	}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(results)
}

// ReadJSON decodes a results.json produced by WriteJSON.
func ReadJSON(r io.Reader) ([]Result, error) {
	var results []Result
	if err := json.NewDecoder(r).Decode(&results); err != nil {
		return nil, fmt.Errorf("decode results json: %w", err)
	}
	return results, nil
}

// csvHeader is the flattened column order for results.csv; latency_ms is
// spread into p50/p95/p99 columns.
var csvHeader = []string{
	"schema_version",
	"framework",
	"framework_version",
	"runtime",
	"runtime_version",
	"benchmark",
	"database",
	"concurrency",
	"trial",
	"duration_seconds",
	"requests",
	"requests_per_second",
	"latency_p50_ms",
	"latency_p95_ms",
	"latency_p99_ms",
	"errors",
	"cpu_percent",
	"rss_bytes",
}

// WriteCSV writes results as CSV with a fixed header, sorted deterministically
// (framework, benchmark, concurrency, trial) so a regenerated file diffs
// cleanly regardless of the order rows were collected in.
func WriteCSV(w io.Writer, results []Result) error {
	sorted := make([]Result, len(results))
	copy(sorted, results)
	sort.SliceStable(sorted, func(i, j int) bool {
		a, b := sorted[i], sorted[j]
		switch {
		case a.Framework != b.Framework:
			return a.Framework < b.Framework
		case a.Benchmark != b.Benchmark:
			return a.Benchmark < b.Benchmark
		case a.Concurrency != b.Concurrency:
			return a.Concurrency < b.Concurrency
		default:
			return a.Trial < b.Trial
		}
	})

	cw := csv.NewWriter(w)
	if err := cw.Write(csvHeader); err != nil {
		return fmt.Errorf("write csv header: %w", err)
	}
	for _, r := range sorted {
		row := []string{
			strconv.Itoa(r.SchemaVersion),
			r.Framework,
			r.FrameworkVersion,
			r.Runtime,
			r.RuntimeVersion,
			r.Benchmark,
			r.Database,
			strconv.Itoa(r.Concurrency),
			strconv.Itoa(r.Trial),
			formatFloat(r.DurationSeconds),
			strconv.FormatInt(r.Requests, 10),
			formatFloat(r.RequestsPerSecond),
			formatFloat(r.LatencyMs.P50),
			formatFloat(r.LatencyMs.P95),
			formatFloat(r.LatencyMs.P99),
			strconv.FormatInt(r.Errors, 10),
			formatFloat(r.CPUPercent),
			strconv.FormatInt(r.RSSBytes, 10),
		}
		if err := cw.Write(row); err != nil {
			return fmt.Errorf("write csv row: %w", err)
		}
	}
	cw.Flush()
	return cw.Error()
}

// formatFloat renders a float without a trailing ".0" for integers and with
// no scientific notation, so the CSV stays human-diffable.
func formatFloat(f float64) string {
	return strconv.FormatFloat(f, 'f', -1, 64)
}
