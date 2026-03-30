Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"
. (Join-Path $PSScriptRoot "lib/native-strict.ps1")

$repoRoot = Resolve-Path (Join-Path $PSScriptRoot "..")
Set-Location $repoRoot

Write-Host "[sandbox-rollout-governance-gate] runtime rollout config/readiness contracts"
Invoke-NativeStrict -Label "go test ./runtime/config -run 'Test(SandboxRolloutPhaseTransitionValidation|ManagerSandboxRolloutGovernanceRecordRunAutoFreeze|ManagerSandboxRolloutUnfreezeRequiresCooldownAndToken|ManagerSandboxCapacityActionDeterministicFromQueueAndInflight|ManagerReadinessPreflightSandboxRolloutFrozenFinding|ManagerReadinessPreflightSandboxRolloutHealthBudgetBreachedFinding|ManagerReadinessPreflightSandboxCapacityStrictMapping|ManagerReadinessAdmissionSandboxRolloutFrozenDeny|ManagerReadinessAdmissionSandboxCapacityPolicyMapping)' -count=1" -Command {
    go test ./runtime/config -run 'Test(SandboxRolloutPhaseTransitionValidation|ManagerSandboxRolloutGovernanceRecordRunAutoFreeze|ManagerSandboxRolloutUnfreezeRequiresCooldownAndToken|ManagerSandboxCapacityActionDeterministicFromQueueAndInflight|ManagerReadinessPreflightSandboxRolloutFrozenFinding|ManagerReadinessPreflightSandboxRolloutHealthBudgetBreachedFinding|ManagerReadinessPreflightSandboxCapacityStrictMapping|ManagerReadinessAdmissionSandboxRolloutFrozenDeny|ManagerReadinessAdmissionSandboxCapacityPolicyMapping)' -count=1
}

Write-Host "[sandbox-rollout-governance-gate] composer run/stream rollout parity contracts"
Invoke-NativeStrict -Label "go test ./orchestration/composer -run 'TestComposerReadinessAdmissionSandbox(RolloutFrozenRunAndStreamEquivalent|CapacityThrottlePolicyParity|RolloutTimelineReasonParity)' -count=1" -Command {
    go test ./orchestration/composer -run 'TestComposerReadinessAdmissionSandbox(RolloutFrozenRunAndStreamEquivalent|CapacityThrottlePolicyParity|RolloutTimelineReasonParity)' -count=1
}

Write-Host "[sandbox-rollout-governance-gate] runtime recorder a52 additive contracts"
Invoke-NativeStrict -Label "go test ./observability/event -run 'TestRuntimeRecorder(A52ParserCompatibilityAdditiveNullableDefault|ParsesA52RolloutGovernanceFields)' -count=1" -Command {
    go test ./observability/event -run 'TestRuntimeRecorder(A52ParserCompatibilityAdditiveNullableDefault|ParsesA52RolloutGovernanceFields)' -count=1
}

Write-Host "[sandbox-rollout-governance-gate] diagnostics replay a52 fixture drift contracts"
Invoke-NativeStrict -Label "go test ./tool/diagnosticsreplay -run 'TestReplayContract(PrimaryReasonArbitrationFixtureSuccessAndDeterministicOutput|PrimaryReasonArbitrationFixtureDriftClassification|ArbitrationMixedA51A52Compatibility)' -count=1" -Command {
    go test ./tool/diagnosticsreplay -run 'TestReplayContract(PrimaryReasonArbitrationFixtureSuccessAndDeterministicOutput|PrimaryReasonArbitrationFixtureDriftClassification|ArbitrationMixedA51A52Compatibility)' -count=1
}
