package main

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/LAA-Software-Engineering/gombit/config"
)

func TestRunMakeMigrationsInvokesAtlasAndWritesMigration(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("fake atlas shell script uses POSIX sh")
	}

	migrationDir := t.TempDir()
	atlas := fakeAtlas(t)
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	stubConfig(t, config.Default())

	err := run(context.Background(), []string{
		"db",
		"makemigrations",
		"create_products",
		"--driver",
		"sqlite",
		"--dir",
		migrationDir,
		"--atlas-bin",
		atlas,
		"--model",
		"github.com/LAA-Software-Engineering/gombit/migrations/testmodels.Product",
	}, stdout, stderr)
	if err != nil {
		t.Fatalf("run() error = %v, want nil; stderr=%q", err, stderr.String())
	}

	migrationFile := filepath.Join(migrationDir, "20260101000000_create_products.sql")
	// #nosec G304 -- migrationFile is built from t.TempDir and a fixed filename.
	data, err := os.ReadFile(migrationFile)
	if err != nil {
		t.Fatalf("read migration file: %v", err)
	}
	if !strings.Contains(string(data), "fake atlas migration") {
		t.Fatalf("migration file = %q, want fake atlas content", string(data))
	}
	if !strings.Contains(stdout.String(), "created migration") {
		t.Fatalf("stdout = %q, want atlas output", stdout.String())
	}
}

func TestRunMakeMigrationsUsesConfiguredDriver(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("fake atlas shell script uses POSIX sh")
	}

	cfg := config.Default()
	cfg.Database.Driver = config.DatabaseDriverPostgres
	stubConfig(t, cfg)
	migrationDir := t.TempDir()
	atlas := fakeAtlas(t)

	err := run(context.Background(), []string{
		"db",
		"makemigrations",
		"create_products",
		"--dir",
		migrationDir,
		"--atlas-bin",
		atlas,
		"--model",
		"github.com/LAA-Software-Engineering/gombit/migrations/testmodels.Product",
	}, ioDiscard{}, ioDiscard{})
	if err != nil {
		t.Fatalf("run() error = %v, want nil", err)
	}

	migrationFile := filepath.Join(migrationDir, "20260101000000_create_products.sql")
	// #nosec G304 -- migrationFile is built from t.TempDir and a fixed filename.
	data, err := os.ReadFile(migrationFile)
	if err != nil {
		t.Fatalf("read migration file: %v", err)
	}
	if !strings.Contains(string(data), "docker://postgres/15/dev?search_path=public") {
		t.Fatalf("migration file = %q, want postgres dev URL from config", string(data))
	}
}

func TestRunMakeMigrationsRequiresModel(t *testing.T) {
	stubConfig(t, config.Default())

	err := run(context.Background(), []string{
		"db",
		"makemigrations",
		"create_products",
		"--driver",
		"sqlite",
	}, ioDiscard{}, ioDiscard{})
	if err == nil {
		t.Fatal("run() error = nil, want missing model error")
	}
	if !strings.Contains(err.Error(), "at least one model") {
		t.Fatalf("run() error = %q, want missing model message", err)
	}
}

func TestRunRejectsUnknownCommand(t *testing.T) {
	err := run(context.Background(), []string{"unknown"}, ioDiscard{}, ioDiscard{})
	if err == nil {
		t.Fatal("run() error = nil, want unknown command error")
	}
	if !strings.Contains(err.Error(), "unknown command") {
		t.Fatalf("run() error = %q, want unknown command message", err)
	}
}

func TestRunMigrateRollbackStatus(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("fake atlas shell script uses POSIX sh")
	}

	workDir := t.TempDir()
	migrationDir := filepath.Join(workDir, "migrations")
	if err := os.MkdirAll(migrationDir, 0o750); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	writeFile(t, filepath.Join(migrationDir, "20260101000000_create_widgets.sql"),
		"CREATE TABLE widgets (id INTEGER PRIMARY KEY, name TEXT NOT NULL);")
	writeFile(t, filepath.Join(migrationDir, "20260101000000_create_widgets.down.sql"),
		"DROP TABLE IF EXISTS widgets;")

	dbPath := filepath.Join(workDir, "app.db")
	cfg := config.Default()
	cfg.Database.Driver = config.DatabaseDriverSQLite
	cfg.Database.DSN = "file:" + dbPath + "?cache=shared&_fk=1"
	stubConfig(t, cfg)

	atlas := fakeAtlasApplyStatus(t)
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)

	err := run(context.Background(), []string{
		"db", "status",
		"--dir", migrationDir,
		"--atlas-bin", atlas,
	}, stdout, stderr)
	if err != nil {
		t.Fatalf("status before migrate error = %v; stderr=%q", err, stderr.String())
	}
	if !strings.Contains(stdout.String(), "pending") {
		t.Fatalf("status stdout = %q, want pending", stdout.String())
	}
	stdout.Reset()

	err = run(context.Background(), []string{
		"db", "migrate",
		"--dir", migrationDir,
		"--atlas-bin", atlas,
	}, stdout, stderr)
	if err != nil {
		t.Fatalf("migrate error = %v; stderr=%q", err, stderr.String())
	}
	if !strings.Contains(stdout.String(), "Applied 1 migration(s) in batch 1") {
		t.Fatalf("migrate stdout = %q", stdout.String())
	}
	stdout.Reset()

	err = run(context.Background(), []string{
		"db", "rollback",
		"--dir", migrationDir,
		"--atlas-bin", atlas,
	}, stdout, stderr)
	if err != nil {
		t.Fatalf("rollback error = %v; stderr=%q", err, stderr.String())
	}
	if !strings.Contains(stdout.String(), "Rolled back batch 1") {
		t.Fatalf("rollback stdout = %q", stdout.String())
	}
}

func TestRunRejectsUnknownDBSubcommand(t *testing.T) {
	err := run(context.Background(), []string{"db", "seed"}, ioDiscard{}, ioDiscard{})
	if err == nil {
		t.Fatal("run() error = nil, want unknown subcommand error")
	}
	if !strings.Contains(err.Error(), "unknown subcommand") {
		t.Fatalf("run() error = %q", err)
	}
}

type ioDiscard struct{}

func (ioDiscard) Write(p []byte) (int, error) {
	return len(p), nil
}

func stubConfig(t *testing.T, cfg config.Config) {
	t.Helper()

	previous := loadConfig
	loadConfig = func() (config.Config, error) {
		return cfg, nil
	}
	t.Cleanup(func() {
		loadConfig = previous
	})
}

func writeFile(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func fakeAtlas(t *testing.T) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "atlas")
	script := `#!/bin/sh
set -eu
config=""
name=""
prev=""
for arg in "$@"; do
  if [ "$prev" = "--config" ]; then
    config="$arg"
  fi
  if [ "$prev" = "diff" ]; then
    name="$arg"
  fi
  prev="$arg"
done
config_path="${config#file://}"
dir="$(sed -n 's/^    dir = "file:\/\/\(.*\)"$/\1/p' "$config_path")"
dev="$(sed -n 's/^  dev = "\(.*\)"$/\1/p' "$config_path")"
mkdir -p "$dir"
{
  printf '%s\n' '-- fake atlas migration'
  printf '%s\n' "-- dev ${dev}"
} > "$dir/20260101000000_${name}.sql"
printf '%s\n' 'created migration'
`
	if err := os.WriteFile(path, []byte(script), 0o600); err != nil {
		t.Fatalf("write fake atlas: %v", err)
	}
	// #nosec G302 -- the fake Atlas script must be executable by this test process.
	if err := os.Chmod(path, 0o700); err != nil {
		t.Fatalf("chmod fake atlas: %v", err)
	}
	return path
}

func fakeAtlasApplyStatus(t *testing.T) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "atlas")
	script := `#!/bin/sh
set -eu
cmd=""
prev=""
for arg in "$@"; do
  if [ "$prev" = "migrate" ]; then
    cmd="$arg"
  fi
  prev="$arg"
done
case "$cmd" in
  apply)
    printf '%s\n' 'fake atlas apply ok'
    ;;
  status)
    printf '%s\n' 'fake atlas status'
    ;;
  *)
    printf 'unexpected atlas command: %s\n' "$*" >&2
    exit 1
    ;;
esac
`
	if err := os.WriteFile(path, []byte(script), 0o600); err != nil {
		t.Fatalf("write fake atlas: %v", err)
	}
	// #nosec G302 -- the fake Atlas script must be executable by this test process.
	if err := os.Chmod(path, 0o700); err != nil {
		t.Fatalf("chmod fake atlas: %v", err)
	}
	return path
}
