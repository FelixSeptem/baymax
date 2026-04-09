#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
cd "${REPO_ROOT}"

echo "[agent-mode-real-runtime-semantic-contract] validating agent mode semantic runtime contract"

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

runtime_file="examples/agent-modes/internal/runtimeexample/runtime.go"
spec_file="examples/agent-modes/internal/runtimeexample/specs.go"
matrix_file="examples/agent-modes/MATRIX.md"

shared_semantic_engine=()
semantic_ownership_missing=()
runtime_path_missing=()

if [[ ! -f "${spec_file}" ]]; then
  semantic_ownership_missing+=("${spec_file}:missing")
fi
if [[ ! -f "${matrix_file}" ]]; then
  runtime_path_missing+=("${matrix_file}:missing")
fi

if [[ -f "${runtime_file}" ]]; then
  if grep -Eq 'func[[:space:]]+MustRun\(|type[[:space:]]+semanticModel|semanticStepTool' "${runtime_file}"; then
    shared_semantic_engine+=("${runtime_file}:legacy-shared-semantic-engine")
  fi
fi

mapfile -t main_files < <(find examples/agent-modes -mindepth 3 -maxdepth 3 -type f -name main.go | sort)
if (( ${#main_files[@]} == 0 )); then
  semantic_ownership_missing+=("examples/agent-modes/*/*/main.go:missing")
fi

for file in "${main_files[@]}"; do
  pattern="$(basename "$(dirname "$(dirname "${file}")")")"
  variant="$(basename "$(dirname "${file}")")"

  if grep -Fq 'runtimeexample.MustRun(' "${file}"; then
    shared_semantic_engine+=("${file}:shared-wrapper-detected")
  fi

  expected_import="modeimpl \"github.com/FelixSeptem/baymax/examples/agent-modes/${pattern}\""
  if ! grep -Fq "${expected_import}" "${file}"; then
    semantic_ownership_missing+=("${file}:missing-mode-owned-import:${expected_import}")
  fi

  expected_call='modeimpl.RunProduction()'
  if [[ "${variant}" == "minimal" ]]; then
    expected_call='modeimpl.RunMinimal()'
  fi
  if ! grep -Fq "${expected_call}" "${file}"; then
    semantic_ownership_missing+=("${file}:missing-mode-owned-entry:${expected_call}")
  fi

done

required_runtime_tokens=(
  "verification.mainline_runtime_path="
  "verification.semantic.anchor="
  "verification.semantic.classification="
  "verification.semantic.runtime_path="
  "verification.semantic.expected_markers="
  "verification.semantic.governance="
  "verification.semantic.marker_count="
  "verification.semantic.marker."
)

for pattern in "${required_patterns[@]}"; do
  semantic_file="examples/agent-modes/${pattern}/semantic_example.go"
  if [[ ! -f "${semantic_file}" ]]; then
    semantic_ownership_missing+=("${semantic_file}:missing")
    continue
  fi

  if ! grep -Fq "patternName      = \"${pattern}\"" "${semantic_file}"; then
    semantic_ownership_missing+=("${semantic_file}:missing-pattern-constant")
  fi

  required_semantic_tokens=(
    "func RunMinimal()"
    "func RunProduction()"
    "var minimalSemanticSteps"
    "var productionGovernanceSteps"
    "semanticToolName = \"mode_"
  )
  for token in "${required_semantic_tokens[@]}"; do
    if ! grep -Fq "${token}" "${semantic_file}"; then
      semantic_ownership_missing+=("${semantic_file}:missing-token:${token}")
    fi
  done

  marker_count="$(grep -c 'Marker:' "${semantic_file}" || true)"
  if [[ -z "${marker_count}" || "${marker_count}" -lt 5 ]]; then
    semantic_ownership_missing+=("${semantic_file}:insufficient-semantic-steps")
  fi

  for token in "${required_runtime_tokens[@]}"; do
    if ! grep -Fq "${token}" "${semantic_file}"; then
      runtime_path_missing+=("${semantic_file}:missing-token:${token}")
    fi
  done

done

if [[ -f "${spec_file}" ]]; then
  duplicate_anchors="$(grep -oE 'SemanticAnchor:[[:space:]]+"[^"]+"' "${spec_file}" | sed -E 's/^SemanticAnchor:[[:space:]]+"(.*)"/\1/' | sort | uniq -d || true)"
  if [[ -n "${duplicate_anchors}" ]]; then
    while IFS= read -r anchor; do
      [[ -n "${anchor}" ]] || continue
      semantic_ownership_missing+=("${spec_file}:duplicate-semantic-anchor:${anchor}")
    done <<< "${duplicate_anchors}"
  fi

  for pattern in "${required_patterns[@]}"; do
    if ! grep -Fq "\"${pattern}\":" "${spec_file}"; then
      semantic_ownership_missing+=("${spec_file}:missing-pattern-spec:${pattern}")
    fi
  done
fi

if [[ -f "${matrix_file}" ]]; then
  if ! grep -Fq "semantic_anchor -> runtime_path_evidence -> expected_verification_markers" "${matrix_file}" &&
    ! grep -Fq "semantic_anchor -> runtime_path_evidence" "${matrix_file}"; then
    runtime_path_missing+=("${matrix_file}:missing-semantic-runtime-columns")
  fi

  for pattern in "${required_patterns[@]}"; do
    row="$(grep -E "^\| \`${pattern}\` \|" "${matrix_file}" || true)"
    if [[ -z "${row}" ]]; then
      runtime_path_missing+=("${matrix_file}:missing-row:${pattern}")
      continue
    fi
    if [[ "${row}" != *"runtime/config"* ]]; then
      runtime_path_missing+=("${matrix_file}:missing-runtime-path-evidence:${pattern}")
    fi
    if [[ "${row}" != *"minimal:"* || "${row}" != *"production-ish:"* ]]; then
      runtime_path_missing+=("${matrix_file}:missing-expected-marker-evidence:${pattern}")
    fi
  done
fi

if (( ${#shared_semantic_engine[@]} > 0 )); then
  echo "[agent-mode-real-runtime-semantic-contract][agent-mode-shared-semantic-engine-detected] shared semantic engine regressions detected:" >&2
  printf '  - %s\n' "${shared_semantic_engine[@]}" >&2
fi

if (( ${#semantic_ownership_missing[@]} > 0 )); then
  echo "[agent-mode-real-runtime-semantic-contract][agent-mode-semantic-ownership-missing] per-mode semantic ownership is incomplete:" >&2
  printf '  - %s\n' "${semantic_ownership_missing[@]}" >&2
fi

if (( ${#runtime_path_missing[@]} > 0 )); then
  echo "[agent-mode-real-runtime-semantic-contract][agent-mode-missing-runtime-path-evidence] runtime path evidence is incomplete:" >&2
  printf '  - %s\n' "${runtime_path_missing[@]}" >&2
fi

if (( ${#shared_semantic_engine[@]} > 0 || ${#semantic_ownership_missing[@]} > 0 || ${#runtime_path_missing[@]} > 0 )); then
  exit 1
fi

echo "[agent-mode-real-runtime-semantic-contract] passed"
