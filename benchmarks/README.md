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
│   ├── metadata/         reproducibility metadata struct + host/toolchain collector
│   ├── k6/               parses a crud-list.js summary into result rows (+ Validate)
│   └── summary/          aggregates trials into per-group stats + renders summary.md (CoV flag)
├── workloads/
│   └── crud-list.js      the headline GET /api/projects read workload (k6)
├── scripts/
│   ├── collect-host-info/ CLI over internal/metadata: writes metadata.json for a run
│   ├── run-crud/          runs crud-list.js (via the pinned k6 image) against one app → results snapshot
│   └── summarize/         results.json -> summary.md (make benchmark-summary)
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

All six canonical CRUD implementations exist, the result schema / metadata
collector / run-config pins are in place, and the headline CRUD-read workload
runs end to end against one implementation (`make benchmark-crud`). Still
scoped to later phases: `docs/methodology.md`, per-app `compose.yml` services
and the loop that brings all six up and runs `run-crud` over each, per-app
resource/RSS capture (Phase 6 footprint), the other `make benchmark-*`
workloads, and extending `fairness_test.go` to all six.

## Result schema and run metadata

`go run ./benchmarks/scripts/collect-host-info` prints the reproducibility
metadata (git SHA + dirty state, OS/kernel/arch, CPU/RAM, Go/Docker/Compose
versions, plus the run parameters passed as flags) as `metadata.json`. The
canonical results shape lives in
[`benchmarks/internal/result`](internal/result) (JSON + CSV encoders); the
Markdown report is always generated from that, never the other way around.
Pinned versions and limits are in
[`benchmarks/config/versions.env`](config/versions.env).

## Running the CRUD-read workload

The headline workload is `GET /api/projects?page=1&limit=20`
([`workloads/crud-list.js`](workloads/crud-list.js)), driven by the pinned k6
image. **Load model and topology (read before citing any number):** the
workload uses a closed-loop `constant-vus` executor — `VUS` concurrent clients
for the measured window — because the issue's concurrency sweep
(1/10/100/500/1000) is a client-count axis. Closed-loop load is subject to
**coordinated omission**: when the app slows, a client waits before its next
request, so the reported tail latency understates true client-observed wait.
This is the issue's "constant-rate *or* document the limitation" path, taken
by documenting it. `gracefulStop` is `0s`, so the measured window is exactly
the requested duration and `duration_seconds` records the *actual* elapsed run
time (from k6's state), not the flag. The k6 container runs on the host
network on the **same machine** as the app (the issue's "another container on
the same host"), so at high concurrency k6 contends for CPU with the app —
this is the recorded topology, not a separate load-generation host.

Against **one already-running, already-seeded** implementation:

```sh
# start + seed the app per its own README, e.g. benchmarks/apps/gin-gorm, then:
make benchmark-crud FRAMEWORK=gin-gorm FRAMEWORK_VERSION=v1.11.0 \
  RUNTIME=go RUNTIME_VERSION=go1.25.7 \
  TARGET_URL='http://127.0.0.1:8081/api/projects?page=1&limit=20'
```

It warms up (a discarded run), measures `TRIALS` times at each concurrency
level in `benchmarks/config/versions.env`, and **merges** its rows into
`benchmarks/results/latest/{results.json,results.csv,metadata.json}` (raw k6
summaries under `raw/`) — re-running one framework replaces that framework's
rows while other frameworks are kept, so running each in turn accumulates all
six. A trial that sends no traffic, has any HTTP error, or fails a content
check (a 200 with the wrong page shape) fails the command loudly **with
nothing written** rather than recording a bogus row — the read workload
against a healthy app must be error-free (`benchmarks/internal/k6`'s
`Summary.Validate`). `run-crud` does not start or resource-constrain the app,
so `metadata.resource_limits` says so honestly; the compose loop that enforces
the §7 limits and captures `cpu_percent`/`rss_bytes` (Phase 6) is the next
slice.

Once `results.json` exists, `make benchmark-summary` regenerates `summary.md`
from it — one table per benchmark, one row per (framework, concurrency). The
headline throughput and latency numbers are the **median** across trials (issue
§7's "report at minimum the median result", robust to one noisy trial); each row
also carries the throughput coefficient of variation (stddev/mean) and a ⚠ flag
on any group whose trials vary by more than 5% (issue §7). The Markdown is
generated from the structured rows, never hand-edited, and leads with the
coordinated-omission / same-host caveats so a copied table can't be read as
more than it is.

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
