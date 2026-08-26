package metadata

import (
	"encoding/json"
	"io"
)

// WriteJSON encodes the metadata as pretty-printed JSON (results/latest/
// metadata.json).
func WriteJSON(w io.Writer, m Metadata) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(m)
}
