Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"
. (Join-Path $PSScriptRoot "lib/native-strict.ps1")

$repoRoot = Resolve-Path (Join-Path $PSScriptRoot "..")
Set-Location $repoRoot

Write-Host "[security-delivery-gate] runner security delivery contracts"
Invoke-NativeStrict -Label "go test ./core/runner -run '^TestSecurityDeliveryContract' -count=1" -Command {
    go test ./core/runner -run '^TestSecurityDeliveryContract' -count=1
}

Write-Host "[security-delivery-gate] runtime config security delivery contracts"
Invoke-NativeStrict -Label "go test ./runtime/config -run '^TestSecurityDeliveryContract' -count=1" -Command {
    go test ./runtime/config -run '^TestSecurityDeliveryContract' -count=1
}
