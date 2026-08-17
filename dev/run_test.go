package dev

import (
	"bytes"
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestRunMissingFrontendPackageJSON(t *testing.T) {
	t.Parallel()

	workDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(workDir, "cmd", "server"), 0o750); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	err := Run(context.Background(), Options{WorkDir: workDir})
	if err == nil {
		t.Fatal("Run() error = nil, want missing frontend/package.json")
	}
	if !strings.Contains(err.Error(), "frontend/package.json") {
		t.Fatalf("Run() error = %q, want frontend/package.json", err)
	}
	if !strings.Contains(err.Error(), "backend-only") {
		t.Fatalf("Run() error = %q, want backend-only hint", err)
	}
}

func TestRunMissingServer(t *testing.T) {
	t.Parallel()

	workDir := t.TempDir()
	writeFile(t, filepath.Join(workDir, "frontend", "package.json"), `{"name":"frontend"}`)
	err := Run(context.Background(), Options{WorkDir: workDir})
	if err == nil {
		t.Fatal("Run() error = nil, want missing cmd/server")
	}
	if !strings.Contains(err.Error(), "cmd/server") {
		t.Fatalf("Run() error = %q, want cmd/server", err)
	}
}

func TestRunPrintsServiceTableAndShutsDown(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX sh process-group shutdown")
	}

	workDir := writeDevApp(t)
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	var started atomic.Int32
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	startedCh := make(chan struct{}, 2)
	opts := Options{
		WorkDir:      workDir,
		HTTPAddr:     ":18080",
		FrontendPort: 15173,
		PollInterval: 50 * time.Millisecond,
		ShutdownWait: 2 * time.Second,
		Stdout:       stdout,
		Stderr:       stderr,
		LookPath: func(file string) (string, error) {
			switch file {
			case "go", "npm":
				return file, nil
			default:
				return "", errors.New("not found")
			}
		},
		Command: func(name string, args ...string) *exec.Cmd {
			started.Add(1)
			startedCh <- struct{}{}
			return exec.Command("sh", "-c", "echo ready; trap 'exit 0' TERM; sleep 60")
		},
		HTTPGet: func(ctx context.Context, rawURL string) ([]byte, error) {
			return nil, errors.New("backend not ready")
		},
		Generate: func(ctx context.Context, spec []byte) error {
			t.Fatal("Generate should not run when HTTPGet fails")
			return nil
		},
	}

	errCh := make(chan error, 1)
	go func() {
		errCh <- Run(ctx, opts)
	}()

	waitStarts(t, startedCh, 2)
	got := stdout.String()
	if !strings.Contains(got, "/docs") || !strings.Contains(got, "/openapi.json") {
		t.Fatalf("stdout missing service table:\n%s", got)
	}
	if !strings.Contains(got, "http://127.0.0.1:18080") {
		t.Fatalf("stdout missing backend URL:\n%s", got)
	}
	if !strings.Contains(got, "http://127.0.0.1:15173") {
		t.Fatalf("stdout missing frontend URL:\n%s", got)
	}
	if !strings.Contains(stderr.String(), "without reload") {
		t.Fatalf("stderr = %q, want reload hint", stderr.String())
	}
	if started.Load() < 2 {
		t.Fatalf("started %d child commands, want at least 2", started.Load())
	}

	cancel()
	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("Run() error = %v, want nil after cancel", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Run() did not return after cancel")
	}
}

func TestRunProcessesShutdownOnCancel(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX sh process-group shutdown")
	}

	var started atomic.Int32
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	startedCh := make(chan struct{}, 2)
	specs := []ProcSpec{
		{Name: "backend", Path: "sh", Args: []string{"-c", "echo ready; trap 'exit 0' TERM; sleep 60"}},
		{Name: "frontend", Path: "sh", Args: []string{"-c", "echo ready; trap 'exit 0' TERM; sleep 60"}},
	}
	command := func(name string, args ...string) *exec.Cmd {
		started.Add(1)
		startedCh <- struct{}{}
		return exec.Command(name, args...) //nolint:gosec // test helper rebuilds the injected sh command
	}

	errCh := make(chan error, 1)
	go func() {
		errCh <- runProcesses(ctx, specs, ioDiscard{}, ioDiscard{}, command, 2*time.Second)
	}()

	waitStarts(t, startedCh, 2)
	if started.Load() != 2 {
		t.Fatalf("started = %d, want 2", started.Load())
	}
	cancel()
	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("runProcesses() error = %v, want nil after cancel", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("runProcesses() did not return after cancel")
	}
}

type ioDiscard struct{}

func (ioDiscard) Write(p []byte) (int, error) {
	return len(p), nil
}

func writeDevApp(t *testing.T) string {
	t.Helper()
	workDir := t.TempDir()
	writeFile(t, filepath.Join(workDir, "frontend", "package.json"), `{"name":"frontend","scripts":{"dev":"vite"}}`)
	if err := os.MkdirAll(filepath.Join(workDir, "frontend", "node_modules"), 0o750); err != nil {
		t.Fatalf("mkdir node_modules: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(workDir, "cmd", "server"), 0o750); err != nil {
		t.Fatalf("mkdir cmd/server: %v", err)
	}
	writeFile(t, filepath.Join(workDir, "cmd", "server", "main.go"), "package main\nfunc main() {}\n")
	return workDir
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func waitStarts(t *testing.T, startedCh <-chan struct{}, n int) {
	t.Helper()
	deadline := time.After(3 * time.Second)
	for i := 0; i < n; i++ {
		select {
		case <-startedCh:
		case <-deadline:
			t.Fatalf("timed out waiting for %d child starts (got %d)", n, i)
		}
	}
}

func TestPlanBackendPrefersAir(t *testing.T) {
	t.Parallel()
	plan, err := planBackend(t.TempDir(), func(file string) (string, error) {
		if file == "air" {
			return "/usr/bin/air", nil
		}
		return "", errors.New("missing")
	})
	if err != nil {
		t.Fatalf("planBackend() error = %v", err)
	}
	if plan.Name != "air" {
		t.Fatalf("plan.Name = %q, want air", plan.Name)
	}
	if plan.Hint != "" {
		t.Fatalf("plan.Hint = %q, want empty", plan.Hint)
	}
}

func TestPlanBackendPrefersWatchexecOverGoRun(t *testing.T) {
	t.Parallel()
	plan, err := planBackend(t.TempDir(), func(file string) (string, error) {
		switch file {
		case "watchexec":
			return "/usr/bin/watchexec", nil
		case "go":
			return "/usr/bin/go", nil
		default:
			return "", errors.New("missing")
		}
	})
	if err != nil {
		t.Fatalf("planBackend() error = %v", err)
	}
	if plan.Name != "watchexec" {
		t.Fatalf("plan.Name = %q, want watchexec", plan.Name)
	}
}

func TestPlanBackendUsesAirConfigWhenPresent(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, ".air.toml"), "root = \".\"\n")
	plan, err := planBackend(dir, func(file string) (string, error) {
		if file == "air" {
			return "/usr/bin/air", nil
		}
		return "", errors.New("missing")
	})
	if err != nil {
		t.Fatalf("planBackend() error = %v", err)
	}
	if len(plan.Args) != 2 || plan.Args[0] != "-c" || plan.Args[1] != ".air.toml" {
		t.Fatalf("plan.Args = %v, want -c .air.toml", plan.Args)
	}
}

func TestPlanBackendFallsBackToGoRun(t *testing.T) {
	t.Parallel()
	plan, err := planBackend(t.TempDir(), func(file string) (string, error) {
		if file == "go" {
			return "/usr/bin/go", nil
		}
		return "", errors.New("missing")
	})
	if err != nil {
		t.Fatalf("planBackend() error = %v", err)
	}
	if plan.Name != "go" {
		t.Fatalf("plan.Name = %q, want go", plan.Name)
	}
	if !strings.Contains(plan.Hint, "air and watchexec") {
		t.Fatalf("plan.Hint = %q, want reload hint", plan.Hint)
	}
}

func TestDetectPackageManagerPrefersPnpm(t *testing.T) {
	t.Parallel()
	name, _, err := detectPackageManager(t.TempDir(), func(file string) (string, error) {
		switch file {
		case "pnpm":
			return "/usr/bin/pnpm", nil
		case "npm":
			return "/usr/bin/npm", nil
		default:
			return "", errors.New("missing")
		}
	})
	if err != nil {
		t.Fatalf("detectPackageManager() error = %v", err)
	}
	if name != "pnpm" {
		t.Fatalf("manager = %q, want pnpm", name)
	}
}

func TestDetectPackageManagerRequiresNode(t *testing.T) {
	t.Parallel()
	_, _, err := detectPackageManager(t.TempDir(), func(string) (string, error) {
		return "", errors.New("missing")
	})
	if err == nil {
		t.Fatal("detectPackageManager() error = nil, want npm/pnpm missing")
	}
	if !strings.Contains(err.Error(), "npm and pnpm") {
		t.Fatalf("error = %q, want npm and pnpm", err)
	}
}
