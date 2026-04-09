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
    echo "[context-compression-production-contract-gate] unable to prepare writable cache directory for ${env_name} at ${fallback_path}" >&2
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
  echo "[context-compression-production-contract-gate] ${label}"
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

run_step "context compression runtime config governance suites" \
  go test ./runtime/config -run 'Test(ContextAssemblerContextPressure|RuntimeContextJITConfig|ManagerRuntimeContextJITInvalidReloadRollsBack)' -count=1

run_step "context compression context assembler suites" \
  go test ./context/assembler -run 'Test(AssemblerContextPressure(SemanticCompactionUsesModelClient|SemanticCompactionBestEffortFallback|SemanticCompactionFailFast|SemanticCompactionQualityGateBestEffortFallback|PruneRetainsEvidenceAndReportsCount|SpillIdempotentAcrossRetry|SwapBackAndTieringCombination)|SwapBackIfNeededUsesRelevanceThreshold|ApplyLifecycleTieringTransitionsAndPrune)' -count=1

run_step "context compression run/stream parity suites" \
  go test ./core/runner -run 'Test(RunAndStreamContextPressure(SemanticsEquivalent|GovernanceSemanticsEquivalent)|ContextJITRunAndStreamSemanticEquivalent)' -count=1

run_step "context compression diagnostics + recorder additive suites" \
  go test ./runtime/diagnostics ./observability/event -run 'Test(StoreRunContextJIT(AdditiveFieldsPersistAndReplayIdempotent|QueryRunsParserCompatibilityAdditiveNullableDefault)|RuntimeRecorder(AcceptsSemanticContextPressurePayload|ParsesContextJITOrganizationAdditiveFields|ContextJITParserCompatibilityAdditiveNullableDefault|RecoveryParserCompatibilityAdditiveNullableDefault))' -count=1

run_step "context compression replay fixture + drift taxonomy suites" \
  go test ./tool/diagnosticsreplay ./integration -run 'Test(ReplayContractContextCompressionProductionFixtureSuite|ReplayContractContextCompressionProductionDriftClassification|ReplayContractContextCompressionProductionMixedFixtureBackwardCompatibility|ReplayContractPrimaryReasonArbitrationA69ContextCompressionFixtureSuite|PrimaryReasonArbitrationReplayContractA69ContextCompressionDriftGuard|ReplayContractA69ContextCompressionMixedFixtureBackwardCompatibility)' -count=1

changed_files=()
while IFS= read -r line; do
  [[ -z "${line}" ]] && continue
  changed_files+=("${line}")
done < <(collect_changed_files || true)

context_impacted=false
replay_impacted=false
benchmark_impacted=false
if (( ${#changed_files[@]} == 0 )); then
  context_impacted=true
  replay_impacted=true
  benchmark_impacted=true
else
  if has_changed_prefix "context/assembler/" "${changed_files[@]}" ||
    has_changed_prefix "core/runner/" "${changed_files[@]}" ||
    has_changed_prefix "runtime/config/" "${changed_files[@]}" ||
    has_changed_prefix "runtime/diagnostics/" "${changed_files[@]}" ||
    has_changed_prefix "observability/event/" "${changed_files[@]}"; then
    context_impacted=true
  fi
  if has_changed_prefix "tool/diagnosticsreplay/" "${changed_files[@]}" ||
    has_changed_prefix "integration/" "${changed_files[@]}" ||
    has_changed_prefix "runtime/diagnostics/" "${changed_files[@]}" ||
    has_changed_prefix "observability/event/" "${changed_files[@]}"; then
    replay_impacted=true
  fi
  if has_changed_prefix "context/assembler/" "${changed_files[@]}" ||
    has_changed_prefix "core/runner/" "${changed_files[@]}" ||
    has_changed_prefix "integration/" "${changed_files[@]}" ||
    has_changed_prefix "scripts/check-context-production-hardening-benchmark-regression" "${changed_files[@]}" ||
    has_changed_prefix "scripts/context-production-hardening-benchmark-baseline.env" "${changed_files[@]}"; then
    benchmark_impacted=true
  fi
fi

skip_impacted_suites=false
if is_truthy_env "${BAYMAX_CONTEXT_COMPRESSION_SKIP_IMPACTED_CONTRACT_SUITES:-}"; then
  skip_impacted_suites=true
fi
echo "[context-compression-production-contract-gate] impacted-evaluation context=${context_impacted} replay=${replay_impacted} benchmark=${benchmark_impacted} skip_impacted=${skip_impacted_suites}"

if [[ "${skip_impacted_suites}" != "true" ]]; then
  if [[ "${context_impacted}" == "true" ]]; then
    run_step "impacted-contract suites (context scope): context jit organization gate" \
      env BAYMAX_CONTEXT_JIT_SKIP_IMPACTED_CONTRACT_SUITES=1 bash scripts/check-context-jit-organization-contract.sh
  fi

  if [[ "${replay_impacted}" == "true" ]]; then
    run_step "impacted-contract suites (replay scope): diagnostics replay contract gate" \
      bash scripts/check-diagnostics-replay-contract.sh
  fi

  if [[ "${benchmark_impacted}" == "true" ]]; then
    run_step "impacted-contract suites (benchmark scope): context production hardening benchmark regression gate" \
      bash scripts/check-context-production-hardening-benchmark-regression.sh
  fi
else
  echo "[context-compression-production-contract-gate] skip impacted-contract suites (BAYMAX_CONTEXT_COMPRESSION_SKIP_IMPACTED_CONTRACT_SUITES=${BAYMAX_CONTEXT_COMPRESSION_SKIP_IMPACTED_CONTRACT_SUITES:-})"
fi

run_step "contributioncheck parity suites for context-compression-production gate" \
  go test ./tool/contributioncheck -run 'Test(ContextCompressionProductionGateScriptParity|QualityGateIncludesContextCompressionProductionGate|CIIncludesContextCompressionProductionRequiredCheckCandidate|ContextCompressionProductionRoadmapAndContractIndexClosureMarkers)' -count=1

echo "[context-compression-production-contract-gate] done"
