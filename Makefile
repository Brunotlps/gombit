# Benchmark developer UX (issue #141 §10). More targets (benchmark-micro,
# -auth, -techempower, -footprint, -report, and the all-in-one benchmark) land
# with their phases; this file starts with the pieces that exist.
#
# Run configuration is sourced from benchmarks/config/versions.env (the single
# source of truth for the pinned k6 version, resource limits, and workload
# defaults); any variable can be overridden on the command line.

BENCH_CONFIG ?= benchmarks/config/versions.env
include $(BENCH_CONFIG)

# Identity of the implementation under test, for the recorded result rows.
FRAMEWORK ?=
FRAMEWORK_VERSION ?=
RUNTIME ?=
RUNTIME_VERSION ?=
# The already-running, already-seeded app's list endpoint.
TARGET_URL ?=
# Where the snapshot is written.
OUT_DIR ?= benchmarks/results/latest

# The pinned intended limits (issue #141 §7). Standalone run-crud does NOT apply
# these — it starts nothing — so it records an honest "not applied" string
# instead; these are only surfaced by benchmark-metadata, labelled as intended.
# benchmark-crud-all is what actually enforces them (compose) and records the
# applied verdict per app (via inspect-limits).
INTENDED_LIMITS ?= intended (applied only under benchmark-crud-all): app $(APP_CPUS)cpu/$(APP_MEMORY); postgres $(POSTGRES_CPUS)cpu/$(POSTGRES_MEMORY)

.PHONY: benchmark-crud benchmark-crud-all benchmark-summary benchmark-metadata

## benchmark-crud-all: bring every containerized implementation up under compose
## (with the §7 limits), migrate + seed it, classify + record the applied limit
## off the live container, run the CRUD-read workload against it, and merge all
## six into OUT_DIR/results.json. Each app is measured alone (stopped before the
## next) since the load generator shares the host. Run `make benchmark-summary`
## afterwards. Override the set with APPS="gin-gorm gombit"; the workload pins
## (CONCURRENCY/TRIALS/DURATION_SECONDS/...) are overridable on the command line.
##
##   make benchmark-crud-all
##   make benchmark-crud-all CONCURRENCY=1 TRIALS=1 DURATION_SECONDS=3   # smoke
benchmark-crud-all:
	OUT_DIR="$(OUT_DIR)" bash benchmarks/scripts/run-crud-all.sh

## benchmark-crud: run the headline CRUD-read workload against one running,
## seeded implementation and write results.json/results.csv/metadata.json.
## Requires the app to be up and seeded (see benchmarks/apps/<name>/README.md);
## `benchmark-crud-all` orchestrates all six under compose instead.
##
##   make benchmark-crud FRAMEWORK=gin-gorm FRAMEWORK_VERSION=v1.11.0 \
##     RUNTIME=go RUNTIME_VERSION=go1.25.7 \
##     TARGET_URL='http://127.0.0.1:8081/api/projects?page=1&limit=20'
benchmark-crud:
	@test -n "$(TARGET_URL)" || { echo "error: set TARGET_URL=<app list endpoint>"; exit 1; }
	@test -n "$(FRAMEWORK)" || { echo "error: set FRAMEWORK=<name>"; exit 1; }
	go run ./benchmarks/scripts/run-crud \
		-target-url "$(TARGET_URL)" \
		-framework "$(FRAMEWORK)" -framework-version "$(FRAMEWORK_VERSION)" \
		-runtime "$(RUNTIME)" -runtime-version "$(RUNTIME_VERSION)" \
		-concurrency "$(CONCURRENCY)" \
		-duration "$(DURATION_SECONDS)s" -warmup "$(WARMUP_SECONDS)s" -trials "$(TRIALS)" \
		-k6-image "grafana/k6:$(K6_VERSION)" \
		-out-dir "$(OUT_DIR)" \
		-postgres-version "$(POSTGRES_IMAGE)"
	@# resource-limits deliberately omitted: run-crud does not start or
	@# constrain the app, so it records its honest "not applied" default
	@# rather than the intended pins this target never enforces.

## benchmark-summary: regenerate the human report (summary.md) from the
## structured OUT_DIR/results.json — per-(framework, concurrency) aggregates
## with trial variance and the >5% coefficient-of-variation flag. Markdown is
## never hand-edited; re-run this after any run-crud that changed results.json.
benchmark-summary:
	go run ./benchmarks/scripts/summarize \
		-results "$(OUT_DIR)/results.json" \
		-out "$(OUT_DIR)/summary.md"

## benchmark-metadata: write just the reproducibility metadata for the current
## host and pinned run configuration to OUT_DIR/metadata.json.
benchmark-metadata:
	@mkdir -p "$(OUT_DIR)"
	go run ./benchmarks/scripts/collect-host-info \
		-out "$(OUT_DIR)/metadata.json" \
		-benchmark-tool "grafana/k6:$(K6_VERSION)" \
		-postgres-version "$(POSTGRES_IMAGE)" \
		-concurrency "$(CONCURRENCY)" \
		-duration-seconds "$(DURATION_SECONDS)" -warmup-seconds "$(WARMUP_SECONDS)" \
		-trials "$(TRIALS)" \
		-resource-limits "$(INTENDED_LIMITS)"
