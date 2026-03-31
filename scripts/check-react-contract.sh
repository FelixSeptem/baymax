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
    echo "[react-contract-gate] unable to prepare writable cache directory for ${env_name} at ${fallback_path}" >&2
    exit 1
  fi
  export "${env_name}=${fallback_path}"
}

ensure_writable_cache_env "GOCACHE" "${REPO_ROOT}/.gocache"

if [[ "${GODEBUG:-}" != *"goindex="* ]]; then
  if [[ -z "${GODEBUG:-}" ]]; then
    export GODEBUG="goindex=0"
  else
    export GODEBUG="${GODEBUG},goindex=0"
  fi
fi

run_step() {
  local label="$1"
  shift
  echo "[react-contract-gate] ${label}"
  "$@"
}

run_step "runner react taxonomy and budget suites" \
  go test ./core/runner -run 'Test(RunAndStreamToolCallLimitFailFast|ResolveReactTerminationReasonDeterministicMapping|StreamReactDuplicateToolCallEventsAreIdempotent|StreamReactCancellationUsesCanonicalTerminationReason|StreamReactToolDispatchFailureUsesCanonicalTerminationReason)' -count=1

run_step "integration react parity + readiness + sandbox suites" \
  go test ./integration -run 'Test(ReactLoopRunStreamParityIntegrationContract|RuntimeReadinessAdmissionReact|SandboxExecutionIsolationContractReactActionResolutionRunStreamParity|SandboxExecutionIsolationContractReactFallbackTaxonomyAndCountersParity|SandboxExecutionIsolationContractReactCapabilityMismatchRunStreamParity)' -count=1

run_step "runtime readiness react mapping suites" \
  go test ./runtime/config -run 'Test(ManagerReadinessPreflightReact|ArbitratePrimaryReasonReactProviderUnsupportedOutranksRecoverableReactFindings)' -count=1

run_step "provider tool-calling canonicalization suites" \
  go test ./model/openai ./model/anthropic ./model/gemini ./model/providererror ./model/toolcontract -count=1

run_step "diagnostics replay react.v1 suites" \
  go test ./tool/diagnosticsreplay -run 'TestReplayContract(PrimaryReasonArbitrationFixtureSuccessAndDeterministicOutput|PrimaryReasonArbitrationFixtureDriftClassification|ArbitrationMixedA48A52MemoryCompatibility)' -count=1

echo "[react-contract-gate] done"
