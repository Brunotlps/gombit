package migrations

import (
	"bytes"
	"context"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gombit-dev/gombit/config"
	"github.com/gombit-dev/gombit/database"
)

func TestMigrateStatusRollbackAtlasCLISQLiteWhenAvailable(t *testing.T) {
	atlasBin := os.Getenv("ATLAS_BINARY")
	if atlasBin == "" {
		var err error
		atlasBin, err = exec.LookPath("atlas")
		if err != nil {
			t.Skip("Atlas CLI not found; set ATLAS_BINARY to run the real SQLite migrate smoke test")
		}
	}

	workDir := t.TempDir()
	migrationDir := filepath.Join(workDir, "database", "migrations")
	if err := os.MkdirAll(migrationDir, 0o750); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	writeFile(t, filepath.Join(migrationDir, "20260101000000_create_widgets.sql"),
		"-- Create \"widgets\" table\nCREATE TABLE `widgets` (`id` integer NULL PRIMARY KEY AUTOINCREMENT, `name` text NOT NULL);\n")
	writeFile(t, filepath.Join(migrationDir, "downs", "20260101000000_create_widgets.down.sql"),
		"DROP TABLE IF EXISTS `widgets`;\n")

	hashArgs := []string{
		"migrate",
		"hash",
		"--dir",
		"file://" + filepath.ToSlash(migrationDir),
	}
	var hashErr bytes.Buffer
	if err := (execRunner{}).Run(context.Background(), workDir, atlasBin, hashArgs, io.Discard, &hashErr); err != nil {
		t.Fatalf("atlas migrate hash: %v\n%s", err, hashErr.String())
	}

	dbPath := filepath.Join(workDir, "app.db")
	cfg := config.DatabaseConfig{
		Driver: config.DatabaseDriverSQLite,
		DSN:    "file:" + dbPath + "?cache=shared&_fk=1",
	}
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	opts := ApplyOptions{
		WorkDir:      workDir,
		MigrationDir: "database/migrations",
		AtlasBinary:  atlasBin,
		Database:     cfg,
		Stdout:       stdout,
		Stderr:       stderr,
	}

	if err := Migrate(context.Background(), opts); err != nil {
		t.Fatalf("Migrate() error = %v; stderr=%q", err, stderr.String())
	}
	if !strings.Contains(stdout.String(), "Applied 1 migration(s) in batch 1") {
		t.Fatalf("migrate stdout = %q", stdout.String())
	}

	db, err := database.Open(cfg)
	if err != nil {
		t.Fatalf("database.Open() error = %v", err)
	}
	defer func() { _ = db.Close() }()
	if !db.Migrator().HasTable("widgets") {
		t.Fatal("expected widgets table after real atlas apply")
	}
	if !db.Migrator().HasTable(atlasRevisionsTable) {
		t.Fatal("expected atlas_schema_revisions after real atlas apply")
	}
	revisions, err := listRevisions(db.DB)
	if err != nil {
		t.Fatalf("listRevisions() error = %v", err)
	}
	if len(revisions) != 1 || revisions[0].Version != "20260101000000" {
		t.Fatalf("revisions = %#v", revisions)
	}

	stdout.Reset()
	stderr.Reset()
	if err := Status(context.Background(), opts); err != nil {
		t.Fatalf("Status() error = %v; stderr=%q", err, stderr.String())
	}
	if !strings.Contains(stdout.String(), "applied") || !strings.Contains(stdout.String(), "20260101000000") {
		t.Fatalf("status stdout = %q", stdout.String())
	}
	if !strings.Contains(stdout.String(), "Atlas migration status") {
		t.Fatalf("status stdout = %q, want atlas status section", stdout.String())
	}

	stdout.Reset()
	if err := Rollback(context.Background(), opts); err != nil {
		t.Fatalf("Rollback() error = %v", err)
	}
	if db.Migrator().HasTable("widgets") {
		t.Fatal("widgets table should be gone after rollback")
	}
	revisions, err = listRevisions(db.DB)
	if err != nil {
		t.Fatalf("listRevisions() after rollback error = %v", err)
	}
	if len(revisions) != 0 {
		t.Fatalf("revisions after rollback = %#v", revisions)
	}

	stdout.Reset()
	stderr.Reset()
	if err := Migrate(context.Background(), ApplyOptions{
		WorkDir:      workDir,
		MigrationDir: "database/migrations",
		AtlasBinary:  atlasBin,
		Database:     cfg,
		Stdout:       stdout,
		Stderr:       stderr,
	}); err != nil {
		t.Fatalf("Migrate() after rollback error = %v; stderr=%q", err, stderr.String())
	}
	if !db.Migrator().HasTable("widgets") {
		t.Fatal("expected widgets table after re-migrate")
	}
}
