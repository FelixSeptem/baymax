Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"
. (Join-Path $PSScriptRoot "lib/native-strict.ps1")

$repoRoot = Resolve-Path (Join-Path $PSScriptRoot "..")
Set-Location $repoRoot
if (-not $env:GOCACHE) {
    $env:GOCACHE = Join-Path $repoRoot ".gocache"
}

Write-Host "[sandbox-egress-allowlist-gate] runtime config + readiness + admission contracts"
Invoke-NativeStrict -Label "go test ./runtime/config -run 'Test(SecuritySandboxEgress|AdapterAllowlist|SandboxEgressReadiness|ManagerReadinessPreflightSandboxEgress|ManagerReadinessPreflightAdapterAllowlist|ManagerReadinessAdmissionAdapterAllowlist|ArbitratePrimaryReasonSandboxEgress|ArbitratePrimaryReasonAdapterAllowlist)' -count=1" -Command {
    go test ./runtime/config -run 'Test(SecuritySandboxEgress|AdapterAllowlist|SandboxEgressReadiness|ManagerReadinessPreflightSandboxEgress|ManagerReadinessPreflightAdapterAllowlist|ManagerReadinessAdmissionAdapterAllowlist|ArbitratePrimaryReasonSandboxEgress|ArbitratePrimaryReasonAdapterAllowlist)' -count=1
}

Write-Host "[sandbox-egress-allowlist-gate] adapter manifest allowlist activation contracts"
Invoke-NativeStrict -Label "go test ./adapter/manifest -run 'Test(ParseManifestAllowlist|ActivateManifestAllowlist)' -count=1" -Command {
    go test ./adapter/manifest -run 'Test(ParseManifestAllowlist|ActivateManifestAllowlist)' -count=1
}

Write-Host "[sandbox-egress-allowlist-gate] sandbox adapter conformance egress + allowlist matrix"
Invoke-NativeStrict -Label "go test ./integration/adapterconformance -run 'TestSandboxAdapterConformance(EgressPolicyMatrix|EgressSelectorOverridePrecedence|AllowlistActivationMatrix|AllowlistTaxonomyDriftClassification|CanonicalDriftClasses)' -count=1" -Command {
    go test ./integration/adapterconformance -run 'TestSandboxAdapterConformance(EgressPolicyMatrix|EgressSelectorOverridePrecedence|AllowlistActivationMatrix|AllowlistTaxonomyDriftClassification|CanonicalDriftClasses)' -count=1
}

Write-Host "[sandbox-egress-allowlist-gate] diagnostics additive fields and run/stream parity"
Invoke-NativeStrict -Label "go test ./runtime/diagnostics ./observability/event ./core/runner ./integration -run 'Test(StoreRunSandboxEgressAdditiveFieldsPersistAndReplayIdempotent|RuntimeRecorderParsesSandboxEgressAdditiveFields|RunSandboxEgressAdditiveFieldsPropagateToRunFinishedPayload|RuntimeReadinessAdmissionContractAdapterAllowlistMissingEntryRunStreamParity)' -count=1" -Command {
    go test ./runtime/diagnostics ./observability/event ./core/runner ./integration -run 'Test(StoreRunSandboxEgressAdditiveFieldsPersistAndReplayIdempotent|RuntimeRecorderParsesSandboxEgressAdditiveFields|RunSandboxEgressAdditiveFieldsPropagateToRunFinishedPayload|RuntimeReadinessAdmissionContractAdapterAllowlistMissingEntryRunStreamParity)' -count=1
}

Write-Host "[sandbox-egress-allowlist-gate] replay fixture + drift classification"
Invoke-NativeStrict -Label "go test ./tool/diagnosticsreplay ./integration -run 'Test(ReplayContractPrimaryReasonArbitrationFixtureSuccessAndDeterministicOutput|ReplayContractPrimaryReasonArbitrationFixtureDriftClassification|ReplayContractArbitrationMixedSandboxRolloutMemoryReactSandboxEgressCompatibility|ReplayContractSandboxEgressAllowlistFixture|PrimaryReasonArbitrationReplayContractFixtureSuite|PrimaryReasonArbitrationReplayContractDriftGuardFailFast)' -count=1" -Command {
    go test ./tool/diagnosticsreplay ./integration -run 'Test(ReplayContractPrimaryReasonArbitrationFixtureSuccessAndDeterministicOutput|ReplayContractPrimaryReasonArbitrationFixtureDriftClassification|ReplayContractArbitrationMixedSandboxRolloutMemoryReactSandboxEgressCompatibility|ReplayContractSandboxEgressAllowlistFixture|PrimaryReasonArbitrationReplayContractFixtureSuite|PrimaryReasonArbitrationReplayContractDriftGuardFailFast)' -count=1
}
