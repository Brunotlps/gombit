package migrations

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/LAA-Software-Engineering/gombit/config"
)

func TestParseModel(t *testing.T) {
	tests := []struct {
		name    string
		spec    string
		want    Model
		wantErr bool
	}{
		{
			name: "valid",
			spec: "github.com/example/app/internal/product.Product",
			want: Model{
				ImportPath: "github.com/example/app/internal/product",
				TypeName:   "Product",
			},
		},
		{name: "empty", spec: "", wantErr: true},
		{name: "missing type", spec: "github.com/example/app/internal/product.", wantErr: true},
		{name: "missing import", spec: ".Product", wantErr: true},
		{name: "bad type", spec: "github.com/example/app/internal/product.123Product", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseModel(tt.spec)
			if tt.wantErr {
				if err == nil {
					t.Fatal("ParseModel() error = nil, want error")
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseModel() error = %v, want nil", err)
			}
			if got != tt.want {
				t.Fatalf("ParseModel() = %#v, want %#v", got, tt.want)
			}
		})
	}
}

func TestMakeMigrationsRunsAtlasDiffForSupportedDrivers(t *testing.T) {
	tests := []struct {
		driver config.DatabaseDriver
		devURL string
	}{
		{driver: config.DatabaseDriverSQLite, devURL: "sqlite://file?mode=memory&_fk=1"},
		{driver: config.DatabaseDriverPostgres, devURL: "docker://postgres/15/dev?search_path=public"},
		{driver: config.DatabaseDriverMySQL, devURL: "docker://mysql/8/dev"},
	}

	for _, tt := range tests {
		t.Run(string(tt.driver), func(t *testing.T) {
			workDir := t.TempDir()
			runner := &recordingRunner{t: t}
			stdout := new(bytes.Buffer)

			err := MakeMigrations(context.Background(), Options{
				WorkDir:      workDir,
				Name:         "create_products",
				Driver:       tt.driver,
				MigrationDir: "database/migrations",
				AtlasBinary:  "atlas-test",
				Models: []Model{
					{ImportPath: "github.com/example/app/internal/product", TypeName: "Product"},
					{ImportPath: "github.com/example/app/internal/account", TypeName: "Account"},
				},
				Stdout: stdout,
				runner: runner,
			})
			if err != nil {
				t.Fatalf("MakeMigrations() error = %v, want nil", err)
			}

			wantArgs := []string{"migrate", "diff", "create_products", "--env", "gombit", "--config"}
			if len(runner.args) != len(wantArgs)+1 {
				t.Fatalf("atlas args = %v, want prefix %v plus config URL", runner.args, wantArgs)
			}
			for i, want := range wantArgs {
				if runner.args[i] != want {
					t.Fatalf("atlas args[%d] = %q, want %q (all args: %v)", i, runner.args[i], want, runner.args)
				}
			}
			if runner.dir != workDir {
				t.Fatalf("atlas dir = %q, want %q", runner.dir, workDir)
			}
			if runner.name != "atlas-test" {
				t.Fatalf("atlas name = %q, want atlas-test", runner.name)
			}

			atlasHCL := runner.atlasHCL
			for _, want := range []string{
				`env "gombit"`,
				`"./.gombit/`,
				tt.devURL,
				"file://" + filepath.ToSlash(filepath.Join(workDir, "database/migrations")),
			} {
				if !strings.Contains(atlasHCL, want) {
					t.Fatalf("atlas.hcl = %q, want it to contain %q", atlasHCL, want)
				}
			}

			loader := runner.loader
			for _, want := range []string{
				`"ariga.io/atlas-provider-gorm/gormschema"`,
				`model0 "github.com/example/app/internal/product"`,
				`model1 "github.com/example/app/internal/account"`,
				`gormschema.New("` + string(tt.driver) + `").Load(`,
				"&model0.Product{}",
				"&model1.Account{}",
			} {
				if !strings.Contains(loader, want) {
					t.Fatalf("loader = %q, want it to contain %q", loader, want)
				}
			}

			migrationFile := filepath.Join(workDir, "database/migrations", "20260101000000_create_products.sql")
			if _, err := os.Stat(migrationFile); err != nil {
				t.Fatalf("expected fake atlas migration file: %v", err)
			}
			if !strings.Contains(stdout.String(), "created migration") {
				t.Fatalf("stdout = %q, want fake atlas output", stdout.String())
			}
		})
	}
}

func TestMakeMigrationsValidatesOptions(t *testing.T) {
	tests := []struct {
		name string
		opts Options
	}{
		{name: "missing name", opts: Options{Driver: config.DatabaseDriverSQLite, Models: []Model{{ImportPath: "example.com/app/product", TypeName: "Product"}}}},
		{name: "unsupported driver", opts: Options{Name: "x", Driver: "oracle", Models: []Model{{ImportPath: "example.com/app/product", TypeName: "Product"}}}},
		{name: "missing models", opts: Options{Name: "x", Driver: config.DatabaseDriverSQLite}},
		{name: "invalid model", opts: Options{Name: "x", Driver: config.DatabaseDriverSQLite, Models: []Model{{ImportPath: "example.com/app/product", TypeName: "123"}}}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := MakeMigrations(context.Background(), tt.opts); err == nil {
				t.Fatal("MakeMigrations() error = nil, want error")
			}
		})
	}
}

type recordingRunner struct {
	t    *testing.T
	dir  string
	name string
	args []string

	atlasHCL string
	loader   string
}

func (r *recordingRunner) Run(_ context.Context, dir string, name string, args []string, stdout io.Writer, stderr io.Writer) error {
	r.dir = dir
	r.name = name
	r.args = append([]string(nil), args...)
	r.atlasHCL = r.readConfig()
	r.loader = r.readLoader(r.atlasHCL)
	migrationDir := filepath.Join(dir, "database/migrations")
	if err := os.MkdirAll(migrationDir, 0o750); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(migrationDir, "20260101000000_create_products.sql"), []byte("-- fake atlas migration\n"), 0o600); err != nil {
		return err
	}
	_, _ = stdout.Write([]byte("created migration\n"))
	return nil
}

func (r *recordingRunner) readConfig() string {
	r.t.Helper()
	if len(r.args) == 0 {
		r.t.Fatal("atlas args are empty")
	}
	configURL := r.args[len(r.args)-1]
	const prefix = "file://"
	if !strings.HasPrefix(configURL, prefix) {
		r.t.Fatalf("config arg = %q, want file URL", configURL)
	}
	data, err := os.ReadFile(strings.TrimPrefix(configURL, prefix))
	if err != nil {
		r.t.Fatalf("read atlas config: %v", err)
	}
	return string(data)
}

func (r *recordingRunner) readLoader(atlasHCL string) string {
	r.t.Helper()
	start := strings.Index(atlasHCL, `"./.gombit/`)
	if start < 0 {
		r.t.Fatalf("atlas.hcl = %q, want loader path", atlasHCL)
	}
	start += len(`"./`)
	end := strings.Index(atlasHCL[start:], `"`)
	if end < 0 {
		r.t.Fatalf("atlas.hcl = %q, want quoted loader path", atlasHCL)
	}
	loaderRel := atlasHCL[start : start+end]
	data, err := os.ReadFile(filepath.Join(r.dir, filepath.FromSlash(loaderRel), "main.go"))
	if err != nil {
		r.t.Fatalf("read loader: %v", err)
	}
	return string(data)
}
