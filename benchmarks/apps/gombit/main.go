// Command gombit is the BENCH-1 Gombit-runtime implementation of the
// canonical /api/projects CRUD API (issue #141, benchmarks/docs/schema.md):
// a normal Gombit app — Huma handlers, GORM, framework.App — using Atlas
// migrations (`gombit db makemigrations`/`migrate`, not AutoMigrate;
// AGENTS.md D3) applied as a separate step before this binary runs, the
// same way a deployed Gombit app would. See
// benchmarks/apps/gombit/database/migrations/ and this directory's README
// for exact commands.
package main

import (
	"context"
	"flag"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/gombit-dev/gombit/benchmarks/apps/gombit/internal/project"
	"github.com/gombit-dev/gombit/config"
	"github.com/gombit-dev/gombit/database"
	"github.com/gombit-dev/gombit/framework"
)

func main() {
	seed := flag.Bool("seed", false, "seed the deterministic benchmark dataset and exit")
	flag.Parse()

	cfg := config.DefaultFor(config.EnvironmentProduction)
	cfg.HTTP.Addr = ":" + envOr("PORT", "8080")
	// Gombit's own default API.Prefix is "/api/v1"; the canonical route
	// spec (benchmarks/docs/schema.md, issue #141) and
	// benchmarks/apps/gin-gorm both use "/api" with no version segment.
	// Left at the framework default, this app's route surface wouldn't
	// match its own control's — a real fairness bug, not a style choice.
	cfg.API.Prefix = "/api"
	cfg.Database = config.DatabaseConfig{
		Driver: config.DatabaseDriverPostgres,
		DSN:    envOr("DATABASE_URL", "postgres://gombit:gombit@127.0.0.1:55432/gombit_bench_gombit?sslmode=disable"),
		// issue #141 "Connection pooling" pins 20/20 across every
		// implementation, overriding Gombit's own 25/5 Postgres default
		// (database/database.go) — the fairness pin is the issue's, not
		// the framework's, matching benchmarks/apps/gin-gorm/main.go.
		MaxOpenConns: envOrInt("POOL_MAX_OPEN", 20),
		MaxIdleConns: envOrInt("POOL_MAX_IDLE", 20),
	}

	db, err := database.Open(cfg.Database)
	if err != nil {
		log.Fatalf("gombit: open database: %v", err)
	}

	if *seed {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()
		if err := seedDatabase(ctx, db.DB); err != nil {
			log.Fatalf("gombit: seed: %v", err)
		}
		log.Println("gombit: seed complete")
		return
	}

	app, err := framework.New(framework.WithConfig(cfg), framework.WithDatabase(db))
	if err != nil {
		log.Fatalf("gombit: build app: %v", err)
	}

	project.Register(app)

	log.Printf("gombit: listening on %s", cfg.HTTP.Addr)
	if err := framework.Run(app); err != nil {
		log.Fatalf("gombit: serve: %v", err)
	}
}

func envOr(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func envOrInt(key string, fallback int) int {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}
