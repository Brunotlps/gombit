// Command inspect-limits reports whether a running container actually received
// the resource ceiling the benchmark intended (issue #141 §7 requires the suite
// to "detect and report that fact rather than silently pretending limits were
// applied"). It reads the limits Docker recorded for the container from
// `docker inspect` and classifies them against the intended budget via
// benchmarks/internal/reslimits, printing the honest verdict.
//
//	# after `docker compose ... up -d gin-gorm`:
//	go run ./benchmarks/scripts/inspect-limits \
//	  -container benchmarks-gin-gorm-1 -cpus 2 -memory 1g
//
// This command only *prints* the verdict. Wiring the string into what a run
// records as metadata.resource_limits — automatically in the six-app compose
// loop, or by hand via `run-crud -resource-limits "$(inspect-limits …)"` — is a
// later slice. With -strict it exits non-zero unless the budget was fully
// enforced, so a bring-up script can refuse to record numbers gathered under
// the wrong limits.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"

	"github.com/gombit-dev/gombit/benchmarks/internal/reslimits"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

// run parses args, classifies the container's limits, writes the verdict, and
// returns a process exit code (0 ok, 1 not-enforced under -strict, 2 usage or
// runtime error). It is separated from main so the -strict fail-closed contract
// and the output formats are testable via the -inspect-file seam.
func run(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("inspect-limits", flag.ContinueOnError)
	fs.SetOutput(stderr)
	container := fs.String("container", "", "container name or id to inspect (required)")
	cpus := fs.Float64("cpus", 0, "intended CPU ceiling in whole vCPUs (issue §7: 2)")
	memory := fs.String("memory", "", "intended memory ceiling, compose-style (issue §7: 1g app / 2g postgres)")
	inspectFile := fs.String("inspect-file", "", "read `docker inspect` JSON from this file instead of running docker (testing/offline)")
	format := fs.String("format", "string", "output: string | json")
	strict := fs.Bool("strict", false, "exit non-zero unless the budget was fully enforced")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	if *container == "" {
		return fail(stderr, "set -container=<name|id>")
	}
	if *cpus <= 0 || *memory == "" {
		return fail(stderr, "set the intended budget: -cpus and -memory")
	}
	memBytes, err := reslimits.ParseBytes(*memory)
	if err != nil {
		return fail(stderr, fmt.Sprintf("-memory: %v", err))
	}
	if *format != "string" && *format != "json" {
		return fail(stderr, fmt.Sprintf("-format must be string or json, got %q", *format))
	}

	raw, err := inspect(*container, *inspectFile)
	if err != nil {
		return fail(stderr, err.Error())
	}
	applied, err := reslimits.ParseInspect(raw)
	if err != nil {
		return fail(stderr, err.Error())
	}
	report := reslimits.Classify(*container, reslimits.Budget{CPUs: *cpus, MemoryBytes: memBytes}, applied)

	if *format == "json" {
		enc := json.NewEncoder(stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(report); err != nil {
			return fail(stderr, fmt.Sprintf("encode: %v", err))
		}
	} else {
		_, _ = fmt.Fprintln(stdout, report.String())
	}

	if *strict && report.Status != reslimits.Enforced {
		_, _ = fmt.Fprintf(stderr, "inspect-limits: budget not enforced (%s)\n", report.Status)
		return 1
	}
	return 0
}

// inspect returns the `docker inspect` JSON for container, or reads it from
// file when file != "" (so the tool and its callers can run offline).
func inspect(container, file string) ([]byte, error) {
	if file != "" {
		return os.ReadFile(file) //nolint:gosec // operator-supplied inspect fixture path
	}
	out, err := exec.Command("docker", "inspect", container).Output() //nolint:gosec // container id is an operator-supplied argument
	if err != nil {
		return nil, fmt.Errorf("docker inspect %s: %w", container, err)
	}
	return out, nil
}

// fail writes a diagnostic and returns exit code 2 (usage or runtime error).
func fail(w io.Writer, msg string) int {
	_, _ = fmt.Fprintf(w, "inspect-limits: %s\n", msg)
	return 2
}
