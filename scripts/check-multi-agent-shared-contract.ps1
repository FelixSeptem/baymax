Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$repoRoot = Resolve-Path (Join-Path $PSScriptRoot "..")
Set-Location $repoRoot

if ([string]::IsNullOrWhiteSpace($env:GOCACHE)) {
    $env:GOCACHE = Join-Path $repoRoot ".gocache"
}
if (-not (Test-Path $env:GOCACHE)) {
    New-Item -ItemType Directory -Path $env:GOCACHE | Out-Null
}

Write-Host "[multi-agent-shared-contract-gate] repository snapshot contract"
go test ./tool/contributioncheck -run '^TestMultiAgentSharedContractSnapshotPass$' -count=1

Write-Host "[multi-agent-shared-contract-gate] validator negative contract cases"
go test ./tool/contributioncheck -run '^TestValidateMultiAgentSharedContractDetectsViolations$' -count=1
