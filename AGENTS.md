# AGENTS.md

Gombit is a Django-for-Go full-stack framework: Go backend (Gin + Huma + GORM),
a generated React+TypeScript frontend, Atlas-backed migrations. Module path
`github.com/LAA-Software-Engineering/gombit`.

## Current state

This repo is in the **M0 bootstrap** stage. It has a root `go.mod`, a minimal
root Go package, CI/lint wiring, and docs. It does not yet have runtime
framework source, generated app templates, migrations, frontend source, or
example apps. Don't assume a runtime codebase layout exists; check `git log` /
`ls` before describing "how the code works."

## Source of truth

- `docs/GOMBIT_BUILD_PLAN.md` is authoritative for scope, decisions, and the
  issue backlog (§4). It wins over the design doc on any conflict.
- `docs/GO_FULLSTACK_FRAMEWORK_DESIGN.md` is rationale/prose only, cited by
  backlog entries (e.g. "draft §41") for context — never a source of
  additional scope on its own.
- GitHub issues, one per §4 backlog entry, titled `[ID] ...` (e.g. `[M1-2]
  framework.App + lifecycle + hooks`), are the unit of work. Milestones run
  `M0 spike` → `M1 runtime` → `M2 migrations` → `M3 contract` → `M4 cli` →
  `M5 frontend-auth` → `M6 admin` → `post-v0.1`. Don't start an issue whose
  "Depends on #N" is still open.

## Locked architecture decisions (build plan §1-§3 — do not re-litigate)

- **Contract layer:** Huma-typed handlers over Gin are the source of truth for
  the API contract (OpenAPI 3.1 emitted, not hand-written). Raw `*gin.Engine`
  stays reachable via `app.Router()` as a first-class, tested escape hatch.
- **App layout (generated apps):** feature-package under `internal/<feature>/`
  (model, handler, routes; `service.go`/`repo.go` only with `--service`/
  `--repo`). Never Laravel-style `app/controllers`, `app/models`.
- **Migrations:** wrap `ariga.io/atlas-provider-gorm` (Program Mode) +
  `atlas migrate diff`. Never hand-roll a migration DSL.
- **Auth:** Bearer JWT (access token in memory, never `localStorage`) is the
  v0.1 API default; session/cookie is first-class, not a preset, and is a
  hard prerequisite of the admin milestone.
- **Response envelope (D10):** success `{"data": ..., "meta"?: ...}`, error
  `{"error": {code, message, fields?, request_id}}`. Don't invent another
  shape.
- **Generators:** idempotent and additive, with `--dry-run`/`--force`. Go
  source is modified via `go/ast`/`go/format` only — never regex — and never
  overwrites user-owned files.

## Agent working agreement (definition of done — build plan §5)

A change is not done unless:

1. New behavior has tests; DB-touching changes pass the SQLite + PostgreSQL
   matrix.
2. Stable features ship docs and appear in an example app.
3. Extraction from `golang-rest-api-template` preserves contracts — refactor
   and move, don't rewrite code that already passes its tests.
4. Any API change regenerates the OpenAPI doc + TS client in the same PR.
5. No secrets in generated frontend source; `VITE_*` is treated as public.
6. Scope stays inside the issue's milestone. If work starts pulling in an M6
   "battery" (jobs, events, scheduler, mail, storage, gRPC, multi-tenancy,
   i18n), stop and split it out — v0.1 is one CRUD loop, nothing more.
7. The PR links its issue and states which acceptance criteria it satisfies.

## Working conventions

- One issue → one PR where practical; reference the issue number in the PR.
- Conventional commit prefixes (`feat:`, `fix:`, `docs:`, `chore:`) are fine;
  the build plan doesn't mandate anything stricter.
- Don't create milestones/labels beyond build plan §6, and don't create
  issues beyond §4 backlog entries, without asking first.
- If something looks missing from the backlog, flag it — don't silently add
  scope.

## Code review

Before opening or merging a PR, review the diff against the working
agreement above. In Claude Code, run the project `code-review` skill
(`.claude/skills/code-review/SKILL.md`); in Cursor, run the project
`code-review` skill (`.cursor/skills/code-review/SKILL.md`).

## Cursor skills

Project skills live in `.cursor/skills/` and encode the workflows above.
Invoke them with `/create-feature`, `/code-review`, or `/bugfix`, or by
asking in those terms.

- **create-feature** — implement one backlog issue or new capability.
  Place code per the generate-vs-runtime rule and feature-package layout;
  finish only when the working agreement is met.
- **code-review** — review a diff/PR against this file and build plan §5.
  Use it before opening or merging a PR. It is the Cursor checklist for
  this repo (not a generic review).
- **bugfix** — reproduce, add a failing test, fix the root cause only,
  then verify. Do not use it for new features.
