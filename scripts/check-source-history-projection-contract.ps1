Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"
. (Join-Path $PSScriptRoot "lib/native-strict.ps1")
$repoRoot = Resolve-Path (Join-Path $PSScriptRoot "..")
Set-Location $repoRoot

Write-Host "[source-history-projection-contract] committed facts, projection idempotency and recovery-required"
Invoke-NativeStrict -Label "source history projection ledger suites" -Command {
    go test ./core/history ./core/types ./orchestration/planbridge ./core/runner ./orchestration/composer ./orchestration/teams ./orchestration/workflow -run '^TestProjectionLedger|^TestJournalAppend.*ProjectionLedger|^TestJournalProjectionRecovery|^TestJournalCapabilitiesDeclareProjectionRecoveryOnlyForDurableLedger|^Test.*BindsSourceProjectionLedger|^TestCapabilityManifestHistoryProjectionRecoveryRequiresDurableOwnerLedger$' -count=1
}

$fixture = "tool/diagnosticsreplay/testdata/source-history-projection.v1.json"
if (-not (Test-Path -LiteralPath $fixture)) { throw "[source-history-projection-contract] missing fixture: $fixture" }
$raw = Get-Content -Raw $fixture | ConvertFrom-Json
if ($raw.version -ne "source-history-projection.v1" -or $raw.cases.Count -lt 8) { throw "[source-history-projection-contract] fixture schema/case coverage invalid" }
if (@($raw.cases | Where-Object { $_.side_effects -ne $false }).Count -ne 0) { throw "[source-history-projection-contract] fixture permits replay side effects" }
Write-Host "[source-history-projection-contract] passed"
