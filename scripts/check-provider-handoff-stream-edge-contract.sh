#!/usr/bin/env bash
set -euo pipefail
repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"; cd "${repo_root}"
: "${GOCACHE:=${repo_root}/.gocache}"; export GOCACHE
fixture="tool/diagnosticsreplay/testdata/provider_handoff_stream_edge.v1.json"
[[ -f "${fixture}" ]] || { echo "provider_handoff_schema_drift: missing fixture" >&2; exit 1; }
[[ "$(wc -c < "${fixture}")" -le 2097152 ]] || { echo "provider_overflow_drift: fixture exceeds 2 MiB" >&2; exit 1; }
go test ./tool/diagnosticsreplay -run 'ProviderHandoffStreamEdge' -count=2
echo "[provider-handoff-stream-edge-contract] passed"
