package main

import (
	"fmt"
	"io"

	"github.com/LAA-Software-Engineering/gombit/resourcegen"
	"github.com/spf13/cobra"
)

func newMakeCommand(stdout io.Writer, stderr io.Writer) *cobra.Command {
	cmd := silence(&cobra.Command{
		Use:   "make",
		Short: "Generate application code",
		Args:  cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			makeUsage(stderr)
			if len(args) == 0 {
				return fmt.Errorf("gombit make: subcommand is required")
			}
			return fmt.Errorf("gombit make: unknown subcommand %q", args[0])
		},
	})
	cmd.AddCommand(newMakeResourceCommand(stdout, stderr))
	return cmd
}

func newMakeResourceCommand(stdout io.Writer, stderr io.Writer) *cobra.Command {
	var (
		service bool
		repo    bool
		dryRun  bool
		force   bool
	)
	cmd := silence(&cobra.Command{
		Use:   "resource <Name> [field:type[:modifiers]...]",
		Short: "Generate a feature-package resource (AST-safe)",
		Long: `Generate a feature-package under internal/<name>/ with a GORM model,
thin Huma handler (list/get/create), routes, and vanilla TypeScript
list/table + create form pages.

Route registration is appended in cmd/server/main.go via go/ast (never
regex), next to product.Register(app). AutoMigrate is updated the same way.

Field grammar (design §27 subset):

  name:type[:required][,unique][,index]

Supported types: string, text, int, int64, bool, uint.

Examples:

  gombit make resource Widget name:string:required price:int
  gombit make resource Invoice --service --repo --dry-run
  gombit make resource Widget --force

--service / --repo are opt-in pass-through files (C6). Default is a thin
handler over GORM. --dry-run prints the file list without writing.
Re-running refuses to clobber user edits unless --force is set.

Frontend pages import types from frontend/src/api/generated. Run
gombit client generate or gombit dev after the API is up so those types
exist. Atlas SQL is generated when the atlas binary is on PATH, using every
AutoMigrate model in internal/platform (not only the new resource).
Otherwise the GORM model is still loader-ready for gombit db makemigrations.`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			err := resourcegen.Generate(cmd.Context(), resourcegen.Options{
				WorkDir: ".",
				Name:    args[0],
				Fields:  args[1:],
				Service: service,
				Repo:    repo,
				DryRun:  dryRun,
				Force:   force,
				Stdout:  stdout,
				Stderr:  stderr,
			})
			if err != nil {
				return fmt.Errorf("gombit make resource: %w", err)
			}
			return nil
		},
	})
	cmd.Flags().BoolVar(&service, "service", false, "also write a pass-through service.go")
	cmd.Flags().BoolVar(&repo, "repo", false, "also write a pass-through repo.go")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "print files that would be written without writing")
	cmd.Flags().BoolVar(&force, "force", false, "overwrite files that differ from this run")
	return cmd
}

func makeUsage(w io.Writer) {
	_, _ = fmt.Fprintln(w, "available make subcommands:")
	_, _ = fmt.Fprintln(w, "  resource <Name> [field:type[:modifiers]...] [--service] [--repo] [--dry-run] [--force]")
}
