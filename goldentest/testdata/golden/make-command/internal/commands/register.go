package commands

import "github.com/LAA-Software-Engineering/gombit/cli"

// Register attaches this feature-package's management commands to the
// app-owned gombit Cobra tree via cli.AddCommand (D13 / ADR-014).
// Called explicitly from cmd/gombit; Gombit does not discover commands
// by reflection.
func Register(root *cli.Command) {
	cli.AddCommand(root, NewGreetCommand())
}
