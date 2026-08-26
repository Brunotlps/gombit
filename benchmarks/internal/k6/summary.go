// Package k6 parses the raw k6 summary that benchmarks/workloads/crud-list.js
// dumps (via handleSummary) into the load-generator-derived fields of a
// benchmark result. The workload does no interpretation — it forwards k6's own
// metrics and state — so every mapping that used to be a footgun in
// JavaScript (the http_req_failed Rate whose FAILED count is `passes` not
// `fails`, the percentile keys, the elapsed run time) lives here and is
// unit-tested against captured k6 goldens (testdata/).
package k6

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/gombit-dev/gombit/benchmarks/internal/result"
)

// rawSummary is the shape of k6's `{metrics, state}` dump. Only the fields the
// benchmark records are declared; k6's numbers are JSON floats.
type rawSummary struct {
	Metrics struct {
		HTTPReqs struct {
			Values struct {
				Count float64 `json:"count"`
				Rate  float64 `json:"rate"`
			} `json:"values"`
		} `json:"http_reqs"`
		HTTPReqDuration struct {
			Values map[string]float64 `json:"values"`
		} `json:"http_req_duration"`
		// http_req_failed is a Rate: a request is "added" as true when it
		// failed, so `passes` is the count of FAILED requests and `fails` the
		// count of successful ones (verified against testdata: an all-200 run
		// has passes:0, an all-refused run has passes==count).
		HTTPReqFailed struct {
			Values struct {
				Passes float64 `json:"passes"`
			} `json:"values"`
		} `json:"http_req_failed"`
		// checks is a Rate over individual check evaluations: `fails` is the
		// count of FAILED checks (a 200 with the wrong page shape).
		Checks struct {
			Values struct {
				Fails float64 `json:"fails"`
			} `json:"values"`
		} `json:"checks"`
	} `json:"metrics"`
	State struct {
		TestRunDurationMs float64 `json:"testRunDurationMs"`
	} `json:"state"`
}

// Summary is the interpreted, load-generator-measured half of a result row.
// The orchestrator fills in the identity/config fields (framework,
// concurrency, trial, ...) around it via Merge. DurationSeconds is the ACTUAL
// elapsed measured window (from k6's state), not the requested flag.
type Summary struct {
	Requests          int64
	RequestsPerSecond float64
	LatencyMs         result.Latency
	Errors            int64
	ChecksFailed      int64
	DurationSeconds   float64
}

// ParseSummary decodes and interprets one raw k6 summary. Decoding/mapping
// only — validity is Validate's job, so a caller can inspect a failed run's
// numbers before rejecting it.
func ParseSummary(r io.Reader) (Summary, error) {
	var raw rawSummary
	if err := json.NewDecoder(r).Decode(&raw); err != nil {
		return Summary{}, fmt.Errorf("decode k6 summary: %w", err)
	}
	d := raw.Metrics.HTTPReqDuration.Values
	return Summary{
		Requests:          int64(raw.Metrics.HTTPReqs.Values.Count),
		RequestsPerSecond: raw.Metrics.HTTPReqs.Values.Rate,
		LatencyMs: result.Latency{
			P50: d["p(50)"],
			P95: d["p(95)"],
			P99: d["p(99)"],
		},
		Errors:          int64(raw.Metrics.HTTPReqFailed.Values.Passes),
		ChecksFailed:    int64(raw.Metrics.Checks.Values.Fails),
		DurationSeconds: raw.State.TestRunDurationMs / 1000,
	}, nil
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
// carries the identity/config fields the orchestrator set), including the
// actual measured DurationSeconds, and returns the completed result row.
func (s Summary) Merge(base result.Result) result.Result {
	base.Requests = s.Requests
	base.RequestsPerSecond = s.RequestsPerSecond
	base.LatencyMs = s.LatencyMs
	base.Errors = s.Errors
	base.DurationSeconds = s.DurationSeconds
	return base
}
