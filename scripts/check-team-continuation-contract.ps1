Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"
. (Join-Path $PSScriptRoot "lib/native-strict.ps1")
$repoRoot = Resolve-Path (Join-Path $PSScriptRoot "..")
Set-Location $repoRoot

Write-Host "[team-continuation-contract] policy binding, provenance, restore, and capability evidence"
Invoke-NativeStrict -Label "team continuation suites" -Command {
    go test ./orchestration/teams ./orchestration/planbridge ./core/types -count=1 -run 'TestTeam|TestContinuationCapabilities|TestVersionedPauseReasonMatrixFixtures'
}
$matches = @(rg -n 'teamPolicyDigest|teamAggregateProvenance|policy_binding|ambiguous_in_flight|run_stream_parity' orchestration/teams/continuation.go orchestration/teams/continuation_test.go)
if ($LASTEXITCODE -ne 0) { throw "[team-continuation-contract] evidence scan failed" }
Write-Host "[team-continuation-contract] passed"
