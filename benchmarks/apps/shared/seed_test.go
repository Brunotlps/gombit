package shared

import "testing"

// TestSeedContentIsDeterministic pins the exact deterministic-content
// formulas issue #141's seed dataset requires ("the same values for every
// implementation"). No database, no build tag.
func TestSeedContentIsDeterministic(t *testing.T) {
	tests := []struct {
		name string
		i    int
		want string
	}{
		{"user email first", 1, "user-0001@example.com"},
		{"user email mid", 42, "user-0042@example.com"},
		{"user email last", 1000, "user-1000@example.com"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := UserEmail(tt.i); got != tt.want {
				t.Errorf("UserEmail(%d) = %q, want %q", tt.i, got, tt.want)
			}
		})
	}

	if got, want := UserName(1), "User 0001"; got != want {
		t.Errorf("UserName(1) = %q, want %q", got, want)
	}
	if got, want := UserName(1000), "User 1000"; got != want {
		t.Errorf("UserName(1000) = %q, want %q", got, want)
	}

	if got, want := ProjectName(1), "Project 000001"; got != want {
		t.Errorf("ProjectName(1) = %q, want %q", got, want)
	}
	if got, want := ProjectDescription(1), "Seeded benchmark project 000001"; got != want {
		t.Errorf("ProjectDescription(1) = %q, want %q", got, want)
	}
}

// TestProjectOwnerIDRoundRobin pins the round-robin ownership formula: every
// user owns projectCount/userCount projects, project 1 belongs to user 1,
// and ownership wraps back to user 1 immediately after the last user — the
// property benchmarks/docs/schema.md's "byte-identical row N" claim depends
// on other implementations reproducing exactly.
func TestProjectOwnerIDRoundRobin(t *testing.T) {
	const userCount = 1000

	tests := []struct {
		project int
		want    uint
	}{
		{1, 1},
		{2, 2},
		{1000, 1000},
		{1001, 1}, // wraps back to user 1 right after the last user
		{1002, 2},
		{100000, 1000}, // last seeded project, per benchmarks/docs/schema.md
	}
	for _, tt := range tests {
		if got := ProjectOwnerID(tt.project, userCount); got != tt.want {
			t.Errorf("ProjectOwnerID(%d, %d) = %d, want %d", tt.project, userCount, got, tt.want)
		}
	}

	// Every user owns exactly projectCount/userCount projects across the
	// full canonical range -- not just spot-checked individual values.
	const projectCount = 100000
	counts := make(map[uint]int, userCount)
	for i := 1; i <= projectCount; i++ {
		counts[ProjectOwnerID(i, userCount)]++
	}
	if len(counts) != userCount {
		t.Fatalf("projects owned by %d distinct users, want %d", len(counts), userCount)
	}
	for owner, count := range counts {
		if count != projectCount/userCount {
			t.Fatalf("user %d owns %d projects, want %d", owner, count, projectCount/userCount)
		}
	}
}

// TestSeedCountsMatchIssueRecommendation pins SeedUserCount/SeedProjectCount
// to the issue's recommended dataset size, so a future "let's shrink this
// for CI speed" edit has to touch this test deliberately instead of
// silently drifting from what benchmarks/docs/schema.md documents.
func TestSeedCountsMatchIssueRecommendation(t *testing.T) {
	if SeedUserCount != 1000 {
		t.Errorf("SeedUserCount = %d, want 1000", SeedUserCount)
	}
	if SeedProjectCount != 100000 {
		t.Errorf("SeedProjectCount = %d, want 100000", SeedProjectCount)
	}
}
