//go:build integration

package main

import (
	"context"
	"flag"
	"testing"

	"github.com/gombit-dev/gombit/benchmarks/apps/gombit/internal/project"
	"github.com/gombit-dev/gombit/benchmarks/apps/shared"
	"github.com/gombit-dev/gombit/config"
	"github.com/gombit-dev/gombit/database"
)

// Run against a real, already-Atlas-migrated PostgreSQL instance — see
// internal/project/handler_test.go's doc comment for the exact setup.
var databaseDSN = flag.String("database.dsn", "", "PostgreSQL DSN for gombit seed tests (schema already migrated)")

func testDB(t *testing.T) *database.DB {
	t.Helper()

	if *databaseDSN == "" {
		t.Skip("set -database.dsn to run gombit seed tests")
	}

	db, err := database.Open(config.DatabaseConfig{
		Driver: config.DatabaseDriverPostgres,
		DSN:    *databaseDSN,
	})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

// TestSeedDatabaseNIsIdempotentAndCorrect mirrors
// benchmarks/apps/gin-gorm/main_test.go's test of the same name exactly:
// exercises the real truncate-then-seed path (seedDatabaseN, which
// seedDatabase calls at production scale) at a small scale against real
// PostgreSQL — exact row counts, deterministic content for known rows,
// round-robin ownership, and — running it twice — that repeated seeding
// truncates rather than accumulates duplicate data. An earlier review round
// added this test for gin-gorm specifically because the seed contract had
// no automated coverage; this app copied seedDatabaseN without also
// copying the test that earns it.
func TestSeedDatabaseNIsIdempotentAndCorrect(t *testing.T) {
	db := testDB(t)
	const userCount, projectCount = 7, 23 // deliberately not a multiple, to exercise the round-robin remainder

	for run := 1; run <= 2; run++ {
		if err := seedDatabaseN(context.Background(), db.DB, userCount, projectCount); err != nil {
			t.Fatalf("run %d: seedDatabaseN: %v", run, err)
		}

		var userTotal, projectTotal int64
		if err := db.Model(&project.User{}).Count(&userTotal).Error; err != nil {
			t.Fatalf("run %d: count users: %v", run, err)
		}
		if err := db.Model(&project.Project{}).Count(&projectTotal).Error; err != nil {
			t.Fatalf("run %d: count projects: %v", run, err)
		}
		if userTotal != userCount {
			t.Fatalf("run %d: user count = %d, want %d (seed did not truncate before reseeding)", run, userTotal, userCount)
		}
		if projectTotal != projectCount {
			t.Fatalf("run %d: project count = %d, want %d (seed did not truncate before reseeding)", run, projectTotal, projectCount)
		}

		var firstUser project.User
		if err := db.First(&firstUser, 1).Error; err != nil {
			t.Fatalf("run %d: load user 1: %v", run, err)
		}
		if firstUser.Email != shared.UserEmail(1) || firstUser.Name != shared.UserName(1) {
			t.Fatalf("run %d: user 1 = %+v, want email=%s name=%s", run, firstUser, shared.UserEmail(1), shared.UserName(1))
		}

		// Project userCount+1 (8th project, userCount=7) is the first to
		// wrap back to owner 1 — the round-robin boundary a naive off-by-one
		// would break silently.
		var wrapped project.Project
		if err := db.First(&wrapped, userCount+1).Error; err != nil {
			t.Fatalf("run %d: load project %d: %v", run, userCount+1, err)
		}
		if wrapped.OwnerID != 1 {
			t.Fatalf("run %d: project %d owner = %d, want 1 (round-robin wrap)", run, userCount+1, wrapped.OwnerID)
		}
		if wrapped.Name != shared.ProjectName(userCount+1) {
			t.Fatalf("run %d: project %d name = %q, want %q", run, userCount+1, wrapped.Name, shared.ProjectName(userCount+1))
		}
	}
}
