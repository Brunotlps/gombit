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

// seedDatabase truncates and repopulates the canonical benchmark dataset:
// deterministic content so every implementation seeds byte-identical rows,
// never inside a timed benchmark run. Truncating first (RESTART IDENTITY)
// makes repeated `-seed` invocations idempotent instead of accumulating
// duplicate data, and resets the users sequence to 1 so seedProjects'
// ownerID computation (round-robin over 1..seedUserCount) stays correct
// without reading back generated IDs.
func seedDatabase(ctx context.Context, db *gorm.DB) error {
	if err := db.WithContext(ctx).Exec("TRUNCATE TABLE projects, users RESTART IDENTITY CASCADE").Error; err != nil {
		return fmt.Errorf("truncate: %w", err)
	}
	if err := seedUsers(ctx, db); err != nil {
		return err
	}
	return seedProjects(ctx, db)
}

func seedUsers(ctx context.Context, db *gorm.DB) error {
	batch := make([]User, 0, seedBatchSize)
	for i := 1; i <= seedUserCount; i++ {
		batch = append(batch, User{
			Email: fmt.Sprintf("user-%04d@example.com", i),
			Name:  fmt.Sprintf("User %04d", i),
		})
		if len(batch) == seedBatchSize || i == seedUserCount {
			if err := db.WithContext(ctx).Create(&batch).Error; err != nil {
				return fmt.Errorf("seed users: %w", err)
			}
			batch = batch[:0]
		}
	}
	return nil
}

// seedProjects distributes projects round-robin across users (project i
// belongs to user ((i-1) % seedUserCount) + 1), so every user owns
// seedProjectCount/seedUserCount projects and two implementations' seeded
// row N are content-identical.
func seedProjects(ctx context.Context, db *gorm.DB) error {
	batch := make([]Project, 0, seedBatchSize)
	for i := 1; i <= seedProjectCount; i++ {
		ownerID := uint((i-1)%seedUserCount + 1)
		batch = append(batch, Project{
			OwnerID:     ownerID,
			Name:        fmt.Sprintf("Project %06d", i),
			Description: fmt.Sprintf("Seeded benchmark project %06d", i),
		})
		if len(batch) == seedBatchSize || i == seedProjectCount {
			if err := db.WithContext(ctx).Create(&batch).Error; err != nil {
				return fmt.Errorf("seed projects: %w", err)
			}
			batch = batch[:0]
		}
	}
	return nil
}
