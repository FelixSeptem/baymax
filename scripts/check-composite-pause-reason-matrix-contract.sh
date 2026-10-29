#!/usr/bin/env bash
set -euo pipefail
repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$repo_root"
echo '[composite-pause-reason-matrix] source evidence, per-cell promotion, and lifecycle-only mailbox boundary'
go test ./core/types -run 'TestContinuationCapabilitiesAllowExplicitPartialPauseMatrix' -count=1
go test ./orchestration/planbridge -run 'TestContinuationManifestPromotesOnlyEvidencedPauseCells|TestVersionedPauseReasonMatrixFixtures|TestBoundPauseRequirementsUseExactOwnerEvidenceAtCompileAndActivation|TestBoundPauseRequirementsRequireMatchingDeploymentOwnerAndVersion|TestMailboxContinuationManifestRequiresDurableOwnerEvidence|TestTeamOwnerContinuationManifestUsesPublicContractVersions' -count=1
go test ./core/runner -run 'TestNativeContinuationCrossProcess|TestNativeContinuationApprovalAndBudgetPauseBeforeDispatch' -count=1
go test ./orchestration/workflow -run 'TestEngineResumeContinuationRestoresFromFileStoreOnNewEngine|TestWorkflowContinuationRunStreamParity|TestWorkflowPauseControllerStopsApprovalBeforeEffectDispatch|TestWorkflowBudgetPausePreservesConsumedBudgetAndOriginalDeadline|TestWorkflowContinuationRetainsCompletedEffectAndOutputEvidence' -count=1
go test ./orchestration/teams -run 'TestTeamParallelPauseAndFreshProcessResumeRunsOriginalGraph|TestTeamPauseReasonsPersistAcrossRunAndStream|TestTeamVoteContinuationRestoresPendingMemberWithoutRerunningSettledVote' -count=1
go test ./orchestration/mailbox -run 'TestMailboxPauseReasonsReopenAndClaimSameDelivery|TestHostPlanOutcomeProjectsLeaseLossRetryDLQAndTimeout' -count=1
status=0
rg -n -- 'InputRequired,[[:space:]]*entry\.ApprovalRequired,[[:space:]]*entry\.BudgetExhausted[[:space:]]*=[[:space:]]*CapabilitySupported' orchestration/planbridge && status=0 || status=$?
if [[ "$status" == "0" ]]; then
  echo '[composite-pause-reason-matrix][promotion] found unconditional pause-cell promotion' >&2
  exit 1
fi
if [[ "$status" != "1" ]]; then
  echo '[composite-pause-reason-matrix][promotion] static scan failed' >&2
  exit 1
fi
echo '[composite-pause-reason-matrix] passed'
