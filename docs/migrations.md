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

Repeat `--model` for every GORM model that should be part of the desired
schema:

```sh
gombit db makemigrations create_accounts \
  --driver postgres \
  --model github.com/acme/shop/internal/account.Account \
  --model github.com/acme/shop/internal/product.Product
```

The command writes a temporary Atlas Program Mode loader under `.gombit`,
passes all supplied model types to `gormschema.New(driver).Load(...)`, and
then runs:

```sh
atlas migrate diff <name> --env gombit --config file://<generated atlas.hcl>
```

The temporary loader is removed after Atlas exits. Migration files are written
to `database/migrations` by default; override that with `--dir`.

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

This explicit list is the Program Mode equivalent of importing each feature
package and passing concrete model values to Atlas. It avoids runtime
reflection discovery and keeps the loader reviewable.
