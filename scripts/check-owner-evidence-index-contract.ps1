Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"
. (Join-Path $PSScriptRoot "lib/native-strict.ps1")
$repoRoot = Resolve-Path (Join-Path $PSScriptRoot "..")
Set-Location $repoRoot

Write-Host "[owner-evidence-index] canonical index and negative validation"
Invoke-NativeStrict -Label "owner evidence index tests" -Command {
    go test ./tool/contributioncheck -run '^TestValidateOwnerEvidenceIndex$' -count=1
}
Write-Host "[owner-evidence-index-contract] passed"
