// Package goldentest is the M4-5 generator golden suite.
//
// For each generator it runs a fixed, non-interactive fixture, diffs the
// output tree against testdata/golden, compiles the backend (with a local
// replace in a temp copy, never in committed goldens), typechecks the
// frontend when Node is present, and checks idempotency.
//
// Refresh committed trees after intentional generator changes:
//
//	go test ./goldentest -update
package goldentest
