#!/bin/sh
# gombit container entrypoint. Three verbs so the compose loop drives migrate /
# seed / serve explicitly (benchmarks/apps/gombit/README.md):
#
#   migrate  apply Atlas migrations (`gombit db migrate`) and exit
#   seed     run the deterministic seed (truncate + insert) and exit
#   serve    run the API (default)
#
# The app binary reads DATABASE_URL; the `gombit` CLI reads
# GOMBIT_DATABASE_DRIVER/GOMBIT_DATABASE_DSN, so `migrate` maps the one
# DATABASE_URL a caller sets onto the CLI's env — one variable configures all
# three verbs.
set -eu

case "${1:-serve}" in
  migrate)
    export GOMBIT_DATABASE_DRIVER=postgres
    export GOMBIT_DATABASE_DSN="${DATABASE_URL}"
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
