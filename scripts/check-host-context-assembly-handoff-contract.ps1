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

Write-Host "[host-context-assembly-handoff] contribution boundary"
Invoke-NativeStrict -Label "host context contribution gate" -Command {
    go test ./tool/contributioncheck -count=1 -run 'TestHostContextProjectionGateScriptsAreEquivalentAndBlocking|TestHostContextProjectionOwnershipBoundary'
}

Write-Host "[host-context-assembly-handoff] passed"
