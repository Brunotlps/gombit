# Gombit benchmarks

Reproducible benchmark suite for BENCH-1 ([issue #141](https://github.com/gombit-dev/gombit/issues/141)).
Full plan: [docs/plans/BENCH-1-benchmark-suite.md](../docs/plans/BENCH-1-benchmark-suite.md).

## What's here today

```text
benchmarks/
└── micro/              Go abstraction-overhead microbenchmarks (Phase 2)
    ├── scenario/        shared resource types, Huma route registration, correctness assertions
    ├── nethttp/         net/http row (no router, no framework)
    ├── gin/             idiomatic plain-Gin row
    ├── huma/            bare Huma-over-Gin row
    └── gombit/           full framework.App row
```

Everything else in the suite's target layout (`apps/`, `workloads/`, `scripts/`,
`config/`, `results/`, `docs/methodology.md`, `compose.yml`, `Makefile`) is
scoped to later phases of the plan above and doesn't exist yet.

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
