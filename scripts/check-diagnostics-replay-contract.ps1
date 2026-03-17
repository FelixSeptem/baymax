Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$repoRoot = Resolve-Path (Join-Path $PSScriptRoot "..")
Set-Location $repoRoot

Write-Host "[diagnostics-replay-gate] replay contract tests"
go test ./tool/diagnosticsreplay -run '^TestReplayContract' -count=1
