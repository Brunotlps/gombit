#!/usr/bin/env bash
# Tests for run-suite.sh (the benchmarks.yml dispatch mapping). Sourcing defines
# the functions but runs nothing (main is guarded), so the test stubs run_make
# and asserts: each SUITE maps to the right make targets, only set pins are
# forwarded (a blank input never becomes `VAR=`), OUT_DIR defaults away from
# results/latest, a pin value is passed as ONE inert argv token (no shell
# injection), and an unknown suite fails.
#
#   bash benchmarks/scripts/run-suite_test.sh
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT"

fail=0
note() { echo "FAIL: $*" >&2; fail=1; }

# Run run-suite's main in a subshell with a recording run_make stub, capturing
# one line per make invocation ("<target> | <arg> | <arg> ..."). Env for the run
# is passed as leading VAR=val assignments.
run_suite() {
	env "$@" bash -c '
		# shellcheck source=/dev/null
		source benchmarks/scripts/run-suite.sh
		run_make() { local IFS=" "; printf "%s | %s\n" "$1" "${*:2}"; }
		main
	'
}

# ---- 1. SUITE=full -> `make benchmark` with OUT_DIR (default dispatch dir) ----
out="$(run_suite SUITE=full)"
case "$out" in
	"benchmark | OUT_DIR=benchmarks/results/dispatch") : ;;
	*) note "full mapping wrong: '$out'" ;;
esac

# ---- 2. SUITE=crud -> crud-all then summary; only set pins forwarded ----
out="$(run_suite SUITE=crud CONCURRENCY=1,10 TRIALS=2)"
echo "$out" | grep -qE "^benchmark-crud-all \| .*CONCURRENCY=1,10.*TRIALS=2" \
	|| note "crud did not forward the set pins: '$out'"
echo "$out" | grep -qE "^benchmark-summary \| " \
	|| note "crud did not also run benchmark-summary: '$out'"
# DURATION_SECONDS / WARMUP_SECONDS were unset -> must NOT appear as empty vars.
echo "$out" | grep -qE "DURATION_SECONDS=|WARMUP_SECONDS=" \
	&& note "crud forwarded an unset pin as an empty override: '$out'"

# ---- 3. SUITE=footprint / micro map to their targets with their pins ----
out="$(run_suite SUITE=footprint COLD_START_RUNS=3)"
echo "$out" | grep -qE "^benchmark-footprint \| .*COLD_START_RUNS=3" \
	|| note "footprint mapping wrong: '$out'"
out="$(run_suite SUITE=micro MICRO_COUNT=2)"
echo "$out" | grep -qE "^benchmark-micro \| .*MICRO_COUNT=2" \
	|| note "micro mapping wrong: '$out'"

# ---- 4. OUT_DIR is always passed and never defaults to results/latest ----
out="$(run_suite SUITE=micro)"
echo "$out" | grep -q "OUT_DIR=benchmarks/results/dispatch" \
	|| note "OUT_DIR not defaulted to the dispatch dir: '$out'"
echo "$out" | grep -q "OUT_DIR=benchmarks/results/latest" \
	&& note "a dispatch must not target the committed results/latest: '$out'"
# An explicit OUT_DIR override is honored.
out="$(run_suite SUITE=micro OUT_DIR=/tmp/x)"
echo "$out" | grep -q "OUT_DIR=/tmp/x" || note "explicit OUT_DIR not honored: '$out'"

# ---- 5. a malicious pin value is an inert single make token, not a command ----
canary="$(mktemp -d)/pwned"
out="$(run_suite SUITE=micro "MICRO_COUNT=1; touch $canary")"
[ -e "$canary" ] && note "pin value was executed as a shell command (injection)"
echo "$out" | grep -qF "MICRO_COUNT=1; touch $canary" \
	|| note "pin value was not passed through as one make token: '$out'"
rm -rf "$(dirname "$canary")"

# ---- 6. unknown suite fails ----
rc=0
run_suite SUITE=bogus >/dev/null 2>&1 || rc=$?
[ "$rc" -ne 0 ] || note "unknown suite should fail"

if [ "$fail" -ne 0 ]; then
	echo "run-suite_test: FAILED" >&2
	exit 1
fi
echo "run-suite_test: suite mapping, set-only pin forwarding, dispatch OUT_DIR, injection-safety, and unknown-suite all pass"
