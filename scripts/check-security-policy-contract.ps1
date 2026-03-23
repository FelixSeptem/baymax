Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"
. (Join-Path $PSScriptRoot "lib/native-strict.ps1")

$repoRoot = Resolve-Path (Join-Path $PSScriptRoot "..")
Set-Location $repoRoot

Write-Host "[security-policy-gate] runner security policy contracts"
Invoke-NativeStrict -Label "go test ./core/runner -run '^TestSecurityPolicyContract' -count=1" -Command {
    go test ./core/runner -run '^TestSecurityPolicyContract' -count=1
}

Write-Host "[security-policy-gate] runtime config reload rollback contracts"
Invoke-NativeStrict -Label "go test ./runtime/config -run '^TestSecurityPolicyContract' -count=1" -Command {
    go test ./runtime/config -run '^TestSecurityPolicyContract' -count=1
}
