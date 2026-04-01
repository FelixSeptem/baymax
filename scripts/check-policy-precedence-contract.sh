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
    echo "[policy-precedence-gate] unable to prepare writable cache directory for ${env_name} at ${fallback_path}" >&2
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
  echo "[policy-precedence-gate] ${label}"
  "$@"
}

run_step "runtime policy precedence config/evaluator/rollback suites" \
  go test ./runtime/config -run 'Test(RuntimePolicyConfig|EvaluateRuntimePolicyDecision|ManagerRuntimePolicyInvalidReloadRollsBack|ManagerReadinessPreflightPolicyCandidatesWinnerMetadata|ManagerReadinessAdmissionPolicyDecisionTraceFields)' -count=1

run_step "runner run/stream parity and deny side-effect-free suites" \
  go test ./core/runner -run 'Test(ActionGateRunAndStreamDenySemanticsEquivalent|ActionGateRunAndStreamTimeoutSemanticsEquivalent|SecurityEventContractSandboxPolicyDenyRunAndStreamEquivalent)' -count=1

run_step "diagnostics and recorder additive/replay-idempotent suites" \
  go test ./runtime/diagnostics ./observability/event -run 'Test(StoreRunPolicyPrecedenceAdditiveFieldsPersistAndReplayIdempotent|StoreRunPolicyPrecedenceAdditiveFieldsBoundedCardinality|RuntimeRecorderParsesA58AdditiveFields|RuntimeRecorderA58ParserCompatibilityAdditiveNullableDefault)' -count=1

run_step "policy stack replay and drift taxonomy suites" \
  go test ./tool/diagnosticsreplay ./integration -run 'Test(ReplayContractPolicyPrecedenceFixture|ReplayContractMixedA50ReactSandboxEgressPolicyStackCompatibility|ReplayContractPrimaryReasonArbitrationFixtureSuccessAndDeterministicOutput|ReplayContractPrimaryReasonArbitrationFixtureDriftClassification|PrimaryReasonArbitrationReplayContractFixtureSuite|PrimaryReasonArbitrationReplayContractDriftGuardFailFast)' -count=1

run_step "docs parity suites" \
  bash scripts/check-docs-consistency.sh

echo "[policy-precedence-gate] done"
