// Command summarize reads a results.json snapshot and writes the human report
// (summary.md), generated from the structured per-trial rows — Markdown is
// never the canonical source (issue #141 §9). Per-(framework, concurrency)
// aggregates carry the trial variance and the >5% coefficient-of-variation
// flag (issue §7).
//
//	go run ./benchmarks/scripts/summarize \
//	  -results benchmarks/results/latest/results.json \
//	  -out benchmarks/results/latest/summary.md
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/gombit-dev/gombit/benchmarks/internal/result"
	"github.com/gombit-dev/gombit/benchmarks/internal/summary"
)

func main() {
	results := flag.String("results", "benchmarks/results/latest/results.json", "input results.json")
	out := flag.String("out", "", "output summary.md (default: stdout)")
	covThreshold := flag.Float64("cov-threshold", summary.DefaultCoVThreshold, "throughput CoV above which a group is flagged high-variance")
	flag.Parse()

	rows, err := readResults(*results)
	if err != nil {
		fatalf("read results: %v", err)
	}
	groups := summary.Summarize(rows, *covThreshold)

	if err := write(*out, func(w *os.File) error {
		return summary.WriteMarkdown(w, groups)
	}); err != nil {
		fatalf("%v", err)
	}
}

func readResults(path string) ([]result.Result, error) {
	f, err := os.Open(path) //nolint:gosec // operator-supplied input path
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()
	return result.ReadJSON(f)
}

func write(path string, encode func(*os.File) error) error {
	if path == "" {
		return encode(os.Stdout)
	}
	f, err := os.Create(path) //nolint:gosec // operator-supplied output path
	if err != nil {
		return err
	}
	if err := encode(f); err != nil {
		_ = f.Close()
		return err
	}
	return f.Close()
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "summarize: "+format+"\n", args...)
	os.Exit(1)
}
