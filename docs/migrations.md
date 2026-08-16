# Migrations

M2 introduces `gombit db` migration commands as a thin wrapper around Atlas
versioned migrations and `ariga.io/atlas-provider-gorm` Program Mode. Gombit
does not define its own migration DSL.

## Generate A Migration

Run the command from the application module root:

```sh
gombit db makemigrations create_products \
  --driver sqlite \
  --model github.com/acme/shop/internal/product.Product
```

The Atlas Community Edition CLI must be installed and available on `PATH`, or
supplied with `--atlas-bin`:

```sh
curl -sSf https://atlasgo.sh | sh -s -- --community
```

Repeat `--model` for every GORM model that should be part of the desired
schema:

```sh
gombit db makemigrations create_accounts \
  --driver postgres \
  --model github.com/acme/shop/internal/account.Account \
  --model github.com/acme/shop/internal/product.Product
```

The command writes a temporary Atlas Program Mode loader under `.gombit`,
passes all supplied model types to `gormschema.New(driver).Load(...)`, writes
the generated SQL schema to a temporary `schema.sql`, and then runs:

```sh
atlas migrate diff <name> --env gombit --config file://<generated atlas.hcl>
```

The temporary loader is removed after Atlas exits. Migration files are written
to `database/migrations` by default; override that with `--dir`.

`gombit db makemigrations` depends only on Atlas Community Edition features:
the generated config points `src` at the temporary schema file and uses
`atlas migrate diff`. It does not depend on Atlas Cloud, drift monitoring,
external schema data sources, or migration linting.

Gombit does not add a separate `--dry-run` flag in M2-1. Atlas owns the diff
preview behavior: if there is no model/schema change, `atlas migrate diff`
exits without writing a new migration.

Migration names may contain letters, numbers, underscores, and hyphens, and
must not start with a hyphen.

## Apply / Status / Rollback

M2-2 adds apply, status, and rollback. These commands read the configured
database from `config.Load()` (`GOMBIT_DATABASE_DRIVER` / `GOMBIT_DATABASE_DSN`)
and the migration directory (default `database/migrations`).

```sh
gombit db migrate [--dir database/migrations] [--atlas-bin atlas]
gombit db status  [--dir database/migrations] [--atlas-bin atlas]
gombit db rollback [--dir database/migrations]
```

### Apply

`gombit db migrate`:

1. Ensures the Gombit revision table `framework_migrations` exists.
2. Runs Atlas Community Edition
   `atlas migrate apply --url ... --dir file://... --allow-dirty`.
   `--allow-dirty` is required because Gombit creates `framework_migrations`
   before apply, and real apps already have schema tables.
3. Records into `framework_migrations` only the pending versions that appear in
   `atlas_schema_revisions` after apply (`version`, `name`, `batch`,
   `applied_at`; no checksum; D4). This keeps the Gombit ledger aligned with
   Atlas when the two previously diverged.

If nothing is pending relative to `framework_migrations`, migrate prints
`No pending migrations.` and does not invoke Atlas apply.

Unrecognized `*.sql` filenames in the migration directory are skipped with a
warning on stderr.

### Status

`gombit db status` prints Gombit applied/pending rows from the migration
directory plus `framework_migrations`, then runs `atlas migrate status` for
Atlas bookkeeping.

### Rollback

Rollback is Gombit-owned and does **not** wrap `atlas migrate down` (that
command is outside Atlas Community Edition).

`gombit db rollback` rolls back the **latest batch** only:

1. Loads the highest `batch` from `framework_migrations`.
2. Requires a companion down file for every version in that batch (missing
   downs fail before any SQL runs; migrate does not require downs).
3. Executes those down files in reverse version order.
4. Deletes matching rows from `framework_migrations` and
   `atlas_schema_revisions` so a later `gombit db migrate` can re-apply.

On SQLite and PostgreSQL, downs and revision deletes run in one transaction:
a mid-batch failure aborts the transaction and leaves revision rows unchanged.
MySQL DDL often auto-commits, so a mid-batch failure can leave the schema
partially rolled back while revision rows remain; the error lists completed
downs and revision rows are only removed after every down succeeds.

### Down files

Atlas writes up migrations such as:

```text
database/migrations/20260101000000_create_products.sql
```

Gombit-owned down SQL lives in a subdirectory so Atlas never scans it (Atlas
panics if `.down.sql` files sit beside versioned up migrations):

```text
database/migrations/downs/20260101000000_create_products.down.sql
```

If any down file in the latest batch is missing, rollback fails before
executing any down SQL. Migrate does not require downs.

## Revision metadata

| Column | Notes |
| --- | --- |
| `version` | Atlas migration version prefix |
| `name` | Migration name suffix |
| `batch` | Incremented once per successful migrate that applies files |
| `applied_at` | UTC timestamp when the batch was recorded |

Atlas may still maintain `atlas.sum` and `atlas_schema_revisions` for apply
integrity. That is separate from D4: Gombit does not store checksums in
`framework_migrations`.

## Drivers

`--driver` (makemigrations) accepts the supported v0.1 database drivers:

| Driver | Atlas dev database |
| --- | --- |
| `sqlite` | `sqlite://file?mode=memory&_fk=1` |
| `postgres` | `docker://postgres/15/dev?search_path=public` |
| `mysql` | `docker://mysql/8/dev` |

SQLite runs without Docker. PostgreSQL and MySQL use Atlas dev-database Docker
URLs, so Docker must be available when generating those migrations.

Apply/status convert the configured GORM DSN into an Atlas `--url` for the
same three drivers.

## Model Registration

M2-1 keeps model enumeration explicit. Generated apps should pass feature
package models from `internal/<feature>` using repeated `--model` flags until
later generator work adds an application-owned registry.

The model spec format is:

```text
<go import path>.<exported model type>
```

For example:

```text
github.com/acme/shop/internal/product.Product
```

The repository example model can be passed with:

```sh
gombit db makemigrations create_products \
  --driver sqlite \
  --model github.com/LAA-Software-Engineering/gombit/examples/migrations/internal/product.Product
```

This explicit list is the Program Mode equivalent of importing each feature
package and passing concrete model values to Atlas. It avoids runtime
reflection discovery and keeps the loader reviewable.
