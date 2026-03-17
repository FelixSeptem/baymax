Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$repoRoot = Resolve-Path (Join-Path $PSScriptRoot "..")
Set-Location $repoRoot

Write-Host "[security-policy-gate] runner security policy contracts"
go test ./core/runner -run '^TestSecurityPolicyContract' -count=1

Write-Host "[security-policy-gate] runtime config reload rollback contracts"
go test ./runtime/config -run '^TestSecurityPolicyContract' -count=1
