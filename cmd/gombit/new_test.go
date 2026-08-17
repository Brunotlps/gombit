package main

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunHelpListsCommandFamilies(t *testing.T) {
	stdout := new(bytes.Buffer)
	err := run(context.Background(), []string{"--help"}, stdout, ioDiscard{})
	if err != nil {
		t.Fatalf("run(--help) error = %v", err)
	}
	got := stdout.String()
	for _, family := range []string{"new", "dev", "make", "db", "openapi", "client", "routes", "doctor", "config"} {
		if !strings.Contains(got, family) {
			t.Fatalf("root help missing %q:\n%s", family, got)
		}
	}
}

func TestRunDBStatusHelp(t *testing.T) {
	stdout := new(bytes.Buffer)
	err := run(context.Background(), []string{"db", "status", "--help"}, stdout, ioDiscard{})
	if err != nil {
		t.Fatalf("run(db status --help) error = %v", err)
	}
	got := stdout.String()
	if !strings.Contains(got, "--dir") {
		t.Fatalf("status help missing --dir:\n%s", got)
	}
	if !strings.Contains(got, "--atlas-bin") {
		t.Fatalf("status help missing --atlas-bin:\n%s", got)
	}
}

func TestRunNewDryRunWritesNothing(t *testing.T) {
	workDir := t.TempDir()
	chdir(t, workDir)

	stdout := new(bytes.Buffer)
	err := run(context.Background(), []string{"new", "demo", "--database", "sqlite", "--dry-run"}, stdout, ioDiscard{})
	if err != nil {
		t.Fatalf("run() error = %v", err)
	}
	if _, err := os.Stat(filepath.Join(workDir, "demo")); !os.IsNotExist(err) {
		t.Fatal("dry-run created destination")
	}
	if !strings.Contains(stdout.String(), "demo/go.mod") {
		t.Fatalf("stdout = %q, want file list", stdout.String())
	}
}

func TestRunNewRequiresForceForNonEmptyDest(t *testing.T) {
	workDir := t.TempDir()
	chdir(t, workDir)
	dest := filepath.Join(workDir, "demo")
	if err := os.MkdirAll(dest, 0o750); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dest, "owned.txt"), []byte("user"), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}

	err := run(context.Background(), []string{"new", "demo", "--database", "sqlite"}, ioDiscard{}, ioDiscard{})
	if err == nil {
		t.Fatal("run() error = nil, want non-empty destination")
	}
	if !strings.Contains(err.Error(), "--force") {
		t.Fatalf("run() error = %q, want --force", err)
	}

	err = run(context.Background(), []string{"new", "demo", "--database", "sqlite", "--force"}, ioDiscard{}, ioDiscard{})
	if err != nil {
		t.Fatalf("run(--force) error = %v", err)
	}
}

func TestRunNewFlagValidation(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{name: "unknown database", args: []string{"new", "demo", "--database", "oracle"}, want: "database"},
		{name: "unknown cache", args: []string{"new", "demo", "--cache", "memcached"}, want: "cache"},
		{name: "unknown auth", args: []string{"new", "demo", "--auth", "oauth"}, want: "auth"},
		{name: "unknown ui", args: []string{"new", "demo", "--ui", "bootstrap"}, want: "ui"},
		{name: "invalid name", args: []string{"new", "../evil", "--database", "sqlite"}, want: "project name"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			chdir(t, t.TempDir())
			err := run(context.Background(), tt.args, ioDiscard{}, ioDiscard{})
			if err == nil {
				t.Fatal("run() error = nil, want validation error")
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("run() error = %q, want %q", err, tt.want)
			}
		})
	}
}

func TestRunNewDemoCompiles(t *testing.T) {
	workDir := t.TempDir()
	chdir(t, workDir)

	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	err := run(context.Background(), []string{"new", "demo", "--database", "sqlite"}, stdout, stderr)
	if err != nil {
		t.Fatalf("run() error = %v; stderr=%q stdout=%q", err, stderr.String(), stdout.String())
	}

	dest := filepath.Join(workDir, "demo")
	goModPath := filepath.Join(dest, "go.mod")
	// #nosec G304 -- goModPath is under t.TempDir
	mod, err := os.OpenFile(goModPath, os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		t.Fatalf("open go.mod: %v", err)
	}
	if _, err := mod.WriteString("\nreplace github.com/LAA-Software-Engineering/gombit => " + cmdModuleRoot(t) + "\n"); err != nil {
		_ = mod.Close()
		t.Fatalf("write replace: %v", err)
	}
	if err := mod.Close(); err != nil {
		t.Fatalf("close go.mod: %v", err)
	}

	tidy := exec.Command("go", "mod", "tidy")
	tidy.Dir = dest
	if out, err := tidy.CombinedOutput(); err != nil {
		t.Fatalf("go mod tidy: %v\n%s", err, out)
	}
	build := exec.Command("go", "build", "./...")
	build.Dir = dest
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("go build: %v\n%s", err, out)
	}

	body := readFileString(t, filepath.Join(dest, "internal", "product", "routes.go"))
	if !strings.Contains(body, "/products") {
		t.Fatalf("routes.go missing products path:\n%s", body)
	}
	mainSrc := readFileString(t, filepath.Join(dest, "cmd", "server", "main.go"))
	if !strings.Contains(mainSrc, "product.Register") {
		t.Fatal("cmd/server/main.go does not register product routes explicitly")
	}
	if !strings.Contains(mainSrc, "framework.New") {
		t.Fatal("cmd/server/main.go does not use framework.New")
	}
	cliSrc := readFileString(t, filepath.Join(dest, "cmd", "gombit", "main.go"))
	if !strings.Contains(cliSrc, "product.RegisterCommands") {
		t.Fatal("cmd/gombit/main.go does not call product.RegisterCommands")
	}
	if !strings.Contains(cliSrc, "cli.NewRoot") {
		t.Fatal("cmd/gombit/main.go does not use cli.NewRoot")
	}
	help := exec.Command("go", "run", "./cmd/gombit", "--help")
	help.Dir = dest
	out, err := help.CombinedOutput()
	if err != nil {
		t.Fatalf("go run ./cmd/gombit --help: %v\n%s", err, out)
	}
	if !strings.Contains(string(out), "make") {
		t.Fatalf("app gombit --help missing make:\n%s", out)
	}
}

func chdir(t *testing.T, dir string) {
	t.Helper()
	previous, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(previous) })
}
