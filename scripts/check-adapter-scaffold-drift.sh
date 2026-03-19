#!/usr/bin/env bash
set -euo pipefail

if [[ -z "${GOCACHE:-}" ]]; then
  export GOCACHE="$(pwd)/.gocache"
fi

echo "[adapter-scaffold-drift] running fixture drift validation"
if ! go test ./adapter/scaffold -run '^TestScaffoldDriftFixtures$' -count=1; then
  echo "[adapter-scaffold-drift][fixture-mismatch] generated scaffold output diverged from committed fixtures"
  exit 1
fi

echo "[adapter-scaffold-drift] passed"
