// Package summary turns the per-trial rows a benchmark run records
// (benchmarks/internal/result) into per-(framework, benchmark, concurrency)
// aggregates with trial variance, and renders the Markdown report from them.
// Markdown is generated from this structured summary, never hand-authored
// (issue #141 §9); the coefficient-of-variation flag surfaces runs whose
// trials disagree enough to distrust (issue §7 "coefficient-of-variation flag
// at >5%").
package summary

import (
	"fmt"
	"io"
	"math"
	"sort"

	"github.com/gombit-dev/gombit/benchmarks/internal/result"
)

// DefaultCoVThreshold is the >5% trial-variance flag from issue #141 §7.
const DefaultCoVThreshold = 0.05

// Stats is the median, mean, sample standard deviation, and coefficient of
// variation (stddev/mean) of one metric across a group's trials. Median is the
// issue's published location statistic (#141: "report at minimum the median
// result across trials") — it is the reported headline, so one noisy trial
// can't become the number someone copies; Mean/StdDev feed CoV, the stability
// measure, not the headline.
type Stats struct {
	Median float64 `json:"median"`
	Mean   float64 `json:"mean"`
	StdDev float64 `json:"stddev"`
	CoV    float64 `json:"cov"`
}

// Group aggregates every trial that shares a (Framework, Benchmark,
// Concurrency) key. RequestsPerSecond variance drives the CoV flag; latency
// percentiles are averaged across trials. Errors is the total observed (any
// non-zero is worth surfacing even though the runner rejects error trials —
// a summary of previously-collected data may still contain them).
type Group struct {
	Framework         string  `json:"framework"`
	FrameworkVersion  string  `json:"framework_version"`
	Runtime           string  `json:"runtime"`
	Benchmark         string  `json:"benchmark"`
	Concurrency       int     `json:"concurrency"`
	Trials            int     `json:"trials"`
	RequestsPerSecond Stats   `json:"requests_per_second"`
	LatencyP50        Stats   `json:"latency_p50_ms"`
	LatencyP95        Stats   `json:"latency_p95_ms"`
	LatencyP99        Stats   `json:"latency_p99_ms"`
	Errors            int64   `json:"errors"`
	CoVThreshold      float64 `json:"cov_threshold"`
	// HighVariance is true when the throughput CoV exceeds CoVThreshold — the
	// group's trials disagree enough that the mean should be read with care.
	HighVariance bool `json:"high_variance"`
}

// Summarize groups the results and computes per-group stats. covThreshold is
// the fraction (e.g. 0.05) above which a group's throughput CoV sets
// HighVariance. Groups are returned benchmark-primary — deterministic order
// (benchmark, framework, concurrency) — so every framework for one workload is
// contiguous, which is the order WriteMarkdown's one-table-per-benchmark walk
// requires and the comparison the report exists to show.
func Summarize(results []result.Result, covThreshold float64) []Group {
	type key struct {
		framework, benchmark string
		concurrency          int
	}
	order := []key{}
	buckets := map[key][]result.Result{}
	for _, r := range results {
		k := key{r.Framework, r.Benchmark, r.Concurrency}
		if _, seen := buckets[k]; !seen {
			order = append(order, k)
		}
		buckets[k] = append(buckets[k], r)
	}

	groups := make([]Group, 0, len(order))
	for _, k := range order {
		rows := buckets[k]
		rps := make([]float64, len(rows))
		p50 := make([]float64, len(rows))
		p95 := make([]float64, len(rows))
		p99 := make([]float64, len(rows))
		var errors int64
		for i, r := range rows {
			rps[i] = r.RequestsPerSecond
			p50[i] = r.LatencyMs.P50
			p95[i] = r.LatencyMs.P95
			p99[i] = r.LatencyMs.P99
			errors += r.Errors
		}
		rpsStats := computeStats(rps)
		groups = append(groups, Group{
			Framework:         rows[0].Framework,
			FrameworkVersion:  rows[0].FrameworkVersion,
			Runtime:           rows[0].Runtime,
			Benchmark:         k.benchmark,
			Concurrency:       k.concurrency,
			Trials:            len(rows),
			RequestsPerSecond: rpsStats,
			LatencyP50:        computeStats(p50),
			LatencyP95:        computeStats(p95),
			LatencyP99:        computeStats(p99),
			Errors:            errors,
			CoVThreshold:      covThreshold,
			HighVariance:      rpsStats.CoV > covThreshold,
		})
	}

	sort.SliceStable(groups, func(i, j int) bool {
		a, b := groups[i], groups[j]
		switch {
		case a.Benchmark != b.Benchmark:
			return a.Benchmark < b.Benchmark
		case a.Framework != b.Framework:
			return a.Framework < b.Framework
		default:
			return a.Concurrency < b.Concurrency
		}
	})
	return groups
}

// computeStats returns the median, mean, sample (n-1) standard deviation, and
// CoV of xs. A single trial has no dispersion (stddev/CoV 0) and median==mean;
// a zero mean yields CoV 0 rather than a division by zero.
func computeStats(xs []float64) Stats {
	n := len(xs)
	if n == 0 {
		return Stats{}
	}
	var sum float64
	for _, x := range xs {
		sum += x
	}
	mean := sum / float64(n)
	med := median(xs)
	if n == 1 {
		return Stats{Median: med, Mean: mean}
	}
	var ss float64
	for _, x := range xs {
		d := x - mean
		ss += d * d
	}
	std := math.Sqrt(ss / float64(n-1))
	cov := 0.0
	if mean != 0 {
		cov = std / mean
	}
	return Stats{Median: med, Mean: mean, StdDev: std, CoV: cov}
}

// median returns the middle value of xs (mean of the two middle values for an
// even count), without mutating the caller's slice. xs must be non-empty.
func median(xs []float64) float64 {
	s := append([]float64(nil), xs...)
	sort.Float64s(s)
	n := len(s)
	if n%2 == 1 {
		return s[n/2]
	}
	return (s[n/2-1] + s[n/2]) / 2
}

// WriteMarkdown renders the human report from the groups — one table per
// benchmark, one row per (framework, concurrency). The headline numbers are the
// **median** across trials (issue #141's published location statistic, robust
// to one noisy trial); `rps CoV` is the coefficient of variation of throughput
// (on the mean) and ⚠ marks high-variance rows. It leads with the methodology
// caveats the numbers must be read with (closed-loop coordinated omission,
// same-host topology) so a copied table can't be read as more than it is.
//
// Groups must be benchmark-contiguous (as Summarize returns them): the
// one-table-per-benchmark layout is a walk over that order, so an unsorted or
// framework-primary slice would split a workload across several tables.
func WriteMarkdown(w io.Writer, groups []Group) error {
	bw := &errWriter{w: w}
	bw.printf("# Benchmark summary\n\n")
	bw.printf("Generated from `results.json` (`benchmarks/internal/summary`) — do not edit by hand.\n\n")
	bw.printf("**Read with:** the load model is closed-loop `constant-vus` (N concurrent clients), ")
	bw.printf("so reported tail latency is subject to **coordinated omission** and understates true ")
	bw.printf("client-observed wait; the load generator shares the host with the app. ")
	bw.printf("Headline `rps`/`p50`/`p95`/`p99` are the **median** across trials; ")
	bw.printf("`rps CoV` is the coefficient of variation of throughput (stddev/mean) across trials; ")
	bw.printf("⚠ marks a group whose trials vary by more than %.0f%%, whose numbers should be distrusted.\n",
		groupsThreshold(groups)*100)

	// One table per benchmark. Summarize returns benchmark-contiguous groups, so
	// a heading opens each time the benchmark changes and every framework for a
	// workload lands in that one table.
	var lastBench string
	for i, g := range groups {
		if g.Benchmark != lastBench {
			if i != 0 {
				bw.printf("\n")
			}
			bw.printf("\n## %s\n\n", g.Benchmark)
			bw.printf("| framework | concurrency | trials | rps (median) | rps CoV | p50 ms | p95 ms | p99 ms | errors |\n")
			bw.printf("| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: |\n")
			lastBench = g.Benchmark
		}
		flag := ""
		if g.HighVariance {
			flag = " ⚠"
		}
		bw.printf("| %s | %d | %d | %.0f | %.1f%%%s | %.1f | %.1f | %.1f | %d |\n",
			g.Framework, g.Concurrency, g.Trials,
			g.RequestsPerSecond.Median, g.RequestsPerSecond.CoV*100, flag,
			g.LatencyP50.Median, g.LatencyP95.Median, g.LatencyP99.Median, g.Errors)
	}
	return bw.err
}

func groupsThreshold(groups []Group) float64 {
	if len(groups) == 0 {
		return DefaultCoVThreshold
	}
	return groups[0].CoVThreshold
}

// errWriter collapses repeated write-error checks in the Markdown rendering.
type errWriter struct {
	w   io.Writer
	err error
}

func (e *errWriter) printf(format string, args ...any) {
	if e.err != nil {
		return
	}
	_, e.err = fmt.Fprintf(e.w, format, args...)
}
