package contract

import "github.com/danielgtaylor/huma/v2"

const (
	// OpenAPIPath is the Huma OpenAPI prefix. Clients fetch /openapi.json.
	OpenAPIPath = "/openapi"
	// DocsPath is the FastAPI-style interactive docs UI.
	DocsPath = "/docs"
)

// HumaConfig returns the default Huma configuration for a Gombit app.
//
// Interactive docs are enabled at /docs (Swagger UI). The schema browser stays
// disabled. Pass docsEnabled=false to omit the docs route (production default).
func HumaConfig(title, version string) huma.Config {
	return HumaConfigFor(title, version, true)
}

// HumaConfigFor returns Huma configuration with docs explicitly enabled or not.
func HumaConfigFor(title, version string, docsEnabled bool) huma.Config {
	if title == "" {
		title = "Gombit"
	}
	if version == "" {
		version = "0.0.0"
	}
	config := huma.DefaultConfig(title, version)
	config.OpenAPIPath = OpenAPIPath
	config.SchemasPath = ""
	if docsEnabled {
		config.DocsPath = DocsPath
		config.DocsRenderer = huma.DocsRendererSwaggerUI
		return config
	}
	config.DocsPath = ""
	return config
}
