#!/usr/bin/env bash
# Map a dispatch SUITE to the make targets that produce its artifacts, for the
# manual .github/workflows/benchmarks.yml (workflow_dispatch). Kept as a tested
# script rather than inline YAML so the suite -> target mapping, the
# forward-only-what-is-set pin handling, and injection safety are locked by
# run-suite_test.sh without a real run.
#
# Inputs come from the environment (the workflow binds workflow inputs to env,
# never interpolating ${{ inputs.* }} into a shell string):
#   SUITE      full | crud | footprint | micro   (default full)
#   OUT_DIR    where artifacts are written (default benchmarks/results/dispatch,
#              deliberately NOT results/latest so a dispatch never overwrites the
#              committed snapshot even on the runner checkout)
#   CONCURRENCY TRIALS DURATION_SECONDS WARMUP_SECONDS APPS   (crud pins)
#   COLD_START_RUNS LOAD_SECONDS                               (footprint pins)
#   MICRO_COUNT                                                (micro samples)
# Any pin left empty falls back to versions.env — it is only forwarded to make
# when set, so a blank workflow input never becomes `CONCURRENCY=`.
#
#   SUITE=crud CONCURRENCY=1,10 bash benchmarks/scripts/run-suite.sh
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT"

SUITE="${SUITE:-full}"
OUT_DIR="${OUT_DIR:-benchmarks/results/dispatch}"

# Seam so the test can drive the mapping without invoking a real (Docker-bound,
# hours-long) make.
run_make() { make "$@"; }

# make_args echoes the make variable overrides for this run, one per line: always
# OUT_DIR, plus every pin that is actually set. NUL-safe not needed — make
# variable values here are simple tokens, and each is passed as ONE argv element
# by the caller's array read, so a value like "1; rm -rf /" is an inert
# make-variable string, never a shell command.
make_args() {
	printf '%s\n' "OUT_DIR=${OUT_DIR}"
	local name
	for name in CONCURRENCY TRIALS DURATION_SECONDS WARMUP_SECONDS APPS \
		COLD_START_RUNS LOAD_SECONDS MICRO_COUNT; do
		if [ -n "${!name:-}" ]; then
			printf '%s=%s\n' "$name" "${!name}"
		fi
	done
}

main() {
	local args=()
	local line
	while IFS= read -r line; do
		args+=("$line")
	done < <(make_args)

	case "$SUITE" in
	full)
		run_make benchmark "${args[@]}"
		;;
	crud)
		run_make benchmark-crud-all "${args[@]}"
		run_make benchmark-summary "${args[@]}"
		;;
	footprint)
		run_make benchmark-footprint "${args[@]}"
		;;
	micro)
		run_make benchmark-micro "${args[@]}"
		;;
	*)
		echo "run-suite: unknown SUITE '$SUITE' (want: full | crud | footprint | micro)" >&2
		exit 2
		;;
	esac
}

# Only orchestrate when executed directly; sourcing (the test) just defines the
# functions so it can stub run_make.
if [ "${BASH_SOURCE[0]}" = "${0}" ]; then
	main "$@"
fi
