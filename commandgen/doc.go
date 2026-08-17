// Package commandgen implements `gombit make command`.
//
// It writes a Cobra management command in a feature-package (default
// internal/commands) as RegisterCommands in commands.go, matching the
// product template, and registers it from cmd/gombit via go/ast (never
// regex). Generators are idempotent and additive.
package commandgen
