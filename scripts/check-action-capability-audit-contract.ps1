Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$repoRoot = Resolve-Path (Join-Path $PSScriptRoot "..")
Set-Location $repoRoot
# ACTION_CAPABILITY_TAXONOMY_BEGIN
    "action_capability_approval_scope_drift"
    "action_capability_declared_observed_drift"
    "action_capability_duplicate_conflict"
    "action_capability_evidence_correlation_drift"
    "action_capability_evidence_insufficient"
    "action_capability_execution_or_discovery_detected"
    "action_capability_library_first_boundary_violation"
    "action_capability_metadata_missing"
    "action_capability_privacy_or_bound_violation"
    "action_capability_retry_idempotency_conflict"
    "action_capability_run_stream_evidence_parity_drift"
    "action_capability_schema_drift"
    "action_capability_stage_evidence_missing"
    "action_capability_stage_evidence_insufficient"
    "action_capability_unknown_version"
    "action_capability_verdict_drift"
    "action_capability_verify_evidence_missing"
# ACTION_CAPABILITY_TAXONOMY_END

$cache = if ($env:GOCACHE) { $env:GOCACHE } else { Join-Path $repoRoot ".gocache-action-capability-audit-gate" }
New-Item -ItemType Directory -Force -Path $cache | Out-Null
$env:GOCACHE = $cache

# The suite verifies a library-first, offline ActionCapabilityAudit fixture:
# bounded privacy facts, deterministic replay, fixture expectations, Run/Stream
# parity, historical compatibility, and contributioncheck boundary enforcement.
go test ./tool/diagnosticsreplay ./tool/contributioncheck -run ActionCapabilityAudit -count=1

$fixture = Join-Path $repoRoot "tool/diagnosticsreplay/testdata/action_capability_audit.v1.json"
if (-not (Test-Path -LiteralPath $fixture)) {
    throw "[action-capability-audit-gate] missing fixture $fixture"
}

$implementation = Join-Path $repoRoot "tool/diagnosticsreplay/action_capability_audit.go"
$matches = rg -n 'github\.com/(openai|anthropics)|google\.golang\.org/genai|net/http|os\.Open|os\.WriteFile|RuntimeRecorder|RunRecord|ModelRequest|registry\.Register|dynamicDownload|marketplaceResolve|credentialStore|globalActionQueue|hostedExecution|automaticCompensation' $implementation 2>$null
if ($LASTEXITCODE -eq 0 -and $matches) {
    throw "[action-capability-audit-gate] forbidden runtime/provider/persistence reference detected"
}

Write-Host "[action-capability-audit-gate] offline privacy, fixture, Run/Stream, and library-first contract passed"
