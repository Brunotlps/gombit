package migrations

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/LAA-Software-Engineering/gombit/config"
	"github.com/LAA-Software-Engineering/gombit/database"
)

const defaultSeedDir = "database/seeds"

// SeedOptions configures gombit db seed.
type SeedOptions struct {
	WorkDir  string
	SeedDir  string
	Database config.DatabaseConfig
	Stdout   io.Writer
	Stderr   io.Writer

	openDB func(config.DatabaseConfig) (*database.DB, error)
}

func withSeedDefaults(opts SeedOptions) SeedOptions {
	if opts.WorkDir == "" {
		opts.WorkDir = "."
	}
	if opts.SeedDir == "" {
		opts.SeedDir = defaultSeedDir
	}
	if opts.Stdout == nil {
		opts.Stdout = io.Discard
	}
	if opts.Stderr == nil {
		opts.Stderr = io.Discard
	}
	if opts.openDB == nil {
		opts.openDB = database.Open
	}
	return opts
}

func resolveSeedDir(opts SeedOptions) (string, error) {
	absWorkDir, err := filepath.Abs(opts.WorkDir)
	if err != nil {
		return "", fmt.Errorf("migrations: resolve work dir: %w", err)
	}
	seedDir := opts.SeedDir
	if !filepath.IsAbs(seedDir) {
		seedDir = filepath.Join(absWorkDir, seedDir)
	}
	return seedDir, nil
}

// Seed executes SQL seed files from the configured seed directory in lexical order.
//
// Missing or empty seed directories succeed with "No seed files.". Non-.sql entries
// are skipped with a warning on stderr. Execution stops on the first SQL error.
func Seed(ctx context.Context, opts SeedOptions) error {
	if ctx == nil {
		return errors.New("migrations: nil context")
	}
	opts = withSeedDefaults(opts)
	if err := config.ValidateDatabase(opts.Database); err != nil {
		return err
	}

	seedDir, err := resolveSeedDir(opts)
	if err != nil {
		return err
	}
	files, skipped, err := listSeedFiles(seedDir)
	if err != nil {
		return err
	}
	warnSkippedSeedFiles(opts.Stderr, skipped)
	if len(files) == 0 {
		_, _ = fmt.Fprintln(opts.Stdout, "No seed files.")
		return nil
	}

	db, err := opts.openDB(opts.Database)
	if err != nil {
		return fmt.Errorf("migrations: open database: %w", err)
	}
	defer func() { _ = db.Close() }()

	for _, path := range files {
		// #nosec G304 -- seed paths are resolved from the configured seed directory.
		sqlBytes, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("migrations: read seed %s: %w", filepath.Base(path), err)
		}
		sqlText := strings.TrimSpace(string(sqlBytes))
		if sqlText == "" {
			return fmt.Errorf("migrations: seed %s is empty", filepath.Base(path))
		}
		if err := db.Exec(sqlText).Error; err != nil {
			return fmt.Errorf("migrations: execute seed %s: %w", filepath.Base(path), err)
		}
	}
	_, _ = fmt.Fprintf(opts.Stdout, "Applied %d seed file(s).\n", len(files))
	return nil
}

func listSeedFiles(dir string) (files []string, skipped []string, err error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil, nil
		}
		return nil, nil, fmt.Errorf("migrations: read seed dir: %w", err)
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.HasSuffix(strings.ToLower(name), ".sql") {
			files = append(files, filepath.Join(dir, name))
			continue
		}
		skipped = append(skipped, name)
	}
	sort.Strings(files)
	return files, skipped, nil
}

func warnSkippedSeedFiles(stderr io.Writer, skipped []string) {
	if stderr == nil || len(skipped) == 0 {
		return
	}
	for _, name := range skipped {
		_, _ = fmt.Fprintf(stderr, "migrations: warning: skipping non-SQL seed file %s\n", name)
	}
}
