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
    echo "[react-plan-notebook-gate] unable to prepare writable cache directory for ${env_name} at ${fallback_path}" >&2
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
  echo "[react-plan-notebook-gate] ${label}"
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

run_step "react plan notebook runner lifecycle + hook + boundary suites" \
  go test ./core/runner -run 'Test(ReactPlan|ReactPlanNotebookDoesNotBypass)' -count=1

run_step "react plan notebook config + diagnostics + recorder additive suites" \
  go test ./runtime/config ./runtime/diagnostics ./observability/event -run 'Test(RuntimeReactPlanNotebook|ManagerRuntimeReactPlanNotebook|StoreRunReactPlanNotebook|RuntimeRecorderParsesReactPlanNotebookAdditiveFields|RuntimeRecorderReactPlanNotebookParserCompatibilityAdditiveNullableDefault)' -count=1

run_step "react plan notebook replay fixture + drift taxonomy suites" \
  go test ./tool/diagnosticsreplay ./integration -run 'Test(ReplayContractPrimaryReasonArbitrationFixtureSuccessAndDeterministicOutput|ReplayContractPrimaryReasonArbitrationFixtureDriftClassification|PrimaryReasonArbitrationReplayContractFixtureSuite)' -count=1

changed_files=()
while IFS= read -r line; do
  [[ -z "${line}" ]] && continue
  changed_files+=("${line}")
done < <(collect_changed_files || true)

runner_impacted=false
security_impacted=false
replay_impacted=false
if (( ${#changed_files[@]} == 0 )); then
  runner_impacted=true
  security_impacted=true
  replay_impacted=true
else
  if has_changed_prefix "core/runner/" "${changed_files[@]}" ||
    has_changed_prefix "core/types/" "${changed_files[@]}" ||
    has_changed_prefix "runtime/config/" "${changed_files[@]}"; then
    runner_impacted=true
  fi
  if has_changed_prefix "core/runner/" "${changed_files[@]}" ||
    has_changed_prefix "runtime/config/" "${changed_files[@]}" ||
    has_changed_prefix "runtime/diagnostics/" "${changed_files[@]}" ||
    has_changed_prefix "observability/event/" "${changed_files[@]}"; then
    security_impacted=true
  fi
  if has_changed_prefix "tool/diagnosticsreplay/" "${changed_files[@]}" ||
    has_changed_prefix "runtime/diagnostics/" "${changed_files[@]}" ||
    has_changed_prefix "observability/event/" "${changed_files[@]}"; then
    replay_impacted=true
  fi
fi

if [[ "${runner_impacted}" == "true" ]]; then
  run_step "impacted-contract suites (runner scope): react contract baseline" \
    bash scripts/check-react-contract.sh
fi

if [[ "${security_impacted}" == "true" ]]; then
  run_step "impacted-contract suites (boundary scope): policy precedence gate" \
    bash scripts/check-policy-precedence-contract.sh
  run_step "impacted-contract suites (boundary scope): sandbox egress + allowlist gate" \
    bash scripts/check-sandbox-egress-allowlist-contract.sh
fi

if [[ "${replay_impacted}" == "true" ]]; then
  run_step "impacted-contract suites (replay scope): diagnostics replay contract gate" \
    bash scripts/check-diagnostics-replay-contract.sh
fi

run_step "contributioncheck parity suites for react-plan-notebook gate" \
  go test ./tool/contributioncheck -run 'Test(ReactPlanNotebookGateScriptParity|QualityGateIncludesReactPlanNotebookGate|CIIncludesReactPlanNotebookRequiredCheckCandidate)' -count=1

echo "[react-plan-notebook-gate] done"
