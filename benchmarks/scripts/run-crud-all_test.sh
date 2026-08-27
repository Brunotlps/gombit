#!/usr/bin/env bash
# Tests for run-crud-all.sh. Sourcing runs the derivation code but not the
# orchestration (main is guarded), so the test can drive run_one with fake
# COMPOSE / inspect_limits / run_crud / wait_healthy and assert the contracts
# the README claims: the framework key matches the harness, the inspect verdict
# is what gets recorded, the SUT is stopped before the next app, and an inspect
# TOOL failure never publishes a blank resource_limits.
#
#   bash benchmarks/scripts/run-crud-all_test.sh
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT"
# shellcheck source=/dev/null
APPS="" source benchmarks/scripts/run-crud-all.sh

fail=0
note() { echo "FAIL: $*" >&2; fail=1; }

# ---- 1. identity tuples: framework == compose service name (the merge key) ----
# port/runtime pinned; framework MUST equal the service name so a run-crud row
# merges under the same key the rest of the harness (and the standalone recipe)
# uses — recording "gin" here would be a second identity for gin-gorm.
check_identity() {
  local app="$1" want_port="$2" want_rt="$3"
  app_identity "$app"
  [ "$FRAMEWORK" = "$app" ] || note "$app: FRAMEWORK=$FRAMEWORK, want $app (the merge key)"
  [ "$PORT" = "$want_port" ] || note "$app: PORT=$PORT, want $want_port"
  [ "$RUNTIME" = "$want_rt" ] || note "$app: RUNTIME=$RUNTIME, want $want_rt"
  case "$FRAMEWORK_VERSION" in [0-9]*|v[0-9]*) : ;; *) note "$app: framework_version '$FRAMEWORK_VERSION' not version-shaped" ;; esac
  [ -n "$RUNTIME_VERSION" ] || note "$app: empty runtime_version"
}
check_identity gin-gorm 8081 go
check_identity gombit   8080 go
check_identity django   8082 python
check_identity rails    8083 ruby
check_identity laravel  8084 php
check_identity nestjs   8085 node

# ---- fakes for driving run_one without Docker / Go ----
CALLS=()
fake_compose() {
  # `ps -q <svc>` must return a container id; everything else is recorded.
  if [ "$1" = ps ]; then echo "cid-$3"; return 0; fi
  CALLS+=("compose $*")
}
called() { printf '%s\n' "${CALLS[@]}" | grep -qF "$1"; }

# ---- 2. happy path: inspect verdict is recorded, SUT stopped ----
CALLS=()
COMPOSE=(fake_compose)
wait_healthy() { CALLS+=("wait_healthy $1"); return 0; }
inspect_limits() { printf 'enforced: cpu 2.00 vCPU, memory 1 GiB'; }
RUN_CRUD_ARGS=""
run_crud() { RUN_CRUD_ARGS="$*"; CALLS+=("run_crud"); }

run_one gin-gorm

called "compose build gin-gorm"      || note "did not build gin-gorm"
called "compose run --rm gin-gorm migrate" || note "did not migrate gin-gorm"
called "compose run --rm gin-gorm seed"    || note "did not seed gin-gorm"
called "wait_healthy cid-gin-gorm"   || note "did not wait for health"
called "run_crud"                    || note "did not run run_crud"
called "compose stop gin-gorm"       || note "did not stop gin-gorm before next"
# the recorded resource_limits is exactly the inspect verdict, and the framework
# key is the service name.
case "$RUN_CRUD_ARGS" in
  *"-framework gin-gorm "*) : ;;
  *) note "run_crud framework is not the merge key gin-gorm: $RUN_CRUD_ARGS" ;;
esac
case "$RUN_CRUD_ARGS" in
  *"-resource-limits enforced: cpu 2.00 vCPU, memory 1 GiB"*) : ;;
  *) note "run_crud did not receive the inspect verdict as -resource-limits: $RUN_CRUD_ARGS" ;;
esac

# ---- 3. inspect TOOL failure must NOT publish a row, but still stop the SUT ----
CALLS=()
RAN_CRUD=0
inspect_limits() { echo "inspect-limits: boom" >&2; return 2; }
run_crud() { RAN_CRUD=1; }
rc=0
run_one gombit || rc=$?
[ "$rc" -ne 0 ]        || note "run_one should fail when inspect-limits fails"
[ "$RAN_CRUD" -eq 0 ]  || note "run_crud ran despite inspect failure (would publish a blank resource_limits)"
called "compose stop gombit" || note "SUT not stopped after inspect failure"

# ---- 4. load_pins respects an overriding environment (Make/CLI override) ----
# A revert to `set -a; . versions.env` (which clobbers exported vars) fails here.
if ! CONCURRENCY=__override__ bash -c '
      APPS="" source benchmarks/scripts/run-crud-all.sh
      [ "$CONCURRENCY" = "__override__" ] || { echo "CONCURRENCY clobbered to $CONCURRENCY" >&2; exit 1; }
      [ -n "$TRIALS" ] || { echo "TRIALS not loaded from the file" >&2; exit 1; }
    '; then
  note "load_pins does not let the environment override versions.env"
fi

if [ "$fail" -ne 0 ]; then
  echo "run-crud-all_test: FAILED" >&2
  exit 1
fi
echo "run-crud-all_test: identity tuples, inspect consumption, stop-before-next, fail-closed inspect, and env override all pass"
