#!/bin/sh
# gin-gorm container entrypoint. Two verbs, matching the app's own two modes
# (benchmarks/apps/gin-gorm/README.md) so the compose loop controls seeding
# explicitly rather than reseeding 100k rows on every serve start:
#
#   seed    run the deterministic seed (truncate + insert) and exit
#   serve   run the API (default)
set -eu

case "${1:-serve}" in
  seed)
    exec gin-gorm -seed
    ;;
  serve)
    exec gin-gorm
    ;;
  *)
    echo "entrypoint: unknown command '$1' (want: seed | serve)" >&2
    exit 2
    ;;
esac
