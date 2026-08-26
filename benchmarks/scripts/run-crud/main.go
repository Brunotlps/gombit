// Command run-crud runs the headline CRUD-read workload
// (benchmarks/workloads/crud-list.js) against one already-running,
// already-seeded implementation and appends its rows to a results snapshot.
//
// It drives the pinned k6 image (the load generator runs in its own
// container, off the application host — issue #141 requirement), warms up
// with a discarded run, then measures TRIALS times at each concurrency level,
// parses each summary, and writes results.json/results.csv plus metadata.json.
// A failed k6 run or an empty summary fails the command loudly rather than
// recording a silent zero (issue #141 §10).
//
// The full make benchmark-crud that brings up all six apps under compose and
// loops this over them is the next slice; this is the per-implementation
// engine, verified end-to-end against benchmarks/apps/gin-gorm.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gombit-dev/gombit/benchmarks/internal/k6"
	"github.com/gombit-dev/gombit/benchmarks/internal/metadata"
	"github.com/gombit-dev/gombit/benchmarks/internal/result"
)

func main() {
	var (
		targetURL       = flag.String("target-url", "", "the app's GET list endpoint, e.g. http://127.0.0.1:8081/api/projects?page=1&limit=20")
		framework       = flag.String("framework", "", "framework name for the result rows, e.g. gin-gorm")
		frameworkVer    = flag.String("framework-version", "", "framework version")
		runtimeName     = flag.String("runtime", "", "runtime, e.g. go")
		runtimeVer      = flag.String("runtime-version", "", "runtime version")
		concurrency     = flag.String("concurrency", "1,10,100", "comma-separated concurrency levels (VUs)")
		duration        = flag.String("duration", "30s", "measured per-trial duration (k6 duration string)")
		warmup          = flag.String("warmup", "10s", "warm-up duration, discarded")
		trials          = flag.Int("trials", 5, "measured trials per concurrency level")
		outDir          = flag.String("out-dir", "benchmarks/results/latest", "output directory for results.json/results.csv/metadata.json")
		k6Image         = flag.String("k6-image", "grafana/k6:0.55.0", "pinned k6 image (the load generator)")
		workload        = flag.String("workload", "benchmarks/workloads/crud-list.js", "k6 workload script")
		postgresVersion = flag.String("postgres-version", "", "PostgreSQL version, for metadata")
		resourceLimits  = flag.String("resource-limits", "", "documented resource limits, for metadata")
	)
	flag.Parse()

	if *targetURL == "" || *framework == "" {
		fatalf("-target-url and -framework are required")
	}
	levels, err := parseIntList(*concurrency)
	if err != nil {
		fatalf("-concurrency: %v", err)
	}
	if len(levels) == 0 {
		fatalf("-concurrency must list at least one level")
	}

	workloadAbs, err := filepath.Abs(*workload)
	if err != nil {
		fatalf("resolve workload path: %v", err)
	}
	outAbs, err := filepath.Abs(*outDir)
	if err != nil {
		fatalf("resolve out dir: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(outAbs, "raw"), 0o750); err != nil {
		fatalf("create out dir: %v", err)
	}

	ctx := context.Background()
	base := result.Result{
		SchemaVersion:    result.SchemaVersion,
		Framework:        *framework,
		FrameworkVersion: *frameworkVer,
		Runtime:          *runtimeName,
		RuntimeVersion:   *runtimeVer,
		Benchmark:        "crud-list",
		Database:         "postgresql",
		DurationSeconds:  durationSeconds(*duration),
	}

	var rows []result.Result
	for _, vus := range levels {
		// One discarded warm-up per concurrency level before its measured
		// trials, so the app is at steady state.
		if err := runK6(ctx, *k6Image, workloadAbs, *targetURL, vus, *warmup, ""); err != nil {
			fatalf("warm-up (vus=%d): %v", vus, err)
		}
		for trial := 1; trial <= *trials; trial++ {
			summaryPath := filepath.Join(outAbs, "raw", fmt.Sprintf("%s_c%d_t%d.json", *framework, vus, trial))
			if err := runK6(ctx, *k6Image, workloadAbs, *targetURL, vus, *duration, summaryPath); err != nil {
				fatalf("k6 run (vus=%d trial=%d): %v", vus, trial, err)
			}
			summary, err := parseSummaryFile(summaryPath)
			if err != nil {
				fatalf("parse summary (vus=%d trial=%d): %v", vus, trial, err)
			}
			if err := summary.Validate(); err != nil {
				fatalf("%s vus=%d trial=%d: %v", *framework, vus, trial, err)
			}
			row := base
			row.Concurrency = vus
			row.Trial = trial
			rows = append(rows, summary.Merge(row))
			fmt.Fprintf(os.Stderr, "run-crud: %s vus=%d trial=%d -> %.0f rps, p95=%.1fms, errors=%d\n",
				*framework, vus, trial, summary.RequestsPerSecond, summary.LatencyMs.P95, summary.Errors)
		}
	}

	if err := writeOutputs(outAbs, rows, metadata.Collect(ctx, metadata.Options{
		PostgresVersion:   *postgresVersion,
		FrameworkVersions: map[string]string{*framework: *frameworkVer},
		RuntimeVersions:   map[string]string{*runtimeName: *runtimeVer},
		BenchmarkTool:     *k6Image,
		ResourceLimits:    *resourceLimits,
		DurationSeconds:   durationSeconds(*duration),
		WarmupSeconds:     durationSeconds(*warmup),
		Concurrency:       levels,
		Trials:            *trials,
	})); err != nil {
		fatalf("write outputs: %v", err)
	}
	fmt.Fprintf(os.Stderr, "run-crud: wrote %d rows to %s\n", len(rows), outAbs)
}

// runK6 runs the pinned k6 image against targetURL at vus concurrency for
// duration. When summaryPath is non-empty the workload writes its summary
// there (mounted into the container); an empty summaryPath is a warm-up whose
// output is discarded. --network host lets the containerized load generator
// reach an app listening on the host.
func runK6(ctx context.Context, image, workloadAbs, targetURL string, vus int, duration, summaryPath string) error {
	args := []string{
		"run", "--rm", "--network", "host",
		// Run k6 as the current user so the summary it writes into the
		// mounted output dir is owned by (and writable for) us — the k6
		// image's own non-root user otherwise can't write a host dir we own.
		"--user", fmt.Sprintf("%d:%d", os.Getuid(), os.Getgid()),
		"-e", "TARGET_URL=" + targetURL,
		"-e", "VUS=" + strconv.Itoa(vus),
		"-e", "DURATION=" + duration,
		"-v", workloadAbs + ":/workload/crud-list.js:ro",
	}
	if summaryPath != "" {
		args = append(args,
			"-e", "SUMMARY_OUT=/out/summary.json",
			"-v", filepath.Dir(summaryPath)+":/out",
		)
	}
	args = append(args, image, "run", "--quiet", "/workload/crud-list.js")

	cmd := exec.CommandContext(ctx, "docker", args...) //nolint:gosec // fixed argv, operator-supplied target only
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return err
	}
	if summaryPath != "" {
		// k6 wrote /out/summary.json; move it to the trial-specific name.
		if err := os.Rename(filepath.Join(filepath.Dir(summaryPath), "summary.json"), summaryPath); err != nil {
			return fmt.Errorf("k6 produced no summary: %w", err)
		}
	}
	return nil
}

func parseSummaryFile(path string) (k6.Summary, error) {
	f, err := os.Open(path) //nolint:gosec // path is composed from the operator-supplied out-dir
	if err != nil {
		return k6.Summary{}, err
	}
	defer func() { _ = f.Close() }()
	return k6.ParseSummary(f)
}

func writeOutputs(outDir string, rows []result.Result, meta metadata.Metadata) error {
	if err := writeFile(filepath.Join(outDir, "results.json"), func(f *os.File) error {
		return result.WriteJSON(f, rows)
	}); err != nil {
		return err
	}
	if err := writeFile(filepath.Join(outDir, "results.csv"), func(f *os.File) error {
		return result.WriteCSV(f, rows)
	}); err != nil {
		return err
	}
	return writeFile(filepath.Join(outDir, "metadata.json"), func(f *os.File) error {
		return metadata.WriteJSON(f, meta)
	})
}

func writeFile(path string, encode func(*os.File) error) error {
	f, err := os.Create(path) //nolint:gosec // path composed from operator-supplied out-dir
	if err != nil {
		return err
	}
	if err := encode(f); err != nil {
		_ = f.Close()
		return err
	}
	return f.Close()
}

// durationSeconds converts a k6 duration string ("30s", "1m30s") to seconds
// for the metadata / result duration_seconds field; unparseable input is 0.
func durationSeconds(s string) float64 {
	d, err := time.ParseDuration(s)
	if err != nil {
		return 0
	}
	return d.Seconds()
}

func parseIntList(s string) ([]int, error) {
	if strings.TrimSpace(s) == "" {
		return nil, nil
	}
	var out []int
	for _, part := range strings.Split(s, ",") {
		n, err := strconv.Atoi(strings.TrimSpace(part))
		if err != nil {
			return nil, fmt.Errorf("invalid integer %q", strings.TrimSpace(part))
		}
		out = append(out, n)
	}
	return out, nil
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "run-crud: "+format+"\n", args...)
	os.Exit(1)
}
