package scaffold

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGenerateWritesFeaturePackageLayout(t *testing.T) {
	workDir := t.TempDir()
	stdout := new(bytes.Buffer)
	err := Generate(context.Background(), Options{
		Name:     "demo",
		Database: "sqlite",
		WorkDir:  workDir,
		Stdout:   stdout,
	})
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	dest := filepath.Join(workDir, "demo")
	wantFiles := []string{
		"cmd/server/main.go",
		"internal/platform/database.go",
		"internal/product/product.go",
		"internal/product/handler.go",
		"internal/product/routes.go",
		"database/migrations/.gitkeep",
		"database/seeds/.gitkeep",
		"config/README.md",
		"frontend/package.json",
		"frontend/vite.config.ts",
		"frontend/index.html",
		"frontend/src/main.ts",
		"frontend/src/vite-env.d.ts",
		"frontend/tsconfig.json",
		"frontend/README.md",
		".air.toml",
		"gombit.yaml",
		".env.example",
		"go.mod",
		"README.md",
		".gitignore",
	}
	for _, rel := range wantFiles {
		path := filepath.Join(dest, filepath.FromSlash(rel))
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("missing %s: %v", rel, err)
		}
	}
	for _, rel := range []string{"internal/product/service.go", "internal/product/repo.go"} {
		path := filepath.Join(dest, filepath.FromSlash(rel))
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Fatalf("unexpected generated file %s", rel)
		}
	}

	goMod := readFile(t, filepath.Join(dest, "go.mod"))
	if !strings.Contains(goMod, "module github.com/example/demo") {
		t.Fatalf("go.mod module = %q, want github.com/example/demo", goMod)
	}
	if !strings.Contains(goMod, "github.com/LAA-Software-Engineering/gombit") {
		t.Fatalf("go.mod missing gombit require:\n%s", goMod)
	}
	if strings.Contains(goMod, "replace ") {
		t.Fatal("generated go.mod must not include a replace directive")
	}

	yaml := readFile(t, filepath.Join(dest, "gombit.yaml"))
	for _, want := range []string{"database: sqlite", "cache: memory", "auth: jwt", "ui: minimal"} {
		if !strings.Contains(yaml, want) {
			t.Fatalf("gombit.yaml missing %q:\n%s", want, yaml)
		}
	}

	envExample := readFile(t, filepath.Join(dest, ".env.example"))
	if !strings.Contains(envExample, "GOMBIT_DATABASE_DRIVER=sqlite") {
		t.Fatalf(".env.example missing sqlite driver:\n%s", envExample)
	}
	if !strings.Contains(envExample, "VITE_API_URL=http://127.0.0.1:8080/api/v1") {
		t.Fatalf(".env.example missing public VITE_API_URL:\n%s", envExample)
	}
	if !strings.Contains(stdout.String(), "create demo/go.mod") {
		t.Fatalf("stdout = %q, want created file list", stdout.String())
	}

	viteConfig := readFile(t, filepath.Join(dest, "frontend", "vite.config.ts"))
	for _, want := range []string{"/api", "/openapi.json", "/docs"} {
		if !strings.Contains(viteConfig, want) {
			t.Fatalf("vite.config.ts missing proxy %q:\n%s", want, viteConfig)
		}
	}
	pkg := readFile(t, filepath.Join(dest, "frontend", "package.json"))
	if !strings.Contains(pkg, "vite") {
		t.Fatalf("frontend/package.json missing vite:\n%s", pkg)
	}
	if strings.Contains(pkg, "react-router") || strings.Contains(pkg, "react-hook-form") {
		t.Fatal("frontend stub must not include React Router or React Hook Form (M5-1)")
	}
	mainTS := readFile(t, filepath.Join(dest, "frontend", "src", "main.ts"))
	if strings.Contains(strings.ToLower(mainTS), "localstorage") {
		t.Fatal("frontend/src/main.ts uses localStorage")
	}
}

func TestGenerateDryRunWritesNothing(t *testing.T) {
	workDir := t.TempDir()
	stdout := new(bytes.Buffer)
	err := Generate(context.Background(), Options{
		Name:     "demo",
		Database: "sqlite",
		WorkDir:  workDir,
		DryRun:   true,
		Stdout:   stdout,
	})
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if _, err := os.Stat(filepath.Join(workDir, "demo")); !os.IsNotExist(err) {
		t.Fatal("dry-run created destination directory")
	}
	if !strings.Contains(stdout.String(), "create demo/go.mod") {
		t.Fatalf("stdout = %q, want file list", stdout.String())
	}
}

func TestGenerateRefusesNonEmptyDestinationWithoutForce(t *testing.T) {
	workDir := t.TempDir()
	dest := filepath.Join(workDir, "demo")
	if err := os.MkdirAll(dest, 0o750); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dest, "owned.txt"), []byte("user"), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}

	err := Generate(context.Background(), Options{
		Name:     "demo",
		Database: "sqlite",
		WorkDir:  workDir,
	})
	if err == nil {
		t.Fatal("Generate() error = nil, want non-empty destination")
	}
	if !strings.Contains(err.Error(), "--force") {
		t.Fatalf("Generate() error = %q, want --force hint", err)
	}
	if _, err := os.Stat(filepath.Join(dest, "go.mod")); !os.IsNotExist(err) {
		t.Fatal("refused generate still wrote go.mod")
	}

	err = Generate(context.Background(), Options{
		Name:     "demo",
		Database: "sqlite",
		WorkDir:  workDir,
		Force:    true,
	})
	if err != nil {
		t.Fatalf("Generate(--force) error = %v", err)
	}
	if _, err := os.Stat(filepath.Join(dest, "go.mod")); err != nil {
		t.Fatalf("force did not write go.mod: %v", err)
	}
	owned := readFile(t, filepath.Join(dest, "owned.txt"))
	if owned != "user" {
		t.Fatalf("force overwrote user-owned file: %q", owned)
	}
}

func TestGenerateFlagValidation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		opts Options
		want string
	}{
		{
			name: "unknown database",
			opts: Options{Name: "demo", Database: "oracle"},
			want: "database",
		},
		{
			name: "unknown cache",
			opts: Options{Name: "demo", Cache: "memcached"},
			want: "cache",
		},
		{
			name: "unknown auth",
			opts: Options{Name: "demo", Auth: "oauth"},
			want: "auth",
		},
		{
			name: "unknown ui",
			opts: Options{Name: "demo", UI: "bootstrap"},
			want: "ui",
		},
		{
			name: "path name",
			opts: Options{Name: "../evil"},
			want: "project name",
		},
		{
			name: "missing name non-interactive",
			opts: Options{Database: "sqlite", IsTTY: func() bool { return false }},
			want: "project name is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			tt.opts.WorkDir = t.TempDir()
			err := Generate(context.Background(), tt.opts)
			if err == nil {
				t.Fatal("Generate() error = nil, want validation error")
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("Generate() error = %q, want %q", err, tt.want)
			}
		})
	}
}

func TestGenerateRecordsAuthAndUIChoices(t *testing.T) {
	workDir := t.TempDir()
	err := Generate(context.Background(), Options{
		Name:     "shop",
		Module:   "example.com/shop",
		Database: "postgres",
		Cache:    "redis",
		Auth:     "cookie",
		UI:       "mui",
		WorkDir:  workDir,
	})
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	yaml := readFile(t, filepath.Join(workDir, "shop", "gombit.yaml"))
	for _, want := range []string{
		"module: example.com/shop",
		"database: postgres",
		"cache: redis",
		"auth: cookie",
		"ui: mui",
	} {
		if !strings.Contains(yaml, want) {
			t.Fatalf("gombit.yaml missing %q:\n%s", want, yaml)
		}
	}
	envExample := readFile(t, filepath.Join(workDir, "shop", ".env.example"))
	if !strings.Contains(envExample, "GOMBIT_DATABASE_DRIVER=postgres") {
		t.Fatalf(".env.example missing postgres driver:\n%s", envExample)
	}
	if !strings.Contains(envExample, "USER:PASSWORD") {
		t.Fatalf(".env.example missing DSN placeholders:\n%s", envExample)
	}
}

func TestGenerateSourceHasNoLocalStorageOrViteSecrets(t *testing.T) {
	workDir := t.TempDir()
	err := Generate(context.Background(), Options{
		Name:     "demo",
		Database: "sqlite",
		WorkDir:  workDir,
	})
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	err = filepath.Walk(filepath.Join(workDir, "demo"), func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		if !isGeneratedSource(path) {
			return nil
		}
		body := readFile(t, path)
		lower := strings.ToLower(body)
		if strings.Contains(lower, "localstorage") || strings.Contains(lower, "sessionstorage") {
			t.Errorf("%s contains localStorage/sessionStorage", path)
		}
		for _, line := range strings.Split(body, "\n") {
			trimmed := strings.TrimSpace(line)
			if !strings.HasPrefix(trimmed, "VITE_") {
				continue
			}
			if strings.Contains(strings.ToLower(trimmed), "secret") || strings.Contains(strings.ToLower(trimmed), "password") {
				t.Errorf("%s puts a secret in VITE_*: %s", path, trimmed)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk: %v", err)
	}
}

func TestGenerateInteractivePromptsOnTTY(t *testing.T) {
	workDir := t.TempDir()
	stdin := strings.NewReader("demo\nsqlite\nmemory\njwt\nminimal\n\n")
	stdout := new(bytes.Buffer)
	err := Generate(context.Background(), Options{
		WorkDir: workDir,
		Stdin:   stdin,
		Stdout:  stdout,
		IsTTY:   func() bool { return true },
		DryRun:  true,
	})
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if !strings.Contains(stdout.String(), "Project name") {
		t.Fatalf("stdout = %q, want interactive prompts", stdout.String())
	}
	if !strings.Contains(stdout.String(), "create demo/go.mod") {
		t.Fatalf("stdout = %q, want dry-run file list for prompted name", stdout.String())
	}
}

func isGeneratedSource(path string) bool {
	base := filepath.Base(path)
	switch strings.ToLower(filepath.Ext(path)) {
	case ".go", ".ts", ".tsx", ".js", ".jsx", ".json", ".html", ".mjs":
		return true
	}
	return base == ".env.example" || base == "gombit.yaml"
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
