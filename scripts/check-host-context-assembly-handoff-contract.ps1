Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"
. (Join-Path $PSScriptRoot "lib/native-strict.ps1")
$repoRoot = Resolve-Path (Join-Path $PSScriptRoot "..")
Set-Location $repoRoot
if (-not (Get-Command go -ErrorAction SilentlyContinue)) {
    throw "[host-context-assembly-handoff][unavailable] go command is required"
}
if (-not $env:GOCACHE) { $env:GOCACHE = Join-Path $repoRoot ".gocache" }

Write-Host "[host-context-assembly-handoff] projection contract, resolver, and privacy"
Invoke-NativeStrict -Label "host_context_projection contract" -Command {
    go test ./context/hostprojection ./context/assembler -count=1 -run 'TestAdmit|TestResolve|TestAdmissionGapFixtureUsesSyntheticOpaqueReferences|TestAssembler|TestProjectedHandoffDoesNotPersistResolvedBody'
}

Write-Host "[host-context-assembly-handoff] Run/Stream parity and explicit restore"
Invoke-NativeStrict -Label "host context Run/Stream" -Command {
    go test ./core/runner -count=1 -run 'TestRunAndStreamUseEquivalentHostProjectionAssembly|TestHostProjectionPreservesReadOnlyBudgetProjectionAndExistingPressureTerminal|TestRunAndStreamNormalizeProjectionBudgetHandoffAndRejectionEquivalently|TestEngineRestoreProjectedHandoffRequiresExplicitCurrentResolver|TestEquivalentRunStreamProjectedHandoffsRestoreToSameNormalizedOutcome'
}

Write-Host "[host-context-assembly-handoff] deterministic offline replay"
Invoke-NativeStrict -Label "host context replay" -Command {
    go test ./tool/diagnosticsreplay -count=2 -run 'TestReplayHostContextProjection'
}

Write-Host "[host-context-assembly-handoff] offline example"
Invoke-NativeStrict -Label "host context offline example" -Command {
    go test ./examples/host-context-assembly -count=1 -run 'TestOfflineExampleEmitsContractMarkers'
}

Write-Host "[host-context-assembly-handoff] composite owner binding and capability evidence"
# The offline marker is asserted by the example test; keep the source-owned
# cross-step rejection visible in this blocking gate's contract vocabulary.
Write-Host "[host-context-assembly-handoff] host_context.workflow_cross_step_rejected"
Invoke-NativeStrict -Label "host context composite owners" -Command {
    go test ./orchestration/composer ./orchestration/teams ./orchestration/workflow ./orchestration/planbridge -count=1 -run 'TestComposerResumeChildContinuationResolvesBoundContextBeforeExecution|TestComposerResumeChildContinuationRechecksSchedulerFenceAfterResolver|TestTeamContinuationResolvesContextOnlyForBoundPendingMember|TestTeamContinuationRejectsForeignMemberProjectionBeforeResolverOrClaim|TestTeamContextResumeRechecksDeadlineAfterResolver|TestTeamContextResumeRunStreamParity|TestTeamContextResumeReopensFileOwnerWithBoundMember|TestTeamPauseRejectsRemoteMemberContextBinding|TestTeamVoteContinuationResolvesBoundPendingMemberWithoutRerunningSettledVote|TestWorkflowContinuationRejectsForeignStepProjectionBeforeResolverOrClaim|TestWorkflowContinuationResolvesBoundContextOnlyForPendingStep|TestWorkflowContextResumeRunStreamParity|TestWorkflowContextResumeRechecksExpiryAfterResolver|TestCapabilityManifestDeclaresContextDimensionsIndependently'
}

Write-Host "[host-context-assembly-handoff] contribution boundary"
Invoke-NativeStrict -Label "host context contribution gate" -Command {
    go test ./tool/contributioncheck -count=1 -run 'TestHostContextProjectionGateScriptsAreEquivalentAndBlocking|TestHostContextProjectionOwnershipBoundary'
}

Write-Host "[host-context-assembly-handoff] passed"
