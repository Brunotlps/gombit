package migrations

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/LAA-Software-Engineering/gombit/config"
	"github.com/LAA-Software-Engineering/gombit/database"
)

func TestParseMigrationFilename(t *testing.T) {
	tests := []struct {
		name    string
		file    string
		version string
		migName string
		wantErr bool
	}{
		{name: "valid", file: "20260101000000_create_products.sql", version: "20260101000000", migName: "create_products"},
		{name: "with path", file: "/tmp/20260101000000_create_products.sql", version: "20260101000000", migName: "create_products"},
		{name: "down rejected", file: "20260101000000_create_products.down.sql", wantErr: true},
		{name: "atlas sum", file: "atlas.sum", wantErr: true},
		{name: "invalid", file: "create_products.sql", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			version, migName, err := ParseMigrationFilename(tt.file)
			if tt.wantErr {
				if err == nil {
					t.Fatal("ParseMigrationFilename() error = nil, want error")
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseMigrationFilename() error = %v", err)
			}
			if version != tt.version || migName != tt.migName {
				t.Fatalf("ParseMigrationFilename() = (%q, %q), want (%q, %q)", version, migName, tt.version, tt.migName)
			}
		})
	}
}

func TestParseDownFilenameAndDownFilename(t *testing.T) {
	version, name, err := ParseDownFilename("20260101000000_create_products.down.sql")
	if err != nil {
		t.Fatalf("ParseDownFilename() error = %v", err)
	}
	if version != "20260101000000" || name != "create_products" {
		t.Fatalf("ParseDownFilename() = (%q, %q)", version, name)
	}

	got, err := DownFilename("20260101000000_create_products.sql")
	if err != nil {
		t.Fatalf("DownFilename() error = %v", err)
	}
	if got != "20260101000000_create_products.down.sql" {
		t.Fatalf("DownFilename() = %q", got)
	}
}

func TestListMigrationFiles(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "20260102000000_add_index.sql"), "CREATE INDEX;")
	writeFile(t, filepath.Join(dir, "20260101000000_create_products.sql"), "CREATE TABLE products;")
	writeFile(t, filepath.Join(dir, "20260101000000_create_products.down.sql"), "DROP TABLE products;")
	writeFile(t, filepath.Join(dir, "atlas.sum"), "h1:fake")
	writeFile(t, filepath.Join(dir, "readme.txt"), "ignore")

	files, err := ListMigrationFiles(dir)
	if err != nil {
		t.Fatalf("ListMigrationFiles() error = %v", err)
	}
	if len(files) != 2 {
		t.Fatalf("ListMigrationFiles() len = %d, want 2", len(files))
	}
	if files[0].Version != "20260101000000" || files[0].DownPath == "" {
		t.Fatalf("first file = %#v, want version with down path", files[0])
	}
	if files[1].Version != "20260102000000" || files[1].DownPath != "" {
		t.Fatalf("second file = %#v, want version without down path", files[1])
	}
}

func TestRevisionsCRUD(t *testing.T) {
	db := openSQLite(t)
	if err := ensureRevisionsTable(db.DB); err != nil {
		t.Fatalf("ensureRevisionsTable() error = %v", err)
	}

	batch, err := nextBatch(db.DB)
	if err != nil {
		t.Fatalf("nextBatch() error = %v", err)
	}
	if batch != 1 {
		t.Fatalf("nextBatch() = %d, want 1", batch)
	}

	now := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	files := []MigrationFile{
		{Version: "20260101000000", Name: "create_products"},
		{Version: "20260102000000", Name: "add_index"},
	}
	if err := insertBatch(db.DB, 1, files, now); err != nil {
		t.Fatalf("insertBatch() error = %v", err)
	}

	revisions, err := listRevisions(db.DB)
	if err != nil {
		t.Fatalf("listRevisions() error = %v", err)
	}
	if len(revisions) != 2 {
		t.Fatalf("listRevisions() len = %d, want 2", len(revisions))
	}
	if revisions[0].Batch != 1 || revisions[0].AppliedAt.UTC() != now {
		t.Fatalf("revision[0] = %#v", revisions[0])
	}

	gotBatch, last, err := lastBatch(db.DB)
	if err != nil {
		t.Fatalf("lastBatch() error = %v", err)
	}
	if gotBatch != 1 || len(last) != 2 || last[0].Version != "20260102000000" {
		t.Fatalf("lastBatch() = %d %#v", gotBatch, last)
	}

	if err := deleteRevisions(db.DB, []string{"20260101000000", "20260102000000"}); err != nil {
		t.Fatalf("deleteRevisions() error = %v", err)
	}
	revisions, err = listRevisions(db.DB)
	if err != nil {
		t.Fatalf("listRevisions() after delete error = %v", err)
	}
	if len(revisions) != 0 {
		t.Fatalf("listRevisions() after delete len = %d, want 0", len(revisions))
	}
}

func openSQLite(t *testing.T) *database.DB {
	t.Helper()
	db, err := database.Open(config.DatabaseConfig{
		Driver: config.DatabaseDriverSQLite,
		DSN:    "file:" + filepath.Join(t.TempDir(), "test.db") + "?cache=shared&_fk=1",
	})
	if err != nil {
		t.Fatalf("database.Open() error = %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
	})
	return db
}
