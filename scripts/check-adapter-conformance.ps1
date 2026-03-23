Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"
. (Join-Path $PSScriptRoot "lib/native-strict.ps1")

$repoRoot = Resolve-Path (Join-Path $PSScriptRoot "..")
Set-Location $repoRoot
if (-not $env:GOCACHE) {
    $env:GOCACHE = Join-Path $repoRoot ".gocache"
}

Write-Host "[adapter-conformance] running offline deterministic harness"
Invoke-NativeStrict -Label "go test ./integration/adapterconformance -count=1" -Command {
    go test ./integration/adapterconformance -count=1
}
Write-Host "[adapter-conformance] passed"
