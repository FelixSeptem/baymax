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

echo "[multi-agent-shared-contract-gate] scheduler qos/dlq suite"
go test ./integration -run '^TestSchedulerQoS' -count=1

echo "[multi-agent-shared-contract-gate] sync invocation suite"
go test ./integration -run '^TestSyncInvocationContract' -count=1

echo "[multi-agent-shared-contract-gate] async reporting suite"
go test ./integration -run '^TestAsyncReportingContract' -count=1

echo "[multi-agent-shared-contract-gate] async-await lifecycle suite"
go test ./integration -run '^TestAsyncReportingContractAwaitingLifecycle' -count=1

echo "[multi-agent-shared-contract-gate] async-await reconcile fallback suite"
go test ./integration -run '^TestAsyncAwaitReconcileContract' -count=1

echo "[multi-agent-shared-contract-gate] delayed dispatch suite"
go test ./integration -run '^TestDelayedDispatchContract' -count=1

echo "[multi-agent-shared-contract-gate] mailbox convergence suite"
go test ./integration -run '^TestMailboxContractSyncAsyncDelayedConvergenceRunStreamSemanticEquivalence$' -count=1

echo "[multi-agent-shared-contract-gate] mailbox backend parity suite"
go test ./integration -run '^TestMailboxContractMemoryFileParityAndRestoreReplayDeterminism$' -count=1

echo "[multi-agent-shared-contract-gate] a14 closure matrix suite"
go test ./integration -run '^TestTailGovernanceA14' -count=1

echo "[multi-agent-shared-contract-gate] workflow graph composability suite"
go test ./integration -run '^TestWorkflowGraphComposabilityA15' -count=1

echo "[multi-agent-shared-contract-gate] collaboration primitives suite"
go test ./integration -run '^TestCollaborationPrimitivesA16' -count=1

echo "[multi-agent-shared-contract-gate] recovery boundary suite"
go test ./integration -run '^TestRecoveryBoundaryA17' -count=1

echo "[multi-agent-shared-contract-gate] unified query suite"
go test ./integration -run '^TestUnifiedQueryContract' -count=1

echo "[multi-agent-shared-contract-gate] task board query suite"
go test ./integration -run '^TestTaskBoardQueryContract' -count=1

echo "[multi-agent-shared-contract-gate] composer closure suite"
go test ./integration -run '^TestComposerContract' -count=1

echo "[multi-agent-shared-contract-gate] composer recovery suite"
go test ./integration -run '^TestComposerRecovery' -count=1
