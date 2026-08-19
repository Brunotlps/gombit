package scaffold

import (
	"context"
	"fmt"
	"os/exec"

	"github.com/LAA-Software-Engineering/gombit/config"
	"github.com/LAA-Software-Engineering/gombit/migrations"
)

// bootstrapModels lists the GORM models internal/platform/database.go.tmpl
// registers for AutoMigrate: the framework's own auth tables plus the
// product/ example, in the same order the template writes them.
//
// AutoMigrate runs at every app startup (app.OnStart), creating these tables
// directly through GORM. Without a migration on disk for them from the
// start, Atlas's tracked history never learns they exist, so the first later
// diff that names a model not yet in the registry "discovers" them as
// missing too and tries to CREATE tables that are already there — and
// applying that migration fails with "table already exists" (#96). Seeding
// this migration, and the model registry it writes (see #97's
// migrations.MakeMigrations / database/migrations/models.json), at scaffold
// time keeps Atlas's history in sync with what AutoMigrate creates from the
// very first run: a later `gombit db makemigrations create_x --model X`
// (chapter 3's own pattern) merges with this registry instead of treating
// its single model as the entire desired schema.
func bootstrapModels(module string) []migrations.Model {
	const authImport = "github.com/LAA-Software-Engineering/gombit/auth"
	return []migrations.Model{
		{ImportPath: authImport, TypeName: "User"},
		{ImportPath: authImport, TypeName: "RefreshToken"},
		{ImportPath: authImport, TypeName: "Group"},
		{ImportPath: authImport, TypeName: "Permission"},
		{ImportPath: module + "/internal/product", TypeName: "Product"},
	}
}

// seedBootstrapMigration writes the initial database/migrations/ entry
// covering bootstrapModels. It requires a module that already resolves
// (go mod tidy succeeded) and the atlas binary. Any failure to seed it —
// atlas missing, or atlas present but unable to run (e.g. --database
// postgres/mysql need a running Docker daemon for Atlas's dev-database, and
// gombit new never required Docker before) — prints the equivalent
// gombit db makemigrations command instead of failing gombit new outright.
// Bootstrapping the migration is an automation on top of an otherwise
// complete, working scaffold, not a precondition for one — the same
// reasoning behind resourcegen's own atlas-missing fallback, extended here
// to cover atlas errors too since this runs unprompted, not because the
// user explicitly asked to generate a migration.
func seedBootstrapMigration(ctx context.Context, opts Options) error {
	models := bootstrapModels(opts.Module)

	atlasPath, lookErr := scaffoldLookPath("atlas")
	if lookErr != nil || atlasPath == "" {
		return printBootstrapMigrationHint(opts, models, "atlas not on PATH")
	}

	driver := config.DatabaseDriver(opts.Database)
	err := scaffoldMakeMigrations(ctx, migrations.Options{
		WorkDir:      opts.Dest,
		Name:         "bootstrap",
		Driver:       driver,
		MigrationDir: "database/migrations",
		Models:       models,
		Stdout:       opts.Stdout,
		Stderr:       opts.Stderr,
	})
	if err != nil {
		return printBootstrapMigrationHint(opts, models, fmt.Sprintf("could not seed it (%v)", err))
	}
	return nil
}

func printBootstrapMigrationHint(opts Options, models []migrations.Model, reason string) error {
	var args string
	for _, model := range models {
		args += " --model " + model.ImportPath + "." + model.TypeName
	}
	_, err := fmt.Fprintf(
		opts.Stdout,
		"note: %s; run this once before gombit db migrate so Atlas's history matches "+
			"what AutoMigrate creates at startup:\n  gombit db makemigrations bootstrap%s\n",
		reason, args,
	)
	return err
}

// scaffoldLookPath and scaffoldMakeMigrations are indirected for tests, the
// same pattern resourcegen uses for its own atlas/makemigrations seams.
var scaffoldLookPath = func(name string) (string, error) {
	return exec.LookPath(name)
}

var scaffoldMakeMigrations = migrations.MakeMigrations
