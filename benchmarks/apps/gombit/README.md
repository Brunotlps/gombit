# gombit

The BENCH-1 Gombit-runtime implementation of the canonical `/api/projects`
CRUD API (issue #141, [../../docs/schema.md](../../docs/schema.md)): a
normal Gombit app — Huma handlers (`internal/project`), GORM, `framework.App`
— using Gombit's normal public APIs unmodified, run in production mode.
Compared against [`../gin-gorm`](../gin-gorm), the primary framework-tax
control implementing the same API without Gombit.

## Migrations

Unlike `gin-gorm` (GORM `AutoMigrate`, its own idiomatic mechanism), this app
uses Gombit's real migration path — Atlas, via `gombit db makemigrations`/
`migrate` (AGENTS.md D3: never `AutoMigrate` for a real Gombit app). The
generated migration is committed under `database/migrations/` like any real
Gombit app's would be; regenerate it only if the models change:

```sh
cd benchmarks/apps/gombit
go run github.com/gombit-dev/gombit/cmd/gombit db makemigrations create_projects \
  --driver postgres \
  --model github.com/gombit-dev/gombit/benchmarks/apps/gombit/internal/project.User \
  --model github.com/gombit-dev/gombit/benchmarks/apps/gombit/internal/project.Project
```

## Run

```sh
docker compose -f benchmarks/compose.yml up -d postgres

# create a database separate from gin-gorm's (see benchmarks/apps/gin-gorm/README.md
# for why AutoMigrate and Atlas schemas shouldn't share a database) and migrate — once
createdb -h 127.0.0.1 -p 55432 -U gombit gombit_bench_gombit
cd benchmarks/apps/gombit
GOMBIT_DATABASE_DRIVER=postgres \
GOMBIT_DATABASE_DSN="postgres://gombit:gombit@127.0.0.1:55432/gombit_bench_gombit?sslmode=disable" \
  go run github.com/gombit-dev/gombit/cmd/gombit db migrate

# seed (1,000 users, 100,000 projects) — idempotent, truncates first
go run . -seed

# serve
go run .
```

Env vars (all optional, defaults match `benchmarks/compose.yml`):

| Var | Default | |
| --- | --- | --- |
| `DATABASE_URL` | `postgres://gombit:gombit@127.0.0.1:55432/gombit_bench_gombit?sslmode=disable` | |
| `PORT` | `8080` | |
| `POOL_MAX_OPEN` | `20` | issue #141 "Connection pooling" pins this to 20 across every implementation |
| `POOL_MAX_IDLE` | `20` | |

`cfg.API.Prefix` is set to `/api` in `main.go`, overriding Gombit's own
`/api/v1` default — the canonical route spec and `gin-gorm` both use `/api`
with no version segment; left at the framework default this app's own route
surface wouldn't have matched its own control.

### Under compose (containerized, with the §7 resource budget)

The app is containerized (`Dockerfile`, built from the **repo root**) and wired
into `benchmarks/compose.yml` with the §7 app ceiling (2 vCPU / 1 GiB). Because
it applies **real Atlas migrations** (not AutoMigrate), the image also carries
the `gombit` CLI and the pinned `atlas` binary (`ATLAS_IMAGE` in
`benchmarks/config/versions.env`, the *-alpine* variant so the binary matches
the runtime base), plus a psql client, and its entrypoint has three verbs —
`migrate`, `seed`, `serve`. `serve` does **not** migrate, so run them in order.
The `gombit_bench_gombit` database is created by the `migrate` verb if absent
(idempotent, every bring-up — not by a fresh-volume init script, which wouldn't
fire on this suite's already-initialized Postgres volume):

```sh
docker compose --env-file benchmarks/config/versions.env \
  -f benchmarks/compose.yml up -d postgres

# apply migrations (also creates gombit_bench_gombit if missing), then seed
docker compose --env-file benchmarks/config/versions.env \
  -f benchmarks/compose.yml run --rm gombit migrate
docker compose --env-file benchmarks/config/versions.env \
  -f benchmarks/compose.yml run --rm gombit seed

# now serve, and confirm the §7 ceiling actually landed on the live container
docker compose --env-file benchmarks/config/versions.env \
  -f benchmarks/compose.yml up -d gombit
go run ./benchmarks/scripts/inspect-limits \
  -container "$(docker compose -f benchmarks/compose.yml ps -q gombit)" \
  -cpus 2 -memory 1g
```

`make benchmark-crud-all` brings all six apps up and runs `run-crud` over each;
this containerizes the second of the two Go apps on the same pattern as
`gin-gorm`.

## Test

```sh
# pure unit tests — none of this package's own; the seed formulas it shares
# with gin-gorm are tested in benchmarks/apps/shared

# per-app integration suite, needs a live, already-migrated PostgreSQL
# instance. -p 1 is required, not cosmetic: this expands to two packages
# (this one and internal/project), each its own test binary, and both
# TRUNCATE the same tables on every test's setup. Without -p 1, go test
# runs those binaries in parallel (bounded by GOMAXPROCS) and one's
# TRUNCATE can land mid-assertion in the other.
go test -tags integration -p 1 ./benchmarks/apps/gombit/... \
  -database.dsn "postgres://gombit:gombit@127.0.0.1:55432/gombit_bench_gombit?sslmode=disable"

# cross-implementation fairness check: builds and runs both real binaries,
# needs both databases already seeded (`-seed` on each app) — heavier than
# the per-app suite, so it isn't wired into routine PR CI yet (see
# docs/plans/BENCH-1-benchmark-suite.md Phase 3b); run it locally:
go test -tags integration ./benchmarks/apps/ -run TestCrossImplementationFairness \
  -gin-gorm.dsn "postgres://gombit:gombit@127.0.0.1:55432/gombit_bench?sslmode=disable" \
  -gombit.dsn "postgres://gombit:gombit@127.0.0.1:55432/gombit_bench_gombit?sslmode=disable"
```

CI runs the per-app suite in `.github/workflows/ci.yml`'s `database-postgres`
job, against a third database (`gombit_bench_gombit`) separate from both
`gombit` (auth/database/admin) and `gombit_bench` (gin-gorm) — this app's
`users`/`projects` tables come from a real Atlas migration, not
`AutoMigrate`, so they shouldn't share a database with either.

Covers the same contract `gin-gorm`'s suite does — full CRUD round trip,
blank-name rejection on both create and update, pagination/ordering, and the
3-query / 2-query (empty page) N+1 guard — plus one difference documented
below.

## A discovered framework gap, not a benchmark bug

`POST /api/projects` with a nonexistent `owner_id` returns **500 internal**,
not a 4xx client error. This app uses Gombit's normal
`database.MapPersistError` (`github.com/gombit-dev/gombit/database`)
unmodified, and that function only special-cases unique-constraint
violations — a foreign-key violation falls through to `internal`.
`gin-gorm`'s control implementation doesn't use that framework helper (see
its README) and correctly maps the same input to 422.

This is deliberately **not patched here**: issue #141 requires using
Gombit's normal public APIs as-is ("do not bypass ... normal Gombit response
handling"), and the entire point of this app is measuring what a real
Gombit user gets today. `TestCreateInvalidOwnerIDReturnsInternalError` pins
this actual behavior so a future framework fix (or regression) shows up
here as an expected test change, not a mysterious fairness-check failure.
Fixing `database.MapPersistError` itself is out of scope for BENCH-1 — see
docs/plans/BENCH-1-benchmark-suite.md Phase 3b for the writeup and a
pointer to file it as its own issue.

## Status

Implementation, migration, and both test suites are done and verified
against real PostgreSQL (schema diffed statement-log-for-statement-log
against `gin-gorm`'s N+1 behavior, and cross-checked live via
`TestCrossImplementationFairness`). Both Go apps now have `benchmarks/compose.yml`
services with the §7 budget (see [Under compose](#under-compose-containerized-with-the-7-resource-budget)
above). Still open (tracked in
[docs/plans/BENCH-1-benchmark-suite.md](../../../docs/plans/BENCH-1-benchmark-suite.md)
Phase 3b/6): wiring the cross-implementation fairness check into CI (needs a
lighter seed-size mechanism than the canonical 1,000/100,000, which is
Phase 8's concern), the last ecosystem app's compose service (`nestjs`;
django/rails/laravel are in), and the six-app bring-up/`run-crud` loop.
