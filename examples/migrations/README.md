# Migration Example

Run these commands from the repository root. Install Atlas Community Edition
first (`curl -sSf https://atlasgo.sh | sh -s -- --community`).

## Generate

```sh
gombit db makemigrations create_products \
  --driver sqlite \
  --model github.com/LAA-Software-Engineering/gombit/examples/migrations/internal/product.Product
```

This demonstrates the feature-package model path used by the temporary Atlas
Program Mode loader. The generated migration files are written to
`database/migrations` unless `--dir` is provided.

## Apply, status, and rollback

After generating an up migration, add a companion down file with the same
version/name prefix, for example:

```text
database/migrations/20260101000000_create_products.sql
database/migrations/downs/20260101000000_create_products.down.sql
```

Configure the database (defaults are SQLite) and run:

```sh
export GOMBIT_DATABASE_DRIVER=sqlite
export GOMBIT_DATABASE_DSN='file:gombit.db?cache=shared&_fk=1'

gombit db status
gombit db migrate
gombit db status
gombit db rollback
```

`migrate` wraps `atlas migrate apply` and records the applied versions in
`framework_migrations` as one batch. `rollback` executes the latest batch's
`downs/*.down.sql` files and clears both Gombit and Atlas revision rows for those
versions. Seed/reset commands are tracked separately in M2-3.
