#!/usr/bin/env bash
# Tests for footprint-all.sh. The Docker orchestration IS the measurement — when
# the SUT is stopped, and whether a failed/absent load can still publish a
# loaded/CPU row — so beyond the pure to_bytes parser, this drives
# measure_container with fake COMPOSE / run_load / record_footprint / stats
# seams and locks those two contracts. Sourcing runs no real measurement (main
# is guarded).
#
#   bash benchmarks/scripts/footprint-all_test.sh
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT"
# shellcheck source=/dev/null
APPS="" source benchmarks/scripts/footprint-all.sh

fail=0
note() { echo "FAIL: $*" >&2; fail=1; }

# ---- 1. to_bytes conversions ----
check_bytes() {
  local got; got="$(printf '%s' "$1" | to_bytes)"
  [ "$got" = "$2" ] || note "to_bytes($1) = $got, want $2"
}
check_bytes "1GiB"     1073741824
check_bytes "45.66MiB" 47877980
check_bytes "512KiB"   524288
check_bytes "900B"     900
check_bytes "2MB"      2000000
check_bytes "1.5GiB"   1610612736

# ---- fakes shared by the orchestration tests ----
CALLS=()
fake_compose() { if [ "$1" = ps ]; then echo "cid-$3"; return 0; fi; CALLS+=("compose $*"); }
called() { printf '%s\n' "${CALLS[@]}" | grep -qF "$1"; }
# neutralise the slow/real bits
app_identity() { PORT=8081; FRAMEWORK=gin-gorm; FRAMEWORK_VERSION=v1; RUNTIME=go; RUNTIME_VERSION=go1; }
wait_healthy() { return 0; }
cold_start_samples() { echo "100,110,120"; }
mem_bytes() { echo 8000000; }
cpu_pct() { echo 150; }
docker() { echo 22000000; } # only used for `docker image/inspect` size here
IDLE_SETTLE=0

# ---- 2. happy path: records a row (with load numbers) and stops the SUT ----
CALLS=()
COMPOSE=(fake_compose)
run_load() { return 0; }              # clean load
RECORD_ARGS=""
record_footprint() { RECORD_ARGS="$*"; CALLS+=("record"); }
run_one_ok=0
measure_container gin-gorm && run_one_ok=1
[ "$run_one_ok" = 1 ]         || note "happy path returned non-zero"
called "record"               || note "happy path did not record a row"
called "compose stop gin-gorm" || note "happy path did not stop the SUT"
case "$RECORD_ARGS" in *"-loaded-rss-bytes "*"-cpu-percent "*) : ;; *) note "record missing loaded/cpu: $RECORD_ARGS";; esac

# ---- 3. failed load: NO row published, but SUT still stopped ----
CALLS=()
RECORDED=0
run_load() { return 1; }              # dirty/absent load
record_footprint() { RECORDED=1; }
rc=0
measure_container gin-gorm || rc=$?
[ "$rc" -ne 0 ]  || note "failed load should fail the app"
[ "$RECORDED" -eq 0 ] || note "failed load still published a loaded/CPU row"
called "compose stop gin-gorm" || note "SUT not stopped after failed load"

# ---- 4. mid-measure failure (health) still stops the SUT ----
CALLS=()
wait_healthy() { return 1; }
record_footprint() { :; }
run_load() { return 0; }
rc=0
measure_container gin-gorm || rc=$?
[ "$rc" -ne 0 ] || note "unhealthy app should fail"
called "compose stop gin-gorm" || note "SUT not stopped after health failure"

if [ "$fail" -ne 0 ]; then
  echo "footprint-all_test: FAILED" >&2
  exit 1
fi
echo "footprint-all_test: to_bytes, record-on-clean-load, no-row-on-failed-load, and stop-before-return all pass"
