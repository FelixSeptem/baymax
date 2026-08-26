package diagnosticsreplay

import (
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
