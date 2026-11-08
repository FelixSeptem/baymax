Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"
. (Join-Path $PSScriptRoot "lib/native-strict.ps1")
$repoRoot = Resolve-Path (Join-Path $PSScriptRoot "..")
Set-Location $repoRoot

Write-Host "[source-history-store-contract] owner binding, fencing, bounded replay and reconciliation"
Invoke-NativeStrict -Label "owner store deployment suites" -Command {
    go test ./core/history ./core/sourcecoordination ./core/types ./tool/diagnosticsreplay -run 'Test(OwnerStore|Journal|CoordinatedJournal|DeploymentScope|ReplayDeploymentScope)' -count=1
}

$fixture = "core/history/testdata/owner_store_deployment.v1.json"
if (-not (Test-Path -LiteralPath $fixture)) { throw "[source-history-store-contract] missing fixture: $fixture" }
$raw = Get-Content -LiteralPath $fixture -Raw | ConvertFrom-Json
if ($raw.version -ne "owner_store_deployment.v1" -or $raw.cases.Count -ne 7) { throw "[source-history-store-contract] fixture schema/case coverage invalid" }
if (@($raw.cases | Where-Object { $_.side_effects -ne $false }).Count -ne 0) { throw "[source-history-store-contract] fixture permits replay side effects" }
Write-Host "[source-history-store-contract] passed"
