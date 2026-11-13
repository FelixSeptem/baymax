Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"
. (Join-Path $PSScriptRoot "lib/native-strict.ps1")

$repoRoot = Resolve-Path (Join-Path $PSScriptRoot "..")
Set-Location $repoRoot

Invoke-NativeCaptureStrict -Label "go test sandboxexternal + sandboxconformance" -Command {
	go test ./adapter/sandboxexternal/... ./integration/sandboxconformance/... -count=1
}

Write-Host "[sandbox-external-executor-conformance] passed"
