package cli

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func TestBuildWithoutEmbedRefusesSplitDefault(t *testing.T) {
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	err := ExecuteRoot(context.Background(), NewRoot(stdout, stderr), []string{"build"})
	if err == nil {
		t.Fatal("gombit build error = nil, want --embed required")
	}
	msg := err.Error()
	if !strings.Contains(msg, "--embed") {
		t.Fatalf("error = %q, want --embed", msg)
	}
	if !strings.Contains(msg, "split") {
		t.Fatalf("error = %q, want split default", msg)
	}
}

func TestBuildEmbedFalseRefuses(t *testing.T) {
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	err := ExecuteRoot(context.Background(), NewRoot(stdout, stderr), []string{"build", "--embed=false"})
	if err == nil {
		t.Fatal("gombit build --embed=false error = nil, want refusal")
	}
	if !strings.Contains(err.Error(), "--embed") {
		t.Fatalf("error = %q, want --embed", err)
	}
}

func TestBuildHelpDescribesEmbed(t *testing.T) {
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	err := ExecuteRoot(context.Background(), NewRoot(stdout, stderr), []string{"build", "--help"})
	if err != nil {
		t.Fatalf("gombit build --help: %v", err)
	}
	out := stdout.String() + stderr.String()
	for _, want := range []string{"--embed", "--out", "--dry-run", "split", "collectstatic"} {
		if !strings.Contains(out, want) {
			t.Fatalf("help missing %q:\n%s", want, out)
		}
	}
}

func TestRootHelpListsBuild(t *testing.T) {
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	err := ExecuteRoot(context.Background(), NewRoot(stdout, stderr), []string{"--help"})
	if err != nil {
		t.Fatalf("gombit --help: %v", err)
	}
	out := stdout.String() + stderr.String()
	if !strings.Contains(out, "build") {
		t.Fatalf("root help missing build:\n%s", out)
	}
}

func TestRootUsageListsBuild(t *testing.T) {
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	err := ExecuteRoot(context.Background(), NewRoot(stdout, stderr), []string{})
	if err == nil {
		t.Fatal("gombit with no command: error = nil, want command required")
	}
	out := stderr.String()
	if !strings.Contains(out, "build") {
		t.Fatalf("usage missing build:\n%s", out)
	}
}
