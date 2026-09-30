package scenariosimulation

import (
	"testing"

	"github.com/FelixSeptem/baymax/core/types"
	"github.com/FelixSeptem/baymax/integration/fakes"
)

func TestBuilderUsesExistingFakesAndRejectsLiveScope(t *testing.T) {
	if _, err := NewBuilder(Scenario{Version: ScenarioVersionV1, ID: "live-case", ExecutionMode: "live"}); err == nil || err.Error() != ReasonOfflineScope {
		t.Fatalf("live scenario error = %v, want %q", err, ReasonOfflineScope)
	}
	builder, err := NewBuilder(Scenario{Version: ScenarioVersionV1, ID: "offline-case"})
	if err != nil {
		t.Fatalf("NewBuilder(): %v", err)
	}
	model := builder.Model(nil, []types.ModelEvent{{Type: types.ModelEventTypeFinalAnswer, TextDelta: "ok"}}, nil)
	if err := model.Stream(t.Context(), types.ModelRequest{}, func(types.ModelEvent) error { return nil }); err != nil {
		t.Fatalf("fake stream: %v", err)
	}
}

func TestComposedScenarioPlansRemainDeterministicWithoutBuilderOwnedTerminalState(t *testing.T) {
	scenario := Scenario{Version: ScenarioVersionV1, ID: "approval-tool-recovery", Events: []PlannedEvent{
		{ID: "approval", Sequence: 1, Kind: EventApproval, Owner: "host"},
		{ID: "tool", Sequence: 2, Kind: EventTool, Owner: "tool.local", CausationID: "approval"},
		{ID: "truncate", Sequence: 3, Kind: EventStreamTruncation, Owner: "stream", CausationID: "tool"},
		{ID: "recover", Sequence: 4, Kind: EventRecovery, Owner: "runtime", CausationID: "truncate", Reference: EvidenceReference{Owner: "runtime", ID: "completion-1"}, Expected: true},
		{ID: "cancel", Sequence: 5, Kind: EventCancel, Owner: "host", CausationID: "recover"},
	}}
	builder, err := NewBuilder(scenario)
	if err != nil {
		t.Fatalf("NewBuilder(): %v", err)
	}
	if got := builder.PlannedEvents(); len(got) != 5 || got[0].Kind != EventApproval || got[4].Kind != EventCancel {
		t.Fatalf("planned event projection = %#v", got)
	}
	_, digest1, _ := NormalizeScenario(scenario)
	_, digest2, _ := NormalizeScenario(scenario)
	if digest1 != digest2 {
		t.Fatalf("composed plan digest drift: %q != %q", digest1, digest2)
	}
	deps, err := builder.Dependencies()
	if err != nil {
		t.Fatalf("Dependencies(): %v", err)
	}
	if deps.Approval.Decision != fakes.ApprovalPending || !deps.Cancellation.Requested || deps.StreamFault == nil || deps.Recovery == nil || deps.Recovery.ReferenceID != "completion-1" {
		t.Fatalf("dependency projection = %#v", deps)
	}
}

func TestBuilderMapsApprovalDecisionsAndRejectsAmbiguousRecovery(t *testing.T) {
	builder, err := NewBuilder(Scenario{Version: ScenarioVersionV1, ID: "approval", Events: []PlannedEvent{{ID: "approval", Sequence: 1, Kind: EventApproval, Owner: "host", Decision: fakes.ApprovalGranted}}})
	if err != nil {
		t.Fatalf("NewBuilder(): %v", err)
	}
	deps, err := builder.Dependencies()
	if err != nil || deps.Approval.Decision != fakes.ApprovalGranted {
		t.Fatalf("approval projection = %#v err=%v", deps, err)
	}
	ambiguous, err := NewBuilder(Scenario{
		Version: ScenarioVersionV1,
		ID:      "recovery",
		Events: []PlannedEvent{
			{ID: "r1", Sequence: 1, Kind: EventRecovery, Owner: "runtime", Reference: EvidenceReference{Owner: "runtime", ID: "x"}},
			{ID: "r2", Sequence: 2, Kind: EventRecovery, Owner: "runtime", Reference: EvidenceReference{Owner: "runtime", ID: "y"}},
		},
	})
	if err != nil {
		t.Fatalf("NewBuilder(ambiguous): %v", err)
	}
	if _, err := ambiguous.Dependencies(); err == nil || err.Error() != ReasonScenarioSchemaDrift {
		t.Fatalf("ambiguous recovery error = %v", err)
	}
}
