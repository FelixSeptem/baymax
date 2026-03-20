#!/usr/bin/env bash
set -euo pipefail

if [[ -z "${GOCACHE:-}" ]]; then
  export GOCACHE="$(pwd)/.gocache"
fi

echo "[adapter-capability] running offline deterministic negotiation contract checks"
go test ./adapter/capability ./adapter/manifest ./integration/adapterconformance ./adapter/scaffold -count=1
echo "[adapter-capability] passed"
