package migrations

import (
	"context"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"time"

	"github.com/LAA-Software-Engineering/gombit/config"
	"github.com/LAA-Software-Engineering/gombit/database"
)

// ApplyOptions configures migrate, rollback, and status commands.
type ApplyOptions struct {
	WorkDir      string
	MigrationDir string
	AtlasBinary  string
	Database     config.DatabaseConfig
	Stdout       io.Writer
	Stderr       io.Writer

	runner commandRunner
	openDB func(config.DatabaseConfig) (*database.DB, error)
	now    func() time.Time
}

func withApplyDefaults(opts ApplyOptions) ApplyOptions {
	if opts.WorkDir == "" {
		opts.WorkDir = "."
	}
	if opts.MigrationDir == "" {
		opts.MigrationDir = defaultMigrationDir
	}
	if opts.AtlasBinary == "" {
		opts.AtlasBinary = defaultAtlasBinary
	}
	if opts.Stdout == nil {
		opts.Stdout = io.Discard
	}
	if opts.Stderr == nil {
		opts.Stderr = io.Discard
	}
	if opts.runner == nil {
		opts.runner = execRunner{}
	}
	if opts.openDB == nil {
		opts.openDB = database.Open
	}
	if opts.now == nil {
		opts.now = time.Now
	}
	return opts
}

func resolveMigrationDir(opts ApplyOptions) (string, error) {
	absWorkDir, err := filepath.Abs(opts.WorkDir)
	if err != nil {
		return "", fmt.Errorf("migrations: resolve work dir: %w", err)
	}
	migrationDir := opts.MigrationDir
	if !filepath.IsAbs(migrationDir) {
		migrationDir = filepath.Join(absWorkDir, migrationDir)
	}
	return migrationDir, nil
}

func openConfiguredDB(opts ApplyOptions) (*database.DB, error) {
	db, err := opts.openDB(opts.Database)
	if err != nil {
		return nil, fmt.Errorf("migrations: open database: %w", err)
	}
	return db, nil
}

// Migrate applies pending Atlas migrations and mirrors them into framework_migrations.
func Migrate(ctx context.Context, opts ApplyOptions) error {
	if ctx == nil {
		return errors.New("migrations: nil context")
	}
	opts = withApplyDefaults(opts)
	if err := config.ValidateDatabase(opts.Database); err != nil {
		return err
	}

	migrationDir, err := resolveMigrationDir(opts)
	if err != nil {
		return err
	}
	atlasURL, err := AtlasURL(opts.Database)
	if err != nil {
		return err
	}

	db, err := openConfiguredDB(opts)
	if err != nil {
		return err
	}
	defer func() { _ = db.Close() }()

	if err := ensureRevisionsTable(db.DB); err != nil {
		return err
	}

	files, err := ListMigrationFiles(migrationDir)
	if err != nil {
		return err
	}
	revisions, err := listRevisions(db.DB)
	if err != nil {
		return err
	}
	pending := pendingFiles(files, appliedVersionSet(revisions))
	if len(pending) == 0 {
		_, _ = fmt.Fprintln(opts.Stdout, "No pending migrations.")
		return nil
	}

	absWorkDir, err := filepath.Abs(opts.WorkDir)
	if err != nil {
		return fmt.Errorf("migrations: resolve work dir: %w", err)
	}
	args := []string{
		"migrate",
		"apply",
		"--url",
		atlasURL,
		"--dir",
		"file://" + filepath.ToSlash(migrationDir),
	}
	if err := opts.runner.Run(ctx, absWorkDir, opts.AtlasBinary, args, opts.Stdout, opts.Stderr); err != nil {
		return fmt.Errorf("migrations: atlas migrate apply: %w", err)
	}

	batch, err := nextBatch(db.DB)
	if err != nil {
		return err
	}
	if err := insertBatch(db.DB, batch, pending, opts.now().UTC()); err != nil {
		return err
	}
	_, _ = fmt.Fprintf(opts.Stdout, "Applied %d migration(s) in batch %d.\n", len(pending), batch)
	return nil
}
