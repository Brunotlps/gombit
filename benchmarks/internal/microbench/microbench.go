// Package microbench is the schema, `go test -bench` parser, and encoders for
// the framework-tax microbenchmark (issue #141 §13 A): the per-request
// abstraction cost of each layer — net/http → Gin → Huma → Gombit — across the
// five scenarios (plaintext, json, path-param, valid-post, invalid-post),
// reported as ns/op, B/op, and allocs/op.
//
// Each stack is a separate `go test` process (constructing framework.App
// mutates a process-global; see benchmarks/micro/gombit), so the parser tags
// rows with the stack the orchestrator ran, and rows merge by (stack,
// scenario). Keeping the parse + schema here (not in the shell) lets it be
// unit-tested against captured `go test -bench` output.
package microbench

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"
)

// SchemaVersion is bumped on any breaking change to the row shape (issue §9).
const SchemaVersion = 1

// benchPrefix is the sub-benchmark name the scenarios live under.
const benchPrefix = "BenchmarkFrameworkTax/"

// Row is one (stack, scenario) measurement.
type Row struct {
	SchemaVersion int     `json:"schema_version"`
	Stack         string  `json:"stack"`
	Scenario      string  `json:"scenario"`
	NsPerOp       float64 `json:"ns_per_op"`
	BytesPerOp    int64   `json:"bytes_per_op"`
	AllocsPerOp   int64   `json:"allocs_per_op"`
}

// Key identifies a row for merge/replace.
func (r Row) Key() string { return r.Stack + "\x00" + r.Scenario }

// ParseBenchOutput reads `go test -bench` text output for one stack and returns
// a row per BenchmarkFrameworkTax/<scenario> line. A run with -count>1 emits a
// line per iteration; the last one for each scenario wins (deterministic, and
// enough for a headline table — variance is not part of this microbenchmark's
// contract). Lines that are not benchmark result lines are ignored.
func ParseBenchOutput(stack string, r io.Reader) ([]Row, error) {
	if stack == "" {
		return nil, fmt.Errorf("stack is required")
	}
	byScenario := map[string]Row{}
	order := []string{}
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 0, 64*1024), 1<<20)
	for sc.Scan() {
		fields := strings.Fields(sc.Text())
		if len(fields) < 2 || !strings.HasPrefix(fields[0], benchPrefix) {
			continue
		}
		scenario := strings.TrimPrefix(fields[0], benchPrefix)
		// Drop the trailing -<GOMAXPROCS> Go appends to a benchmark name.
		if i := strings.LastIndex(scenario, "-"); i >= 0 {
			scenario = scenario[:i]
		}
		row := Row{SchemaVersion: SchemaVersion, Stack: stack, Scenario: scenario}
		if !parseMetrics(fields, &row) {
			continue // a benchmark line with no ns/op (e.g. a "--- FAIL" isn't one)
		}
		if _, seen := byScenario[scenario]; !seen {
			order = append(order, scenario)
		}
		byScenario[scenario] = row
	}
	if err := sc.Err(); err != nil {
		return nil, fmt.Errorf("read bench output: %w", err)
	}
	rows := make([]Row, 0, len(order))
	for _, s := range order {
		rows = append(rows, byScenario[s])
	}
	return rows, nil
}

// parseMetrics fills ns/op, B/op, allocs/op from the "<value> <unit>" pairs in a
// benchmark line. Returns false if ns/op is absent.
func parseMetrics(fields []string, row *Row) bool {
	gotNs := false
	for i := 1; i < len(fields); i++ {
		switch fields[i] {
		case "ns/op":
			if v, err := strconv.ParseFloat(fields[i-1], 64); err == nil {
				row.NsPerOp = v
				gotNs = true
			}
		case "B/op":
			if v, err := strconv.ParseInt(fields[i-1], 10, 64); err == nil {
				row.BytesPerOp = v
			}
		case "allocs/op":
			if v, err := strconv.ParseInt(fields[i-1], 10, 64); err == nil {
				row.AllocsPerOp = v
			}
		}
	}
	return gotNs
}

// Merge returns existing with the given (stack, scenario) rows replaced by
// incoming, keeping other rows, sorted deterministically.
func Merge(existing, incoming []Row) []Row {
	replaced := make(map[string]bool, len(incoming))
	for _, r := range incoming {
		replaced[r.Key()] = true
	}
	out := make([]Row, 0, len(existing)+len(incoming))
	for _, r := range existing {
		if !replaced[r.Key()] {
			out = append(out, r)
		}
	}
	out = append(out, incoming...)
	sortRows(out)
	return out
}

func sortRows(rows []Row) {
	sort.SliceStable(rows, func(i, j int) bool {
		if rows[i].Stack != rows[j].Stack {
			return rows[i].Stack < rows[j].Stack
		}
		return rows[i].Scenario < rows[j].Scenario
	})
}

// WriteJSON / ReadJSON are the canonical microbench.json encoders.
func WriteJSON(w io.Writer, rows []Row) error {
	sorted := append([]Row(nil), rows...)
	sortRows(sorted)
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(sorted)
}

func ReadJSON(r io.Reader) ([]Row, error) {
	var rows []Row
	if err := json.NewDecoder(r).Decode(&rows); err != nil {
		return nil, fmt.Errorf("decode microbench json: %w", err)
	}
	return rows, nil
}
