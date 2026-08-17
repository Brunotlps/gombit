package main

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	clientpkg "github.com/LAA-Software-Engineering/gombit/client"
	"github.com/LAA-Software-Engineering/gombit/config"
	"github.com/LAA-Software-Engineering/gombit/contract"
	"github.com/LAA-Software-Engineering/gombit/framework"
	"github.com/danielgtaylor/huma/v2"
	"github.com/gin-gonic/gin"
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

func TestRunOpenAPIGenerateMatchesLiveFrameworkRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cfg := config.Default()
	cfg.Environment = config.EnvironmentTest
	cfg.HTTP.Addr = "127.0.0.1:0"
	app, err := framework.New(framework.WithConfig(cfg))
	if err != nil {
		t.Fatalf("framework.New() error = %v", err)
	}
	huma.Register(app.API(), huma.Operation{
		OperationID: "list-widgets",
		Method:      http.MethodGet,
		Path:        app.Config().API.Prefix + "/widgets",
		Summary:     "List widgets",
	}, func(ctx context.Context, input *struct{}) (*struct{}, error) {
		return nil, nil
	})
	app.Router().GET("/raw/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"data": gin.H{"status": "ok"}})
	})

	server := httptest.NewServer(app.Router())
	t.Cleanup(server.Close)

	out := filepath.Join(t.TempDir(), "openapi.json")
	stdout := new(bytes.Buffer)
	err = run(context.Background(), []string{
		"openapi", "generate",
		"--url", server.URL + "/openapi.json",
		"--out", out,
	}, stdout, ioDiscard{})
	if err != nil {
		t.Fatalf("run() error = %v, want nil", err)
	}
	// #nosec G304 -- out is built from t.TempDir
	got, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("read generated spec: %v", err)
	}
	if !strings.Contains(string(got), "/api/v1/widgets") {
		t.Fatalf("generated spec missing live Huma route; body=%s", got)
	}
	if strings.Contains(string(got), "/raw/ping") {
		t.Fatal("generated spec unexpectedly contains raw Gin route")
	}

	docs, err := http.Get(server.URL + "/docs")
	if err != nil {
		t.Fatalf("GET /docs: %v", err)
	}
	t.Cleanup(func() { _ = docs.Body.Close() })
	if docs.StatusCode != http.StatusOK {
		t.Fatalf("GET /docs status = %d, want %d", docs.StatusCode, http.StatusOK)
	}
}

func TestRunOpenAPIGenerateWritesValidatedSpec(t *testing.T) {
	spec := `{
  "openapi": "3.1.0",
  "info": {
    "title": "Gombit",
    "version": "0.0.0"
  },
  "paths": {
    "/api/v1/widgets": {
      "get": {
        "operationId": "list-widgets",
        "responses": {
          "200": {
            "description": "OK"
          }
        }
      }
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

	out := filepath.Join(t.TempDir(), "openapi.json")
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	err := run(context.Background(), []string{
		"openapi", "generate",
		"--url", server.URL + "/openapi.json",
		"--out", out,
	}, stdout, stderr)
	if err != nil {
		t.Fatalf("run() error = %v, want nil; stderr=%q", err, stderr.String())
	}
	if !strings.Contains(stdout.String(), out) {
		t.Fatalf("stdout = %q, want output path %q", stdout.String(), out)
	}
	// #nosec G304 -- out is built from t.TempDir
	got, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("read generated spec: %v", err)
	}
	if !strings.Contains(string(got), "/api/v1/widgets") {
		t.Fatalf("generated spec missing live route; body=%s", got)
	}
	if strings.Contains(string(got), "/raw/ping") {
		t.Fatal("generated spec unexpectedly contains raw Gin route")
	}
}

func TestRunOpenAPIGenerateRejectsInvalidSpec(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"not":"openapi"}`))
	}))
	t.Cleanup(server.Close)

	err := run(context.Background(), []string{
		"openapi", "generate",
		"--url", server.URL,
		"--out", filepath.Join(t.TempDir(), "openapi.json"),
	}, ioDiscard{}, ioDiscard{})
	if err == nil {
		t.Fatal("run() error = nil, want invalid spec error")
	}
	if !strings.Contains(err.Error(), "OpenAPI") {
		t.Fatalf("run() error = %q, want OpenAPI validation message", err)
	}
}

func TestRunOpenAPIGenerateRejectsOversizedSpec(t *testing.T) {
	previous := maxOpenAPISize
	maxOpenAPISize = 16
	t.Cleanup(func() { maxOpenAPISize = previous })

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"openapi":"3.1.0","info":{"title":"t","version":"0"},"paths":{}}`))
	}))
	t.Cleanup(server.Close)

	err := run(context.Background(), []string{
		"openapi", "generate",
		"--url", server.URL,
		"--out", filepath.Join(t.TempDir(), "openapi.json"),
	}, ioDiscard{}, ioDiscard{})
	if err == nil {
		t.Fatal("run() error = nil, want oversized spec error")
	}
	if !strings.Contains(err.Error(), "exceeds 8MiB") {
		t.Fatalf("run() error = %q, want exceeds 8MiB", err)
	}
}

func TestRunRejectsUnknownCommandUsageListsFamilies(t *testing.T) {
	stderr := new(bytes.Buffer)
	err := run(context.Background(), []string{"unknown"}, ioDiscard{}, stderr)
	if err == nil {
		t.Fatal("run() error = nil, want unknown command error")
	}
	got := stderr.String()
	if !strings.Contains(got, "openapi generate") {
		t.Fatalf("usage = %q, want openapi generate", got)
	}
	if !strings.Contains(got, "client generate") {
		t.Fatalf("usage = %q, want client generate", got)
	}
	if !strings.Contains(got, "client check") {
		t.Fatalf("usage = %q, want client check", got)
	}
	if !strings.Contains(got, "make resource") {
		t.Fatalf("usage = %q, want make resource", got)
	}
	if !strings.Contains(got, "see gombit db") {
		t.Fatalf("usage = %q, want pointer to gombit db", got)
	}
	if strings.Contains(got, "makemigrations") {
		t.Fatalf("usage = %q, does not want full db help", got)
	}
}

func TestRunOpenAPIGenerateRejectsBadURL(t *testing.T) {
	err := run(context.Background(), []string{
		"openapi", "generate",
		"--url", "file:///tmp/openapi.json",
	}, ioDiscard{}, ioDiscard{})
	if err == nil {
		t.Fatal("run() error = nil, want scheme error")
	}
	if !strings.Contains(err.Error(), "http or https") {
		t.Fatalf("run() error = %q, want scheme message", err)
	}
}

func TestRunClientGenerateDryRun(t *testing.T) {
	app, err := clientpkg.SampleApp()
	if err != nil {
		t.Fatalf("SampleApp() error = %v", err)
	}
	workDir := t.TempDir()
	spec := filepath.Join(workDir, "openapi.json")
	if err := contract.WriteOpenAPI(spec, app.API()); err != nil {
		t.Fatalf("WriteOpenAPI: %v", err)
	}

	previous, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(workDir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(previous) })

	stdout := new(bytes.Buffer)
	err = run(context.Background(), []string{
		"client", "generate",
		"--spec", "openapi.json",
		"--out", "frontend/src/api/generated",
		"--dry-run",
	}, stdout, ioDiscard{})
	if err != nil {
		t.Fatalf("run() error = %v", err)
	}
	if !strings.Contains(stdout.String(), "schema.ts") {
		t.Fatalf("stdout = %q, want schema.ts", stdout.String())
	}
	if _, err := os.Stat(filepath.Join(workDir, "frontend", "src", "api", "generated", "schema.ts")); !os.IsNotExist(err) {
		t.Fatal("dry-run wrote schema.ts")
	}
}

func TestRunClientCheckDetectsStaleSpec(t *testing.T) {
	requireNodeCLI(t)

	root := cmdModuleRoot(t)
	workDir := t.TempDir()
	copyFile(t, filepath.Join(root, clientpkg.SampleSpecPath), filepath.Join(workDir, "openapi.json"))
	for _, name := range []string{"schema.ts", "client.ts", "error.ts"} {
		copyFile(t,
			filepath.Join(root, clientpkg.SampleOutDir, name),
			filepath.Join(workDir, "frontend", "src", "api", "generated", name),
		)
	}

	spec := readFileString(t, filepath.Join(workDir, "openapi.json"))
	tampered := strings.Replace(spec, "/api/v1/widgets", "/api/v1/stale-widgets", 1)
	if tampered == spec {
		t.Fatal("failed to tamper committed spec")
	}
	writeFile(t, filepath.Join(workDir, "openapi.json"), tampered)

	previous, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(workDir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(previous) })

	err = run(context.Background(), []string{
		"client", "check",
		"--spec", "openapi.json",
		"--out", "frontend/src/api/generated",
	}, ioDiscard{}, ioDiscard{})
	if err == nil {
		t.Fatal("run() error = nil, want contract drift")
	}
	if !strings.Contains(err.Error(), "contract drift") {
		t.Fatalf("run() error = %q, want contract drift", err)
	}
}

func TestRunClientCheckWriteThenClean(t *testing.T) {
	requireNodeCLI(t)

	workDir := t.TempDir()
	previous, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(workDir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(previous) })

	stdout := new(bytes.Buffer)
	err = run(context.Background(), []string{
		"client", "check",
		"--write",
		"--spec", "openapi.json",
		"--out", "frontend/src/api/generated",
	}, stdout, ioDiscard{})
	if err != nil {
		t.Fatalf("run(check --write) error = %v", err)
	}
	if !strings.Contains(stdout.String(), "openapi.json") {
		t.Fatalf("stdout = %q, want wrote openapi.json", stdout.String())
	}

	stdout.Reset()
	err = run(context.Background(), []string{
		"client", "check",
		"--spec", "openapi.json",
		"--out", "frontend/src/api/generated",
	}, stdout, ioDiscard{})
	if err != nil {
		t.Fatalf("run(check) error = %v, want nil after write", err)
	}
	if !strings.Contains(stdout.String(), "no contract drift") {
		t.Fatalf("stdout = %q, want no contract drift", stdout.String())
	}
}

func TestRunRejectsUnknownClientSubcommand(t *testing.T) {
	stderr := new(bytes.Buffer)
	err := run(context.Background(), []string{"client", "unknown"}, ioDiscard{}, stderr)
	if err == nil {
		t.Fatal("run() error = nil, want unknown subcommand error")
	}
	if !strings.Contains(err.Error(), "unknown subcommand") {
		t.Fatalf("run() error = %q, want unknown subcommand message", err)
	}
	if !strings.Contains(stderr.String(), "--npx") {
		t.Fatalf("client usage = %q, want --npx", stderr.String())
	}
	if !strings.Contains(stderr.String(), "check") {
		t.Fatalf("client usage = %q, want check", stderr.String())
	}
}

func TestRunRejectsUnknownOpenAPISubcommand(t *testing.T) {
	err := run(context.Background(), []string{"openapi", "unknown"}, ioDiscard{}, ioDiscard{})
	if err == nil {
		t.Fatal("run() error = nil, want unknown subcommand error")
	}
	if !strings.Contains(err.Error(), "unknown subcommand") {
		t.Fatalf("run() error = %q, want unknown subcommand message", err)
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
	writeFile(t, filepath.Join(migrationDir, "downs", "20260101000000_create_widgets.down.sql"),
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
	err := run(context.Background(), []string{"db", "unknown"}, ioDiscard{}, ioDiscard{})
	if err == nil {
		t.Fatal("run() error = nil, want unknown subcommand error")
	}
	if !strings.Contains(err.Error(), "unknown subcommand") {
		t.Fatalf("run() error = %q", err)
	}
}

func TestRunSeedAndReset(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("fake atlas shell script uses POSIX sh")
	}

	workDir := t.TempDir()
	migrationDir := filepath.Join(workDir, "migrations")
	seedDir := filepath.Join(workDir, "seeds")
	if err := os.MkdirAll(migrationDir, 0o750); err != nil {
		t.Fatalf("mkdir migrations: %v", err)
	}
	if err := os.MkdirAll(seedDir, 0o750); err != nil {
		t.Fatalf("mkdir seeds: %v", err)
	}
	writeFile(t, filepath.Join(migrationDir, "20260101000000_create_widgets.sql"),
		"CREATE TABLE widgets (id INTEGER PRIMARY KEY, name TEXT NOT NULL);")
	writeFile(t, filepath.Join(seedDir, "01_widgets.sql"),
		"INSERT INTO widgets (id, name) VALUES (1, 'from-seed');")

	dbPath := filepath.Join(workDir, "app.db")
	cfg := config.Default()
	cfg.Database.Driver = config.DatabaseDriverSQLite
	cfg.Database.DSN = "file:" + dbPath + "?cache=shared&_fk=1"
	stubConfig(t, cfg)

	atlas := fakeAtlasApplyStatus(t)
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)

	err := run(context.Background(), []string{
		"db", "reset",
		"--dir", migrationDir,
		"--seeds", seedDir,
		"--atlas-bin", atlas,
	}, stdout, stderr)
	if err != nil {
		t.Fatalf("reset error = %v; stderr=%q", err, stderr.String())
	}
	if !strings.Contains(stdout.String(), "Dropped database schema") {
		t.Fatalf("reset stdout = %q, want drop message", stdout.String())
	}
	if !strings.Contains(stdout.String(), "Applied 1 seed file(s)") {
		t.Fatalf("reset stdout = %q, want seed message", stdout.String())
	}
	stdout.Reset()

	emptySeeds := filepath.Join(workDir, "empty-seeds")
	if err := os.MkdirAll(emptySeeds, 0o750); err != nil {
		t.Fatalf("mkdir empty seeds: %v", err)
	}
	err = run(context.Background(), []string{
		"db", "seed",
		"--seeds", emptySeeds,
	}, stdout, stderr)
	if err != nil {
		t.Fatalf("seed empty dir error = %v; stderr=%q", err, stderr.String())
	}
	if !strings.Contains(stdout.String(), "No seed files") {
		t.Fatalf("seed stdout = %q, want no seed files", stdout.String())
	}
}

func TestRunResetRefusesProductionWithoutForce(t *testing.T) {
	cfg := config.Default()
	cfg.Environment = config.EnvironmentProduction
	stubConfig(t, cfg)

	err := run(context.Background(), []string{"db", "reset"}, ioDiscard{}, ioDiscard{})
	if err == nil {
		t.Fatal("run() error = nil, want production refusal")
	}
	if !strings.Contains(err.Error(), "refuse reset in production without --force") {
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
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func copyFile(t *testing.T, src, dest string) {
	t.Helper()
	writeFile(t, dest, readFileString(t, src))
}

func readFileString(t *testing.T, path string) string {
	t.Helper()
	// #nosec G304 -- test fixture path
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(data)
}

func requireNodeCLI(t *testing.T) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("npx openapi-typescript path handling differs on Windows")
	}
	if _, err := exec.LookPath("npx"); err != nil {
		t.Skip("npx not available")
	}
	if _, err := exec.LookPath("npm"); err != nil {
		t.Skip("npm not available")
	}
}

func cmdModuleRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
	if _, err := os.Stat(filepath.Join(root, "go.mod")); err != nil {
		t.Fatalf("module root %s: %v", root, err)
	}
	return root
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
url=""
dir=""
prev=""
for arg in "$@"; do
  if [ "$prev" = "migrate" ]; then
    cmd="$arg"
  fi
  if [ "$prev" = "--url" ]; then
    url="$arg"
  fi
  if [ "$prev" = "--dir" ]; then
    dir="$arg"
  fi
  prev="$arg"
done
case "$cmd" in
  apply)
    python3 - "$url" "$dir" <<'PY'
import os
import sqlite3
import sys
from datetime import datetime, timezone

url, dir_url = sys.argv[1], sys.argv[2]
if not url.startswith("sqlite://"):
    raise SystemExit(f"unsupported url: {url}")
rest = url[len("sqlite://"):]
db_path = rest.split("?", 1)[0]
mig_dir = dir_url.removeprefix("file://")

conn = sqlite3.connect(db_path)
try:
    conn.execute(
        """
        CREATE TABLE IF NOT EXISTS atlas_schema_revisions (
          version TEXT PRIMARY KEY,
          description TEXT,
          type TEXT,
          applied_at TEXT
        )
        """
    )
    existing = {
        row[0]
        for row in conn.execute("SELECT version FROM atlas_schema_revisions")
    }
    for name in sorted(os.listdir(mig_dir)):
        if (
            not name.endswith(".sql")
            or name.endswith(".down.sql")
            or name == "atlas.sum"
        ):
            continue
        version = name.split("_", 1)[0]
        if version in existing:
            continue
        with open(os.path.join(mig_dir, name), encoding="utf-8") as f:
            sql = f.read()
        conn.executescript(sql)
        conn.execute(
            "INSERT INTO atlas_schema_revisions (version, description, type, applied_at) VALUES (?, ?, ?, ?)",
            (version, name, "sql", datetime.now(timezone.utc).isoformat()),
        )
    conn.commit()
finally:
    conn.close()
PY
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
