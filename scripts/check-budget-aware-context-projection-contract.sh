#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
cd "${REPO_ROOT}"

# Git Bash reports MSYS-style paths (/d/...) which the Go toolchain on Windows
# rejects as "not an absolute path"; normalize through cygpath when present.
if [[ -z "${GOCACHE:-}" ]]; then
  if command -v cygpath >/dev/null 2>&1; then
    GOCACHE="$(cygpath -m "${REPO_ROOT}/.gocache")"
  else
    GOCACHE="${REPO_ROOT}/.gocache"
  fi
fi
export GOCACHE
if [[ "${GODEBUG:-}" != *"goindex="* ]]; then
  if [[ -z "${GODEBUG:-}" ]]; then
    export GODEBUG="goindex=0"
  else
    export GODEBUG="${GODEBUG},goindex=0"
  fi
fi

if ! command -v rg >/dev/null 2>&1; then
  echo "[budget-aware-context-projection-gate] rg is required" >&2
  exit 1
fi

CONTRACT_DIR="context/budgetprojection"
FIXTURE="tool/diagnosticsreplay/testdata/budget_projection.v1.json"
FIXTURE_VERSION="budget_projection.v1"

assert_contains_literal() {
  local assertion="$1"
  local file="$2"
  local literal="$3"
  if ! rg --fixed-strings --quiet -- "${literal}" "${file}"; then
    echo "[budget-aware-context-projection-gate][${assertion}] missing marker '${literal}' in ${file}" >&2
    exit 1
  fi
}

assert_absent_regex() {
  local assertion="$1"
  local regex="$2"
  local path="$3"
  if rg -n --glob '!openspec/changes/archive/**' -- "${regex}" "${path}"; then
    echo "[budget-aware-context-projection-gate][${assertion}] unexpected matches found for /${regex}/ in ${path}" >&2
    exit 1
  fi
}

run_step() {
  local label="$1"
  shift
  echo "[budget-aware-context-projection-gate] ${label}"
  "$@"
}

# --- bounded fixture ---------------------------------------------------------

if [[ ! -f "${FIXTURE}" ]]; then
  echo "[budget-aware-context-projection-gate][budget_projection_bounded] missing fixture ${FIXTURE}" >&2
  exit 1
fi

FIXTURE_BYTES="$(wc -c < "${FIXTURE}" | tr -d '[:space:]')"
if (( FIXTURE_BYTES > 2097152 )); then
  echo "[budget-aware-context-projection-gate][budget_projection_bounded] fixture is ${FIXTURE_BYTES} bytes, limit 2097152" >&2
  exit 1
fi

run_step "assertion budget_projection_bounded: fixture carries the pinned version" \
  assert_contains_literal "budget_projection_bounded" "${FIXTURE}" "${FIXTURE_VERSION}"

# --- taxonomy stability ------------------------------------------------------

TAXONOMY_CODES=(
  budget_projection_schema_drift
  budget_projection_unknown_version
  budget_facts_missing_iteration_limit
  budget_facts_missing_tool_call_limit
  budget_facts_missing_time_budget
  budget_facts_missing_cost_threshold
  budget_projection_negative_remaining
  budget_projection_ratio_out_of_range
  budget_projection_pressure_level_mismatch
  budget_projection_note_unbounded
  budget_projection_digest_mismatch
  budget_projection_overflow_drift
  budget_projection_writeback_shape_detected
  budget_benchmark_schema_drift
  budget_benchmark_metric_mismatch
  budget_benchmark_recovery_recompute_drift
  budget_benchmark_overflow_drift
  budget_projection_replay_not_idempotent
)

for code in "${TAXONOMY_CODES[@]}"; do
  run_step "assertion budget_projection_taxonomy_stable: ${code}" \
    assert_contains_literal "budget_projection_taxonomy_stable" \
    "${CONTRACT_DIR}/projection.go" "\"${code}\""
  run_step "assertion budget_projection_taxonomy_stable: README documents ${code}" \
    assert_contains_literal "budget_projection_taxonomy_stable" "context/README.md" "${code}"
done

# --- read-only / no second ledger -------------------------------------------

run_step "assertion budget_derived_projection_readonly: contract stays dependency free" \
  assert_absent_regex "budget_derived_projection_readonly" \
  "\"github.com/FelixSeptem/baymax/" "${CONTRACT_DIR}"

run_step "assertion budget_no_new_ledger: no runtime admission key introduced" \
  assert_absent_regex "budget_no_new_ledger" \
  "runtime\\.admission\\.[a-zA-Z0-9_.-]*(derived|remaining|projection|ledger)" "${CONTRACT_DIR}"

run_step "assertion budget_no_new_ledger: runtime budget owners are untouched" \
  assert_contains_literal "budget_no_new_ledger" \
  "openspec/changes/establish-budget-aware-derived-context-projection-contract/proposal.md" \
  "不修改 \`runtime/config\`、\`orchestration/scheduler\`、\`core/runner\`"

# --- nullability and boundedness markers in the contract ---------------------

run_step "assertion budget_projection_nullable_default: absent dimension marker" \
  assert_contains_literal "budget_projection_nullable_default" \
  "${CONTRACT_DIR}/projection.go" "Available bool"

run_step "assertion budget_projection_bounded: projection byte bound" \
  assert_contains_literal "budget_projection_bounded" \
  "${CONTRACT_DIR}/projection.go" "MaxProjectionBytes = 4096"

run_step "assertion budget_projection_bounded: benchmark scale bound" \
  assert_contains_literal "budget_projection_bounded" \
  "${CONTRACT_DIR}/projection.go" "MaxTraces          = 32"

# --- deterministic replay and contract suites --------------------------------

run_step "contract and boundary suites" \
  go test ./context/budgetprojection ./tool/diagnosticsreplay ./tool/contributioncheck -count=1 -run 'Budget'

run_step "replay idempotency suite" \
  go test ./tool/diagnosticsreplay -run 'BudgetProjection' -count=2

echo "[budget-aware-context-projection-gate] done"
