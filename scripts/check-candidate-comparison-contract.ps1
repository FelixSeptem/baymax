param([string]$FixturePath = "")

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"
. (Join-Path $PSScriptRoot "lib/native-strict.ps1")
$repoRoot = Resolve-Path (Join-Path $PSScriptRoot "..")
Set-Location $repoRoot
$env:GOCACHE = if ($env:BAYMAX_CANDIDATE_COMPARISON_GOCACHE) { $env:BAYMAX_CANDIDATE_COMPARISON_GOCACHE } else { Join-Path $repoRoot ".gocache-candidate-comparison" }
if ($FixturePath) { $env:BAYMAX_CANDIDATE_COMPARISON_FIXTURE = (Resolve-Path -LiteralPath $FixturePath).Path }

Write-Host "[candidate-comparison-contract] bounded identity, isolation, selection and cost replay"
Invoke-NativeStrict -Label "candidate comparison contract suites" -Command {
    go test ./runtime/evalcontract ./tool/diagnosticsreplay -run 'Test(NormalizeCandidateExperiment|CompareCandidateIdentity|AdmitCandidateAndSelect|BuildCandidateComparison|CandidateComparisonReplay|EvalContractFixtureSuccessAndLegacyShape)' -count=1
}
$pattern = 'github.com/FelixSeptem/baymax/(model/|core/|runtime/runner)|\.(Run|Stream|Retry|Resume|Execute)\('
$matches = @(rg -n -- $pattern tool/diagnosticsreplay/candidate_comparison.go)
if ($LASTEXITCODE -eq 0) { throw "[candidate-comparison-contract] replay owns an executable surface: $($matches -join '; ')" }
if ($LASTEXITCODE -ne 1) { throw "[candidate-comparison-contract] ownership scan failed" }
Write-Host "[candidate-comparison-contract] passed"
