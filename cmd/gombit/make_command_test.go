package main

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/LAA-Software-Engineering/gombit/commandgen"
)

func TestRunMakeCommandGreetRunnable(t *testing.T) {
	workDir := t.TempDir()
	chdir(t, workDir)

	if err := run(context.Background(), []string{"new", "demo", "--database", "sqlite"}, ioDiscard{}, ioDiscard{}); err != nil {
		t.Fatalf("gombit new: %v", err)
	}
	dest := filepath.Join(workDir, "demo")
	appendReplace(t, dest)
	chdir(t, dest)

	stdout := new(bytes.Buffer)
	err := run(context.Background(), []string{"make", "command", "greet"}, stdout, ioDiscard{})
	if err != nil {
		t.Fatalf("make command: %v; stdout=%q", err, stdout.String())
	}
	if !strings.Contains(stdout.String(), "internal/commands/greet.go") {
		t.Fatalf("stdout = %q, want greet.go", stdout.String())
	}

	tidy := exec.Command("go", "mod", "tidy")
	tidy.Dir = dest
	if out, err := tidy.CombinedOutput(); err != nil {
		t.Fatalf("go mod tidy: %v\n%s", err, out)
	}

	greet := exec.Command("go", "run", "./cmd/gombit", "greet")
	greet.Dir = dest
	out, err := greet.CombinedOutput()
	if err != nil {
		t.Fatalf("go run ./cmd/gombit greet: %v\n%s", err, out)
	}
	if !strings.Contains(string(out), "greet: ok") {
		t.Fatalf("greet output = %q, want greet: ok", out)
	}

	help := exec.Command("go", "run", "./cmd/gombit", "--help")
	help.Dir = dest
	out, err = help.CombinedOutput()
	if err != nil {
		t.Fatalf("go run ./cmd/gombit --help: %v\n%s", err, out)
	}
	if !strings.Contains(string(out), "greet") {
		t.Fatalf("app --help missing greet:\n%s", out)
	}

	mainSrc := readFileString(t, filepath.Join(dest, "cmd", "gombit", "main.go"))
	count, err := commandgen.CountRegisterCalls([]byte(mainSrc), "commands")
	if err != nil {
		t.Fatalf("CountRegisterCalls: %v", err)
	}
	if count != 1 {
		t.Fatalf("commands.Register count = %d, want 1\n%s", count, mainSrc)
	}

	err = run(context.Background(), []string{"make", "command", "greet"}, ioDiscard{}, ioDiscard{})
	if err != nil {
		t.Fatalf("re-run make command: %v", err)
	}
	mainSrc = readFileString(t, filepath.Join(dest, "cmd", "gombit", "main.go"))
	count, err = commandgen.CountRegisterCalls([]byte(mainSrc), "commands")
	if err != nil {
		t.Fatalf("CountRegisterCalls re-run: %v", err)
	}
	if count != 1 {
		t.Fatalf("re-run duplicated Register: %d", count)
	}

	dryStdout := new(bytes.Buffer)
	err = run(context.Background(), []string{"make", "command", "hello", "--dry-run"}, dryStdout, ioDiscard{})
	if err != nil {
		t.Fatalf("dry-run: %v", err)
	}
	if !strings.Contains(dryStdout.String(), "hello.go") {
		t.Fatalf("dry-run stdout = %q", dryStdout.String())
	}
	if _, err := os.Stat(filepath.Join(dest, "internal", "commands", "hello.go")); !os.IsNotExist(err) {
		t.Fatal("dry-run wrote hello.go")
	}
}

func TestRunMakeCommandHelp(t *testing.T) {
	stdout := new(bytes.Buffer)
	err := run(context.Background(), []string{"make", "command", "--help"}, stdout, ioDiscard{})
	if err != nil {
		t.Fatalf("make command --help: %v", err)
	}
	got := stdout.String()
	if !strings.Contains(got, "--dry-run") || !strings.Contains(got, "--force") {
		t.Fatalf("make command help missing flags:\n%s", got)
	}
	if !strings.Contains(got, "--package") {
		t.Fatalf("make command help missing --package:\n%s", got)
	}
}
