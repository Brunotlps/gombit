#!/bin/sh
# django container entrypoint. Three verbs so the compose loop drives migrate /
# seed / serve explicitly (benchmarks/apps/django/README.md):
#
#   migrate  ensure the target database exists, apply Django migrations, exit
#   seed     run the deterministic seed (truncate + insert) and exit
#   serve    run gunicorn (WSGI, issue #141 §17) — does NOT migrate
#
# gunicorn is started with GUNICORN_WORKERS workers, the same value config/
# settings.py divides POOL_MAX_OPEN by, so the per-worker pool math stays honest
# (the two must match — see the app README's pooling note).
set -eu

case "${1:-serve}" in
  migrate)
    # ensure_db.py creates the target database if absent (idempotent, every
    # bring-up), so provisioning does not depend on a fresh Postgres volume.
    python ensure_db.py
    exec python manage.py migrate
    ;;
  seed)
    exec python manage.py seed
    ;;
  serve)
    exec gunicorn config.wsgi:application \
      --bind "0.0.0.0:${PORT:-8082}" \
      --workers "${GUNICORN_WORKERS:-4}"
    ;;
  *)
    echo "entrypoint: unknown command '$1' (want: migrate | seed | serve)" >&2
    exit 2
    ;;
esac
