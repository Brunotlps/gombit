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
pkg: github.com/LAA-Software-Engineering/gombit/internal/contractspike
cpu: 12th Gen Intel(R) Core(TM) i7-12650H
BenchmarkHumaGinListWidgets-16       812252      1617 ns/op   1364 B/op   17 allocs/op
BenchmarkPlainGinListWidgets-16     1309664       781.8 ns/op 1129 B/op   11 allocs/op
```

On this spike path, Huma-over-Gin adds roughly 835 ns/op and 6 allocations over
the equivalent plain Gin handler while providing typed inputs/outputs,
validation, and OpenAPI 3.1 emission. This is acceptable for the M0 go/no-go
gate; ADR-011 should rerun the benchmark after review and record the final
decision.
