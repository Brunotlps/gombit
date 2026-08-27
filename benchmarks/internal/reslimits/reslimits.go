// Package reslimits answers one question honestly: did a container actually
// receive the resource ceiling the benchmark intended for it?
//
// Issue #141 §7 pins every app to 2 vCPU / 1 GiB and PostgreSQL to 2 vCPU /
// 2 GiB, and is explicit that if a limit can't be enforced the suite must
// "detect and report that fact rather than silently pretending limits were
// applied." A budget written in compose.yml is an intention, not proof.
// Whether `deploy.resources.limits` takes effect depends on the engine/Compose
// version and the host's cgroup support: Docker Compose v2 applies the limits
// on a plain `docker compose up` (verified locally against Compose v5.x), but a
// Swarm-mode stack, an older Compose (v1, and early v2 that needed
// `--compatibility`), or a host missing the required cgroup controllers can
// leave the container unlimited. So the evidence is the *running container*,
// not the file. This package reads the limits Docker recorded for the container
// (via `docker inspect`'s HostConfig — NanoCpus and Memory, where 0 means
// unlimited). These are the daemon's post-adapt create config, not a read of
// `/sys/fs/cgroup`, but the daemon zeroes or rejects a limit it cannot apply,
// so reading them back is an honest signal that an intended ceiling was
// dropped. It classifies them against the intended budget, producing the honest
// string a run records as metadata.resource_limits (the `make benchmark-crud-all`
// loop wires it per app; standalone run-crud records its own honest default).
package reslimits

import (
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"
)

// Budget is an intended per-container ceiling (issue #141 §7).
type Budget struct {
	CPUs        float64 // whole vCPUs, e.g. 2
	MemoryBytes int64   // e.g. 1 GiB = 1073741824
}

// Applied is what a container actually received, read from `docker inspect`
// (.HostConfig.NanoCpus and .HostConfig.Memory). Docker uses 0 as the
// "unlimited" sentinel for both fields — what you get whenever the intended
// limit never reached the kernel.
type Applied struct {
	NanoCPUs    int64 `json:"nano_cpus"`
	MemoryBytes int64 `json:"memory_bytes"`
}

// CPUs is the applied CPU ceiling in whole vCPUs (0 means unlimited).
func (a Applied) CPUs() float64 { return float64(a.NanoCPUs) / 1e9 }

// Status is the overall enforcement verdict for one container.
type Status string

const (
	// Enforced: both CPU and memory match the intended budget.
	Enforced Status = "enforced"
	// NotApplied: neither limit was applied — the container runs unlimited,
	// so the compose.yml budget never reached the kernel.
	NotApplied Status = "not-applied"
	// Partial: some limits are applied and some are missing or wrong, so the
	// container is not running under the intended budget.
	Partial Status = "partial"
)

// cpuTolerance absorbs float noise in NanoCpus↔vCPUs (e.g. 2.0 vs 1.9999999).
const cpuTolerance = 1e-3

// dimVerdict is the per-dimension classification behind the overall Status.
type dimVerdict string

const (
	dimUnlimited dimVerdict = "unlimited" // 0 sentinel: no limit applied
	dimMatches   dimVerdict = "matches"   // limited to the intended value
	dimDiffers   dimVerdict = "differs"   // limited, but not to the intended value
)

// Report is the classification of one container against its budget, plus the
// human-readable, honest string for metadata.resource_limits.
type Report struct {
	Container   string  `json:"container,omitempty"`
	Budget      Budget  `json:"budget"`
	Applied     Applied `json:"applied"`
	Status      Status  `json:"status"`
	CPUVerdict  string  `json:"cpu"`    // e.g. "2.00 vCPU (intended 2.00)"
	MemVerdict  string  `json:"memory"` // e.g. "unlimited (intended 1.0 GiB)"
	Explanation string  `json:"explanation,omitempty"`
}

// Classify compares what a container actually received against the intended
// budget and returns the honest verdict.
func Classify(container string, budget Budget, applied Applied) Report {
	cpu := classifyDim(applied.CPUs(), budget.CPUs, func(a, b float64) bool {
		return math.Abs(a-b) <= cpuTolerance
	})
	mem := classifyDim(float64(applied.MemoryBytes), float64(budget.MemoryBytes), func(a, b float64) bool {
		return a == b
	})

	r := Report{
		Container:  container,
		Budget:     budget,
		Applied:    applied,
		CPUVerdict: fmt.Sprintf("%s (intended %.2f vCPU)", cpuDesc(applied), budget.CPUs),
		MemVerdict: fmt.Sprintf("%s (intended %s)", memDesc(applied.MemoryBytes), FormatBytes(budget.MemoryBytes)),
	}
	switch {
	case cpu == dimMatches && mem == dimMatches:
		r.Status = Enforced
	case cpu == dimUnlimited && mem == dimUnlimited:
		r.Status = NotApplied
		r.Explanation = "container runs unlimited; the intended compose limits were " +
			"not enforced by this engine/Compose — check the Compose version, Swarm " +
			"mode, and host cgroup support before trusting any numbers"
	default:
		r.Status = Partial
	}
	return r
}

func classifyDim(applied, intended float64, eq func(a, b float64) bool) dimVerdict {
	if applied == 0 {
		return dimUnlimited
	}
	if eq(applied, intended) {
		return dimMatches
	}
	return dimDiffers
}

func cpuDesc(a Applied) string {
	if a.NanoCPUs == 0 {
		return "unlimited"
	}
	return fmt.Sprintf("%.2f vCPU", a.CPUs())
}

func memDesc(b int64) string {
	if b == 0 {
		return "unlimited"
	}
	return FormatBytes(b)
}

// String is the honest one-line resource_limits value for run metadata.
func (r Report) String() string {
	switch r.Status {
	case Enforced:
		return fmt.Sprintf("enforced: cpu %s, memory %s", r.CPUVerdict, r.MemVerdict)
	case NotApplied:
		return fmt.Sprintf("not applied: cpu %s, memory %s (%s)", r.CPUVerdict, r.MemVerdict, r.Explanation)
	default:
		return fmt.Sprintf("partial: cpu %s, memory %s", r.CPUVerdict, r.MemVerdict)
	}
}

// ParseInspect reads the Applied limits from `docker inspect <container>`
// output (a JSON array; the first element is used).
func ParseInspect(data []byte) (Applied, error) {
	var arr []struct {
		HostConfig struct {
			NanoCpus int64 `json:"NanoCpus"`
			Memory   int64 `json:"Memory"`
		} `json:"HostConfig"`
	}
	if err := json.Unmarshal(data, &arr); err != nil {
		return Applied{}, fmt.Errorf("parse docker inspect: %w", err)
	}
	if len(arr) == 0 {
		return Applied{}, fmt.Errorf("docker inspect returned no containers")
	}
	return Applied{NanoCPUs: arr[0].HostConfig.NanoCpus, MemoryBytes: arr[0].HostConfig.Memory}, nil
}

// ParseBytes parses a compose-style memory string ("1g", "512m", "2048k", or a
// bare byte count) into bytes, using 1024-based units the way Docker does.
func ParseBytes(s string) (int64, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, fmt.Errorf("empty memory value")
	}
	mult := int64(1)
	switch unit := s[len(s)-1]; unit {
	case 'b', 'B':
		s = s[:len(s)-1]
	case 'k', 'K':
		mult, s = 1<<10, s[:len(s)-1]
	case 'm', 'M':
		mult, s = 1<<20, s[:len(s)-1]
	case 'g', 'G':
		mult, s = 1<<30, s[:len(s)-1]
	default:
		if unit < '0' || unit > '9' {
			return 0, fmt.Errorf("unknown memory unit %q in %q", string(unit), s)
		}
	}
	n, err := strconv.ParseInt(strings.TrimSpace(s), 10, 64)
	if err != nil {
		return 0, fmt.Errorf("parse memory %q: %w", s, err)
	}
	if n < 0 {
		return 0, fmt.Errorf("negative memory value %q", s)
	}
	return n * mult, nil
}

// FormatBytes renders a byte count as a compact 1024-based string (GiB/MiB/KiB)
// for human verdicts; whole units drop the fraction.
func FormatBytes(b int64) string {
	switch {
	case b == 0:
		return "0"
	case b >= 1<<30:
		return trimUnit(float64(b)/(1<<30), "GiB")
	case b >= 1<<20:
		return trimUnit(float64(b)/(1<<20), "MiB")
	case b >= 1<<10:
		return trimUnit(float64(b)/(1<<10), "KiB")
	default:
		return fmt.Sprintf("%d B", b)
	}
}

func trimUnit(v float64, unit string) string {
	if v == math.Trunc(v) {
		return fmt.Sprintf("%.0f %s", v, unit)
	}
	return fmt.Sprintf("%.1f %s", v, unit)
}
