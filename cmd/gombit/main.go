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
	switch args[0] {
	case "openapi":
		return runOpenAPI(ctx, args[1:], stdout, stderr)
	case "client":
		return runClient(ctx, args[1:], stdout, stderr)
	case "db":
		return runDB(ctx, args[1:], stdout, stderr)
	default:
		usage(stderr)
		return fmt.Errorf("gombit: unknown command %q", args[0])
	}
}

func runDB(ctx context.Context, args []string, stdout io.Writer, stderr io.Writer) error {
	if len(args) == 0 {
		dbUsage(stderr)
		return errors.New("gombit db: subcommand is required")
	}
	switch args[0] {
	case "makemigrations":
		return runMakeMigrations(ctx, args[1:], stdout, stderr)
	case "migrate":
		return runMigrate(ctx, args[1:], stdout, stderr)
	case "rollback":
		return runRollback(ctx, args[1:], stdout, stderr)
	case "status":
		return runStatus(ctx, args[1:], stdout, stderr)
	case "seed":
		return runSeed(ctx, args[1:], stdout, stderr)
	case "reset":
		return runReset(ctx, args[1:], stdout, stderr)
	default:
		dbUsage(stderr)
		return fmt.Errorf("gombit db: unknown subcommand %q", args[0])
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

func runSeed(ctx context.Context, args []string, stdout io.Writer, stderr io.Writer) error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	flags := flag.NewFlagSet("gombit db seed", flag.ContinueOnError)
	flags.SetOutput(stderr)
	seedDir := flags.String("seeds", "database/seeds", "seed directory")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		flags.Usage()
		return fmt.Errorf("gombit db seed: unexpected argument %q", flags.Arg(0))
	}

	return migrations.Seed(ctx, migrations.SeedOptions{
		WorkDir:  ".",
		SeedDir:  *seedDir,
		Database: cfg.Database,
		Stdout:   stdout,
		Stderr:   stderr,
	})
}

func runReset(ctx context.Context, args []string, stdout io.Writer, stderr io.Writer) error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	flags := flag.NewFlagSet("gombit db reset", flag.ContinueOnError)
	flags.SetOutput(stderr)
	migrationDir := flags.String("dir", "database/migrations", "migration directory")
	seedDir := flags.String("seeds", "database/seeds", "seed directory")
	atlasBin := flags.String("atlas-bin", "atlas", "Atlas CLI binary path")
	force := flags.Bool("force", false, "allow reset when GOMBIT_ENV=production")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		flags.Usage()
		return fmt.Errorf("gombit db reset: unexpected argument %q", flags.Arg(0))
	}

	return migrations.Reset(ctx, migrations.ResetOptions{
		ApplyOptions: migrations.ApplyOptions{
			WorkDir:      ".",
			MigrationDir: *migrationDir,
			AtlasBinary:  *atlasBin,
			Database:     cfg.Database,
			Stdout:       stdout,
			Stderr:       stderr,
		},
		SeedDir:     *seedDir,
		Force:       *force,
		Environment: cfg.Environment,
	})
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
	_, _ = fmt.Fprintln(w, "usage: gombit <command>")
	_, _ = fmt.Fprintln(w, "available commands:")
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
