package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/LAA-Software-Engineering/gombit/config"
	"github.com/LAA-Software-Engineering/gombit/migrations"
)

var loadConfig = config.Load

func main() {
	if err := run(context.Background(), os.Args[1:], os.Stdout, os.Stderr); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(ctx context.Context, args []string, stdout io.Writer, stderr io.Writer) error {
	if len(args) == 0 {
		usage(stderr)
		return errors.New("gombit: command is required")
	}
	if args[0] != "db" {
		usage(stderr)
		return fmt.Errorf("gombit: unknown command %q", args[0])
	}
	if len(args) < 2 {
		dbUsage(stderr)
		return errors.New("gombit db: subcommand is required")
	}
	switch args[1] {
	case "makemigrations":
		return runMakeMigrations(ctx, args[2:], stdout, stderr)
	case "migrate":
		return runMigrate(ctx, args[2:], stdout, stderr)
	case "rollback":
		return runRollback(ctx, args[2:], stdout, stderr)
	case "status":
		return runStatus(ctx, args[2:], stdout, stderr)
	default:
		dbUsage(stderr)
		return fmt.Errorf("gombit db: unknown subcommand %q", args[1])
	}
}

func runMakeMigrations(ctx context.Context, args []string, stdout io.Writer, stderr io.Writer) error {
	if len(args) == 0 {
		return errors.New("gombit db makemigrations: migration name is required")
	}
	name := args[0]
	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	flags := flag.NewFlagSet("gombit db makemigrations", flag.ContinueOnError)
	flags.SetOutput(stderr)
	driver := flags.String("driver", string(cfg.Database.Driver), "database driver: sqlite, postgres, or mysql")
	migrationDir := flags.String("dir", "database/migrations", "migration directory")
	atlasBin := flags.String("atlas-bin", "atlas", "Atlas CLI binary path")
	var modelValues modelFlags
	flags.Var(&modelValues, "model", "GORM model import path and type, e.g. github.com/acme/app/internal/product.Product; repeat for multiple models")
	if err := flags.Parse(args[1:]); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		flags.Usage()
		return fmt.Errorf("gombit db makemigrations: unexpected argument %q", flags.Arg(0))
	}

	models := make([]migrations.Model, 0, len(modelValues))
	for _, value := range modelValues {
		model, err := migrations.ParseModel(value)
		if err != nil {
			return err
		}
		models = append(models, model)
	}

	return migrations.MakeMigrations(ctx, migrations.Options{
		WorkDir:      ".",
		Name:         name,
		Driver:       config.DatabaseDriver(*driver),
		MigrationDir: *migrationDir,
		AtlasBinary:  *atlasBin,
		Models:       models,
		Stdout:       stdout,
		Stderr:       stderr,
	})
}

func runMigrate(ctx context.Context, args []string, stdout io.Writer, stderr io.Writer) error {
	opts, err := parseApplyFlags("gombit db migrate", args, stderr)
	if err != nil {
		return err
	}
	return migrations.Migrate(ctx, opts.withWriters(stdout, stderr))
}

func runRollback(ctx context.Context, args []string, stdout io.Writer, stderr io.Writer) error {
	opts, err := parseApplyFlags("gombit db rollback", args, stderr)
	if err != nil {
		return err
	}
	return migrations.Rollback(ctx, opts.withWriters(stdout, stderr))
}

func runStatus(ctx context.Context, args []string, stdout io.Writer, stderr io.Writer) error {
	opts, err := parseApplyFlags("gombit db status", args, stderr)
	if err != nil {
		return err
	}
	return migrations.Status(ctx, opts.withWriters(stdout, stderr))
}

type applyFlagValues struct {
	opts migrations.ApplyOptions
}

func (v applyFlagValues) withWriters(stdout io.Writer, stderr io.Writer) migrations.ApplyOptions {
	v.opts.Stdout = stdout
	v.opts.Stderr = stderr
	return v.opts
}

func parseApplyFlags(name string, args []string, stderr io.Writer) (applyFlagValues, error) {
	cfg, err := loadConfig()
	if err != nil {
		return applyFlagValues{}, err
	}

	flags := flag.NewFlagSet(name, flag.ContinueOnError)
	flags.SetOutput(stderr)
	migrationDir := flags.String("dir", "database/migrations", "migration directory")
	atlasBin := flags.String("atlas-bin", "atlas", "Atlas CLI binary path")
	if err := flags.Parse(args); err != nil {
		return applyFlagValues{}, err
	}
	if flags.NArg() != 0 {
		flags.Usage()
		return applyFlagValues{}, fmt.Errorf("%s: unexpected argument %q", name, flags.Arg(0))
	}

	return applyFlagValues{opts: migrations.ApplyOptions{
		WorkDir:      ".",
		MigrationDir: *migrationDir,
		AtlasBinary:  *atlasBin,
		Database:     cfg.Database,
	}}, nil
}

type modelFlags []string

func (m *modelFlags) String() string {
	return fmt.Sprint([]string(*m))
}

func (m *modelFlags) Set(value string) error {
	*m = append(*m, value)
	return nil
}

func usage(w io.Writer) {
	_, _ = fmt.Fprintln(w, "usage: gombit db <subcommand>")
	dbUsage(w)
}

func dbUsage(w io.Writer) {
	_, _ = fmt.Fprintln(w, "available db subcommands:")
	_, _ = fmt.Fprintln(w, "  makemigrations <name> --model <import.Type> [--driver sqlite|postgres|mysql]")
	_, _ = fmt.Fprintln(w, "  migrate [--dir database/migrations] [--atlas-bin atlas]")
	_, _ = fmt.Fprintln(w, "  rollback [--dir database/migrations]")
	_, _ = fmt.Fprintln(w, "  status [--dir database/migrations] [--atlas-bin atlas]")
}
