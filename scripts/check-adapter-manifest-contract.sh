#!/usr/bin/env bash
set -euo pipefail

if [[ -z "${GOCACHE:-}" ]]; then
  export GOCACHE="$(pwd)/.gocache"
fi

echo "[adapter-manifest] running offline deterministic manifest contract checks"
go test ./adapter/manifest ./integration/adapterconformance ./adapter/scaffold -count=1
echo "[adapter-manifest] passed"
