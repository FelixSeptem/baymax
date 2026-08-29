package event

import (
	"context"
	"testing"
	"time"

	"github.com/FelixSeptem/baymax/core/types"
	runtimeconfig "github.com/FelixSeptem/baymax/runtime/config"
)

func TestRuntimeRecorderParsesBoundedEventStreamBindingFields(t *testing.T) {
	mgr, err := runtimeconfig.NewManager(runtimeconfig.ManagerOptions{EnvPrefix: "BAYMAX"})
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}
	defer func() { _ = mgr.Close() }()
	rec := NewRuntimeRecorder(mgr)
	rec.OnEvent(context.Background(), types.Event{
		Version: types.EventSchemaVersionV1,
		Type:    "run.finished",
		RunID:   "run-binding-recorder",
		Time:    time.Now(),
		Payload: map[string]any{
			"status":                           "success",
			"stream_subscription_id":           "sub-1",
			"stream_binding_phase":             "live",
			"stream_binding_decision":          "accepted",
			"stream_binding_reason":            "realtime.binding.live",
			"stream_binding_cursor_mode":       "after_cursor",
			"stream_binding_sequence_boundary": int64(42),
			"stream_cursor_body":               "must-not-be-promoted",
		},
	})
	items := mgr.RecentRuns(1)
	if len(items) != 1 {
		t.Fatalf("run records len = %d, want 1", len(items))
	}
	got := items[0]
	if got.StreamSubscriptionID != "sub-1" || got.StreamBindingPhase != "live" || got.StreamBindingDecision != "accepted" || got.StreamBindingReason != "realtime.binding.live" || got.StreamBindingCursorMode != "after_cursor" || got.StreamBindingSequenceBoundary != 42 {
		t.Fatalf("binding fields mismatch: %#v", got)
	}
}
