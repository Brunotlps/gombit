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
