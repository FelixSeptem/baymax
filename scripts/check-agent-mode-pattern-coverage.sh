#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
cd "${REPO_ROOT}"

MATRIX_PATH="examples/agent-modes/MATRIX.md"
ROOT_PATH="examples/agent-modes"

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

required_families=(
  "agent"
  "workflow"
  "rag"
  "mapreduce"
  "structured-output"
  "multi-agents"
  "skill"
  "mcp"
  "react"
  "hitl"
  "context"
  "sandbox"
  "realtime"
)

echo "[agent-mode-pattern-coverage] validating matrix and skeleton coverage"

if [[ ! -f "${MATRIX_PATH}" ]]; then
  echo "[agent-mode-pattern-coverage] missing matrix: ${MATRIX_PATH}" >&2
  exit 1
fi
if ! grep -q "pattern -> phase -> a71_scope -> a71_status -> semantic_anchor -> runtime_path_evidence -> expected_verification_markers -> minimal -> production-ish -> contracts -> gates -> replay" "${MATRIX_PATH}"; then
  echo "[agent-mode-pattern-coverage] matrix missing canonical column declaration" >&2
  exit 1
fi

declare -A family_hits=()
for family in "${required_families[@]}"; do
  family_hits["${family}"]=0
done

missing_matrix_rows=()
missing_files=()
missing_semantic_evidence=()

for pattern in "${required_patterns[@]}"; do
  row="$(grep -E "^\\| \`${pattern}\` \\|" "${MATRIX_PATH}" || true)"
  if [[ -z "${row}" ]]; then
    missing_matrix_rows+=("${pattern}")
  else
    if [[ "${row}" != *"runtime/config"* || "${row}" != *"minimal:"* || "${row}" != *"production-ish:"* ]]; then
      missing_semantic_evidence+=("${pattern}")
    fi
  fi
  for variant in minimal production-ish; do
    base="${ROOT_PATH}/${pattern}/${variant}"
    [[ -d "${base}" ]] || missing_files+=("${base}/")
    [[ -f "${base}/main.go" ]] || missing_files+=("${base}/main.go")
    [[ -f "${base}/README.md" ]] || missing_files+=("${base}/README.md")
  done

  if [[ "${pattern}" == *agent* ]]; then
    family_hits["agent"]=1
  fi
  if [[ "${pattern}" == workflow-* ]]; then
    family_hits["workflow"]=1
  fi
  if [[ "${pattern}" == rag-* ]]; then
    family_hits["rag"]=1
  fi
  if [[ "${pattern}" == mapreduce-* ]]; then
    family_hits["mapreduce"]=1
  fi
  if [[ "${pattern}" == structured-output-* ]]; then
    family_hits["structured-output"]=1
  fi
  if [[ "${pattern}" == multi-agents-* ]]; then
    family_hits["multi-agents"]=1
  fi
  if [[ "${pattern}" == skill-* ]]; then
    family_hits["skill"]=1
  fi
  if [[ "${pattern}" == mcp-* || "${pattern}" == custom-adapter-mcp-* ]]; then
    family_hits["mcp"]=1
  fi
  if [[ "${pattern}" == react-* ]]; then
    family_hits["react"]=1
  fi
  if [[ "${pattern}" == hitl-* ]]; then
    family_hits["hitl"]=1
  fi
  if [[ "${pattern}" == context-* ]]; then
    family_hits["context"]=1
  fi
  if [[ "${pattern}" == sandbox-* ]]; then
    family_hits["sandbox"]=1
  fi
  if [[ "${pattern}" == realtime-* ]]; then
    family_hits["realtime"]=1
  fi
done

missing_families=()
for family in "${required_families[@]}"; do
  if [[ "${family_hits[${family}]}" != "1" ]]; then
    missing_families+=("${family}")
  fi
done

if (( ${#missing_matrix_rows[@]} > 0 )); then
  echo "[agent-mode-pattern-coverage] missing matrix rows:" >&2
  printf '  - %s\n' "${missing_matrix_rows[@]}" >&2
fi

if (( ${#missing_files[@]} > 0 )); then
  echo "[agent-mode-pattern-coverage] missing pattern skeleton files:" >&2
  printf '  - %s\n' "${missing_files[@]}" >&2
fi

if (( ${#missing_families[@]} > 0 )); then
  echo "[agent-mode-pattern-coverage] missing required mode families:" >&2
  printf '  - %s\n' "${missing_families[@]}" >&2
fi
if (( ${#missing_semantic_evidence[@]} > 0 )); then
  echo "[agent-mode-pattern-coverage] rows missing semantic/runtime evidence columns:" >&2
  printf '  - %s\n' "${missing_semantic_evidence[@]}" >&2
fi

if (( ${#missing_matrix_rows[@]} > 0 || ${#missing_files[@]} > 0 || ${#missing_families[@]} > 0 || ${#missing_semantic_evidence[@]} > 0 )); then
  exit 1
fi

echo "[agent-mode-pattern-coverage] coverage is complete"
