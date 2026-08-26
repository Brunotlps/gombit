// Package metadata collects the reproducibility metadata every full benchmark
// run must capture (issue #141 "Reproducibility metadata"): enough about the
// host, toolchain, and run parameters that a published table can be
// reproduced. "Never publish a table without enough metadata to reproduce
// it."
package metadata

import (
	"context"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// SchemaVersion is bumped when the Metadata shape changes incompatibly.
const SchemaVersion = 1

// Metadata is the machine-readable run environment (written to
// results/latest/metadata.json). Discovered fields come from the host and
// toolchain; the rest are run parameters the orchestrator supplies via
// Options because they are choices, not facts about the machine.
type Metadata struct {
	SchemaVersion int    `json:"schema_version"`
	Timestamp     string `json:"timestamp"`

	GitCommit string `json:"git_commit"`
	GitDirty  bool   `json:"git_dirty"`

	OS          string `json:"os"`
	Kernel      string `json:"kernel"`
	Arch        string `json:"arch"`
	CPUModel    string `json:"cpu_model"`
	LogicalCPUs int    `json:"logical_cpus"`
	RAMBytes    int64  `json:"ram_bytes"`

	GoVersion            string `json:"go_version"`
	DockerVersion        string `json:"docker_version"`
	DockerComposeVersion string `json:"docker_compose_version"`
	PostgresVersion      string `json:"postgres_version"`

	FrameworkVersions map[string]string `json:"framework_versions"`
	RuntimeVersions   map[string]string `json:"runtime_versions"`
	BenchmarkTool     string            `json:"benchmark_tool"`

	ResourceLimits  string  `json:"resource_limits"`
	DurationSeconds float64 `json:"duration_seconds"`
	WarmupSeconds   float64 `json:"warmup_seconds"`
	Concurrency     []int   `json:"concurrency"`
	Trials          int     `json:"trials"`
}

// Runner runs a command and returns its trimmed stdout. It exists so tests can
// stub git/uname/docker without those tools being installed; production uses
// execRunner.
type Runner func(ctx context.Context, name string, args ...string) (string, error)

// Options carries the injectable seams (Now, Run) and the run-parameter fields
// the orchestrator knows but the host does not.
type Options struct {
	Now func() time.Time
	Run Runner

	PostgresVersion   string
	FrameworkVersions map[string]string
	RuntimeVersions   map[string]string
	BenchmarkTool     string
	ResourceLimits    string
	DurationSeconds   float64
	WarmupSeconds     float64
	Concurrency       []int
	Trials            int
}

// Collect gathers the metadata. Discovery is best-effort: a missing tool
// (docker on a CI runner without it, git outside a checkout) leaves its field
// empty rather than failing the whole collection — a benchmark run should
// still record everything it *can*.
func Collect(ctx context.Context, opts Options) Metadata {
	now := opts.Now
	if now == nil {
		now = func() time.Time { return time.Now() }
	}
	run := opts.Run
	if run == nil {
		run = execRunner
	}

	commit, _ := run(ctx, "git", "rev-parse", "HEAD")
	status, _ := run(ctx, "git", "status", "--porcelain")
	kernel, _ := run(ctx, "uname", "-r")
	dockerVersion, _ := run(ctx, "docker", "version", "--format", "{{.Server.Version}}")
	composeVersion, _ := run(ctx, "docker", "compose", "version", "--short")

	return Metadata{
		SchemaVersion: SchemaVersion,
		Timestamp:     now().UTC().Format(time.RFC3339),

		GitCommit: commit,
		GitDirty:  strings.TrimSpace(status) != "",

		OS:          runtime.GOOS,
		Kernel:      kernel,
		Arch:        runtime.GOARCH,
		CPUModel:    cpuModel(),
		LogicalCPUs: runtime.NumCPU(),
		RAMBytes:    ramBytes(),

		GoVersion:            runtime.Version(),
		DockerVersion:        dockerVersion,
		DockerComposeVersion: composeVersion,
		PostgresVersion:      opts.PostgresVersion,

		FrameworkVersions: opts.FrameworkVersions,
		RuntimeVersions:   opts.RuntimeVersions,
		BenchmarkTool:     opts.BenchmarkTool,

		ResourceLimits:  opts.ResourceLimits,
		DurationSeconds: opts.DurationSeconds,
		WarmupSeconds:   opts.WarmupSeconds,
		Concurrency:     opts.Concurrency,
		Trials:          opts.Trials,
	}
}

func execRunner(ctx context.Context, name string, args ...string) (string, error) {
	// The command names are fixed literals at the call sites (git, uname,
	// docker), never user input — G204 does not apply.
	out, err := exec.CommandContext(ctx, name, args...).Output() //nolint:gosec
	return strings.TrimSpace(string(out)), err
}

// cpuModel and ramBytes read Linux's /proc directly (best-effort); on other
// platforms, or if the files are unreadable, they return the zero value. The
// parsing is split into pure functions so it's unit-tested without /proc.
func cpuModel() string {
	data, err := os.ReadFile("/proc/cpuinfo")
	if err != nil {
		return ""
	}
	return parseCPUModel(string(data))
}

func ramBytes() int64 {
	data, err := os.ReadFile("/proc/meminfo")
	if err != nil {
		return 0
	}
	return parseMemTotalBytes(string(data))
}

func parseCPUModel(cpuinfo string) string {
	for _, line := range strings.Split(cpuinfo, "\n") {
		key, value, ok := strings.Cut(line, ":")
		if ok && strings.TrimSpace(key) == "model name" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

// parseMemTotalBytes reads the "MemTotal:" line of /proc/meminfo, which is in
// kibibytes ("MemTotal:  16311072 kB"), and returns bytes.
func parseMemTotalBytes(meminfo string) int64 {
	for _, line := range strings.Split(meminfo, "\n") {
		rest, ok := strings.CutPrefix(line, "MemTotal:")
		if !ok {
			continue
		}
		fields := strings.Fields(rest)
		if len(fields) == 0 {
			return 0
		}
		kib, err := strconv.ParseInt(fields[0], 10, 64)
		if err != nil {
			return 0
		}
		return kib * 1024
	}
	return 0
}
