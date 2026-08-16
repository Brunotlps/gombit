# ADR-011: Contract Layer Uses Huma over Gin

## Status

Accepted.

## Context

Gombit needs Go source to be the source of truth for the HTTP API contract.
The locked build plan requires Huma-typed handlers over Gin, OpenAPI 3.1
emission, and a first-class raw `*gin.Engine` escape hatch for routes that are
intentionally outside the public contract.

M0-2 implemented the decision spike in `internal/contractspike`:

- `GET /widgets` and `POST /widgets` are Huma-typed routes and appear in the
  emitted OpenAPI document.
- `GET /raw/ping` is registered directly on Gin and works at runtime while
  remaining absent from `openapi.json`.
- The spike emits OpenAPI 3.1 through Huma and validates the generated document
  in tests.

## Decision

Gombit will use Huma over Gin as the contract layer.

Public API routes that belong in the contract must be registered as Huma typed
operations. Huma input and output structs define request bodies, responses,
validation metadata, and OpenAPI schemas.

Raw Gin remains reachable as a first-class escape hatch through the underlying
router. Routes registered directly on `*gin.Engine` are intentionally outside
the generated OpenAPI contract. Reserve raw Gin for surfaces that should stay
out of the contract, such as webhooks, server-sent events, legacy
compatibility, or protocol-specific endpoints. Framework-owned probes,
metrics, and OpenAPI endpoints remain runtime responsibilities in M1-3; if any
are raw Gin routes, they must stay absent from the spec.

The fallback from the build plan, a bespoke `Bind` layer plus custom OpenAPI
emission, is rejected for now because M0-2 proved the required escape hatch and
contract emission paths.

## Benchmark

The benchmark was rerun locally for M0-3 on 2026-08-15:

```text
goos: linux
goarch: amd64
pkg: github.com/LAA-Software-Engineering/gombit/internal/contractspike
cpu: 12th Gen Intel(R) Core(TM) i7-12650H
BenchmarkHumaGinListWidgets-16      677042      1599 ns/op   1365 B/op   17 allocs/op
BenchmarkPlainGinListWidgets-16    1430974       830.2 ns/op 1130 B/op   11 allocs/op
```

On the spike path, Huma over Gin adds about 770 ns/op and 6 allocations over an
equivalent plain Gin handler. The comparison includes spike implementation
details, including an in-memory store copy that should not be treated as a
runtime service-level objective. The overhead is acceptable for the M0 go/no-go
gate because it buys typed inputs and outputs, validation integration, and
OpenAPI 3.1 emission without a custom contract framework.

## Consequences

- M3 contract work should build on Huma operations, not comment annotations,
  hand-written OpenAPI files, or a custom binding framework.
- Raw Gin routes must remain tested as an escape hatch and must not silently
  leak into the OpenAPI document.
- The current spike `openapi.json` includes Huma's default RFC 9457 Problem
  Details error schema. That is not the Gombit public error contract. **M3-1
  replaces validation (and other Huma-generated) errors with the locked D10
  envelope** via `contract.Install` / `framework.App.API()` — see
  [`docs/contract.md`](../contract.md). M3-2 owns application error categories
  and the category→status mapping table (`contract.StatusFor` /
  `contract.NotFound`, etc.). The `fields` member is
  optional and should appear only when field-level details exist:

```json
{"error":{"code":"...","message":"...","fields":{},"request_id":"..."}}
```

- The runtime extraction work should keep `*gin.Engine` reachable from the app
  surface, matching the build plan requirement for `app.Router()`.
- `internal/contractspike` is an M0 fixture, not runtime source. M1 work must
  not import its `Widget` model or route setup into the framework runtime; keep
  or delete the spike package based on its value as a regression fixture.

## References

- Issue #2: `[M0-2] Contract-layer spike: Huma over Gin`
- Issue #3: `[M0-3] ADR-011: Contract layer = Huma`
- `docs/spikes/M0-2_HUMA_GIN_SPIKE.md`
- `docs/GOMBIT_BUILD_PLAN.md`
