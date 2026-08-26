package shared

import "fmt"

// SeedUserCount and SeedProjectCount are issue #141's recommended initial
// dataset (benchmarks/docs/schema.md "Seed dataset"), shared so every
// implementation seeds the same row counts without two copies of these
// numbers to keep in sync by hand.
const (
	SeedUserCount    = 1000
	SeedProjectCount = 100000
)

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
