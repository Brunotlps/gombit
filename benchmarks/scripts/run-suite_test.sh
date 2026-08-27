#!/usr/bin/env bash
# Tests for run-suite.sh (the benchmarks.yml dispatch mapping). Sourcing defines
# the functions but runs nothing (main is guarded), so the test stubs run_make
# and asserts: each SUITE maps to the right make targets, `full` NEVER invokes
# the README-rewriting publish path (benchmark-report / bare benchmark), the
# reduced runner-survivable pins default in (and overrides replace them),
# unset-only pins (APPS/MICRO_COUNT/LOAD_SECONDS) are forwarded only when set,
# OUT_DIR defaults away from results/latest, a pin value is one inert argv token
# (no shell-layer injection), and an unknown suite fails.
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

# ---- 1. SUITE=full -> the three measurement targets + summary, NO report ----
out="$(run_suite SUITE=full)"
for want in benchmark-crud-all benchmark-footprint benchmark-micro benchmark-summary; do
	echo "$out" | grep -qE "^$want \| " || note "full did not run '$want': $out"
done
# The publish path must never run in a dispatch: it rewrites the committed README.
echo "$out" | grep -qE "^benchmark-report \| " && note "full invoked benchmark-report (rewrites README): $out"
echo "$out" | grep -qE "^benchmark \| " && note "full invoked bare 'make benchmark' (the publish recipe): $out"

# ---- 2. reduced runner-survivable pins default in; 500/1000 never appear ----
echo "$out" | grep -qE "CONCURRENCY=1,10,100( |$)" || note "full did not default to reduced concurrency: $out"
echo "$out" | grep -qE "TRIALS=3( |$)" || note "full did not default TRIALS=3: $out"
echo "$out" | grep -qE "1,10,100,500,1000" && note "default dispatch loaded the canonical 500/1000 sweep: $out"

# ---- 3. explicit pins override the reduced defaults ----
out="$(run_suite SUITE=crud CONCURRENCY=1,10,100,500,1000 TRIALS=5)"
echo "$out" | grep -qE "^benchmark-crud-all \| .*CONCURRENCY=1,10,100,500,1000.*TRIALS=5" \
	|| note "explicit pins did not override the defaults: $out"
echo "$out" | grep -qE "^benchmark-summary \| " || note "crud did not also run benchmark-summary: $out"
echo "$out" | grep -qE "CONCURRENCY=1,10,100( |$)" && note "reduced default leaked past an explicit override: $out"

# ---- 4. footprint / micro map to their targets ----
out="$(run_suite SUITE=footprint)"
echo "$out" | grep -qE "^benchmark-footprint \| .*COLD_START_RUNS=5" || note "footprint mapping/default wrong: $out"
out="$(run_suite SUITE=micro MICRO_COUNT=2)"
echo "$out" | grep -qE "^benchmark-micro \| .*MICRO_COUNT=2" || note "micro mapping wrong: $out"

# ---- 5. defaultless pins (MICRO_COUNT/APPS/LOAD_SECONDS) only when set ----
out="$(run_suite SUITE=micro)"
echo "$out" | grep -qE "MICRO_COUNT=|APPS=|LOAD_SECONDS=" \
	&& note "an unset defaultless pin was forwarded as an empty override: $out"

# ---- 6. OUT_DIR always passed and never defaults to results/latest ----
out="$(run_suite SUITE=micro)"
echo "$out" | grep -q "OUT_DIR=benchmarks/results/dispatch" || note "OUT_DIR not defaulted to the dispatch dir: $out"
echo "$out" | grep -q "OUT_DIR=benchmarks/results/latest" && note "a dispatch must not target the committed results/latest: $out"
out="$(run_suite SUITE=micro OUT_DIR=/tmp/x)"
echo "$out" | grep -q "OUT_DIR=/tmp/x" || note "explicit OUT_DIR not honored: $out"

# ---- 7. a malicious pin value is an inert single argv token (shell layer) ----
canary="$(mktemp -d)/pwned"
out="$(run_suite SUITE=micro "MICRO_COUNT=1; touch $canary")"
[ -e "$canary" ] && note "pin value was executed as a shell command (injection)"
echo "$out" | grep -qF "MICRO_COUNT=1; touch $canary" || note "pin value was not passed through as one argv token: $out"
rm -rf "$(dirname "$canary")"

# ---- 8. unknown suite fails ----
rc=0
run_suite SUITE=bogus >/dev/null 2>&1 || rc=$?
[ "$rc" -ne 0 ] || note "unknown suite should fail"

if [ "$fail" -ne 0 ]; then
	echo "run-suite_test: FAILED" >&2
	exit 1
fi
echo "run-suite_test: full-drops-report, reduced-default pins, override, dispatch OUT_DIR, set-only defaultless pins, injection-safety, and unknown-suite all pass"
