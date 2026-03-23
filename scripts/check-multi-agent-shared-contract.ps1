Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"
. (Join-Path $PSScriptRoot "lib/native-strict.ps1")

$repoRoot = Resolve-Path (Join-Path $PSScriptRoot "..")
Set-Location $repoRoot

if ([string]::IsNullOrWhiteSpace($env:GOCACHE)) {
    $env:GOCACHE = Join-Path $repoRoot ".gocache"
}
if (-not (Test-Path $env:GOCACHE)) {
    New-Item -ItemType Directory -Path $env:GOCACHE | Out-Null
}

Write-Host "[multi-agent-shared-contract-gate] repository snapshot contract"
Invoke-NativeStrict -Label "go test ./tool/contributioncheck -run '^TestMultiAgentSharedContractSnapshotPass$' -count=1" -Command {
    go test ./tool/contributioncheck -run '^TestMultiAgentSharedContractSnapshotPass$' -count=1
}

Write-Host "[multi-agent-shared-contract-gate] validator negative contract cases"
Invoke-NativeStrict -Label "go test ./tool/contributioncheck -run '^TestValidateMultiAgentSharedContractDetectsViolations$' -count=1" -Command {
    go test ./tool/contributioncheck -run '^TestValidateMultiAgentSharedContractDetectsViolations$' -count=1
}

Write-Host "[multi-agent-shared-contract-gate] scheduler/subagent closure suite"
Invoke-NativeStrict -Label "go test ./integration -run '^TestSchedulerRecovery' -count=1" -Command {
    go test ./integration -run '^TestSchedulerRecovery' -count=1
}

Write-Host "[multi-agent-shared-contract-gate] scheduler qos/dlq suite"
Invoke-NativeStrict -Label "go test ./integration -run '^TestSchedulerQoS' -count=1" -Command {
    go test ./integration -run '^TestSchedulerQoS' -count=1
}

Write-Host "[multi-agent-shared-contract-gate] sync invocation suite"
Invoke-NativeStrict -Label "go test ./integration -run '^TestSyncInvocationContract' -count=1" -Command {
    go test ./integration -run '^TestSyncInvocationContract' -count=1
}

Write-Host "[multi-agent-shared-contract-gate] async reporting suite"
Invoke-NativeStrict -Label "go test ./integration -run '^TestAsyncReportingContract' -count=1" -Command {
    go test ./integration -run '^TestAsyncReportingContract' -count=1
}

Write-Host "[multi-agent-shared-contract-gate] canonical mailbox entrypoint guard suite"
Invoke-NativeStrict -Label "pwsh -File scripts/check-canonical-mailbox-entrypoints.ps1" -Command {
    pwsh -File scripts/check-canonical-mailbox-entrypoints.ps1
}

Write-Host "[multi-agent-shared-contract-gate] async-await lifecycle suite"
Invoke-NativeStrict -Label "go test ./integration -run '^TestAsyncReportingContractAwaitingLifecycle' -count=1" -Command {
    go test ./integration -run '^TestAsyncReportingContractAwaitingLifecycle' -count=1
}

Write-Host "[multi-agent-shared-contract-gate] async-await reconcile fallback suite"
Invoke-NativeStrict -Label "go test ./integration -run '^TestAsyncAwaitReconcileContract' -count=1" -Command {
    go test ./integration -run '^TestAsyncAwaitReconcileContract' -count=1
}

Write-Host "[multi-agent-shared-contract-gate] delayed dispatch suite"
Invoke-NativeStrict -Label "go test ./integration -run '^TestDelayedDispatchContract' -count=1" -Command {
    go test ./integration -run '^TestDelayedDispatchContract' -count=1
}

Write-Host "[multi-agent-shared-contract-gate] mailbox convergence suite"
Invoke-NativeStrict -Label "go test ./integration -run '^TestMailboxContract' -count=1" -Command {
    go test ./integration -run '^TestMailboxContract' -count=1
}

Write-Host "[multi-agent-shared-contract-gate] mailbox worker lifecycle/recover/reclaim suite"
Invoke-NativeStrict -Label "go test ./integration -run '^TestMailboxContractWorker(Lifecycle|RecoverReclaim|PanicNackPolicy|Heartbeat)|^TestMailboxContractLifecycleReasonTaxonomyGuard$' -count=1" -Command {
    go test ./integration -run '^TestMailboxContractWorker(Lifecycle|RecoverReclaim|PanicNackPolicy|Heartbeat)|^TestMailboxContractLifecycleReasonTaxonomyGuard$' -count=1
}

Write-Host "[multi-agent-shared-contract-gate] mailbox backend parity suite"
Invoke-NativeStrict -Label "go test ./integration -run '^TestMailboxContractMemoryFileParityAndRestoreReplayDeterminism$' -count=1" -Command {
    go test ./integration -run '^TestMailboxContractMemoryFileParityAndRestoreReplayDeterminism$' -count=1
}

Write-Host "[multi-agent-shared-contract-gate] mailbox runtime wiring suite"
Invoke-NativeStrict -Label "go test ./integration -run '^TestComposerContractMailboxRuntimeWiring' -count=1" -Command {
    go test ./integration -run '^TestComposerContractMailboxRuntimeWiring' -count=1
}

Write-Host "[multi-agent-shared-contract-gate] a14 closure matrix suite"
Invoke-NativeStrict -Label "go test ./integration -run '^TestTailGovernanceA14' -count=1" -Command {
    go test ./integration -run '^TestTailGovernanceA14' -count=1
}

Write-Host "[multi-agent-shared-contract-gate] workflow graph composability suite"
Invoke-NativeStrict -Label "go test ./integration -run '^TestWorkflowGraphComposabilityA15' -count=1" -Command {
    go test ./integration -run '^TestWorkflowGraphComposabilityA15' -count=1
}

Write-Host "[multi-agent-shared-contract-gate] collaboration primitives suite"
Invoke-NativeStrict -Label "go test ./integration -run '^TestCollaborationPrimitivesA16' -count=1" -Command {
    go test ./integration -run '^TestCollaborationPrimitivesA16' -count=1
}

Write-Host "[multi-agent-shared-contract-gate] collaboration retry suite"
Invoke-NativeStrict -Label "go test ./integration -run '^TestCollaborationRetryContract' -count=1" -Command {
    go test ./integration -run '^TestCollaborationRetryContract' -count=1
}

Write-Host "[multi-agent-shared-contract-gate] recovery boundary suite"
Invoke-NativeStrict -Label "go test ./integration -run '^TestRecoveryBoundaryA17' -count=1" -Command {
    go test ./integration -run '^TestRecoveryBoundaryA17' -count=1
}

Write-Host "[multi-agent-shared-contract-gate] unified query suite"
Invoke-NativeStrict -Label "go test ./integration -run '^TestUnifiedQueryContract' -count=1" -Command {
    go test ./integration -run '^TestUnifiedQueryContract' -count=1
}

Write-Host "[multi-agent-shared-contract-gate] task board query suite"
Invoke-NativeStrict -Label "go test ./integration -run '^TestTaskBoardQueryContract' -count=1" -Command {
    go test ./integration -run '^TestTaskBoardQueryContract' -count=1
}

Write-Host "[multi-agent-shared-contract-gate] composer closure suite"
Invoke-NativeStrict -Label "go test ./integration -run '^TestComposerContract' -count=1" -Command {
    go test ./integration -run '^TestComposerContract' -count=1
}

Write-Host "[multi-agent-shared-contract-gate] composer recovery suite"
Invoke-NativeStrict -Label "go test ./integration -run '^TestComposerRecovery' -count=1" -Command {
    go test ./integration -run '^TestComposerRecovery' -count=1
}
