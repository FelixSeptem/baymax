Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$repoRoot = Resolve-Path (Join-Path $PSScriptRoot "..")
Set-Location $repoRoot
if (-not $env:GOCACHE) {
    $env:GOCACHE = Join-Path $repoRoot ".gocache"
}

Write-Host "[adapter-contract-replay] running offline deterministic replay checks"
go test ./integration/adaptercontractreplay -run '^TestReplayContract' -count=1
Write-Host "[adapter-contract-replay] passed"
