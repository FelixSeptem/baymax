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
  echo "[state-snapshot-contract-gate] rg is required" >&2
  exit 1
fi

assert_contains_literal() {
  local assertion="$1"
  local file="$2"
  local literal="$3"
  if ! rg --fixed-strings --quiet -- "${literal}" "${file}"; then
    echo "[state-snapshot-contract-gate][${assertion}] missing marker '${literal}' in ${file}" >&2
    exit 1
  fi
}

assert_absent_regex() {
  local assertion="$1"
  local regex="$2"
  if rg -n --glob '!openspec/changes/archive/**' -- "${regex}" .; then
    echo "[state-snapshot-contract-gate][${assertion}] unexpected matches found for /${regex}/" >&2
    exit 1
  fi
}

assert_no_parallel_state_snapshot_changes() {
  local assertion="$1"
  local canonical_change_hint="introduce-unified-state-and-session-snapshot-contract"
  local violations=()

  shopt -s nullglob
  for dir in openspec/changes/*/; do
    local name="${dir%/}"
    name="${name##*/}"
    [[ "${name}" == "archive" ]] && continue
    local lower="${name,,}"
    if [[ "${lower}" != *"${canonical_change_hint}"* &&
      "${lower}" == *snapshot* &&
      ( "${lower}" == *state* || "${lower}" == *session* ) ]]; then
      violations+=("${name}")
    fi
  done
  shopt -u nullglob

  if (( ${#violations[@]} > 0 )); then
    echo "[state-snapshot-contract-gate][${assertion}] parallel state/session snapshot proposal detected: ${violations[*]}" >&2
    exit 1
  fi
}

resolve_state_snapshot_change_dir() {
  local candidate
  shopt -s nullglob
  for candidate in openspec/changes/*introduce-unified-state-and-session-snapshot-contract*; do
    if [[ -d "${candidate}" ]]; then
      echo "${candidate}"
      shopt -u nullglob
      return 0
    fi
  done
  for candidate in openspec/changes/archive/*introduce-unified-state-and-session-snapshot-contract*; do
    if [[ -d "${candidate}" ]]; then
      echo "${candidate}"
      shopt -u nullglob
      return 0
    fi
  done
  shopt -u nullglob

  echo "[state-snapshot-contract-gate] unable to locate state/session snapshot change directory in active or archive paths" >&2
  exit 1
}

run_step() {
  local label="$1"
  shift
  echo "[state-snapshot-contract-gate] ${label}"
  "$@"
}

STATE_SNAPSHOT_CHANGE_DIR="$(resolve_state_snapshot_change_dir)"

run_step "assertion state_control_plane_absent: design marker" \
  assert_contains_literal "state_control_plane_absent" \
  "${STATE_SNAPSHOT_CHANGE_DIR}/design.md" \
  "不引入托管状态控制面、远程恢复调度服务或平台化迁移中心。"

run_step "assertion state_control_plane_absent: gate spec marker" \
  assert_contains_literal "state_control_plane_absent" \
  "${STATE_SNAPSHOT_CHANGE_DIR}/specs/go-quality-gate/spec.md" \
  "check-state-snapshot-contract.sh/.ps1"

run_step "assertion state_control_plane_absent: active change set closure" \
  assert_no_parallel_state_snapshot_changes "state_control_plane_absent"

run_step "assertion state_control_plane_absent: reject hosted control-plane config drift" \
  assert_absent_regex "state_control_plane_absent" \
  "runtime\.(state\.snapshot|session\.state)\.[a-zA-Z0-9_.-]*(control_plane|controlplane|state_service|orchestrator|controller|managed_state|hosted_state|remote_state|migration_center)"

run_step "assertion state_source_of_truth_reuse_required: canonical source-of-truth marker" \
  assert_contains_literal "state_source_of_truth_reuse_required" \
  "${STATE_SNAPSHOT_CHANGE_DIR}/specs/memory-scope-and-builtin-filesystem-v2-governance-contract/spec.md" \
  "MUST NOT redefine memory source-of-truth behavior."

run_step "assertion state_source_of_truth_reuse_required: roadmap closure marker" \
  assert_contains_literal "state_source_of_truth_reuse_required" \
  "docs/development-roadmap.md" \
  "State/session snapshot 必须复用现有 checkpoint/snapshot 语义与既有 memory lifecycle，不得重写存储层事实源。"

run_step "assertion state_source_of_truth_reuse_required: reject duplicated memory source aliases in snapshot config" \
  assert_absent_regex "state_source_of_truth_reuse_required" \
  "runtime\.state\.snapshot\.[a-zA-Z0-9_.-]*(memory_mode|memory_provider|memory_profile|memory_contract_version|memory_scope_selected|memory_budget_used|memory_hits|memory_rerank_stats|memory_lifecycle_action)"

run_step "snapshot config governance suites" \
  go test ./runtime/config -run 'Test(RuntimeStateSnapshotSessionConfig|ManagerRuntimeStateSnapshotInvalidReloadRollsBack)' -count=1

run_step "unified snapshot manifest contract suites" \
  go test ./orchestration/snapshot -run '^Test(ExportImportRoundTripStable|ImportIdempotencyNoInflation|ImportStrictRejectsIncompatibleVersion|ImportCompatibleWithinWindow|ImportSameOperationDifferentDigestConflict)$' -count=1

run_step "composer unified snapshot runtime suites" \
  go test ./orchestration/composer -run '^TestComposerUnifiedSnapshot' -count=1

run_step "state/session snapshot restore integration suites" \
  go test ./integration -run '^TestUnifiedSnapshot' -count=1

run_step "shared recovery suites for impacted scope" \
  go test ./integration -run '^Test(SchedulerRecovery|ComposerRecovery)' -count=1

run_step "diagnostics replay suites for impacted scope" \
  go test ./tool/diagnosticsreplay -run '^TestReplayContractPrimaryReasonArbitrationFixture(SuccessAndDeterministicOutput|DriftClassification)$' -count=1

echo "[state-snapshot-contract-gate] done"
