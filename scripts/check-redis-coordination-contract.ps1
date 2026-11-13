Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"
. (Join-Path $PSScriptRoot "lib/native-strict.ps1")

$repoRoot = Resolve-Path (Join-Path $PSScriptRoot "..")
Set-Location $repoRoot

Invoke-NativeCaptureStrict -Label "go test redis coordination contract" -Command {
	go test ./core/sourcecoordination/... ./orchestration/mailbox/... ./orchestration/instanceregistry/... ./core/types/... ./core/history/... ./runtime/config/... -count=1
}

Write-Host "[redis-coordination-contract] passed"
