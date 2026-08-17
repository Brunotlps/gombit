package main

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/LAA-Software-Engineering/gombit/config"
)

func TestRunHelpListsIntrospectionCommands(t *testing.T) {
	stdout := new(bytes.Buffer)
	err := run(context.Background(), []string{"--help"}, stdout, ioDiscard{})
	if err != nil {
		t.Fatalf("run(--help) error = %v", err)
	}
	got := stdout.String()
	for _, name := range []string{"routes", "doctor", "config"} {
		if !strings.Contains(got, name) {
			t.Fatalf("root help missing %q:\n%s", name, got)
		}
	}
}

func TestRunRejectsUnknownCommandUsageListsIntrospection(t *testing.T) {
	stderr := new(bytes.Buffer)
	err := run(context.Background(), []string{"unknown"}, ioDiscard{}, stderr)
	if err == nil {
		t.Fatal("run() error = nil, want unknown command error")
	}
	got := stderr.String()
	for _, want := range []string{"routes", "doctor", "config show"} {
		if !strings.Contains(got, want) {
			t.Fatalf("usage = %q, want %q", got, want)
		}
	}
}

func TestRunConfigShowRedactsSecrets(t *testing.T) {
	cfg := config.Default()
	cfg.Database.Driver = config.DatabaseDriverPostgres
	cfg.Database.DSN = "postgres://gombit:db-super-secret@127.0.0.1:5432/gombit?sslmode=disable"
	cfg.Cache.Redis.Password = "redis-super-secret"
	stubConfig(t, cfg)

	stdout := new(bytes.Buffer)
	err := run(context.Background(), []string{"config", "show"}, stdout, ioDiscard{})
	if err != nil {
		t.Fatalf("run(config show) error = %v", err)
	}
	got := stdout.String()
	if strings.Contains(got, "db-super-secret") || strings.Contains(got, "redis-super-secret") {
		t.Fatalf("config show leaked secrets:\n%s", got)
	}
	if !strings.Contains(got, config.RedactedSecret) {
		t.Fatalf("config show = %q, want redacted placeholder", got)
	}
	if !strings.Contains(got, "Database.DSN") {
		t.Fatalf("config show = %q, want Database.DSN", got)
	}
	if !strings.Contains(got, "Cache.Redis.Password") {
		t.Fatalf("config show = %q, want Cache.Redis.Password", got)
	}
}

func TestRunRejectsUnknownConfigSubcommand(t *testing.T) {
	stderr := new(bytes.Buffer)
	err := run(context.Background(), []string{"config", "unknown"}, ioDiscard{}, stderr)
	if err == nil {
		t.Fatal("run() error = nil, want unknown subcommand error")
	}
	if !strings.Contains(err.Error(), "unknown subcommand") {
		t.Fatalf("run() error = %q, want unknown subcommand", err)
	}
	if !strings.Contains(stderr.String(), "show") {
		t.Fatalf("config usage = %q, want show", stderr.String())
	}
}

func TestRunRoutesListsFrameworkPaths(t *testing.T) {
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	err := run(context.Background(), []string{"routes"}, stdout, stderr)
	if err != nil {
		t.Fatalf("run(routes) error = %v; stderr=%q", err, stderr.String())
	}
	got := stdout.String()
	for _, path := range []string{"/openapi.json", "/docs", "/livez"} {
		if !strings.Contains(got, path) {
			t.Fatalf("routes output missing %q:\n%s", path, got)
		}
	}
	if !strings.Contains(got, "GET") {
		t.Fatalf("routes output missing GET:\n%s", got)
	}
}

func TestRunRoutesURLListsOpenAPIPaths(t *testing.T) {
	spec := `{
  "openapi": "3.1.0",
  "info": {"title": "Gombit", "version": "0.0.0"},
  "paths": {
    "/api/v1/widgets": {
      "get": {"operationId": "list-widgets", "responses": {"200": {"description": "OK"}}}
    }
  }
}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/openapi.json" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(spec))
	}))
	t.Cleanup(server.Close)

	stdout := new(bytes.Buffer)
	err := run(context.Background(), []string{"routes", "--url", server.URL}, stdout, ioDiscard{})
	if err != nil {
		t.Fatalf("run(routes --url) error = %v", err)
	}
	got := stdout.String()
	if !strings.Contains(got, "/api/v1/widgets") {
		t.Fatalf("routes --url = %q, want /api/v1/widgets", got)
	}
	if !strings.Contains(got, "GET") {
		t.Fatalf("routes --url = %q, want GET", got)
	}
}

func TestRunDoctorFlagsBrokenConfig(t *testing.T) {
	t.Setenv("GOMBIT_DATABASE_DRIVER", "not-a-driver")

	stdout := new(bytes.Buffer)
	err := run(context.Background(), []string{"doctor"}, stdout, ioDiscard{})
	if err == nil {
		t.Fatal("run(doctor) error = nil, want non-zero for broken config")
	}
	if !strings.Contains(err.Error(), "gombit doctor: one or more checks failed") {
		t.Fatalf("run(doctor) error = %q, want doctor failure", err)
	}
	got := stdout.String()
	if !strings.Contains(got, "FAIL") {
		t.Fatalf("doctor output = %q, want FAIL", got)
	}
	if !strings.Contains(got, "config") {
		t.Fatalf("doctor output = %q, want named config check", got)
	}
	if !strings.Contains(got, "GOMBIT_DATABASE_DRIVER") && !strings.Contains(got, "Database.Driver") {
		t.Fatalf("doctor output = %q, want driver field error", got)
	}
}

func TestRunDoctorFlagsUnwritableSQLitePath(t *testing.T) {
	blocked := filepath.Join(t.TempDir(), "not-a-dir")
	if err := os.WriteFile(blocked, []byte("x"), 0o600); err != nil {
		t.Fatalf("write blocker: %v", err)
	}
	cfg := config.Default()
	cfg.HTTP.Addr = "127.0.0.1:0"
	cfg.Database.Driver = config.DatabaseDriverSQLite
	cfg.Database.DSN = "file:" + blocked + "/app.db?cache=shared&_fk=1"
	stubConfig(t, cfg)

	stdout := new(bytes.Buffer)
	err := run(context.Background(), []string{"doctor"}, stdout, ioDiscard{})
	if err == nil {
		t.Fatal("run(doctor) error = nil, want failure for unwritable sqlite path")
	}
	got := stdout.String()
	if !strings.Contains(got, "FAIL") {
		t.Fatalf("doctor output = %q, want FAIL", got)
	}
	if !strings.Contains(got, "insecure") && !strings.Contains(got, "database") {
		t.Fatalf("doctor output = %q, want named insecure or database failure", got)
	}
}

func TestRunDoctorHealthySQLiteMemory(t *testing.T) {
	cfg := config.Default()
	cfg.Environment = config.EnvironmentTest
	cfg.HTTP.Addr = "127.0.0.1:0"
	cfg.Database.Driver = config.DatabaseDriverSQLite
	cfg.Database.DSN = "file::memory:?cache=shared&_fk=1"
	cfg.Cache.Driver = config.CacheDriverMemory
	stubConfig(t, cfg)

	stdout := new(bytes.Buffer)
	err := run(context.Background(), []string{"doctor"}, stdout, ioDiscard{})
	if err != nil {
		t.Fatalf("run(doctor) error = %v; output=%q", err, stdout.String())
	}
	got := stdout.String()
	if strings.Contains(got, "FAIL") {
		t.Fatalf("doctor output = %q, did not want FAIL", got)
	}
	if !strings.Contains(got, "config") || !strings.Contains(got, "database") {
		t.Fatalf("doctor output = %q, want config and database checks", got)
	}
}
