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
