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

Env vars (all optional, defaults match `benchmarks/compose.yml`):

| Var | Default | |
| --- | --- | --- |
| `DATABASE_URL` | `postgres://gombit:gombit@127.0.0.1:55432/gombit_bench?sslmode=disable` | |
| `PORT` | `8081` | |
| `POOL_MAX_OPEN` | `20` | issue #141 "Connection pooling" pins this to 20 across every implementation |
| `POOL_MAX_IDLE` | `20` | |

## Test

```sh
docker compose -f benchmarks/compose.yml up -d postgres
go test -tags integration ./benchmarks/apps/gin-gorm/... \
  -database.dsn "postgres://gombit:gombit@127.0.0.1:55432/gombit_bench?sslmode=disable"
```

Gated behind the `integration` build tag (matching `database/integration_test.go`'s
convention) since it needs a live PostgreSQL instance — `go test ./...`
without the tag skips this package's tests entirely, no DSN required.

Covers: full create/get/update/delete/404/validation-failure round trip;
list pagination meta (`page`/`limit`/`total`) and deterministic `id DESC`
ordering across a page boundary; and a query-count regression guard
(`TestListDoesNotN1`) asserting the list endpoint issues exactly 3 SQL
statements — count, page, one batched owner `IN (...)` preload — regardless
of page size, not one owner query per row.

## Status

Schema, seed, and this control app are done. Still open (tracked in
[docs/plans/BENCH-1-benchmark-suite.md](../../../docs/plans/BENCH-1-benchmark-suite.md)
Phase 3): the `gombit` app implementing the same API, and the
cross-implementation fairness checks (same route surface, same JSON shape,
same query count) that only make sense once a second implementation exists
to compare against. `benchmarks/compose.yml` currently only runs the
`postgres` service; an app service for this container is added once
resource-limit parity across implementations is being measured, not before.
