package contract

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/danielgtaylor/huma/v2"
)

// OpenAPIJSON marshals the Huma-generated OpenAPI 3.1 document.
func OpenAPIJSON(api huma.API) ([]byte, error) {
	if api == nil {
		return nil, fmt.Errorf("contract: nil API")
	}
	spec, err := json.MarshalIndent(api.OpenAPI(), "", "  ")
	if err != nil {
		return nil, fmt.Errorf("contract: marshal OpenAPI: %w", err)
	}
	return spec, nil
}

// WriteOpenAPI writes the Huma-generated OpenAPI document to path.
func WriteOpenAPI(path string, api huma.API) error {
	spec, err := OpenAPIJSON(api)
	if err != nil {
		return err
	}
	return WriteOpenAPIFile(path, spec)
}

// WriteOpenAPIFile writes a pre-marshaled OpenAPI document to path.
func WriteOpenAPIFile(path string, spec []byte) error {
	if path == "" {
		return fmt.Errorf("contract: OpenAPI output path is required")
	}
	if len(spec) == 0 {
		return fmt.Errorf("contract: OpenAPI document is empty")
	}
	if spec[len(spec)-1] != '\n' {
		copied := make([]byte, len(spec)+1)
		copy(copied, spec)
		copied[len(spec)] = '\n'
		spec = copied
	}
	dir := filepath.Dir(path)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o750); err != nil {
			return fmt.Errorf("contract: create OpenAPI directory: %w", err)
		}
	}
	//nolint:gosec // openapi.json is a non-secret artifact meant to be broadly readable
	if err := os.WriteFile(path, spec, 0o644); err != nil {
		return fmt.Errorf("contract: write OpenAPI: %w", err)
	}
	return nil
}
