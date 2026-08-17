package main

import (
	"fmt"
	"io"

	"github.com/LAA-Software-Engineering/gombit/scaffold"
	"github.com/spf13/cobra"
)

func newNewCommand(stdout io.Writer, stderr io.Writer) *cobra.Command {
	var (
		database string
		cache    string
		auth     string
		ui       string
		module   string
		dryRun   bool
		force    bool
	)
	cmd := silence(&cobra.Command{
		Use:   "new [name]",
		Short: "Scaffold a new Gombit application",
		Long: `Scaffold a new Gombit application using the feature-package layout.

Non-interactive example:

  gombit new demo --database sqlite

Defaults: --database sqlite, --cache memory, --auth jwt, --ui minimal.
The Go module path defaults to github.com/example/<name> (--module).
--auth cookie and --ui mui are recorded in gombit.yaml only (M5 implements them).
--dry-run prints the file list without writing. A non-empty destination
requires --force.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts := scaffold.Options{
				Database: database,
				Cache:    cache,
				Auth:     auth,
				UI:       ui,
				Module:   module,
				DryRun:   dryRun,
				Force:    force,
				WorkDir:  ".",
				Stdin:    cmd.InOrStdin(),
				Stdout:   stdout,
				Stderr:   stderr,
			}
			if len(args) == 1 {
				opts.Name = args[0]
			}
			if err := scaffold.Generate(cmd.Context(), opts); err != nil {
				return fmt.Errorf("gombit new: %w", err)
			}
			return nil
		},
	})
	cmd.Flags().StringVar(&database, "database", scaffold.DefaultDatabase, "database driver: sqlite, postgres, or mysql")
	cmd.Flags().StringVar(&cache, "cache", scaffold.DefaultCache, "cache driver: memory, redis, or noop")
	cmd.Flags().StringVar(&auth, "auth", scaffold.DefaultAuth, "auth mode: jwt, cookie, or none (recorded in gombit.yaml)")
	cmd.Flags().StringVar(&ui, "ui", scaffold.DefaultUI, "UI preset: minimal or mui (recorded in gombit.yaml)")
	cmd.Flags().StringVar(&module, "module", "", "Go module path (default github.com/example/<name>)")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "print files that would be written without writing")
	cmd.Flags().BoolVar(&force, "force", false, "overwrite files in a non-empty destination")
	return cmd
}
