#!/usr/bin/env bash
set -euo pipefail
repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$repo_root"
export GOCACHE="${GOCACHE:-$repo_root/.gocache-deployment-scope}"
echo '[durable-store-deployment-scope] local ownership, fail-closed admission and read-only replay'
go test ./core/types ./core/checkpoint ./core/history ./core/runner ./orchestration/composer ./orchestration/workflow ./orchestration/mailbox ./orchestration/planbridge ./tool/diagnosticsreplay -run 'TestDeploymentScope|TestCheckpointClaimRejectsUnsupportedDeploymentBeforeMutation|TestHostExecutionPlanRejectsUnsupportedDeploymentBeforeActivation|TestReplayDeploymentScopeFixture' -count=1
for file in core/types/deployment_scope.go tool/diagnosticsreplay/deployment_scope.go; do
  pattern='(Run|Stream|Resume|Retry|Append|ClaimResume)\('
  if rg -n -- "$pattern" "$file"; then
    echo "[durable-store-deployment-scope][ownership] $file contains source execution or mutation" >&2
    exit 1
  elif [[ $? -ne 1 ]]; then
    echo "[durable-store-deployment-scope] ownership scan failed for $file" >&2
    exit 1
  fi
done
echo '[durable-store-deployment-scope] passed'
