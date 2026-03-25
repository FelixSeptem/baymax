#!/usr/bin/env bash
set -euo pipefail

if [[ -z "${GOCACHE:-}" ]]; then
  export GOCACHE="$(pwd)/.gocache"
fi

echo "[adapter-conformance] running offline deterministic harness"
echo "[adapter-conformance] adapter-health matrix"
go test ./integration/adapterconformance -run '^TestAdapterConformanceHealthMatrix' -count=1
echo "[adapter-conformance] adapter-health matrix passed"

echo "[adapter-conformance] running full conformance harness"
go test ./integration/adapterconformance -count=1
echo "[adapter-conformance] passed"
