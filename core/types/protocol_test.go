package types

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestProtocolReferencesValidateAndRoundTrip(t *testing.T) {
	createdAt := time.Date(2026, time.August, 12, 9, 30, 0, 0, time.UTC)
	refs := []ProtocolReference{
		SessionRef{SessionID: "session-1", Source: ProtocolSourceRunner},
		RunRef{RunID: "run-1", SessionID: "session-1", State: RunStateWorking, Source: ProtocolSourceRunner},
		StepRef{StepID: "step-1", RunID: "run-1", SessionID: "session-1", Kind: ProtocolStepKindTool, Source: ProtocolSourceRunner},
		EventEnvelope{EventID: "event-1", RunID: "run-1", StepID: "step-1", Source: ProtocolSourceRealtime, Kind: ProtocolEventKindProgress, Time: createdAt, Sequence: 2, Payload: map[string]any{"state": "working"}},
		ArtifactRef{ArtifactID: "artifact-1", Type: "markdown", Locator: "artifact://memo-1", Digest: "sha256:abc", ProducedByRunID: "run-1", ProducedByStepID: "step-1"},
		CheckpointRef{CheckpointID: "checkpoint-1", SchemaVersion: "state_session_snapshot.v1", SourceComponent: "composer", Digest: "abc", RunID: "run-1", SessionID: "session-1"},
	}

	for _, ref := range refs {
		if err := ref.ValidateProtocolReference(); err != nil {
			t.Fatalf("validate %T: %v", ref, err)
		}
		encoded, err := json.Marshal(ref)
		if err != nil {
			t.Fatalf("marshal %T: %v", ref, err)
		}
		if len(encoded) == 0 {
			t.Fatalf("marshal %T produced empty payload", ref)
		}
	}
}

func TestProtocolReferencesRejectMissingRequiredIdentifiers(t *testing.T) {
	tests := []struct {
		name string
		ref  ProtocolReference
		want string
	}{
		{name: "session", ref: SessionRef{}, want: "session_id is required"},
		{name: "run", ref: RunRef{State: RunStateWorking}, want: "run_id is required"},
		{name: "step", ref: StepRef{RunID: "run-1", Kind: ProtocolStepKindTool}, want: "step_id is required"},
		{name: "event", ref: EventEnvelope{RunID: "run-1", Source: ProtocolSourceRunner, Kind: ProtocolEventKindProgress, Time: time.Now()}, want: "event_id is required"},
		{name: "artifact", ref: ArtifactRef{Type: "markdown", Locator: "artifact://memo-1"}, want: "artifact_id is required"},
		{name: "checkpoint", ref: CheckpointRef{SchemaVersion: "v1", SourceComponent: "runner", Digest: "abc"}, want: "checkpoint_id is required"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.ref.ValidateProtocolReference()
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("ValidateProtocolReference() error = %v, want %q", err, tt.want)
			}
		})
	}
}

func TestCheckpointHistoryAndWorkspaceProvenanceValidation(t *testing.T) {
	root := CheckpointRef{CheckpointID: "checkpoint-root", SchemaVersion: "state_session_snapshot.v1", SourceComponent: "composer", Digest: "digest-root", SessionID: "session-1", Relation: CheckpointRelationRoot, HistoryIndex: 0}
	derived := root
	derived.CheckpointID = "checkpoint-derived"
	derived.Digest = "digest-derived"
	derived.Relation = CheckpointRelationDerived
	derived.ParentCheckpointID = root.CheckpointID
	derived.HistoryIndex = 1
	derived.WorkspaceProvenance = &WorkspaceProvenance{WorkspaceID: "workspace-1", ChangeSetID: "change-1", BeforeIntegrity: "before-1", AfterIntegrity: "after-1", ProducedByRunID: "run-1", ProducedByStepID: "step-1", CheckpointID: derived.CheckpointID}
	if err := ValidateCheckpointHistory([]CheckpointRef{root, derived}); err != nil {
		t.Fatalf("ValidateCheckpointHistory() error = %v", err)
	}
	if err := derived.ValidateProtocolReference(); err != nil {
		t.Fatalf("derived.ValidateProtocolReference() error = %v", err)
	}
	broken := derived
	broken.ParentCheckpointID = "missing"
	if err := ValidateCheckpointHistory([]CheckpointRef{root, broken}); err == nil || !strings.Contains(err.Error(), "history_disconnected") {
		t.Fatalf("missing parent error = %v, want history_disconnected", err)
	}
	conflict := derived
	conflict.ReplayKey = "replay-1"
	other := conflict
	other.Digest = "different"
	if err := ValidateCheckpointReplay(conflict, other); err == nil || !strings.Contains(err.Error(), "replay_conflict") {
		t.Fatalf("replay conflict error = %v, want replay_conflict", err)
	}
	if err := ValidateWorkspaceIntegrity("different", *derived.WorkspaceProvenance); err == nil || !strings.Contains(err.Error(), "integrity_drift") {
		t.Fatalf("workspace drift error = %v, want integrity_drift", err)
	}
}

func TestCheckpointHistoryLeafAssociationAndBranchRunValidation(t *testing.T) {
	ref := CheckpointRef{
		CheckpointID:       "checkpoint-branch",
		SchemaVersion:      "state_session_snapshot.v1",
		SourceComponent:    "composer",
		Digest:             "digest-branch",
		RunID:              "run-branch",
		SessionID:          "session-1",
		Relation:           CheckpointRelationBranch,
		ParentCheckpointID: "checkpoint-root",
		BranchID:           "branch-1",
		HistoryRootID:      "entry-0",
		HistoryLeafID:      "entry-1",
		HistoryDigest:      "history-digest",
	}
	if err := ref.ValidateProtocolReference(); err != nil {
		t.Fatalf("branch checkpoint with history association should validate: %v", err)
	}
	ref.HistoryLeafID = ""
	if err := ref.ValidateProtocolReference(); err == nil || !strings.Contains(err.Error(), "history_leaf_id") {
		t.Fatalf("branch checkpoint without history leaf should fail: %v", err)
	}
}

func TestCheckpointReferenceRemainsBackwardCompatibleWithoutProvenance(t *testing.T) {
	ref := CheckpointRef{CheckpointID: "checkpoint-1", SchemaVersion: "v1", SourceComponent: "runner", Digest: "digest"}
	if err := ref.ValidateProtocolReference(); err != nil {
		t.Fatalf("legacy checkpoint validation error = %v", err)
	}
	raw, err := json.Marshal(ref)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), "workspace_provenance") || strings.Contains(string(raw), "relation") {
		t.Fatalf("legacy checkpoint unexpectedly serialized optional fields: %s", raw)
	}
}

func TestRunLifecycleValidatesTransitions(t *testing.T) {
	tests := []struct {
		name string
		from RunState
		to   RunState
		ok   bool
	}{
		{name: "submit work", from: RunStateSubmitted, to: RunStateWorking, ok: true},
		{name: "work input", from: RunStateWorking, to: RunStateInputRequired, ok: true},
		{name: "resume", from: RunStateInputRequired, to: RunStateWorking, ok: true},
		{name: "complete", from: RunStateWorking, to: RunStateCompleted, ok: true},
		{name: "idempotent cancel", from: RunStateCanceled, to: RunStateCanceled, ok: true},
		{name: "terminal resume", from: RunStateCompleted, to: RunStateWorking, ok: false},
		{name: "skip submit", from: RunStateSubmitted, to: RunStateCompleted, ok: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateRunStateTransition(tt.from, tt.to)
			if tt.ok && err != nil {
				t.Fatalf("ValidateRunStateTransition(%q, %q): %v", tt.from, tt.to, err)
			}
			if !tt.ok && err == nil {
				t.Fatalf("ValidateRunStateTransition(%q, %q) succeeded", tt.from, tt.to)
			}
		})
	}
}

func TestRetryCreatesCausalRunReference(t *testing.T) {
	retry, err := NewRetryRunRef(RunRef{RunID: "run-1", SessionID: "session-1", State: RunStateFailed, Source: ProtocolSourceRunner}, "run-2")
	if err != nil {
		t.Fatalf("NewRetryRunRef() error = %v", err)
	}
	if retry.RunID != "run-2" || retry.CausationID != "run-1" || retry.State != RunStateSubmitted {
		t.Fatalf("retry = %#v", retry)
	}
}

func TestRecoverableStepOutcomePreservesFailFastBoundaries(t *testing.T) {
	if !CanRepresentRecoverableStepOutcome(ProtocolFailureTool, true) {
		t.Fatal("recoverable tool failure must be representable as step outcome")
	}
	if !CanRepresentRecoverableStepOutcome(ProtocolFailureBusiness, true) {
		t.Fatal("recoverable business failure must be representable as step outcome")
	}
	if CanRepresentRecoverableStepOutcome(ProtocolFailureTool, false) {
		t.Fatal("owner-declared non-recoverable tool failure must remain fail-fast")
	}

	for _, boundary := range []ProtocolFailureBoundary{
		ProtocolFailureConfiguration,
		ProtocolFailureSecurity,
		ProtocolFailurePermission,
		ProtocolFailureValidation,
		ProtocolFailureSnapshotCompatibility,
		ProtocolFailureModuleBoundary,
	} {
		if CanRepresentRecoverableStepOutcome(boundary, true) {
			t.Fatalf("%q must remain fail-fast", boundary)
		}
	}
}

func TestProtocolMapsRunnerRequestResultAndToolSteps(t *testing.T) {
	req := RunRequest{RunID: "run-1", SessionID: "session-1"}
	run, err := NewProtocolRunFromRequest(req, RunStateWorking)
	if err != nil {
		t.Fatalf("NewProtocolRunFromRequest() error = %v", err)
	}
	if run.RunID != "run-1" || run.SessionID != "session-1" || run.Source != ProtocolSourceRunner {
		t.Fatalf("run = %#v", run)
	}

	result := RunResult{RunID: "run-1", ToolCalls: []ToolCallSummary{{CallID: "call-1", Name: "search"}}}
	terminal, steps, err := MapRunnerResultToProtocol(result, "session-1")
	if err != nil {
		t.Fatalf("MapRunnerResultToProtocol() error = %v", err)
	}
	if terminal.State != RunStateCompleted || len(steps) != 1 || steps[0].StepID != "call-1" || steps[0].Kind != ProtocolStepKindTool {
		t.Fatalf("terminal=%#v steps=%#v", terminal, steps)
	}
}

func TestProtocolMapsStandardAndRealtimeEvents(t *testing.T) {
	timestamp := time.Date(2026, time.August, 12, 10, 0, 0, 0, time.UTC)
	standard, err := MapEventToProtocol(Event{
		Version:   EventSchemaVersionV1,
		Type:      EventTypeActionTimeline,
		RunID:     "run-1",
		Iteration: 2,
		CallID:    "call-1",
		Time:      timestamp,
		Payload:   map[string]any{"phase": "tool", "status": "running"},
	}, ProtocolSourceTimeline)
	if err != nil {
		t.Fatalf("MapEventToProtocol() error = %v", err)
	}
	if standard.EventID == "" || standard.Sequence != 0 || standard.Source != ProtocolSourceTimeline || standard.StepID != "call-1" {
		t.Fatalf("standard = %#v", standard)
	}

	realtime, err := MapRealtimeEventToProtocol(RealtimeEventEnvelope{
		EventID:   "event-1",
		SessionID: "session-1",
		RunID:     "run-1",
		Seq:       3,
		Type:      RealtimeEventTypeInterrupt,
		TS:        timestamp,
		Payload:   map[string]any{"resume_cursor": "cursor-1"},
	})
	if err != nil {
		t.Fatalf("MapRealtimeEventToProtocol() error = %v", err)
	}
	if realtime.EventID != "event-1" || realtime.Sequence != 3 || realtime.Kind != ProtocolEventKindState || realtime.Source != ProtocolSourceRealtime {
		t.Fatalf("realtime = %#v", realtime)
	}
}

func TestArtifactReferenceProjectsDeferredHandoffArtifact(t *testing.T) {
	ref, err := MapHandoffArtifactToProtocol(IsolateHandoffArtifact{ID: "artifact-1", Type: "markdown", Locator: "artifact://memo-1", Body: "secret"}, "run-1", "step-1")
	if err != nil {
		t.Fatalf("MapHandoffArtifactToProtocol() error = %v", err)
	}
	if ref.ArtifactID != "artifact-1" || ref.Locator != "artifact://memo-1" || ref.ProducedByRunID != "run-1" || ref.ProducedByStepID != "step-1" {
		t.Fatalf("ref = %#v", ref)
	}
}

func TestProtocolDescriptorNegotiatesCapabilities(t *testing.T) {
	descriptor := ProtocolDescriptor{
		ProtocolName:   AgentRuntimeProtocolName,
		ProfileVersion: "v1",
		RuntimeID:      "runner",
		Capabilities: []ProtocolCapability{
			{Name: "interrupt_resume", Version: "v1"},
		},
		Actions: []ProtocolAction{ProtocolActionCancel, ProtocolActionRetry},
	}
	if err := descriptor.Validate(); err != nil {
		t.Fatalf("descriptor.Validate() error = %v", err)
	}

	accepted, err := NegotiateProtocolCapabilities(descriptor, ProtocolCapabilityRequest{
		Required: []string{"interrupt_resume"},
	})
	if err != nil {
		t.Fatalf("NegotiateProtocolCapabilities() error = %v", err)
	}
	if accepted.Decision != ProtocolCapabilityDecisionAccepted || accepted.ReasonCode != "" {
		t.Fatalf("accepted = %#v", accepted)
	}

	downgraded, err := NegotiateProtocolCapabilities(descriptor, ProtocolCapabilityRequest{
		Optional: []string{"workspace_provenance"},
		Strategy: ProtocolCapabilityStrategyBestEffort,
	})
	if err != nil {
		t.Fatalf("NegotiateProtocolCapabilities() downgrade error = %v", err)
	}
	if downgraded.Decision != ProtocolCapabilityDecisionDowngraded || downgraded.ReasonCode != ProtocolReasonCapabilityOptionalDowngraded {
		t.Fatalf("downgraded = %#v", downgraded)
	}

	_, err = NegotiateProtocolCapabilities(descriptor, ProtocolCapabilityRequest{Required: []string{"workspace_provenance"}})
	if err == nil || !strings.Contains(err.Error(), ProtocolReasonCapabilityMissingRequired) {
		t.Fatalf("required capability error = %v, want %q", err, ProtocolReasonCapabilityMissingRequired)
	}

	_, err = NegotiateProtocolCapabilities(descriptor, ProtocolCapabilityRequest{
		Optional: []string{"workspace_provenance"},
		Strategy: ProtocolCapabilityStrategyFailFast,
	})
	if err == nil || !strings.Contains(err.Error(), ProtocolReasonCapabilityMissingRequired) {
		t.Fatalf("fail-fast optional error = %v, want %q", err, ProtocolReasonCapabilityMissingRequired)
	}
	_, err = NegotiateProtocolCapabilities(descriptor, ProtocolCapabilityRequest{
		Required: []string{"workspace_provenance"},
		Optional: []string{"workspace_provenance"},
	})
	if err == nil || strings.Count(err.Error(), "workspace_provenance") != 1 {
		t.Fatalf("duplicate missing capability should be canonicalized: %v", err)
	}

	versioned, err := NegotiateProtocolCapabilities(descriptor, ProtocolCapabilityRequest{
		ProfileVersion: "v1",
		Required:       []string{"interrupt_resume"},
	})
	if err != nil || versioned.ProfileVersion != "v1" {
		t.Fatalf("versioned negotiation = %#v err=%v", versioned, err)
	}
	_, err = NegotiateProtocolCapabilities(descriptor, ProtocolCapabilityRequest{
		ProfileVersion: "v2",
	})
	if err == nil || !strings.Contains(err.Error(), ProtocolReasonCapabilityProfileMismatch) {
		t.Fatalf("profile mismatch error = %v, want %q", err, ProtocolReasonCapabilityProfileMismatch)
	}
}

func TestProtocolSessionContextIsBoundedAndReferenceOnly(t *testing.T) {
	context := ProtocolSessionContext{
		Session: SessionRef{SessionID: "session-1", Source: ProtocolSourceRunner},
		Scope:   ProtocolContextScopeSession,
		Agent:   &ProtocolParticipantRef{ID: "agent-1", Role: "primary", Source: ProtocolSourceRunner},
		Participants: []ProtocolParticipantRef{
			{ID: "participant-1", Role: "observer", Source: ProtocolSourceRunner},
		},
		Metadata: map[string]string{"requester": "host"},
	}
	limits := ProtocolContextLimits{
		AllowedMetadataKeys:   []string{"requester"},
		MaxParticipants:       2,
		MaxMetadataEntries:    2,
		MaxMetadataKeyBytes:   32,
		MaxMetadataValueBytes: 32,
		MaxSerializedBytes:    512,
	}
	if err := context.Validate(limits); err != nil {
		t.Fatalf("context.Validate() error = %v", err)
	}

	context.Metadata["prompt"] = "embedded history is not allowed"
	if err := context.Validate(limits); err == nil || !strings.Contains(err.Error(), "metadata key") {
		t.Fatalf("context.Validate() error = %v, want metadata key validation", err)
	}
}

func TestProtocolDescriptorForSourcePreservesOwnership(t *testing.T) {
	for _, source := range []ProtocolSource{ProtocolSourceRunner, ProtocolSourceWorkflow, ProtocolSourceTeams, ProtocolSourceScheduler, ProtocolSourceA2A, ProtocolSourceRealtime} {
		descriptor, err := ProtocolDescriptorForSource(source, "runtime-1", "v1", nil, []ProtocolAction{ProtocolActionCancel})
		if err != nil {
			t.Fatalf("source %q: %v", source, err)
		}
		if descriptor.Source != source || descriptor.RuntimeID != "runtime-1" || !descriptor.SupportsAction(ProtocolActionCancel) {
			t.Fatalf("descriptor=%#v", descriptor)
		}
	}
}

func TestProtocolDescriptorActionsDoNotImplyAuthorization(t *testing.T) {
	descriptor := ProtocolDescriptor{
		ProtocolName:   AgentRuntimeProtocolName,
		ProfileVersion: "v1",
		RuntimeID:      "runner",
		Actions:        []ProtocolAction{ProtocolActionCancel, ProtocolActionRetry},
	}
	if !descriptor.SupportsAction(ProtocolActionCancel) {
		t.Fatal("cancel must be reported as available")
	}
	if descriptor.SupportsAction(ProtocolActionResume) {
		t.Fatal("resume must not be inferred from lifecycle states")
	}
	if err := ValidateProtocolAction(descriptor, ProtocolActionResume); err == nil || !strings.Contains(err.Error(), ProtocolReasonActionUnsupported) {
		t.Fatalf("ValidateProtocolAction() error = %v, want %q", err, ProtocolReasonActionUnsupported)
	}
}

func TestConcurrentRunAdmissionValidatesPolicyOutcomePairs(t *testing.T) {
	queued := ProtocolRunAdmission{
		SessionID:         "session-1",
		RunID:             "run-2",
		Policy:            ProtocolConcurrentRunPolicySerialize,
		Decision:          ProtocolRunAdmissionDecisionQueued,
		ReasonCode:        "scheduler.session_busy",
		ConflictingRunIDs: []string{"run-1"},
	}
	if err := queued.Validate(); err != nil {
		t.Fatalf("queued.Validate() error = %v", err)
	}

	incompatible := queued
	incompatible.Policy = ProtocolConcurrentRunPolicyReject
	if err := incompatible.Validate(); err == nil || !strings.Contains(err.Error(), ProtocolReasonAdmissionIncompatibleOutcome) {
		t.Fatalf("incompatible.Validate() error = %v, want %q", err, ProtocolReasonAdmissionIncompatibleOutcome)
	}

	unknown := ProtocolRunAdmission{
		SessionID:  "session-1",
		RunID:      "run-2",
		Policy:     ProtocolConcurrentRunPolicyUnknown,
		Decision:   ProtocolRunAdmissionDecisionUnresolved,
		ReasonCode: ProtocolReasonAdmissionPolicyUnknown,
	}
	if err := unknown.Validate(); err != nil {
		t.Fatalf("unknown.Validate() error = %v", err)
	}
	unknown.ReasonCode = "scheduler.unknown"
	if err := unknown.Validate(); err == nil || !strings.Contains(err.Error(), ProtocolReasonAdmissionPolicyUnknown) {
		t.Fatalf("unknown policy reason should be deterministic: %v", err)
	}
}

func TestProjectProtocolRunAdmissionClonesSourceOutcome(t *testing.T) {
	source := ProtocolRunAdmission{
		SessionID:         "session-1",
		RunID:             "run-2",
		Policy:            ProtocolConcurrentRunPolicyReject,
		Decision:          ProtocolRunAdmissionDecisionRejected,
		ReasonCode:        "scheduler.session_busy",
		ConflictingRunIDs: []string{"run-1"},
	}
	projected, err := ProjectProtocolRunAdmission(source)
	if err != nil {
		t.Fatalf("ProjectProtocolRunAdmission() error = %v", err)
	}
	projected.ConflictingRunIDs[0] = "mutated"
	if source.ConflictingRunIDs[0] != "run-1" {
		t.Fatalf("projection mutated source outcome: %#v", source)
	}
}

func TestProtocolSessionContextRejectsInvalidAgentWithoutMutation(t *testing.T) {
	context := ProtocolSessionContext{
		Session: SessionRef{SessionID: "session-1", Source: ProtocolSourceRunner},
		Scope:   ProtocolContextScopeAgent,
		Agent:   &ProtocolParticipantRef{ID: "", Role: "primary", Source: ProtocolSourceRunner},
	}
	if err := context.Validate(ProtocolContextLimits{}); err == nil {
		t.Fatal("invalid agent reference should be rejected")
	}
	if context.Agent.ID != "" {
		t.Fatalf("validation must not mutate context: %#v", context)
	}
}

func TestProjectProtocolSessionContextClonesReferencesAndMetadata(t *testing.T) {
	participants := []ProtocolParticipantRef{{ID: "participant-1", Role: "observer", Source: ProtocolSourceRunner}}
	metadata := map[string]string{"requester": "host"}
	projected, err := ProjectProtocolSessionContext(SessionRef{SessionID: "session-1", Source: ProtocolSourceRunner}, ProtocolContextScopeSession, nil, participants, metadata, ProtocolContextLimits{AllowedMetadataKeys: []string{"requester"}, MaxParticipants: 2, MaxMetadataEntries: 2})
	if err != nil {
		t.Fatalf("ProjectProtocolSessionContext() error = %v", err)
	}
	projected.Participants[0].ID = "mutated"
	projected.Metadata["requester"] = "mutated"
	if participants[0].ID != "participant-1" || metadata["requester"] != "host" {
		t.Fatalf("projection mutated source context: participants=%#v metadata=%#v", participants, metadata)
	}
}
