// Command gin-gorm is the BENCH-1 primary framework-tax control (issue #141
// "Gin + GORM ... This comparison is especially important because it
// isolates the incremental cost of adopting Gombit rather than changing
// programming languages"): the canonical /api/projects CRUD API
// (benchmarks/docs/schema.md) implemented with idiomatic Gin + GORM,
// deliberately without Huma or framework.App.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {
	seed := flag.Bool("seed", false, "seed the deterministic benchmark dataset and exit")
	flag.Parse()

	db, err := openDB()
	if err != nil {
		log.Fatalf("gin-gorm: open database: %v", err)
	}

	if err := db.AutoMigrate(&User{}, &Project{}); err != nil {
		log.Fatalf("gin-gorm: migrate: %v", err)
	}

	if *seed {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()
		if err := seedDatabase(ctx, db); err != nil {
			log.Fatalf("gin-gorm: seed: %v", err)
		}
		log.Println("gin-gorm: seed complete")
		return
	}

	gin.SetMode(gin.ReleaseMode)
	router := newRouter(db)

	addr := ":" + envOr("PORT", "8081")
	log.Printf("gin-gorm: listening on %s", addr)
	if err := router.Run(addr); err != nil {
		log.Fatalf("gin-gorm: serve: %v", err)
	}
}

// openDB connects with the same pool-limit fairness the issue requires
// across every implementation (issue #141 "Connection pooling": "max open
// connections: 20, max idle connections: 20"). Gombit's own database
// package defaults Postgres to 25/5 (database/database.go) — this
// benchmark app intentionally overrides that default rather than
// inheriting it, since the fairness requirement is pinned by the issue,
// not by whatever either framework ships as its own default.
func openDB() (*gorm.DB, error) {
	dsn := envOr("DATABASE_URL", "postgres://gombit:gombit@127.0.0.1:55432/gombit_bench?sslmode=disable")

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("open: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("underlying sql.DB: %w", err)
	}
	maxOpen := envOrInt("POOL_MAX_OPEN", 20)
	maxIdle := envOrInt("POOL_MAX_IDLE", 20)
	sqlDB.SetMaxOpenConns(maxOpen)
	sqlDB.SetMaxIdleConns(maxIdle)
	sqlDB.SetConnMaxLifetime(30 * time.Minute)

	return db, nil
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
