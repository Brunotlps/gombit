package main

import (
	"context"
	"fmt"

	"github.com/gombit-dev/gombit/benchmarks/apps/shared"
	"gorm.io/gorm"
)

// seedBatchSize is a gin-gorm-specific implementation detail (how many rows
// per INSERT), not part of the cross-implementation seed contract, so it
// stays local rather than moving to benchmarks/apps/shared with the
// content formulas and row counts.
const seedBatchSize = 1000

// seedDatabase truncates and repopulates the canonical benchmark dataset at
// production scale. See seedDatabaseN.
func seedDatabase(ctx context.Context, db *gorm.DB) error {
	return seedDatabaseN(ctx, db, shared.SeedUserCount, shared.SeedProjectCount)
}

// seedDatabaseN is seedDatabase parameterized by row counts, so tests can
// exercise the real truncate-then-seed path (including idempotency) at a
// small scale instead of paying for a 100,000-row insert in CI.
// seedDatabase always calls this with the canonical
// shared.SeedUserCount/SeedProjectCount; tests call it directly with small
// counts.
//
// Truncating first (RESTART IDENTITY) makes repeated invocations idempotent
// instead of accumulating duplicate data, and resets the users sequence to
// 1 so shared.ProjectOwnerID's round-robin computation stays correct
// without reading back generated IDs.
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
		batch = append(batch, User{Email: shared.UserEmail(i), Name: shared.UserName(i)})
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
// belongs to user shared.ProjectOwnerID(i, userCount)), so every user owns
// projectCount/userCount projects and two implementations' seeded row N are
// content-identical.
func seedProjectsN(ctx context.Context, db *gorm.DB, userCount, projectCount int) error {
	batch := make([]Project, 0, seedBatchSize)
	for i := 1; i <= projectCount; i++ {
		batch = append(batch, Project{
			OwnerID:     shared.ProjectOwnerID(i, userCount),
			Name:        shared.ProjectName(i),
			Description: shared.ProjectDescription(i),
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
