Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"
$repoRoot = Resolve-Path (Join-Path $PSScriptRoot "..")
Set-Location $repoRoot
$env:GOTOOLCHAIN = "local"
go test ./context/handoff ./tool/diagnosticsreplay -count=1
Write-Host "[context-handoff-contract-gate] done"
