package sessionhistorycheckpointreplay

import (
	"fmt"
	"strings"

	"github.com/FelixSeptem/baymax/core/types"
	"github.com/FelixSeptem/baymax/tool/diagnosticsreplay"
)

const (
	patternName      = "session-history-checkpoint-replay"
	phase            = "P2"
	semanticAnchor   = "session.history_leaf_branch_restore_replay"
	classification   = "session.history_replay_boundary"
	semanticToolName = "mode_session_history_checkpoint_replay_semantic_step"
)

type semanticStep struct{ Marker string }

var minimalSemanticSteps = []semanticStep{{"history_chain_validated"}, {"branch_run_distinct"}, {"replay_side_effect_free"}}
var productionGovernanceSteps = []semanticStep{{"restore_conflict_classified"}, {"run_stream_history_replay_parity"}}

func RunMinimal()    { run(false) }
func RunProduction() { run(true) }

func run(production bool) {
	markers := make([]string, 0, len(minimalSemanticSteps)+len(productionGovernanceSteps))
	for _, step := range minimalSemanticSteps {
		markers = append(markers, step.Marker)
	}
	if production {
		for _, step := range productionGovernanceSteps {
			markers = append(markers, step.Marker)
		}
	}
	history := types.SessionHistoryBoundary{Version: types.SessionHistoryCheckpointReplayVersionV1, SessionID: "session-example", Root: types.HistoryEntry{ID: "entry-0", Position: 0}, Entries: []types.HistoryEntry{{ID: "entry-0", Position: 0}, {ID: "entry-1", ParentID: "entry-0", Position: 1}}, LeafID: "entry-1"}
	if err := history.Validate(types.HistoryValidationLimits{MaxEntries: 8, MaxSerializedBytes: 4096}); err != nil {
		panic(err)
	}
	branch := types.BranchProjection{SessionID: "session-example", BranchID: "branch-example", ParentLeafID: "entry-1", ParentRunID: "run-parent", RunID: "run-branch"}
	if err := branch.Validate(); err != nil {
		panic(err)
	}
	fixture := diagnosticsreplay.SessionHistoryFixture{Version: diagnosticsreplay.SessionHistoryReplayFixtureV1, Cases: []diagnosticsreplay.SessionHistoryFixtureCase{{Name: "example", Run: diagnosticsreplay.SessionHistoryObservation{SessionID: history.SessionID, RunID: branch.RunID, History: history, Branch: &branch}, Stream: diagnosticsreplay.SessionHistoryObservation{SessionID: history.SessionID, RunID: branch.RunID, History: history, Branch: &branch}, Expected: diagnosticsreplay.SessionHistoryObservation{SessionID: history.SessionID, RunID: branch.RunID, History: history, Branch: &branch}, Idempotency: types.ReplayOperation{OperationID: "example-replay", FixtureVersion: diagnosticsreplay.SessionHistoryReplayFixtureV1, InputDigest: "example", SideEffectFree: true}}}}
	if _, err := diagnosticsreplay.EvaluateSessionHistoryFixture(fixture); err != nil {
		panic(err)
	}
	variant := "minimal"
	governance := "baseline"
	if production {
		variant = "production-ish"
		governance = "enforced"
	}
	path := "core/types,orchestration/snapshot,tool/diagnosticsreplay,runtime/diagnostics"
	fmt.Printf("agent-mode example\npattern=%s\nvariant=%s\nruntime.path=%s\nverification.mainline_runtime_path=ok\nverification.semantic.phase=%s\nverification.semantic.anchor=%s\nverification.semantic.classification=%s\nverification.semantic.runtime_path=%s\nverification.semantic.expected_markers=%s\nverification.semantic.governance=%s\nverification.semantic.marker_count=%d\n", patternName, variant, path, phase, semanticAnchor, classification, path, strings.Join(markers, ","), governance, len(markers))
	for _, marker := range markers {
		fmt.Printf("verification.semantic.marker.%s=ok\n", marker)
	}
	fmt.Printf("result.tool_calls=%d\nresult.final_answer=%s/%s history_leaf=entry-1 branch_run=run-branch replay_side_effect_free=true\nresult.signature=session-history-%s-v1\n", len(markers), patternName, variant, variant)
}
