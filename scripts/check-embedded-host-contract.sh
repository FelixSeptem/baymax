#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "${ROOT}"

echo "[embedded-host-contract-gate] focused contract suites"
go test ./core/types ./core/runner ./host/... ./observability/event ./tool/diagnosticsreplay -run 'Host|Realtime|RuntimeRecorder' -count=1

echo "[embedded-host-contract-gate] ownership and non-goal assertions"
if rg -n --glob 'host/**/*.go' --glob 'host/*.go' --glob '!host/**/*_test.go' --glob '!host/*_test.go' 'runtime/diagnostics|mcp/(http|stdio)|net/http|gorilla/websocket|database/sql' .; then
  echo "[embedded-host-contract-gate][source_owner] host must not own diagnostics, MCP, network, or persistence dependencies" >&2
  exit 1
fi
if rg -n --glob 'host/**/*.go' --glob 'host/*.go' --glob '!host/**/*_test.go' --glob '!host/*_test.go' '(hosted listener|remote Session|Artifact store|global queue|terminal state machine)' .; then
  echo "[embedded-host-contract-gate][non_goals] forbidden hosted control-plane semantics detected" >&2
  exit 1
fi
if rg -n --glob 'host/**/*.go' --glob 'host/*.go' --glob '!host/**/*_test.go' --glob '!host/*_test.go' '^[[:space:]]*var[[:space:]]+[A-Za-z0-9_]+[[:space:]]*=[[:space:]]*make\((map|chan)' .; then
  echo "[embedded-host-contract-gate][bounded_state] package-global mutable host state detected" >&2
  exit 1
fi

echo "[embedded-host-contract-gate] replay and framing markers"
# embedded_host_protocol.v1 is the versioned replay contract marker.
rg -n --glob 'tool/diagnosticsreplay/host_transcript*' 'embedded_host_protocol\.v1' . >/dev/null
rg -n --glob 'host/jsonl/*.go' 'LF|CRLF|Frame|Negotiat|MaxFrame' . >/dev/null
rg -n --glob 'host/host.go' 'RuntimeInputControl|AdmitHostRuntimeInput|run.input.steering|run.input.follow_up' . >/dev/null
echo "[embedded-host-contract-gate] done"
