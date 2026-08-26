# nestjs

The BENCH-1 NestJS + TypeORM ecosystem-context app (issue
[#141](https://github.com/gombit-dev/gombit/issues/141) Phase 4): the same
canonical `/api/projects` CRUD API ([../../docs/schema.md](../../docs/schema.md))
as `../gin-gorm`, `../gombit`, `../django`, `../rails`, and `../laravel`,
implemented idiomatically in NestJS — non-Go ecosystem context, not another
framework-tax control.

## Versions (issue #141 §16/§17: pin exact versions)

| Package | Version |
| --- | --- |
| Node.js | 24 |
| NestJS (`@nestjs/core`) | 11.2.3 |
| `@nestjs/typeorm` | 11.0.3 |
| TypeORM | 0.3.31 |
| pg | 8.23.0 |
| TypeScript | 5.9.3 |

`package.json` pins every direct dependency to an exact version;
`package-lock.json` (committed) pins the full transitive tree, so `npm ci`
reproduces it exactly. `node_modules/` and `dist/` are not committed.

**TypeORM 0.3.31, not the 1.x major**: TypeORM 1.0 shipped in mid-2026, but
`@nestjs/typeorm@11.0.3`'s integration is built around the mature 0.3.x line
(its peer range only tentatively lists `^1.0.0-dev`). Issue #141 allows "a
conventional Nest ORM"; the canonical, battle-tested NestJS+TypeORM stack is
0.3.x, and 0.3.31 is the latest of that line — a deliberate choice for a
reproducible benchmark over the very new 1.x boundary.

Node is the host toolchain here (no Docker needed, unlike `../rails`/
`../laravel`).

## Run

```sh
docker compose -f benchmarks/compose.yml up -d postgres
createdb -h 127.0.0.1 -p 55432 -U gombit gombit_bench_nestjs   # once

cd benchmarks/apps/nestjs
npm ci
DATABASE_URL="postgres://gombit:gombit@127.0.0.1:55432/gombit_bench_nestjs?sslmode=disable" \
  npm run migration:run

# seed (1,000 users, 100,000 projects) — idempotent, truncates first
DATABASE_URL="postgres://gombit:gombit@127.0.0.1:55432/gombit_bench_nestjs?sslmode=disable" \
  npm run seed

# production (issue #141 §17): compile, then run the compiled output — never
# a ts-node / watch dev server
npm run build
NODE_ENV=production PORT=8085 \
  DATABASE_URL="postgres://gombit:gombit@127.0.0.1:55432/gombit_bench_nestjs?sslmode=disable" \
  npm run start:prod
```

Env vars:

| Var | Default | |
| --- | --- | --- |
| `DATABASE_URL` | `postgres://gombit:gombit@127.0.0.1:55432/gombit_bench_nestjs?sslmode=disable` | `src/data-source.ts` reads it |
| `PORT` | `8085` | |
| `NODE_ENV` | (set to `production` for a benchmark run) | issue #141 §17 |
| `POOL_MAX_OPEN` | `20` | issue #141 §18; see below |

### Production configuration (issue #141 §17)

```text
NODE_ENV=production
compiled output: npm run build (tsc -> dist/), then node dist/main
```

The app is a single Node process (one event loop) — the same booted-once,
persistent-process model as `../gin-gorm`/`../gombit` (a Go binary) and
`../rails`/`../django` (a long-lived worker), and unlike `../laravel`'s
per-request PHP-FPM re-bootstrap. No cluster/PM2 fork is pinned, so the whole
server is one process.

### Database connection pooling (issue #141 §18)

A single Node process means one global connection pool for the whole server —
the same single-pool topology `../gin-gorm`/`../gombit`'s single Go binary
has, with no per-worker division (unlike `../laravel`'s FPM or `../django`'s
gunicorn). `src/data-source.ts` pins `extra.max = POOL_MAX_OPEN` (default 20),
matching the ceiling every implementation uses.

### Request logging (issue #141 §19)

TypeORM query logging is off (`logging: false` in `src/data-source.ts`), and
NestFactory's default logger prints startup/error lines but not a line per
request. No per-request access logging is enabled.

## Test

```sh
docker compose -f benchmarks/compose.yml up -d postgres
createdb -h 127.0.0.1 -p 55432 -U gombit gombit_bench_nestjs_test   # once
cd benchmarks/apps/nestjs
npm ci
DATABASE_URL="postgres://gombit:gombit@127.0.0.1:55432/gombit_bench_nestjs_test?sslmode=disable" \
  npm test
```

Tests run against real PostgreSQL (`ts-jest`, `--runInBand`); the helper runs
the migrations on the connected database and each test truncates first. The
scaffold's sqlite/`:memory:` was not used — the canonical schema is
Postgres-specific (TEXT, TIMESTAMPTZ, a real FK).

Covers the same contract the five sibling suites pin: full
create/get/update/delete/404/validation-failure round trip; blank-name
rejection on create and update; the present-null `description` contract
(`422`, matching rails/django/laravel — the DTO's `@ValidateIf(present)
@IsString` rejects an explicit null while an omitted key defaults to `""`);
whitespace and empty-string preservation (NestJS does not trim request
strings, so no middleware to disable unlike Laravel); zero/nonexistent
`owner_id` → `422` leaning on the FK; malformed JSON → `422`; 404 for a
nonexistent and a non-numeric id; list pagination/ordering; a query-count
guard (**3** for a non-empty page, **2** for empty — matching `../gin-gorm`'s
pinned shape via TypeORM's `relationLoadStrategy: 'query'`, a batched owner
`IN (...)` rather than a JOIN); the seed content formulas (ported by hand);
the seed contract's DB-backed idempotency; and the schema contract itself
(see below). Every rejection test asserts the D10 `error.code`.

## Schema and precision notes

Verified against real PostgreSQL (`psql \d`, and `schema-contract.e2e-spec.ts`
in CI):

- `text` columns (not varchar); a plain, non-deferrable FK; `timestamptz(6)`
  with a DB-side `now()` default — the migration is written as explicit SQL
  (`src/migrations/`) rather than generated from entities so the exact DDL is
  controlled.
- **Microsecond timestamp fidelity is a real NestJS-specific risk on the
  *read* path**: the `pg` driver parses `timestamptz` into a JS `Date`, which
  holds only milliseconds, silently dropping the microseconds a
  `timestamptz(6)` column stores. `src/data-source.ts` overrides the `pg`
  type parsers for OID 1114/1184 to return the raw string, and the serializer
  reshapes it to the canonical `...Z` ISO form — preserving full microseconds.
  Pinned by `schema-contract.e2e-spec.ts`'s read-path test, which round-trips
  a known `.123456` through the API and fails if the override is removed (the
  column-precision and FK assertions still pass — the DDL is unchanged, so
  they can't see this). *Writes* carry microseconds inherently: created_at is
  the DB `now()` default and updated_at is set to the SQL `now()` on update
  (`src/project/project.service.ts`), never a JS `Date`.
- `id`/`owner_id` are `bigint`, which TypeORM surfaces as strings; the
  serializer converts them to numbers for the response (ids fit in a JS
  number).

## Status

Schema, seed, CRUD app, its own test suite, and CI (a Node 24 + `npm test`
step in `.github/workflows/ci.yml`'s `database-postgres` job) are done
(tracked in
[docs/plans/BENCH-1-benchmark-suite.md](../../../docs/plans/BENCH-1-benchmark-suite.md)
Phase 4). Still open, matching the Phase 3/4 precedent: a
`benchmarks/compose.yml` app service, and extending the Go
`benchmarks/apps/fairness_test.go` cross-implementation check to include this
app.
