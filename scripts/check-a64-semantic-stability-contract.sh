#!/usr/bin/env bash
set -euo pipefail

enabled="${BAYMAX_A64_SEMANTIC_STABILITY_ENABLED:-true}"
if [[ "${enabled}" != "true" ]]; then
  echo "[a64-semantic-stability] skipped by BAYMAX_A64_SEMANTIC_STABILITY_ENABLED=${enabled}"
  exit 0
fi

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(cd "${script_dir}/.." && pwd)"
cd "${repo_root}"

log() {
  echo "[a64-semantic-stability] $*"
}

log "substep: go split semantic equivalence strong checks"
bash scripts/check-go-split-semantic-equivalence.sh

log "substep: state snapshot contract suites"
bash scripts/check-state-snapshot-contract.sh

log "substep: runtime budget admission contract suites"
bash scripts/check-runtime-budget-admission-contract.sh

log "substep: observability export + diagnostics bundle contract suites"
bash scripts/check-observability-export-and-bundle-contract.sh

log "substep: diagnostics replay contract suites"
bash scripts/check-diagnostics-replay-contract.sh

log "substep: hard-constraint semantic invariants (backpressure/fail_fast/timeout-cancel/decision-trace)"
go test ./core/runner ./runtime/config ./integration -run 'Test(RunBackpressureBlockDiagnosticsAndTimeline|RunBackpressureDropLowPriorityAllDroppedFailsFast|RunBackpressureDropLowPriorityMCPAndSkillAllDroppedFailsFast|RunAndStreamCancelPropagationSemanticsEquivalent|ActionGateRunAndStreamDenySemanticsEquivalent|ActionGateRunAndStreamTimeoutSemanticsEquivalent|EvaluateRuntimePolicyDecisionDenyPrecedence|EvaluateRuntimePolicyDecisionSameStageTieBreakDeterministic|TimeoutResolutionContract)' -count=1

log "passed"
