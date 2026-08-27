#!/usr/bin/env bash
# Smoke test for run-crud-all.sh's version derivation: sourcing must define the
# functions (the main loop is guarded), and every app's identity must resolve to
# a non-empty framework/runtime version from its own source-of-truth files. This
# is what catches an app rename or a moved manifest before a real run does.
#
#   bash benchmarks/scripts/run-crud-all_test.sh
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT"

# Sourcing runs the derivation code but not the orchestration (main guard).
APPS="" source benchmarks/scripts/run-crud-all.sh

fail=0
for app in gin-gorm gombit django rails laravel nestjs; do
  app_identity "$app"
  for pair in "PORT=$PORT" "FRAMEWORK=$FRAMEWORK" "FRAMEWORK_VERSION=$FRAMEWORK_VERSION" \
              "RUNTIME=$RUNTIME" "RUNTIME_VERSION=$RUNTIME_VERSION"; do
    if [ -z "${pair#*=}" ]; then
      echo "FAIL: $app has empty ${pair%%=*}" >&2
      fail=1
    fi
  done
  # A version should start with a digit or 'v' — a stray label would sail past
  # the non-empty check but is not a version.
  case "$FRAMEWORK_VERSION" in
    [0-9]*|v[0-9]*) : ;;
    *) echo "FAIL: $app framework_version '$FRAMEWORK_VERSION' is not version-shaped" >&2; fail=1 ;;
  esac
  echo "ok: $app -> $FRAMEWORK $FRAMEWORK_VERSION / $RUNTIME $RUNTIME_VERSION (:$PORT)"
done

if [ "$fail" -ne 0 ]; then
  echo "run-crud-all_test: FAILED" >&2
  exit 1
fi
echo "run-crud-all_test: all six identities resolve"
