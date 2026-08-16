package migrations

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/LAA-Software-Engineering/gombit/config"
)

// Rollback rolls back the latest framework_migrations batch using companion down SQL.
func Rollback(ctx context.Context, opts ApplyOptions) error {
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

	db, err := openConfiguredDB(opts)
	if err != nil {
		return err
	}
	defer func() { _ = db.Close() }()

	if err := ensureRevisionsTable(db.DB); err != nil {
		return err
	}

	batch, revisions, err := lastBatch(db.DB)
	if err != nil {
		return err
	}
	if batch == 0 || len(revisions) == 0 {
		_, _ = fmt.Fprintln(opts.Stdout, "Nothing to roll back.")
		return nil
	}

	files, err := ListMigrationFiles(migrationDir)
	if err != nil {
		return err
	}
	byVersion := make(map[string]MigrationFile, len(files))
	for _, file := range files {
		byVersion[file.Version] = file
	}

	downs := make([]MigrationFile, 0, len(revisions))
	missing := make([]string, 0)
	for _, rev := range revisions {
		file, ok := byVersion[rev.Version]
		if !ok || file.DownPath == "" {
			missing = append(missing, fmt.Sprintf("%s_%s.down.sql", rev.Version, rev.Name))
			continue
		}
		downs = append(downs, file)
	}
	if len(missing) > 0 {
		return fmt.Errorf("migrations: missing down migration(s) for batch %d: %s", batch, strings.Join(missing, ", "))
	}

	versions := make([]string, 0, len(downs))
	for _, file := range downs {
		// #nosec G304 -- down paths are resolved from the configured migration directory.
		sqlBytes, err := os.ReadFile(file.DownPath)
		if err != nil {
			return fmt.Errorf("migrations: read down %s: %w", filepath.Base(file.DownPath), err)
		}
		sqlText := strings.TrimSpace(string(sqlBytes))
		if sqlText == "" {
			return fmt.Errorf("migrations: empty down migration %s", filepath.Base(file.DownPath))
		}
		if err := db.Exec(sqlText).Error; err != nil {
			return fmt.Errorf("migrations: execute down %s: %w", filepath.Base(file.DownPath), err)
		}
		versions = append(versions, file.Version)
	}

	if err := deleteRevisions(db.DB, versions); err != nil {
		return err
	}
	if err := deleteAtlasRevisions(db.DB, versions); err != nil {
		return err
	}

	_, _ = fmt.Fprintf(opts.Stdout, "Rolled back batch %d (%d migration(s)).\n", batch, len(versions))
	return nil
}
