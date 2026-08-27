#!/bin/sh
# gin-gorm container entrypoint. Three verbs so the compose loop can drive every
# app the same way (benchmarks/apps/gin-gorm/README.md):
#
#   migrate  no-op: this app uses GORM AutoMigrate (run on seed/serve) and its
#            database is the postgres service's POSTGRES_DB, so there is no
#            separate migrate or database-create step — the verb exists only so
#            the orchestrator's uniform migrate/seed/serve works here too
#   seed     run the deterministic seed (truncate + insert) and exit
#   serve    run the API (default)
set -eu

case "${1:-serve}" in
  migrate)
    echo "gin-gorm: schema is AutoMigrated on seed/serve; nothing to migrate"
    ;;
  seed)
    exec gin-gorm -seed
    ;;
  serve)
    exec gin-gorm
    ;;
  *)
    echo "entrypoint: unknown command '$1' (want: migrate | seed | serve)" >&2
    exit 2
    ;;
esac
