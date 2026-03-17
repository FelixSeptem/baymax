Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$repoRoot = Resolve-Path (Join-Path $PSScriptRoot "..")
Set-Location $repoRoot

Write-Host "[security-delivery-gate] runner security delivery contracts"
go test ./core/runner -run '^TestSecurityDeliveryContract' -count=1

Write-Host "[security-delivery-gate] runtime config security delivery contracts"
go test ./runtime/config -run '^TestSecurityDeliveryContract' -count=1
