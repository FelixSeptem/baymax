#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
cd "${REPO_ROOT}"

echo "[agent-mode-anti-template-contract] validating anti-template constraints for agent-mode examples"

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

template_skeleton_detected=()
semantic_ownership_missing=()
variant_behavior_not_diverged=()
structural_homogeneity_detected=()
missing_semantic_files=()
wrapper_only_entrypoints=()

declare -A hash_counts=()
declare -A hash_patterns=()

for pattern in "${required_patterns[@]}"; do
  semantic_file="examples/agent-modes/${pattern}/semantic_example.go"
  if [[ ! -f "${semantic_file}" ]]; then
    missing_semantic_files+=("${semantic_file}:missing")
    continue
  fi

  if grep -Fq "type modeSemanticModel struct" "${semantic_file}" &&
    grep -Fq "type modeSemanticStepTool struct{}" "${semantic_file}" &&
    grep -Fq "func runVariant(variant string)" "${semantic_file}"; then
    template_skeleton_detected+=("${pattern}:modeSemanticModel/modeSemanticStepTool skeleton detected")
  fi

  if grep -Fq "steps := expectedSemanticSteps(m.variant)" "${semantic_file}" &&
    grep -Fq "for idx, step := range steps" "${semantic_file}" &&
    grep -Fq "semanticToolName = \"mode_" "${semantic_file}"; then
    semantic_ownership_missing+=("${pattern}:generic expectedSemanticSteps pipeline detected")
  fi

  if grep -Fq 'governance := strings.HasPrefix(marker, "governance_") || variant == modecommon.VariantProduction' "${semantic_file}"; then
    variant_behavior_not_diverged+=("${pattern}:governance branch inferred from marker naming")
  fi

  fingerprint="$(
    sed -E 's/"([^"\\]|\\.)*"/"<s>"/g; s/[[:space:]]+//g' "${semantic_file}" \
      | sha256sum \
      | awk '{print $1}'
  )"
  hash_counts["${fingerprint}"]=$(( ${hash_counts["${fingerprint}"]:-0} + 1 ))
  if [[ -z "${hash_patterns["${fingerprint}"]:-}" ]]; then
    hash_patterns["${fingerprint}"]="${pattern}"
  else
    hash_patterns["${fingerprint}"]="${hash_patterns["${fingerprint}"]},${pattern}"
  fi
done

homogeneity_threshold="${BAYMAX_AGENT_MODE_TEMPLATE_HOMOGENEITY_THRESHOLD:-3}"
for hash in "${!hash_counts[@]}"; do
  count="${hash_counts["${hash}"]}"
  if (( count >= homogeneity_threshold )); then
    structural_homogeneity_detected+=("hash=${hash} count=${count} patterns=${hash_patterns["${hash}"]}")
  fi
done

mapfile -t main_files < <(find examples/agent-modes -mindepth 3 -maxdepth 3 -type f -name main.go | sort)
for file in "${main_files[@]}"; do
  if grep -Fq "runtimeexample.MustRun(" "${file}"; then
    wrapper_only_entrypoints+=("${file}:runtimeexample.MustRun wrapper detected")
  fi
done

if (( ${#missing_semantic_files[@]} > 0 )); then
  echo "[agent-mode-anti-template-contract][agent-mode-template-skeleton-detected] missing semantic files:" >&2
  printf '  - %s\n' "${missing_semantic_files[@]}" >&2
fi

if (( ${#template_skeleton_detected[@]} > 0 || ${#wrapper_only_entrypoints[@]} > 0 || ${#structural_homogeneity_detected[@]} > 0 )); then
  echo "[agent-mode-anti-template-contract][agent-mode-template-skeleton-detected] template skeleton regressions detected:" >&2
  printf '  - %s\n' "${template_skeleton_detected[@]}" >&2 || true
  printf '  - %s\n' "${wrapper_only_entrypoints[@]}" >&2 || true
  printf '  - %s\n' "${structural_homogeneity_detected[@]}" >&2 || true
fi

if (( ${#semantic_ownership_missing[@]} > 0 )); then
  echo "[agent-mode-anti-template-contract][agent-mode-semantic-ownership-missing] mode-owned semantic execution missing:" >&2
  printf '  - %s\n' "${semantic_ownership_missing[@]}" >&2
fi

if (( ${#variant_behavior_not_diverged[@]} > 0 )); then
  echo "[agent-mode-anti-template-contract][agent-mode-variant-behavior-not-diverged] variant behavior appears marker-only:" >&2
  printf '  - %s\n' "${variant_behavior_not_diverged[@]}" >&2
fi

if (( ${#missing_semantic_files[@]} > 0 ||
  ${#template_skeleton_detected[@]} > 0 ||
  ${#wrapper_only_entrypoints[@]} > 0 ||
  ${#structural_homogeneity_detected[@]} > 0 ||
  ${#semantic_ownership_missing[@]} > 0 ||
  ${#variant_behavior_not_diverged[@]} > 0 )); then
  exit 1
fi

echo "[agent-mode-anti-template-contract] passed"
