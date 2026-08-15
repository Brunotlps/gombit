# ADR-012: Migrations Use Atlas GORM Provider

## Status

Accepted.

## Context

Gombit needs Django-style `makemigrations` for Go/GORM models without
inventing a schema diff engine or a migration DSL. The build plan locks the
direction to `ariga.io/atlas-provider-gorm` plus Atlas versioned migrations,
but M2-0 is the go/no-go gate for licensing, driver coverage, Program Mode,
and open-core boundaries.

This ADR was verified against upstream Atlas documentation on 2026-08-15.

Atlas provides two relevant pieces:

- `ariga.io/atlas-provider-gorm`, a Go module that loads GORM models into an
  Atlas external schema.
- The Atlas CLI, which can compare that desired schema to the current
  migration directory state and write versioned SQL migrations with
  `atlas migrate diff`.

Gombit's generated apps use feature packages under `internal/<feature>/`, so
the migration loader must handle GORM models spread across multiple packages.
Atlas's GORM Program Mode supports this shape: the application-owned loader is
a Go program that imports each feature package and passes every model to
`gormschema.New(driver).Load(...)`.

## Decision

Gombit will wrap Atlas and `ariga.io/atlas-provider-gorm` for M2 migrations.

`gombit db makemigrations` will:

1. Resolve the configured database driver.
2. Generate or run an application-owned Atlas loader program that imports the
   registered feature-package models.
3. Use `ariga.io/atlas-provider-gorm/gormschema` in Program Mode to expose the
   GORM schema as an Atlas external schema.
4. Invoke `atlas migrate diff` against Gombit's migration directory to write a
   versioned SQL migration for the selected driver.

The supported v0.1 drivers remain SQLite, PostgreSQL, and MySQL. Although
issue #11 mentions SQLite and PostgreSQL, the build plan is authoritative and
M1-4 has already established MySQL as part of the runtime database boundary.
Atlas documents SQLite, PostgreSQL, and MySQL as open-supported drivers, and
the GORM provider documents support for the same three databases.

The fallback from the build plan, a hand-rolled migration DSL and diff engine,
is rejected. The open-source Atlas path covers the required v0.1 workflow:
database inspection, schema diffing, migration planning, versioned migrations,
and execution for the supported drivers.

## Licensing and Feature Boundary

The provider repository is Apache-2.0 licensed. Atlas describes its core
engine as open source under Apache-2.0 and lists database inspection, schema
diffing, versioned migrations, declarative migrations, PostgreSQL, MySQL, and
SQLite as open features.

M2 must not depend on Atlas Pro-only features unless a later issue explicitly
changes the product decision. The following are outside the v0.1 open-source
dependency surface:

- `atlas migrate lint` migration safety analysis.
- Pre-migration checks and deploy-time drift detection.
- Atlas Registry and Cloud-backed drift monitoring.
- Atlas migration/schema testing framework.
- Advanced schema objects that Atlas marks as Pro for the supported drivers,
  including views, triggers, functions, procedures, row-level security,
  partitions, and similar database-specific extensions.

Gombit may document those as optional future integrations, but they are not
part of M2 acceptance criteria.

## Consequences

- M2-1 should build a thin wrapper around Atlas Program Mode and
  `atlas migrate diff`; it should not introduce a Gombit migration DSL.
- Model discovery remains explicit and generator-owned. Generated apps should
  register model types in a known place so the loader can pass concrete model
  values to the provider without runtime reflection discovery.
- Migration output is SQL owned by the application. Users can still add
  hand-written SQL or HCL migrations in the migration directory as an escape
  hatch.
- Atlas writes and validates an `atlas.sum` file for its versioned migration
  directory. That does not replace Gombit's runtime migration metadata
  decision from D4: applied revisions are still tracked as
  `version, name, batch, applied_at`, with no checksum in Gombit's own
  revision table.
- Safety checks in M2/M4 CI must use open-source mechanisms unless the project
  intentionally accepts an Atlas Pro dependency. In particular, do not make
  `atlas migrate lint`, drift detection, or the Atlas Registry required for
  the default v0.1 workflow.
- The database conformance matrix remains required for SQLite, PostgreSQL, and
  MySQL because Atlas support does not replace runtime verification against
  real drivers.

## References

- Issue #11: `[M2-0] ADR-012: Migrations = Atlas GORM provider`
- Issue #7: `[M1-4] Multi-driver database.Open + capability model`
- `docs/GOMBIT_BUILD_PLAN.md`
- Atlas GORM Program Mode:
  https://atlasgo.io/guides/orms/gorm/program
- Atlas feature compatibility:
  https://atlasgo.io/features
- Atlas community edition:
  https://atlasgo.io/community-edition
- GORM provider repository:
  https://github.com/ariga/atlas-provider-gorm
- GORM provider package:
  https://pkg.go.dev/ariga.io/atlas-provider-gorm
