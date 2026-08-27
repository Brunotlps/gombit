#!/usr/bin/env bash
# Measure the operational footprint of every containerized implementation (issue
# #141 §"Operational footprint") into OUT_DIR/footprint.{json,csv}: container-
# start → ready cold-start (median/p95 over COLD_START_RUNS restarts), idle
# memory, and memory + CPU under load, for all six apps.
#
# It reuses run-crud-all.sh (sourced) for app identity derivation, the compose
# wrapper, and the health wait; the throughput half is `make benchmark-crud-all`.
#
# The embedded-Gombit single-binary variant (`gombit build --embed`: binary +
# image size, cold start, idle memory) is a follow-up slice — the footprint
# schema already carries it (variant "embedded", binary_size_bytes) and the
# footprint CLI accepts it; only the `gombit build --embed` frontend-build step
# is not wired here yet.
#
#   make benchmark-footprint
#   COLD_START_RUNS=3 APPS=gin-gorm make benchmark-footprint   # smoke
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT"
# shellcheck source=/dev/null
APPS="" source benchmarks/scripts/run-crud-all.sh

OUT="${OUT_DIR:-benchmarks/results/latest}/footprint.json"
APPS="${APPS:-gin-gorm gombit django rails laravel nestjs}"
COLD_START_RUNS="${COLD_START_RUNS:-20}"
LOAD_VUS="${LOAD_VUS:-100}"
LOAD_SECONDS="${LOAD_SECONDS:-10}"

# to_bytes converts a `docker stats` size token (45.2MiB, 1.1GiB, 900B, …) to an
# integer byte count on stdin.
to_bytes() {
  awk '{
    n = $0 + 0
    if ($0 ~ /GiB/)      printf "%d", n * 1073741824
    else if ($0 ~ /MiB/) printf "%d", n * 1048576
    else if ($0 ~ /KiB/) printf "%d", n * 1024
    else if ($0 ~ /GB/)  printf "%d", n * 1000000000
    else if ($0 ~ /MB/)  printf "%d", n * 1000000
    else if ($0 ~ /kB|KB/) printf "%d", n * 1000
    else                 printf "%d", n
  }'
}

# mem_bytes CONTAINER — the container's memory usage (cgroup working set), bytes.
mem_bytes() {
  docker stats --no-stream --format '{{.MemUsage}}' "$1" | awk -F' / ' '{print $1}' | to_bytes
}
# cpu_pct CONTAINER — CPU percent (docker stats; 100 == one core).
cpu_pct() { docker stats --no-stream --format '{{.CPUPerc}}' "$1" | tr -d '% '; }

# wait_ready_ms URL — poll until HTTP 200, echo elapsed milliseconds (or fail).
wait_ready_ms() {
  local url="$1" start now code
  start="$(date +%s%3N)"
  for _ in $(seq 1 600); do
    code="$(curl -s -o /dev/null -w '%{http_code}' --max-time 2 "$url" 2>/dev/null || true)"
    if [ "$code" = "200" ]; then
      now="$(date +%s%3N)"
      echo "$((now - start))"
      return 0
    fi
    sleep 0.05
  done
  return 1
}

# generate_load TARGET_URL — drive the crud-list workload for LOAD_SECONDS at
# LOAD_VUS (same env contract as run-crud), output discarded.
generate_load() {
  docker run --rm --network host \
    -e TARGET_URL="$1" -e VUS="$LOAD_VUS" -e DURATION="${LOAD_SECONDS}s" \
    -v "$ROOT/benchmarks/workloads/crud-list.js:/workload/crud-list.js:ro" \
    "grafana/k6:$K6_VERSION" run --quiet /workload/crud-list.js >/dev/null 2>&1 || true
}

# cold_start_samples APP LIVEZ_URL — echo COLD_START_RUNS comma-separated
# start→ready samples (ms), stopping/starting the existing container each time.
cold_start_samples() {
  local app="$1" url="$2" samples="" ms
  for _ in $(seq 1 "$COLD_START_RUNS"); do
    "${COMPOSE[@]}" stop "$app" >/dev/null
    "${COMPOSE[@]}" start "$app" >/dev/null
    ms="$(wait_ready_ms "$url")" || { echo "footprint: $app cold-start did not become ready" >&2; return 1; }
    samples="${samples:+$samples,}$ms"
  done
  printf '%s' "$samples"
}

# measure_container APP — full container footprint for one app.
measure_container() {
  local app="$1" cid livez turl samples idle loaded cpu m c image
  app_identity "$app"
  livez="http://127.0.0.1:${PORT}/livez"
  turl="http://127.0.0.1:${PORT}/api/projects?page=1&limit=20"
  echo "=== footprint: $app ($FRAMEWORK_VERSION on $RUNTIME $RUNTIME_VERSION) ==="

  "${COMPOSE[@]}" build "$app" >/dev/null
  "${COMPOSE[@]}" run --rm "$app" migrate
  "${COMPOSE[@]}" run --rm "$app" seed
  "${COMPOSE[@]}" up -d "$app" >/dev/null
  cid="$("${COMPOSE[@]}" ps -q "$app")"
  wait_healthy "$cid"

  samples="$(cold_start_samples "$app" "$livez")"

  sleep 3 # settle before reading idle memory
  idle="$(mem_bytes "$cid")"

  # Under load: sample memory (peak) and CPU (peak) while k6 runs.
  loaded=0
  cpu=0
  generate_load "$turl" &
  local loadpid=$!
  sleep 2
  for _ in 1 2 3 4; do
    m="$(mem_bytes "$cid")"; c="$(cpu_pct "$cid")"
    [ "$m" -gt "$loaded" ] && loaded="$m"
    cpu="$(awk -v a="$cpu" -v b="$c" 'BEGIN { print (b > a) ? b : a }')"
    sleep 1
  done
  wait "$loadpid" 2>/dev/null || true

  image="$(docker image inspect "$(docker inspect "$cid" --format '{{.Image}}')" --format '{{.Size}}')"

  go run ./benchmarks/scripts/footprint \
    -framework "$app" -framework-version "$FRAMEWORK_VERSION" \
    -runtime "$RUNTIME" -runtime-version "$RUNTIME_VERSION" \
    -variant container \
    -cold-start-ms "$samples" \
    -idle-rss-bytes "$idle" -loaded-rss-bytes "$loaded" -cpu-percent "$cpu" \
    -image-size-bytes "$image" \
    -out "$OUT"

  "${COMPOSE[@]}" stop "$app" >/dev/null
}

main() {
  echo "footprint: ensuring postgres is up"
  "${COMPOSE[@]}" up -d postgres >/dev/null
  for app in $APPS; do
    measure_container "$app"
  done
  echo "footprint: done. Rows in $OUT (and .csv)."
}

if [ "${BASH_SOURCE[0]}" = "${0}" ]; then
  main "$@"
fi
