#!/usr/bin/env bash
# Prefetches Go module dependencies with bounded retries so a transient
# module-proxy failure (e.g. "stream error: INTERNAL_ERROR" from
# proxy.golang.org/sum.golang.org) doesn't fail CI jobs that would otherwise
# resolve modules in the middle of a test (see issue #208). Only dependency
# acquisition is retried here — never `go test` — so a genuine test failure
# still fails immediately, with no retry hiding it.
#
# `go mod download all` warms the versions *this* module's build list selects.
# That is enough for the generated Atlas GORM loader, which `go run`s inside
# this module. It is not enough for the tests in ./cmd/gombit and ./goldentest:
# those scaffold an app and run `go mod tidy` in it, and a fresh module both
# resolves its own graph and needs the test dependencies of the packages it
# imports. Those land on versions this module never selects — huma pins
# go-spew v1.1.1 / go-difflib v1.0.0 for its own tests while gombit's graph
# selects newer ones — so --with-scaffold additionally warms that closure by
# doing the same thing those tests do.
#
#   bash scripts/ci-prefetch-go-modules.sh [--with-scaffold]
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

with_scaffold=false
while [ $# -gt 0 ]; do
  case "$1" in
  --with-scaffold) with_scaffold=true ;;
  *)
    echo "usage: ${BASH_SOURCE[0]} [--with-scaffold]" >&2
    exit 2
    ;;
  esac
  shift
done

# `go mod download all` records checksums for the *entire* module graph, not
# just what this module builds, so it rewrites go.sum with hundreds of lines
# that `go mod tidy` immediately removes again. Only the populated module cache
# is wanted here, so go.sum is put back afterwards — otherwise every run of
# this script leaves a tracked file dirty, which is how that churn got
# committed once already.
gosum_backup="$(mktemp)"
cp go.sum "$gosum_backup"
trap 'cp "$gosum_backup" go.sum; rm -f "$gosum_backup"' EXIT

attempts=3

# retry runs "$@" up to $attempts times with an attempt*5s backoff, naming
# $what in the exhausted-attempts message. It only ever wraps dependency
# acquisition — never a test command.
retry() {
  local what="$1"
  shift
  local attempt
  for attempt in $(seq 1 "$attempts"); do
    if "$@"; then
      return 0
    fi

    if [ "$attempt" -eq "$attempts" ]; then
      echo "failed to $what after ${attempts} attempts" >&2
      return 1
    fi

    sleep $((attempt * 5))
  done
}

# warm_scaffold resolves the module graph of a throwaway generated app exactly
# the way ./cmd/gombit and ./goldentest do at test time: scaffold it, point its
# framework requirement at this checkout, then tidy. --skip-tidy keeps the
# scaffold itself offline-neutral so the tidy below is the only resolution
# step, and the local replace is what the tests append too — without it this
# would warm the graph of a published gombit release instead of the one under
# test.
#
# One scaffold covers every variant ./goldentest exercises (cookie auth, MUI,
# make resource, make command) because they all generate code into the same
# module and import the same packages — verified by running the whole suite
# offline. That is an assumption, not a guarantee: if a variant ever pulls in a
# Go dependency the default app doesn't, the offline test step fails naming
# that module, and the fix is to warm that variant here too.
warm_scaffold() {
  local tmp rc=0
  tmp="$(mktemp -d)" || return 1
  (
    set -e
    go build -o "$tmp/gombit" ./cmd/gombit
    cd "$tmp"
    ./gombit new demo --module github.com/example/demo --skip-tidy >/dev/null
    printf '\nreplace github.com/gombit-dev/gombit => %s\n' "$ROOT" >>demo/go.mod
    cd demo
    go mod tidy
  ) || rc=$?
  rm -rf "$tmp"
  return "$rc"
}

retry "download Go dependencies" go mod download all

if [ "$with_scaffold" = true ]; then
  retry "warm the generated-app module closure" warm_scaffold
fi
