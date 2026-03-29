Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"
. (Join-Path $PSScriptRoot "lib/native-strict.ps1")

$repoRoot = Resolve-Path (Join-Path $PSScriptRoot "..")
Set-Location $repoRoot

Write-Host "[sandbox-conformance] running offline deterministic harness"
Invoke-NativeStrict -Label "go test ./integration/sandboxconformance -count=1" -Command {
    go test ./integration/sandboxconformance -count=1
}
Write-Host "[sandbox-conformance] passed"
