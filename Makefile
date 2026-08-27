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

.PHONY: benchmark-crud benchmark-crud-all benchmark-micro benchmark-footprint benchmark-summary benchmark-metadata benchmark-report benchmark-report-check

# Framework-tax microbenchmark iterations (-count). The last iteration per
# scenario wins; more iterations warm the numbers, they don't add variance rows.
MICRO_COUNT ?= 10

## benchmark-micro: run the framework-tax microbenchmark (net/http -> Gin ->
## Huma -> Gombit) and persist ns/op / B/op / allocs/op to OUT_DIR/microbench.json
## for the report. Each stack is its own `go test` process (a framework.App
## constructor mutates a process global), piped through the microbench parser.
benchmark-micro:
	bash -c 'set -euo pipefail; for s in nethttp gin huma gombit; do \
		go test ./benchmarks/micro/$$s -bench=BenchmarkFrameworkTax -benchmem -run="^$$" -count=$(MICRO_COUNT) \
			| go run ./benchmarks/scripts/microbench -stack $$s -out "$(OUT_DIR)/microbench.json"; \
	done'

## benchmark-report: regenerate the derived Markdown from OUT_DIR — the root
## README's `## Performance` block and summary.md. Markdown is generated, never
## hand-edited (issue #141 §9); benchmark-report-check fails on drift (for CI).
benchmark-report:
	@if [ -f "$(OUT_DIR)/results.json" ]; then \
		go run ./benchmarks/scripts/summarize -results "$(OUT_DIR)/results.json" -out "$(OUT_DIR)/summary.md"; \
	else echo "benchmark-report: no $(OUT_DIR)/results.json yet; skipping summary.md"; fi
	go run ./benchmarks/scripts/report \
		-results "$(OUT_DIR)/results.json" -footprint "$(OUT_DIR)/footprint.json" \
		-micro "$(OUT_DIR)/microbench.json" -metadata "$(OUT_DIR)/metadata.json"

## benchmark-report-check: fail if the committed README's Performance block no
## longer matches OUT_DIR (drift). Run before committing; wiring it into CI is
## Phase 8.
benchmark-report-check:
	go run ./benchmarks/scripts/report -check \
		-results "$(OUT_DIR)/results.json" -footprint "$(OUT_DIR)/footprint.json" \
		-micro "$(OUT_DIR)/microbench.json" -metadata "$(OUT_DIR)/metadata.json"

## benchmark-footprint: measure the operational footprint (container-start cold
## start median/p95, idle memory, memory + CPU under load) of all six
## containerized apps into OUT_DIR/footprint.{json,csv}. Reduce COLD_START_RUNS /
## LOAD_SECONDS for a smoke; APPS="gin-gorm gombit" narrows the set. The
## embedded-Gombit single-binary variant is a follow-up slice.
##
##   make benchmark-footprint
##   make benchmark-footprint COLD_START_RUNS=3 APPS=gin-gorm   # smoke
benchmark-footprint:
	OUT_DIR="$(OUT_DIR)" bash benchmarks/scripts/footprint-all.sh

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
