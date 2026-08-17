// Package commandgen implements `gombit make command`.
//
// It writes a Cobra management command in the internal/commands feature-package
// and registers it from cmd/gombit via go/ast (never regex). Generators are
// idempotent and additive.
package commandgen
