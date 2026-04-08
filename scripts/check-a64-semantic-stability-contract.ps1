Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"
. (Join-Path $PSScriptRoot "lib/native-strict.ps1")

$enabled = if ($env:BAYMAX_A64_SEMANTIC_STABILITY_ENABLED) {
    $env:BAYMAX_A64_SEMANTIC_STABILITY_ENABLED.Trim().ToLowerInvariant()
}
else {
    "true"
}
if ($enabled -ne "true") {
    Write-Host "[a64-semantic-stability] skipped by BAYMAX_A64_SEMANTIC_STABILITY_ENABLED=$enabled"
    exit 0
}

$repoRoot = Resolve-Path (Join-Path $PSScriptRoot "..")
Set-Location $repoRoot
if (-not $env:GOCACHE) {
    $env:GOCACHE = Join-Path $repoRoot ".gocache"
}

Write-Host "[a64-semantic-stability] substep: go split semantic equivalence strong checks"
Invoke-NativeStrict -Label "pwsh -File scripts/check-go-split-semantic-equivalence.ps1" -Command {
    pwsh -File scripts/check-go-split-semantic-equivalence.ps1
}

Write-Host "[a64-semantic-stability] substep: state snapshot contract suites"
Invoke-NativeStrict -Label "pwsh -File scripts/check-state-snapshot-contract.ps1" -Command {
    pwsh -File scripts/check-state-snapshot-contract.ps1
}

Write-Host "[a64-semantic-stability] substep: runtime budget admission contract suites"
Invoke-NativeStrict -Label "pwsh -File scripts/check-runtime-budget-admission-contract.ps1" -Command {
    pwsh -File scripts/check-runtime-budget-admission-contract.ps1
}

Write-Host "[a64-semantic-stability] substep: observability export + diagnostics bundle contract suites"
Invoke-NativeStrict -Label "pwsh -File scripts/check-observability-export-and-bundle-contract.ps1" -Command {
    pwsh -File scripts/check-observability-export-and-bundle-contract.ps1
}

Write-Host "[a64-semantic-stability] substep: diagnostics replay contract suites"
Invoke-NativeStrict -Label "pwsh -File scripts/check-diagnostics-replay-contract.ps1" -Command {
    pwsh -File scripts/check-diagnostics-replay-contract.ps1
}

Write-Host "[a64-semantic-stability] substep: hard-constraint semantic invariants (backpressure/fail_fast/timeout-cancel/decision-trace)"
Invoke-NativeStrict -Label "go test ./core/runner ./runtime/config ./integration -run hard-constraint-invariants -count=1" -Command {
    go test ./core/runner ./runtime/config ./integration -run 'Test(RunBackpressureBlockDiagnosticsAndTimeline|RunBackpressureDropLowPriorityAllDroppedFailsFast|RunBackpressureDropLowPriorityMCPAndSkillAllDroppedFailsFast|RunAndStreamCancelPropagationSemanticsEquivalent|ActionGateRunAndStreamDenySemanticsEquivalent|ActionGateRunAndStreamTimeoutSemanticsEquivalent|EvaluateRuntimePolicyDecisionDenyPrecedence|EvaluateRuntimePolicyDecisionSameStageTieBreakDeterministic|TimeoutResolutionContract)' -count=1
}

Write-Host "[a64-semantic-stability] passed"
