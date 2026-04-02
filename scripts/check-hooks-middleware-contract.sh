#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
cd "${REPO_ROOT}"

if [[ -z "${GOCACHE:-}" ]]; then
  export GOCACHE="${REPO_ROOT}/.gocache"
fi
if [[ "${GODEBUG:-}" != *"goindex="* ]]; then
  if [[ -z "${GODEBUG:-}" ]]; then
    export GODEBUG="goindex=0"
  else
    export GODEBUG="${GODEBUG},goindex=0"
  fi
fi

if ! command -v rg >/dev/null 2>&1; then
  echo "[hooks-middleware-contract-gate] rg is required" >&2
  exit 1
fi

assert_contains_literal() {
  local assertion="$1"
  local file="$2"
  local literal="$3"
  if ! rg --fixed-strings --quiet -- "${literal}" "${file}"; then
    echo "[hooks-middleware-contract-gate][${assertion}] missing marker '${literal}' in ${file}" >&2
    exit 1
  fi
}

assert_absent_regex() {
  local assertion="$1"
  local regex="$2"
  if rg -n --glob '!openspec/changes/archive/**' -- "${regex}" .; then
    echo "[hooks-middleware-contract-gate][${assertion}] unexpected matches found for /${regex}/" >&2
    exit 1
  fi
}

assert_no_parallel_a65_changes() {
  local assertion="$1"
  local canonical_change="introduce-agent-lifecycle-hooks-and-tool-middleware-contract-a65"
  local violations=()

  shopt -s nullglob
  for dir in openspec/changes/*/; do
    local name="${dir%/}"
    name="${name##*/}"
    [[ "${name}" == "archive" ]] && continue
    local lower="${name,,}"
    if [[ "${name}" != "${canonical_change}" && "${lower}" == *hook* && "${lower}" == *middleware* ]]; then
      violations+=("${name}")
    fi
  done
  shopt -u nullglob

  if (( ${#violations[@]} > 0 )); then
    echo "[hooks-middleware-contract-gate][${assertion}] parallel hooks/middleware proposal detected: ${violations[*]}" >&2
    exit 1
  fi
}

resolve_a65_change_dir() {
  local active="openspec/changes/introduce-agent-lifecycle-hooks-and-tool-middleware-contract-a65"
  if [[ -d "${active}" ]]; then
    echo "${active}"
    return 0
  fi

  local candidate
  shopt -s nullglob
  for candidate in openspec/changes/archive/*introduce-agent-lifecycle-hooks-and-tool-middleware-contract-a65; do
    if [[ -d "${candidate}" ]]; then
      echo "${candidate}"
      shopt -u nullglob
      return 0
    fi
  done
  shopt -u nullglob

  echo "[hooks-middleware-contract-gate] unable to locate A65 change directory in active or archive paths" >&2
  exit 1
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

run_step() {
  local label="$1"
  shift
  echo "[hooks-middleware-contract-gate] ${label}"
  "$@"
}

A65_CHANGE_DIR="$(resolve_a65_change_dir)"

run_step "assertion control_plane_absent: contract marker" \
  assert_contains_literal "control_plane_absent" \
  "${A65_CHANGE_DIR}/specs/agent-lifecycle-hooks-and-tool-middleware-contract/spec.md" \
  "MUST NOT require hosted control-plane services"

run_step "assertion control_plane_absent: gate spec marker" \
  assert_contains_literal "control_plane_absent" \
  "${A65_CHANGE_DIR}/specs/go-quality-gate/spec.md" \
  "control_plane_absent"

run_step "assertion control_plane_absent: active change set closure" \
  assert_no_parallel_a65_changes "control_plane_absent"

run_step "assertion control_plane_absent: reject hooks/middleware control-plane key drift" \
  assert_absent_regex "control_plane_absent" \
  "runtime\\.(hooks|tool_middleware)\\.[a-zA-Z0-9_.-]*(control_plane|controlplane|orchestrator|controller|service_endpoint|remote_hook|hosted_hook|managed_middleware)"

run_step "assertion a65_same_domain_closure: roadmap marker" \
  assert_contains_literal "a65_same_domain_closure" \
  "docs/development-roadmap.md" \
  "A65 hooks/middleware 同域增量需求（lifecycle、middleware、discovery、preprocess、mapping、回放、门禁）仅允许在 A65 内以增量任务吸收，不再新开平行提案。"

run_step "a65 runner hooks/middleware run-stream parity suites" \
  go test ./core/runner -run 'Test(LifecycleHooksRunAndStreamPhaseOrderParity|LifecycleHooksFailFastStopsRunAndStream|LifecycleHooksDegradeContinuesRunAndStream|ToolMiddlewareTimeoutClassifiedAsPolicyTimeoutInRunAndStream|SkillPreprocessRunsBeforeRunAndStreamModelLoop|SkillPreprocessFailFastAbortsRunAndStream|SkillPreprocessDegradeContinuesRunAndStream|SkillBundlePromptMappingAppendDeterministicForRunAndStream|SkillBundlePromptMappingConflictFailFastForRunAndStream|SkillBundleWhitelistFailFastRejectsBlockedToolForRunAndStream|SkillBundleWhitelistUpperBoundSandboxRejectsDuringPreprocess|SkillBundleWhitelistFirstWinFiltersBlockedToolForRunAndStream)' -count=1

run_step "a65 diagnostics additive compatibility suites" \
  go test ./runtime/diagnostics ./observability/event -run 'Test(StoreRunA65AdditiveFieldsPersistAndReplayIdempotent|RuntimeRecorderParsesA65HooksMiddlewareSkillAdditiveFields|RuntimeRecorderA65ParserCompatibilityAdditiveNullableDefault|RuntimeRecorderA65ReasonTaxonomyDriftGuardCanonicalFallback)' -count=1

run_step "a65 replay fixture + drift suites" \
  go test ./tool/diagnosticsreplay ./integration -run 'TestReplayContractA65HooksMiddleware(FixtureSuite|DriftClassification|DriftGuardFailFast)' -count=1

changed_files=()
while IFS= read -r line; do
  [[ -z "${line}" ]] && continue
  changed_files+=("${line}")
done < <(collect_changed_files || true)

runner_impacted=false
skill_impacted=false
observability_impacted=false
if (( ${#changed_files[@]} == 0 )); then
  runner_impacted=true
  skill_impacted=true
  observability_impacted=true
else
  if has_changed_prefix "core/runner/" "${changed_files[@]}" ||
    has_changed_prefix "tool/local/" "${changed_files[@]}" ||
    has_changed_prefix "core/types/" "${changed_files[@]}"; then
    runner_impacted=true
  fi
  if has_changed_prefix "skill/loader/" "${changed_files[@]}" ||
    has_changed_prefix "core/runner/" "${changed_files[@]}" ||
    has_changed_prefix "runtime/config/runtime_hooks_middleware" "${changed_files[@]}"; then
    skill_impacted=true
  fi
  if has_changed_prefix "runtime/diagnostics/" "${changed_files[@]}" ||
    has_changed_prefix "observability/event/" "${changed_files[@]}" ||
    has_changed_prefix "tool/diagnosticsreplay/" "${changed_files[@]}"; then
    observability_impacted=true
  fi
fi

if [[ "${runner_impacted}" == "true" ]]; then
  run_step "impacted-contract suites (runner scope): security policy/event gates" \
    bash scripts/check-security-policy-contract.sh
  run_step "impacted-contract suites (runner scope): security event gate" \
    bash scripts/check-security-event-contract.sh
fi

if [[ "${skill_impacted}" == "true" ]]; then
  run_step "impacted-contract suites (skill scope): skill loader + runtime skill config suites" \
    go test ./skill/loader ./runtime/config -run 'Test(Compile|RuntimeHooksToolMiddlewareSkillConfig|ManagerRuntimeHooksAndSkillInvalidReloadRollsBack)' -count=1
  run_step "impacted-contract suites (skill scope): replay/contract compatibility suites" \
    go test ./tool/diagnosticsreplay ./integration -run 'Test(ReplayContractA65HooksMiddleware|ReplayContractPrimaryReasonArbitrationFixtureSuccessAndDeterministicOutput|PrimaryReasonArbitrationReplayContractFixtureSuite)' -count=1
fi

if [[ "${observability_impacted}" == "true" ]]; then
  run_step "impacted-contract suites (observability scope): observability export+bundle gate" \
    bash scripts/check-observability-export-and-bundle-contract.sh
  run_step "impacted-contract suites (observability scope): diagnostics replay compatibility suites" \
    go test ./tool/diagnosticsreplay ./integration -run 'Test(ReplayContractA65HooksMiddleware|ReplayContractA61|ReplayContractPrimaryReasonArbitrationFixtureSuccessAndDeterministicOutput)' -count=1
fi

run_step "contributioncheck parity suites for hooks/middleware gate" \
  go test ./tool/contributioncheck -run 'Test(HooksMiddlewareGateScriptParity|QualityGateIncludesHooksMiddlewareGate|HooksMiddlewareRoadmapAndContractIndexClosureMarkers)' -count=1

echo "[hooks-middleware-contract-gate] done"
