Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$repoRoot = Resolve-Path (Join-Path $PSScriptRoot "..")
Set-Location $repoRoot

Write-Host "[security-event-gate] runner security event contracts"
go test ./core/runner -run '^TestSecurityEventContract' -count=1

Write-Host "[security-event-gate] runtime config security event contracts"
go test ./runtime/config -run '^TestSecurityEventContract' -count=1
