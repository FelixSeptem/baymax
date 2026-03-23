Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"
. (Join-Path $PSScriptRoot "lib/native-strict.ps1")

$repoRoot = Resolve-Path (Join-Path $PSScriptRoot "..")
Set-Location $repoRoot
if (-not $env:GOCACHE) {
    $env:GOCACHE = Join-Path $repoRoot ".gocache"
}

Write-Host "[adapter-capability] running offline deterministic negotiation contract checks"
Invoke-NativeStrict -Label "go test ./adapter/capability ./adapter/manifest ./integration/adapterconformance ./adapter/scaffold -count=1" -Command {
    go test ./adapter/capability ./adapter/manifest ./integration/adapterconformance ./adapter/scaffold -count=1
}
Write-Host "[adapter-capability] passed"
