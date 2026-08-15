# Go Full-Stack Web Framework — Build Plan & Locked Decisions

**Status:** Build-ready v1.0
**Date:** 2026-08-14
**Supersedes:** the "Open Decisions" (§57) of `GO_FULLSTACK_FRAMEWORK_DESIGN.md`, and the specific sections noted inline below.
**Purpose:** This document is the authoritative source for creating GitHub issues and driving agent-based implementation. The original design doc remains the reference for *rationale and prose*; where the two conflict, **this document wins**.

---

## 0. How to use this document

1. Read §1 (Decisions I changed) first — these reverse the original draft. Veto any you disagree with **before** issues are created; every downstream issue assumes them.
2. §2 locks all remaining open questions so no agent is ever blocked on a product decision.
3. §3 restates the contested architecture decisively (contract pipeline, layout, auth, envelope, generate-vs-runtime rule).
4. §4 is the **issue backlog** — each entry maps 1:1 to a GitHub issue.
5. §5 is the **agent working agreement** — the definition of done. Put this in `CONTRIBUTING.md` and reference it from every issue.

There is exactly **one** human decision left blocking repo creation: the framework name (§2, D1). Everything else is decided here.

---

## 1. Decisions I changed from the draft (veto these first)

| # | Topic | Draft proposed | Locked here | Why the reversal |
|---|---|---|---|---|
| C1 | **Contract layer** | Bespoke `framework.Bind` + hand-built OpenAPI emission | **Adopt Huma over Gin** for typed handlers + validation + OpenAPI 3.1 | Building your own is the "second inferior framework inside the framework" trap (draft §55.4). Huma is the only low-magic way to make "Go is the source of truth" true without comment annotations. |
| C2 | **App layout** | Laravel-style `app/controllers`, `app/models`, `app/services` | **Feature-package under `internal/<feature>/`** | The Laravel tree violates your own principle 6.2 and reads as non-Go. Buffalo/Bud partly died on feeling un-idiomatic; adoption risk you can't afford. Cohesion is kept per-feature. |
| C3 | **Auth default** | Cookie/session default | **Bearer JWT default for v0.1**, cookie/session as M5 preset | Cookie+CSRF+SPA is greenfield and risky; bearer is what already exists (true extraction). Ships faster, de-risks v0.1. (Generated frontend stores the token **in memory, never localStorage**.) |
| C4 | **UI preset default** | MUI default | **Minimal/headless default**, MUI opt-in (`--ui mui`) | Don't couple core to a design system's lifecycle (your own §24.2). Less surface to maintain solo. |
| C5 | **Frontend embedding** | Embed-on-build default | **Split default**, embed via `fw build --embed` | Split is the simpler mental model and the common deploy; embedding is a nice-to-have, not the default path. |
| C6 | **Per-resource repo/service** | Four-layer stack scaffolded per resource | **Thin controller-over-GORM default**, `--service`/`--repo` opt-in | Resolves the §15.1↔§15.2 contradiction. Pass-through service/repo layers for plain CRUD are boilerplate users delete. |

If any of C1–C6 is wrong for you, say which — each has a cluster of issues hanging off it.

---

## 2. Locked decisions (closes original §57)

- **D1 — Name (HUMAN TODO, blocking):** Framework name is the one decision left to you. Until set, all code uses module path `github.com/<org>/<fw>` and binary `fw`. Pick before M0 issue #1.
- **D2 — Repo/org:** New dedicated public repo, not a fork of the template. License **MIT** (matches existing repo). Governance: BDFL/solo for now.
- **D3 — Migration representation:** Go-file migrations with a small fluent `migration.Schema` builder over GORM's migrator. Locked.
- **D4 — Migration metadata:** Track `version, name, batch, applied_at`. **No checksums** — they are near-useless for Go-source migrations (formatting/comments change the hash without changing schema). Locked.
- **D5 — OpenAPI → TS toolchain:** Go side emits OpenAPI 3.1 via Huma. TS side uses **`openapi-typescript`** (types) + **`openapi-fetch`** (client). Both are low-magic and generation-only. Locked.
- **D6 — Package manager:** Detect (prefer `pnpm`, else `npm`). Default `npm` when none present. Locked.
- **D7 — Generated Go file naming:** `snake_case` filenames, PascalCase exported types. Locked.
- **D8 — API prefix:** Default `/api/v1`, configurable. Locked (matches existing repo).
- **D9 — Generic CRUD repo:** `repository.New[T]` lives in the **runtime** as an optional helper. Never generated per-model. Locked.
- **D10 — Response envelope:** `{"data": ..., "meta"?: ...}` success; `{"error": {code, message, fields?, request_id}}` error. This is a **redesign** of the existing `{"error":"string"}`; acceptable for a new framework, but note it in the migration guide. Locked.
- **D11 — Pre-v1 compatibility:** No guarantees before v1.0. Breaking changes documented in CHANGELOG. Locked.
- **D12 — Databases in v0.1:** SQLite + PostgreSQL required and CI-gated. **MySQL deferred to M2 stretch / post-v0.1** — do not let a third dialect slow the first working loop. (Draft made all three v0.1; MySQL demoted to keep scope honest.)

---

## 3. Decisive architecture (replaces contested sections)

### 3.1 Contract pipeline — replaces §13.4/13.5, §14, §23

The Go handler is the source of truth **via Huma typed handlers**, not comments and not a separate OpenAPI file.

```
Huma-typed handler (input/output structs, validated)
        ↓  (Huma emits)
OpenAPI 3.1 document  (served at /openapi.json, written by `fw openapi generate`)
        ↓  (openapi-typescript)
TypeScript types
        ↓  (openapi-fetch, thin generated wrapper)
React client
```

Rules:
- Anything in the public API contract is a Huma handler. Raw `*gin.Engine` remains reachable (`app.Router()`) for anything outside the contract (webhooks, SSE, legacy). The **escape hatch is a first-class, tested path**, not an afterthought.
- Validation lives in the Huma input struct (tags). Validation failures render the D10 error envelope with `fields`.
- The generated TS client and the OpenAPI doc are **build/CI artifacts** — drift between server and client fails CI.

### 3.2 Generated application layout — replaces §10

Feature-package, idiomatic Go:

```
myapp/
├── cmd/server/main.go
├── internal/
│   ├── platform/            # app wiring the framework owns the shape of
│   └── product/             # one package per resource
│       ├── product.go       # model
│       ├── handler.go       # Huma handlers (thin, over GORM by default)
│       ├── service.go       # ONLY if --service
│       ├── repo.go          # ONLY if --repo
│       └── routes.go        # registration
├── database/migrations/
├── database/seeds/
├── config/
├── frontend/                # Vite React app
├── fw.yaml
├── .env.example
├── go.mod
└── README.md
```

`routes.go` per feature is registered explicitly from `main.go` (no reflection discovery — principle 6.2).

### 3.3 The generate-vs-runtime rule — NEW, resolves §55.7

This single rule determines whether `fw upgrade` is ever feasible. State it as law:

> **Behavior lives in the versioned runtime. Generated code is a thin, one-time scaffold the user owns. The framework never rewrites user-owned files.**

Consequences, enforced in review:
- Generators are **idempotent and additive**. Re-running never clobbers edits. `--dry-run` and `--force` required.
- Route registration is edited via `go/ast`, never regex, and only appends to a known registration point.
- `fw upgrade` bumps dependencies and emits **reviewable codemod diffs** — it never edits in place.
- If a feature can live in the runtime instead of in generated code, it must.

### 3.4 Auth — replaces §20 default

- **v0.1 default:** Bearer JWT + refresh rotation (extracted from the template). Generated frontend holds the access token **in memory only**; refresh via the rotation endpoint. No localStorage.
- **M5 preset:** Cookie/session + CSRF (`--auth cookie`). This is greenfield; it gets its own hardening issues and threat-model doc (SPA + separate dev origin + SameSite + double-submit).
- The `X-API-Key` service gate stays available but is **off by default** for browser apps and documented as server-to-server only.

---

## 4. Issue backlog (issue-ready)

Format per issue: **[ID] Title** — scope · acceptance criteria · deps · size (S/M/L) · labels.
Milestones are dependency-ordered; do not start Mn+1 issues that depend on unfinished Mn gates.

### M0 — Bootstrap + Contract Spike (the gate)

- **[M0-1] Create repo, module path, CI skeleton** — new repo, `go.mod` with `github.com/<org>/<fw>`, MIT license, golangci-lint, GitHub Actions running `go test ./...` + lint. AC: green CI on an empty package; branch protection on `main`. deps: D1. size: S. labels: `infra`.
- **[M0-2] Contract-layer spike: Huma over Gin** — wire Huma to a Gin engine; implement two handlers: one Huma-typed resource (`GET/POST /widgets`) that appears in the emitted OpenAPI, and one raw `*gin.Engine` route (`/raw/ping`) that does not. Emit `openapi.json`. AC: OpenAPI 3.1 validates; typed handler shows request/response schema; raw route works and is absent from the spec; a short latency benchmark vs plain Gin recorded. **This issue is a go/no-go gate.** deps: M0-1. size: M. labels: `spike`, `contract`.
- **[M0-3] ADR-011: Contract layer = Huma** — record the decision, the escape-hatch pattern, and the benchmark. If M0-2 fails the escape-hatch test, this ADR instead records the fallback (bespoke `Bind` + emission) and M3 issues are rewritten. deps: M0-2. size: S. labels: `adr`.

### M1 — Runtime extraction from `golang-rest-api-template`

One issue per seam. Each must keep the existing tests green (see §5).

- **[M1-1] Typed config boundary** — introduce a typed `config.Config`; move all `os.Getenv` reads to the config boundary; low-level packages receive typed config. AC: no `os.Getenv` outside `config`; existing behavior unchanged; config validation errors are explicit. deps: M0-1. size: M. labels: `runtime`, `config`.
- **[M1-2] `framework.App` + lifecycle + hooks** — extract app construction and the 15-step lifecycle (draft §11.2) into the runtime; add `OnStart`/`OnStop` with deterministic ordering and bounded shutdown context. AC: minimal example boots via `framework.Run`; graceful shutdown test passes. deps: M1-1. size: L. labels: `runtime`, `lifecycle`.
- **[M1-3] De-domain the router** — remove `Book`/`User` knowledge from router/bootstrap; route registration becomes application-owned; framework mounts only its own endpoints (probes, metrics, openapi). AC: runtime package contains zero example-domain models; middleware order preserved and tested. deps: M1-2. size: M. labels: `runtime`, `http`.
- **[M1-4] Multi-driver `database.Open` + capability model** — SQLite + Postgres via `gorm.Open` switch; `Driver()` and `Capabilities()` exposed; driver-aware pool defaults. AC: same code opens both; capability flags covered by tests. (MySQL is M2 stretch — D12.) deps: M1-1. size: M. labels: `runtime`, `database`.
- **[M1-5] Normalize cache interface** — replace the go-redis-leaking interface with `Get/Set/Delete/Increment` value semantics; memory + redis + noop drivers; `app.Redis()` escape hatch when redis is enabled. AC: rate limiter and cache users compile against new interface; memory driver used in tests. deps: M1-1. size: M. labels: `runtime`, `cache`.
- **[M1-6] Optional Mongo log sink** — Zap stays; Mongo becomes a selectable sink/module, not a runtime dependency; default sink stdout/stderr. AC: app boots and logs with Mongo absent. deps: M1-1. size: S. labels: `runtime`, `logging`.
- **[M1-7] Preserve observability + security tests** — carry over metrics, tracing, probes, request-id/timeout, security headers, trusted-proxy tests into the runtime package. AC: parity test suite green. deps: M1-2. size: M. labels: `runtime`, `tests`.

**M1 exit gate:** a minimal example app boots through the runtime with no example-domain code in the runtime, on both SQLite and Postgres, all extracted tests green.

### M2 — Migrations

- **[M2-1] `migration.Schema` fluent builder** — `CreateTable/DropTable/AddColumn/...` over GORM migrator, mapping portable ops to SQLite + Postgres; `Exec` + `Driver()` escape hatch. AC: portable migration runs on both DBs. deps: M1-4. size: L. labels: `migrations`.
- **[M2-2] `framework_migrations` tracking** — version/name/batch/applied_at (no checksum, D4). AC: up/down/status reflected in table. deps: M2-1. size: S. labels: `migrations`.
- **[M2-3] `fw db` commands** — `migrate/rollback/status/seed/reset`. AC: each command works on both DBs. deps: M2-2. size: M. labels: `cli`, `migrations`.
- **[M2-4] Multi-DB conformance CI** — matrix job runs the DB conformance suite (CRUD, tx, migrate up/down, timestamps, nullable, unique, index, decimal, pagination) on SQLite + Postgres. AC: matrix green; MySQL job added as allowed-failure stretch. deps: M2-3. size: M. labels: `ci`, `database`.

### M3 — Contract pipeline

- **[M3-1] Huma DTO + validation conventions** — request/response struct conventions, validation tags → D10 error envelope with `fields`. AC: invalid request returns structured field errors. deps: M0-3, M1-3. size: M. labels: `contract`, `http`.
- **[M3-2] Response envelope + error mapping** — `{data, meta}` / `{error{code,message,fields,request_id}}`; error categories (draft §41) mapped centrally. AC: envelope covered by tests; category→status mapping table tested. deps: M3-1. size: M. labels: `contract`.
- **[M3-3] OpenAPI emission + `fw openapi generate`** — serve `/openapi.json`; CLI writes it to disk. AC: spec validates; matches live routes. deps: M3-1. size: S. labels: `cli`, `contract`.
- **[M3-4] TS types + client generation + `fw client generate`** — `openapi-typescript` + `openapi-fetch` wrapper; typed errors map to the D10 envelope. AC: generated client compiles against a sample spec. deps: M3-3, D5. size: M. labels: `cli`, `frontend`, `contract`.
- **[M3-5] Contract drift check in CI** — regenerate spec + client; fail if the working tree changes. AC: intentional server change without regen fails CI. deps: M3-4. size: S. labels: `ci`, `contract`.

### M4 — CLI + generators

- **[M4-1] `fw new`** — interactive + non-interactive scaffold; DB/cache/auth/UI flags; feature-package layout (§3.2); `fw.yaml`, `.env.example` splitting server vs `VITE_*` public values. AC: `fw new demo --database sqlite` produces a compiling app. deps: M1 exit, M2, M3. size: L. labels: `cli`, `generator`.
- **[M4-2] `fw dev`** — Go reload (air/watchexec), Vite, `/api` proxy, OpenAPI watch→regenerate, service table. AC: one command runs backend+frontend with HMR and live contract regen. deps: M4-1. size: L. labels: `cli`, `devx`.
- **[M4-3] `fw make resource` (AST-safe)** — generates model, Huma handler (thin over GORM), routes, migration, and frontend pages/forms/table; registers routes via `go/ast`; idempotent; `--dry-run`/`--force`; `--service`/`--repo` opt-in (C6). AC: generated resource works backend→frontend with no manual type duplication; re-run doesn't clobber edits. deps: M4-1, M3-4. size: L. labels: `cli`, `generator`.
- **[M4-4] Introspection: `fw routes`, `fw doctor`, `fw config show`** — routes table; doctor checks (Go/Node, config, DB/Redis connectivity, migration status, ports, insecure prod settings). AC: doctor flags a deliberately-broken config. deps: M4-1. size: M. labels: `cli`, `devx`.
- **[M4-5] Generator golden tests** — for each generator: run against a fixture, diff against golden, compile backend, typecheck frontend, verify idempotency. AC: golden suite green in CI. deps: M4-3. size: M. labels: `tests`, `generator`.

**M4 exit gate:** `fw new` → `fw dev` → `fw make resource` → working authenticated CRUD app, backend-to-frontend, no hand-written contract, on SQLite + Postgres.

### M5 — Frontend + auth polish

- **[M5-1] Vite React skeleton (minimal preset)** — router, providers, generated client wiring, error→form mapping (React Hook Form). AC: skeleton builds and talks to the API. deps: M3-4. size: M. labels: `frontend`.
- **[M5-2] Bearer auth integration** — in-memory access token, refresh rotation, protected routes. AC: login→access protected route→refresh→logout E2E. deps: M5-1. size: M. labels: `frontend`, `auth`.
- **[M5-3] Cookie/session + CSRF preset (`--auth cookie`)** — HttpOnly/Secure/SameSite cookies, CSRF for state-changing requests, threat-model doc. AC: CSRF + cookie-attribute security tests pass. deps: M5-2. size: L. labels: `auth`, `security`, `greenfield`.
- **[M5-4] MUI preset (`--ui mui`)** — port the monorepo's MUI CRUD patterns as an opt-in preset. AC: `--ui mui` scaffolds MUI screens. deps: M5-1. size: M. labels: `frontend`, `preset`.
- **[M5-5] Optional `go:embed` build (`fw build --embed`)** — single-binary with SPA fallback. AC: embedded binary serves API + static + index fallback. deps: M4-2. size: M. labels: `build`.

### M6 — Deferred batteries (POST-v0.1, not on the critical path)

Each is a **future epic**, explicitly out of v0.1. Do not create these as active issues until v0.1 ships: jobs/queues, events, scheduler, mail, storage, optional gRPC, multi-tenancy hooks, i18n. Park them in a "post-v0.1" project column.

---

## 5. Agent working agreement (definition of done)

Put this in `CONTRIBUTING.md`; reference from every issue. A PR is **not** done unless:

1. **Tests:** new behavior has unit tests; runtime changes keep the extracted suite green; DB-touching changes pass the SQLite + Postgres matrix.
2. **Docs + example:** every stable feature ships docs and appears in an example app.
3. **Extraction discipline:** do not rewrite code that passes its tests. Refactor and move; preserve contracts (draft §2.3). If a "small extraction" turns into a rewrite, stop and open a discussion issue.
4. **Generator safety:** Go source is modified via `go/ast`/`go/format` only — never regex. Generators are idempotent, support `--dry-run`/`--force`, print created/modified files, and never silently overwrite user-owned files (§3.3).
5. **Security invariants:** no secrets in generated frontend source; `VITE_*` treated as public; production config validation must loudly fail the cases in draft Appendix C.
6. **Contract integrity:** any API change regenerates OpenAPI + TS client in the same PR; CI drift check must pass (M3-5).
7. **Scope guard:** if an issue starts pulling in an M6 battery, split it out. v0.1 is the one CRUD loop, nothing more.
8. **Every PR links its issue** and states which acceptance criteria it satisfies.

---

## 6. Suggested GitHub labels & milestones

Milestones: `M0 spike`, `M1 runtime`, `M2 migrations`, `M3 contract`, `M4 cli`, `M5 frontend-auth`, `post-v0.1`.
Labels: `infra`, `runtime`, `config`, `lifecycle`, `http`, `database`, `cache`, `logging`, `migrations`, `contract`, `cli`, `generator`, `devx`, `frontend`, `auth`, `security`, `build`, `preset`, `tests`, `ci`, `adr`, `spike`, `greenfield`, `good-first-issue`.

`good-first-issue` candidates for agents to warm up on: M0-1, M1-6, M2-2, M3-3, M4-4.

---

## 7. Critical-path summary

```
D1 (name) → M0-1 → M0-2/M0-3 (GATE) → M1 (extraction) → M2 (migrations)
                                              ↓
                                     M3 (contract pipeline)
                                              ↓
                                   M4 (cli + generators) → v0.1 GATE
                                              ↓
                                     M5 (frontend + auth polish) → v0.1 release
```

Everything in M6 waits behind the v0.1 release. If the M0-2 spike fails its escape-hatch test, pause and rework M3 before proceeding — that is the one place a bad result invalidates downstream issues.
