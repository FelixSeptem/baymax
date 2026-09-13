package event

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/FelixSeptem/baymax/core/types"
	runtimeconfig "github.com/FelixSeptem/baymax/runtime/config"
)

func TestRuntimeRecorderProviderHandoffDiagnosticsRemainBoundedAndRedacted(t *testing.T) {
	mgr, err := runtimeconfig.NewManager(runtimeconfig.ManagerOptions{})
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}
	defer func() { _ = mgr.Close() }()
	recorder := NewRuntimeRecorder(mgr)
	rawPayload := strings.Repeat("provider-secret-payload-", 1000)
	recorder.OnEvent(context.Background(), types.Event{Type: "run.finished", RunID: "run-provider-edge", Payload: map[string]any{
		"status": "success", "model_provider": "openai", "provider_handoff_reason": "provider_stream_boundary_drift",
		"provider_handoff_stream_edge": rawPayload, "raw_provider_payload": rawPayload,
		"reasoning_body": strings.Repeat("private reasoning ", 1000), "credential": "sk-secret",
	}})
	records := mgr.RecentRuns(1)
	if len(records) != 1 {
		t.Fatalf("RecentRuns() len = %d, want 1", len(records))
	}
	b, err := json.Marshal(records[0])
	if err != nil {
		t.Fatal(err)
	}
	serialized := string(b)
	for _, secret := range []string{"provider-secret-payload", "private reasoning", "sk-secret", "raw_provider_payload", "provider_handoff_stream_edge"} {
		if strings.Contains(serialized, secret) {
			t.Fatalf("diagnostics leaked %q: %s", secret, serialized)
		}
	}
	if records[0].ModelProvider != "openai" {
		t.Fatalf("bounded provider field missing: %#v", records[0])
	}
}
