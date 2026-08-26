// Command collect-host-info writes the reproducibility metadata for a
// benchmark run (issue #141 "Reproducibility metadata") as JSON. The host and
// toolchain fields are discovered automatically; the run-parameter fields
// (durations, concurrency, trials, resource limits, tool versions) are passed
// as flags by the orchestrator that knows them.
//
//	go run ./benchmarks/scripts/collect-host-info -out benchmarks/results/latest/metadata.json
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/gombit-dev/gombit/benchmarks/internal/metadata"
)

func main() {
	out := flag.String("out", "", "output file for metadata.json (default: stdout)")
	postgres := flag.String("postgres-version", "", "PostgreSQL version under test")
	benchmarkTool := flag.String("benchmark-tool", "", "load generator name+version, e.g. 'k6 0.55.0'")
	resourceLimits := flag.String("resource-limits", "", "documented resource limits for this run")
	duration := flag.Float64("duration-seconds", 0, "per-trial measured duration")
	warmup := flag.Float64("warmup-seconds", 0, "warm-up duration before measuring")
	concurrency := flag.String("concurrency", "", "comma-separated concurrency levels, e.g. '1,10,100'")
	trials := flag.Int("trials", 0, "number of trials per concurrency level")
	flag.Parse()

	m := metadata.Collect(context.Background(), metadata.Options{
		PostgresVersion: *postgres,
		BenchmarkTool:   *benchmarkTool,
		ResourceLimits:  *resourceLimits,
		DurationSeconds: *duration,
		WarmupSeconds:   *warmup,
		Concurrency:     parseIntList(*concurrency),
		Trials:          *trials,
	})

	if err := write(*out, m); err != nil {
		fmt.Fprintf(os.Stderr, "collect-host-info: %v\n", err)
		os.Exit(1)
	}
}

// write encodes m to path (or stdout when path is empty), checking the close
// error on the output file so a full-disk / short-write failure is not lost.
func write(path string, m metadata.Metadata) error {
	if path == "" {
		return metadata.WriteJSON(os.Stdout, m)
	}
	// path is the operator-supplied -out flag (a CLI writing its own output
	// file), not untrusted input — G304 does not apply.
	f, err := os.Create(path) //nolint:gosec
	if err != nil {
		return err
	}
	if err := metadata.WriteJSON(f, m); err != nil {
		_ = f.Close()
		return err
	}
	return f.Close()
}

func parseIntList(s string) []int {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	var out []int
	for _, part := range strings.Split(s, ",") {
		n, err := strconv.Atoi(strings.TrimSpace(part))
		if err != nil {
			continue
		}
		out = append(out, n)
	}
	return out
}
