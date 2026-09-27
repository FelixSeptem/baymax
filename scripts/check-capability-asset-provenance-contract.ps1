Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$repoRoot = Resolve-Path (Join-Path $PSScriptRoot "..")
Set-Location $repoRoot
# CAPABILITY_ASSET_PROVENANCE_TAXONOMY_BEGIN
    "capability_asset_duplicate_conflict"
    "capability_asset_impact_conflict"
    "capability_asset_impact_incomplete"
    "capability_asset_privacy_or_bound_violation"
    "capability_asset_provenance_drift"
    "capability_asset_missing_evidence"
    "capability_asset_reference_integrity"
    "capability_asset_replacement_compatible"
    "capability_asset_replacement_incompatible"
    "capability_asset_run_stream_parity_drift"
    "capability_asset_schema_drift"
    "capability_asset_scope_violation"
    "capability_asset_unknown_version"
# CAPABILITY_ASSET_PROVENANCE_TAXONOMY_END

$cache = if ($env:GOCACHE) { $env:GOCACHE } else { Join-Path $repoRoot ".gocache-capability-asset-provenance-gate" }
New-Item -ItemType Directory -Force -Path $cache | Out-Null
$env:GOCACHE = $cache

go test ./tool/diagnosticsreplay -run CapabilityAssetProvenance -count=1

$fixture = Join-Path $repoRoot "tool/diagnosticsreplay/testdata/capability_asset_provenance.v1.json"
if (-not (Test-Path -LiteralPath $fixture)) {
    throw "[capability-asset-provenance-gate] missing fixture $fixture"
}

$implementation = Join-Path $repoRoot "tool/diagnosticsreplay/capability_asset_provenance.go"
$matches = rg -n 'github\.com/(openai|anthropics)|google\.golang\.org/genai|net/http|os\.Open|os\.WriteFile|RuntimeRecorder|RunRecord|ModelRequest|registry\.Register|dynamicDownload|marketplaceResolve|credentialStore|hostedExecution|automaticRollback' $implementation 2>$null
if ($LASTEXITCODE -eq 0 -and $matches) {
    throw "[capability-asset-provenance-gate] forbidden runtime/provider/persistence reference detected"
}

Write-Host "[capability-asset-provenance-gate] offline privacy, fixture, replay, and library-first contract passed"
