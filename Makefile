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

RESOURCE_LIMITS ?= app $(APP_CPUS)cpu/$(APP_MEMORY); postgres $(POSTGRES_CPUS)cpu/$(POSTGRES_MEMORY)

.PHONY: benchmark-crud benchmark-metadata

## benchmark-crud: run the headline CRUD-read workload against one running,
## seeded implementation and write results.json/results.csv/metadata.json.
## Requires the app to be up and seeded (see benchmarks/apps/<name>/README.md);
## the full "bring up all six under compose" loop is a later slice.
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
		-postgres-version "$(POSTGRES_IMAGE)" \
		-resource-limits "$(RESOURCE_LIMITS)"

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
		-resource-limits "$(RESOURCE_LIMITS)"
