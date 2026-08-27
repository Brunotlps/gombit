#!/usr/bin/env bash
# Tests for footprint-all.sh. The Docker orchestration IS the measurement, so
# beyond the pure to_bytes parser this locks the two contracts a bad row hides
# behind: the load aggregator refuses a too-short sample series (no 0-byte
# "loaded" sentinel), and measure_container records the real loaded/CPU values
# on a clean load, publishes nothing on a failed one, and always stops the SUT.
# Sourcing runs no real measurement (main is guarded).
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

# ---- 2. aggregate_load_samples: fail-closed on a too-short series ----
sample_file() { local f; f="$(mktemp)"; printf '%b' "$1" > "$f"; echo "$f"; }

for n_lines in "" "10 1\n" "10 1\n20 2\n"; do
  f="$(sample_file "$n_lines")"
  if aggregate_load_samples "$f" >/dev/null 2>&1; then
    note "aggregate_load_samples accepted a <2-in-load-sample series ($(printf '%b' "$n_lines" | wc -l) lines)"
  fi
  rm -f "$f"
done
# Three samples -> drop the ramp, median of the last two.
f="$(sample_file '10 1\n20 2\n30 3\n')"
got="$(aggregate_load_samples "$f")"
[ "$got" = "25 2.5" ] || note "aggregate_load_samples(3 samples) = '$got', want '25 2.5'"
rm -f "$f"

# ---- static: image size is inspected by TAG (.Config.Image), never the
#      resolved image ID (.Image). measure_container rebuilds the image, which
#      retags to a new ID and can leave the old ID a reused container pins
#      dangling; `docker image inspect <old-id>` then fails "No such image" and
#      aborts the whole run after a clean load (seen on a real canonical run).
#      The runtime docker is stubbed below, so this source check is what locks it.
if grep -qE "docker inspect [^|]*'\{\{\.Image\}\}'" benchmarks/scripts/footprint-all.sh; then
  note "image-size inspects the resolved .Image ID (dangles after rebuild); use .Config.Image (the tag)"
fi
grep -qF '{{.Config.Image}}' benchmarks/scripts/footprint-all.sh \
  || note "image-size no longer inspects the app image by tag (.Config.Image)"

# ---- now_ms is monotonic + non-negative: elapsed timing can't go backward on a
#      wall-clock step (that's why cold starts aren't timed with date +%s%3N,
#      whose two reads produced -1609081096361592473 on a real run) ----
a="$(now_ms)"; sleep 0.02; b="$(now_ms)"
[[ "$a" =~ ^[0-9]+$ ]] || note "now_ms is not a non-negative integer: '$a'"
[ "$b" -ge "$a" ] 2>/dev/null || note "now_ms went backwards: $a -> $b"

# ---- cold_start_samples: a malformed sample is rejected, and the fix REDOES the
#      stop/start (a fresh cold start), never a warm re-poll of the up container;
#      it neither records garbage nor fail-closes the whole run on a transient
#      one. The COMPOSE recorder counts start calls so the restart invariant is
#      actually tested, not just the value. ----
COLD_START_RUNS=1
PORT=8081
cs_starts=0
cs_count="$(mktemp)"
# COMPOSE recorder runs in THIS shell (not a $()), so cs_starts survives.
cs_rec() { case "$1" in start) cs_starts=$((cs_starts + 1)) ;; esac; return 0; }
COMPOSE=(cs_rec)
# wait_ready_ms runs in its own $() each call, so it counts via a file.
wait_ready_ms() {
  local n; n=$(( $(cat "$cs_count") + 1 )); echo "$n" > "$cs_count"
  if [ "$n" -eq 1 ]; then echo "-1609081096361592473"; else echo "120"; fi
}

# One malformed reading then a good one: retries to 120 AND restarts twice.
echo 0 > "$cs_count"; cs_starts=0; csout="$(mktemp)"
cold_start_samples gin-gorm http://x/livez > "$csout" 2>/dev/null || note "cold_start_samples failed on a recoverable sample"
[ "$(cat "$csout")" = "120" ] || note "cold_start_samples did not retry to the good sample: '$(cat "$csout")'"
[ "$cs_starts" -eq 2 ] || note "a rejected sample did not trigger a second stop/start (starts=$cs_starts, want 2)"

# Persistently malformed: fails, restarts all 5 attempts, records nothing.
wait_ready_ms() { echo "-1609081096361592473"; }
cs_starts=0
if cold_start_samples gin-gorm http://x/livez >/dev/null 2>&1; then
  note "cold_start_samples recorded a persistently malformed sample instead of failing"
fi
[ "$cs_starts" -eq 5 ] || note "persistent-malformed path did not restart 5 times (starts=$cs_starts, want 5)"

# A huge positive (ms-minus-ns shape) is rejected too.
wait_ready_ms() { echo "1609081096361592473"; }
if cold_start_samples gin-gorm http://x/livez >/dev/null 2>&1; then
  note "cold_start_samples accepted an absurdly large sample"
fi
rm -f "$cs_count" "$csout"

# ---- fakes for the measure_container control-flow tests ----
CALLS=()
fake_compose() { if [ "$1" = ps ]; then echo "cid-$3"; return 0; fi; CALLS+=("compose $*"); }
called() { printf '%s\n' "${CALLS[@]}" | grep -qF "$1"; }
app_identity() { PORT=8081; FRAMEWORK=gin-gorm; FRAMEWORK_VERSION=v1; RUNTIME=go; RUNTIME_VERSION=go1; }
wait_healthy() { return 0; }
cold_start_samples() { echo "100,110,120"; }
mem_bytes() { echo 5000000; }
docker() { echo 22000000; } # stands in for the image-size inspects
IDLE_SETTLE=0

# ---- 3. clean load: records the REAL loaded/CPU values, stops the SUT ----
CALLS=()
COMPOSE=(fake_compose)
measure_under_load() { printf '8000000 150'; }   # a clean, well-sampled load
RECORD_ARGS=""
record_footprint() { RECORD_ARGS="$*"; CALLS+=("record"); }
ok=0
measure_container gin-gorm && ok=1
[ "$ok" = 1 ]                 || note "clean-load path returned non-zero"
called "record"              || note "clean-load path did not record a row"
called "compose stop gin-gorm" || note "clean-load path did not stop the SUT"
case "$RECORD_ARGS" in
  *"-loaded-rss-bytes 8000000 "*"-cpu-percent 150 "*) : ;;
  *) note "record did not carry the real loaded/CPU values: $RECORD_ARGS" ;;
esac

# ---- 4. failed load (or too-few samples): NO row, SUT still stopped ----
CALLS=()
RECORDED=0
measure_under_load() { return 1; }
record_footprint() { RECORDED=1; }
rc=0
measure_container gin-gorm || rc=$?
[ "$rc" -ne 0 ]        || note "failed load should fail the app"
[ "$RECORDED" -eq 0 ]  || note "failed load still published a loaded/CPU row"
called "compose stop gin-gorm" || note "SUT not stopped after failed load"

# ---- 5. mid-measure failure (health) still stops the SUT ----
CALLS=()
wait_healthy() { return 1; }
record_footprint() { :; }
measure_under_load() { printf '8000000 150'; }
rc=0
measure_container gin-gorm || rc=$?
[ "$rc" -ne 0 ] || note "unhealthy app should fail"
called "compose stop gin-gorm" || note "SUT not stopped after health failure"

# ---- 6. main runs to a clean exit under set -u (the EXIT-trap bindir bug) ----
# main is guarded, so drive the real script as a subprocess with docker/go
# stubbed. APPS=" " (a single space) stays set — so ${APPS:-default} does not
# re-expand to the six apps — but word-splits to nothing, so the measured loop
# is empty and main still reaches the EXIT trap. A `local bindir` (unbound in the
# trap's global scope) makes `set -u` abort with exit 1 there.
shimdir="$(mktemp -d)"
printf '#!/bin/sh\nexit 0\n' > "$shimdir/docker"
printf '#!/bin/sh\nexit 0\n' > "$shimdir/go"
chmod +x "$shimdir/docker" "$shimdir/go"
if PATH="$shimdir:$PATH" APPS=" " OUT_DIR="$(mktemp -d)" \
     bash benchmarks/scripts/footprint-all.sh >/dev/null 2>&1; then
  : # exit 0 — trap cleaned up without an unbound-variable abort
else
  note "footprint-all.sh main did not exit 0 under set -u (EXIT-trap bindir regression)"
fi
rm -rf "$shimdir"

if [ "$fail" -ne 0 ]; then
  echo "footprint-all_test: FAILED" >&2
  exit 1
fi
echo "footprint-all_test: to_bytes, fail-closed aggregator, real-value record, no-row-on-failure, and stop-before-return all pass"
