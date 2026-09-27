#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$repo_root"
# CAPABILITY_ASSET_PROVENANCE_TAXONOMY_BEGIN
taxonomy_codes=(
  "capability_asset_duplicate_conflict"
  "capability_asset_impact_conflict"
  "capability_asset_impact_incomplete"
  "capability_asset_privacy_or_bound_violation"
  "capability_asset_provenance_drift"
  "capability_asset_reference_integrity"
  "capability_asset_replacement_compatible"
  "capability_asset_replacement_incompatible"
  "capability_asset_run_stream_parity_drift"
  "capability_asset_schema_drift"
  "capability_asset_scope_violation"
  "capability_asset_unknown_version"
)
# CAPABILITY_ASSET_PROVENANCE_TAXONOMY_END

cache_dir="${GOCACHE:-$repo_root/.gocache-capability-asset-provenance-gate}"
mkdir -p "$cache_dir"
GOCACHE="$cache_dir" go test ./tool/diagnosticsreplay -run CapabilityAssetProvenance -count=1

test -f "tool/diagnosticsreplay/testdata/capability_asset_provenance.v1.json" || {
  echo "[capability-asset-provenance-gate] missing fixture" >&2
  exit 1
}

if rg -n 'github\.com/(openai|anthropics)|google\.golang\.org/genai|net/http|os\.Open|os\.WriteFile|RuntimeRecorder|RunRecord|ModelRequest|registry\.Register|dynamicDownload|marketplaceResolve|credentialStore|hostedExecution|automaticRollback' tool/diagnosticsreplay/capability_asset_provenance.go >/dev/null; then
  echo "[capability-asset-provenance-gate] forbidden runtime/provider/persistence reference detected" >&2
  exit 1
fi

echo "[capability-asset-provenance-gate] offline privacy, fixture, replay, and library-first contract passed"
