package handoff

import "testing"

func TestHandoffValidateAndRestore(t *testing.T) {
	h := Handoff{
		Version:            VersionV1,
		RunID:              "run-1",
		SessionID:          "session-1",
		SourceCheckpointID: "cp-1",
		Cut:                CutCheckpoint,
		Objective:          "ship feature",
		Completed:          []string{"tests"},
		Pending:            []string{"deploy"},
		Facts:              []Evidence{{Kind: EvidenceFact, Value: "file changed", SourceID: "event-1", Protected: true}},
		Inferences:         []Inference{{Value: "deploy is next", SourceIDs: []string{"event-1"}, Confidence: 0.8}},
		References:         []Reference{{Kind: ReferenceCheckpoint, ID: "cp-1"}},
		NextActions:        []Action{{Description: "deploy", SourceID: "cp-1"}},
	}
	if err := h.Validate(DefaultLimits()); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	resolver := ResolverFunc(func(ref Reference) error {
		if ref.ID == "cp-1" {
			return nil
		}
		return ErrReferenceNotFound
	})
	one, err := Restore(h, resolver)
	if err != nil {
		t.Fatalf("Restore() error = %v", err)
	}
	two, err := Restore(h, resolver)
	if err != nil || one != two {
		t.Fatalf("Restore() not idempotent: first=%+v second=%+v err=%v", one, two, err)
	}
}

func TestHandoffRejectsInvalidCutAndUnprovenInference(t *testing.T) {
	h := Handoff{Version: VersionV1, RunID: "run-1", Cut: CutUnflushedStream, Objective: "x"}
	if err := h.Validate(DefaultLimits()); err == nil {
		t.Fatal("Validate() accepted invalid cut")
	}
	h.Cut = CutCheckpoint
	h.Inferences = []Inference{{Value: "guess", Confidence: 0.5}}
	if err := h.Validate(DefaultLimits()); err == nil {
		t.Fatal("Validate() accepted inference without provenance")
	}
}

func TestBuildUsesDeterministicFallbackWhenQualityIsLow(t *testing.T) {
	h, err := Build(BuildRequest{RunID: "run-1", Cut: CutCheckpoint, Objective: "x", Messages: []Message{{Role: "user", Content: "short"}}}, BuildOptions{MinQuality: 0.99})
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	if h.Fallback == nil || h.Fallback.Reason != FallbackQualityBelowThreshold {
		t.Fatalf("fallback = %+v, want quality fallback", h.Fallback)
	}
}

func TestHandoffValidateRejectsDuplicateAndOversizedReferences(t *testing.T) {
	h := Handoff{Version: VersionV1, RunID: "run-1", Cut: CutCheckpoint, Objective: "x", References: []Reference{
		{Kind: ReferenceCheckpoint, ID: "cp-1"},
		{Kind: ReferenceCheckpoint, ID: "cp-1"},
	}}
	if err := h.Validate(DefaultLimits()); err == nil {
		t.Fatal("Validate() accepted duplicate references")
	}
	h.References = []Reference{{Kind: ReferenceCheckpoint, ID: "cp-1"}}
	if err := h.Validate(Limits{MaxSerializedSize: 8}); err == nil {
		t.Fatal("Validate() accepted oversized handoff")
	}
}

func TestRestoreRejectsMissingReferenceBeforeReturningReady(t *testing.T) {
	h := Handoff{Version: VersionV1, RunID: "run-1", Cut: CutCheckpoint, Objective: "x", References: []Reference{{Kind: ReferenceArtifact, ID: "artifact-1"}}}
	_, err := Restore(h, ResolverFunc(func(Reference) error { return ErrReferenceNotFound }))
	if err == nil {
		t.Fatal("Restore() accepted missing reference")
	}
}

func TestOwnerResolverDispatchesReferenceKinds(t *testing.T) {
	seen := make([]ReferenceKind, 0, 4)
	resolver := OwnerResolver{
		Artifact:       ResolverFunc(func(ref Reference) error { seen = append(seen, ref.Kind); return nil }),
		Checkpoint:     ResolverFunc(func(ref Reference) error { seen = append(seen, ref.Kind); return nil }),
		SessionHistory: ResolverFunc(func(ref Reference) error { seen = append(seen, ref.Kind); return nil }),
		Snapshot:       ResolverFunc(func(ref Reference) error { seen = append(seen, ref.Kind); return nil }),
	}
	h := Handoff{Version: VersionV1, RunID: "run-owner", Cut: CutCheckpoint, Objective: "resume", References: []Reference{
		{Kind: ReferenceArtifact, ID: "artifact-1"},
		{Kind: ReferenceCheckpoint, ID: "checkpoint-1"},
		{Kind: ReferenceSessionHistory, ID: "history-1"},
		{Kind: ReferenceSnapshot, ID: "snapshot-1"},
	}}
	if _, err := Restore(h, resolver); err != nil {
		t.Fatalf("Restore() error = %v", err)
	}
	if len(seen) != 4 {
		t.Fatalf("resolved references = %v, want 4", seen)
	}
}

type restoreStore struct{ values map[string]RestoreResult }

func (s *restoreStore) Lookup(id string) (RestoreResult, bool, error) {
	result, ok := s.values[id]
	return result, ok, nil
}

func (s *restoreStore) Save(id string, result RestoreResult) error {
	if s.values == nil {
		s.values = map[string]RestoreResult{}
	}
	s.values[id] = result
	return nil
}

func TestRestoreWithStoreIsIdempotentAcrossResolverInstances(t *testing.T) {
	h := Handoff{Version: VersionV1, RunID: "run-durable", Cut: CutCheckpoint, Objective: "resume", References: []Reference{{Kind: ReferenceCheckpoint, ID: "cp-1"}}}
	store := &restoreStore{}
	first, err := RestoreWithStore(h, ResolverFunc(func(Reference) error { return nil }), store)
	if err != nil {
		t.Fatalf("first restore error = %v", err)
	}
	secondResolverCalled := false
	second, err := RestoreWithStore(h, ResolverFunc(func(Reference) error { secondResolverCalled = true; return ErrReferenceNotFound }), store)
	if err != nil {
		t.Fatalf("second restore error = %v", err)
	}
	if second != first || secondResolverCalled {
		t.Fatalf("second restore = %+v, first=%+v resolver_called=%v", second, first, secondResolverCalled)
	}
}
