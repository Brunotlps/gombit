#!/usr/bin/env bash
# Tests for ci-install-atlas.sh. Stubs `curl` on PATH with a fake installer so
# the pin contract is exercised hermetically (no network, no real Atlas): the
# script must refuse to run unpinned, must hand the pinned version to the
# installer through the environment (atlasgo.sh reads ATLAS_VERSION from
# there), and must fail the job when the binary that actually landed is a
# different version — the canary case from issue #208.
#
#   bash scripts/ci-install-atlas_test.sh
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

fail=0
note() { echo "FAIL: $*" >&2; fail=1; }

SCRIPT="scripts/ci-install-atlas.sh"

fakebin="$(mktemp -d)"
trap 'rm -rf "$fakebin"' EXIT

# Stands in for `curl -sSf https://atlasgo.sh`, printing an installer that the
# real `sh` in the script's pipeline then executes. It records the URL asked
# for and the ATLAS_VERSION it saw in its environment, and installs a fake
# atlas reporting $INSTALLED_VERSION — which the tests set independently of
# ATLAS_VERSION to simulate the installer handing back something else.
cat > "$fakebin/curl" <<'EOF'
#!/usr/bin/env bash
echo "$*" >> "$CURL_LOG"
echo "$ATLAS_VERSION" > "$INSTALLER_SAW_VERSION"
cat <<INSTALLER
#!/bin/sh
cat > "$FAKE_BIN/atlas" <<'ATLASBIN'
#!/usr/bin/env bash
echo "atlas community version $INSTALLED_VERSION"
echo "https://github.com/ariga/atlas/releases/tag/$INSTALLED_VERSION"
echo "To download an official version, visit: https://atlasgo.io/getting-started#installation"
# Real 'atlas version' prints more than one line, and a reader that stops
# after the first one kills it with SIGPIPE. Under 'set -o pipefail' that
# surfaces as exit 141 and fails the step at random, so make the tail long
# enough that any such reader hits it on every run instead of sometimes.
# (Escaped: this text is produced by the unquoted INSTALLER heredoc below.)
for i in \$(seq 1 5000); do echo "filler line \$i"; done
ATLASBIN
chmod +x "$FAKE_BIN/atlas"
INSTALLER
EOF
chmod +x "$fakebin/curl"

# Runs the script under test with the fakes on PATH, setting $rc and the log
# globals. $1 is the pinned version to request (empty means "unset"); $2 the
# version the installer actually delivers.
run_install() {
  local want="$1" got="$2"
  curl_log="$(mktemp)"; installer_saw="$(mktemp)"; out="$(mktemp)"; errout="$(mktemp)"
  rm -f "$fakebin/atlas"
  rc=0
  if [ -n "$want" ]; then
    PATH="$fakebin:$PATH" ATLAS_VERSION="$want" \
      CURL_LOG="$curl_log" INSTALLER_SAW_VERSION="$installer_saw" \
      FAKE_BIN="$fakebin" INSTALLED_VERSION="$got" \
      bash "$SCRIPT" >"$out" 2>"$errout" || rc=$?
  else
    # `env -u`, not merely omitting the assignment: CI sets ATLAS_VERSION at the
    # workflow level, so the variable is already in this test's own environment
    # and would be inherited by the script under test, making the "unpinned"
    # case silently untestable there.
    env -u ATLAS_VERSION \
      PATH="$fakebin:$PATH" \
      CURL_LOG="$curl_log" INSTALLER_SAW_VERSION="$installer_saw" \
      FAKE_BIN="$fakebin" INSTALLED_VERSION="$got" \
      bash "$SCRIPT" >"$out" 2>"$errout" || rc=$?
  fi
}

# ---- 1. an unset ATLAS_VERSION is refused before anything is downloaded ----
# Falling through to the installer here is exactly how CI silently went back
# to "latest"; the script must not do the install at all.
run_install "" v1.3.0
[ "$rc" -eq 1 ]           || note "unpinned: exit=$rc, want 1"
[ ! -s "$curl_log" ]      || note "unpinned: ran the installer anyway ($(cat "$curl_log"))"
grep -qF "ATLAS_VERSION is not set" "$errout" \
  || note "unpinned: stderr did not explain that ATLAS_VERSION is unset"

# ---- 2. the pinned version reaches the installer through the environment ----
run_install v1.3.0 v1.3.0
[ "$rc" -eq 0 ]                              || note "pinned: exit=$rc, want 0"
grep -qF "https://atlasgo.sh" "$curl_log"    || note "pinned: did not fetch the installer"
[ "$(cat "$installer_saw")" = "v1.3.0" ] \
  || note "pinned: installer saw ATLAS_VERSION=$(cat "$installer_saw"), want v1.3.0"
grep -qF "atlas community version v1.3.0" "$out" \
  || note "pinned: did not report the installed version"
# Guards the SIGPIPE trap above specifically: 141 means the script read the
# version through something that closed the pipe early.
[ "$rc" -ne 141 ] || note "pinned: exit 141 — the version check died of SIGPIPE under pipefail"

# ---- 3. a canary (or any other version) landing instead fails the job ----
# This is the guard that makes the pin enforced rather than merely requested.
run_install v1.3.0 v1.3.1-9a6bc60-canary
[ "$rc" -eq 1 ]  || note "canary: exit=$rc, want 1"
grep -qF "want version v1.3.0" "$errout" \
  || note "canary: stderr did not report the version mismatch"

# ---- 4. the workflow's pin and the script agree ----
# A pin nothing references is the defect this replaced; keep them coupled.
workflow_version="$(sed -n 's/^  ATLAS_VERSION: "\(.*\)"$/\1/p' .github/workflows/ci.yml)"
[ -n "$workflow_version" ] \
  || note "ci.yml no longer sets a workflow-level ATLAS_VERSION"
grep -qF "bash scripts/ci-install-atlas.sh" .github/workflows/ci.yml \
  || note "ci.yml does not install Atlas through $SCRIPT"
if grep -vE '^\s*#' .github/workflows/ci.yml | grep -qF "atlasgo.sh"; then
  note "ci.yml still installs Atlas inline; every install must go through $SCRIPT"
fi

if [ "$fail" -ne 0 ]; then
  echo "ci-install-atlas_test: FAILED" >&2
  exit 1
fi
echo "ci-install-atlas_test: unpinned refusal, environment pin, canary rejection, and workflow wiring all pass"
