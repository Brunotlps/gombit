//go:build integration

package auth_test

import (
	"flag"
	"testing"

	"github.com/LAA-Software-Engineering/gombit/auth"
	"github.com/LAA-Software-Engineering/gombit/config"
	"github.com/LAA-Software-Engineering/gombit/database"
	"github.com/gin-gonic/gin"
)

var (
	postgresDSN = flag.String("auth.postgres-dsn", "", "PostgreSQL DSN for auth integration tests")
	mysqlDSN    = flag.String("auth.mysql-dsn", "", "MySQL DSN for auth integration tests")
)

func TestE2EPostgres(t *testing.T) {
	if *postgresDSN == "" {
		t.Skip("set -auth.postgres-dsn to run Postgres auth integration tests")
	}
	gin.SetMode(gin.TestMode)
	db := openDriver(t, config.DatabaseDriverPostgres, *postgresDSN)
	app := newAuthAppWithDB(t, db)
	runBearerE2E(t, app)
}

func TestE2EMySQL(t *testing.T) {
	if *mysqlDSN == "" {
		t.Skip("set -auth.mysql-dsn to run MySQL auth integration tests")
	}
	gin.SetMode(gin.TestMode)
	db := openDriver(t, config.DatabaseDriverMySQL, *mysqlDSN)
	app := newAuthAppWithDB(t, db)
	runBearerE2E(t, app)
}

func openDriver(t *testing.T, driver config.DatabaseDriver, dsn string) *database.DB {
	t.Helper()
	db, err := database.Open(config.DatabaseConfig{Driver: driver, DSN: dsn})
	if err != nil {
		t.Fatalf("database.Open() error = %v", err)
	}
	t.Cleanup(func() {
		_ = db.Migrator().DropTable(auth.Models()...)
		_ = db.Close()
	})
	return db
}
