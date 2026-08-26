package main

import (
	"context"
	"fmt"

	"gorm.io/gorm"
)

// seedUserCount/seedProjectCount are the issue's recommended initial
// dataset (benchmarks/docs/schema.md "Seed dataset").
const (
	seedUserCount    = 1000
	seedProjectCount = 100000
	seedBatchSize    = 1000
)

// seedDatabase truncates and repopulates the canonical benchmark dataset at
// production scale. See seedDatabaseN.
func seedDatabase(ctx context.Context, db *gorm.DB) error {
	return seedDatabaseN(ctx, db, seedUserCount, seedProjectCount)
}

// seedDatabaseN is seedDatabase parameterized by row counts, so tests can
// exercise the real truncate-then-seed path (including idempotency) at a
// small scale instead of paying for a 100,000-row insert in CI.
// seedDatabase always calls this with the canonical
// seedUserCount/seedProjectCount; tests call it directly with small counts.
//
// Truncating first (RESTART IDENTITY) makes repeated invocations idempotent
// instead of accumulating duplicate data, and resets the users sequence to
// 1 so projectOwnerID's round-robin computation stays correct without
// reading back generated IDs.
func seedDatabaseN(ctx context.Context, db *gorm.DB, userCount, projectCount int) error {
	if err := db.WithContext(ctx).Exec("TRUNCATE TABLE projects, users RESTART IDENTITY CASCADE").Error; err != nil {
		return fmt.Errorf("truncate: %w", err)
	}
	if err := seedUsersN(ctx, db, userCount); err != nil {
		return err
	}
	return seedProjectsN(ctx, db, userCount, projectCount)
}

func seedUsersN(ctx context.Context, db *gorm.DB, userCount int) error {
	batch := make([]User, 0, seedBatchSize)
	for i := 1; i <= userCount; i++ {
		batch = append(batch, User{Email: userEmail(i), Name: userName(i)})
		if len(batch) == seedBatchSize || i == userCount {
			if err := db.WithContext(ctx).Create(&batch).Error; err != nil {
				return fmt.Errorf("seed users: %w", err)
			}
			batch = batch[:0]
		}
	}
	return nil
}

// seedProjectsN distributes projects round-robin across users (project i
// belongs to user projectOwnerID(i, userCount)), so every user owns
// projectCount/userCount projects and two implementations' seeded row N are
// content-identical.
func seedProjectsN(ctx context.Context, db *gorm.DB, userCount, projectCount int) error {
	batch := make([]Project, 0, seedBatchSize)
	for i := 1; i <= projectCount; i++ {
		batch = append(batch, Project{
			OwnerID:     projectOwnerID(i, userCount),
			Name:        projectName(i),
			Description: projectDescription(i),
		})
		if len(batch) == seedBatchSize || i == projectCount {
			if err := db.WithContext(ctx).Create(&batch).Error; err != nil {
				return fmt.Errorf("seed projects: %w", err)
			}
			batch = batch[:0]
		}
	}
	return nil
}

// userEmail, userName, projectOwnerID, projectName, and projectDescription
// are the pure, deterministic-content formulas issue #141's seed dataset
// requires ("the same values for every implementation"). Extracted to plain
// functions (rather than left inline in the batch-building loops) so
// TestSeedContentIsDeterministic can check the actual formulas directly,
// without a database, instead of only exercising them indirectly through a
// full seed run that no CI job executes.
func userEmail(i int) string { return fmt.Sprintf("user-%04d@example.com", i) }
func userName(i int) string  { return fmt.Sprintf("User %04d", i) }

func projectOwnerID(i, userCount int) uint {
	if i < 1 || userCount < 1 {
		panic("projectOwnerID: i and userCount must be positive")
	}
	return uint((i-1)%userCount + 1) //nolint:gosec // i,userCount > 0 is checked above; (i-1)%userCount+1 is always in [1,userCount]
}

func projectName(i int) string        { return fmt.Sprintf("Project %06d", i) }
func projectDescription(i int) string { return fmt.Sprintf("Seeded benchmark project %06d", i) }
