# rails

The BENCH-1 Ruby on Rails + ActiveRecord ecosystem-context app (issue
[#141](https://github.com/gombit-dev/gombit/issues/141) Phase 4): the same
canonical `/api/projects` CRUD API ([../../docs/schema.md](../../docs/schema.md))
as `../gin-gorm`, `../gombit`, and `../django`, implemented idiomatically in
Rails — non-Go ecosystem context, not another framework-tax control.

## Versions (issue #141 §16/§17: pin exact versions)

| Package | Version |
| --- | --- |
| Ruby | 3.3.12 |
| Rails | 8.1.3.1 |
| pg | 1.6.3 |
| puma | 8.0.2 |

See [Gemfile](Gemfile). No Docker/Node toolchain needed to develop this app
beyond a Ruby install matching [.ruby-version](.ruby-version); local
development in this repo used `ruby:3.3` in Docker only because the host
had no Ruby installed at all.

## Run

```sh
docker compose -f benchmarks/compose.yml up -d postgres
createdb -h 127.0.0.1 -p 55432 -U gombit gombit_bench_rails   # once

cd benchmarks/apps/rails
bundle install
DATABASE_URL="postgres://gombit:gombit@127.0.0.1:55432/gombit_bench_rails?sslmode=disable" \
  bin/rails db:migrate

# seed (1,000 users, 100,000 projects) — idempotent, truncates first
DATABASE_URL="postgres://gombit:gombit@127.0.0.1:55432/gombit_bench_rails?sslmode=disable" \
  bin/rails db:seed

# production configuration (issue #141 §17) — never bin/rails server in
# development mode, and RAILS_ENV=production requires SECRET_KEY_BASE
# (Rails itself refuses to boot without one — no insecure placeholder
# default here, unlike ../django's DJANGO_SECRET_KEY)
RAILS_ENV=production SECRET_KEY_BASE=$(bin/rails secret) \
DATABASE_URL="postgres://gombit:gombit@127.0.0.1:55432/gombit_bench_rails?sslmode=disable" \
  bin/rails server
```

Env vars (all optional except `SECRET_KEY_BASE` in production; defaults
match `benchmarks/compose.yml`):

| Var | Default | |
| --- | --- | --- |
| `DATABASE_URL` | `postgres://gombit:gombit@127.0.0.1:55432/gombit_bench_rails?sslmode=disable` | |
| `PORT` | `8083` | |
| `SECRET_KEY_BASE` | none — Rails refuses to boot in production without it | |
| `RAILS_ALLOWED_HOSTS` | `127.0.0.1,localhost` | comma-separated; matches `../django`'s `DJANGO_ALLOWED_HOSTS` |
| `RAILS_MAX_THREADS` | `3` | Puma thread pool size per process |
| `POOL_MAX_OPEN` | `20` | issue #141 §18 "Connection pooling"; see below |

### Production configuration (issue #141 §17)

```text
RAILS_ENV=production
Puma (single process, RAILS_MAX_THREADS threads) — never bin/rails server
in development mode
```

`config/environments/production.rb` disables `assume_ssl`/`force_ssl`
(Rails' scaffolded defaults expect a TLS-terminating reverse proxy in
front; every `benchmarks/apps/` implementation is served plain HTTP
directly for this benchmark, so a stock Rails production config would 301
or reject every `curl`/fairness-check request).

### Database connection pooling (issue #141 §18)

Puma's *default* topology (no `WEB_CONCURRENCY` set — this app's pinned
configuration) is a single process with a thread pool, not pre-forked
workers — unlike `../django`'s gunicorn, which needed `POOL_MAX_OPEN`
divided across worker processes because each gunicorn worker is a separate
OS process with its own connection pool. A single Puma process means
`config/database.yml`'s `pool: <%= ENV.fetch("POOL_MAX_OPEN", 20) %>` is
already the *one* global pool for the whole server, the same single-pool
topology `gin-gorm`/`gombit`'s single Go binary has — no per-worker division
needed. If this app is ever run with `WEB_CONCURRENCY > 1`, `POOL_MAX_OPEN`
would need the same per-worker split `../django`'s README documents; not
done here because the pinned configuration doesn't use clustering.

## Test

```sh
docker compose -f benchmarks/compose.yml up -d postgres
createdb -h 127.0.0.1 -p 55432 -U gombit gombit_bench_rails_test   # once —
  # a separate database from gombit_bench_rails: Rails' db:test:prepare
  # would otherwise load schema.rb over whatever the dev/prod database
  # currently holds
cd benchmarks/apps/rails
DATABASE_URL="postgres://gombit:gombit@127.0.0.1:55432/gombit_bench_rails_test?sslmode=disable" \
  RAILS_ENV=test bin/rails test
```

`bin/rails test` loads `db/schema.rb` into the test database automatically
(`ActiveRecord::Migration.maintain_test_schema!`, wired into
`test/test_helper.rb` by Rails' own generator) if it's out of date — no
separate `db:test:prepare` step needed as long as `db/schema.rb` is
committed and current.

Covers the same contract `../gin-gorm/main_test.go`,
`../gombit/internal/project/handler_test.go`, and
`../django/projects/tests.py` pin for their own implementations: full
create/get/update/delete/404/validation-failure round trip, including that
a foreign-key violation (`owner_id` referencing no existing user) is
rejected as `422` with D10's `validation_error` code — for free, via
`belongs_to`'s default required-association validation, not a hand-rolled
check (see "Schema and validation notes" below); list pagination meta and
deterministic `id DESC` ordering across a page boundary; a query-count
regression guard for both a non-empty page (`test_list_does_not_n_plus_1`,
exactly 3 SQL statements — matching `gin-gorm`'s pinned shape exactly, see
below) and an empty one (`test_list_does_not_n_plus_1_on_empty_page`,
exactly 2); the seed content formulas' own determinism and round-robin math
(`CanonicalSeedTest`, ported by hand — this app can't import
`benchmarks/apps/shared` or reuse `../django`'s Python port); and the seed
contract's DB-backed half, run twice at reduced scale to confirm it's
idempotent rather than accumulating duplicate data
(`SeedDatabaseTest`).

## Schema and validation notes

Verified against real PostgreSQL (`psql \d users`, `\d projects`) against
[../../docs/schema.md](../../docs/schema.md) — matched from the start,
rather than discovered as bugs afterward the way `../django`'s review round
found several:

- `t.text`, not Rails' migration-generator default `t.string`
  (`VARCHAR(255)`), for `users.email`/`name` and `projects.name`: the
  canonical schema's columns are unbounded `TEXT`.
- `t.references :owner, foreign_key: { to_table: :users }` with no
  `on_delete:`/`deferrable:` option: Rails only adds `ON DELETE
  CASCADE`/etc. or a `DEFERRABLE` clause if explicitly asked, so the
  generated constraint matches the canonical FK's `NO ACTION`, immediately
  checked, by default. `../django`'s Django ORM instead *always* emits
  `DEFERRABLE INITIALLY DEFERRED` for Postgres FKs regardless of any model
  option, which broke its `IntegrityError` handling inside any wrapping
  transaction until a follow-up migration fixed it — verified here from the
  start that no such follow-up is needed (`psql \d projects` shows no
  `DEFERRABLE` clause).
- `created_at`/`updated_at` are `TIMESTAMPTZ` (`timestamp(6) with time
  zone`), not the Postgres adapter's own default (`timestamp without time
  zone`) — set globally via
  `config/initializers/datetime_type.rb`'s
  `ActiveSupport.on_load(:active_record_postgresqladapter) { self.datetime_type = :timestamptz }`,
  the Rails-documented way to opt into this for every `t.datetime`/
  `t.timestamps` column.
- `belongs_to :owner` defaults to required in Rails 5+, which validates
  that the association actually loads — this single idiomatic default
  rejects `owner_id: 0` *and* a nonexistent `owner_id` uniformly (no user
  has id 0 either), the same case `gin-gorm`'s `binding:"required"` and
  `../django`'s serializer `min_value=1` both needed dedicated code to
  cover. `app/controllers/concerns/d10_envelope.rb`'s
  `render_invalid_foreign_key` stays as a backstop for any future write
  that bypasses model validations (e.g. `update_column`), mirroring
  `gin-gorm`'s own comment about keeping a currently-unreachable case.
- `validates :name, presence: true` alone rejects both `""` and
  whitespace-only names: ActiveSupport's `String#blank?` (which Rails'
  presence validator uses) already treats a whitespace-only string as
  blank — `gin-gorm`'s Gin `binding:"required"` and `../django`'s DRF
  `CharField` both only reject the empty string and needed a dedicated
  `strip.empty?`-style check added separately.
- The list endpoint's N+1 guard matches `gin-gorm`'s pinned shape exactly
  (3 queries for a non-empty page — page `SELECT`, one batched
  `SELECT ... WHERE id IN (...)` for owners, `COUNT`; 2 for an empty page,
  no owners to preload), verified against real Postgres query logs.
  `Project.includes(:owner)` (not `.joins`/`.references`, which would force
  a `JOIN` instead) is what produces this — unlike `../django`'s
  `select_related`, which compiles to a single `JOIN` and needed its own
  2-query deviation documented in `benchmarks/docs/schema.md`.
- `POST`/`PATCH` with an unparseable JSON body is rejected as `422` with
  `validation_error`, mapped explicitly in
  `app/controllers/concerns/d10_envelope.rb`'s `render_malformed_json` —
  Rails' own `ActionDispatch::Http::Parameters::ParseError` is native HTTP
  400, the same status mismatch `../django`'s `envelope.exception_handler`
  had to fix after review; matched here from the start instead of
  relearned.
- `description` is never trimmed: Ruby has no ActiveSupport equivalent of
  DRF's `CharField(trim_whitespace=True)` default, so there was no
  framework default to override in the first place — unlike `../django`,
  which needed `trim_whitespace=False` added explicitly after review found
  it missing on `description` (but present on `name`).

## Status

Schema, seed, CRUD app, its own test suite, and CI (a Ruby 3.3.12 +
`bin/rails test` step in `.github/workflows/ci.yml`'s `database-postgres`
job) are done (tracked in
[docs/plans/BENCH-1-benchmark-suite.md](../../../docs/plans/BENCH-1-benchmark-suite.md)
Phase 4). Still open, matching the Phase 3a/3b/4a precedent: a
`Dockerfile`/`benchmarks/compose.yml` service, and extending the Go
`benchmarks/apps/fairness_test.go` cross-implementation check to include
this app as a fourth leg.
