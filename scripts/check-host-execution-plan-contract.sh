#!/usr/bin/env bash
set -euo pipefail
repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$repo_root"

echo '[host-execution-plan-contract] bounded compile, native delegation, replay and example'
go test ./orchestration/planbridge ./orchestration/teams ./orchestration/workflow ./orchestration/scheduler ./integration ./tool/diagnosticsreplay ./tool/contributioncheck -run 'TestHostExecutionPlan|TestHostExecutionPlanOwnershipBoundary' -count=1

pattern='github.com/FelixSeptem/baymax/(observability|mcp/)|^[[:space:]]*go[[:space:]]+|\.(Resume|Retry|Cancel|Record|AppendEvent)\('
if rg -n -- "$pattern" orchestration/planbridge tool/diagnosticsreplay/host_execution_plan.go; then
  echo '[host-execution-plan-contract][ownership] bridge or replay owns forbidden recovery/diagnostic behavior' >&2
  exit 1
fi

if BAYMAX_HOST_EXECUTION_PLAN_FIXTURE=testdata/host_execution_plan.invalid.json go test ./tool/diagnosticsreplay -run TestHostExecutionPlanReplayGoldenAndDrift -count=1; then
  echo '[host-execution-plan-contract] invalid replay fixture unexpectedly passed' >&2
  exit 1
fi

smoke="$(go run ./examples/host-execution-plan)"
for marker in plan-compiled source-owner-preserved run-stream-parity mailbox-logical-admission mailbox-lifecycle-source-owned invalid-plan-rejected unsupported-capability-rejected no-side-effects-before-rejection host-commit-required; do
  if ! grep -Fxq "$marker" <<< "${smoke//$'\r'/}"; then
    echo "[host-execution-plan-contract] missing marker $marker" >&2
    exit 1
  fi
done
echo '[host-execution-plan-contract] passed'
