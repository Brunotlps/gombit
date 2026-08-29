package framework

import (
	"context"
	"errors"
	"testing"

	"gorm.io/gorm"

	"github.com/gombit-dev/gombit/config"
	"github.com/gombit-dev/gombit/database"
)

type txRow struct {
	ID   uint `gorm:"primaryKey"`
	Name string
}

func newDBApp(t *testing.T) *App {
	t.Helper()
	db, err := database.Open(config.DatabaseConfig{
		Driver: config.DatabaseDriverSQLite,
		DSN:    "file::memory:?cache=shared&_fk=1",
	})
	if err != nil {
		t.Fatalf("database.Open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := db.AutoMigrate(&txRow{}); err != nil {
		t.Fatalf("AutoMigrate: %v", err)
	}
	return newTestApp(t, WithDatabase(db))
}

func TestTxCommitsOnSuccess(t *testing.T) {
	app := newDBApp(t)
	err := app.Tx(context.Background(), func(tx *gorm.DB) error {
		if err := tx.Create(&txRow{Name: "a"}).Error; err != nil {
			return err
		}
		return tx.Create(&txRow{Name: "b"}).Error
	})
	if err != nil {
		t.Fatalf("Tx error = %v, want nil", err)
	}
	var count int64
	app.DB().Model(&txRow{}).Count(&count)
	if count != 2 {
		t.Fatalf("row count = %d, want 2", count)
	}
}

func TestTxRollsBackOnError(t *testing.T) {
	app := newDBApp(t)
	sentinel := errors.New("boom")
	err := app.Tx(context.Background(), func(tx *gorm.DB) error {
		if err := tx.Create(&txRow{Name: "a"}).Error; err != nil {
			return err
		}
		return sentinel
	})
	if !errors.Is(err, sentinel) {
		t.Fatalf("Tx error = %v, want sentinel", err)
	}
	var count int64
	app.DB().Model(&txRow{}).Count(&count)
	if count != 0 {
		t.Fatalf("row count = %d, want 0 (transaction must roll back)", count)
	}
}

func TestTxNoDatabase(t *testing.T) {
	app := newTestApp(t)
	err := app.Tx(context.Background(), func(*gorm.DB) error { return nil })
	if err == nil {
		t.Fatal("Tx without database error = nil, want error")
	}
}
