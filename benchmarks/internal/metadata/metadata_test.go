package metadata

import (
	"context"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestCollectUsesInjectedRunnerAndClock(t *testing.T) {
	fixedTime := time.Date(2026, 8, 26, 12, 0, 0, 0, time.UTC)
	run := func(_ context.Context, name string, args ...string) (string, error) {
		switch {
		case name == "git" && len(args) > 0 && args[0] == "rev-parse":
			return "abc123def456", nil
		case name == "git" && len(args) > 0 && args[0] == "status":
			return " M some/file.go", nil // non-empty -> dirty
		case name == "uname":
			return "6.1.0-test", nil
		case name == "docker" && args[0] == "version":
			return "27.0.1", nil
		case name == "docker" && args[0] == "compose":
			return "v2.29.0", nil
		default:
			return "", nil
		}
	}

	m := Collect(context.Background(), Options{
		Now:             func() time.Time { return fixedTime },
		Run:             run,
		PostgresVersion: "16.4",
		BenchmarkTool:   "k6 0.55.0",
		ResourceLimits:  "app 1cpu/512m; pg 2cpu/2g",
		DurationSeconds: 30,
		WarmupSeconds:   10,
		Concurrency:     []int{1, 10, 100},
		Trials:          5,
	})

	if m.SchemaVersion != SchemaVersion {
		t.Errorf("SchemaVersion = %d, want %d", m.SchemaVersion, SchemaVersion)
	}
	if m.Timestamp != "2026-08-26T12:00:00Z" {
		t.Errorf("Timestamp = %q, want the injected fixed time", m.Timestamp)
	}
	if m.GitCommit != "abc123def456" {
		t.Errorf("GitCommit = %q", m.GitCommit)
	}
	if m.GitDirty == nil || !*m.GitDirty {
		t.Errorf("GitDirty = %v, want a non-nil true (git status was non-empty)", m.GitDirty)
	}
	if m.Kernel != "6.1.0-test" {
		t.Errorf("Kernel = %q", m.Kernel)
	}
	if m.DockerVersion != "27.0.1" || m.DockerComposeVersion != "v2.29.0" {
		t.Errorf("docker versions = %q / %q", m.DockerVersion, m.DockerComposeVersion)
	}
	// Deterministic host fields come straight from the Go runtime.
	if m.OS != runtime.GOOS || m.Arch != runtime.GOARCH || m.LogicalCPUs != runtime.NumCPU() {
		t.Errorf("runtime fields = %q/%q/%d", m.OS, m.Arch, m.LogicalCPUs)
	}
	if m.GoVersion != runtime.Version() {
		t.Errorf("GoVersion = %q, want %q", m.GoVersion, runtime.Version())
	}
	// Provided run-parameter fields are copied through verbatim.
	if m.PostgresVersion != "16.4" || m.BenchmarkTool != "k6 0.55.0" || m.Trials != 5 {
		t.Errorf("provided fields not copied: %+v", m)
	}
	if len(m.Concurrency) != 3 || m.Concurrency[2] != 100 {
		t.Errorf("Concurrency = %v", m.Concurrency)
	}
}

func TestCollectCleanTreeAndMissingToolsDegrade(t *testing.T) {
	// A runner that returns "" for git status (clean) and errors for docker.
	run := func(_ context.Context, name string, args ...string) (string, error) {
		if name == "git" && args[0] == "status" {
			return "", nil
		}
		if name == "docker" {
			return "", context.DeadlineExceeded // tool "unavailable"
		}
		return "ok", nil
	}
	m := Collect(context.Background(), Options{Run: run})
	if m.GitDirty == nil || *m.GitDirty {
		t.Errorf("GitDirty = %v, want a non-nil false for an empty git status", m.GitDirty)
	}
	// Missing docker must leave the field empty, not fail collection.
	if m.DockerVersion != "" || m.DockerComposeVersion != "" {
		t.Errorf("docker versions should be empty when unavailable: %q/%q", m.DockerVersion, m.DockerComposeVersion)
	}
}

// git_dirty must reflect the SOURCE tree, excluding benchmarks/results/ — a
// suite writes result files for earlier apps before Collect runs for a later
// one, and that must not flag an otherwise-clean tree dirty.
func TestCollectExcludesResultsFromDirty(t *testing.T) {
	var statusArgs []string
	run := func(_ context.Context, name string, args ...string) (string, error) {
		if name == "git" && len(args) > 0 && args[0] == "status" {
			statusArgs = args
			// A runner honoring the exclude pathspec would report clean here;
			// simulate that (only benchmarks/results changed).
			return "", nil
		}
		return "", nil
	}
	m := Collect(context.Background(), Options{Run: run, Now: time.Now})
	if m.GitDirty == nil || *m.GitDirty {
		t.Errorf("GitDirty = %v, want non-nil false (results-only changes are not source dirt)", m.GitDirty)
	}
	joined := strings.Join(statusArgs, " ")
	if !strings.Contains(joined, "benchmarks/results") || !strings.Contains(joined, "exclude") {
		t.Errorf("git status must exclude benchmarks/results, got args: %v", statusArgs)
	}
}

// A failed `git status` (or missing git) must leave GitDirty unknown (nil),
// never fail-open to a clean-tree claim — even when rev-parse succeeded and a
// SHA was recorded.
func TestCollectGitStatusErrorIsUnknownNotClean(t *testing.T) {
	run := func(_ context.Context, name string, args ...string) (string, error) {
		if name == "git" && args[0] == "rev-parse" {
			return "deadbeef", nil
		}
		if name == "git" && args[0] == "status" {
			return "fatal: not a git repository", context.Canceled // status failed
		}
		return "", nil
	}
	m := Collect(context.Background(), Options{Run: run})
	if m.GitCommit != "deadbeef" {
		t.Errorf("GitCommit = %q, want the SHA rev-parse returned", m.GitCommit)
	}
	if m.GitDirty != nil {
		t.Errorf("GitDirty = %v, want nil (unknown) when git status failed", *m.GitDirty)
	}
}

func TestParseCPUModelArmFallback(t *testing.T) {
	// aarch64 /proc/cpuinfo: no "model name"; devicetree "Model" is the label.
	arm := "processor\t: 0\nBogoMIPS\t: 108.00\nCPU implementer\t: 0x41\nCPU part\t: 0xd0b\nModel\t\t: Raspberry Pi 5 Model B Rev 1.0\n"
	if got := parseCPUModel(arm); got != "Raspberry Pi 5 Model B Rev 1.0" {
		t.Errorf("parseCPUModel(arm) = %q, want the Model line", got)
	}
	// A bare ARM cpuinfo with only CPU part codes records unknown, not junk.
	bare := "processor\t: 0\nCPU implementer\t: 0x41\nCPU part\t: 0xd0b\n"
	if got := parseCPUModel(bare); got != "" {
		t.Errorf("parseCPUModel(bare arm) = %q, want empty (unknown)", got)
	}
}

func TestParseCPUModel(t *testing.T) {
	cpuinfo := "processor\t: 0\nvendor_id\t: GenuineIntel\nmodel name\t: Intel(R) Core(TM) i7-9750H CPU @ 2.60GHz\ncpu MHz\t: 2600\n"
	if got := parseCPUModel(cpuinfo); got != "Intel(R) Core(TM) i7-9750H CPU @ 2.60GHz" {
		t.Errorf("parseCPUModel = %q", got)
	}
	if got := parseCPUModel("no model here\n"); got != "" {
		t.Errorf("parseCPUModel(no model) = %q, want empty", got)
	}
}

func TestParseMemTotalBytes(t *testing.T) {
	meminfo := "MemTotal:       16311072 kB\nMemFree:         1234567 kB\n"
	if got := parseMemTotalBytes(meminfo); got != 16311072*1024 {
		t.Errorf("parseMemTotalBytes = %d, want %d", got, int64(16311072)*1024)
	}
	if got := parseMemTotalBytes("MemFree: 100 kB\n"); got != 0 {
		t.Errorf("parseMemTotalBytes(no MemTotal) = %d, want 0", got)
	}
}
