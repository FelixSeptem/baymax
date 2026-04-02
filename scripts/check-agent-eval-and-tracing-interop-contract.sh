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
  echo "[agent-eval-tracing-interop-gate] rg is required" >&2
  exit 1
fi

assert_contains_literal() {
  local assertion="$1"
  local file="$2"
  local literal="$3"
  if ! rg --fixed-strings --quiet -- "${literal}" "${file}"; then
    echo "[agent-eval-tracing-interop-gate][${assertion}] missing marker '${literal}' in ${file}" >&2
    exit 1
  fi
}

assert_absent_regex() {
  local assertion="$1"
  local regex="$2"
  if rg -n --glob '!openspec/changes/archive/**' -- "${regex}" .; then
    echo "[agent-eval-tracing-interop-gate][${assertion}] unexpected matches found for /${regex}/" >&2
    exit 1
  fi
}

assert_no_parallel_a61_changes() {
  local assertion="$1"
  local canonical_change="introduce-otel-tracing-and-agent-eval-interoperability-contract-a61"
  local violations=()

  shopt -s nullglob
  for dir in openspec/changes/*/; do
    local name="${dir%/}"
    name="${name##*/}"
    [[ "${name}" == "archive" ]] && continue
    local lower="${name,,}"
    if [[ "${name}" != "${canonical_change}" && "${lower}" == *eval* && ( "${lower}" == *otel* || "${lower}" == *tracing* ) ]]; then
      violations+=("${name}")
    fi
  done
  shopt -u nullglob

  if (( ${#violations[@]} > 0 )); then
    echo "[agent-eval-tracing-interop-gate][${assertion}] parallel tracing/eval proposal detected: ${violations[*]}" >&2
    exit 1
  fi
}

resolve_a61_change_dir() {
  local active="openspec/changes/introduce-otel-tracing-and-agent-eval-interoperability-contract-a61"
  if [[ -d "${active}" ]]; then
    echo "${active}"
    return 0
  fi

  local candidate
  shopt -s nullglob
  for candidate in openspec/changes/archive/*introduce-otel-tracing-and-agent-eval-interoperability-contract-a61; do
    if [[ -d "${candidate}" ]]; then
      echo "${candidate}"
      shopt -u nullglob
      return 0
    fi
  done
  shopt -u nullglob

  echo "[agent-eval-tracing-interop-gate] unable to locate A61 change directory in active or archive paths" >&2
  exit 1
}

run_step() {
  local label="$1"
  shift
  echo "[agent-eval-tracing-interop-gate] ${label}"
  "$@"
}

A61_CHANGE_DIR="$(resolve_a61_change_dir)"

run_step "assertion control_plane_absent: contract marker" \
  assert_contains_literal "control_plane_absent" \
  "${A61_CHANGE_DIR}/specs/runtime-otel-tracing-and-agent-eval-interoperability-contract/spec.md" \
  "embedded library behavior"

run_step "assertion control_plane_absent: gate spec marker" \
  assert_contains_literal "control_plane_absent" \
  "${A61_CHANGE_DIR}/specs/go-quality-gate/spec.md" \
  "control_plane_absent"

run_step "assertion control_plane_absent: active change set closure" \
  assert_no_parallel_a61_changes "control_plane_absent"

run_step "assertion control_plane_absent: reject eval execution control-plane key drift" \
  assert_absent_regex "control_plane_absent" \
  "runtime\.eval\.execution\.[a-zA-Z0-9_.-]*(control_plane|controlplane|scheduler_service|orchestrator_endpoint|controller_endpoint|hosted_scheduler|remote_scheduler)"

run_step "assertion a61_field_reuse_required: upstream reuse marker" \
  assert_contains_literal "a61_field_reuse_required" \
  "${A61_CHANGE_DIR}/specs/runtime-otel-tracing-and-agent-eval-interoperability-contract/spec.md" \
  "Tracing and eval outputs SHALL reuse canonical upstream explainability fields"

run_step "assertion a61_field_reuse_required: gate spec marker" \
  assert_contains_literal "a61_field_reuse_required" \
  "${A61_CHANGE_DIR}/specs/go-quality-gate/spec.md" \
  "Quality gate SHALL include tracing and eval interoperability contract checks"

run_step "assertion a61_field_reuse_required: roadmap closure marker" \
  assert_contains_literal "a61_field_reuse_required" \
  "docs/development-roadmap.md" \
  "A61 tracing+eval 同域增量需求（语义映射、指标汇总、执行治理、回放、门禁）仅允许在 A61 内以增量任务吸收，不再新开平行提案。"

run_step "assertion a61_field_reuse_required: reject duplicated upstream alias fields" \
  assert_absent_regex "a61_field_reuse_required" \
  "runtime\.eval\.[a-zA-Z0-9_.-]*(policy_decision_path|deny_source|winner_stage|memory_scope_selected|memory_budget_used|memory_hits|memory_rerank_stats|memory_lifecycle_action|budget_snapshot|budget_decision|degrade_action)"

run_step "runtime config tracing+eval schema and rollback suites" \
  go test ./runtime/config \
    -run 'Test(RuntimeObservabilityConfigDefaults|RuntimeObservabilityConfigEnvOverridePrecedence|RuntimeObservabilityConfigValidationRejectsInvalidValues|RuntimeObservabilityConfigInvalidBoolFailsFast|RuntimeObservabilityTracingEndpointFallbackToExportOTLPEndpoint|ManagerRuntimeObservabilityTracingInvalidReloadRollsBack|RuntimeEvalConfigDefaults|RuntimeEvalConfigEnvOverridePrecedence|RuntimeEvalConfigValidationRejectsInvalidValues|RuntimeEvalConfigInvalidBoolFailsFast|ManagerRuntimeEvalInvalidReloadRollsBack|BuildEvalSummaryThresholdBoundaries|AggregateEvalShardMetricsLocalAndDistributedEquivalence|AggregateEvalShardMetricsResumeIdempotent|RuntimeEvalExecutionConfigBoundaryNoControlPlaneDependency)' \
    -count=1

run_step "tracing semconv/export + diagnostics additive suites" \
  go test ./observability/trace ./observability/event ./runtime/diagnostics \
    -run 'Test(CanonicalSemconvTopologyV1CoversCoreDomains|CanonicalAttributeMapInjectsSchemaAndFiltersUnknownKeys|RunStreamSemanticEquivalenceAllowsOrderingDifferences|RunStreamSemanticEquivalenceDetectsTopologyDrift|ExportRuntime.*|RuntimeRecorderParsesA61TracingEvalAdditiveFields|RuntimeRecorderA61ParserCompatibilityAdditiveNullableDefault|StoreRunA61TracingEvalAdditiveFieldsPersistAndReplayIdempotent|StoreRunA61TracingEvalAdditiveFieldsBoundedCardinality)' \
    -count=1

run_step "replay fixtures and drift taxonomy suites (A61)" \
  go test ./tool/diagnosticsreplay ./integration \
    -run 'TestReplayContract.*(Otel|Eval|A61)' \
    -count=1

run_step "contributioncheck parity suites for A61 gate" \
  go test ./tool/contributioncheck \
    -run 'Test(AgentEvalTracingInteropGateScriptParity|QualityGateIncludesAgentEvalTracingInteropGate|AgentEvalTracingInteropRoadmapAndContractIndexClosureMarkers)' \
    -count=1

echo "[agent-eval-tracing-interop-gate] done"
