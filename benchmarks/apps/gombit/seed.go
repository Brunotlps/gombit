package main

import (
	"context"
	"fmt"

	"github.com/gombit-dev/gombit/benchmarks/apps/gombit/internal/project"
	"github.com/gombit-dev/gombit/benchmarks/apps/shared"
	"gorm.io/gorm"
)

// seedBatchSize is an implementation detail of this app's seeder (how many
// rows per INSERT), not part of the cross-implementation seed contract, so
// it stays local rather than living in benchmarks/apps/shared with the
// content formulas and row counts — matching benchmarks/apps/gin-gorm/seed.go.
const seedBatchSize = 1000

// seedDatabase truncates and repopulates the canonical benchmark dataset at
// production scale using benchmarks/apps/shared's deterministic content
// formulas, the same ones benchmarks/apps/gin-gorm/seed.go uses, so the two
// implementations' seeded row N are content-identical. See
// benchmarks/apps/gin-gorm/seed.go's seedDatabaseN for the truncate/identity
// reasoning this mirrors.
func seedDatabase(ctx context.Context, db *gorm.DB) error {
	return seedDatabaseN(ctx, db, shared.SeedUserCount, shared.SeedProjectCount)
}

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
	batch := make([]project.User, 0, seedBatchSize)
	for i := 1; i <= userCount; i++ {
		batch = append(batch, project.User{Email: shared.UserEmail(i), Name: shared.UserName(i)})
		if len(batch) == seedBatchSize || i == userCount {
			if err := db.WithContext(ctx).Create(&batch).Error; err != nil {
				return fmt.Errorf("seed users: %w", err)
			}
			batch = batch[:0]
		}
	}
	return nil
}

func seedProjectsN(ctx context.Context, db *gorm.DB, userCount, projectCount int) error {
	batch := make([]project.Project, 0, seedBatchSize)
	for i := 1; i <= projectCount; i++ {
		batch = append(batch, project.Project{
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
