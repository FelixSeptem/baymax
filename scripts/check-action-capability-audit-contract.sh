#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$repo_root"
# ACTION_CAPABILITY_TAXONOMY_BEGIN
taxonomy_codes=(
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
)
# ACTION_CAPABILITY_TAXONOMY_END

cache_dir="${GOCACHE:-$repo_root/.gocache-action-capability-audit-gate}"
mkdir -p "$cache_dir"

# The suite verifies a library-first, offline ActionCapabilityAudit fixture:
# bounded privacy facts, deterministic replay, fixture expectations, Run/Stream
# parity, historical compatibility, and contributioncheck boundary enforcement.
GOCACHE="$cache_dir" go test ./tool/diagnosticsreplay ./tool/contributioncheck -run ActionCapabilityAudit -count=1

test -f "tool/diagnosticsreplay/testdata/action_capability_audit.v1.json" || { echo "[action-capability-audit-gate] missing fixture" >&2; exit 1; }

if rg -n 'github\.com/(openai|anthropics)|google\.golang\.org/genai|net/http|os\.Open|os\.WriteFile|RuntimeRecorder|RunRecord|ModelRequest|registry\.Register|dynamicDownload|marketplaceResolve|credentialStore|globalActionQueue|hostedExecution|automaticCompensation' tool/diagnosticsreplay/action_capability_audit.go >/dev/null; then
  echo "[action-capability-audit-gate] forbidden runtime/provider/persistence reference detected" >&2
  exit 1
fi

echo "[action-capability-audit-gate] offline privacy, fixture, Run/Stream, and library-first contract passed"
