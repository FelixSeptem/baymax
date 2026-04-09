#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
cd "${REPO_ROOT}"

export GOCACHE="${GOCACHE:-${REPO_ROOT}/.tmp/go-cache-agent-mode-smoke}"
mkdir -p "${GOCACHE}"

required_patterns=(
  "rag-hybrid-retrieval"
  "structured-output-schema-contract"
  "skill-driven-discovery-hybrid"
  "mcp-governed-stdio-http"
  "hitl-governed-checkpoint"
  "context-governed-reference-first"
  "sandbox-governed-toolchain"
  "realtime-interrupt-resume"
  "multi-agents-collab-recovery"
  "workflow-branch-retry-failfast"
  "mapreduce-large-batch"
  "state-session-snapshot-recovery"
  "policy-budget-admission"
  "tracing-eval-smoke"
  "react-plan-notebook-loop"
  "hooks-middleware-extension-pipeline"
  "observability-export-bundle"
  "adapter-onboarding-manifest-capability"
  "security-policy-event-delivery"
  "config-hot-reload-rollback"
  "workflow-routing-strategy-switch"
  "multi-agents-hierarchical-planner-validator"
  "mainline-mailbox-async-delayed-reconcile"
  "mainline-task-board-query-control"
  "mainline-scheduler-qos-backoff-dlq"
  "mainline-readiness-admission-degradation"
  "custom-adapter-mcp-model-tool-memory-pack"
  "custom-adapter-health-readiness-circuit"
)

contains_pattern() {
  local target="$1"
  shift
  local item=""
  for item in "$@"; do
    if [[ "${item}" == "${target}" ]]; then
      return 0
    fi
  done
  return 1
}

extract_value() {
  local output="$1"
  local key="$2"
  local line
  line="$(printf '%s\n' "${output}" | grep -E "^${key}=" | tail -n1 || true)"
  line="${line#${key}=}"
  echo "${line}"
}

assert_contains() {
  local output="$1"
  local token="$2"
  local entry="$3"
  if [[ "${output}" != *"${token}"* ]]; then
    echo "[agent-mode-examples-smoke][agent-mode-smoke-semantic-evidence-missing] missing token '${token}': ${entry}" >&2
    exit 1
  fi
}

selected_patterns=()
if [[ -n "${BAYMAX_AGENT_MODE_SMOKE_PATTERNS:-}" ]]; then
  IFS=',' read -r -a requested <<<"${BAYMAX_AGENT_MODE_SMOKE_PATTERNS}"
  for raw in "${requested[@]}"; do
    pattern="$(echo "${raw}" | xargs)"
    [[ -n "${pattern}" ]] || continue
    if ! contains_pattern "${pattern}" "${required_patterns[@]}"; then
      echo "[agent-mode-examples-smoke] unsupported pattern in BAYMAX_AGENT_MODE_SMOKE_PATTERNS: ${pattern}" >&2
      exit 1
    fi
    selected_patterns+=("${pattern}")
  done
else
  selected_patterns=("${required_patterns[@]}")
fi

selected_variants=()
if [[ -n "${BAYMAX_AGENT_MODE_SMOKE_VARIANTS:-}" ]]; then
  IFS=',' read -r -a variants <<<"${BAYMAX_AGENT_MODE_SMOKE_VARIANTS}"
  for raw in "${variants[@]}"; do
    variant="$(echo "${raw}" | xargs)"
    [[ -n "${variant}" ]] || continue
    if [[ "${variant}" != "minimal" && "${variant}" != "production-ish" ]]; then
      echo "[agent-mode-examples-smoke] unsupported variant: ${variant}" >&2
      exit 1
    fi
    selected_variants+=("${variant}")
  done
else
  selected_variants=("minimal" "production-ish")
fi

if (( ${#selected_patterns[@]} == 0 )); then
  echo "[agent-mode-examples-smoke] no patterns selected" >&2
  exit 1
fi
if (( ${#selected_variants[@]} == 0 )); then
  echo "[agent-mode-examples-smoke] no variants selected" >&2
  exit 1
fi

echo "[agent-mode-examples-smoke] running smoke checks for ${#selected_patterns[@]} patterns and ${#selected_variants[@]} variants"

for pattern in "${selected_patterns[@]}"; do
  declare -A variant_output=()

  for variant in "${selected_variants[@]}"; do
    entry="./examples/agent-modes/${pattern}/${variant}"
    if [[ ! -d "${entry}" ]]; then
      echo "[agent-mode-examples-smoke] missing example directory: ${entry}" >&2
      exit 1
    fi

    echo "[agent-mode-examples-smoke] go run ${entry}"
    output="$(go run "${entry}" 2>&1)" || {
      echo "${output}" >&2
      echo "[agent-mode-examples-smoke] run failed: ${entry}" >&2
      exit 1
    }
    echo "${output}"

    assert_contains "${output}" "verification.mainline_runtime_path=ok" "${entry}"
    assert_contains "${output}" "verification.semantic.anchor=" "${entry}"
    assert_contains "${output}" "verification.semantic.classification=" "${entry}"
    assert_contains "${output}" "verification.semantic.runtime_path=" "${entry}"
    assert_contains "${output}" "verification.semantic.expected_markers=" "${entry}"
    assert_contains "${output}" "verification.semantic.governance=" "${entry}"
    assert_contains "${output}" "verification.semantic.marker_count=" "${entry}"
    assert_contains "${output}" "verification.semantic.marker." "${entry}"
    assert_contains "${output}" "result.final_answer=" "${entry}"
    assert_contains "${output}" "result.signature=" "${entry}"

    variant_output["${variant}"]="${output}"
  done

  if contains_pattern "minimal" "${selected_variants[@]}" && contains_pattern "production-ish" "${selected_variants[@]}"; then
    minimal_output="${variant_output[minimal]:-}"
    production_output="${variant_output[production-ish]:-}"

    if [[ -z "${minimal_output}" || -z "${production_output}" ]]; then
      echo "[agent-mode-examples-smoke][agent-mode-smoke-semantic-evidence-missing] missing dual-variant output for pattern=${pattern}" >&2
      exit 1
    fi

    min_expected="$(extract_value "${minimal_output}" "verification.semantic.expected_markers")"
    prod_expected="$(extract_value "${production_output}" "verification.semantic.expected_markers")"
    if [[ -z "${min_expected}" || -z "${prod_expected}" || "${min_expected}" == "${prod_expected}" ]]; then
      echo "[agent-mode-examples-smoke][agent-mode-smoke-semantic-evidence-missing] expected marker set did not diverge for pattern=${pattern}" >&2
      exit 1
    fi

    min_gov="$(extract_value "${minimal_output}" "verification.semantic.governance")"
    prod_gov="$(extract_value "${production_output}" "verification.semantic.governance")"
    if [[ "${min_gov}" != "baseline" || "${prod_gov}" != "enforced" ]]; then
      echo "[agent-mode-examples-smoke][agent-mode-smoke-semantic-evidence-missing] governance marker mismatch for pattern=${pattern} minimal=${min_gov} production-ish=${prod_gov}" >&2
      exit 1
    fi

    min_sig="$(extract_value "${minimal_output}" "result.signature")"
    prod_sig="$(extract_value "${production_output}" "result.signature")"
    if [[ -z "${min_sig}" || -z "${prod_sig}" || "${min_sig}" == "${prod_sig}" ]]; then
      echo "[agent-mode-examples-smoke][agent-mode-smoke-semantic-evidence-missing] result.signature must differ between variants for pattern=${pattern}" >&2
      exit 1
    fi
  fi

done

echo "[agent-mode-examples-smoke] done"