package scenariosimulation

import "testing"

func TestReplayClassifiesSemanticDriftAndPreservesDigests(t *testing.T) {
	scenario := Scenario{Version: ScenarioVersionV1, ID: "case-1"}
	expected := RunResult{Version: RunResultVersionV1, ScenarioID: "case-1", Execution: Verdict{Status: VerdictPass}, Evidence: Verdict{Status: VerdictPass}, Outcome: Verdict{Status: VerdictPass}, Admission: Verdict{Status: VerdictPass}, Events: []ObservedEvent{{ID: "complete", Sequence: 1, Kind: EventCompletion, Owner: "runtime", CausationID: "tool-1"}}}
	observed := expected
	observed.Events = []ObservedEvent{{ID: "complete", Sequence: 1, Kind: EventCompletion, Owner: "runtime", CausationID: "tool-2"}}
	got, err := Replay(ReplayCase{ID: "case-1", Scenario: scenario, Expected: expected, Observed: observed})
	if err == nil || err.Error() != ReasonCausationDrift {
		t.Fatalf("Replay() error = %v, want %q", err, ReasonCausationDrift)
	}
	if got.ExpectedDigest == "" || got.ObservedDigest == "" || got.ExpectedDigest == got.ObservedDigest {
		t.Fatalf("replay did not preserve distinct digests: %#v", got)
	}
}

func TestReplayBatchIsIdempotentAndRejectsConflictingDuplicateIdentity(t *testing.T) {
	base := ReplayCase{ID: "stable", Scenario: Scenario{Version: ScenarioVersionV1, ID: "stable"}, Expected: RunResult{Version: RunResultVersionV1, ScenarioID: "stable"}, Observed: RunResult{Version: RunResultVersionV1, ScenarioID: "stable"}}
	if _, err := ReplayBatch([]ReplayCase{base, base}); err != nil {
		t.Fatalf("identical duplicate should be idempotent: %v", err)
	}
	conflict := base
	conflict.Observed.Execution = Verdict{Status: VerdictFail}
	if _, err := ReplayBatch([]ReplayCase{base, conflict}); err == nil || err.Error() != ReasonDuplicateConflict {
		t.Fatalf("conflicting duplicate error = %v, want %q", err, ReasonDuplicateConflict)
	}
}

func TestReplayClassifiesEvidenceCompletionOutcomeAndAdmissionDrift(t *testing.T) {
	base := RunResult{
		Version: RunResultVersionV1, ScenarioID: "case",
		Execution: Verdict{Status: VerdictPass}, Evidence: Verdict{Status: VerdictPass},
		Outcome: Verdict{Status: VerdictPass}, Admission: Verdict{Status: VerdictPass},
		Events:             []ObservedEvent{{ID: "done", Sequence: 1, Kind: EventCompletion, Owner: "runner"}},
		EvidenceReferences: []EvidenceReference{{Owner: "host", ID: "proof", Digest: "sha256:1"}},
		OutcomeReference:   &OutcomeReference{Owner: "host", ID: "outcome", Digest: "sha256:outcome"},
		Completion:         &CompletionReference{ID: "done", Committed: true},
	}
	tests := []struct {
		name   string
		mutate func(*RunResult)
		want   string
	}{
		{name: "evidence", mutate: func(r *RunResult) { r.EvidenceReferences[0].Digest = "sha256:2" }, want: ReasonEvidenceIncomplete},
		{name: "completion", mutate: func(r *RunResult) { r.Completion.Committed = false }, want: ReasonCompletionDrift},
		{name: "outcome", mutate: func(r *RunResult) { r.OutcomeReference.ID = "other" }, want: ReasonOutcomeDrift},
		{name: "admission", mutate: func(r *RunResult) { r.Admission.Status = VerdictIndeterminate }, want: ReasonAdmissionDrift},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			observed := base
			observed.Events = append([]ObservedEvent(nil), base.Events...)
			observed.EvidenceReferences = append([]EvidenceReference(nil), base.EvidenceReferences...)
			outcome := *base.OutcomeReference
			observed.OutcomeReference = &outcome
			completion := *base.Completion
			observed.Completion = &completion
			tc.mutate(&observed)
			_, err := Replay(ReplayCase{ID: "case", Scenario: Scenario{Version: ScenarioVersionV1, ID: "case"}, Expected: base, Observed: observed})
			if err == nil || err.Error() != tc.want {
				t.Fatalf("Replay() error=%v want=%q", err, tc.want)
			}
		})
	}
}

func TestRunStreamNormalizedResultsHaveSemanticParity(t *testing.T) {
	run := RunResult{Version: RunResultVersionV1, ScenarioID: "parity", Execution: Verdict{Status: VerdictPass}, Evidence: Verdict{Status: VerdictPass}, Outcome: Verdict{Status: VerdictIndeterminate, Reason: ReasonOutcomeIndeterminate}, Admission: Verdict{Status: VerdictIndeterminate, Reason: ReasonAdmissionIndeterminate}, Events: []ObservedEvent{{ID: "complete", Sequence: 1, Kind: EventCompletion, Owner: "runner"}}}
	stream := run
	stream.Events = []ObservedEvent{{ID: "chunk-1", Sequence: 1, Kind: "stream_chunk", Owner: "provider"}, {ID: "complete", Sequence: 2, Kind: EventCompletion, Owner: "runner"}}
	if !SemanticParity(run, stream) {
		t.Fatalf("Run and Stream semantics diverged after allowed stream-only event normalization")
	}
	stream.Admission = Verdict{Status: VerdictPass}
	if SemanticParity(run, stream) {
		t.Fatalf("Run and Stream admission drift was not detected")
	}
}
