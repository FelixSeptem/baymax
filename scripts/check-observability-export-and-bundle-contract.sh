#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
cd "${REPO_ROOT}"

is_writable_dir() {
  local path="${1:-}"
  [[ -n "${path}" ]] || return 1
  mkdir -p "${path}" 2>/dev/null || return 1
  local probe="${path}/._write_probe_$$"
  : > "${probe}" 2>/dev/null || return 1
  rm -f "${probe}" 2>/dev/null || true
  return 0
}

ensure_writable_cache_env() {
  local env_name="$1"
  local fallback_path="$2"
  local current="${!env_name:-}"
  if is_writable_dir "${current}"; then
    return 0
  fi
  if ! is_writable_dir "${fallback_path}"; then
    echo "[observability-export-bundle-gate] unable to prepare writable cache directory for ${env_name} at ${fallback_path}" >&2
    exit 1
  fi
  export "${env_name}=${fallback_path}"
}

ensure_writable_cache_env "GOCACHE" "${REPO_ROOT}/.gocache"

echo "[observability-export-bundle-gate] runtime config + readiness contracts"
go test ./runtime/config -run 'Test(RuntimeObservabilityConfig|ManagerRuntimeObservabilityInvalidReloadRollsBack|ManagerReadinessPreflightObservability|ManagerReadinessPreflightDiagnosticsBundleOutputUnavailableStrictMapping|ObservabilityReadinessFindingsCoverProfileAndPolicyInvalidCodes|ArbitratePrimaryReasonObservabilityPolicyInvalidOutranksSinkUnavailable)' -count=1

echo "[observability-export-bundle-gate] bundle generator + recorder + run/stream contracts"
go test ./runtime/config ./runtime/diagnostics ./observability/event ./integration -run 'Test(ManagerGenerateDiagnosticsBundle|StoreRunObservabilityAdditiveFieldsPersistAndReplayIdempotent|StoreRunObservabilityAdditiveFieldsBoundedCardinality|RuntimeRecorderAutoGeneratesObservabilityDiagnosticsBundleSuccess|RuntimeRecorderAutoGeneratesObservabilityDiagnosticsBundleFailureReason|ObservabilityExportBundleContractRunStreamSemanticEquivalenceSuccess|ObservabilityExportBundleContractRunStreamBundleFailureTaxonomyEquivalent)' -count=1

echo "[observability-export-bundle-gate] diagnostics replay observability.v1 contracts"
go test ./tool/diagnosticsreplay -run 'TestReplayContract(PrimaryReasonArbitrationFixtureSuccessAndDeterministicOutput|PrimaryReasonArbitrationFixtureDriftClassification|ArbitrationMixedSandboxRolloutMemoryReactSandboxEgressCompatibility)' -count=1

