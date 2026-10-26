#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
cd "${REPO_ROOT}"

if ! command -v go >/dev/null 2>&1; then
  echo "[host-context-assembly-handoff][unavailable] go command is required" >&2
  exit 2
fi

if [[ -z "${GOCACHE:-}" ]]; then
  GOCACHE="${REPO_ROOT}/.gocache"
fi
export GOCACHE

# host_context_projection is the stable dual-platform contract anchor.

echo "[host-context-assembly-handoff] projection contract, resolver, and privacy"
go test ./context/hostprojection ./context/assembler -count=1 -run 'TestAdmit|TestResolve|TestAdmissionGapFixtureUsesSyntheticOpaqueReferences|TestAssembler|TestProjectedHandoffDoesNotPersistResolvedBody'

echo "[host-context-assembly-handoff] Run/Stream parity and explicit restore"
go test ./core/runner -count=1 -run 'TestRunAndStreamUseEquivalentHostProjectionAssembly|TestHostProjectionPreservesReadOnlyBudgetProjectionAndExistingPressureTerminal|TestRunAndStreamNormalizeProjectionBudgetHandoffAndRejectionEquivalently|TestEngineRestoreProjectedHandoffRequiresExplicitCurrentResolver|TestEquivalentRunStreamProjectedHandoffsRestoreToSameNormalizedOutcome'

echo "[host-context-assembly-handoff] deterministic offline replay"
go test ./tool/diagnosticsreplay -count=2 -run 'TestReplayHostContextProjection'

echo "[host-context-assembly-handoff] offline example"
go test ./examples/host-context-assembly -count=1 -run 'TestOfflineExampleEmitsContractMarkers'

echo "[host-context-assembly-handoff] composite owner binding and capability evidence"
# The offline marker is asserted by the example test; keep the source-owned
# cross-step rejection visible in this blocking gate's contract vocabulary.
echo "[host-context-assembly-handoff] host_context.workflow_cross_step_rejected"
go test ./orchestration/composer ./orchestration/teams ./orchestration/workflow ./orchestration/planbridge -count=1 -run 'TestComposerResumeChildContinuationResolvesBoundContextBeforeExecution|TestComposerResumeChildContinuationRechecksSchedulerFenceAfterResolver|TestTeamContinuationResolvesContextOnlyForBoundPendingMember|TestTeamContinuationRejectsForeignMemberProjectionBeforeResolverOrClaim|TestTeamContextResumeRechecksDeadlineAfterResolver|TestTeamContextResumeRunStreamParity|TestTeamContextResumeReopensFileOwnerWithBoundMember|TestTeamPauseRejectsRemoteMemberContextBinding|TestTeamVoteContinuationResolvesBoundPendingMemberWithoutRerunningSettledVote|TestWorkflowContinuationRejectsForeignStepProjectionBeforeResolverOrClaim|TestWorkflowContinuationResolvesBoundContextOnlyForPendingStep|TestWorkflowContextResumeRunStreamParity|TestWorkflowContextResumeRechecksExpiryAfterResolver|TestCapabilityManifestDeclaresContextDimensionsIndependently'

echo "[host-context-assembly-handoff] contribution boundary"
go test ./tool/contributioncheck -count=1 -run 'TestHostContextProjectionGateScriptsAreEquivalentAndBlocking|TestHostContextProjectionOwnershipBoundary'

echo "[host-context-assembly-handoff] passed"
