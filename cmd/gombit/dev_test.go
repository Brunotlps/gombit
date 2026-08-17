package main

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/LAA-Software-Engineering/gombit/cli"
	"github.com/LAA-Software-Engineering/gombit/config"
	"github.com/LAA-Software-Engineering/gombit/dev"
)

func TestRunHelpListsDev(t *testing.T) {
	stdout := new(bytes.Buffer)
	err := run(context.Background(), []string{"--help"}, stdout, ioDiscard{})
	if err != nil {
		t.Fatalf("run(--help) error = %v", err)
	}
	if !strings.Contains(stdout.String(), "dev") {
		t.Fatalf("root help missing dev:\n%s", stdout.String())
	}
}

func TestRunDevHelp(t *testing.T) {
	stdout := new(bytes.Buffer)
	err := run(context.Background(), []string{"dev", "--help"}, stdout, ioDiscard{})
	if err != nil {
		t.Fatalf("run(dev --help) error = %v", err)
	}
	got := stdout.String()
	for _, want := range []string{"--http", "--frontend-port", "--frontend-host", "--poll", "--client-out", "/docs"} {
		if !strings.Contains(got, want) {
			t.Fatalf("dev help missing %q:\n%s", want, got)
		}
	}
}

func TestRunDevFlagValidation(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{name: "zero frontend port", args: []string{"dev", "--frontend-port", "0"}, want: "--frontend-port"},
		{name: "negative frontend port", args: []string{"dev", "--frontend-port", "-1"}, want: "--frontend-port"},
		{name: "frontend port too large", args: []string{"dev", "--frontend-port", "70000"}, want: "--frontend-port"},
		{name: "empty http", args: []string{"dev", "--http", " "}, want: "--http"},
		{name: "zero poll", args: []string{"dev", "--poll", "0s"}, want: "--poll"},
		{name: "empty client out", args: []string{"dev", "--client-out", " "}, want: "--client-out"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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

func TestRunDevMissingFrontendPackageJSON(t *testing.T) {
	chdir(t, t.TempDir())
	if err := os.MkdirAll(filepath.Join("cmd", "server"), 0o750); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	err := run(context.Background(), []string{"dev"}, ioDiscard{}, ioDiscard{})
	if err == nil {
		t.Fatal("run() error = nil, want missing frontend/package.json")
	}
	if !strings.Contains(err.Error(), "frontend/package.json") {
		t.Fatalf("run() error = %q, want frontend/package.json", err)
	}
}

func TestRunDevUsesConfiguredHTTPAddr(t *testing.T) {
	cfg := config.Default()
	cfg.HTTP.Addr = "127.0.0.1:19090"
	stubConfig(t, cfg)

	var got dev.Options
	previous := cli.RunDev
	cli.RunDev = func(ctx context.Context, opts dev.Options) error {
		got = opts
		return errors.New("stopped")
	}
	t.Cleanup(func() { cli.RunDev = previous })

	chdir(t, t.TempDir())
	err := run(context.Background(), []string{"dev"}, ioDiscard{}, ioDiscard{})
	if err == nil || !strings.Contains(err.Error(), "stopped") {
		t.Fatalf("run() error = %v, want stopped", err)
	}
	if got.HTTPAddr != "127.0.0.1:19090" {
		t.Fatalf("HTTPAddr = %q, want configured addr", got.HTTPAddr)
	}
}

func TestRunDevHTTPFlagOverridesConfig(t *testing.T) {
	cfg := config.Default()
	cfg.HTTP.Addr = ":8080"
	stubConfig(t, cfg)

	var got dev.Options
	previous := cli.RunDev
	cli.RunDev = func(ctx context.Context, opts dev.Options) error {
		got = opts
		return errors.New("stopped")
	}
	t.Cleanup(func() { cli.RunDev = previous })

	chdir(t, t.TempDir())
	err := run(context.Background(), []string{"dev", "--http", ":9090"}, ioDiscard{}, ioDiscard{})
	if err == nil || !strings.Contains(err.Error(), "stopped") {
		t.Fatalf("run() error = %v, want stopped", err)
	}
	if got.HTTPAddr != ":9090" {
		t.Fatalf("HTTPAddr = %q, want :9090", got.HTTPAddr)
	}
	if got.PollInterval != time.Second {
		t.Fatalf("PollInterval = %s, want 1s", got.PollInterval)
	}
}

func TestRunRejectsUnknownCommandUsageListsDev(t *testing.T) {
	stderr := new(bytes.Buffer)
	err := run(context.Background(), []string{"unknown"}, ioDiscard{}, stderr)
	if err == nil {
		t.Fatal("run() error = nil, want unknown command error")
	}
	if !strings.Contains(stderr.String(), "dev") {
		t.Fatalf("usage = %q, want dev", stderr.String())
	}
}
