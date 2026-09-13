package event

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/FelixSeptem/baymax/core/types"
	runtimeconfig "github.com/FelixSeptem/baymax/runtime/config"
)

func TestRuntimeRecorderCompletionReferenceDiagnosticsAreBounded(t *testing.T) {
	mgr, err := runtimeconfig.NewManager(runtimeconfig.ManagerOptions{EnvPrefix: "BAYMAX_COMPLETION_REFERENCE_DIAGNOSTICS"})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = mgr.Close() }()
	recorder := NewRuntimeRecorder(mgr)
	recorder.OnEvent(context.Background(), types.Event{Version: types.EventSchemaVersionV1, Type: types.EventTypeRuntimeInputAdmission, RunID: "run-completion", Time: time.Now().UTC(), Payload: map[string]any{
		"input_id": "idem-1", "input_kind": "follow_up", "admission_status": "accepted", "reason_code": "runtime_input.accepted", "source_correlation": "corr-1", "payload": "secret completion body",
	}})
	items := mgr.RecentRuns(1)
	if len(items) != 1 {
		t.Fatalf("runs=%#v", items)
	}
	raw := items[0].ProtocolState + "|" + items[0].ProtocolAdmissionReason + "|" + items[0].ProtocolSource
	if strings.Contains(raw, "secret") || len(items[0].ProtocolAdmissionReason) > 96 || len(items[0].ProtocolSource) > 48 {
		t.Fatalf("unbounded completion diagnostics=%#v", items[0])
	}
}
