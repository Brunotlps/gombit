// Command inspect-limits reports whether a running container actually received
// the resource ceiling the benchmark intended (issue #141 §7 requires the suite
// to "detect and report that fact rather than silently pretending limits were
// applied"). It reads the container's applied limits from `docker inspect` and
// classifies them against the intended budget via benchmarks/internal/reslimits,
// printing the honest resource_limits string the compose loop records in run
// metadata.
//
//	# after `docker compose ... up -d gin-gorm`:
//	go run ./benchmarks/scripts/inspect-limits \
//	  -container benchmarks-gin-gorm-1 -cpus 2 -memory 1g
//
// With -strict it exits non-zero when the budget was not enforced, so a
// bring-up script can refuse to record numbers gathered under the wrong limits.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"

	"github.com/gombit-dev/gombit/benchmarks/internal/reslimits"
)

func main() {
	container := flag.String("container", "", "container name or id to inspect (required)")
	cpus := flag.Float64("cpus", 0, "intended CPU ceiling in whole vCPUs (issue §7: 2)")
	memory := flag.String("memory", "", "intended memory ceiling, compose-style (issue §7: 1g app / 2g postgres)")
	inspectFile := flag.String("inspect-file", "", "read `docker inspect` JSON from this file instead of running docker (testing/offline)")
	format := flag.String("format", "string", "output: string | json")
	strict := flag.Bool("strict", false, "exit non-zero unless the budget was fully enforced")
	flag.Parse()

	if *container == "" {
		fatalf("set -container=<name|id>")
	}
	if *cpus <= 0 || *memory == "" {
		fatalf("set the intended budget: -cpus and -memory")
	}
	memBytes, err := reslimits.ParseBytes(*memory)
	if err != nil {
		fatalf("-memory: %v", err)
	}
	budget := reslimits.Budget{CPUs: *cpus, MemoryBytes: memBytes}

	raw, err := inspect(*container, *inspectFile)
	if err != nil {
		fatalf("%v", err)
	}
	applied, err := reslimits.ParseInspect(raw)
	if err != nil {
		fatalf("%v", err)
	}
	report := reslimits.Classify(*container, budget, applied)

	switch *format {
	case "json":
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(report); err != nil {
			fatalf("encode: %v", err)
		}
	case "string":
		fmt.Println(report.String())
	default:
		fatalf("-format must be string or json, got %q", *format)
	}

	if *strict && report.Status != reslimits.Enforced {
		fmt.Fprintf(os.Stderr, "inspect-limits: budget not enforced (%s)\n", report.Status)
		os.Exit(1)
	}
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

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "inspect-limits: "+format+"\n", args...)
	os.Exit(1)
}
