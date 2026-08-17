// Package resourcegen implements `gombit make resource`.
//
// It writes a feature-package (model, thin Huma handler, routes; optional
// service.go / repo.go), vanilla TypeScript list/form pages that import
// generated OpenAPI client types, and registers the package from
// cmd/server/main.go via go/ast. Generators are idempotent and additive;
// Go source is never patched with regular expressions.
package resourcegen
