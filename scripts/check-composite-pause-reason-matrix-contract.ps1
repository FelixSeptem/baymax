Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"
. (Join-Path $PSScriptRoot "lib/native-strict.ps1")
$repoRoot = Resolve-Path (Join-Path $PSScriptRoot "..")
Set-Location $repoRoot

Write-Host "[composite-pause-reason-matrix] source evidence, per-cell promotion, and lifecycle-only mailbox boundary"
Invoke-NativeStrict -Label "pause matrix contract suites" -Command {
	go test ./core/types -run 'TestContinuationCapabilitiesAllowExplicitPartialPauseMatrix' -count=1
	go test ./orchestration/planbridge -run 'TestContinuationManifestPromotesOnlyEvidencedPauseCells|TestVersionedPauseReasonMatrixFixtures|TestBoundPauseRequirementsUseExactOwnerEvidenceAtCompileAndActivation|TestBoundPauseRequirementsRequireMatchingDeploymentOwnerAndVersion|TestMailboxContinuationManifestRequiresDurableOwnerEvidence|TestTeamOwnerContinuationManifestUsesPublicContractVersions' -count=1
	go test ./core/runner -run 'TestNativeContinuationCrossProcess|TestNativeContinuationApprovalAndBudgetPauseBeforeDispatch' -count=1
	go test ./orchestration/workflow -run 'TestEngineResumeContinuationRestoresFromFileStoreOnNewEngine|TestWorkflowContinuationRunStreamParity|TestWorkflowPauseControllerStopsApprovalBeforeEffectDispatch|TestWorkflowBudgetPausePreservesConsumedBudgetAndOriginalDeadline|TestWorkflowContinuationRetainsCompletedEffectAndOutputEvidence' -count=1
	go test ./orchestration/teams -run 'TestTeamParallelPauseAndFreshProcessResumeRunsOriginalGraph|TestTeamPauseReasonsPersistAcrossRunAndStream|TestTeamVoteContinuationRestoresPendingMemberWithoutRerunningSettledVote' -count=1
	go test ./orchestration/mailbox -run 'TestMailboxPauseReasonsReopenAndClaimSameDelivery|TestHostPlanOutcomeProjectsLeaseLossRetryDLQAndTimeout' -count=1
}

$matches = @(rg -n -- 'InputRequired,\s*entry\.ApprovalRequired,\s*entry\.BudgetExhausted\s*=\s*CapabilitySupported' orchestration/planbridge)
if ($LASTEXITCODE -eq 0) { throw "[composite-pause-reason-matrix][promotion] found unconditional pause-cell promotion: $($matches -join '; ')" }
if ($LASTEXITCODE -ne 1) { throw "[composite-pause-reason-matrix][promotion] static scan failed" }

Write-Host "[composite-pause-reason-matrix] passed"
