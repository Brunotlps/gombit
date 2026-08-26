# Implementation Plan: [BENCH-1] Reproducible framework benchmark suite + README performance results

**Issue:** [#141](https://github.com/gombit-dev/gombit/issues/141)
**Backlog ID:** BENCH-1 (new — issue #141 is explicit authorization to add this one
entry to `docs/GOMBIT_BUILD_PLAN.md` §4; see §2 below)
**Milestone:** `post-v0.1`
**Size:** XL — too large for one PR; executed as ~8 sequential PR slices against
this one issue (see §6)
**Labels:** `infra`, `devx`, `tests`, `ci`

This plan condenses issue #141's ~30 sections and 100+ acceptance-criteria
checkboxes into an engineering plan: what already exists, what decisions the
issue leaves open, and a PR-sized execution order. It does not restate every
line of the issue — read #141 for the full requirements; this doc is the "how."

---

## 1. Goal

Answer two questions with reproducible, checked-in evidence, and publish a
concise, honest README section from it:

1. **Framework tax** — what does Gombit's Huma/GORM/validation/auth layer cost
   versus hand-assembled Go (`net/http` → Gin → Huma+Gin → Gombit)?
2. **Application-level cost** — how does a realistic Gombit app compare with
   Gin+GORM (primary control, same language/runtime/ORM family) and, for
   ecosystem context only, Django+DRF / Rails / Laravel / NestJS, under
   identical PostgreSQL, schema, seed data, and resource limits?

**Definition of done** (full checklist is issue #141 §25; the load-bearing
items are):

- [x] `[BENCH-1]` added to the build-plan backlog (§2)
- [ ] `benchmarks/` harness runs end-to-end from a clean checkout with one
      command, and each family (`micro`, `crud`, `auth`, `techempower`,
      `footprint`) is independently runnable
- [ ] All 6 required implementations (Gombit, Gin+GORM, Django+DRF, Rails,
      Laravel, NestJS) exist, share one PostgreSQL schema/seed, and pass
      fairness/equivalence checks
- [ ] `results.json` / `results.csv` / `summary.md` are generated, never
      hand-edited; README numbers are generated from the same data with a
      drift check in CI
- [ ] PR CI runs a smoke benchmark only (correctness, not performance gate);
      a `workflow_dispatch` job runs the full suite on demand
- [ ] Root `README.md` gets a `## Performance` section: framework-tax table,
      PostgreSQL CRUD table, footprint table, methodology link, no unqualified
      "fastest" claims
- [x] The existing M0 Huma/Gin spike benchmark (`internal/contractspike`,
      `docs/spikes/M0-2_HUMA_GIN_SPIKE.md`, ADR-011) is preserved as historical
      record, not deleted or silently reused as the final number

---

## 2. Governance: build-plan entry to add in Phase 1

Issue #141 pre-authorizes exactly one new backlog line. Add it to
`docs/GOMBIT_BUILD_PLAN.md` under a new `### BENCH — Benchmarks (POST-v0.1)`
subsection, sibling to `M6 — Deferred batteries`:

```markdown
### BENCH — Reproducible benchmarks (POST-v0.1)

- **[BENCH-1] Reproducible framework benchmark suite + README performance
  results** — checked-in `benchmarks/` harness quantifying Gombit's
  abstraction cost (net/http → Gin → Huma+Gin → Gombit) and realistic
  PostgreSQL CRUD/auth performance against Gin+GORM (primary control) and
  Django+DRF/Rails/Laravel/NestJS (ecosystem context), plus a generated
  README `## Performance` section. AC: see issue #141 §25. deps: none
  (post-v0.1, no locked architecture change). size: XL. labels: `infra`,
  `devx`, `tests`, `ci`.
```

Do not add BENCH-2..N sub-IDs — the issue authorizes one entry. If a phase
below needs to survive as its own PR, it's tracked as a checklist item in
this plan document and cross-referenced to #141, not as a new backlog ID.

---

## 3. Non-goals (do not pull into this work)

Straight from issue #141 — the ones most likely to cause scope creep:

| Do not | Why |
| --- | --- |
| Tune Gombit specially or cripple competitors' defaults | Both directions falsify the result |
| Use SQLite for the cross-framework comparison | D12 databases are fine for framework tests; this benchmark requires PostgreSQL parity across all 6 apps |
| Add hard perf-regression gates to normal PR CI | Shared runners are too noisy; smoke-only per issue §11 |
| Require Redis/queues/mail/gRPC/multi-tenancy | M6 batteries stay out of v0.1 and out of this benchmark |
| Benchmark `/admin/` UI or frontend rendering in a browser | Explicitly excluded |
| Copy competitor numbers from framework marketing sites | Every published number must come from this repo's harness |
| Build Echo/Fiber/Buffalo/Encore implementations | Optional, post-BENCH-1, must not block the required 6 |
| Change any locked architecture decision (§1-§3 of the build plan) for benchmark convenience | Non-negotiable per AGENTS.md |

---

## 4. Key engineering decisions

The issue deliberately leaves these open ("pick one and document why"). Recommendations, to confirm before Phase 1 starts:

| Decision | Recommendation | Rationale |
| --- | --- | --- |
| Load generator | **k6** | Only one of {k6, wrk2, oha} with a built-in constant-arrival-rate executor (`ramping-arrival-rate`), so the coordinated-omission requirement (issue §"Load generator") is met natively instead of documented as a gap. JS scripting lets one workload script parametrize all 6 targets. Ships JSON summary export natively (`--summary-export`), feeding `results.json` directly. |
| PostgreSQL pin | `postgres:16.4-alpine`, digest captured into `metadata.json` at run time via `docker inspect` | Repo's existing CI convention is `postgres:16-alpine` (major-only); issue requires major.minor or digest. Bump precision without diverging from the CI convention's major version. |
| Go microbenchmark location | `benchmarks/micro/{nethttp,gin,huma,gombit}`, one package per row, sharing types/route registration/assertions via `benchmarks/micro/scenario`; `internal/contractspike`'s M0-2 spike stays untouched as a historical, standalone artifact, cross-linked but not extended | Matches the issue's own suggested tree and keeps one discoverable home for all benchmark code. "Reuse, not reimplement" is satisfied by cross-linking the spike's result, not by physically colocating new code in `internal/`; the spike's widget routes were never going to be literally extended anyway. One package per row also gets Huma's `contract.Install` process-isolation requirement (the Gombit row mutates process-global `huma.NewError`) for free from `go test`'s one-process-per-package model, instead of needing a special-cased package boundary for just that row. |
| Gombit/Gin+GORM benchmark *server* apps (real HTTP, not in-process) | Live in `benchmarks/apps/gombit` and `benchmarks/apps/gin-gorm` as `package main` inside the **root Go module** | They only need deps already in `go.sum` (Gin, Huma, GORM, `pgx`/`lib/pq`). Matches the existing precedent of `examples/` living in the root module and being covered by `go build ./...` in CI — no nested `go.mod` needed. |
| Django/Rails/Laravel/NestJS apps | Own directories under `benchmarks/apps/`, each with its ecosystem's standard lockfile and a documented production server (gunicorn/uvicorn, Puma, PHP-FPM+Nginx, compiled Nest + `NODE_ENV=production`) | Not Go — no module question; production-server choice is fixed per issue §17 and documented per app. |
| Orchestration language | Thin `bash` entrypoints (`scripts/run.sh`, `scripts/smoke.sh`) that shell out to a small Go CLI (`benchmarks/internal/...`, invoked via `go run`) for anything structured: metadata capture, result aggregation, schema/fairness checks, README generation | Issue explicitly forbids "one giant shell script." Go gives testable, typed result-schema code reusing the repo's existing tooling conventions (cf. `resourcegen`, `commandgen`); bash stays a thin process-orchestration layer only. |
| README integration mechanism | Option A — stable HTML comment markers (`<!-- benchmark-results:start/end -->`) regenerated in place | Single source file to read, no second "generated include" file to keep in sync; the regenerator is idempotent and the CI drift check is `git diff --exit-code README.md` after regenerating, mirroring the existing `contract-drift` job pattern in `ci.yml`. |
| Committed results | `benchmarks/results/latest/` (metadata.json, results.json, results.csv, summary.md) committed; per-run raw output under `benchmarks/results/runs/<timestamp>/` is a CI artifact, **not** committed | Keeps the repo from accumulating raw load-test logs while still satisfying "raw results retained" (retained as workflow artifacts + regenerable locally). |

Flag any of these you'd rather change before Phase 1 — they're recommendations, not commitments.

---

## 5. Repository structure

Adopts the issue's suggested tree with the module-placement decisions above:

```text
benchmarks/
├── README.md                  # how to run, how to add a framework
├── Makefile                   # benchmark-smoke / micro / crud / auth / techempower / footprint / benchmark / benchmark-report
├── compose.yml                # postgres + all 6 app services, resource limits
├── config/benchmark.env       # pinned versions, resource budgets, pool limits
├── micro/                      # landed (Phase 2) — Go framework-tax microbenchmarks
│   ├── scenario/                # shared resource types, Huma route registration, correctness assertions
│   ├── nethttp/                  # net/http row
│   ├── gin/                      # idiomatic plain-Gin row
│   ├── huma/                     # bare Huma-over-Gin row
│   └── gombit/                    # full framework.App row (own package: contract.Install isolation)
├── apps/
│   ├── gombit/                # root-module package main; also built via `gombit build --embed` for footprint
│   ├── gin-gorm/               # root-module package main; primary control
│   ├── django/
│   ├── rails/
│   ├── laravel/
│   └── nestjs/
├── workloads/                  # k6 scripts: plaintext, json, crud-read, crud-write, concurrency, techempower
├── scripts/                    # run.sh, smoke.sh, collect-host-info.sh -> Go helpers
├── internal/                   # Go: result schema, metadata collector, summarizer, README regenerator, fairness checks
├── results/
│   ├── README.md
│   └── latest/{metadata.json, raw/, results.json, results.csv, summary.md}
└── docs/methodology.md

internal/contractspike/         # existing M0 spike — historical, untouched, cross-linked only
docs/spikes/M0-2_HUMA_GIN_SPIKE.md   # preserved, cross-linked to benchmarks/README.md
.github/workflows/
├── ci.yml                      # + benchmark-smoke job
└── benchmarks.yml              # new: workflow_dispatch full suite
```

---

## 6. Phased execution (PR-sized slices under issue #141)

Each phase is independently reviewable and mergeable; later phases depend on
earlier ones landing. All phases keep `go test ./...`, lint, and the existing
SQLite/PostgreSQL/MySQL matrix green throughout (AGENTS.md §5.1).

### Phase 1 — Governance + scaffolding

- Add the BENCH-1 backlog entry (§2). — **done**
- Create the `benchmarks/` directory tree (§5); `micro/` landed in Phase 2,
  the rest (`apps/`, `workloads/`, `scripts/`, `config/`, `results/`,
  `docs/`) is still open.
- Implement the result schema + metadata collector in `benchmarks/internal/`
  (Go structs matching the issue's `results.json` shape; a `metadata.json`
  collector reading `git rev-parse`, `uname`, `go version`, `docker version`,
  `docker compose version`, CPU/RAM). — not yet done.
- Pin and document the load generator (k6) and PostgreSQL image (§4). — not
  yet done.
- **AC:** `go build ./benchmarks/...` succeeds; `docs/GOMBIT_BUILD_PLAN.md`
  has the new entry. Partially satisfied — the backlog entry and `micro/`
  build, but the result-schema/metadata-collector/load-generator work is
  still open.

### Phase 2 — Go abstraction-overhead microbenchmarks — **done**

- Added `benchmarks/micro/{nethttp,gin,huma,gombit}`, one package per row,
  each carrying the plaintext/JSON/path-parameter/valid-POST/invalid-POST
  scenarios via shared types and Huma route registration in
  `benchmarks/micro/scenario`. `internal/contractspike`'s M0-2 spike is
  untouched, not extended (see §4's Go-microbenchmark-location decision).
- Each row's `TestScenarios` structurally decodes and checks responses
  (`benchmarks/micro/scenario/assert.go`), not substring matching, so a
  handler that stops doing real JSON work can't silently keep passing.
- Cross-linked `docs/spikes/M0-2_HUMA_GIN_SPIKE.md` to
  `benchmarks/README.md`; noted the spike's result is historical, not the
  current number.
- **Post-landing correction, round 1 (external review on PR #174, HEAD
  `3090c5f`):** the first landed version had two measurement defects serious
  enough that the "done" stamp was premature.
  - The timed loop called `httptest.NewRequest`/`NewRecorder` every
    iteration for every scenario. On a trivial handler that overhead
    (~5.3KB / 12 allocs) dwarfed the handler's own cost (~1KB / 10 allocs),
    and — being roughly constant across all four stacks — compressed the
    reported ratios. GET scenarios were fixed to build their request once
    outside the `b.N` loop. POST scenarios were left building one per
    iteration, on the claim that a drained body reader can't be reused —
    **that claim was itself wrong** and got caught in round 2 below.
  - The bare Huma+Gin row wrapped responses in the D10 `SuccessEnvelope`,
    which is a Gombit convention Huma does not apply by default. That billed
    part of Gombit's own cost to the Huma+Gin baseline, understating the
    delta the huma/gombit comparison exists to isolate. `scenario.go` gained
    two registrations — `RegisterRoutes` (bare, used by
    `benchmarks/micro/huma`) and `RegisterEnvelopedRoutes` (D10, used by
    `benchmarks/micro/gombit`) — verified to produce different wire shapes.
  - Also tightened `assertInvalidPost` from "any 4xx" to exactly 422 (a 404
    from a broken route registration was passing the same check a real
    validation rejection was), now that per-package process isolation makes
    an exact status assertion safe.
- **Post-landing correction, round 2 (external review on PR #174, HEAD
  `1677063`):** round 1's POST reasoning was wrong, not just incomplete. A
  *drained reader* can't be reused, but the *`*http.Request`* holding it can
  — only `Body` and `ContentLength` need replacing per iteration
  (`request.Body = io.NopCloser(bytes.NewReader(payload))`), which is what
  `runGET` already did for its own per-iteration state. Verified safe with a
  30-iteration reuse loop against a full `framework.App` (Gombit's XSS
  middleware, the only thing in the stack that touches `Body`/`Content-Length`
  in place, operates on a `WithContext`-derived copy per request, not the
  shared object). Verified effect: `net-http`/valid-post went from 7563 B/op
  (32 allocs) to 2097 B/op (21 allocs), matching the review's own
  independently-measured prediction (2080 B/op, 21 allocs) almost exactly;
  the gombit/net-http ratio for that scenario went from a confounded ~1.7x to
  ~3.8x. Also fixed two stale comments the round-1 fix left behind (the
  `Envelope` field doc still said "Huma, Gombit"; `assertInvalidPost`'s
  comment attributed bare Huma's 422 to Gombit's `contract` package, which
  bare Huma+Gin never calls — verified against Huma's own source, which
  hardcodes 422 for validation failures independently of Gombit), and
  tightened every scenario's benchmark-loop status guard (not just the
  correctness assertions) to the exact expected status, so a routing
  regression can't silently get timed as if it were the scenario it broke.
- **Deferred to a later Phase 1/2 follow-up:** `benchstat`-based
  `-count=10` summarization is documented (run manually) but not yet wired
  into a `make benchmark-micro` target or `results.json` output — that
  depends on Phase 1's still-open result-schema/summarizer work.
- **AC:** `go test ./benchmarks/micro/... -bench=BenchmarkFrameworkTax -benchmem -count=10`
  runs all four stacks × five scenarios and reports `ns/op`/`B/op`/`allocs/op`
  for the handler, not the benchmark harness — true for all five scenarios,
  not just the three GET ones, as of round 2 above.
  `make benchmark-micro` / `results.json` output is not yet satisfied (see
  above).

### Phase 3 — Canonical schema, seed, and CRUD apps (Gombit + Gin+GORM)

**Phase 3a — schema + seed + Gin+GORM control — done.**

- Canonical schema documented at
  [benchmarks/docs/schema.md](../../benchmarks/docs/schema.md): `users`/
  `projects` DDL, the five canonical routes, the D10-envelope-for-both-apps
  decision (deliberately different from `benchmarks/micro/scenario`'s
  per-stack-idiomatic choice — see that doc for why), response field list,
  and the deterministic-seed spec (1,000 users / 100,000 projects — the
  issue's recommended size worked fine, no need to shrink it).
- `benchmarks/apps/shared`: `PageMeta` (`page`/`limit`/`total` — not
  `contract.PageMeta`, which uses `per_page`; issue #141 specifies `limit`)
  and `ProjectData`, shared by every Go implementation so "same JSON shape"
  (issue §15) is true by construction between them, not by two
  hand-maintained struct definitions staying in sync.
- `benchmarks/apps/gin-gorm`: the primary framework-tax control (issue
  "isolates the incremental cost of adopting Gombit rather than changing
  programming languages"). GORM `AutoMigrate` (not Atlas — this is the
  control's own idiomatic migration mechanism); `.Preload("Owner")` on the
  list query, verified against real Postgres statement logs to issue
  exactly 3 SQL statements per list request (count + page + one batched
  owner `IN (...)`, not one owner query per row) before writing that as an
  automated regression guard (`TestListDoesNotN1`, gated behind `-tags
  integration` per `database/integration_test.go`'s existing convention).
  `-seed` flag truncates and reseeds deterministically. 20/20 connection
  pool limits per issue "Connection pooling" (overriding Gombit's own
  25/5 Postgres default — the fairness pin is the issue's, not either
  framework's).
- `benchmarks/compose.yml`: pinned `postgres:16.4-alpine`, resource-limit
  block (documented as best-effort outside Swarm, per issue's own
  "detect/report rather than silently pretend" requirement).
- Schema verified column-by-column against a real, running Postgres
  instance (`psql \d`), not assumed from the model definitions — caught and
  fixed a real gap this way (`Project.Name`/timestamp columns were
  nullable; the schema doc requires `NOT NULL`).

**Post-landing correction, round 1 (PR review on #176, github.com/gombit-dev/gombit/pull/176#pullrequestreview-5017796308):**
Phase 3a's first landed version had a real specification contradiction and
several fairness/coverage gaps; all fixed:

- `benchmarks/docs/schema.md` claimed the list endpoint issues 2 SQL
  statements; the handler and `TestListDoesNotN1` both said 3 (`COUNT` +
  page + batched owner `IN (...)`) — the document future implementations
  are supposed to converge on disagreed with the implementation it was
  documenting. Corrected to pin 3 for a non-empty page (and 2 for an empty
  one, now covered by `TestListDoesNotN1EmptyPage`), with the reasoning for
  why an empty page differs spelled out so it reads as a documented
  boundary, not an inconsistency.
- `POST /api/projects` with a nonexistent `owner_id` hit the FK constraint
  and returned 500 `internal`, not a 4xx — a bad client-supplied reference
  is invalid input (issue §15), not a server fault. Fixed
  (`TestCreateRejectsInvalidOwnerID`) by detecting the Postgres SQLSTATE
  (`23503`) directly rather than through `database.MapPersistError`, which
  only special-cased unique violations.
- The control app imported `github.com/gombit-dev/gombit/contract` for its
  D10 envelope types, and `contract.ErrorEnvelope` is a `huma.StatusError`
  — so the "no Huma" control linked Huma in its dependency graph even
  though no handler touched it directly. Fixed by adding framework-free
  `Data`/`DataMeta`/`ErrorEnvelope` types (same JSON shape, zero Huma/GORM-
  helper dependency) to `benchmarks/apps/shared`; verified with
  `go list -deps` that `contract`/`database`/`huma` no longer appear in
  `benchmarks/apps/gin-gorm`'s dependency graph at all. This also fixed the
  FK bug above at the root: `database.MapPersistError` was never the right
  tool for a table with no unique business key.
- The seed contract (deterministic 1,000/100,000 content, round-robin
  ownership, truncate+`RESTART IDENTITY` idempotency) had no automated
  coverage — `seedDatabase` was only exercised manually. Fixed by
  extracting the pure content formulas into their own functions (tested
  without a database or build tag: `TestSeedContentIsDeterministic`,
  `TestProjectOwnerIDRoundRobin`) and parameterizing the truncate-then-seed
  path (`seedDatabaseN`) so an integration test can run the real path twice
  at reduced scale and assert it doesn't accumulate duplicate data
  (`TestSeedDatabaseNIsIdempotentAndCorrect`).
- None of the above had CI coverage: every test in the package was
  `//go:build integration`, and no CI job passed that tag for this
  package — a green PR was not evidence any of this worked. Fixed by
  adding a step to `.github/workflows/ci.yml`'s existing `database-postgres`
  job. Uses a **separate** `gombit_bench` database on that job's Postgres
  service, not the `gombit` database `auth`/`database`/`admin` already use:
  `auth.User` also maps to a table named `users` (no `TableName()`
  override), with different columns than `gin-gorm`'s `User` — sharing a
  database would let one `AutoMigrate` alter the schema the other relies
  on. Validated the exact CI command (`psql -c 'CREATE DATABASE ...'` then
  `go test -tags integration ...`) locally against a freshly created
  database before trusting the YAML, not just written and assumed correct.

**Post-landing correction, round 2 (Merge Warden review on #176,
github.com/gombit-dev/gombit/pull/176#pullrequestreview-5018114825):** one
claim checked and found not applicable, three real gaps fixed:

- Claimed `TestListDoesNotN1`'s exact-count assertions were unreliable
  because `gorm.Open` issues initialization queries the counting logger
  would pick up. Checked directly against this driver/version (`gorm.Open`,
  then a forced `Ping()`, with a throwaway counting logger attached) — zero
  queries traced either way, so the specific failure mode doesn't reproduce
  here. Switched both counting tests to `db.Session(&gorm.Session{Logger:
  counter})` on the already-open connection anyway: strictly more robust
  regardless (no first-connection cost of any kind, present or future
  driver behavior), and it was the suggested fix, so there was no reason
  not to adopt the better pattern even without a reproducing bug.
- `updateProjectRequest.Name`'s `binding:"omitempty,max=255"` skips
  `max=255` once the pointed-to value is empty, not only when the pointer
  is nil — checked directly (a standalone `omitempty,max=255` validator
  call against `&""`) and confirmed `{"name":""}` passes binding
  unchanged, silently bypassing the same non-blank-name rule `POST
  /api/projects` enforces via `required`. Fixed with an explicit check in
  the handler (`TestUpdateRejectsBlankName`, including that a rejected
  PATCH doesn't partially apply).
- `queryCounter` read/wrote its fields with no synchronization. Not
  currently racing (every caller drives it from one goroutine per test),
  but GORM doesn't guarantee `Trace` runs on the caller's goroutine, so
  this was correct-by-accident, not correct-by-construction. Added a
  mutex and thread-safe `Count()`/`Queries()` accessors; verified the full
  suite still passes under `-race`.
- `updated.UpdatedAt.After(created.UpdatedAt)` asserted a stronger
  invariant (strictly later) than the one that actually matters (not
  earlier), risking flakiness if two round trips ever land in the same
  timestamp-resolution window. Changed to `!updated.UpdatedAt.Before(...)`.

**Post-landing correction, round 3 (PR review on #176,
github.com/gombit-dev/gombit/pull/176#pullrequestreview-5022158018):** round
2's blank-name fix was itself incomplete — applied to `update` only, not
`create`, which taught the resource two different name contracts depending
on which verb touched it. `create`'s `binding:"required,max=255"` rejects
the empty string but not a whitespace-only one; verified directly against
live Postgres before fixing: `POST {"owner_id":1,"name":"   "}` returned
`201 Created` with the name stored as three spaces, while the identical
value already failed on `PATCH`. Fixed by extracting `blankNameError` as a
shared check both handlers call — the point being that a rule expressed
identically in two places, the way the previous round's inline
`strings.TrimSpace` check in `update` alone was, is exactly how this class
of asymmetry gets introduced in the first place; a rule that can't drift
between call sites because there's only one call site is the actual fix,
not just symmetric coverage today. `TestCreateRejectsBlankName` and
`TestUpdateRejectsBlankName` now share one `blankNames = []string{"",
"   "}` table so the same asymmetry can't quietly return through one test
being updated and the other not.

**Post-landing correction, round 4 (Merge Warden review on #176,
github.com/gombit-dev/gombit/pull/176#pullrequestreview-5024765519):**
claimed, not applicable — `Project.Owner` being a value `User` rather than
`*User` would supposedly make `db.Save(&row)` attempt to upsert a zero-value
`User` whenever a `Project` is saved without `.Preload("Owner")` first
(exactly what `create` and `update` both do). Checked three ways before
concluding otherwise, not just reasoned about:

- Source: `gorm@v1.31.1/callbacks/associations.go`'s `SaveBeforeAssociations`,
  struct-kind branch — `if _, zero := rel.Field.ValueOf(ctx, reflectValue);
  !zero { ... }` — explicitly skips saving a belongs-to association when
  the field is GORM's own notion of the zero value, before any SQL is
  built.
- Empirical, the exact `update()` path: loaded a real `Project` via
  `First` (no `Preload`, `Owner` left zero-value), mutated it, called
  `Save`, with GORM's own verbose logger attached. One `UPDATE projects`
  statement; no statement against `users` at all; `SELECT count(*) FROM
  users` unchanged before/after; zero rows with `email = ''`.
- Empirical, the exact `create()` path: built a `Project{OwnerID: 1, ...}`
  with `Owner` left zero-value, called `Create`, same logger. One `INSERT
  INTO projects` statement; same unchanged user-count and empty-email
  checks.

Both handlers' actual call patterns are covered by this, not just a
similar-looking synthetic case. Not changing `Owner` to `*User`: doing so
isn't free the way the round-2 `db.Session` swap was — every read site
(`toProjectData`) would need a nil guard and a decision about what
`OwnerName` means for an un-preloaded project, for a bug that doesn't
exist on this GORM version against this exact code. `owner_id` is also a
`NOT NULL` foreign key (`benchmarks/docs/schema.md`) — every `Project`
genuinely always has an owner once preloaded, so a non-pointer association
isn't a modeling mismatch either, just an ORM-level "not loaded yet" state
a pointer wouldn't describe any more precisely.

**Post-landing correction, round 5 (Merge Warden review on #176,
github.com/gombit-dev/gombit/pull/176#pullrequestreview-5024924257):**
false positive, no functional bug — the review claimed
`database.MapPersistError` was still mapping FK violations to 500 and
should be fixed in `github.com/gombit-dev/gombit/database`. That framework
function isn't called anywhere in `benchmarks/apps/gin-gorm` — round 1
already replaced it with a local, independent `mapPersistError` that
handles the FK case correctly (`shared.ValidationError`, 422), verified
repeatedly since. The finding's own diff hunk showed why: it's anchored
right where `mapLoadError`/`mapPersistError`'s doc comments *mention*
`database.MapLoadError`/`database.MapPersistError` by name — explaining
what's deliberately *not* used and why — which a lightweight scanner (or a
human skimming fast) can misread as describing what the code still calls.
Re-verified before concluding it was a false positive, not just assumed:
`grep` for `gombit-dev/gombit/database` in the package matches only inside
those two comments, `go list -deps` confirms `contract`/`database`/`huma`
are absent from the package's dependency graph, and
`TestCreateRejectsInvalidOwnerID` still passes against real Postgres, 422
not 500. No code-behavior change; reworded both comments to lead with
"local, standalone implementation, not a call to \[framework function\]"
instead of naming the framework function first — the ambiguity was real
even though the bug wasn't, and it's a one-time fix to stop tripping the
same misread again.

**Phase 3b — the `gombit` app and cross-implementation fairness checks — done
(CI wiring for the fairness check itself deferred to Phase 8; see below).**

- `benchmarks/apps/gombit/internal/project`: the canonical API as a normal
  Gombit app — Huma handlers, `framework.App`, GORM, using
  `benchmarks/apps/shared`'s response types for the success envelope and
  Gombit's own `contract`/`database` packages for error mapping,
  unmodified (issue "do not bypass ... normal Gombit response handling").
- Real Atlas migration (`gombit db makemigrations`/`migrate`, AGENTS.md D3
  — not `AutoMigrate`), committed under
  `benchmarks/apps/gombit/database/migrations/`. Verified the generated
  DDL is structurally identical to `gin-gorm`'s `AutoMigrate` output down
  to the auto-generated index names (`idx_users_email`,
  `idx_projects_owner_id`, `idx_projects_created_at`,
  `fk_projects_owner`) — both use the same `gormschema` code underneath,
  just through different appliers.
- **Two real bugs caught by testing against the live control, not just
  against the spec:**
  - Huma defaults `POST` to 200; `gin-gorm` returns 201. Fixed with
    `huma.Operation.DefaultStatus`.
  - Gombit's own `API.Prefix` default is `/api/v1`; the canonical route
    spec and `gin-gorm` both use `/api` with no version segment. Left at
    the framework default, this app's route surface would not have
    matched its own control's. Fixed by setting `cfg.API.Prefix = "/api"`
    explicitly in `main.go`.
- **One discovered, deliberately unpatched framework gap:**
  `database.MapPersistError` only special-cases unique-constraint
  violations, so `POST /api/projects` with a nonexistent `owner_id` (a
  foreign-key violation) falls through to 500 `internal`, unlike
  `gin-gorm`'s 422. Not fixed here — issue #141 requires using Gombit's
  normal public APIs as-is, and this app's whole point is measuring what a
  real Gombit user gets today, warts included. Pinned by
  `TestCreateInvalidOwnerIDReturnsInternalError` so a future framework fix
  (or regression) shows up as an expected test change; see
  `benchmarks/apps/gombit/README.md` for the full writeup. Fixing
  `database.MapPersistError` is out of scope for BENCH-1 — worth its own
  follow-up issue against the `database` package.
- `internal/project`'s own integration suite (`TestCRUDRoundTrip`,
  blank-name rejection on create and update, pagination/ordering, the
  3-query/2-query N+1 guard via mutating `app.DB().Logger` — `gorm.DB`
  embeds `*Config` by pointer, so this affects every query the app
  instance issues without needing a second `database.DB`) mirrors
  `gin-gorm`'s suite test-for-test, gated behind `-tags integration` the
  same way, wired into the same CI job against a third database
  (`gombit_bench_gombit`, since this app's Atlas-migrated tables shouldn't
  share a database with `gin-gorm`'s `AutoMigrate`'d ones either).
  Verified the full CI recipe (create database, install Atlas, `gombit db
  migrate`, run tests) locally against a freshly created database, not
  just written and assumed correct.
- `benchmarks/apps/fairness_test.go`: `TestCrossImplementationFairness`
  builds and runs both real binaries (subprocess + HTTP, not shared Go
  internals — gin-gorm is `package main`, and this is also the only shape
  of comparison that will still work once Phase 4 adds
  Django/Rails/Laravel/NestJS) against their own already-seeded databases
  and checks: identical paginated content and ordering for the canonical
  seed (timestamps excluded — each binary was seeded at a different real
  time), and identical 404 behavior for a nonexistent id. Passes,
  verified twice for flakiness, with confirmed clean subprocess teardown.
  **Not yet wired into routine PR CI**: it needs both databases seeded at
  the full canonical 1,000/100,000 scale, which is heavier than what
  belongs in a PR smoke job — needs the lighter seed-size mechanism
  Phase 8 (CI integration, smoke vs. full) already anticipates. Verified
  locally in this phase instead.
- **AC:** `make benchmark-crud` runs Gombit and Gin+GORM against the same
  Postgres instance and produces comparable `results.json` rows; fairness
  tests fail loudly on route/shape/query-count divergence. The fairness
  check itself is done and passing; `make benchmark-crud` and
  `results.json` still depend on Phase 1's open result-schema/summarizer
  work and are not yet satisfied.
- `benchmarks/compose.yml` app services for both apps — still open, moved
  to a follow-up alongside Phase 8's CI work.

**Post-landing correction (review on PR #177, github.com/gombit-dev/gombit/pull/177#pullrequestreview-5026216966):**
one claim checked and found incorrect, three real gaps fixed:

- Claimed the child-process env override in `startApp` (`cmd.Env =
  append(cmd.Environ(), "DATABASE_URL="+dsn, ...)`) loses to a
  pre-existing `DATABASE_URL` in the parent's environment, because
  `os.Getenv` returns the first match. Checked at the source rather than
  argued about: `syscall`'s `copyenv()` does document first-occurrence-wins
  for a process's *own* already-received environ — but `os/exec.Cmd.Start`
  calls `dedupEnv` before ever calling `execve`, and that function's own
  comment says "Construct the output in reverse order, to preserve the
  *last* occurrence of each key." Verified directly with a real child Go
  binary reading via `os.Getenv` while the parent had `DATABASE_URL` set to
  a different value — the appended override won, every time. No code
  change; added a comment at the call site citing the exact source so the
  same claim doesn't recur.
- `TestCrossImplementationFairness` compared the two implementations'
  pages to each other but never against the actual canonical dataset —
  two empty, unseeded (but migrated) databases would return matching
  `{total:0, data:[]}` on both sides and pass. Confirmed by literally
  reproducing it: pointed the test at two fresh, Atlas-migrated-but-never-seeded
  databases before the fix (would have passed) and after (fails with
  `meta.total = 0, want 100000`). Fixed by adding `assertCanonicalSeed`,
  checking each side independently against `shared.SeedProjectCount`,
  `shared.ProjectName`, `shared.ProjectOwnerID`, and page size *before*
  the relative comparison runs.
- `createProjectBody.OwnerID` had no lower bound, so `{"owner_id":0,...}`
  — a present, well-typed field, not the "nonexistent id" case
  `TestCreateInvalidOwnerIDReturnsInternalError` documents — passed Huma
  validation and hit the same FK-violation 500. gin-gorm's
  `binding:"required"` already rejects this input (Gin's `required`
  treats a non-pointer `uint` zero value as absent), so this was a real,
  fixable asymmetry, not another instance of the discovered
  `database.MapPersistError` gap. Fixed with `minimum:"1"` on `OwnerID`;
  verified live that `owner_id:0` now 422s while `owner_id:999999` still
  500s — two different inputs, two different (and now each individually
  correct) outcomes. Added `TestCreateRejectsZeroOwnerID`.
- `benchmarks/apps/gombit`'s `seedDatabaseN` was copied from `gin-gorm`
  without the idempotency test an earlier review round added specifically
  because the seed contract had no automated coverage
  (`TestSeedDatabaseNIsIdempotentAndCorrect`). Added the same test for
  `gombit` (`benchmarks/apps/gombit/main_test.go`), verified passing
  against real Postgres.
- `benchmarks/apps/gin-gorm/README.md` still described Phase 3b (the
  `gombit` app, the fairness check) as future work, and its Test section
  still pointed at `TestSeedContentIsDeterministic`/
  `TestProjectOwnerIDRoundRobin` as if they were still gin-gorm's own —
  they moved to `shared` in this same PR. Updated both.

**Post-landing correction, round 2 (review on PR #177,
github.com/gombit-dev/gombit/pull/177#pullrequestreview-5026402067):** one
real, blocking gap, confirmed by reproducing the failure rather than taking
the claim on faith.

- Claim: `go test -tags integration ./benchmarks/apps/gombit/...` expands to
  two packages — `benchmarks/apps/gombit` (`main_test.go`, added in round 1)
  and `benchmarks/apps/gombit/internal/project` (`handler_test.go`) — each
  compiled to its own test binary, and both `TRUNCATE TABLE projects, users
  RESTART IDENTITY CASCADE` at the start of every test against the same
  `gombit_bench_gombit` database. `go test` runs different packages' test
  binaries in parallel by default (bounded by `-p`, which defaults to
  `GOMAXPROCS`), so one package's `TRUNCATE` can land mid-assertion in the
  other. CI being green doesn't rule this out — it's a race, not a
  deterministic failure.
- Verified true two ways. First, built both packages' test binaries
  separately (`go test -tags integration -c`) and ran them concurrently
  against one throwaway, freshly Atlas-migrated database: 15/15 runs failed,
  every time on the expected symptom (`TestSeedDatabaseNIsIdempotentAndCorrect`
  or `TestCRUDRoundTrip` seeing row counts/content from the other package's
  concurrent truncate). Second — to rule out an artifact of manually racing
  the binaries — ran the actual documented command,
  `go test -tags integration ./benchmarks/apps/gombit/...`, 25 times
  unmodified against a fresh throwaway database: 2/25 failed with the same
  symptom, confirming it reproduces (intermittently, as expected of a race)
  through `go test`'s own scheduling, not just under contrived concurrency.
- Fixed with the reviewer's own "cheapest, honest" option: added `-p 1` to
  the CI step (`.github/workflows/ci.yml`), the `benchmarks/apps/gombit`
  README recipe, and `internal/project/handler_test.go`'s doc-comment
  recipe, each with a comment explaining why it's load-bearing rather than
  a style choice. Chose this over moving the seed test into
  `internal/project` (the reviewer's "better" alternative) because
  `seedDatabaseN` is unexported in `package main` alongside `main.go`,
  mirroring where a real generated Gombit app's seed command would live —
  moving it would misrepresent that layout, and `internal/project` can't
  import `package main` to reuse it without a cycle in the wrong direction.
  A third database (the reviewer's "heavier than the bug" fallback) was not
  needed. Verified the fix at the same rigor as the failure: 25/25 passes
  with `-p 1` added to the identical command that failed 2/25 times without
  it, against the same throwaway database.

### Phase 4 — Remaining competitor apps (Django, Rails, Laravel, NestJS)

- One sub-slice per framework (can be up to 4 separate PRs if easier to
  review): same schema/seed/routes/pagination contract as Phase 3, each in
  its documented production configuration (issue §17), each with pinned
  lockfiles.
- Extend the fairness checks to cover all 6 implementations.
- **AC:** all 6 apps pass the same fairness suite; `docker compose` brings
  all 6 up against one PostgreSQL instance with documented resource limits.

**Phase 4a — Django + DRF — done (fairness-check extension and compose/CI
app service deferred, same as Phase 3a/3b's own still-open items).**

- `benchmarks/apps/django`: the canonical API idiomatically in Django +
  Django REST Framework — Python 3.12, Django 5.2.17 (current LTS),
  djangorestframework 3.18.0, psycopg 3.3.4 (with the `pool` extra),
  gunicorn 26.2.0, all pinned exact (issue §16/§17). D10 envelope and error
  mapping reimplemented by hand in `projects/envelope.py` (this app can't
  import `benchmarks/apps/shared`, which is Go-only) — same shape as
  `gin-gorm`'s own independent reimplementation, verified against
  `benchmarks/docs/schema.md`, not against the Go source.
- Schema verified column-by-column against real PostgreSQL (`psql \d`), not
  assumed from Django's defaults: `users.email`/`name` and `projects.name`
  needed `models.TextField`, not Django's usual `CharField`/`EmailField` —
  Django's default is `VARCHAR(n)`, but the canonical schema (and what
  GORM's plain `string` actually generates for `gin-gorm`/`gombit`) is
  unbounded `TEXT`. `Project.owner` needed `on_delete=DO_NOTHING` and
  `db_index=False` (with an explicit named `Meta.indexes` entry instead) —
  Django's own defaults (`CASCADE`, an automatic same-column index
  alongside a hand-added one) would have both been real, silent
  divergences from the canonical `ON DELETE NO ACTION` FK and produced a
  redundant duplicate index.
- **One discovered Django-specific correctness issue, found by testing
  against a live database rather than trusting the obvious code:** Django's
  PostgreSQL backend always emits foreign keys as `DEFERRABLE INITIALLY
  DEFERRED` (a fixed backend-level choice, not a per-field option — unlike
  `gin-gorm`/`gombit`'s plain, immediately-checked FK). Outside any wrapping
  transaction this is invisible, but inside one — this app's own test suite
  (`TestCase` wraps every test in one) or a production deployment with
  `ATOMIC_REQUESTS=True` — a foreign-key violation does not raise at
  `Project.objects.create()`/`.save()` at all; the write silently "succeeds"
  until the wrapping transaction commits, and the immediate
  `select_related(...).get(...)` reload right after raises
  `Project.DoesNotExist` instead (the joined row is invisible until the
  constraint is actually enforced) — the wrong exception type, uncaught,
  producing an unhandled 500 for what should be a 422. Verified by
  reproducing it (removing the fix reproduces exactly that `DoesNotExist`
  under `TestCase`) and by fixing it two ways at once:
  `views._write_project` wraps every create/update in its own
  `transaction.atomic()` block and calls `connection.check_constraints()`
  before that block exits, forcing the deferred check immediately in an
  isolated savepoint. Verified fixed both under the test suite (which runs
  inside a wrapping transaction) and live against gunicorn (autocommit, no
  wrapping transaction) — confirming the fix isn't just papering over a
  test-only symptom. Full writeup in `benchmarks/apps/django/README.md`
  "Discovered Django-specific issue."
- **One deliberate behavioral choice, unlike `benchmarks/apps/gombit`:**
  `owner_id:0` and a nonexistent `owner_id` both correctly reject as 422
  from the start (`min_value=1` in the serializer; the FK violation mapped
  by SQLSTATE via `_map_integrity_error`, the same policy as `gin-gorm`'s
  `mapPersistError`) — this is a from-scratch app, not a framework
  exercising Gombit's own real (and deliberately unpatched, see Phase 3b)
  gap, so there's no reason to reproduce a bug that only exists because of
  how Gombit's `database.MapPersistError` works today.
- The list endpoint's N+1 guard is **2 queries** (`COUNT` + one
  `select_related("owner")` JOIN), not `gin-gorm`/`gombit`'s pinned 3 —
  `benchmarks/docs/schema.md` explicitly allows a different fixed-count
  eager-load strategy as long as it's documented rather than silently
  claimed to match; documented in `benchmarks/apps/django/README.md`
  "Schema and query-shape notes."
- `django.contrib.auth`/`django.contrib.contenttypes` needed to be in
  `INSTALLED_APPS` even though this app has no login routes: DRF's
  `request.user` handling unconditionally imports `django.contrib.auth.models`,
  which raises at import time otherwise. `DEFAULT_AUTHENTICATION_CLASSES`/
  `DEFAULT_PERMISSION_CLASSES` are set to `[]`/`AllowAny` so `/api/projects`
  itself stays unauthenticated, matching every other implementation (no
  cross-framework auth comparison on the CRUD apps).
- Connection pooling (issue §18, 20 max open/idle): gunicorn's pre-fork
  model has no single global pool the way a single Go binary does, so
  `POOL_MAX_OPEN` is divided by `GUNICORN_WORKERS` and each worker's
  psycopg pool is fixed at that per-worker size (`min_size == max_size`) —
  4 workers × 5 connections = 20 total, documented in
  `benchmarks/apps/django/README.md` as required non-default tuning (issue
  §17).
- 13-test suite (`projects/tests.py`, `manage.py test`) mirrors `gin-gorm`/
  `gombit`'s contract test-for-test: CRUD round trip, blank-name rejection
  on create and update, zero/nonexistent `owner_id` rejection, 404 for both
  a nonexistent and a non-numeric id, pagination/ordering, the N+1 query
  count guard (`assertNumQueries`), the seed content formulas' own
  determinism (ported by hand from `benchmarks/apps/shared`, verified
  independently rather than trusted), and the seed contract's DB-backed
  idempotency check run twice at reduced scale. One test-infrastructure-only
  wrinkle found and fixed along the way: `TestCase` isolates rows via a
  rolled-back transaction, but Postgres sequences are not transactional, so
  a list-test fixture assuming freshly-created users get pks 1..N broke
  once an earlier test's sequence usage leaked forward — fixed by
  round-robining over the actual pks `bulk_create` returns instead of an
  assumed range; and the seed-idempotency test needed `TransactionTestCase`
  instead of plain `TestCase` since two `TRUNCATE`s inside one wrapping
  transaction hit the same deferred-FK "pending trigger events" restriction
  Postgres itself imposes.
- Verified against real PostgreSQL throughout, not just the test suite:
  full CRUD flow via live `curl` against gunicorn (single- and
  4-worker configurations), the production-scale seed command
  (1,000/100,000 rows, ~4.4s, idempotent — re-run and row counts confirmed
  unchanged), and the schema itself (`psql \d users`, `\d projects`)
  against `benchmarks/docs/schema.md` column-by-column.
- CI: added a Python 3.12 setup + `pip install` + `manage.py test` step to
  the existing `database-postgres` job, targeting a fourth database
  (`gombit_bench_django`) — no separate "create database" step needed
  first, unlike `gin-gorm`/`gombit`: Django's own test runner creates and
  destroys its own throwaway `test_gombit_bench_django`, verified locally
  by pointing `DATABASE_URL` at a database that doesn't exist yet and
  confirming the suite still passes unmodified.
- **Deferred, same pattern as Phase 3a/3b's own still-open items:** a
  `Dockerfile`/`benchmarks/compose.yml` app service (the Go apps don't have
  one yet either), and extending `benchmarks/apps/fairness_test.go` to
  include this app as a third leg — the Phase 3a→3b precedent (gin-gorm
  landed alone before the fairness check existed at all) supports doing
  this as its own follow-up rather than growing this PR further.

**Post-landing correction (review on PR #178,
github.com/gombit-dev/gombit/pull/178#pullrequestreview-5026651450):** one
architectural finding accepted and fixed at the root rather than patched
around, three real gaps closed, all confirmed true before fixing.

- **Accepted and fixed at the root, not patched around:** the first version
  of `_write_project` (wrapping every create/update in `transaction.atomic()`
  + `connection.check_constraints()` to force Django's deferred FK check
  immediately) was a `TestCase`-convenience workaround left on the
  production write path issue #141 §17 actually benchmarks — an extra
  `BEGIN`/`SET CONSTRAINTS ALL IMMEDIATE`/`COMMIT` on every `POST`/`PATCH`
  that gunicorn's real per-request autocommit never needed. Fixed by making
  the schema match the canonical contract instead:
  `projects/migrations/0002_owner_fk_not_deferrable.py` drops Django's
  auto-generated `DEFERRABLE INITIALLY DEFERRED` constraint and recreates it
  `NOT DEFERRABLE INITIALLY IMMEDIATE`, matching `benchmarks/docs/schema.md`
  exactly (no `DEFERRABLE` clause — Postgres's own default meaning
  immediate). `_write_project` removed entirely; `views.py` now does plain
  `Project.objects.create()`/`.save()`, the same shape as `gin-gorm`'s own
  handlers. Verified: `psql \d projects` shows no `DEFERRABLE` clause, and
  `test_create_rejects_nonexistent_owner_id` passes unmodified under
  `TestCase`'s wrapping transaction. This incidentally also let
  `SeedDatabaseIsIdempotentAndCorrectTests` revert from `TransactionTestCase`
  back to plain `TestCase` (verified both pass post-fix) — the same
  deferred constraint had been the root cause of both symptoms.
- `description` had no `trim_whitespace=False` on either serializer, unlike
  `name` (which had it with an explicit comment stating the invariant) — a
  client-supplied `"  hello  "` was silently stored as `"hello"`, diverging
  from `gin-gorm`/`gombit`, neither of which trims. Fixed on both
  `CreateProjectSerializer` and `UpdateProjectSerializer`; verified live
  (gunicorn) and via a new
  `test_create_and_update_preserve_description_whitespace`.
- `envelope.exception_handler` relabeled DRF's own rejected-request bodies
  (malformed JSON, etc.) with a D10 `code` but kept DRF's native status —
  a `ParseError` (native 400) became `{"code":"validation_error"}` at
  status 400, not D10's fixed 422 for that category, so
  `POST {"` and `gin-gorm`'s equivalent `ShouldBindJSON` failure (422)
  didn't actually match despite both claiming `validation_error`. Fixed by
  bucketing DRF's native status into a D10 category first, then always
  returning *that category's* fixed status, never DRF's native one.
  Verified live: malformed JSON now 422, not 400.
- The query-count deviation (2 queries via a JOIN, not `gin-gorm`'s pinned
  3) was documented only in `benchmarks/apps/django/README.md`;
  `benchmarks/docs/schema.md` itself — the file its own text says to
  document a differing count in — didn't mention it, so a future PR
  extending `fairness_test.go` from the canonical doc alone would have
  "discovered" this as a surprise. Added to `schema.md` directly. Also
  fixed a stale claim in the app's own README ("CI wiring... still open")
  that this same PR had already closed.
- Test suite asserted status codes but not D10 `error.code`, so an
  implementation could return the right status with the wrong code (or vice
  versa) and still pass — exactly the gap that let finding 3 above ship.
  Added `error.code` assertions to every rejection test
  (`test_create_rejects_blank_name`, `_zero_owner_id`,
  `_nonexistent_owner_id`, `test_get_nonexistent_id_is_not_found`,
  `test_get_non_numeric_id_is_not_found`), plus new
  `test_create_rejects_malformed_json` and
  `test_create_and_update_preserve_description_whitespace` covering findings
  2 and 3 directly. 15 tests total, all passing against real Postgres.

**Phase 4b — Rails + ActiveRecord — done (fairness-check extension and
compose/Docker app service deferred, same as every prior sub-slice's own
still-open items).**

- `benchmarks/apps/rails`: the canonical API idiomatically in Rails +
  ActiveRecord — Ruby 3.3.12, Rails 8.1.3.1, pg 1.6.3, puma 8.0.2, all
  pinned exact (issue §16/§17). Host had no Ruby installed at all; developed
  and tested via `ruby:3.3` in Docker (bind-mounted source, a persistent
  named volume for the gem cache, `--network host` to reach the shared
  Postgres container) — the app itself has no Docker dependency, only this
  session's development environment did.
- Applied every lesson `benchmarks/apps/django`'s two review rounds
  surfaced, verified from the start instead of relearned: `t.text` (not
  Rails' migration-generator default `t.string`/`VARCHAR(255)`) for
  `email`/`name`; `t.references` with no `on_delete:`/`deferrable:` option,
  which Rails leaves as Postgres's own immediate/`NO ACTION` default
  (verified via `psql \d projects` showing no `DEFERRABLE` clause from the
  very first migration — no follow-up migration needed the way Django's
  was); `TIMESTAMPTZ` columns via
  `config/initializers/datetime_type.rb`'s documented
  `ActiveSupport.on_load(:active_record_postgresqladapter)` hook (Rails'
  Postgres adapter defaults to `timestamp without time zone` otherwise);
  D10's `validation_error` mapped to status 422 explicitly for a malformed
  JSON body (Rails' own `ActionDispatch::Http::Parameters::ParseError` is
  native HTTP 400, the same mismatch Django's `exception_handler` had after
  its own review); and `error.code` assertions in every rejection test from
  the first commit, not status-only.
- **Two idiomatic-Rails defaults turned out to satisfy contract
  requirements other implementations needed dedicated code for, discovered
  while writing the model rather than while debugging a test failure:**
  `belongs_to :owner` defaults to a required association in Rails 5+,
  which validates that the association actually *loads* — this rejects
  `owner_id: 0` and a nonexistent `owner_id` uniformly (no user has id 0
  either) for free, the case `gin-gorm`'s `binding:"required"` and
  `django`'s serializer `min_value=1` both needed dedicated code for.
  `validates :name, presence: true` alone rejects both `""` and
  whitespace-only names, because ActiveSupport's `String#blank?` (which the
  presence validator uses) is already whitespace-aware — `gin-gorm` and
  `django` each needed a separate `strip`/`trim`-based check added
  specifically because their frameworks' own "required"/blank checks only
  catch the empty string.
- The list endpoint's N+1 guard matches `gin-gorm`'s pinned 3-query/2-query
  shape *exactly* (verified against real Postgres query logs), unlike
  Django's 2-query JOIN strategy, which needed its own documented deviation
  in `benchmarks/docs/schema.md`. `Project.includes(:owner)` (not
  `.joins`/`.references`, which would force a JOIN) preloads owners via one
  batched `IN (...)` query, the same strategy `gin-gorm`'s GORM
  `.Preload("Owner")` uses, so no new documentation was needed there.
- 18-test suite (`bin/rails test`) mirrors `gin-gorm`/`gombit`/`django`'s
  contract test-for-test. One test-infrastructure risk anticipated and
  designed around from the start rather than hit and fixed afterward: list
  tests round-robin project ownership over the *actual* ids `User.create!`
  returns, not an assumed `1..user_count` range — Rails' default
  transactional tests roll back each test's rows but not Postgres
  sequences, the exact gap `django`'s own list-test fixture fell into
  before its review round fixed it.
- CI: added a Ruby 3.3.12 setup (`ruby/setup-ruby`, bundler-cache) +
  `bin/rails test` step to the existing `database-postgres` job, against a
  fifth database (`gombit_bench_rails_test`) — unlike Django's test runner,
  verified locally that `bin/rails test` does **not** auto-create a missing
  database (a nonexistent target raises a connection error), so CI needs an
  explicit `CREATE DATABASE` step first, the same pattern `gin-gorm`/
  `gombit` use rather than Django's throwaway-database convenience.
- Removed several `rails new`-generated files that would have referenced
  gems intentionally not pinned (`brakeman`, `bundler-audit`, `rubocop`,
  `thruster`/`bootsnap`) or assumed a `Dockerfile`/TLS-terminating proxy
  this benchmark doesn't have yet (`config.assume_ssl`/`force_ssl` both
  disabled, documented in the app's own README) — and deleted
  `config/master.key`/`credentials.yml.enc` entirely rather than committing
  either, since `SECRET_KEY_BASE` is supplied via env var and Rails itself
  refuses to boot in production without one set (verified: no
  `SECRET_KEY_BASE` set → a loud boot-time `ArgumentError`, not a silent
  insecure fallback the way Django's placeholder default is).

**Post-landing correction (review on PR #179,
github.com/gombit-dev/gombit/pull/179#pullrequestreview-5026967229):** the
CRUD wire contract (envelope, N+1 shape, seed formulas) was correctly
built, but the "production configuration" this PR claimed to pin was still
the Rails scaffold default in three places — all three confirmed true by
reproducing the actual behavior, not by re-reading the diff, and all three
fixed.

- **BLOCKING — request logging (issue §19).** Claim: the documented
  production command logs every request, and the health-check silencing
  targets a route (`/up`) this app doesn't serve, so it silences nothing.
  Verified by booting the pinned config against real Postgres and hitting
  both `/livez` and `/api/projects`: each produced a full
  `Started`/`Processing`/`Completed` log line at `info`, including the
  health check. `config/routes.rb`'s own comment claimed `/up` was "still
  available... left in place" — it was not; only `/livez` was ever defined.
  Fixed: `config/environments/production.rb`'s `RAILS_LOG_LEVEL` default
  changed from `"info"` to `"warn"`, `silence_healthcheck_path` repointed
  at `/livez`, and the false routes.rb comment corrected. Verified live
  after the fix: the same two requests plus a malformed-JSON POST produced
  no log output at all, and a direct `Rails.logger.warn`/`.error` check
  confirmed the `warn` threshold still surfaces real errors — satisfying
  "errors still logged" while eliminating the per-request noise gin-gorm
  and Django's own documented production commands never had.
- **MAJOR — worker topology (issue §18/CPU budget).** Claim: this app
  pinned Puma's single-process generator default onto a 2 vCPU budget,
  which MRI's GVL can't use a second core from, while `django`'s sibling
  had already left the equivalent scaffold default and pinned
  `--workers 4`. Accepted without needing further verification (the GVL's
  single-core-per-process behavior is well-established, not something
  worth re-deriving here) — the fix, not the diagnosis, needed rigor.
  Fixed: `config/puma.rb` pins `WEB_CONCURRENCY` to `2` (one worker per
  pinned vCPU) with `preload_app!`; `config/database.yml`'s pool size is
  now `POOL_MAX_OPEN` divided by `WEB_CONCURRENCY`, the same per-worker
  split `django`'s gunicorn configuration uses. Verified against real
  Postgres: cluster-mode boot with 2 workers, ten concurrent requests
  split across both with no connection errors, and
  `ActiveRecord::Base.connection_pool.size` reporting `10` per worker as
  computed (20 total).
- **MAJOR — schema claims the tests couldn't fail (issue schema.md
  equivalence).** Claim: the "TIMESTAMPTZ and non-deferrable FK verified
  against `psql \d`" narrative was true only of one manual check during
  development; none of the 18 tests queried `information_schema`/
  `pg_constraint`, so a `datetime_type.rb` load hook that silently stopped
  firing (e.g. after a future Rails upgrade renames the hook) would still
  pass the full suite on plain `timestamp without time zone` columns.
  Fixed: added `SchemaContractTest`, querying `information_schema.columns`
  and `pg_constraint` directly. Verified the test actually catches what it
  claims to, not just that it passes today: disabled the `datetime_type.rb`
  initializer and confirmed the timestamptz assertion failed with the exact
  expected message; separately altered the FK to `DEFERRABLE INITIALLY
  DEFERRED` via raw SQL and confirmed only the FK assertion failed, the
  timestamptz one still passing independently. 20 tests total after this
  round, all passing against real Postgres.

**Post-landing correction, round 2 (review on PR #179,
github.com/gombit-dev/gombit/pull/179#pullrequestreview-5027095244):** one
blocking error-path defect, confirmed by reproducing the 500 before fixing.

- `PATCH {"description": null}` returned a raw `ActiveRecord::NotNullViolation`
  **500**, not a D10 envelope. Reproduced live against the production server
  first (`500`, log showing `ActiveRecord::NotNullViolation (PG::NotNullViolation:
  ... null value in column "description")`), confirming the exact
  mechanism: `params.key?(:description)` is true for a present JSON null,
  the attribute was assigned `nil`, `Project` had no validation on
  `description`, and `save!` sent `NULL` to the NOT NULL column — while the
  `D10Envelope` rescued `RecordInvalid`/`RecordNotUnique`/`InvalidForeignKey`
  but not `NotNullViolation`. Create made it worse by silently coalescing a
  client's explicit null to `""` (`params[:description] || ""` → 201), so
  create and update taught two different contracts for one canonical field.
- Checked the siblings' actual behavior before choosing a fix rather than
  guessing: `benchmarks/apps/django` rejects a null `description` on both
  create and update (`422`, DRF `CharField` `allow_null=False` — verified
  live, including create-absent → `""`); `benchmarks/apps/gin-gorm` treats
  null as "not provided" (create `""`, update leaves it unchanged, via its
  `Description *string` update struct). The siblings genuinely disagree, so
  this corner is underspecified. Matched Django: reject a present-but-null
  value as `422 validation_error` uniformly across every canonical field
  (`name` via `presence`, `owner_id` via `belongs_to`, `description` via a
  new `render_null_violation` rescue of `ActiveRecord::NotNullViolation`),
  because that makes the NOT NULL path live-and-mapped (the reviewer's exact
  ask) rather than another dead backstop, keeps `name`/`owner_id` behavior
  unchanged, and makes create and update mean the same thing (create-absent
  still defaults to `""`; create-present-null and update-present-null both
  `422`).
- Added `test_rejects_null_description_on_create` and
  `..._on_update_without_partially_applying` (plus
  `test_create_without_description_defaults_to_empty_string`), and proved
  they earn their place: reverting only the two source files to the prior
  commit while keeping the tests, the create test fails (`201`, not `422`)
  and the update test errors with the exact `ActiveRecord::NotNullViolation`
  500. 23 tests total after this round, all passing against real Postgres;
  the full null matrix (create null → 422, create absent → `""`, update
  null → 422 with no partial apply, normal update still 200 with no poisoned
  connection) also re-verified live against the production server.

**Phase 4c — Laravel + Eloquent — done (fairness-check extension and
compose app service deferred, same as every prior sub-slice).**

- `benchmarks/apps/laravel`: the canonical API idiomatically in Laravel +
  Eloquent — PHP 8.3, Laravel 13.29.0, all direct deps pinned exact in
  `composer.json` and every transitive pinned in the committed
  `composer.lock` (`vendor/` not committed, regenerated by `composer
  install`, mirroring how `../rails` commits `Gemfile.lock` but not the
  gems). Host had no PHP; developed and tested via a `php:8.3` Docker image
  with `pdo_pgsql` + Composer, the app itself having no Docker dependency.
- **Production server model chosen by the user: traditional PHP-FPM + nginx**
  (`deploy/php-fpm.conf`, `deploy/nginx.conf`), not Laravel Octane /
  FrankenPHP's persistent worker. Documented honestly, in the app README and
  `deploy/php-fpm.conf`'s own header, that this re-bootstraps Laravel on
  every request — unlike the booted-once model of the Go binary / Puma
  cluster / gunicorn workers — because that is how most Laravel actually
  deploys, and the published numbers must be read with that in mind rather
  than as a language-speed comparison. Connection pooling (§18): FPM has no
  persistent pool, so `pm = static` / `pm.max_children = 20` caps concurrent
  workers (hence concurrent connections) at the pinned 20. Request logging
  (§19): Laravel does not log per-request by default (unlike Rails' scaffold,
  which needed quieting), and `deploy/nginx.conf` sets `access_log off`.
  Verified the real FPM+nginx path serves the full CRUD contract end to end
  (a scratch `php:8.3-fpm` + nginx container running the committed configs),
  not just `php artisan test`.
- Applied every lesson the Django and Rails review rounds surfaced, from the
  start: `text` columns (not `VARCHAR(255)`); a plain, non-deferrable FK
  (verified via `psql \d` and a DB-backed `SchemaContractTest`, no follow-up
  migration needed the way Django's was); `error.code` asserted on every
  rejection test, not status alone; malformed JSON and present-null
  `description` both `422 validation_error`; and the present-null
  `description` contract matched to `../rails`/`../django` (reject as 422),
  with the siblings' genuine disagreement on that corner documented.
- **Three Laravel-specific defaults caught and fixed before they could ship
  as silent contract violations** — the same class of framework-default
  fighting each prior app needed: (1) Laravel's global `TrimStrings` and
  `ConvertEmptyStringsToNull` middleware, removed in `bootstrap/app.php` —
  they would have stripped the whitespace the contract requires stored
  verbatim (the exact issue Django's review caught) and turned a legitimate
  `description: ""` into null; verified live that `"  padded  "` round-trips
  and `""` stays `""`. (2) `timestampTz` defaults to precision 0 (whole
  seconds) and Eloquent's Postgres grammar *writes* timestamps at
  whole-second precision — caught via `psql` (an API-created row came back
  `.000000` while the seeded rows and every sibling had microseconds), fixed
  with `timestampTz(..., 6)` in the migrations (the column) plus `$dateFormat
  = 'Y-m-d H:i:s.uP'` on the models (the write path). These are two
  independent invariants, each with its own test — the `datetime_precision =
  6` assertion pins only the column DDL (it stays 6 even if `$dateFormat` is
  removed and Eloquent writes whole seconds again), so a separate
  `test_eloquent_writes_microsecond_timestamps` round-trips a frozen
  `.123456` through Postgres and reads the raw stored text back to pin the
  write path; verified the split is real by removing `$dateFormat` and
  confirming that test fails while the precision test still passes. (This
  second test, and the correction of the README/plan claim that the
  precision assertion covered both fixes, landed in response to the PR #181
  review, github.com/gombit-dev/gombit/pull/181#pullrequestreview-5032891511.)
  (3) Laravel skips the non-implicit
  `regex:/\S/` rule on an empty string, so `PATCH {"name": ""}` first
  returned 200 and blanked the name (caught by the update-blank-name test
  failing); fixed with `sometimes|required` so a present empty/blank name is
  rejected while an omitted key is left untouched.
- The N+1 guard matches `../gin-gorm`'s pinned 3-query/2-query shape exactly
  via `Project::with('owner')` (a batched `where id in (...)`, not a JOIN —
  no documented deviation needed, unlike Django's 2-query `select_related`),
  verified against real Postgres in `test_list_does_not_n_plus_1[_on_empty_page]`.
- 21-test suite (`php artisan test`, PHPUnit, `RefreshDatabase` against a
  dedicated `gombit_bench_laravel_test` database) mirrors the four sibling
  suites test-for-test, plus `SchemaContractTest` and the seed idempotency
  and content-formula tests. CI: a `shivammathur/setup-php` 8.3 step +
  `composer install` + `php artisan test` added to the `database-postgres`
  job, against a fifth-app database it creates explicitly (RefreshDatabase
  migrates but does not create the database, like Rails).
- Stripped the scaffold's frontend (`resources/`, `vite`, `package.json`),
  its own `CLAUDE.md`/`AGENTS.md`/`README.md`, the sqlite default, and the
  cache/jobs/sessions migrations (array/sync drivers keep the schema to the
  canonical `users`/`projects` only). The generated `.env` (with a real
  `APP_KEY`) is gitignored and not committed; `.env.example` is the
  documented template and `APP_KEY` is supplied at runtime.

**Phase 4d — NestJS + TypeORM — done (fairness-check extension and compose
app service deferred). This completes all six canonical CRUD implementations
(the Go control, the real Gombit app, and the four ecosystem apps).**

- `benchmarks/apps/nestjs`: the canonical API idiomatically in NestJS +
  TypeORM — Node 24, NestJS 11.2.3, TypeORM 0.3.31, pg 8.23.0, TypeScript
  5.9.3, all direct deps pinned exact in `package.json` with the full tree in
  the committed `package-lock.json` (`npm ci`; `node_modules`/`dist` not
  committed). Node is the host toolchain, so no Docker was needed (unlike
  `../rails`/`../laravel`).
- **Two deliberate version choices, documented rather than defaulted:** (1)
  TypeORM **0.3.31**, not the 1.x major that shipped mid-2026 —
  `@nestjs/typeorm@11.0.3`'s integration is built around the mature 0.3.x
  line (peer only tentatively lists `^1.0.0-dev`), and the issue's
  "conventional Nest ORM" language favors the battle-tested stack over the
  1.x boundary. (2) **TypeScript 5.9.3**, not the 7.0 major now published —
  NestJS 11 is built for TS 5.x, so a 7.0 (native-compiler) jump is
  unnecessary risk. Same reasoning applied to `@types/node` (24.x, matching
  Node 24) and jest (29, matching ts-jest 29).
- Production config (§17): `NODE_ENV=production`, **compiled output**
  (`nest build` → `dist/`, run via `node dist/main`) — never a ts-node/watch
  dev server. Single Node process (one event loop) — the booted-once,
  persistent-process model like the Go binary / Rails / Django, and unlike
  Laravel's per-request FPM re-bootstrap. Pooling (§18): one global pool,
  `extra.max = POOL_MAX_OPEN` (20). Logging (§19): TypeORM query logging off,
  no per-request access log.
- **The one genuinely NestJS-specific correctness risk — microsecond
  timestamp fidelity on the *read* path — found and pinned:** the `pg` driver
  parses `timestamptz` into a JS `Date`, which holds only milliseconds, so a
  `timestamptz(6)` column silently loses microseconds every sibling keeps.
  `src/data-source.ts` overrides the `pg` type parsers (OID 1114/1184) to
  return the raw string, which the serializer reshapes to the canonical
  `...Z` form. Writes carry microseconds inherently (created_at is the DB
  `now()` default, updated_at set to SQL `now()` on update — no JS `Date`).
  Pinned by `schema-contract.e2e-spec.ts`'s read-path test, which round-trips
  a known `.123456` through the API; verified it earns its place by removing
  the parser override and confirming that test fails while the
  column-precision and FK assertions still pass (the DDL is unchanged, so
  they can't see the read-path bug — the same independent-invariant lesson
  from the Laravel review round, applied up front here).
- Applied the rest of the accumulated lessons from the start: `text` columns,
  a non-deferrable FK (verified via `psql` and a DB-backed schema test),
  present-null `description` rejected as 422 (via the DTO's
  `@ValidateIf(present) @IsString`) matching rails/django/laravel, FK-lean
  422 for a bad owner, malformed JSON → 422, `error.code` asserted on every
  rejection. Whitespace/empty-string preservation needed no work — NestJS
  does not trim request strings by default (unlike Laravel's TrimStrings).
- The N+1 guard matches `../gin-gorm`'s pinned 3-query/2-query shape exactly
  via TypeORM's `relationLoadStrategy: 'query'` (a batched owner `IN (...)`,
  not a JOIN — no documented deviation needed, unlike Django/Laravel's
  choices), counted directly with a custom TypeORM logger in
  `query-count.e2e-spec.ts`.
- 21-test suite (`jest`, ts-jest, `--runInBand` against a dedicated
  `gombit_bench_nestjs_test` database; e2e via supertest through the real
  request pipeline, plus a pure formula unit test) mirrors the five sibling
  suites. CI: an `actions/setup-node` 24 step + `npm ci` + `npm run build`
  (the production path is compiled output) + `npm test` added to the
  `database-postgres` job, against a database it creates explicitly (the
  tests run migrations but don't create the database).

**Post-landing correction (review on PR #182,
github.com/gombit-dev/gombit/pull/182#pullrequestreview-5033886519):** three
real MAJOR findings, all confirmed and fixed.

- The microsecond read-path fix was fragile: `ProjectService.iso()` did
  string surgery assuming the value was always a raw pg string with a `+00`
  offset. That held only by two coincidences the review named — TypeORM
  0.3.31 not hydrating the `timestamptz` alias to a JS `Date` (a `Date` has no
  `.replace` → a 500), and the session TZ happening to be UTC (nothing set
  it). Made it an entity-level fact instead: forced the session TZ to UTC
  (`extra.options: '-c timezone=UTC'` in `data-source.ts`, so `+00` is
  guaranteed), moved the string→ISO logic into an `isoTimestamp` column
  transformer that is defensive against receiving a `Date` (returns a valid
  ISO string rather than throwing — so the read-path test catches any
  precision loss instead of the app 500ing) and against a non-`+00` offset,
  and removed the service's string surgery. Re-verified the read-path test
  still catches removing the parser override (now as a value mismatch, not a
  crash) and that live timestamps still render `...Z` with microseconds.
- `query-count.e2e-spec.ts` built its own `DataSource` but never ran
  migrations — it passed only because Jest's file order ran the migrating
  spec first, and it hard-coded a `_test` default URL that disagreed with
  `data-source.ts`. Fixed by reusing `dataSourceOptions` (same url/entities/
  migrations, and the `setTypeParser` side effect) and calling
  `runMigrations()` on its own connection. Verified it now passes standalone
  against a freshly-created database (the review's exact "relation does not
  exist" scenario).
- The PATCH `updated_at: () => 'now()'` write path was claimed but untested —
  if the raw-SQL function value were ever dropped, PATCH would still 200 and
  leave `updated_at == created_at`, and the read-path test (which only INSERTs
  and GETs) could not see it. Added an "advances updated_at" test; verified it
  fails when the `now()` value is removed from the update. 21 tests total
  after this round.

### Phase 5 — Workload depth: auth overhead, TechEmpower-inspired, concurrency sweep

- Gombit-only auth-overhead benchmark: no-auth / JWT / cookie-session /
  cookie+CSRF variants of `GET /api/me` and `POST /api/projects`
  (`make benchmark-auth`). No cross-framework auth comparison (excluded by
  issue).
- TechEmpower-inspired subset (`/plaintext`, `/json`, `/db`,
  `/queries?queries=20`, `/updates?queries=20`) across all 6 apps
  (`make benchmark-techempower`), explicitly labeled "TechEmpower-inspired,"
  never "TechEmpower results."
- Concurrency/tail-latency sweep (1/10/100/500, attempt 1000, document if
  unsustained) on the headline `GET /api/projects?page=1&limit=20` workload,
  5 trials × 30s with 10s warm-up, coefficient-of-variation flag at >5%.
- **AC:** each `make benchmark-*` target runs independently and writes rows
  to `results.json` with trial variance recorded.

### Phase 6 — Operational footprint

- Cold-start (≥20 runs, median/p95), idle RSS, loaded RSS, CPU-under-load for
  all 6 implementations.
- Gombit-specific: build via `gombit build --embed`, verify the embedded
  binary serves API + admin + frontend assets, measure its binary size,
  container image size, cold start, and idle RSS as the headline
  single-binary-deployment number.
- **AC:** `make benchmark-footprint` produces footprint rows for all 6 apps
  plus the embedded-Gombit variant.

### Phase 7 — Reporting, README integration, drift detection

- `benchmarks/internal` summarizer: `results.json` → `results.csv` →
  `summary.md`, plus the README marker-block generator (§4).
- Add the `## Performance` section to root `README.md` (framework-tax table,
  PostgreSQL CRUD table, footprint table, methodology note with hardware/
  commit/date/PostgreSQL-version, "How not to interpret these results"
  link) inside the `<!-- benchmark-results:start/end -->` markers.
- `benchmarks/docs/methodology.md` with the full write-up (issue §12/§23),
  including the required "How not to interpret these results" section.
- `benchmarks/README.md`: how to run, how to add a framework.
- Run the full suite once locally/on a dedicated machine; commit the
  generated `benchmarks/results/latest/` snapshot and the resulting README.
- **AC:** `make benchmark-report` regenerates README + `summary.md` from
  `results.json` with zero manual edits; a drift check script confirms
  README matches generated output.

### Phase 8 — CI integration

- `ci.yml`: add a `benchmark-smoke` job (needs: `test`) running
  `make benchmark-smoke` — tiny seed, 1 short trial, low concurrency,
  correctness-only, on every PR.
- New `.github/workflows/benchmarks.yml`: `workflow_dispatch` with an input
  to select `micro | crud | auth | techempower | footprint | full`; uploads
  `metadata.json`, raw results, `results.json`, `results.csv`, `summary.md`
  as artifacts. Explicitly does not overwrite committed README numbers
  automatically — that stays a reviewed, manual step (Phase 7's local run).
- **AC:** smoke job is green on PRs and fails clearly if any of the 6 apps'
  Docker builds break, a route regresses, or the result parser breaks;
  manual workflow completes and uploads artifacts without touching
  `main`'s README.

---

## 7. Validation commands (per the working agreement, run before closing #141)

```bash
go test ./...
golangci-lint run
go test ./benchmarks/micro/... -bench=BenchmarkFrameworkTax -benchmem -count=10
cd benchmarks && make benchmark-smoke
cd benchmarks && make benchmark   # full suite, run at least once before merge of the reporting PR
```

Plus the existing SQLite/PostgreSQL/MySQL CI matrix must stay green
throughout — Phases 3-6 only add new Docker services in `benchmarks/compose.yml`,
they must not touch `ci.yml`'s existing database/migrations/conformance jobs.

---

## 8. Open questions to confirm before Phase 1

1. **k6 as the load generator** — confirms native constant-arrival-rate
   support satisfies the coordinated-omission requirement; alternative is
   wrk2 (simpler, but text-output-only and no native arrival-rate control
   for the concurrency sweep).
2. **Seed size** (1,000 users / 100,000 projects as specified, vs. a smaller
   documented size) — depends on how slow local/CI seeding turns out to be;
   decide empirically in Phase 3, not up front.
3. **Whether Phase 4's four competitor apps ship as one PR or four** — four is
   easier to review; recommend four sequential PRs against #141.

None of these block starting Phase 1 — they only affect Phase 3-4 specifics.
