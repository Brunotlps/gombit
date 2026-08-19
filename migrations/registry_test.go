package migrations

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestLoadRegistryMissingFileIsEmpty(t *testing.T) {
	models, err := LoadRegistry(t.TempDir())
	if err != nil {
		t.Fatalf("LoadRegistry() error = %v, want nil", err)
	}
	if models != nil {
		t.Fatalf("LoadRegistry() = %#v, want nil", models)
	}
}

func TestSaveThenLoadRegistryRoundTrips(t *testing.T) {
	dir := t.TempDir()
	want := []Model{
		{ImportPath: "github.com/example/app/internal/account", TypeName: "Account"},
		{ImportPath: "github.com/example/app/internal/product", TypeName: "Product"},
	}
	if err := SaveRegistry(dir, want); err != nil {
		t.Fatalf("SaveRegistry() error = %v", err)
	}
	got, err := LoadRegistry(dir)
	if err != nil {
		t.Fatalf("LoadRegistry() error = %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("LoadRegistry() = %#v, want %#v", got, want)
	}
}

func TestSaveRegistrySortsRegardlessOfInputOrder(t *testing.T) {
	dir := t.TempDir()
	unsorted := []Model{
		{ImportPath: "github.com/example/app/internal/task", TypeName: "Task"},
		{ImportPath: "github.com/example/app/internal/account", TypeName: "Account"},
	}
	if err := SaveRegistry(dir, unsorted); err != nil {
		t.Fatalf("SaveRegistry() error = %v", err)
	}
	got, err := LoadRegistry(dir)
	if err != nil {
		t.Fatalf("LoadRegistry() error = %v", err)
	}
	want := []Model{
		{ImportPath: "github.com/example/app/internal/account", TypeName: "Account"},
		{ImportPath: "github.com/example/app/internal/task", TypeName: "Task"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("LoadRegistry() = %#v, want sorted %#v", got, want)
	}
}

func TestLoadRegistryRejectsInvalidEntries(t *testing.T) {
	dir := t.TempDir()
	// #nosec G306 -- test fixture
	if err := os.WriteFile(RegistryPath(dir), []byte(`{"models":[{"import_path":"","type_name":"Product"}]}`), 0o600); err != nil {
		t.Fatalf("write registry fixture: %v", err)
	}
	if _, err := LoadRegistry(dir); err == nil {
		t.Fatal("LoadRegistry() error = nil, want error for invalid entry")
	}
}

func TestLoadRegistryRejectsMalformedJSON(t *testing.T) {
	dir := t.TempDir()
	// #nosec G306 -- test fixture
	if err := os.WriteFile(RegistryPath(dir), []byte(`not json`), 0o600); err != nil {
		t.Fatalf("write registry fixture: %v", err)
	}
	if _, err := LoadRegistry(dir); err == nil {
		t.Fatal("LoadRegistry() error = nil, want error for malformed JSON")
	}
}

func TestRegistryPathIsInsideMigrationDir(t *testing.T) {
	got := RegistryPath("database/migrations")
	want := filepath.Join("database/migrations", "models.json")
	if got != want {
		t.Fatalf("RegistryPath() = %q, want %q", got, want)
	}
}

func TestMergeModels(t *testing.T) {
	product := Model{ImportPath: "github.com/example/app/internal/product", TypeName: "Product"}
	account := Model{ImportPath: "github.com/example/app/internal/account", TypeName: "Account"}
	task := Model{ImportPath: "github.com/example/app/internal/task", TypeName: "Task"}

	tests := []struct {
		name       string
		existing   []Model
		additional []Model
		want       []Model
	}{
		{
			name:       "empty registry keeps additional order",
			existing:   nil,
			additional: []Model{product, account},
			want:       []Model{product, account},
		},
		{
			name:       "new model is unioned with the registry, not replacing it",
			existing:   []Model{product, account},
			additional: []Model{task},
			want:       []Model{task, account, product},
		},
		{
			name:       "duplicate between existing and additional is not repeated",
			existing:   []Model{product},
			additional: []Model{product, task},
			want:       []Model{product, task},
		},
		{
			name:       "no new models still returns the registry",
			existing:   []Model{product, account},
			additional: nil,
			want:       []Model{account, product},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MergeModels(tt.existing, tt.additional)
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("MergeModels() = %#v, want %#v", got, tt.want)
			}
		})
	}
}

func TestSubtractModels(t *testing.T) {
	product := Model{ImportPath: "github.com/example/app/internal/product", TypeName: "Product"}
	account := Model{ImportPath: "github.com/example/app/internal/account", TypeName: "Account"}
	task := Model{ImportPath: "github.com/example/app/internal/task", TypeName: "Task"}

	tests := []struct {
		name   string
		models []Model
		remove []Model
		want   []Model
	}{
		{
			name:   "removes a matching model",
			models: []Model{product, account, task},
			remove: []Model{account},
			want:   []Model{product, task},
		},
		{
			name:   "no-op when remove is empty",
			models: []Model{product, account},
			remove: nil,
			want:   []Model{product, account},
		},
		{
			name:   "removing everything yields empty, not nil-vs-empty ambiguity",
			models: []Model{product},
			remove: []Model{product},
			want:   []Model{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SubtractModels(tt.models, tt.remove)
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("SubtractModels() = %#v, want %#v", got, tt.want)
			}
		})
	}
}
