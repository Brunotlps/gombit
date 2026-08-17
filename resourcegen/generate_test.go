package resourcegen

import (
	"bytes"
	"context"
	"errors"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/LAA-Software-Engineering/gombit/migrations"
	"github.com/LAA-Software-Engineering/gombit/scaffold"
)

func TestGenerateBookFeaturePackage(t *testing.T) {
	workDir := t.TempDir()
	if err := scaffold.Generate(context.Background(), scaffold.Options{
		Name:     "demo",
		Database: "sqlite",
		WorkDir:  workDir,
		Stdout:   ioDiscard{},
	}); err != nil {
		t.Fatalf("scaffold: %v", err)
	}
	appDir := filepath.Join(workDir, "demo")

	previousLook := lookPath
	lookPath = func(string) (string, error) { return "", errors.New("atlas missing") }
	t.Cleanup(func() { lookPath = previousLook })

	stdout := new(bytes.Buffer)
	err := Generate(context.Background(), Options{
		WorkDir:   appDir,
		Name:      "Book",
		Fields:    []string{"title:string:required"},
		Stdout:    stdout,
		skipAtlas: true,
	})
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if !strings.Contains(stdout.String(), "create internal/book/book.go") {
		t.Fatalf("stdout = %q, want created model", stdout.String())
	}
	if !strings.Contains(stdout.String(), "gombit db makemigrations") {
		t.Fatalf("stdout = %q, want makemigrations hint", stdout.String())
	}

	modelPath := filepath.Join(appDir, "internal", "book", "book.go")
	modelSrc := readFile(t, modelPath)
	if !strings.Contains(modelSrc, GeneratedBanner) {
		t.Fatal("model missing generated banner")
	}
	if !strings.Contains(modelSrc, "type Book struct") {
		t.Fatal("model missing Book type")
	}
	fset := token.NewFileSet()
	if _, err := parser.ParseFile(fset, modelPath, modelSrc, 0); err != nil {
		t.Fatalf("generated model is not valid Go: %v", err)
	}

	mod := readModulePathMust(t, appDir)
	spec := mod + "/internal/book.Book"
	parsed, err := migrations.ParseModel(spec)
	if err != nil {
		t.Fatalf("ParseModel(%q) error = %v", spec, err)
	}
	if parsed.TypeName != "Book" {
		t.Fatalf("ParseModel type = %q, want Book", parsed.TypeName)
	}

	mainSrc := readFile(t, filepath.Join(appDir, "cmd", "server", "main.go"))
	count, err := CountRegisterCalls([]byte(mainSrc), "book")
	if err != nil {
		t.Fatalf("CountRegisterCalls: %v", err)
	}
	if count != 1 {
		t.Fatalf("book.Register count = %d, want 1\n%s", count, mainSrc)
	}

	handlerPath := filepath.Join(appDir, "internal", "book", "handler.go")
	handlerSrc := readFile(t, handlerPath)
	if !strings.Contains(handlerSrc, `contract.Internal("list books")`) {
		t.Fatalf("handler Internal message = %q, want list books", handlerSrc)
	}
	if _, err := os.Stat(filepath.Join(appDir, "internal", "book", "service.go")); !os.IsNotExist(err) {
		t.Fatal("default generate wrote service.go")
	}
	if _, err := os.Stat(filepath.Join(appDir, "internal", "book", "repo.go")); !os.IsNotExist(err) {
		t.Fatal("default generate wrote repo.go")
	}

	listTS := readFile(t, filepath.Join(appDir, "frontend", "src", "book", "list.ts"))
	if !strings.Contains(listTS, `from "../api/generated/schema"`) {
		t.Fatal("list.ts does not import generated OpenAPI types")
	}
	if strings.Contains(strings.ToLower(listTS), "localstorage") {
		t.Fatal("list.ts uses localStorage")
	}

	// Idempotent re-run does not duplicate Register.
	err = Generate(context.Background(), Options{
		WorkDir:   appDir,
		Name:      "Book",
		Fields:    []string{"title:string:required"},
		Stdout:    ioDiscard{},
		skipAtlas: true,
	})
	if err != nil {
		t.Fatalf("re-run Generate() error = %v", err)
	}
	mainSrc = readFile(t, filepath.Join(appDir, "cmd", "server", "main.go"))
	count, err = CountRegisterCalls([]byte(mainSrc), "book")
	if err != nil {
		t.Fatalf("CountRegisterCalls after re-run: %v", err)
	}
	if count != 1 {
		t.Fatalf("re-run duplicated book.Register: count = %d", count)
	}

	// User edit is refused without --force.
	if err := os.WriteFile(handlerPath, []byte("package book\n\n// user edit\n"), 0o600); err != nil {
		t.Fatalf("edit handler: %v", err)
	}
	err = Generate(context.Background(), Options{
		WorkDir:   appDir,
		Name:      "Book",
		Fields:    []string{"title:string:required"},
		Stdout:    ioDiscard{},
		skipAtlas: true,
	})
	if err == nil || !strings.Contains(err.Error(), "--force") {
		t.Fatalf("clobber error = %v, want --force", err)
	}
	got := readFile(t, handlerPath)
	if !strings.Contains(got, "user edit") {
		t.Fatal("user handler.go was overwritten without --force")
	}

	err = Generate(context.Background(), Options{
		WorkDir:   appDir,
		Name:      "Book",
		Fields:    []string{"title:string:required"},
		Force:     true,
		Stdout:    ioDiscard{},
		skipAtlas: true,
	})
	if err != nil {
		t.Fatalf("Generate(--force) error = %v", err)
	}
	got = readFile(t, handlerPath)
	if strings.Contains(got, "user edit") {
		t.Fatal("--force did not replace handler.go")
	}
}

func TestGenerateDryRunAndServiceRepo(t *testing.T) {
	workDir := t.TempDir()
	if err := scaffold.Generate(context.Background(), scaffold.Options{
		Name:     "demo",
		Database: "sqlite",
		WorkDir:  workDir,
		Stdout:   ioDiscard{},
	}); err != nil {
		t.Fatalf("scaffold: %v", err)
	}
	appDir := filepath.Join(workDir, "demo")

	stdout := new(bytes.Buffer)
	err := Generate(context.Background(), Options{
		WorkDir:   appDir,
		Name:      "Invoice",
		Service:   true,
		Repo:      true,
		DryRun:    true,
		Stdout:    stdout,
		skipAtlas: true,
	})
	if err != nil {
		t.Fatalf("Generate(--dry-run) error = %v", err)
	}
	out := stdout.String()
	if !strings.Contains(out, "internal/invoice/service.go") || !strings.Contains(out, "internal/invoice/repo.go") {
		t.Fatalf("dry-run stdout = %q, want service and repo", out)
	}
	if _, err := os.Stat(filepath.Join(appDir, "internal", "invoice")); !os.IsNotExist(err) {
		t.Fatal("dry-run wrote invoice package")
	}

	err = Generate(context.Background(), Options{
		WorkDir:   appDir,
		Name:      "Invoice",
		Service:   true,
		Repo:      true,
		Stdout:    ioDiscard{},
		skipAtlas: true,
	})
	if err != nil {
		t.Fatalf("Generate(--service --repo) error = %v", err)
	}
	if _, err := os.Stat(filepath.Join(appDir, "internal", "invoice", "service.go")); err != nil {
		t.Fatalf("missing service.go: %v", err)
	}
	if _, err := os.Stat(filepath.Join(appDir, "internal", "invoice", "repo.go")); err != nil {
		t.Fatalf("missing repo.go: %v", err)
	}
}

func TestGenerateMissingAtlasPrintsHint(t *testing.T) {
	workDir := t.TempDir()
	if err := scaffold.Generate(context.Background(), scaffold.Options{
		Name:     "demo",
		Database: "sqlite",
		WorkDir:  workDir,
		Stdout:   ioDiscard{},
	}); err != nil {
		t.Fatalf("scaffold: %v", err)
	}

	previousLook := lookPath
	lookPath = func(string) (string, error) { return "", errors.New("atlas missing") }
	t.Cleanup(func() { lookPath = previousLook })

	stdout := new(bytes.Buffer)
	err := Generate(context.Background(), Options{
		WorkDir: filepath.Join(workDir, "demo"),
		Name:    "Book",
		Fields:  []string{"title:string:required"},
		Stdout:  stdout,
	})
	if err != nil {
		t.Fatalf("Generate() error = %v, want missing-atlas hint", err)
	}
	if !strings.Contains(stdout.String(), "atlas not on PATH") {
		t.Fatalf("stdout = %q, want atlas not on PATH hint", stdout.String())
	}
}

func TestGenerateSurfacesMakeMigrationsError(t *testing.T) {
	workDir := t.TempDir()
	if err := scaffold.Generate(context.Background(), scaffold.Options{
		Name:     "demo",
		Database: "sqlite",
		WorkDir:  workDir,
		Stdout:   ioDiscard{},
	}); err != nil {
		t.Fatalf("scaffold: %v", err)
	}

	previousLook := lookPath
	lookPath = func(string) (string, error) { return "/usr/bin/atlas", nil }
	t.Cleanup(func() { lookPath = previousLook })

	previousMake := makeMigrations
	makeMigrations = func(context.Context, migrations.Options) error {
		return errors.New("atlas migrate diff failed")
	}
	t.Cleanup(func() { makeMigrations = previousMake })

	stdout := new(bytes.Buffer)
	err := Generate(context.Background(), Options{
		WorkDir: filepath.Join(workDir, "demo"),
		Name:    "Book",
		Fields:  []string{"title:string:required"},
		Stdout:  stdout,
	})
	if err == nil {
		t.Fatal("Generate() error = nil, want makemigrations failure")
	}
	if !strings.Contains(err.Error(), "makemigrations") || !strings.Contains(err.Error(), "atlas migrate diff failed") {
		t.Fatalf("error = %v, want wrapped atlas failure", err)
	}
	if strings.Contains(stdout.String(), "note:") {
		t.Fatalf("stdout = %q, did not want swallowed atlas note", stdout.String())
	}
}

func TestGenerateUnknownType(t *testing.T) {
	workDir := t.TempDir()
	if err := scaffold.Generate(context.Background(), scaffold.Options{
		Name:     "demo",
		Database: "sqlite",
		WorkDir:  workDir,
		Stdout:   ioDiscard{},
	}); err != nil {
		t.Fatalf("scaffold: %v", err)
	}
	err := Generate(context.Background(), Options{
		WorkDir: filepath.Join(workDir, "demo"),
		Name:    "Widget",
		Fields:  []string{"amount:decimal"},
		Stdout:  ioDiscard{},
	})
	if err == nil || !strings.Contains(err.Error(), "unknown type") {
		t.Fatalf("error = %v, want unknown type", err)
	}
}

type ioDiscard struct{}

func (ioDiscard) Write(p []byte) (int, error) {
	return len(p), nil
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	// #nosec G304 -- test fixture path
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(data)
}

func readModulePathMust(t *testing.T, dir string) string {
	t.Helper()
	mod, err := readModulePath(dir)
	if err != nil {
		t.Fatalf("readModulePath: %v", err)
	}
	return mod
}
