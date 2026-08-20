//go:build integration

package database

import (
	"flag"
	"testing"

	"github.com/gombit-dev/gombit/config"
)

var (
	postgresDSN = flag.String("database.postgres-dsn", "", "PostgreSQL DSN for database integration tests")
	mysqlDSN    = flag.String("database.mysql-dsn", "", "MySQL DSN for database integration tests")
)

func TestOpenPostgresRoundTrip(t *testing.T) {
	if *postgresDSN == "" {
		t.Skip("set -database.postgres-dsn to run Postgres integration tests")
	}

	testOpenRoundTrip(t, config.DatabaseConfig{
		Driver: config.DatabaseDriverPostgres,
		DSN:    *postgresDSN,
	}, DriverPostgres)
}

func TestOpenMySQLRoundTrip(t *testing.T) {
	if *mysqlDSN == "" {
		t.Skip("set -database.mysql-dsn to run MySQL integration tests")
	}

	testOpenRoundTrip(t, config.DatabaseConfig{
		Driver: config.DatabaseDriverMySQL,
		DSN:    *mysqlDSN,
	}, DriverMySQL)
}

func testOpenRoundTrip(t *testing.T, cfg config.DatabaseConfig, wantDriver Driver) {
	t.Helper()

	db, err := Open(cfg)
	if err != nil {
		t.Fatalf("Open() error = %v, want nil", err)
	}
	t.Cleanup(func() {
		_ = db.Migrator().DropTable(&testWidget{})
		if err := db.Close(); err != nil {
			t.Fatalf("Close() error = %v, want nil", err)
		}
	})

	if got := db.Driver(); got != wantDriver {
		t.Fatalf("Driver() = %q, want %q", got, wantDriver)
	}
	if err := db.AutoMigrate(&testWidget{}); err != nil {
		t.Fatalf("AutoMigrate() error = %v, want nil", err)
	}
	if err := db.Create(&testWidget{Name: string(wantDriver)}).Error; err != nil {
		t.Fatalf("Create() error = %v, want nil", err)
	}

	var count int64
	if err := db.Model(&testWidget{}).Where("name = ?", string(wantDriver)).Count(&count).Error; err != nil {
		t.Fatalf("Count() error = %v, want nil", err)
	}
	if count != 1 {
		t.Fatalf("count = %d, want 1", count)
	}
}
