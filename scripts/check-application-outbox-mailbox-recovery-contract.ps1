Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"
$repoRoot = Resolve-Path (Join-Path $PSScriptRoot "..")
Set-Location $repoRoot
go test ./orchestration/outboxbridge ./orchestration/mailbox ./orchestration/scheduler ./tool/diagnosticsreplay -count=1
$output = go run ./examples/application-outbox-recovery
foreach ($marker in @("mailbox_dispatch_accepted","scheduler_dispatch_accepted","publish_before_outbox_confirmation_reconciled","effect_committed_before_target_reconciliation","duplicate_converged","rollback_preserves_pending_facts")) { if (-not ($output -match [regex]::Escape($marker))) { throw "missing marker: $marker" } }
Write-Host "[application-outbox] contract gate passed"
