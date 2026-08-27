#!/bin/sh
# nestjs container entrypoint. Three verbs so the compose loop drives migrate /
# seed / serve explicitly (benchmarks/apps/nestjs/README.md):
#
#   migrate  ensure the target database exists, run TypeORM migrations, exit
#   seed     run the deterministic seed (truncate + insert) and exit
#   serve    run the compiled app (node dist/main) — does NOT migrate
#
# migrate/seed use the committed npm scripts (ts-node-driven); serve runs the
# compiled dist directly so node is the process, for clean signal handling.
set -eu

case "${1:-serve}" in
  migrate)
    # ensure_db.js creates the target database if absent (idempotent, every
    # bring-up), so provisioning does not depend on a fresh Postgres volume.
    node ensure_db.js
    exec npm run migration:run
    ;;
  seed)
    exec npm run seed
    ;;
  serve)
    exec node dist/main
    ;;
  *)
    echo "entrypoint: unknown command '$1' (want: migrate | seed | serve)" >&2
    exit 2
    ;;
esac
