package product

import "github.com/gombit-dev/gombit/cli"

// RegisterCommands attaches this feature-package's management commands to the
// app-owned gombit Cobra tree. Called explicitly from cmd/gombit; Gombit does
// not discover commands by reflection. Use cli.AddCommand (D13 / ADR-014).
func RegisterCommands(root *cli.Command) {
}
