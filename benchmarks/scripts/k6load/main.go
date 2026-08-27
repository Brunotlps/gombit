// Command k6load runs the crud-list workload against a target for a fixed
// window and KEEPS + validates the k6 summary, exiting non-zero unless the load
// was a clean measurement (traffic sent, no HTTP errors, no failed checks —
// benchmarks/internal/k6's Summary.Validate). It is the load half of the
// footprint measurement: the load generator, not a wall-clock guess, is the
// authority for whether load actually happened, so footprint-all.sh only
// records loaded-memory / CPU numbers when this exits 0. Unlike run-crud it
// writes no results row — it exists purely to drive validated load while the
// orchestrator samples `docker stats` concurrently.
//
//	go run ./benchmarks/scripts/k6load \
//	  -target-url 'http://127.0.0.1:8081/api/projects?page=1&limit=20' \
//	  -vus 100 -duration 10s -k6-image grafana/k6:0.55.0 \
//	  -workload benchmarks/workloads/crud-list.js
package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"

	"github.com/gombit-dev/gombit/benchmarks/internal/k6"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("k6load", flag.ContinueOnError)
	fs.SetOutput(stderr)
	var (
		target   = fs.String("target-url", "", "the app's list endpoint (required)")
		vus      = fs.Int("vus", 100, "concurrent VUs")
		duration = fs.String("duration", "10s", "load duration (k6 duration string)")
		image    = fs.String("k6-image", "grafana/k6:0.55.0", "pinned k6 image")
		workload = fs.String("workload", "benchmarks/workloads/crud-list.js", "k6 workload script")
		runner   = fs.String("docker", "docker", "docker binary (overridable for tests)")
	)
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if *target == "" {
		_, _ = fmt.Fprintln(stderr, "k6load: -target-url is required")
		return 2
	}
	workloadAbs, err := filepath.Abs(*workload)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "k6load: %v\n", err)
		return 2
	}

	summaryDir, err := os.MkdirTemp("", "k6load")
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "k6load: %v\n", err)
		return 1
	}
	defer func() { _ = os.RemoveAll(summaryDir) }()

	// Same env/mount contract as run-crud's dockerK6Runner, plus SUMMARY_OUT so
	// the summary is kept for validation (run-crud § dockerK6Runner).
	dockerArgs := []string{
		"run", "--rm", "--network", "host",
		"--user", fmt.Sprintf("%d:%d", os.Getuid(), os.Getgid()),
		"-e", "TARGET_URL=" + *target,
		"-e", "VUS=" + strconv.Itoa(*vus),
		"-e", "DURATION=" + *duration,
		"-e", "SUMMARY_OUT=/out/summary.json",
		"-v", workloadAbs + ":/workload/crud-list.js:ro",
		"-v", summaryDir + ":/out",
		*image, "run", "--quiet", "/workload/crud-list.js",
	}
	cmd := exec.CommandContext(context.Background(), *runner, dockerArgs...) //nolint:gosec // fixed argv, operator-supplied target/image only
	cmd.Stdout = stderr                                                      // k6's own progress goes to stderr, keeping our stdout clean
	cmd.Stderr = stderr
	if err := cmd.Run(); err != nil {
		_, _ = fmt.Fprintf(stderr, "k6load: k6 did not run: %v\n", err)
		return 1
	}

	summary, err := parseSummary(filepath.Join(summaryDir, "summary.json"))
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "k6load: %v\n", err)
		return 1
	}
	if err := summary.Validate(); err != nil {
		_, _ = fmt.Fprintf(stderr, "k6load: load was not a clean measurement: %v\n", err)
		return 1
	}
	_, _ = fmt.Fprintf(stdout, "k6load: %d requests, %d errors over %.1fs — clean\n",
		summary.Requests, summary.Errors, summary.DurationSeconds)
	return 0
}

func parseSummary(path string) (k6.Summary, error) {
	f, err := os.Open(path) //nolint:gosec // path composed from our own temp dir
	if err != nil {
		return k6.Summary{}, fmt.Errorf("k6 produced no summary: %w", err)
	}
	defer func() { _ = f.Close() }()
	return k6.ParseSummary(f)
}
