#!/usr/bin/env bash
set -euo pipefail

if [[ -z "${GOCACHE:-}" ]]; then
  export GOCACHE="$PWD/.gocache"
fi
mkdir -p "${GOCACHE}"

echo "[multi-agent-shared-contract-gate] repository snapshot contract"
go test ./tool/contributioncheck -run '^TestMultiAgentSharedContractSnapshotPass$' -count=1

echo "[multi-agent-shared-contract-gate] validator negative contract cases"
go test ./tool/contributioncheck -run '^TestValidateMultiAgentSharedContractDetectsViolations$' -count=1

echo "[multi-agent-shared-contract-gate] scheduler/subagent closure suite"
go test ./integration -run '^TestSchedulerRecovery' -count=1

echo "[multi-agent-shared-contract-gate] composer closure suite"
go test ./integration -run '^TestComposerContract' -count=1

echo "[multi-agent-shared-contract-gate] composer recovery suite"
go test ./integration -run '^TestComposerRecovery' -count=1
