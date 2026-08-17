# Implementation Plan: [M2-2] `gombit db migrate` / `rollback` / `status`

**Issue:** [#13](https://github.com/LAA-Software-Engineering/gombit/issues/13)
**Backlog ID:** M2-2 (build plan §4)
**Depends on:** M2-1 / #12 (`gombit db makemigrations`) — done on `main`
**Size:** M · Labels: `migrations`, `cli`
**Milestone:** M2 — Migrations

> GitHub issue body was not readable from this environment (API 403 / public URL 404).
> This plan is derived from `docs/GOMBIT_BUILD_PLAN.md` §4 M2-2, locked D3/D4,
> ADR-012, and the landed M2-1 surface under `migrations/` + `cmd/gombit`.

---

## 1. Goal

Ship the apply / rollback / status half of the Django-style migration CLI:

| Command | Behavior |
| --- | --- |
| `gombit db migrate` | Apply pending versioned SQL migrations (Atlas apply) and record them in Gombit’s D4 revision table |
| `gombit db rollback` | Undo the latest batch using application-owned down SQL + D4 rows (not `atlas migrate down`) |
| `gombit db status` | Report applied / pending revisions correctly on SQLite, PostgreSQL, and MySQL |

**Acceptance criteria (build plan):**

- [ ] Up / down / status reflected and correct on all supported DBs (SQLite, PostgreSQL, MySQL)
- [ ] Revision metadata is `version, name, batch, applied_at` with **no checksum** (D4)
- [ ] Forward path uses Atlas Community Edition `migrate apply`
- [ ] Rollback is Gombit-owned over the D4 table + application-owned down SQL (ADR-012)

---

## 2. Non-goals (do not pull into this PR)

| Deferred | Owner |
| --- | --- |
| `gombit db seed` / `reset` | M2-3 |
| Full DB conformance matrix (CRUD, timestamps, pagination, …) | M2-4 |
| Auto-generating `.down.sql` from `makemigrations` | optional follow-up; not required for M2-2 AC |
| App-owned model registry / generator wiring | M4 |
| Atlas Pro commands (`migrate down`, lint, drift, registry) | explicitly rejected by ADR-012 |
| Runtime auto-migrate on `framework.App` boot | later; CLI owns apply for v0.1 |

---

## 3. Locked constraints (do not re-litigate)

1. **D3** — Wrap Atlas; no hand-rolled migration DSL.
2. **D4** — Track `version, name, batch, applied_at`. No checksum column (design doc §17.4 still shows checksum; build plan wins).
3. **ADR-012** — Default path is Atlas **Community Edition** only:
   - Allowed: `atlas migrate apply`, `atlas migrate status`, existing `migrate diff`.
   - Forbidden for default path: `atlas migrate down` and other Pro/MSA-gated commands.
4. **Config boundary** — DSN/driver come from `config.Load` / typed `config.DatabaseConfig`. No new `os.Getenv` in `migrations/`.
5. **Milestone scope** — No M6 batteries; keep behavior inside the `migrations` package + thin CLI wiring.

---

## 4. Current baseline (M2-1)

Already on `main`:

- `migrations.MakeMigrations` — Program Mode loader → temp `schema.sql` → `atlas migrate diff`
- Default dir: `database/migrations`
- CLI: `gombit db makemigrations <name> --model ... [--driver] [--dir] [--atlas-bin]`
- Docs: `docs/migrations.md`, ADR-012, `examples/migrations/`
- CI: `Migrations` job installs Atlas Community Edition and runs `go test ./migrations`
- Database matrix already exists for `./database` (`integration` tag + Postgres/MySQL services)

M2-2 extends the same packages; it should not rewrite makemigrations.

---

## 5. Design

### 5.1 Two ledgers, one source of truth for Gombit UX

Atlas apply maintains its own revision bookkeeping (Atlas schema revisions table).
Gombit additionally maintains:

```text
framework_migrations
  version     text/string   -- Atlas version stamp, e.g. 20260815120000
  name        text/string   -- suffix, e.g. create_products
  batch       int           -- Laravel-style batch id for grouped rollback
  applied_at  timestamptz   -- when Gombit recorded the apply
```

**Source of truth for `status` / `rollback`:** `framework_migrations` + files on disk.
**Forward SQL execution:** `atlas migrate apply` (Community Edition).
**Integrity file:** `atlas.sum` stays Atlas-owned directory integrity; it is **not** mirrored into D4.

### 5.2 Migration file convention

Atlas already writes:

```text
database/migrations/
  atlas.sum
  20260815120000_create_products.sql          # up (Atlas-owned)
  20260815120000_create_products.down.sql     # down (application-owned; required for rollback)
```

Rules:

- **Up** files match Atlas versioned migration naming: `<version>_<name>.sql`.
- **Down** files are optional for `migrate` / `status`, **required** for rolling back that version.
- Missing down on rollback → fail with an actionable error naming the version (do not partially leave the batch inconsistent if avoidable).
- M2-2 does not teach `makemigrations` to emit downs; tests and the example ship hand-written downs. Document the convention in `docs/migrations.md`.

### 5.3 `migrate` algorithm

1. Load typed config (`Driver`, `DSN`); resolve `--dir` / `--atlas-bin`.
2. Ensure `framework_migrations` exists (create if missing via GORM/`database.Open` or raw SQL).
3. List pending ups: migration dir versions not present in `framework_migrations` (and expected by Atlas). Prefer computing pending from dir + D4 before invoke so batch size is known.
4. If nothing pending → print “no pending migrations” and exit 0.
5. Invoke:

   ```text
   atlas migrate apply
     --url <atlas-url-from-dsn>
     --dir file://<migration-dir>
   ```

   (Exact flags may include `--config` if we generate a tiny HCL like makemigrations; prefer the smallest Community Edition surface that works for sqlite/postgres/mysql.)

6. On Atlas success, insert one D4 row per newly applied version with `batch = max(batch)+1` (same batch for the whole invoke) and `applied_at = now()`.
7. If Atlas fails → do not write D4 rows; return the Atlas error.

**DSN → Atlas URL** is a first-class helper (sqlite / postgres / mysql). Cover with table-driven tests; this is the main portability footgun.

### 5.4 `rollback` algorithm (Gombit-owned)

1. Read latest `batch` from `framework_migrations`. If none → exit 0 / “nothing to roll back”.
2. Load all rows for that batch ordered by `version` descending.
3. For each row:
   - Require `<version>_<name>.down.sql`.
   - Execute down SQL against the app DB (via `database.Open` / `sql.DB`), preferably in a transaction when the driver supports it (`Capabilities().Transactions`).
   - Delete the D4 row.
   - Delete / rewind the matching Atlas revision row for that version so a later `migrate` can re-apply (see §5.6).
4. Optional v0.1 flag: `--steps N` (number of batches). Default `1`. Keep the flag even if only `1` is tested first.

Do **not** call `atlas migrate down`.

### 5.5 `status` algorithm

Print a stable, script-friendly table (or aligned text) with at least:

| version | name | state | batch | applied_at |
| --- | --- | --- | --- | --- |

- `applied` — present in D4
- `pending` — up file on disk, absent from D4

Do not depend on Pro features. Optionally call `atlas migrate status` for diagnostics in verbose mode later; v0.1 AC is satisfied by D4 + dir scan.

### 5.6 Atlas revision table vs D4 after rollback

Risk: after Gombit rollback, Atlas may still believe versions are applied, so the next `migrate apply` no-ops while D4 shows pending.

**Plan decision (implement this):** after successful down SQL + D4 delete, also remove the corresponding Atlas revision rows for those versions in the same logical operation (best-effort same transaction when possible). Document the table name Atlas uses and isolate access behind a small internal helper so it is not a public API.

If Community Edition `migrate apply` exposes a safer supported rewind for CE-only workflows during implementation, prefer that over raw deletes — but do not take a Pro dependency.

### 5.7 Package / CLI surface

Extend `migrations` (runtime logic) and thin `cmd/gombit` (flags only):

```text
migrations/
  makemigrations.go          # existing
  migrate.go                 # Migrate(ctx, Options) error
  rollback.go                # Rollback(ctx, Options) error
  status.go                  # Status(ctx, Options) ([]RevisionStatus, error) + render helper
  revisions.go               # framework_migrations ensure/list/insert/delete, batch helpers
  atlas_url.go               # DSN → Atlas URL
  files.go                   # scan ups/downs, parse version/name
  *_test.go
  testdata/                  # fixture up/down SQL for unit/integration tests

cmd/gombit/main.go
  case "migrate" | "rollback" | "status"
```

Shared options (mirror makemigrations style):

```go
type ApplyOptions struct {
    WorkDir      string
    MigrationDir string
    AtlasBinary  string
    Database     config.DatabaseConfig
    Steps        int // rollback only; default 1
    Stdout, Stderr io.Writer
    // injectable runner / db open for tests
}
```

CLI sketches:

```sh
gombit db migrate   [--dir database/migrations] [--atlas-bin atlas]
gombit db rollback  [--dir database/migrations] [--steps 1]
gombit db status    [--dir database/migrations]
```

Driver/DSN always from env/config (`GOMBIT_DATABASE_*`), not duplicated flags, unless a test-only override is needed.

---

## 6. Implementation steps

1. **Revisions model + store**
   - Define unexported or narrowly exported revision type matching D4.
   - `EnsureRevisionsTable`, `ListApplied`, `InsertBatch`, `DeleteVersions`, `LatestBatch`.
   - Unit-test against SQLite in-process (no Atlas).

2. **File scanner**
   - Parse Atlas up filenames; detect companion `.down.sql`.
   - Ignore `atlas.sum` and unrelated files.
   - Table-driven parse tests.

3. **Atlas URL helper**
   - Map `config.DatabaseConfig` → Atlas `--url` for sqlite/postgres/mysql.
   - Fail loudly on unsupported / unparseable DSNs.

4. **`Migrate`**
   - Wire injectable command runner (same pattern as `MakeMigrations`).
   - Fake-runner unit tests for args + D4 inserts.
   - Real Atlas smoke test on SQLite when `ATLAS_BINARY` is set (match M2-1).

5. **`Rollback`**
   - Execute downs; update D4; sync Atlas revision rows.
   - Tests: happy path, missing down file, empty history, multi-version single batch.

6. **`Status`**
   - Pure scan + D4 join; golden/table tests for rendering.

7. **CLI wiring**
   - Extend `cmd/gombit` switch + usage strings + `main_test.go` cases.

8. **Docs + example**
   - Extend `docs/migrations.md` with migrate/rollback/status, down-file convention, CE install reminder.
   - Update `examples/migrations/README.md` with a SQLite walkthrough: makemigrations → migrate → status → rollback → status.
   - Touch ADR-012 only if implementation discovers a CE detail that contradicts it (keep ADR authoritative).

9. **CI**
   - Keep `Migrations` job for package unit/smoke tests with Atlas CE.
   - Add migrate/rollback/status coverage to the existing Database matrix job (or a sibling job) with `-tags integration` against Postgres + MySQL services — required by AC “all supported DBs”.

---

## 7. Test plan

| Layer | What | DBs |
| --- | --- | --- |
| Unit | filename parse, batch numbering, status rendering, Atlas argv, URL mapping | n/a / sqlite memory |
| Package smoke | real `atlas migrate apply` on fixture dir when `ATLAS_BINARY` set | SQLite |
| Integration | migrate → status(applied) → rollback → status(pending) → migrate again | SQLite, PostgreSQL, MySQL |
| CLI | flag parsing / unknown subcommands / config load errors | n/a |

Follow existing patterns:

- Fake `commandRunner` for non-Atlas unit tests (see `makemigrations_test.go`).
- `//go:build integration` + `-database.postgres-dsn` / `-database.mysql-dsn` flags like `database/integration_test.go`.
- Prefer table-driven `t.Run` with useful failure messages.

---

## 8. Risks and open points

1. **Atlas↔D4 desync after rollback** — highest risk; §5.6 is mandatory work, not a follow-up.
2. **DSN dialect gaps** — GORM DSNs are not always valid Atlas URLs; invest in the mapper early.
3. **Transactional DDL** — MySQL/SQLite DDL transaction behavior differs; rollback should still leave D4 consistent even if some drivers auto-commit DDL.
4. **Partial Atlas apply** — if Atlas applies 2 of 3 then fails, D4 must not claim success; prefer “Atlas owns the apply transaction/error” and only insert D4 rows after process exit 0, then verify applied set (re-list Atlas or re-scan) before insert.
5. **Down SQL authorship** — without generator support, rollback UX depends on users writing downs; docs must be explicit.
6. **Issue body drift** — if #13’s GitHub AC text differs from the build plan, build plan §4 + this plan win only after confirming with the issue; re-read #13 when API access allows and adjust the PR description.

---

## 9. Definition of done

Per AGENTS.md / build plan §5:

1. New behavior has tests; DB-touching paths pass SQLite + PostgreSQL + MySQL.
2. Docs updated (`docs/migrations.md`) and example walkthrough updated.
3. No generator/OpenAPI/frontend scope creep.
4. No secrets in tree; config stays typed.
5. PR links #13 and lists which AC items are satisfied.
6. Run project `code-review` skill on the implementation PR before merge.

---

## 10. Suggested PR slice (when implementing)

One PR for the full M2-2 surface is appropriate (size M). If needed, split only as:

1. Revisions store + status (read path) + docs stub
2. Migrate (Atlas apply + D4 write) + integration
3. Rollback (downs + Atlas revision rewind) + integration

Default: single PR titled like `feat: add db migrate rollback status` with `Closes #13`.
