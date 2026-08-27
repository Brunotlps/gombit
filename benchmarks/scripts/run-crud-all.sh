#!/usr/bin/env bash
# Orchestrate the headline CRUD-read workload across all six containerized
# implementations and merge the result into one results.json.
#
# For each app it: builds the image, applies migrations and seeds (the app's
# own `migrate`/`seed` verbs, which create their database idempotently), brings
# the service up under its §7 cgroup budget, waits for /livez to report healthy,
# reads the *actually applied* limit off the live container
# (benchmarks/scripts/inspect-limits), runs the per-implementation measurement
# engine (benchmarks/scripts/run-crud) against it recording that honest limit,
# then stops the service so the next app is measured alone with Postgres — the
# load generator shares the host, so only the app under test should be running.
#
# run-crud merges by framework key, so the six iterations accumulate into
# benchmarks/results/latest/{results.json,results.csv,metadata.json}. Regenerate
# the human report afterwards with `make benchmark-summary`.
#
#   make benchmark-crud-all               # all six
#   APPS="gin-gorm gombit" make benchmark-crud-all   # a subset
#
# Versions are DERIVED from each app's own source-of-truth (framework manifest
# file, Dockerfile base image) and the run fails closed if any comes back empty
# — nothing about the recorded identity is hand-copied here.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT"

CONFIG="benchmarks/config/versions.env"
# shellcheck disable=SC1090
set -a; . "$CONFIG"; set +a

COMPOSE=(docker compose --env-file "$CONFIG" -f benchmarks/compose.yml)
OUT_DIR="${OUT_DIR:-benchmarks/results/latest}"
# Which apps to run (default: all six, in a stable order).
APPS="${APPS:-gin-gorm gombit django rails laravel nestjs}"

# req NAME VALUE — echo VALUE, or abort if it is empty (fail closed so a broken
# extractor never records a blank version).
req() {
  if [ -z "${2:-}" ]; then
    echo "run-crud-all: could not determine $1" >&2
    exit 1
  fi
  printf '%s' "$2"
}

# base_tag APP LANG — the base image tag from the app's Dockerfile `FROM
# <lang>:<tag>` line, with any -slim/-alpine/-fpm suffix stripped (3.12-slim ->
# 3.12, 1.25.7-alpine -> 1.25.7).
base_tag() {
  local tag
  tag="$(sed -nE "s/^FROM ${2}:([^ ]+).*/\1/p" "benchmarks/apps/${1}/Dockerfile" | head -1)"
  printf '%s' "${tag%%-*}"
}

# Per-app identity, derived. Sets: PORT FRAMEWORK FRAMEWORK_VERSION RUNTIME
# RUNTIME_VERSION for the given app.
app_identity() {
  case "$1" in
    gin-gorm)
      PORT=8081; FRAMEWORK=gin; RUNTIME=go
      FRAMEWORK_VERSION="$(req gin "$(awk '/gin-gonic\/gin /{print $2; exit}' go.mod)")"
      RUNTIME_VERSION="go$(req go "$(base_tag gin-gorm golang)")" ;;
    gombit)
      PORT=8080; FRAMEWORK=gombit; RUNTIME=go
      FRAMEWORK_VERSION="$(req gombit "$(git describe --tags --always --dirty)")"
      RUNTIME_VERSION="go$(req go "$(base_tag gombit golang)")" ;;
    django)
      PORT=8082; FRAMEWORK=django; RUNTIME=python
      FRAMEWORK_VERSION="$(req django "$(sed -nE 's/^Django==([0-9.]+).*/\1/p' benchmarks/apps/django/requirements.txt)")"
      RUNTIME_VERSION="$(req python "$(base_tag django python)")" ;;
    rails)
      PORT=8083; FRAMEWORK=rails; RUNTIME=ruby
      FRAMEWORK_VERSION="$(req rails "$(sed -nE 's/^ *rails \(([0-9.]+)\).*/\1/p' benchmarks/apps/rails/Gemfile.lock | head -1)")"
      RUNTIME_VERSION="$(req ruby "$(base_tag rails ruby)")" ;;
    laravel)
      PORT=8084; FRAMEWORK=laravel; RUNTIME=php
      FRAMEWORK_VERSION="$(req laravel "$(sed -nE 's/.*"laravel\/framework": "([0-9.]+)".*/\1/p' benchmarks/apps/laravel/composer.json)")"
      RUNTIME_VERSION="$(req php "$(base_tag laravel php)")" ;;
    nestjs)
      PORT=8085; FRAMEWORK=nestjs; RUNTIME=node
      FRAMEWORK_VERSION="$(req nestjs "$(sed -nE 's/.*"@nestjs\/core": "([0-9.]+)".*/\1/p' benchmarks/apps/nestjs/package.json)")"
      RUNTIME_VERSION="$(req node "$(base_tag nestjs node)")" ;;
    *)
      echo "run-crud-all: unknown app '$1'" >&2; exit 1 ;;
  esac
}

# wait_healthy CONTAINER — block until the container's health check passes.
wait_healthy() {
  local cid="$1" i status
  for i in $(seq 1 60); do
    status="$(docker inspect --format '{{.State.Health.Status}}' "$cid" 2>/dev/null || echo missing)"
    case "$status" in
      healthy) return 0 ;;
      unhealthy|missing) echo "run-crud-all: $cid is $status" >&2; return 1 ;;
    esac
    sleep 2
  done
  echo "run-crud-all: $cid did not become healthy in time" >&2
  return 1
}

# run_one APP — the full measured cycle for one implementation.
run_one() {
  local app="$1" cid limits
  app_identity "$app"
  echo "=== $app ($FRAMEWORK $FRAMEWORK_VERSION on $RUNTIME $RUNTIME_VERSION, :$PORT) ==="

  "${COMPOSE[@]}" build "$app" >/dev/null
  "${COMPOSE[@]}" run --rm "$app" migrate
  "${COMPOSE[@]}" run --rm "$app" seed
  "${COMPOSE[@]}" up -d "$app" >/dev/null

  cid="$("${COMPOSE[@]}" ps -q "$app")"
  wait_healthy "$cid"

  # The honest, applied limit off the live container — recorded as-is.
  limits="$(go run ./benchmarks/scripts/inspect-limits \
    -container "$cid" -cpus "$APP_CPUS" -memory "$APP_MEMORY" || true)"
  echo "run-crud-all: $app limits: $limits"

  go run ./benchmarks/scripts/run-crud \
    -target-url "http://127.0.0.1:${PORT}/api/projects?page=1&limit=20" \
    -framework "$FRAMEWORK" -framework-version "$FRAMEWORK_VERSION" \
    -runtime "$RUNTIME" -runtime-version "$RUNTIME_VERSION" \
    -concurrency "$CONCURRENCY" \
    -duration "${DURATION_SECONDS}s" -warmup "${WARMUP_SECONDS}s" -trials "$TRIALS" \
    -k6-image "grafana/k6:$K6_VERSION" \
    -out-dir "$OUT_DIR" \
    -postgres-version "$POSTGRES_IMAGE" \
    -resource-limits "$limits"

  # Free CPU/RAM for the next app (only the app under test + Postgres should run
  # while k6 — sharing this host — is measuring).
  "${COMPOSE[@]}" stop "$app" >/dev/null
}

main() {
  echo "run-crud-all: ensuring postgres is up"
  "${COMPOSE[@]}" up -d postgres >/dev/null
  for app in $APPS; do
    run_one "$app"
  done
  echo "run-crud-all: done. Results in $OUT_DIR; run 'make benchmark-summary' to render summary.md."
}

# Only run the orchestration when executed directly; sourcing (e.g. from the
# test) just defines the functions above.
if [ "${BASH_SOURCE[0]}" = "${0}" ]; then
  main "$@"
fi
