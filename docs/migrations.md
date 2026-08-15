# Migrations

M2-1 introduces `gombit db makemigrations`, a thin wrapper around Atlas
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

## Drivers

`--driver` accepts the supported v0.1 database drivers:

| Driver | Atlas dev database |
| --- | --- |
| `sqlite` | `sqlite://file?mode=memory&_fk=1` |
| `postgres` | `docker://postgres/15/dev?search_path=public` |
| `mysql` | `docker://mysql/8/dev` |

SQLite runs without Docker. PostgreSQL and MySQL use Atlas dev-database Docker
URLs, so Docker must be available when generating those migrations.

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
