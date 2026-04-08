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
    echo "[context-jit-organization-contract-gate] unable to prepare writable cache directory for ${env_name} at ${fallback_path}" >&2
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
  echo "[context-jit-organization-contract-gate] ${label}"
  "$@"
}

collect_changed_files() {
  local merge_base=""
  if git rev-parse --verify origin/main >/dev/null 2>&1; then
    merge_base="$(git merge-base HEAD origin/main || true)"
  fi
  if [[ -n "${merge_base}" ]]; then
    git diff --name-only --diff-filter=ACMRTUXB "${merge_base}..HEAD"
    return 0
  fi
  if git rev-parse --verify HEAD~1 >/dev/null 2>&1; then
    git diff --name-only --diff-filter=ACMRTUXB HEAD~1..HEAD
    return 0
  fi
  return 0
}

has_changed_prefix() {
  local prefix="$1"
  shift
  local file=""
  for file in "$@"; do
    if [[ "${file}" == "${prefix}"* ]]; then
      return 0
    fi
  done
  return 1
}

is_truthy_env() {
  local value="${1:-}"
  value="$(echo "${value}" | tr '[:upper:]' '[:lower:]' | xargs)"
  case "${value}" in
    1|true|yes|on) return 0 ;;
    *) return 1 ;;
  esac
}

run_step "context jit runtime config governance suites" \
  go test ./runtime/config -run 'Test(RuntimeContextJITConfig|ManagerRuntimeContextJIT)' -count=1

run_step "context jit context assembler organization suites" \
  go test ./context/assembler -run 'Test(DiscoverStage2References|ResolveSelectedStage2References|AssemblerContextStage2ReferenceFirstInjectsRefsBeforeBody|IngestIsolateHandoffChunks|AssemblerContextStage2IsolateHandoff(DefaultConsumption|ReplayIdempotent)|ApplyContextEditGate|AssemblerContextStage2EditGateDenyKeepsSemantics|SwapBackIfNeededUsesRelevanceThreshold|ApplyLifecycleTieringTransitionsAndPrune|AssemblerContextPressureSwapBackAndTieringCombination|AssemblerContextStage2RecapAppended|BuildTaskAwareTailRecapStableOrdering)' -count=1

run_step "context jit run/stream parity + boundary regression suites" \
  go test ./core/runner -run 'Test(ContextJITRunAndStreamSemanticEquivalent|ContextJITDoesNotBypassSandboxEgressRunAndStreamParity)' -count=1

run_step "context jit diagnostics + recorder additive suites" \
  go test ./runtime/diagnostics ./observability/event -run 'Test(StoreRunContextJIT|RuntimeRecorderParsesContextJITOrganizationAdditiveFields|RuntimeRecorderContextJITParserCompatibilityAdditiveNullableDefault)' -count=1

run_step "context jit replay fixture + drift taxonomy suites" \
  go test ./tool/diagnosticsreplay ./integration -run 'Test(ReplayContractPrimaryReasonArbitrationFixtureSuccessAndDeterministicOutput|ReplayContractPrimaryReasonArbitrationFixtureDriftClassification|ReplayContractArbitrationMixedSandboxRolloutMemoryReactSandboxEgressCompatibility|ReplayContractArbitrationMixedPolicyPrecedenceReactSandboxEgressCompatibility|PrimaryReasonArbitrationReplayContractFixtureSuite|PrimaryReasonArbitrationReplayContractDriftGuardFailFast|ReplayContractMixedPolicyPrecedenceReactSandboxEgressCompatibility)' -count=1

run_step "assertion context_provider_sdk_absent" \
  go test ./tool/contributioncheck -run '^TestContextPackagesDoNotDirectlyImportProviderSDKs$' -count=1

changed_files=()
while IFS= read -r line; do
  [[ -z "${line}" ]] && continue
  changed_files+=("${line}")
done < <(collect_changed_files || true)

parity_impacted=false
boundary_impacted=false
replay_impacted=false
if (( ${#changed_files[@]} == 0 )); then
  parity_impacted=true
  boundary_impacted=true
  replay_impacted=true
else
  if has_changed_prefix "context/assembler/" "${changed_files[@]}" ||
    has_changed_prefix "core/runner/" "${changed_files[@]}" ||
    has_changed_prefix "runtime/config/" "${changed_files[@]}"; then
    parity_impacted=true
  fi
  if has_changed_prefix "context/" "${changed_files[@]}" ||
    has_changed_prefix "core/runner/" "${changed_files[@]}" ||
    has_changed_prefix "runtime/config/" "${changed_files[@]}" ||
    has_changed_prefix "runtime/diagnostics/" "${changed_files[@]}" ||
    has_changed_prefix "observability/event/" "${changed_files[@]}"; then
    boundary_impacted=true
  fi
  if has_changed_prefix "tool/diagnosticsreplay/" "${changed_files[@]}" ||
    has_changed_prefix "integration/" "${changed_files[@]}" ||
    has_changed_prefix "runtime/diagnostics/" "${changed_files[@]}" ||
    has_changed_prefix "observability/event/" "${changed_files[@]}"; then
    replay_impacted=true
  fi
fi

skip_impacted_suites=false
if is_truthy_env "${BAYMAX_CONTEXT_JIT_SKIP_IMPACTED_CONTRACT_SUITES:-}"; then
  skip_impacted_suites=true
fi
echo "[context-jit-organization-contract-gate] impacted-evaluation parity=${parity_impacted} boundary=${boundary_impacted} replay=${replay_impacted} skip_impacted=${skip_impacted_suites}"

if [[ "${skip_impacted_suites}" != "true" ]]; then
  if [[ "${parity_impacted}" == "true" ]]; then
    run_step "impacted-contract suites (runner scope): react contract baseline" \
      bash scripts/check-react-contract.sh
    run_step "impacted-contract suites (runner scope): react plan notebook gate" \
      bash scripts/check-react-plan-notebook-contract.sh
    run_step "impacted-contract suites (runner scope): realtime protocol gate" \
      bash scripts/check-realtime-protocol-contract.sh
  fi

  if [[ "${boundary_impacted}" == "true" ]]; then
    run_step "impacted-contract suites (boundary scope): policy precedence gate" \
      bash scripts/check-policy-precedence-contract.sh
    run_step "impacted-contract suites (boundary scope): sandbox egress + allowlist gate" \
      bash scripts/check-sandbox-egress-allowlist-contract.sh
  fi

  if [[ "${replay_impacted}" == "true" ]]; then
    run_step "impacted-contract suites (replay scope): diagnostics replay contract gate" \
      bash scripts/check-diagnostics-replay-contract.sh
  fi
else
  echo "[context-jit-organization-contract-gate] skip impacted-contract suites (BAYMAX_CONTEXT_JIT_SKIP_IMPACTED_CONTRACT_SUITES=${BAYMAX_CONTEXT_JIT_SKIP_IMPACTED_CONTRACT_SUITES:-})"
fi

run_step "contributioncheck parity suites for context-jit-organization gate" \
  go test ./tool/contributioncheck -run 'Test(ContextJITOrganizationGateScriptParity|QualityGateIncludesContextJITOrganizationGate|CIIncludesContextJITOrganizationRequiredCheckCandidate|ContextJITOrganizationRoadmapAndContractIndexClosureMarkers)' -count=1

echo "[context-jit-organization-contract-gate] done"

