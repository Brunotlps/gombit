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
- **Deferred to a later Phase 1/2 follow-up:** `benchstat`-based
  `-count=10` summarization is documented (run manually) but not yet wired
  into a `make benchmark-micro` target or `results.json` output — that
  depends on Phase 1's still-open result-schema/summarizer work.
- **AC:** `go test ./benchmarks/micro/... -bench=BenchmarkFrameworkTax -benchmem -count=10`
  runs all four stacks × five scenarios and reports `ns/op`/`B/op`/`allocs/op`.
  `make benchmark-micro` / `results.json` output is not yet satisfied (see
  above).

### Phase 3 — Canonical schema, seed, and CRUD apps (Gombit + Gin+GORM)

- Define the `users`/`projects` schema + indexes (issue §3) as the canonical
  PostgreSQL DDL, expressed through each framework's own migration
  mechanism but converging on the same SQL.
- Deterministic seed (1,000 users / 100,000 projects, or a documented smaller
  size if local setup time proves prohibitive) with a single reproducible
  command.
- Implement `benchmarks/apps/gombit` (normal Gombit app, production mode) and
  `benchmarks/apps/gin-gorm` (idiomatic Gin+GORM control) exposing the
  canonical CRUD routes (`GET/POST/PATCH/DELETE /api/projects...`) with
  Gombit's `{data, meta}` envelope on both, equivalent pagination.
- Add automated fairness checks (issue §15/§16): same route surface, same
  JSON shape, same paginated record count, same ordered IDs for a known
  query, N+1 detection via SQL query counting for the list endpoint.
- `compose.yml`: pinned PostgreSQL service, resource limits, app services for
  these two.
- **AC:** `make benchmark-crud` runs Gombit and Gin+GORM against the same
  Postgres instance and produces comparable `results.json` rows; fairness
  tests fail loudly on route/shape/query-count divergence.

### Phase 4 — Remaining competitor apps (Django, Rails, Laravel, NestJS)

- One sub-slice per framework (can be up to 4 separate PRs if easier to
  review): same schema/seed/routes/pagination contract as Phase 3, each in
  its documented production configuration (issue §17), each with pinned
  lockfiles.
- Extend the fairness checks to cover all 6 implementations.
- **AC:** all 6 apps pass the same fairness suite; `docker compose` brings
  all 6 up against one PostgreSQL instance with documented resource limits.

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
