package event

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/FelixSeptem/baymax/core/types"
	runtimeconfig "github.com/FelixSeptem/baymax/runtime/config"
)

func TestRuntimeRecorderParsesDynamicActionFieldsAdditiveAndBounded(t *testing.T) {
	mgr, err := runtimeconfig.NewManager(runtimeconfig.ManagerOptions{EnvPrefix: "BAYMAX_DYNAMIC_ACTION_RECORDER"})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = mgr.Close() }()
	rec := NewRuntimeRecorder(mgr)
	rec.OnEvent(context.Background(), types.Event{Version: types.EventSchemaVersionV1, Type: "run.finished", RunID: "run-dynamic-recorder", Time: time.Now().UTC(), Payload: map[string]any{
		"status": "success", "gate_checks": 1, "dynamic_action_count": 1,
		"dynamic_action_reference_digest": strings.Repeat("d", 512), "dynamic_action_checkpoint_id": "checkpoint-1",
		"dynamic_action_checkpoint_version": types.DynamicActionResumeProtocolVersion, "dynamic_action_checkpoint_digest": "digest-1",
		"dynamic_action_pause_reason": "dynamic_action.input_required_same_run", "dynamic_action_resume_attempt": 1, "dynamic_action_resume_admission": "accepted",
	}})
	items := mgr.RecentRuns(1)
	if len(items) != 1 {
		t.Fatalf("runs=%#v", items)
	}
	got := items[0]
	if got.GateChecks != 1 || got.DynamicActionCount != 1 || got.DynamicActionCheckpointID != "checkpoint-1" || got.DynamicActionResumeAdmission != "accepted" {
		t.Fatalf("dynamic fields not recorded: %#v", got)
	}
	if len(got.DynamicActionReferenceDigest) > 128 {
		t.Fatalf("unbounded digest persisted: %d", len(got.DynamicActionReferenceDigest))
	}
	// Legacy run.finished payloads remain valid and default the additive fields.
	rec.OnEvent(context.Background(), types.Event{Version: types.EventSchemaVersionV1, Type: "run.finished", RunID: "run-legacy-recorder", Time: time.Now().UTC(), Payload: map[string]any{"status": "success"}})
	legacy := mgr.RecentRuns(1)
	if len(legacy) != 1 || legacy[0].DynamicActionCount != 0 || legacy[0].DynamicActionResumeAttempt != 0 {
		t.Fatalf("legacy defaults changed: %#v", legacy)
	}
}
