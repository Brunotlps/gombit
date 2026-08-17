package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/LAA-Software-Engineering/gombit/config"
	"github.com/spf13/cobra"
)

var loadConfig = config.Load

func main() {
	if err := run(context.Background(), os.Args[1:], os.Stdout, os.Stderr); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(ctx context.Context, args []string, stdout io.Writer, stderr io.Writer) error {
	cmd := newRootCommand(stdout, stderr)
	cmd.SetArgs(args)
	return cmd.ExecuteContext(ctx)
}

func newRootCommand(stdout io.Writer, stderr io.Writer) *cobra.Command {
	root := &cobra.Command{
		Use:           "gombit",
		Short:         "Gombit is a Django-for-Go full-stack framework",
		Long:          rootLongHelp(),
		SilenceUsage:  true,
		SilenceErrors: true,
		Args:          cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			usage(stderr)
			if len(args) == 0 {
				return errors.New("gombit: command is required")
			}
			return fmt.Errorf("gombit: unknown command %q", args[0])
		},
	}
	root.SetOut(stdout)
	root.SetErr(stderr)
	root.CompletionOptions.DisableDefaultCmd = true
	root.AddCommand(newNewCommand(stdout, stderr))
	root.AddCommand(newDBCommand(stdout, stderr))
	root.AddCommand(newOpenAPICommand(stdout, stderr))
	root.AddCommand(newClientCommand(stdout, stderr))
	return root
}

func rootLongHelp() string {
	return strings.Join([]string{
		"Gombit is a Django-for-Go full-stack framework.",
		"",
		"Command families:",
		"  new       Scaffold a new application",
		"  db        Database migrations (makemigrations, migrate, rollback, status, seed, reset)",
		"  openapi   Write the live OpenAPI 3.1 document",
		"  client    Generate and check the TypeScript client",
	}, "\n")
}

func usage(w io.Writer) {
	_, _ = fmt.Fprintln(w, "usage: gombit <command>")
	_, _ = fmt.Fprintln(w, "available commands:")
	_, _ = fmt.Fprintln(w, "  new [name]        scaffold a new Gombit application")
	_, _ = fmt.Fprintln(w, "  db <subcommand>   see gombit db")
	_, _ = fmt.Fprintln(w, "  openapi generate [--out openapi.json] [--url http://127.0.0.1:8080/openapi.json]")
	_, _ = fmt.Fprintln(w, "  client generate [--spec openapi.json] [--out frontend/src/api/generated] [--dry-run] [--force]")
	_, _ = fmt.Fprintln(w, "  client check [--write] [--spec examples/client/openapi.json] [--out examples/client/frontend/src/api/generated]")
}

func dbUsage(w io.Writer) {
	_, _ = fmt.Fprintln(w, "available db subcommands:")
	_, _ = fmt.Fprintln(w, "  makemigrations <name> --model <import.Type> [--driver sqlite|postgres|mysql]")
	_, _ = fmt.Fprintln(w, "  migrate [--dir database/migrations] [--atlas-bin atlas]")
	_, _ = fmt.Fprintln(w, "  rollback [--dir database/migrations]")
	_, _ = fmt.Fprintln(w, "  status [--dir database/migrations] [--atlas-bin atlas]")
	_, _ = fmt.Fprintln(w, "  seed [--seeds database/seeds]")
	_, _ = fmt.Fprintln(w, "  reset [--dir database/migrations] [--seeds database/seeds] [--atlas-bin atlas] [--force]")
}

func openapiUsage(w io.Writer) {
	_, _ = fmt.Fprintln(w, "available openapi subcommands:")
	_, _ = fmt.Fprintln(w, "  generate [--out openapi.json] [--url http://127.0.0.1:8080/openapi.json]")
}

func clientUsage(w io.Writer) {
	_, _ = fmt.Fprintln(w, "available client subcommands:")
	_, _ = fmt.Fprintln(w, "  generate [--spec openapi.json] [--out frontend/src/api/generated] [--dry-run] [--force] [--npx npx]")
	_, _ = fmt.Fprintln(w, "  check [--write] [--spec examples/client/openapi.json] [--out examples/client/frontend/src/api/generated] [--npx npx]")
}

func silence(cmd *cobra.Command) *cobra.Command {
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true
	return cmd
}
