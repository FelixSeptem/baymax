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

Write-Host "[memory-scope-search-gate] memory governance implementation suites"
Invoke-NativeStrict -Label "go test ./memory ./context/provider ./context/assembler ./core/runner ./runtime/diagnostics ./observability/event -run 'Test(MemoryProviderPassesGovernanceConfigToFacade|AssemblerCA2MemoryGovernanceDiagnosticsFields|MemoryRunDiagnosticsAccumulatorSnapshot|RunFinishedPayloadIncludesMemoryAdditiveFields|StoreRunA59MemoryGovernanceAdditiveFieldsPersistAndReplayIdempotent|RuntimeRecorderParsesA59MemoryGovernanceAdditiveFields|RuntimeRecorderA59ParserCompatibilityAdditiveNullableDefault)' -count=1" -Command {
    go test ./memory ./context/provider ./context/assembler ./core/runner ./runtime/diagnostics ./observability/event -run 'Test(MemoryProviderPassesGovernanceConfigToFacade|AssemblerCA2MemoryGovernanceDiagnosticsFields|MemoryRunDiagnosticsAccumulatorSnapshot|RunFinishedPayloadIncludesMemoryAdditiveFields|StoreRunA59MemoryGovernanceAdditiveFieldsPersistAndReplayIdempotent|RuntimeRecorderParsesA59MemoryGovernanceAdditiveFields|RuntimeRecorderA59ParserCompatibilityAdditiveNullableDefault)' -count=1
}

Write-Host "[memory-scope-search-gate] replay and integration fixture suites"
Invoke-NativeStrict -Label "go test ./tool/diagnosticsreplay ./integration -run 'Test(ReplayContractPrimaryReasonArbitrationFixture|PrimaryReasonArbitrationReplayContract)' -count=1" -Command {
    go test ./tool/diagnosticsreplay ./integration -run 'Test(ReplayContractPrimaryReasonArbitrationFixture|PrimaryReasonArbitrationReplayContract)' -count=1
}

Write-Host "[memory-scope-search-gate] done"
