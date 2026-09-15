package metadata

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// WriteJSON encodes the metadata as pretty-printed JSON (results/latest/
// metadata.json).
func WriteJSON(w io.Writer, m Metadata) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(m)
}

// ReadJSON decodes a metadata.json. It is the inverse of WriteJSON, used by the
// producers that merge into an existing snapshot rather than replacing it.
func ReadJSON(r io.Reader) (Metadata, error) {
	var m Metadata
	if err := json.NewDecoder(r).Decode(&m); err != nil {
		return Metadata{}, err
	}
	return m, nil
}

// StampUnitFile records one unit's provenance into the metadata.json at path,
// preserving everything already there.
//
// It is the single read-modify-write used by every producer that writes rows:
// scripts/microbench, scripts/footprint and scripts/run-crud each call it for
// the unit they just measured. Keeping one implementation is what makes the
// "a producer stamps only what it measured" invariant enforceable rather than a
// convention three call sites are trusted to follow.
//
// A missing file is not an error: the first producer to run in a fresh OUT_DIR
// starts the record. A file that exists but does not parse IS an error —
// silently replacing a corrupt snapshot would discard whatever hours-long run
// produced it.
func StampUnitFile(path, group, unit string, prov Provenance) error {
	if !ValidGroup(group) {
		return fmt.Errorf("metadata: unknown group %q", group)
	}
	if unit == "" {
		return fmt.Errorf("metadata: group %q needs a unit to stamp", group)
	}

	existing := Metadata{SchemaVersion: SchemaVersion}
	// path is composed from an operator-supplied output dir, not untrusted
	// input — G304 does not apply.
	f, err := os.Open(path) //nolint:gosec
	switch {
	case err == nil:
		existing, err = ReadJSON(f)
		_ = f.Close()
		if err != nil {
			return fmt.Errorf("metadata: read %s: %w", path, err)
		}
	case !os.IsNotExist(err):
		return fmt.Errorf("metadata: read %s: %w", path, err)
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil { //nolint:gosec // operator-supplied out dir
		return err
	}
	out, err := os.Create(path) //nolint:gosec
	if err != nil {
		return err
	}
	if err := WriteJSON(out, StampUnit(existing, group, unit, prov)); err != nil {
		_ = out.Close()
		return err
	}
	return out.Close()
}

// SiblingPath returns the metadata.json that belongs beside a data file, so a
// producer given `-out .../footprint.json` writes provenance to the snapshot it
// is contributing to without a second path flag to keep in sync.
func SiblingPath(dataFile string) string {
	return filepath.Join(filepath.Dir(dataFile), "metadata.json")
}
