Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"
. (Join-Path $PSScriptRoot "lib/native-strict.ps1")

$repoRoot = Resolve-Path (Join-Path $PSScriptRoot "..")
Set-Location $repoRoot

Write-Host "[diagnostics-replay-gate] replay contract tests"
Invoke-NativeStrict -Label "go test ./tool/diagnosticsreplay -run '^TestReplayContract' -count=1" -Command {
    go test ./tool/diagnosticsreplay -run '^TestReplayContract' -count=1
}
