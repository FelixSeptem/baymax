#!/usr/bin/env bash
set -euo pipefail
export GOCACHE="${BAYMAX_FIRST_ERROR_GOCACHE:-$(pwd)/.gocache-eval-first-error}"
go test ./runtime/evalcontract ./tool/diagnosticsreplay -run 'Test(NormalizeFirstErrorAttribution|CompareFirstErrorAttribution|FirstErrorAttributionFixture)' -count=1
if rg -n 'os/exec|os/Chdir|net/http|git |workspace mutation|raw_reasoning|transcript_body|provider_response|tool_output|memory_body|workspace_body|credential' runtime/evalcontract/first_error_attribution.go tool/diagnosticsreplay/first_error_attribution.go; then
  echo "[eval-first-error-attribution-contract] forbidden side effects or body persistence" >&2
  exit 1
fi
echo "[eval-first-error-attribution-contract] passed"
