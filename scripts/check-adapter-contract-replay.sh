#!/usr/bin/env bash
set -euo pipefail

if [[ -z "${GOCACHE:-}" ]]; then
  export GOCACHE="$(pwd)/.gocache"
fi

echo "[adapter-contract-replay] running offline deterministic replay checks"
go test ./integration/adaptercontractreplay -run '^TestReplayContract' -count=1
echo "[adapter-contract-replay] passed"
