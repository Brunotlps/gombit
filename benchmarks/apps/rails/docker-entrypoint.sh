#!/bin/sh
# rails container entrypoint. Three verbs so the compose loop drives migrate /
# seed / serve explicitly (benchmarks/apps/rails/README.md):
#
#   migrate  create the database if absent, apply migrations, exit
#   seed     run the deterministic seed (truncate + insert) and exit
#   serve    run Puma in production (clustered, config/puma.rb) — does NOT migrate
#
# All verbs boot Rails in production, which refuses to start without
# SECRET_KEY_BASE. This app commits no throwaway secret (unlike ../django's
# placeholder), so one is generated per boot — a benchmark keeps no sessions
# across restarts, so an ephemeral key is fine and nothing insecure is shipped.
set -eu

export RAILS_ENV=production
: "${SECRET_KEY_BASE:=$(ruby -rsecurerandom -e 'print SecureRandom.hex(64)')}"
export SECRET_KEY_BASE

case "${1:-serve}" in
  migrate)
    # db:create is idempotent (prints "already exists" and exits 0), so this
    # provisions the database on every bring-up without a fresh-volume init
    # script or `down -v` — Rails creates it via the DATABASE_URL credentials.
    exec bin/rails db:create db:migrate
    ;;
  seed)
    exec bin/rails db:seed
    ;;
  serve)
    exec bin/rails server -b 0.0.0.0 -p "${PORT:-8083}"
    ;;
  *)
    echo "entrypoint: unknown command '$1' (want: migrate | seed | serve)" >&2
    exit 2
    ;;
esac
