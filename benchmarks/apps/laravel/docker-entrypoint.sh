#!/bin/sh
# laravel container entrypoint. Three verbs so the compose loop drives migrate /
# seed / serve explicitly (benchmarks/apps/laravel/README.md):
#
#   migrate  ensure the target database exists, apply migrations, exit
#   seed     run the deterministic seed (truncate + insert) and exit
#   serve    run php-fpm + nginx (production topology) — does NOT migrate
#
# Laravel needs an APP_KEY to boot; this app has no encrypted sessions/cookies,
# so the key protects nothing and only needs to exist. It commits none, so one
# is generated per boot — nothing insecure is shipped.
set -eu

: "${APP_KEY:=base64:$(head -c 32 /dev/urandom | base64)}"
export APP_KEY

case "${1:-serve}" in
  migrate)
    # ensure_db.php creates the target database if absent (idempotent, every
    # bring-up), so provisioning does not depend on a fresh Postgres volume.
    php /app/ensure_db.php
    exec php artisan migrate --force
    ;;
  seed)
    exec php artisan db:seed --force
    ;;
  serve)
    # php-fpm daemonizes; nginx stays in the foreground as PID 1's child so the
    # container's lifetime tracks the web server.
    php-fpm -D
    exec nginx -g 'daemon off;'
    ;;
  *)
    echo "entrypoint: unknown command '$1' (want: migrate | seed | serve)" >&2
    exit 2
    ;;
esac
