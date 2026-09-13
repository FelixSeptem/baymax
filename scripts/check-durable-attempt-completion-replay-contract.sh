#!/usr/bin/env bash
set -euo pipefail
echo "[durable-attempt-completion-replay-gate] fixture and offline replay"
export GOCACHE="${GOCACHE:-${PWD}/.gocache-durable-replay}"
mkdir -p "$GOCACHE"
go test ./tool/diagnosticsreplay -run 'Test(Parse|Evaluate)(DurableAttemptWorkspaceBinding|CompletionSafePointOwnership)' -count=1
if rg -n 'os/exec|os/Chdir|git |workspace mutation' tool/diagnosticsreplay/durable_attempt_workspace_binding.go tool/diagnosticsreplay/completion_safe_point_ownership.go; then
  echo "replay implementation contains forbidden side effects" >&2; exit 1
fi
