# Gombit

[![CI](https://github.com/gombit-dev/gombit/actions/workflows/ci.yml/badge.svg)](https://github.com/gombit-dev/gombit/actions/workflows/ci.yml)
[![Release](https://github.com/gombit-dev/gombit/actions/workflows/release.yml/badge.svg)](https://github.com/gombit-dev/gombit/actions/workflows/release.yml)
[![Go 1.25+](https://img.shields.io/badge/Go-1.25+-00ADD8?logo=go)](https://go.dev/dl/)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)
[![Go Reference](https://pkg.go.dev/badge/github.com/gombit-dev/gombit.svg)](https://pkg.go.dev/github.com/gombit-dev/gombit)

**A Django-for-Go full-stack framework.** One CLI scaffolds a typed Go API, its
OpenAPI document, a matching TypeScript client, a React frontend, versioned SQL
migrations, session auth, and a working admin — then builds the whole thing into
a single binary.

```bash
go install github.com/gombit-dev/gombit/cmd/gombit@latest
gombit new tasks --database sqlite --auth cookie --ui mui
cd tasks && gombit dev
```

> **Status: pre-1.0.** M0–M5 and ADMIN-1..3 are complete and CI-gated across
> SQLite, PostgreSQL, and MySQL. APIs may still change between minor versions.

## Why Gombit

Go has excellent HTTP routers. What it doesn't have is the thing Django users
miss on day one: the **batteries**, wired together and agreeing with each other.

Gombit's position is that the pieces you'd otherwise assemble by hand — schema,
API contract, typed client, auth, admin — should be **derived from one source of
truth** instead of hand-synchronized:

- **Your handler signature is the contract.** OpenAPI 3.1 is emitted from
  Huma-typed handlers, never hand-written, and the TypeScript client is
  generated from that. A drift check fails CI when they disagree.
- **Your GORM model is the schema.** Migrations are versioned SQL diffed by
  Atlas from your models — readable, reviewable, and reversible. No
  `AutoMigrate` in production, no hand-rolled migration DSL.
- **Your registry is the admin.** A real Django-style admin at `/admin/`,
  served by the framework at runtime, not generated pages you inherit and
  maintain.

And the escape hatches are real: `app.Router()` hands you the raw `*gin.Engine`,
tested and first-class.

## What's in the box

| | |
| --- | --- |
| **Runtime** | Gin + [Huma](https://huma.rocks/) with typed handlers, `framework.App` lifecycle hooks, graceful shutdown, structured logging, typed env config with secret redaction |
| **Data** | GORM over **SQLite, PostgreSQL, and MySQL** — all three CI-gated on every push, with a shared conformance suite |
| **Migrations** | [Atlas](https://atlasgo.io/)-backed `gombit db makemigrations / migrate / rollback / status / seed / reset` |
| **Contract** | OpenAPI 3.1 emitted from code, interactive `/docs`, generated TypeScript client, contract drift check |
| **Frontend** | Vite + React + TypeScript, React Hook Form, optional Material UI CRUD preset (`--ui mui`) |
| **Auth** | Bearer JWT with refresh rotation (token in memory, **never `localStorage`**), or first-class cookie sessions with CSRF (`--auth cookie`) |
| **Admin** | Runtime generic admin at `/admin/` with introspection API, permissions, groups, and superuser bypass |
| **CLI** | Cobra tree: `new`, `dev`, `build --embed`, `make resource`, `make command`, `db`, `openapi`, `client`, `routes`, `doctor`, `config`, `createsuperuser`, `version` |
| **Deploy** | `gombit build --embed` — API, SPA, and admin in one binary |

## Quick start

**Prerequisites:** Go 1.25+, Node 22+, and a C toolchain (SQLite is cgo-only).
Migrations also need [Atlas](https://atlasgo.io/):
`curl -sSf https://atlasgo.sh | sh -s -- --community`. Full details in
[installation.md](docs/installation.md).

```bash
# 1. Install
go install github.com/gombit-dev/gombit/cmd/gombit@latest

# 2. Scaffold
gombit new tasks --database sqlite --auth cookie --ui mui
cd tasks

# 3. Run the API and frontend together
gombit dev
```

`gombit dev` serves the Go API and Vite together, proxies `/api` and
`/openapi.json`, and regenerates the TypeScript client whenever the spec
changes:

| URL | |
| --- | --- |
| <http://127.0.0.1:5173> | React app |
| <http://127.0.0.1:8080/docs> | interactive API docs |
| <http://127.0.0.1:8080/admin/> | admin SPA |

### The CRUD loop

```bash
# Generate a feature package: model + Huma handler + routes + React pages.
gombit make resource Task title:string:required done:bool

# Diff your models into versioned SQL, then apply it.
gombit db makemigrations create_tasks --model github.com/example/tasks/internal/task.Task
gombit db migrate

# Regenerate the typed client from the live spec.
gombit client generate

# Create an admin account and open /admin/.
export GOMBIT_JWT_SECRET="$(openssl rand -hex 32)"
gombit createsuperuser --email admin@example.com
```

`make resource` edits `cmd/server/main.go` through `go/ast` — never regex — to
register your routes and models. Generators are idempotent and additive, support
`--dry-run` and `--force`, and never overwrite files you own.

Then ship it:

```bash
gombit build --embed   # one binary: API + SPA + admin
```

**Next:** the [tutorial](docs/tutorial.md) walks this whole loop with
explanations, and [`examples/tutorial/`](examples/tutorial) is the finished app.

## Architecture

```mermaid
flowchart LR
  Model[GORM models] --> Atlas[Atlas diff]
  Atlas --> SQL[(versioned SQL)]
  Model --> Handler[Huma-typed handlers]
  Handler --> Gin[Gin router]
  Handler --> Spec[OpenAPI 3.1]
  Spec --> TS[TypeScript client]
  TS --> React[React + Vite]
  Model --> Registry[admin registry]
  Registry --> AdminUI["/admin/ SPA"]
  Gin --> Binary[single binary]
  React --> Binary
  AdminUI --> Binary
```

Both arrows out of your model are the point: one declaration drives the schema
and the API, and one API drives the client.

The response envelope is fixed so clients can rely on it — success is
`{"data": ..., "meta"?: ...}`, and errors carry a machine-readable code plus
per-field detail:

```json
{
  "error": {
    "code": "validation_error",
    "message": "The request contains invalid fields.",
    "fields": {"title": ["expected length >= 1"]},
    "request_id": "5db935cd-7c74-4ffe-a4de-0fa817451f54"
  }
}
```

## The admin

No other Go web framework ships a real Django-style admin — not Gin, Echo,
Fiber, or Encore. Gombit's is a **runtime** surface over an explicit registry:

```go
admin.Register(app, Task{}, admin.Options{
	Slug:   "tasks",
	List:   []string{"title", "done"},
	Search: []string{"title"},
})
```

That gives you list, detail, create, update, and delete at `/admin/`, backed by
`GET /api/v1/admin/meta` and a generic `/api/v1/admin/resources/{slug}` data
plane. Permissions default to `admin.{slug}.{action}`, are granted directly or
through groups, and superusers bypass them.

Registration is explicit and typed — resolved once at startup, with no
request-time reflection over your models, and no generated admin pages for you
to maintain. Requires cookie auth. See [admin.md](docs/admin.md) and
[ADR-013](docs/adr/013-runtime-generic-admin.md).

## Compared with

| | Gombit | Gin / Echo / Fiber | Buffalo | Encore |
| --- | --- | --- | --- | --- |
| HTTP routing | ✅ (Gin) | ✅ | ✅ | ✅ |
| Typed OpenAPI from code | ✅ | ➖ add-on | ➖ | ✅ |
| Generated TS client + drift check | ✅ | ➖ | ➖ | ✅ |
| Versioned SQL migrations | ✅ (Atlas) | ➖ | ✅ (fizz) | ✅ |
| Scaffolding generators | ✅ AST-safe | ➖ | ✅ | ➖ |
| Session auth + CSRF | ✅ | ➖ | ✅ | ➖ |
| **Django-style admin** | ✅ | ➖ | ➖ | ➖ |
| Self-hosted, no vendor runtime | ✅ | ✅ | ✅ | ➖ |

Gombit is younger than all of them. If you want a minimal router, use Gin
directly — Gombit *is* Gin underneath, and hands it back to you on request.

## Performance

The [`benchmarks/`](benchmarks/) suite measures the same canonical
`/api/projects` CRUD app across six stacks (Gin+GORM, Gombit, Django, Rails,
Laravel, NestJS) under fixed resource limits, plus each container's operational
footprint. The block below is generated by `make benchmark-report` from
`benchmarks/results/latest/` — do not edit it by hand; a CI job
(`benchmark-report-drift`) fails if it no longer matches the generator. **Read
[the methodology](benchmarks/docs/methodology.md), especially
"How not to interpret these results", before citing any figure:** these are
same-host, closed-loop numbers, not a cross-language leaderboard.

<!-- benchmark-results:start -->
> ## ⚠️ UNPUBLISHABLE DEVELOPMENT RUN
>
> Generated from a **dirty working tree** (uncommitted changes), so these numbers are **not reproducible** and must not be cited. Commit the tree and re-run the suite (`make benchmark-crud-all benchmark-footprint benchmark-micro benchmark-report`) before publishing.

> ### Reduced development snapshot
>
> This run used a **narrower protocol than the canonical dedicated-host run** (pinned in `benchmarks/config/versions.env`; see [benchmarks/docs/methodology.md](benchmarks/docs/methodology.md)): it is a development sample, not the published benchmark. Differs in concurrency 1/10/100 (canonical 1/10/100/500/1000); 3 trials (canonical 5 trials); 10s per trial (canonical 30s); 3s warm-up (canonical 10s).

_Generated by `make benchmark-report` from `benchmarks/results/latest/` — do not edit by hand._

Numbers are a same-host, closed-loop snapshot under fixed resource limits; read [benchmarks/docs/methodology.md](benchmarks/docs/methodology.md) — especially its "How not to interpret these results" section — before citing any figure.

### Framework tax — `net/http` → Gin → Huma → Gombit

Per-request overhead of each layer on the same machine for the **validated typed POST** (median ns/op, B/op, allocs/op; lower is better; `vs net/http` is the relative cost) — the same-language, same-runtime cost of adopting Gombit. The other four scenarios (plaintext, json, path-param, invalid-post) are in `benchmarks/results/latest/microbench.json`.

| stack | ns/op | B/op | allocs/op | vs net/http |
| --- | ---: | ---: | ---: | ---: |
| net/http | 1560 ⚠ | 2097 | 21 | 1.0× |
| Gin | 3992 ⚠ | 2311 | 29 | 2.6× |
| Huma + Gin | 3463 ⚠ | 2301 | 37 | 2.2× |
| Gombit | 9546 | 7919 | 95 | 6.1× |

⚠ marks a rung whose ns/op varied by more than 5% across samples — a noisy series (e.g. on a contended host); its median, and any ordering against a neighbouring rung, should be distrusted.

### PostgreSQL CRUD read — `GET /api/projects?page=1&limit=20`

At **100 concurrent clients**, median across trials: throughput (higher is better) and tail latency (lower is better). p50/p95/p99 are the median across trials of each per-trial percentile; ⚠ marks a group whose throughput varied by more than 5% across trials — read its row with care.

| framework | req/s | p50 ms | p95 ms | p99 ms |
| --- | ---: | ---: | ---: | ---: |
| django | 357 | 268.3 | 350.6 | 387.6 |
| gin-gorm | 363 | 213.0 | 500.2 | 690.6 |
| gombit | 348 | 291.7 | 503.1 | 695.6 |
| laravel | 115 ⚠ | 877.5 | 911.3 | 988.9 |
| nestjs | 333 ⚠ | 295.9 | 391.7 | 403.8 |
| rails | 455 | 212.3 | 260.5 | 274.0 |

### Operational footprint

Container-start cold start (median) and memory — lower is better. CPU is the median percent one container drew during the closed-loop load (100 = one core); it is *not* a quality score — a faster app that does more work in the window can show higher CPU, so read it against the throughput row.

| framework | cold start (ms) | idle (MB) | loaded (MB) | CPU (%) | image (MB) |
| --- | ---: | ---: | ---: | ---: | ---: |
| django | 846 | 165.1 | 204.2 | 195 | 56.5 |
| gin-gorm | 117 | 5.3 | 24.0 | 28 | 21.1 |
| gombit | 62 | 6.4 | 30.4 | 30 | 87.6 |
| laravel | 229 | 53.9 | 96.1 | 193 | 183.3 |
| nestjs | 457 | 60.0 | 102.4 | 57 | 164.3 |
| rails | 1018 | 95.0 | 213.4 | 128 | 106.7 |

### How these were measured

- **Host:** 12th Gen Intel(R) Core(TM) i7-12650H, 16 logical CPUs, 7.6 GiB RAM (linux/amd64, kernel 5.15.167.4-microsoft-standard-WSL2)
- **Commit / date:** `c6051f20fb3f` (dirty), 2026-08-27T09:41:50Z
- **PostgreSQL:** postgres:16.4-alpine
- **Resource limits:** enforced: cpu 2.00 vCPU (intended 2.00 vCPU), memory 1 GiB (intended 1 GiB)
- **Protocol:** concurrency 1/10/100 VUs, 3 trials × 10s each (warm-up 3s)
- **Load generator:** grafana/k6:0.55.0. Full method: [benchmarks/docs/methodology.md](benchmarks/docs/methodology.md).
<!-- benchmark-results:end -->

## Documentation

Start with [**installation**](docs/installation.md) and the
[**tutorial**](docs/tutorial.md). The full index — runtime, data, contract,
frontend, auth, admin, and ADRs — is at [**docs/README.md**](docs/README.md).

Scope, locked architecture decisions, and the issue backlog live in
[`docs/GOMBIT_BUILD_PLAN.md`](docs/GOMBIT_BUILD_PLAN.md), which is authoritative.

## Roadmap

**Shipped:** typed config and lifecycle (M1) · Atlas migrations (M2) · Huma
contract, OpenAPI, and TS client (M3) · Cobra CLI and generators (M4) · React
frontend, Bearer and cookie auth, MUI preset, embedded builds (M5) · the runtime
admin with permissions (ADMIN-1..3).

**Post-v0.1**, deliberately not here yet: background jobs and queues, events,
scheduler, mail, storage, gRPC, multi-tenancy, i18n.

## Contributing

Issues and pull requests are welcome — start with
[CONTRIBUTING.md](CONTRIBUTING.md).

- [Report a bug or request a feature](https://github.com/gombit-dev/gombit/issues/new/choose)
- [Security policy](SECURITY.md) — report vulnerabilities privately, not as issues
- [Code of conduct](CODE_OF_CONDUCT.md)
- [Changelog](CHANGELOG.md)

## License

[MIT](LICENSE) © Gombit
