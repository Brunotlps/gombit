package shared

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// SeedUserCount and SeedProjectCount are issue #141's recommended initial
// dataset (benchmarks/docs/schema.md "Seed dataset"), shared so every
// implementation seeds the same row counts without two copies of these
// numbers to keep in sync by hand.
const (
	SeedUserCount    = 1000
	SeedProjectCount = 100000
)

// SeedUsersEnv and SeedProjectsEnv override the seeded row counts for a run.
// They exist for the CI smoke's small deterministic seed (issue #141 §11): a
// tiny row count over the SAME deterministic content formulas, so the
// containerized migrate → seed → serve → k6 path is exercised on every PR
// without a 100,000-row insert. Every implementation (Go, Python, Ruby, PHP,
// Node) reads these exact names with the same semantics — unset/empty means the
// canonical default; a non-empty value that is not a positive integer is a
// fatal error, never a silent fall back to 100k. Note the read workload asserts
// a full 20-row first page, so an override must keep at least 20 projects.
const (
	SeedUsersEnv    = "BENCH_SEED_USERS"
	SeedProjectsEnv = "BENCH_SEED_PROJECTS"
)

// SeedCounts resolves the user and project counts to seed: the canonical
// constants, unless SeedUsersEnv / SeedProjectsEnv are set to a positive
// integer. A malformed override is returned as an error rather than silently
// falling back, so a typo in CI fails loudly instead of seeding 100k.
func SeedCounts() (users, projects int, err error) {
	if users, err = seedCount(SeedUsersEnv, SeedUserCount); err != nil {
		return 0, 0, err
	}
	if projects, err = seedCount(SeedProjectsEnv, SeedProjectCount); err != nil {
		return 0, 0, err
	}
	return users, projects, nil
}

func seedCount(env string, def int) (int, error) {
	raw := strings.TrimSpace(os.Getenv(env))
	if raw == "" {
		return def, nil
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n < 1 {
		return 0, fmt.Errorf("%s=%q: must be a positive integer", env, raw)
	}
	return n, nil
}

// UserEmail, UserName, ProjectOwnerID, ProjectName, and ProjectDescription
// are the pure, deterministic-content formulas issue #141's seed dataset
// requires ("the same values for every implementation"). Shared across
// every Go implementation under benchmarks/apps/ so two of them can't
// silently seed different content for the same row N — the exact drift
// risk a hand-duplicated second copy would carry (this package already
// exists to avoid that for response shapes; seed content is the same
// problem).
func UserEmail(i int) string { return fmt.Sprintf("user-%04d@example.com", i) }
func UserName(i int) string  { return fmt.Sprintf("User %04d", i) }

// ProjectOwnerID returns the 1-based owner id for project i, round-robin
// over 1..userCount, so every user owns projectCount/userCount projects and
// two implementations' seeded row N are content-identical.
func ProjectOwnerID(i, userCount int) uint {
	if i < 1 || userCount < 1 {
		panic("shared.ProjectOwnerID: i and userCount must be positive")
	}
	return uint((i-1)%userCount + 1) //nolint:gosec // i,userCount > 0 is checked above; (i-1)%userCount+1 is always in [1,userCount]
}

func ProjectName(i int) string        { return fmt.Sprintf("Project %06d", i) }
func ProjectDescription(i int) string { return fmt.Sprintf("Seeded benchmark project %06d", i) }
