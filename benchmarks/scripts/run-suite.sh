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
#
# The workload pins default to a REDUCED, shared-runner-survivable protocol, NOT
# the versions.env canonical sweep (1/10/100/500/1000 x 5 x 30s): this repo could
# not sustain 500/1000 VUs on a stronger single host than a GitHub-hosted runner,
# and run-crud fail-closes the whole run on any HTTP error. A dispatch aimed at a
# noisy 4-vCPU runner must not default to the dedicated-host recipe. Set any pin
# explicitly (e.g. CONCURRENCY=1,10,100,500,1000) for a wider run on a dedicated
# runner. Pins with no reduced default (APPS, MICRO_COUNT, LOAD_SECONDS) are only
# forwarded when set.
#
#   SUITE=crud CONCURRENCY=1,10 bash benchmarks/scripts/run-suite.sh
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT"

SUITE="${SUITE:-full}"
OUT_DIR="${OUT_DIR:-benchmarks/results/dispatch}"

# Reduced, runner-survivable defaults (the committed results/latest snapshot was
# taken at exactly this protocol on a stronger host — the existence proof).
CONCURRENCY="${CONCURRENCY:-1,10,100}"
TRIALS="${TRIALS:-3}"
DURATION_SECONDS="${DURATION_SECONDS:-10}"
WARMUP_SECONDS="${WARMUP_SECONDS:-3}"
COLD_START_RUNS="${COLD_START_RUNS:-5}"

# Seam so the test can drive the mapping without invoking a real (Docker-bound,
# hours-long) make.
run_make() { make "$@"; }

# make_args echoes the make variable overrides for this run, one per line: always
# OUT_DIR and the reduced pins above, plus APPS/MICRO_COUNT/LOAD_SECONDS when set.
# Each is passed as ONE argv element by the caller's array read, so the workflow
# env -> argv path cannot inject a shell command (a value like "1; rm -rf /" is
# an inert argv token). NOTE: this is the shell-layer safety only — it is not a
# claim about GNU make's own `$(shell ...)` expansion of a command-line variable;
# the exposed inputs feed run-crud flags, not a make `$(shell)` sink, and
# workflow_dispatch already requires write access.
make_args() {
	printf '%s\n' "OUT_DIR=${OUT_DIR}"
	printf '%s\n' "CONCURRENCY=${CONCURRENCY}" "TRIALS=${TRIALS}" \
		"DURATION_SECONDS=${DURATION_SECONDS}" "WARMUP_SECONDS=${WARMUP_SECONDS}" \
		"COLD_START_RUNS=${COLD_START_RUNS}"
	local name
	for name in APPS LOAD_SECONDS MICRO_COUNT; do
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
		# The three MEASUREMENT targets plus summary — deliberately NOT
		# `make benchmark`, whose last stage (benchmark-report) rewrites the
		# published repo-root README.md from OUT_DIR. A dispatch produces
		# artifacts; regenerating the committed README stays the reviewed local
		# `make benchmark` + commit. OUT_DIR alone does not protect the README —
		# dropping the report writer does.
		run_make benchmark-crud-all "${args[@]}"
		run_make benchmark-footprint "${args[@]}"
		run_make benchmark-micro "${args[@]}"
		run_make benchmark-summary "${args[@]}"
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
