package cli

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func TestMakeMigrationsRejectsInvalidForgetModel(t *testing.T) {
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	err := ExecuteRoot(context.Background(), NewRoot(stdout, stderr), []string{
		"db", "makemigrations", "create_products",
		"--model", "github.com/example/app/internal/product.Product",
		"--forget-model", "not-a-valid-spec",
	})
	if err == nil {
		t.Fatal("gombit db makemigrations --forget-model not-a-valid-spec: error = nil, want error")
	}
	if !strings.Contains(err.Error(), "import/path.TypeName") {
		t.Fatalf("error = %q, want it to explain the model spec format", err)
	}
}

func TestMakeMigrationsHelpDescribesForgetModel(t *testing.T) {
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	err := ExecuteRoot(context.Background(), NewRoot(stdout, stderr), []string{"db", "makemigrations", "--help"})
	if err != nil {
		t.Fatalf("gombit db makemigrations --help: %v", err)
	}
	out := stdout.String() + stderr.String()
	if !strings.Contains(out, "--forget-model") {
		t.Fatalf("help missing --forget-model:\n%s", out)
	}
	if !strings.Contains(out, "Merged with models already registered") {
		t.Fatalf("help missing note about the persisted model registry:\n%s", out)
	}
}
