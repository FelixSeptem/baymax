#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "${ROOT}"

echo "[runtime-input-contract-gate] focused contract suites"
go test ./core/types ./core/runner ./host ./observability/event ./tool/diagnosticsreplay -run 'RuntimeInput|Host.*Input|RuntimeRecorderProjectsRuntimeInput|HostTranscript' -count=1

grep -Eq 'pendingSteering|followUpLimit|DrainRuntimeInputSafePoint' core/runner/runtime_input.go
grep -Eq 'applyRuntimeInputSafePoint' core/runner/control.go
grep -Eq 'RunStream|CancelAndTerminal' core/runner/runtime_input_parity_test.go
grep -Eq 'embedded_host_protocol\.v1' tool/diagnosticsreplay/host_transcript.go
grep -Eq 'RuntimeInputControl' host/host.go

if rg -n --glob 'core/runner/*.go' --glob '!core/runner/*_test.go' 'global.*queue|mcp/(http|stdio)|net/http|provider\.NewClient' .; then
  echo "[runtime-input-contract-gate][ownership] forbidden provider/global queue marker detected" >&2
  exit 1
fi
echo "[runtime-input-contract-gate] passed"
