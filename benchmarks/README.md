# Gombit benchmarks

Reproducible benchmark suite for BENCH-1 ([issue #141](https://github.com/gombit-dev/gombit/issues/141)).
Full plan: [docs/plans/BENCH-1-benchmark-suite.md](../docs/plans/BENCH-1-benchmark-suite.md).

## What's here today

```text
benchmarks/
├── compose.yml          PostgreSQL only so far (see docs/schema.md)
├── config/
│   └── versions.env     pinned load generator (k6), Postgres image, resource limits, workload defaults
├── docs/
│   └── schema.md        canonical schema/API/envelope every benchmarks/apps/ implementation targets
├── internal/            machine-readable output plumbing (Phase 1)
│   ├── result/           results.json/results.csv schema + encoders (issue §9)
│   └── metadata/         reproducibility metadata struct + host/toolchain collector
├── scripts/
│   └── collect-host-info/ CLI over internal/metadata: writes metadata.json for a run
├── micro/                Go abstraction-overhead microbenchmarks (Phase 2)
│   ├── scenario/          shared resource types, Huma route registration, correctness assertions
│   ├── nethttp/           net/http row (no router, no framework)
│   ├── gin/               idiomatic plain-Gin row
│   ├── huma/              bare Huma-over-Gin row
│   └── gombit/             full framework.App row
├── apps/                 canonical realistic-application CRUD comparison (Phase 3-4)
│   ├── shared/             response-shape + seed-content types common to both Go implementations
│   ├── gin-gorm/            primary framework-tax control — see its own README
│   ├── gombit/              real Gombit app (Huma, Atlas migrations) — see its own README
│   ├── django/              Django + DRF ecosystem-context app (Phase 4) — see its own README
│   ├── rails/               Rails + ActiveRecord ecosystem-context app (Phase 4) — see its own README
│   ├── laravel/             Laravel + Eloquent ecosystem-context app (Phase 4) — see its own README
│   ├── nestjs/              NestJS + TypeORM ecosystem-context app (Phase 4) — see its own README
│   └── fairness_test.go     cross-implementation check: builds and runs both real binaries, compares over HTTP
└── results/             generated output (issue §9); results/README.md documents the layout
```

All six canonical CRUD implementations now exist (the Go control, the real
Gombit app, and the four ecosystem apps), and the result schema, metadata
collector, and run-config pins are in place. Everything else in the suite's
target layout (`workloads/`, the `make benchmark-*` orchestration that fills
`results/latest/`, `docs/methodology.md`, per-app `compose.yml` services, and
the extension of `fairness_test.go` to all six) is scoped to later phases of
the plan above and doesn't exist yet.

## Result schema and run metadata

`go run ./benchmarks/scripts/collect-host-info` prints the reproducibility
metadata (git SHA + dirty state, OS/kernel/arch, CPU/RAM, Go/Docker/Compose
versions, plus the run parameters passed as flags) as `metadata.json`. The
canonical results shape lives in
[`benchmarks/internal/result`](internal/result) (JSON + CSV encoders); the
Markdown report is always generated from that, never the other way around.
Pinned versions and limits are in
[`benchmarks/config/versions.env`](config/versions.env).

## Running the framework-tax matrix

```sh
go test ./benchmarks/micro/... -bench=BenchmarkFrameworkTax -benchmem -count=10
```

Each row (`nethttp`, `gin`, `huma`, `gombit`) is its own package, so `go test`
runs each as its own process. That's required, not incidental: constructing a
`framework.App` (the `gombit` row) calls `contract.Install`, which replaces
Huma's process-global `huma.NewError` for the rest of that process — see
[micro/gombit/gombit.go](micro/gombit/gombit.go)'s doc comment.

Every row's `TestScenarios` checks the same four scenarios (plaintext, JSON,
path parameter, validated POST) structurally before it's trusted for
benchmarking — see [micro/scenario/assert.go](micro/scenario/assert.go).

## Relationship to the M0-2 spike

This matrix supersedes `internal/contractspike`'s two-stack M0-2 spike
benchmark as the canonical, ongoing framework-tax measurement, but does not
modify or depend on it — that package's widget routes, tests, and committed
`openapi.json` are historical and untouched. See
[docs/spikes/M0-2_HUMA_GIN_SPIKE.md](../docs/spikes/M0-2_HUMA_GIN_SPIKE.md).
