package diagnosticsreplay

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestEvaluateProtocolFixtureSuccessAndRunStreamParity(t *testing.T) {
	raw := []byte(`{"version":"agent_runtime_protocol.v1","cases":[{"name":"success","run":{"run_id":"run-1","session_id":"session-1","state":"completed","step_ids":["step-1"],"event_ids":["event-1"],"event_sequence":[1],"artifact_ids":["artifact-1"],"checkpoint_id":"checkpoint-1"},"stream":{"run_id":"run-1","session_id":"session-1","state":"completed","step_ids":["step-1"],"event_ids":["event-1"],"event_sequence":[1],"artifact_ids":["artifact-1"],"checkpoint_id":"checkpoint-1"},"expected":{"run_id":"run-1","session_id":"session-1","state":"completed","step_ids":["step-1"],"event_ids":["event-1"],"event_sequence":[1],"artifact_ids":["artifact-1"],"checkpoint_id":"checkpoint-1"},"idempotency":{"first_logical_ingest_total":1,"replay_logical_ingest_total":1}}]}`)
	out, err := EvaluateProtocolFixtureJSON(raw)
	if err != nil || len(out.Cases) != 1 {
		t.Fatalf("out=%#v err=%v", out, err)
	}
}

func TestEvaluateProtocolFixtureDetectsMappingDrift(t *testing.T) {
	raw := []byte(`{"version":"agent_runtime_protocol.v1","cases":[{"name":"drift","run":{"run_id":"run-1","state":"completed"},"stream":{"run_id":"run-1","state":"failed"},"expected":{"run_id":"run-1","state":"completed"},"idempotency":{"first_logical_ingest_total":1,"replay_logical_ingest_total":1}}]}`)
	_, err := EvaluateProtocolFixtureJSON(raw)
	if err == nil || !strings.Contains(err.Error(), ReasonCodeProtocolDrift) {
		t.Fatalf("error=%v, want %s", err, ReasonCodeProtocolDrift)
	}
}

func TestEvaluateProtocolFixturePreservesDescriptorContextAndAdmissionParity(t *testing.T) {
	raw := []byte(`{"version":"agent_runtime_protocol.v1","cases":[{"name":"projection","run":{"run_id":"run-1","state":"completed","descriptor":{"profile_version":"v1","capability_decision":"accepted","supported_actions":["cancel"]},"context":{"scope":"session","metadata_keys":["requester"]},"admission":{"policy":"serialize","decision":"queued","reason_code":"scheduler.session_busy","conflicting_run_ids":["run-0"]}},"stream":{"run_id":"run-1","state":"completed","descriptor":{"profile_version":"v1","capability_decision":"accepted","supported_actions":["cancel"]},"context":{"scope":"session","metadata_keys":["requester"]},"admission":{"policy":"serialize","decision":"queued","reason_code":"scheduler.session_busy","conflicting_run_ids":["run-0"]}},"expected":{"run_id":"run-1","state":"completed","descriptor":{"profile_version":"v1","capability_decision":"accepted","supported_actions":["cancel"]},"context":{"scope":"session","metadata_keys":["requester"]},"admission":{"policy":"serialize","decision":"queued","reason_code":"scheduler.session_busy","conflicting_run_ids":["run-0"]}},"idempotency":{"first_logical_ingest_total":1,"replay_logical_ingest_total":1}}]}`)
	out, err := EvaluateProtocolFixtureJSON(raw)
	if err != nil || len(out.Cases) != 1 || out.Cases[0].Canonical.Descriptor == nil || out.Cases[0].Canonical.Admission == nil {
		t.Fatalf("out=%#v err=%v", out, err)
	}
}

func TestEvaluateProtocolFixtureClassifiesProfileDrift(t *testing.T) {
	raw := []byte(`{"version":"agent_runtime_protocol.v1","cases":[{"name":"profile-drift","run":{"run_id":"run-1","state":"completed","descriptor":{"profile_version":"v2"}},"stream":{"run_id":"run-1","state":"completed","descriptor":{"profile_version":"v1"}},"expected":{"run_id":"run-1","state":"completed","descriptor":{"profile_version":"v1"}},"idempotency":{"first_logical_ingest_total":1,"replay_logical_ingest_total":1}}]}`)
	_, err := EvaluateProtocolFixtureJSON(raw)
	if err == nil || !strings.Contains(err.Error(), ReasonCodeProtocolProfileDrift) {
		t.Fatalf("error=%v, want %s", err, ReasonCodeProtocolProfileDrift)
	}
}

func TestEvaluateProtocolFixturePreservesEventStreamBindingParity(t *testing.T) {
	raw := []byte(`{"version":"agent_runtime_protocol.v1","cases":[{"name":"stream-binding","run":{"run_id":"run-1","session_id":"session-1","state":"working","event_ids":["event-1"],"event_sequence":[1],"stream_binding":{"subscription_id":"sub-1","phase":"live","reason_code":"realtime.binding.live","cursor_mode":"after_cursor","sequence_boundary":1}},"stream":{"run_id":"run-1","session_id":"session-1","state":"working","event_ids":["event-1"],"event_sequence":[1],"stream_binding":{"subscription_id":"sub-1","phase":"live","reason_code":"realtime.binding.live","cursor_mode":"after_cursor","sequence_boundary":1}},"expected":{"run_id":"run-1","session_id":"session-1","state":"working","event_ids":["event-1"],"event_sequence":[1],"stream_binding":{"subscription_id":"sub-1","phase":"live","reason_code":"realtime.binding.live","cursor_mode":"after_cursor","sequence_boundary":1}},"idempotency":{"first_logical_ingest_total":1,"replay_logical_ingest_total":1}}]}`)
	out, err := EvaluateProtocolFixtureJSON(raw)
	if err != nil || len(out.Cases) != 1 || out.Cases[0].Canonical.StreamBinding == nil {
		t.Fatalf("out=%#v err=%v", out, err)
	}
}

func TestEvaluateProtocolFixtureClassifiesEventStreamBindingDrift(t *testing.T) {
	raw := []byte(`{"version":"agent_runtime_protocol.v1","cases":[{"name":"stream-binding-drift","run":{"run_id":"run-1","state":"working","stream_binding":{"subscription_id":"sub-1","phase":"expired","reason_code":"realtime.cursor.expired","cursor_mode":"after_cursor"}},"stream":{"run_id":"run-1","state":"working","stream_binding":{"subscription_id":"sub-1","phase":"live","reason_code":"realtime.binding.live","cursor_mode":"after_cursor"}},"expected":{"run_id":"run-1","state":"working","stream_binding":{"subscription_id":"sub-1","phase":"live","reason_code":"realtime.binding.live","cursor_mode":"after_cursor"}},"idempotency":{"first_logical_ingest_total":1,"replay_logical_ingest_total":1}}]}`)
	_, err := EvaluateProtocolFixtureJSON(raw)
	if err == nil || !strings.Contains(err.Error(), ReasonCodeProtocolStreamBindingDrift) {
		t.Fatalf("error=%v, want %s", err, ReasonCodeProtocolStreamBindingDrift)
	}
}

func TestEvaluateCheckpointProvenanceFixture(t *testing.T) {
	raw := []byte(`{"version":"agent_runtime_protocol_checkpoint_provenance.v1","cases":[{"name":"provenance","run":{"run_id":"run-1","session_id":"session-1","state":"completed","checkpoint_id":"checkpoint-1","checkpoint_provenance":{"relation":"derived","parent_checkpoint_id":"checkpoint-root","history_index":1,"restore_source":"resume","replay_key":"replay-1","workspace_id":"workspace-1","change_set_id":"change-1","before_integrity":"before","after_integrity":"after"}},"stream":{"run_id":"run-1","session_id":"session-1","state":"completed","checkpoint_id":"checkpoint-1","checkpoint_provenance":{"relation":"derived","parent_checkpoint_id":"checkpoint-root","history_index":1,"restore_source":"resume","replay_key":"replay-1","workspace_id":"workspace-1","change_set_id":"change-1","before_integrity":"before","after_integrity":"after"}},"expected":{"run_id":"run-1","session_id":"session-1","state":"completed","checkpoint_id":"checkpoint-1","checkpoint_provenance":{"relation":"derived","parent_checkpoint_id":"checkpoint-root","history_index":1,"restore_source":"resume","replay_key":"replay-1","workspace_id":"workspace-1","change_set_id":"change-1","before_integrity":"before","after_integrity":"after"}},"idempotency":{"first_logical_ingest_total":1,"replay_logical_ingest_total":1}}]}`)
	out, err := EvaluateProtocolFixtureJSON(raw)
	if err != nil || out.Version != "agent_runtime_protocol_checkpoint_provenance.v1" || len(out.Cases) != 1 {
		t.Fatalf("out=%#v err=%v", out, err)
	}
}

func TestEvaluateCheckpointProvenanceFixtureDetectsWorkspaceDrift(t *testing.T) {
	raw := []byte(`{"version":"agent_runtime_protocol_checkpoint_provenance.v1","cases":[{"name":"drift","run":{"run_id":"run-1","state":"completed","checkpoint_provenance":{"relation":"root","restore_source":"fresh","workspace_id":"workspace-1","change_set_id":"change-1","before_integrity":"observed","after_integrity":"after"}},"stream":{"run_id":"run-1","state":"completed","checkpoint_provenance":{"relation":"root","restore_source":"fresh","workspace_id":"workspace-1","change_set_id":"change-1","before_integrity":"observed","after_integrity":"after"}},"expected":{"run_id":"run-1","state":"completed","checkpoint_provenance":{"relation":"root","restore_source":"fresh","workspace_id":"workspace-1","change_set_id":"change-1","before_integrity":"before","after_integrity":"after"}},"idempotency":{"first_logical_ingest_total":1,"replay_logical_ingest_total":1}}]}`)
	_, err := EvaluateProtocolFixtureJSON(raw)
	if err == nil || !strings.Contains(err.Error(), ReasonCodeWorkspaceIntegrityDrift) {
		t.Fatalf("error=%v, want %s", err, ReasonCodeWorkspaceIntegrityDrift)
	}
}

func TestEvaluateCheckpointProvenanceFixtureClassifiesFieldDrift(t *testing.T) {
	cases := []struct {
		name   string
		want   string
		mutate func(*ProtocolCheckpointProvenanceObservation)
	}{
		{name: "schema", want: ReasonCodeCheckpointSchemaDrift, mutate: func(p *ProtocolCheckpointProvenanceObservation) { p.SchemaVersion = "v2" }},
		{name: "lineage", want: ReasonCodeCheckpointLineageDrift, mutate: func(p *ProtocolCheckpointProvenanceObservation) { p.ParentCheckpointID = "other" }},
		{name: "branch", want: ReasonCodeCheckpointBranchDrift, mutate: func(p *ProtocolCheckpointProvenanceObservation) { p.BranchID = "branch-2" }},
		{name: "replay", want: ReasonCodeCheckpointReplayDrift, mutate: func(p *ProtocolCheckpointProvenanceObservation) { p.ReplayKey = "replay-2" }},
		{name: "workspace", want: ReasonCodeWorkspaceProvenanceDrift, mutate: func(p *ProtocolCheckpointProvenanceObservation) { p.ChangeSetID = "change-2" }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			base := &ProtocolCheckpointProvenanceObservation{SchemaVersion: "v1", Relation: "derived", ParentCheckpointID: "root", BranchID: "branch-1", HistoryIndex: 1, RestoreSource: "resume", ReplayKey: "replay-1", WorkspaceID: "workspace-1", ChangeSetID: "change-1", BeforeIntegrity: "before", AfterIntegrity: "after"}
			actual := *base
			tc.mutate(&actual)
			raw, err := json.Marshal(ProtocolFixture{Version: AgentRuntimeProtocolCheckpointProvenanceFixtureV1, Cases: []ProtocolFixtureCase{{Name: tc.name, Run: ProtocolObservation{RunID: "run-1", State: "completed", CheckpointProvenance: base}, Stream: ProtocolObservation{RunID: "run-1", State: "completed", CheckpointProvenance: base}, Expected: ProtocolObservation{RunID: "run-1", State: "completed", CheckpointProvenance: &actual}, Idempotency: ProtocolIdempotency{FirstLogicalIngestTotal: 1, ReplayLogicalIngestTotal: 1}}}})
			if err != nil {
				t.Fatalf("marshal fixture: %v", err)
			}
			_, err = EvaluateProtocolFixtureJSON(raw)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error=%v, want %s", err, tc.want)
			}
		})
	}
}

func TestEvaluateCheckpointProvenanceFixtureDetectsRunStreamParityDrift(t *testing.T) {
	base := &ProtocolCheckpointProvenanceObservation{Relation: "root", RestoreSource: "fresh", WorkspaceID: "workspace-1", ChangeSetID: "change-1"}
	stream := *base
	stream.RestoreSource = "resume"
	raw, err := json.Marshal(ProtocolFixture{Version: AgentRuntimeProtocolCheckpointProvenanceFixtureV1, Cases: []ProtocolFixtureCase{{Name: "parity", Run: ProtocolObservation{RunID: "run-1", State: "completed", CheckpointProvenance: base}, Stream: ProtocolObservation{RunID: "run-1", State: "completed", CheckpointProvenance: &stream}, Expected: ProtocolObservation{RunID: "run-1", State: "completed", CheckpointProvenance: base}, Idempotency: ProtocolIdempotency{FirstLogicalIngestTotal: 1, ReplayLogicalIngestTotal: 1}}}})
	if err != nil {
		t.Fatalf("marshal fixture: %v", err)
	}
	_, err = EvaluateProtocolFixtureJSON(raw)
	if err == nil || !strings.Contains(err.Error(), ReasonCodeRunStreamProvenanceParityDrift) {
		t.Fatalf("error=%v, want %s", err, ReasonCodeRunStreamProvenanceParityDrift)
	}
}

func TestParseCheckpointProvenanceFixtureRejectsMalformedProvenance(t *testing.T) {
	raw := []byte(`{"version":"agent_runtime_protocol_checkpoint_provenance.v1","cases":[{"name":"bad","run":{"run_id":"run-1","state":"completed","checkpoint_provenance":{"relation":"root"}},"stream":{"run_id":"run-1","state":"completed","checkpoint_provenance":{"relation":"root"}},"expected":{"run_id":"run-1","state":"completed","checkpoint_provenance":{"relation":"root"}},"idempotency":{"first_logical_ingest_total":1,"replay_logical_ingest_total":1}}]}`)
	_, err := EvaluateProtocolFixtureJSON(raw)
	if err == nil || !strings.Contains(err.Error(), ReasonCodeProtocolSchema) {
		t.Fatalf("error=%v, want %s", err, ReasonCodeProtocolSchema)
	}
}
