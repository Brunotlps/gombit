// Package cli is the Cobra command tree for `gombit` (D13 / ADR-014).
//
// The framework binary (`cmd/gombit`) and generated application binaries
// (`cmd/gombit` inside a `gombit new` app) share NewRoot. Apps attach
// feature-package management commands with AddCommand — the only supported
// registration path. There is no second command router and no reflection
// discovery of commands.
package cli
