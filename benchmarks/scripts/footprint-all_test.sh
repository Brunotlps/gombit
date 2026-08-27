#!/usr/bin/env bash
# Unit test for footprint-all.sh's one bit of parsing logic — to_bytes, which
# turns `docker stats` size tokens into byte counts. Sourcing runs no
# measurement (main is guarded); the Docker orchestration is glue over the
# unit-tested Go footprint package + CLI.
#
#   bash benchmarks/scripts/footprint-all_test.sh
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT"
# shellcheck source=/dev/null
APPS="" source benchmarks/scripts/footprint-all.sh

fail=0
check() {
  local in="$1" want="$2" got
  got="$(printf '%s' "$in" | to_bytes)"
  if [ "$got" != "$want" ]; then
    echo "FAIL: to_bytes($in) = $got, want $want" >&2
    fail=1
  fi
}

check "1GiB"     1073741824
check "45.66MiB" 47877980   # 45.66 * 1048576, truncated
check "512KiB"   524288
check "900B"     900
check "2MB"      2000000     # decimal unit
check "1.5GiB"   1610612736

if [ "$fail" -ne 0 ]; then
  echo "footprint-all_test: FAILED" >&2
  exit 1
fi
echo "footprint-all_test: to_bytes conversions pass"
