# M0-2 Huma over Gin Spike

## Scope

This spike wires Huma to a Gin engine using `humagin`, then proves the raw
Gin router remains available for routes outside the OpenAPI contract.

Implemented routes:

- `GET /widgets` — Huma-typed handler, included in `openapi.json`.
- `POST /widgets` — Huma-typed handler with a JSON request body, included in
  `openapi.json`.
- `GET /raw/ping` — raw Gin handler, intentionally absent from `openapi.json`.

The generated OpenAPI 3.1 document is committed at the repository root as
`openapi.json`.

## Out of Scope

The committed `openapi.json` includes Huma's default RFC 9457 Problem Details
error schema (`ErrorModel`, `application/problem+json`). That is a spike
artifact only. It must not be treated as Gombit's final public error contract
or copied into the runtime unchanged.

M3-1/M3-2 remain responsible for replacing Huma's default error mapping with
the locked D10 shape:

```json
{"error":{"code":"...","message":"...","fields":{},"request_id":"..."}}
```

ADR-011 should repeat this caveat when it records the M0-2 decision.

## Benchmark

Run with:

```sh
go test ./internal/contractspike -bench=. -benchmem -run '^$'
```

Results will be recorded from the PR validation run.

Local result on 2026-08-15:

```text
goos: linux
goarch: amd64
pkg: github.com/gombit-dev/gombit/internal/contractspike
cpu: 12th Gen Intel(R) Core(TM) i7-12650H
BenchmarkHumaGinListWidgets-16       669751      1740 ns/op   1365 B/op   17 allocs/op
BenchmarkPlainGinListWidgets-16     1423093       788.7 ns/op 1130 B/op   11 allocs/op
```

On this spike path, Huma-over-Gin adds roughly 950 ns/op and 6 allocations over
the equivalent plain Gin handler while providing typed inputs/outputs,
validation, and OpenAPI 3.1 emission. This is acceptable for the M0 go/no-go
gate; ADR-011 should rerun the benchmark after review and record the final
decision.

## Successor: BENCH-1

This two-stack `BenchmarkHumaGinListWidgets` / `BenchmarkPlainGinListWidgets`
result stays as the historical M0 go/no-go record above — it is not rerun,
overwritten, or extended in place. The canonical, ongoing framework-tax
benchmark is the four-stack matrix (`net/http` -> Gin -> Huma+Gin -> Gombit)
added for [BENCH-1](../plans/BENCH-1-benchmark-suite.md) (issue #141), living
under [`benchmarks/micro/`](../../benchmarks/README.md) rather than in this
package:

```sh
go test ./benchmarks/micro/... -bench=BenchmarkFrameworkTax -benchmem -count=10
```

Each row (`benchmarks/micro/{nethttp,gin,huma,gombit}`) is its own package
carrying the same four scenarios, sharing resource types and Huma route
registration via `benchmarks/micro/scenario`. Any future README performance
numbers must come from this matrix, not from this spike's 2026-08-15 result.
