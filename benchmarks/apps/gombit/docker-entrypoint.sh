#!/bin/sh
# gombit container entrypoint. Three verbs so the compose loop drives migrate /
# seed / serve explicitly (benchmarks/apps/gombit/README.md):
#
#   migrate  ensure the target database exists, apply Atlas migrations, exit
#   seed     run the deterministic seed (truncate + insert) and exit
#   serve    run the API (default) — does NOT migrate; run `migrate` first
#
# The app binary reads DATABASE_URL; the `gombit` CLI reads
# GOMBIT_DATABASE_DRIVER/GOMBIT_DATABASE_DSN, so `migrate` maps the one
# DATABASE_URL a caller sets onto the CLI's env — one variable configures all
# three verbs.
set -eu

# ensure_database creates the target database if it is absent. This runs as part
# of every `migrate`, so provisioning does NOT depend on a fresh Postgres volume
# (unlike docker-entrypoint-initdb.d, which runs only once on an empty data
# dir) and never requires `down -v` — which would wipe the sibling app's data.
# ADMIN_DATABASE_URL points at an existing database on the same server (the
# maintenance connection); TARGET_DB is the database to ensure. If either is
# unset (e.g. a hand-provisioned local DB) the step is skipped.
ensure_database() {
  [ -n "${ADMIN_DATABASE_URL:-}" ] && [ -n "${TARGET_DB:-}" ] || return 0
  if psql "${ADMIN_DATABASE_URL}" -tAc \
      "SELECT 1 FROM pg_database WHERE datname = '${TARGET_DB}'" | grep -q 1; then
    return 0
  fi
  echo "entrypoint: creating database ${TARGET_DB}" >&2
  # CREATE DATABASE has no IF NOT EXISTS; the catalog check above guards it.
  psql "${ADMIN_DATABASE_URL}" -v ON_ERROR_STOP=1 -c "CREATE DATABASE \"${TARGET_DB}\""
}

case "${1:-serve}" in
  migrate)
    export GOMBIT_DATABASE_DRIVER=postgres
    export GOMBIT_DATABASE_DSN="${DATABASE_URL}"
    ensure_database
    # WORKDIR is /app, which holds database/migrations (the CLI's default --dir).
    exec gombit db migrate
    ;;
  seed)
    exec gombit-app -seed
    ;;
  serve)
    exec gombit-app
    ;;
  *)
    echo "entrypoint: unknown command '$1' (want: migrate | seed | serve)" >&2
    exit 2
    ;;
esac
