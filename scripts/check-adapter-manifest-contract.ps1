Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$repoRoot = Resolve-Path (Join-Path $PSScriptRoot "..")
Set-Location $repoRoot
if (-not $env:GOCACHE) {
    $env:GOCACHE = Join-Path $repoRoot ".gocache"
}

Write-Host "[adapter-manifest] running offline deterministic manifest contract checks"
go test ./adapter/manifest ./integration/adapterconformance ./adapter/scaffold -count=1
Write-Host "[adapter-manifest] passed"
