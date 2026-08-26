// Package k6 parses the machine-readable summary that benchmarks/workloads/
// crud-list.js writes (via k6's handleSummary) into the load-generator-derived
// fields of a benchmark result. Keeping this a pure decode step — no k6
// process, no HTTP — lets the mapping be unit-tested against a captured
// summary fixture.
package k6

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/gombit-dev/gombit/benchmarks/internal/result"
)

// Summary is the shape crud-list.js's handleSummary emits: only the fields a
// load generator can measure (throughput, tail latency, errors). The
// orchestrator fills in the identity/config fields (framework, concurrency,
// trial, ...) around it.
type Summary struct {
	Requests          int64          `json:"requests"`
	RequestsPerSecond float64        `json:"requests_per_second"`
	LatencyMs         result.Latency `json:"latency_ms"`
	Errors            int64          `json:"errors"`
	ChecksFailed      int64          `json:"checks_failed"`
}

// ParseSummary decodes one crud-list.js summary. Decoding only — validity is
// Validate's job, so a caller can inspect a failed run's numbers before
// rejecting it.
func ParseSummary(r io.Reader) (Summary, error) {
	var s Summary
	if err := json.NewDecoder(r).Decode(&s); err != nil {
		return Summary{}, fmt.Errorf("decode k6 summary: %w", err)
	}
	return s, nil
}

// Validate rejects a trial that isn't a clean measurement: no traffic at all
// (target unreachable / workload misconfigured), any HTTP error (the target
// erroring or unreachable — every request failing still yields requests > 0),
// or any failed content check (a 200 with the wrong page shape). The headline
// read workload against a healthy, seeded app must be error-free; the
// orchestrator turns a Validate failure into a loud benchmark failure rather
// than recording a bogus row (issue #141 §10 "a failed implementation must
// make the benchmark command fail clearly", §7 "instead of publishing bogus
// data").
func (s Summary) Validate() error {
	switch {
	case s.Requests <= 0:
		return fmt.Errorf("no traffic: %d requests (target unreachable or workload misconfigured)", s.Requests)
	case s.Errors > 0:
		return fmt.Errorf("%d of %d requests failed (target erroring or unreachable)", s.Errors, s.Requests)
	case s.ChecksFailed > 0:
		return fmt.Errorf("%d content checks failed (target returned a wrong or malformed page)", s.ChecksFailed)
	default:
		return nil
	}
}

// Merge copies the load-generator-measured fields of s onto base (which
// carries the identity/config fields the orchestrator set) and returns the
// completed result row.
func (s Summary) Merge(base result.Result) result.Result {
	base.Requests = s.Requests
	base.RequestsPerSecond = s.RequestsPerSecond
	base.LatencyMs = s.LatencyMs
	base.Errors = s.Errors
	return base
}
