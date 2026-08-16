package contract

import "github.com/danielgtaylor/huma/v2"

// HumaConfig returns the default Huma configuration for a Gombit app.
//
// Docs and schema browser routes stay disabled; OpenAPI JSON remains available
// for local inspection until M3-3 owns the generate CLI.
func HumaConfig(title, version string) huma.Config {
	if title == "" {
		title = "Gombit"
	}
	if version == "" {
		version = "0.0.0"
	}
	config := huma.DefaultConfig(title, version)
	config.DocsPath = ""
	config.SchemasPath = ""
	return config
}
