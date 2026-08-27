// Command report regenerates the root README's `## Performance` block from the
// committed benchmark outputs, or (-check) verifies the committed README still
// matches — the drift guard for issue #141's "README is regenerable, never
// hand-edited" AC.
//
//	go run ./benchmarks/scripts/report                    # rewrite README.md
//	go run ./benchmarks/scripts/report -check             # exit non-zero on drift
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/gombit-dev/gombit/benchmarks/internal/footprint"
	"github.com/gombit-dev/gombit/benchmarks/internal/metadata"
	"github.com/gombit-dev/gombit/benchmarks/internal/microbench"
	"github.com/gombit-dev/gombit/benchmarks/internal/report"
	"github.com/gombit-dev/gombit/benchmarks/internal/result"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("report", flag.ContinueOnError)
	fs.SetOutput(stderr)
	var (
		resultsPath   = fs.String("results", "benchmarks/results/latest/results.json", "throughput results.json")
		footprintPath = fs.String("footprint", "benchmarks/results/latest/footprint.json", "footprint.json")
		microPath     = fs.String("micro", "benchmarks/results/latest/microbench.json", "framework-tax microbench.json")
		metadataPath  = fs.String("metadata", "benchmarks/results/latest/metadata.json", "run metadata.json")
		readmePath    = fs.String("readme", "README.md", "README to regenerate / check")
		check         = fs.Bool("check", false, "verify the README block matches the data instead of rewriting it")
	)
	if err := fs.Parse(args); err != nil {
		return 2
	}

	results, err := readResults(*resultsPath)
	if err != nil {
		return fail(stderr, err)
	}
	prints, err := readFootprint(*footprintPath)
	if err != nil {
		return fail(stderr, err)
	}
	micro, err := readMicro(*microPath)
	if err != nil {
		return fail(stderr, err)
	}
	meta, err := readMetadata(*metadataPath)
	if err != nil {
		return fail(stderr, err)
	}
	readme, err := os.ReadFile(*readmePath) //nolint:gosec // operator-supplied README path
	if err != nil {
		return fail(stderr, err)
	}

	block := report.Render(results, prints, micro, meta)

	if *check {
		ok, err := report.InSync(string(readme), block)
		if err != nil {
			return fail(stderr, err)
		}
		if !ok {
			_, _ = fmt.Fprintf(stderr, "report: %s is out of date — run `make benchmark-report`\n", *readmePath)
			return 1
		}
		_, _ = fmt.Fprintf(stdout, "report: %s matches the benchmark data\n", *readmePath)
		return 0
	}

	updated, err := report.ReplaceBlock(string(readme), block)
	if err != nil {
		return fail(stderr, err)
	}
	if err := os.WriteFile(*readmePath, []byte(updated), 0o644); err != nil { //nolint:gosec // README is world-readable by design
		return fail(stderr, err)
	}
	_, _ = fmt.Fprintf(stdout, "report: regenerated the Performance block in %s\n", *readmePath)
	return 0
}

// readResults / readFootprint / readMetadata treat a missing file as empty, so
// the report is generable before the canonical run (it renders honest
// "not yet recorded" sections).
func readResults(path string) ([]result.Result, error) {
	f, err := open(path)
	if f == nil || err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()
	return result.ReadJSON(f)
}

func readFootprint(path string) ([]footprint.Footprint, error) {
	f, err := open(path)
	if f == nil || err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()
	return footprint.ReadJSON(f)
}

func readMicro(path string) ([]microbench.Row, error) {
	f, err := open(path)
	if f == nil || err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()
	return microbench.ReadJSON(f)
}

func readMetadata(path string) (metadata.Metadata, error) {
	f, err := open(path)
	if f == nil || err != nil {
		return metadata.Metadata{}, err
	}
	defer func() { _ = f.Close() }()
	var m metadata.Metadata
	if err := json.NewDecoder(f).Decode(&m); err != nil {
		return metadata.Metadata{}, fmt.Errorf("decode %s: %w", path, err)
	}
	return m, nil
}

// open returns (nil, nil) if the path does not exist, so callers render empty.
func open(path string) (*os.File, error) {
	f, err := os.Open(path) //nolint:gosec // operator-supplied results path
	if os.IsNotExist(err) {
		return nil, nil
	}
	return f, err
}

func fail(stderr io.Writer, err error) int {
	_, _ = fmt.Fprintf(stderr, "report: %v\n", err)
	return 1
}
