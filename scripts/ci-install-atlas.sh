#!/usr/bin/env bash
# Installs the pinned Atlas Community Edition CLI for CI.
#
# The pin works because atlasgo.sh reads ATLAS_VERSION from the environment
# (`ATLAS_VERSION="${ATLAS_VERSION:-latest}"`) and builds the artifact name
# from it. That coupling is invisible at the call site, and its failure mode is
# silent: rename or drop the variable and CI goes back to installing
# `atlas-community-linux-amd64-latest`, which has resolved to a canary build
# before (issue #208). So this script refuses to run without ATLAS_VERSION and
# verifies afterwards that the binary on PATH really is that version — a
# canary or a newer release fails the job loudly instead of quietly becoming
# the version every migration and conformance job runs against.
#
#   ATLAS_VERSION=v1.3.0 bash scripts/ci-install-atlas.sh
set -euo pipefail

if [ -z "${ATLAS_VERSION:-}" ]; then
  echo "ci-install-atlas: ATLAS_VERSION is not set; refusing to install an unpinned Atlas" >&2
  exit 1
fi

# CI=true suppresses the installer's interactive confirmation prompt.
export ATLAS_VERSION CI=true
curl -sSf https://atlasgo.sh | sh -s -- --community

# `atlas version` prints e.g. "atlas community version v1.3.0" on its first
# line, followed by a release URL and an install hint. Take the first line by
# trimming the captured output rather than piping into `head`: under
# `pipefail`, `head` exiting after one line kills `atlas` with SIGPIPE and the
# pipeline reports 141, which would fail this step at random.
#
# Matching the trailing version rather than the whole line tolerates a reworded
# prefix while still rejecting a canary suffix like "v1.3.1-9a6bc60-canary".
reported="$(atlas version)"
installed="${reported%%$'\n'*}"
case "$installed" in
*" $ATLAS_VERSION") ;;
*)
  echo "ci-install-atlas: installed Atlas is \"$installed\", want version $ATLAS_VERSION" >&2
  exit 1
  ;;
esac
echo "$installed"
