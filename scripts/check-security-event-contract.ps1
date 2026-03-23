Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"
. (Join-Path $PSScriptRoot "lib/native-strict.ps1")

$repoRoot = Resolve-Path (Join-Path $PSScriptRoot "..")
Set-Location $repoRoot

Write-Host "[security-event-gate] runner security event contracts"
Invoke-NativeStrict -Label "go test ./core/runner -run '^TestSecurityEventContract' -count=1" -Command {
    go test ./core/runner -run '^TestSecurityEventContract' -count=1
}

Write-Host "[security-event-gate] runtime config security event contracts"
Invoke-NativeStrict -Label "go test ./runtime/config -run '^TestSecurityEventContract' -count=1" -Command {
    go test ./runtime/config -run '^TestSecurityEventContract' -count=1
}
