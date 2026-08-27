// Command microbench parses `go test -bench=BenchmarkFrameworkTax` output for
// one stack (read from stdin) and merges the rows into OUT/microbench.json by
// (stack, scenario). `make benchmark-micro` pipes each stack's `go test` output
// through it, so the four runs accumulate into one file the report reads.
//
//	go test ./benchmarks/micro/gin -bench=BenchmarkFrameworkTax -benchmem -run='^$' \
//	  | go run ./benchmarks/scripts/microbench -stack gin -out benchmarks/results/latest/microbench.json
package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/gombit-dev/gombit/benchmarks/internal/microbench"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}

func run(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("microbench", flag.ContinueOnError)
	fs.SetOutput(stderr)
	stack := fs.String("stack", "", "the stack whose `go test -bench` output is on stdin (nethttp|gin|huma|gombit)")
	out := fs.String("out", "benchmarks/results/latest/microbench.json", "output microbench.json")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if *stack == "" {
		_, _ = fmt.Fprintln(stderr, "microbench: -stack is required")
		return 2
	}

	rows, err := microbench.ParseBenchOutput(*stack, stdin)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "microbench: %v\n", err)
		return 1
	}
	if len(rows) == 0 {
		_, _ = fmt.Fprintf(stderr, "microbench: no BenchmarkFrameworkTax rows for %s on stdin\n", *stack)
		return 1
	}

	existing, err := readExisting(*out)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "microbench: %v\n", err)
		return 1
	}
	merged := microbench.Merge(existing, rows)

	if err := os.MkdirAll(filepath.Dir(*out), 0o755); err != nil { //nolint:gosec // operator-supplied out dir
		_, _ = fmt.Fprintf(stderr, "microbench: %v\n", err)
		return 1
	}
	f, err := os.Create(*out) //nolint:gosec // operator-supplied out path
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "microbench: %v\n", err)
		return 1
	}
	if err := microbench.WriteJSON(f, merged); err != nil {
		_ = f.Close()
		_, _ = fmt.Fprintf(stderr, "microbench: %v\n", err)
		return 1
	}
	if err := f.Close(); err != nil {
		_, _ = fmt.Fprintf(stderr, "microbench: %v\n", err)
		return 1
	}
	_, _ = fmt.Fprintf(stdout, "microbench: merged %d %s rows into %s\n", len(rows), *stack, *out)
	return 0
}

func readExisting(path string) ([]microbench.Row, error) {
	f, err := os.Open(path) //nolint:gosec // operator-supplied out path
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()
	return microbench.ReadJSON(f)
}
