#!/usr/bin/env bash
set -euo pipefail
export GOCACHE="${BAYMAX_CONTINUITY_GOCACHE:-$(pwd)/.gocache-eval-continuity}"
go test ./runtime/evalcontract ./tool/diagnosticsreplay -run 'Test(Continuity|ReplayContractContinuity)' -count=1
if rg -n 'os/exec|os/Chdir|git |workspace mutation|transcript body|reasoning body' runtime/evalcontract/continuity*.go tool/diagnosticsreplay/continuity*.go; then
  echo "[eval-continuity-comparison-contract] forbidden side effects or body persistence" >&2
  exit 1
fi
echo "[eval-continuity-comparison-contract] passed"
