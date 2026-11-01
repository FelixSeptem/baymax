package diagnosticsreplay

import (
	"testing"
)

func TestReplayDeploymentScopeFixtureIsReadOnlyAndDistinguishesLocalFromDistributed(t *testing.T) {
	raw := []byte(`{"version":"deployment_scope_replay.v1","cases":[{"case_id":"local","advertised_profile":"same_host_single_writer_sequential_reopen","requested_profile":"same_host_single_writer_sequential_reopen","expected_outcome":"admitted","side_effects":0},{"case_id":"distributed","advertised_profile":"same_host_single_writer_sequential_reopen","requested_profile":"multi_instance","expected_outcome":"unsupported_capability","expected_reason":"unsupported_deployment_scope","side_effects":0}]}`)
	first, err := ReplayDeploymentScopeFixtureJSON(raw)
	if err != nil {
		t.Fatal(err)
	}
	second, err := ReplayDeploymentScopeFixtureJSON(raw)
	if err != nil || len(first) != len(second) || first[1] != second[1] {
		t.Fatalf("replay is not deterministic: %#v %#v %v", first, second, err)
	}
}

func TestReplayDeploymentScopeFixtureUsesRecordedCoordinatorEvidence(t *testing.T) {
	raw := []byte(`{"version":"deployment_scope_replay.v1","cases":[{"case_id":"coordinated","advertised_profile":"multi_instance","requested_profile":"multi_instance","expected_outcome":"admitted","side_effects":0},{"case_id":"cross-machine","advertised_profile":"multi_instance","requested_profile":"cross_machine","expected_outcome":"unsupported_capability","expected_reason":"unsupported_deployment_scope","side_effects":0},{"case_id":"unavailable","advertised_profile":"source_unavailable","requested_profile":"multi_instance","expected_outcome":"source_unavailable","expected_reason":"coordinator_unavailable","side_effects":0}]}`)
	results, err := ReplayDeploymentScopeFixtureJSON(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 3 {
		t.Fatalf("result count = %d", len(results))
	}
	if results[0].Outcome != "admitted" || results[1].Outcome != "unsupported_capability" || results[2].Outcome != "source_unavailable" {
		t.Fatalf("outcomes = %#v", results)
	}
}
