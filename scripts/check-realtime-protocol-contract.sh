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
  echo "[realtime-protocol-contract-gate] rg is required" >&2
  exit 1
fi

assert_contains_literal() {
  local assertion="$1"
  local file="$2"
  local literal="$3"
  if ! rg --fixed-strings --quiet -- "${literal}" "${file}"; then
    echo "[realtime-protocol-contract-gate][${assertion}] missing marker '${literal}' in ${file}" >&2
    exit 1
  fi
}

assert_absent_regex() {
  local assertion="$1"
  local regex="$2"
  if rg -n --glob '!openspec/changes/archive/**' -- "${regex}" .; then
    echo "[realtime-protocol-contract-gate][${assertion}] unexpected matches found for /${regex}/" >&2
    exit 1
  fi
}

assert_no_parallel_realtime_protocol_changes() {
  local assertion="$1"
  local canonical_change_hint="introduce-realtime-event-protocol-and-interrupt-resume-contract"
  local violations=()

  shopt -s nullglob
  for dir in openspec/changes/*/; do
    local name="${dir%/}"
    name="${name##*/}"
    [[ "${name}" == "archive" ]] && continue
    local lower="${name,,}"
    if [[ "${lower}" != *"${canonical_change_hint}"* &&
      "${lower}" == *realtime* &&
      ( "${lower}" == *interrupt* || "${lower}" == *resume* || "${lower}" == *protocol* ) ]]; then
      violations+=("${name}")
    fi
  done
  shopt -u nullglob

  if (( ${#violations[@]} > 0 )); then
    echo "[realtime-protocol-contract-gate][${assertion}] parallel realtime proposal detected: ${violations[*]}" >&2
    exit 1
  fi
}

resolve_realtime_protocol_change_dir() {
  local candidate
  shopt -s nullglob
  for candidate in openspec/changes/*introduce-realtime-event-protocol-and-interrupt-resume-contract*; do
    if [[ -d "${candidate}" ]]; then
      echo "${candidate}"
      shopt -u nullglob
      return 0
    fi
  done
  for candidate in openspec/changes/archive/*introduce-realtime-event-protocol-and-interrupt-resume-contract*; do
    if [[ -d "${candidate}" ]]; then
      echo "${candidate}"
      shopt -u nullglob
      return 0
    fi
  done
  shopt -u nullglob

  echo "[realtime-protocol-contract-gate] unable to locate realtime protocol change directory in active or archive paths" >&2
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
  echo "[realtime-protocol-contract-gate] ${label}"
  "$@"
}

REALTIME_PROTOCOL_CHANGE_DIR="$(resolve_realtime_protocol_change_dir)"

run_step "assertion realtime_control_plane_absent: design marker" \
  assert_contains_literal "realtime_control_plane_absent" \
  "${REALTIME_PROTOCOL_CHANGE_DIR}/design.md" \
  "不引入平台化控制面。"

run_step "assertion realtime_control_plane_absent: gate spec marker" \
  assert_contains_literal "realtime_control_plane_absent" \
  "${REALTIME_PROTOCOL_CHANGE_DIR}/specs/go-quality-gate/spec.md" \
  "realtime_control_plane_absent"

run_step "assertion realtime_control_plane_absent: active change set closure" \
  assert_no_parallel_realtime_protocol_changes "realtime_control_plane_absent"

run_step "assertion realtime_control_plane_absent: reject hosted realtime control-plane config drift" \
  assert_absent_regex "realtime_control_plane_absent" \
  "runtime\.realtime\.[a-zA-Z0-9_.-]*(control_plane|controlplane|gateway|connection_router|session_router|managed_connection|hosted_realtime|realtime_service)"

run_step "assertion realtime_same_domain_closure: roadmap marker" \
  assert_contains_literal "realtime_same_domain_closure" \
  "docs/development-roadmap.md" \
  "Realtime 同域增量需求（事件类型扩展、中断恢复语义、顺序/幂等、回放/门禁）仅允许在本提案内以增量任务吸收，不再新增平行 realtime 提案。"

run_step "realtime runtime config governance suites" \
  go test ./runtime/config -run 'Test(RuntimeRealtimeConfig|ManagerRuntimeRealtime)' -count=1

run_step "realtime envelope + runner parity suites" \
  go test ./core/types ./core/runner -run 'Test(ParseRealtimeEventEnvelope|RealtimeEventEnvelope|RealtimeRunStream|RealtimeSequenceGapAndOrderClassification)' -count=1

run_step "realtime diagnostics recorder additive suites" \
  go test ./runtime/diagnostics ./observability/event -run 'Test(StoreRunRealtimeProtocol|RuntimeRecorderParsesRealtimeProtocolAdditiveFields|RuntimeRecorderRealtimeProtocolParserCompatibilityAdditiveNullableDefault)' -count=1

run_step "realtime replay fixture + drift taxonomy suites" \
  go test ./tool/diagnosticsreplay ./integration -run 'Test(ReplayContractPrimaryReasonArbitrationFixtureSuccessAndDeterministicOutput|ReplayContractPrimaryReasonArbitrationFixtureDriftClassification|PrimaryReasonArbitrationReplayContractFixtureSuite|PrimaryReasonArbitrationReplayContractDriftGuardFailFast)' -count=1

changed_files=()
while IFS= read -r line; do
  [[ -z "${line}" ]] && continue
  changed_files+=("${line}")
done < <(collect_changed_files || true)

parity_impacted=false
replay_impacted=false
if (( ${#changed_files[@]} == 0 )); then
  parity_impacted=true
  replay_impacted=true
else
  if has_changed_prefix "core/runner/" "${changed_files[@]}" ||
    has_changed_prefix "core/types/" "${changed_files[@]}" ||
    has_changed_prefix "runtime/config/" "${changed_files[@]}"; then
    parity_impacted=true
  fi
  if has_changed_prefix "tool/diagnosticsreplay/" "${changed_files[@]}" ||
    has_changed_prefix "integration/" "${changed_files[@]}" ||
    has_changed_prefix "runtime/diagnostics/" "${changed_files[@]}" ||
    has_changed_prefix "observability/event/" "${changed_files[@]}"; then
    replay_impacted=true
  fi
fi

if [[ "${parity_impacted}" == "true" ]]; then
  run_step "impacted-contract suites (runner scope): react contract baseline" \
    bash scripts/check-react-contract.sh
  run_step "impacted-contract suites (runner scope): react plan notebook gate" \
    bash scripts/check-react-plan-notebook-contract.sh
fi

if [[ "${replay_impacted}" == "true" ]]; then
  run_step "impacted-contract suites (replay scope): diagnostics replay contract gate" \
    bash scripts/check-diagnostics-replay-contract.sh
fi

run_step "contributioncheck parity suites for realtime protocol gate" \
  go test ./tool/contributioncheck -run 'Test(RealtimeProtocolGateScriptParity|QualityGateIncludesRealtimeProtocolGate|CIIncludesRealtimeProtocolRequiredCheckCandidate|RealtimeProtocolRoadmapAndContractIndexClosureMarkers)' -count=1

echo "[realtime-protocol-contract-gate] done"
