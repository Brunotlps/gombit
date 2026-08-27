# gin-gorm

The BENCH-1 primary framework-tax control (issue #141): the canonical
`/api/projects` CRUD API ([../../docs/schema.md](../../docs/schema.md))
implemented with idiomatic Gin + GORM, deliberately without Huma or
`framework.App`. This is the cleanest isolation of Gombit's incremental cost
— same language, runtime, and ORM family as the `gombit` app being compared
against it.

## Run

```sh
docker compose -f benchmarks/compose.yml up -d postgres

# migrate + seed (1,000 users, 100,000 projects) — idempotent, truncates first
go run ./benchmarks/apps/gin-gorm -seed

# serve
go run ./benchmarks/apps/gin-gorm
```

### Under compose (containerized, with the §7 resource budget)

The app is also containerized (`Dockerfile`, built from the **repo root** since
it imports `benchmarks/apps/shared`) and wired into `benchmarks/compose.yml`
with the issue #141 §7 app ceiling (2 vCPU / 1 GiB). Because a compose budget
is only an *intention* — whether `deploy.resources.limits` is honored is
engine/version-dependent — the running container is verified rather than
trusted:

```sh
# build + start postgres and the app (Compose v2 enforces the limits on `up`)
docker compose --env-file benchmarks/config/versions.env \
  -f benchmarks/compose.yml up -d postgres gin-gorm

# seed once (truncate + insert), then the served container keeps running
docker compose --env-file benchmarks/config/versions.env \
  -f benchmarks/compose.yml run --rm gin-gorm seed

# confirm the ceiling actually landed on the live container
go run ./benchmarks/scripts/inspect-limits \
  -container "$(docker compose -f benchmarks/compose.yml ps -q gin-gorm)" \
  -cpus 2 -memory 1g
```

The entrypoint takes three verbs, `migrate`, `seed`, and `serve` (default), so
the orchestrator drives every app the same way; here `migrate` is a no-op (GORM
AutoMigrate runs on seed/serve, and the database is the postgres service's
`POSTGRES_DB`). `make benchmark-crud-all` brings all six apps up and runs
`run-crud` over each; this app is one leg of it.

Env vars (all optional, defaults match `benchmarks/compose.yml`):

| Var | Default | |
| --- | --- | --- |
| `DATABASE_URL` | `postgres://gombit:gombit@127.0.0.1:55432/gombit_bench?sslmode=disable` | |
| `PORT` | `8081` | |
| `POOL_MAX_OPEN` | `20` | issue #141 "Connection pooling" pins this to 20 across every implementation |
| `POOL_MAX_IDLE` | `20` | |

## Test

```sh
# needs a live PostgreSQL instance — this package has no non-integration
# test files of its own; the seed-content formulas it shares with gombit
# are tested once in benchmarks/apps/shared
docker compose -f benchmarks/compose.yml up -d postgres
go test -tags integration ./benchmarks/apps/gin-gorm/... \
  -database.dsn "postgres://gombit:gombit@127.0.0.1:55432/gombit_bench?sslmode=disable"
```

The DB-backed suite is gated behind the `integration` build tag (matching
`database/integration_test.go`'s convention) since it needs a live
PostgreSQL instance. CI runs it in `.github/workflows/ci.yml`'s
`database-postgres` job, against a separate `gombit_bench` database on that
job's Postgres service — not the `gombit` database the `auth`/`database`/
`admin` integration tests use, since `gin-gorm`'s `User` model maps to a
`users` table too (same name, different columns) and sharing a database
would let one `AutoMigrate` alter the schema the other relies on.

Covers: full create/get/update/delete/404/validation-failure round trip,
including that a foreign-key violation (`owner_id` referencing no existing
user) is rejected as a 422 client error, not reported as a 500
(`TestCreateRejectsInvalidOwnerID`); list pagination meta
(`page`/`limit`/`total`) and deterministic `id DESC` ordering across a page
boundary; query-count regression guards for both a non-empty page
(`TestListDoesNotN1`, exactly 3 SQL statements — count, page, one batched
owner `IN (...)` preload, not one owner query per row) and an empty one
(`TestListDoesNotN1EmptyPage`, exactly 2 — no rows means no owners to
preload); and the seed contract's own DB-backed half — the real
truncate-then-seed path run twice at reduced scale to confirm it's
idempotent rather than accumulating duplicate data
(`TestSeedDatabaseNIsIdempotentAndCorrect`). The seed content formulas
themselves (`UserEmail`, `ProjectOwnerID`, ...) and their pure,
no-database tests (`TestSeedContentIsDeterministic`,
`TestProjectOwnerIDRoundRobin`) live in
[../shared](../shared) — shared with `gombit` so the two apps can't
silently seed different content for the same row N.

## Status

Schema, seed, this control app, the `gombit` app implementing the same API,
and the cross-implementation fairness check comparing them are all done
(tracked in [docs/plans/BENCH-1-benchmark-suite.md](../../../docs/plans/BENCH-1-benchmark-suite.md)
Phase 3). See [../gombit/README.md](../gombit/README.md) for the Gombit-side
details, including one discovered framework gap and two bugs the fairness
comparison caught. All six apps are now containerized with `benchmarks/compose.yml`
services carrying the §7 budget (see [Run](#run) above), and
`make benchmark-crud-all` brings each up, records the live `inspect-limits`
verdict as `metadata.resource_limits`, and runs `run-crud` over each. Still open:
wiring the fairness check into automated CI (it needs both databases at the full
canonical 1,000/100,000 scale, deferred to Phase 8's lighter-seed CI work) and
the per-app operational-footprint capture (Phase 6).
