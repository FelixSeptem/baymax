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
  echo "[runtime-budget-admission-gate] rg is required" >&2
  exit 1
fi

assert_contains_literal() {
  local assertion="$1"
  local file="$2"
  local literal="$3"
  if ! rg --fixed-strings --quiet -- "${literal}" "${file}"; then
    echo "[runtime-budget-admission-gate][${assertion}] missing marker '${literal}' in ${file}" >&2
    exit 1
  fi
}

assert_absent_regex() {
  local assertion="$1"
  local regex="$2"
  if rg -n --glob '!openspec/changes/archive/**' -- "${regex}" .; then
    echo "[runtime-budget-admission-gate][${assertion}] unexpected matches found for /${regex}/" >&2
    exit 1
  fi
}

assert_no_parallel_budget_admission_changes() {
  local assertion="$1"
  local canonical_change="introduce-runtime-cost-latency-budget-and-admission-contract-a60"
  local violations=()

  shopt -s nullglob
  for dir in openspec/changes/*/; do
    local name="${dir%/}"
    name="${name##*/}"
    [[ "${name}" == "archive" ]] && continue
    local lower="${name,,}"
    if [[ "${lower}" == *budget* && "${lower}" == *admission* && "${name}" != "${canonical_change}" ]]; then
      violations+=("${name}")
    fi
  done
  shopt -u nullglob

  if (( ${#violations[@]} > 0 )); then
    echo "[runtime-budget-admission-gate][${assertion}] parallel budget-admission proposal detected: ${violations[*]}" >&2
    exit 1
  fi
}

resolve_budget_a60_change_dir() {
  local active="openspec/changes/introduce-runtime-cost-latency-budget-and-admission-contract-a60"
  if [[ -d "${active}" ]]; then
    echo "${active}"
    return 0
  fi

  local candidate
  shopt -s nullglob
  for candidate in openspec/changes/archive/*introduce-runtime-cost-latency-budget-and-admission-contract-a60; do
    if [[ -d "${candidate}" ]]; then
      echo "${candidate}"
      shopt -u nullglob
      return 0
    fi
  done
  shopt -u nullglob

  echo "[runtime-budget-admission-gate] unable to locate A60 change directory in active or archive paths" >&2
  exit 1
}

run_step() {
  local label="$1"
  shift
  echo "[runtime-budget-admission-gate] ${label}"
  "$@"
}

BUDGET_A60_CHANGE_DIR="$(resolve_budget_a60_change_dir)"

run_step "assertion budget_control_plane_absent: contract markers + no parallel control-plane config key" \
  assert_contains_literal "budget_control_plane_absent" \
  "${BUDGET_A60_CHANGE_DIR}/specs/runtime-cost-latency-budget-and-admission-contract/spec.md" \
  "MUST NOT require hosted control-plane services"

run_step "assertion budget_control_plane_absent: gate spec marker" \
  assert_contains_literal "budget_control_plane_absent" \
  "${BUDGET_A60_CHANGE_DIR}/specs/go-quality-gate/spec.md" \
  "budget_control_plane_absent"

run_step "assertion budget_control_plane_absent: active change set closure" \
  assert_no_parallel_budget_admission_changes "budget_control_plane_absent"

run_step "assertion budget_control_plane_absent: reject runtime admission control-plane key drift" \
  assert_absent_regex "budget_control_plane_absent" \
  "runtime\\.admission\\.[a-zA-Z0-9_.-]*(control_plane|controlplane|admission_service|policy_center)"

run_step "assertion budget_field_reuse_required: canonical field reuse marker" \
  assert_contains_literal "budget_field_reuse_required" \
  "${BUDGET_A60_CHANGE_DIR}/specs/runtime-cost-latency-budget-and-admission-contract/spec.md" \
  "policy_decision_path"

run_step "assertion budget_field_reuse_required: gate spec marker" \
  assert_contains_literal "budget_field_reuse_required" \
  "${BUDGET_A60_CHANGE_DIR}/specs/go-quality-gate/spec.md" \
  "budget_field_reuse_required"

run_step "assertion budget_field_reuse_required: roadmap closure marker" \
  assert_contains_literal "budget_field_reuse_required" \
  "docs/development-roadmap.md" \
  "A60 预算 admission 同域增量需求（阈值、维度、降级动作、回放、门禁）仅允许在 A60 内以增量任务吸收，不再新开平行提案。"

run_step "assertion budget_field_reuse_required: reject duplicated upstream field aliases" \
  assert_absent_regex "budget_field_reuse_required" \
  "runtime\\.admission\\.[a-zA-Z0-9_.-]*(policy_decision_path|deny_source|winner_stage|memory_scope_selected|memory_budget_used|memory_hits|memory_rerank_stats|memory_lifecycle_action)"

run_step "contributioncheck parity suites for runtime budget admission gate" \
  go test ./tool/contributioncheck -run 'Test(RuntimeBudgetAdmissionGateScriptParity|QualityGateIncludesRuntimeBudgetAdmissionGate|RuntimeBudgetAdmissionRoadmapAndContractIndexClosureMarkers)' -count=1

echo "[runtime-budget-admission-gate] done"
