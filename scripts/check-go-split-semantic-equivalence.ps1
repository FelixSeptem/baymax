Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"
. (Join-Path $PSScriptRoot "lib/native-strict.ps1")

$repoRoot = Resolve-Path (Join-Path $PSScriptRoot "..")
Set-Location $repoRoot
if (-not $env:GOCACHE) {
    $env:GOCACHE = Join-Path $repoRoot ".gocache"
}
if ($env:GODEBUG) {
    if ($env:GODEBUG -notmatch "(^|,)goindex=") {
        $env:GODEBUG = "$($env:GODEBUG),goindex=0"
    }
}
else {
    $env:GODEBUG = "goindex=0"
}

Write-Host "[go-split-strong-check] impacted memory contract suites"
Invoke-NativeStrict -Label "go test impacted memory contract suites" -Command {
    go test ./memory ./context/provider ./context/assembler ./core/runner ./runtime/diagnostics ./observability/event -run 'Test(MemoryProviderPassesGovernanceConfigToFacade|AssemblerContextStage2MemoryGovernanceDiagnosticsFields|MemoryRunDiagnosticsAccumulatorSnapshot|RunFinishedPayloadIncludesMemoryAdditiveFields|StoreRunMemoryGovernanceAdditiveFieldsPersistAndReplayIdempotent|RuntimeRecorderParsesMemoryGovernanceAdditiveFields|RuntimeRecorderMemoryGovernanceParserCompatibilityAdditiveNullableDefault)' -count=1
}

Write-Host "[go-split-strong-check] run stream parity suites"
Invoke-NativeStrict -Label "go test run stream parity suites" -Command {
    go test ./integration -run 'Test(TimeoutResolutionContractRunStreamAndMemoryFileParity|RuntimeReadinessAdmissionContractBlockedDenyRunStreamEquivalentAndNoSideEffects|RuntimeReadinessAdmissionContractAdapterCircuitOpenRunStreamParity)' -count=1
}

Write-Host "[go-split-strong-check] replay idempotency and drift suites"
Invoke-NativeStrict -Label "go test replay idempotency and drift suites" -Command {
    go test ./tool/diagnosticsreplay ./integration -run 'Test(TimeoutResolutionContractReplayIdempotency|ReplayContractPrimaryReasonArbitrationFixture|PrimaryReasonArbitrationReplayContract)' -count=1
}

Write-Host "[go-split-strong-check] passed"
