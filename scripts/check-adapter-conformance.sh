#!/usr/bin/env bash
set -euo pipefail

if [[ -z "${GOCACHE:-}" ]]; then
  export GOCACHE="$(pwd)/.gocache"
fi

echo "[adapter-conformance] running offline deterministic harness"
go test ./integration/adapterconformance -count=1
echo "[adapter-conformance] passed"
