package migrations

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

// registryFileName is the model registry Gombit persists inside the
// migration directory, alongside the versioned SQL files and atlas.sum. It
// is not an Atlas file and Atlas ignores it; ListMigrationFiles skips it the
// same way it skips atlas.sum.
const registryFileName = "models.json"

// modelRegistry is the on-disk JSON shape of the persisted model set.
type modelRegistry struct {
	Models []Model `json:"models"`
}

// RegistryPath returns the path to the persisted model registry inside
// migrationDir.
func RegistryPath(migrationDir string) string {
	return filepath.Join(migrationDir, registryFileName)
}

// LoadRegistry reads the model set previously persisted in migrationDir. A
// missing file is not an error — it means no migration has been generated
// there yet — and returns a nil slice.
func LoadRegistry(migrationDir string) ([]Model, error) {
	path := RegistryPath(migrationDir)
	// #nosec G304 -- path is derived from the configured migration directory.
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("migrations: read model registry %s: %w", path, err)
	}
	var reg modelRegistry
	if err := json.Unmarshal(data, &reg); err != nil {
		return nil, fmt.Errorf("migrations: parse model registry %s: %w", path, err)
	}
	for _, model := range reg.Models {
		if err := validateModel(model); err != nil {
			return nil, fmt.Errorf("migrations: model registry %s: %w", path, err)
		}
	}
	return reg.Models, nil
}

// SaveRegistry persists models to migrationDir, sorted for stable diffs
// regardless of the order callers passed them in.
func SaveRegistry(migrationDir string, models []Model) error {
	data, err := json.MarshalIndent(modelRegistry{Models: sortModels(models)}, "", "  ")
	if err != nil {
		return fmt.Errorf("migrations: marshal model registry: %w", err)
	}
	data = append(data, '\n')
	path := RegistryPath(migrationDir)
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("migrations: write model registry %s: %w", path, err)
	}
	return nil
}

// MergeModels unions existing (typically the persisted registry) with
// additional (typically the --model flags for this invocation), removing
// duplicates. additional's relative order is preserved first — it is what
// the caller explicitly asked for this run — followed by any existing-only
// models in sorted order.
func MergeModels(existing, additional []Model) []Model {
	seen := make(map[Model]bool, len(existing)+len(additional))
	merged := make([]Model, 0, len(existing)+len(additional))
	for _, model := range additional {
		if seen[model] {
			continue
		}
		seen[model] = true
		merged = append(merged, model)
	}
	var rest []Model
	for _, model := range existing {
		if seen[model] {
			continue
		}
		seen[model] = true
		rest = append(rest, model)
	}
	return append(merged, sortModels(rest)...)
}

// SubtractModels returns models with every entry in remove removed.
func SubtractModels(models, remove []Model) []Model {
	if len(remove) == 0 {
		return models
	}
	drop := make(map[Model]bool, len(remove))
	for _, model := range remove {
		drop[model] = true
	}
	out := make([]Model, 0, len(models))
	for _, model := range models {
		if drop[model] {
			continue
		}
		out = append(out, model)
	}
	return out
}

func sortModels(models []Model) []Model {
	out := append([]Model(nil), models...)
	sort.Slice(out, func(i, j int) bool {
		if out[i].ImportPath == out[j].ImportPath {
			return out[i].TypeName < out[j].TypeName
		}
		return out[i].ImportPath < out[j].ImportPath
	})
	return out
}
