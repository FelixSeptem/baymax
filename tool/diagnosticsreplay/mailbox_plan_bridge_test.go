package diagnosticsreplay

import (
	"errors"
	"os"
	"testing"
)

func TestMailboxPlanBridgeReplayGoldenAndDrift(t *testing.T) {
	path := os.Getenv("BAYMAX_MAILBOX_PLAN_BRIDGE_FIXTURE")
	if path == "" {
		path = "testdata/mailbox_plan_bridge.v1.json"
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	got, err := ReplayMailboxPlanBridgeFixtureJSON(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 4 || got[0].Reason != "" || got[1].Reason != "" || got[2].Reason != ReasonMailboxPlanManifestDrift || got[3].Reason != ReasonMailboxPlanDuplicateConflict {
		t.Fatalf("unexpected replay: %#v", got)
	}
	if _, err = ReplayMailboxPlanBridgeFixtureJSON([]byte(`{"version":"mailbox_plan_bridge.v1","cases":[{"case_id":"x","manifest_contract_version":"baymax.runtime.contract.v1","manifest_api_version":"host_runtime_capability_manifest.v1","evidence_reference":"fixture","source_available":true,"binding_equivalent":true,"logical_admissions":1,"observation":{},"run":{},"stream":{},"expected_reason":""}]}`)); err == nil {
		t.Fatal("unbounded reference-only fixture accepted")
	}
	var validation *ValidationError
	if !errors.As(err, &validation) || validation.Code != ReasonMailboxPlanReplayDrift {
		t.Fatalf("invalid replay error = %v", err)
	}
}

func TestMailboxPlanBridgeReplayIsReferenceOnly(t *testing.T) {
	raw, err := os.ReadFile("testdata/mailbox_plan_bridge.v1.json")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = ReplayMailboxPlanBridgeFixtureJSON(raw); err != nil {
		t.Fatal(err)
	}
}
