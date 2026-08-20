package database

import (
	"testing"
	"time"

	"github.com/gombit-dev/gombit/config"
)

type testWidget struct {
	ID   uint `gorm:"primaryKey"`
	Name string
}

func TestOpenSQLiteRoundTrip(t *testing.T) {
	db := openSQLite(t)

	if got := db.Driver(); got != DriverSQLite {
		t.Fatalf("Driver() = %q, want %q", got, DriverSQLite)
	}
	if got := db.Capabilities(); !got.Transactions || !got.ForeignKeyConstraints {
		t.Fatalf("Capabilities() = %#v, want sqlite transaction and FK support", got)
	}

	if err := db.AutoMigrate(&testWidget{}); err != nil {
		t.Fatalf("AutoMigrate() error = %v, want nil", err)
	}
	if err := db.Create(&testWidget{Name: "alpha"}).Error; err != nil {
		t.Fatalf("Create() error = %v, want nil", err)
	}

	var count int64
	if err := db.Model(&testWidget{}).Where("name = ?", "alpha").Count(&count).Error; err != nil {
		t.Fatalf("Count() error = %v, want nil", err)
	}
	if count != 1 {
		t.Fatalf("count = %d, want 1", count)
	}
}

func TestCloseClosesUnderlyingSQLDB(t *testing.T) {
	db := openSQLite(t)
	if err := db.Close(); err != nil {
		t.Fatalf("Close() error = %v, want nil", err)
	}
	if err := db.Exec("select 1").Error; err == nil {
		t.Fatal("Exec() after Close() error = nil, want error")
	}
}

func TestCapabilitiesFor(t *testing.T) {
	tests := []struct {
		name   string
		driver Driver
		want   Capabilities
	}{
		{
			name:   "sqlite",
			driver: DriverSQLite,
			want: Capabilities{
				Transactions:          true,
				Savepoints:            true,
				ForeignKeyConstraints: true,
				Returning:             true,
				Upsert:                true,
			},
		},
		{
			name:   "postgres",
			driver: DriverPostgres,
			want: Capabilities{
				Transactions:          true,
				Savepoints:            true,
				ForeignKeyConstraints: true,
				Returning:             true,
				Upsert:                true,
				AdvisoryLocks:         true,
				ConcurrentIndexBuilds: true,
			},
		},
		{
			name:   "mysql",
			driver: DriverMySQL,
			want: Capabilities{
				Transactions:          true,
				Savepoints:            true,
				ForeignKeyConstraints: true,
				Upsert:                true,
			},
		},
		{
			name:   "unknown",
			driver: "unknown",
			want:   Capabilities{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CapabilitiesFor(tt.driver); got != tt.want {
				t.Fatalf("CapabilitiesFor(%q) = %#v, want %#v", tt.driver, got, tt.want)
			}
		})
	}
}

func TestPoolConfigForUsesDriverDefaults(t *testing.T) {
	tests := []struct {
		name   string
		driver Driver
		want   poolConfig
	}{
		{
			name:   "sqlite",
			driver: DriverSQLite,
			want: poolConfig{
				maxOpenConns: 1,
				maxIdleConns: 1,
			},
		},
		{
			name:   "postgres",
			driver: DriverPostgres,
			want: poolConfig{
				maxOpenConns:    25,
				maxIdleConns:    5,
				connMaxLifetime: 30 * time.Minute,
			},
		},
		{
			name:   "mysql",
			driver: DriverMySQL,
			want: poolConfig{
				maxOpenConns:    25,
				maxIdleConns:    5,
				connMaxLifetime: 30 * time.Minute,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.DatabaseConfig{
				Driver: config.DatabaseDriver(tt.driver),
				DSN:    "unused",
			}
			if got := poolConfigFor(tt.driver, cfg); got != tt.want {
				t.Fatalf("poolConfigFor(%q) = %#v, want %#v", tt.driver, got, tt.want)
			}
		})
	}
}

func TestPoolConfigForUsesExplicitOverrides(t *testing.T) {
	cfg := config.DatabaseConfig{
		Driver:          config.DatabaseDriverPostgres,
		DSN:             "unused",
		MaxOpenConns:    12,
		MaxIdleConns:    3,
		ConnMaxLifetime: time.Hour,
	}

	want := poolConfig{
		maxOpenConns:    12,
		maxIdleConns:    3,
		connMaxLifetime: time.Hour,
	}
	if got := poolConfigFor(DriverPostgres, cfg); got != want {
		t.Fatalf("poolConfigFor() = %#v, want %#v", got, want)
	}
}

func TestOpenRejectsInvalidConfig(t *testing.T) {
	_, err := Open(config.DatabaseConfig{
		Driver: "oracle",
		DSN:    "unused",
	})
	if err == nil {
		t.Fatal("Open() error = nil, want invalid driver error")
	}
}

func openSQLite(t *testing.T) *DB {
	t.Helper()

	db, err := Open(config.DatabaseConfig{
		Driver: config.DatabaseDriverSQLite,
		DSN:    "file::memory:?cache=shared&_fk=1",
	})
	if err != nil {
		t.Fatalf("Open() error = %v, want nil", err)
	}
	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Fatalf("Close() error = %v, want nil", err)
		}
	})
	return db
}
