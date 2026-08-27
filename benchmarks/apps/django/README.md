# django

The BENCH-1 Django + Django REST Framework ecosystem-context app (issue
[#141](https://github.com/gombit-dev/gombit/issues/141) Phase 4): the same
canonical `/api/projects` CRUD API
([../../docs/schema.md](../../docs/schema.md)) as `../gin-gorm` and
`../gombit`, implemented idiomatically in Python — non-Go ecosystem context,
not another framework-tax control (that comparison is `gin-gorm` vs
`gombit`, same language/runtime/ORM family).

## Versions (issue #141 §16/§17: pin exact versions)

| Package | Version |
| --- | --- |
| Python | 3.12 |
| Django | 5.2.17 (current LTS) |
| djangorestframework | 3.18.0 |
| psycopg[binary,pool] | 3.3.4 |
| gunicorn | 26.2.0 |

See [requirements.txt](requirements.txt).

## Run

```sh
docker compose -f benchmarks/compose.yml up -d postgres
createdb -h 127.0.0.1 -p 55432 -U gombit gombit_bench_django   # once

cd benchmarks/apps/django
python3.12 -m venv .venv && .venv/bin/pip install -r requirements.txt
.venv/bin/python manage.py migrate

# seed (1,000 users, 100,000 projects) — idempotent, truncates first
.venv/bin/python manage.py seed

# production configuration (issue #141 §17) — never manage.py runserver
.venv/bin/gunicorn config.wsgi:application --bind 0.0.0.0:8082 --workers 4
```

Env vars (all optional, defaults match `benchmarks/compose.yml`):

| Var | Default | |
| --- | --- | --- |
| `DATABASE_URL` | `postgres://gombit:gombit@127.0.0.1:55432/gombit_bench_django?sslmode=disable` | |
| `DJANGO_DEBUG` | `false` | issue #141 §17 requires `DEBUG=False`; only set for local debugging |
| `DJANGO_ALLOWED_HOSTS` | `127.0.0.1,localhost` | comma-separated |
| `DJANGO_SECRET_KEY` | a fixed placeholder | see settings.py — never used to protect anything in this app |
| `POOL_MAX_OPEN` | `20` | issue #141 §18 "Connection pooling"; see below for how this is split |
| `GUNICORN_WORKERS` | `4` | must match the actual `--workers` gunicorn is started with |

### Production configuration (issue #141 §17)

```text
DEBUG=False
gunicorn (WSGI), not manage.py runserver
```

### Database connection pooling (issue #141 §18)

Every other implementation under `benchmarks/apps/` is a single process with
one global connection pool capped at 20 open / 20 idle. gunicorn's pre-fork
model has no equivalent of a single global pool: each worker is its own OS
process with its own memory, so each one holds its own independent
[psycopg3 connection pool](https://www.psycopg.org/psycopg3/docs/advanced/pool.html)
(`config/settings.py`'s `DATABASES["default"]["OPTIONS"]["pool"]`) — there is
no cross-process pool to share.

Rather than let each worker independently open up to 20 connections (4
workers × 20 = 80, four times the pinned ceiling — not comparable to any
other implementation), `POOL_MAX_OPEN` is divided by `GUNICORN_WORKERS` and
each worker's pool is fixed at that size (`min_size == max_size`, so it
behaves like GORM's `SetMaxOpenConns(n) == SetMaxIdleConns(n)`, not a
pool that grows and shrinks): 20 ÷ 4 = 5 connections per worker, 20 total
across the whole server. Changing `--workers` without also setting
`GUNICORN_WORKERS` to match under-reports the real per-worker pool size to
Django, so the two must be kept in sync (documented here per issue #141
§17: "Any non-default tuning must be documented").

## Test

```sh
# Django's test runner creates and migrates its own throwaway PostgreSQL
# database (test_<DATABASES["default"]["NAME"]>) — no separate -dsn flag or
# docker-compose step beyond having a reachable Postgres instance to point
# DATABASE_URL at.
docker compose -f benchmarks/compose.yml up -d postgres
cd benchmarks/apps/django
DATABASE_URL="postgres://gombit:gombit@127.0.0.1:55432/gombit_bench_django?sslmode=disable" \
  .venv/bin/python manage.py test projects -v 2
```

Covers the same contract `../gin-gorm/main_test.go` and
`../gombit/internal/project/handler_test.go` pin for their own
implementations: full create/get/update/delete/404/validation-failure round
trip, including that a foreign-key violation (`owner_id` referencing no
existing user) is rejected as `422`, not `500`
(`test_create_rejects_nonexistent_owner_id` — see "Discovered Django-specific
issues" below for why this needed more than the obvious code); list
pagination meta and deterministic `id DESC` ordering across a page boundary;
a query-count regression guard for both a non-empty and an empty page
(`test_list_does_not_n_plus_1[_empty_page]` — see "Schema and query-shape
notes" below for why this is 2 queries, not gin-gorm's 3); the seed content
formulas' own determinism and round-robin math
(`SeedContentTests`, ported by hand from `benchmarks/apps/shared`, which is
Go-only and can't be imported here); and the seed contract's DB-backed half,
run twice at reduced scale to confirm it's idempotent rather than
accumulating duplicate data (`SeedDatabaseIsIdempotentAndCorrectTests`).

## Schema and query-shape notes

Verified against real PostgreSQL (`psql \d users`, `\d projects`) against
[../../docs/schema.md](../../docs/schema.md):

- `users.email`/`name` and `projects.name` are `models.TextField`, not
  Django's usual `CharField`/`EmailField` — the canonical schema's columns
  are `TEXT`, matching what GORM's plain `string` (no size tag) generates
  for `gin-gorm`/`gombit`. The 255-character cap this app still enforces on
  `name` lives at the serializer layer (`projects/serializers.py`), the same
  app-layer-only validation every other implementation uses, not a DB
  constraint.
- `Project.owner` uses `on_delete=models.DO_NOTHING` and `db_index=False`
  (with an explicit `Meta.indexes` entry instead): Django's usual
  `on_delete=CASCADE` default would delete a user's projects when that user
  is deleted, which the canonical schema's `ON DELETE NO ACTION` FK (verified
  against `gin-gorm`'s real migration) does not do on any other
  implementation.
- The list endpoint's N+1 guard is **2 queries** (`COUNT` + one
  `select_related("owner")` page `SELECT`, a single `JOIN`), for both a
  non-empty and an empty page — stricter than `gin-gorm`/`gombit`'s pinned 3
  (`COUNT` + page + a separate batched owner `IN (...)`).
  [../../docs/schema.md](../../docs/schema.md) explicitly allows "any
  eager-load strategy that keeps the query count independent of page size...
  a single join, a window function" as long as an implementation documents
  its own count rather than silently diverging while claiming the pinned
  `3` — this is that documentation.
- `django.contrib.auth` and `django.contrib.contenttypes` are in
  `INSTALLED_APPS` even though this app has no login routes and no
  `AUTH_USER_MODEL` of its own — DRF's `request.user` handling
  unconditionally imports `django.contrib.auth.models`, which raises at
  import time if that app isn't registered. This adds `auth_user`,
  `auth_permission`, etc. to the database — bookkeeping tables beyond the
  canonical `users`/`projects` schema, the same kind of harmless extra
  Atlas's own revision-tracking table already is for `gombit`.
  `DEFAULT_AUTHENTICATION_CLASSES`/`DEFAULT_PERMISSION_CLASSES` are set to
  `[]`/`AllowAny` so `/api/projects` itself stays unauthenticated, matching
  every other implementation (issue #141: no cross-framework auth
  comparison on the CRUD apps — Gombit-only auth overhead is Phase 5's own
  benchmark).

## Discovered Django-specific issue: deferred foreign keys and IntegrityError

Django's PostgreSQL backend always emits foreign-key constraints as
`DEFERRABLE INITIALLY DEFERRED` (`django/db/backends/postgresql/operations.py`
`deferrable_sql` — a fixed backend-level choice, not a per-field option),
unlike `gin-gorm`/`gombit`'s plain, immediately-checked FK
(`benchmarks/docs/schema.md`'s canonical FK has no `DEFERRABLE` clause at
all — Postgres's own default, meaning immediate). Outside of any wrapping
transaction this was invisible: a request's `INSERT` runs in its own
implicit autocommit transaction, so Postgres checks the deferred constraint
right there anyway, before that statement finishes.

But inside any wrapping transaction — this app's own test suite (Django's
`TestCase` wraps every test in one) or a production deployment with
`ATOMIC_REQUESTS=True` — a foreign-key violation would **not** raise at
`Project.objects.create()`/`.save()` at all: the write would silently
"succeed" until the enclosing transaction finally commits, and the very next
line (reloading the row via `select_related("owner").get(...)`) would
instead raise `Project.DoesNotExist` — the row's `owner_id` doesn't match any
`users` row, so the `JOIN` drops it — an exception this app has no reason to
expect from reloading a row it just wrote, and not the `IntegrityError`
`_map_integrity_error` (`projects/views.py`) is built to classify.

Verified by reproducing it directly: `test_create_rejects_nonexistent_owner_id`
raised exactly that `Project.DoesNotExist`, not the expected `422`, before
the fix. **Fixed at the schema, not in Python**: an early version of this
app compensated in every view (`transaction.atomic()` +
`connection.check_constraints()` around each write) to force the deferred
check immediately — caught on review as a test-only workaround left on the
production write path this benchmark measures (an extra `BEGIN`/`SET
CONSTRAINTS ALL IMMEDIATE`/`COMMIT` on every `POST`/`PATCH`, never exercised
by gunicorn's real per-request autocommit). Replaced with
`projects/migrations/0002_owner_fk_not_deferrable.py`, which drops Django's
auto-generated `DEFERRABLE INITIALLY DEFERRED` constraint and recreates it
`NOT DEFERRABLE INITIALLY IMMEDIATE` — matching the canonical schema exactly,
with no runtime cost and no view-level workaround. Verified: `psql \d
projects` now shows the FK with no `DEFERRABLE` clause at all, and
`test_create_rejects_nonexistent_owner_id` passes under `TestCase`'s
wrapping transaction with the views doing nothing but a plain
`Project.objects.create()`/`.save()` — the same shape as `gin-gorm`'s own
handlers.

### Under compose (containerized, with the §7 resource budget)

The app is containerized (`Dockerfile`; self-contained, so the build context is
this directory) and wired into `benchmarks/compose.yml` with the §7 app ceiling
(2 vCPU / 1 GiB). The entrypoint has three verbs — `migrate`, `seed`, `serve`
(gunicorn, never runserver). `serve` does **not** migrate, so run them in order.
`migrate` creates the `gombit_bench_django` database if absent (idempotent,
every bring-up, via `ensure_db.py` — not a fresh-volume init script):

```sh
docker compose --env-file benchmarks/config/versions.env \
  -f benchmarks/compose.yml up -d postgres

docker compose --env-file benchmarks/config/versions.env \
  -f benchmarks/compose.yml run --rm django migrate
docker compose --env-file benchmarks/config/versions.env \
  -f benchmarks/compose.yml run --rm django seed

docker compose --env-file benchmarks/config/versions.env \
  -f benchmarks/compose.yml up -d django
go run ./benchmarks/scripts/inspect-limits \
  -container "$(docker compose -f benchmarks/compose.yml ps -q django)" \
  -cpus 2 -memory 1g
```

`GUNICORN_WORKERS` (default 4) is passed to both `--workers` and the pool-split
math in `settings.py`, so they stay in sync.

## Status

Schema, seed, CRUD app, its own test suite, and CI (a Python 3.12 +
`manage.py test` step in `.github/workflows/ci.yml`'s `database-postgres`
job) are done (tracked in
[docs/plans/BENCH-1-benchmark-suite.md](../../../docs/plans/BENCH-1-benchmark-suite.md)
Phase 4). This app now has a `Dockerfile` + `benchmarks/compose.yml` `django`
service with the §7 budget (see [Under compose](#under-compose-containerized-with-the-7-resource-budget)).
Still open: extending the Go `benchmarks/apps/fairness_test.go`
cross-implementation check to include this app as a third leg.
