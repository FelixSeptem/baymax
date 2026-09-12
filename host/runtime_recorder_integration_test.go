package host

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/FelixSeptem/baymax/core/types"
	eventrecorder "github.com/FelixSeptem/baymax/observability/event"
	runtimeconfig "github.com/FelixSeptem/baymax/runtime/config"
)

func TestRuntimeRecorderSinkPersistsOnlyBoundedHostFacts(t *testing.T) {
	mgr, err := runtimeconfig.NewManager(runtimeconfig.ManagerOptions{EnvPrefix: "BAYMAX_HOST_FACTS_INTEGRATION"})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = mgr.Close() }()
	recorder := eventrecorder.NewRuntimeRecorder(mgr)
	coord := New(nil, WithEventSink(recorder), WithSourceControl(&contractControl{}))
	conn := coord.Connect(func(any) error { return nil })
	cmd := validHostCommand(types.HostCommandKindAction)
	cmd.Payload = map[string]any{
		"action":      "cancel",
		"cursor":      strings.Repeat("opaque-cursor", 256),
		"raw_payload": strings.Repeat("sensitive-body", 256),
	}
	if _, err := conn.HandleCommand(context.Background(), cmd); err != nil {
		t.Fatal(err)
	}
	// RuntimeRecorder records synchronously, but leave a tiny scheduling margin
	// for future sink implementations without weakening the assertions.
	deadline := time.Now().Add(time.Second)
	var runs = mgr.RecentRuns(4)
	for time.Now().Before(deadline) {
		runs = mgr.RecentRuns(4)
		if len(runs) > 0 {
			break
		}
		time.Sleep(time.Millisecond)
	}
	if len(runs) != 1 {
		t.Fatalf("runtime runs = %#v", runs)
	}
	record := runs[0]
	if record.ProtocolState != "admission" || record.ProtocolAdmissionDecision != string(types.HostAdmissionStatusAccepted) {
		t.Fatalf("host fact projection = %#v", record)
	}
	if strings.Contains(record.ProtocolAdmissionReason, "opaque-cursor") || strings.Contains(record.ProtocolAdmissionReason, "sensitive-body") {
		t.Fatalf("raw payload leaked into recorder: %#v", record)
	}
}

func TestRuntimeRecorderSinkPersistsRecordedProjectionDrop(t *testing.T) {
	mgr, err := runtimeconfig.NewManager(runtimeconfig.ManagerOptions{EnvPrefix: "BAYMAX_HOST_DROP_FACTS_INTEGRATION"})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = mgr.Close() }()
	recorder := eventrecorder.NewRuntimeRecorder(mgr)
	writer := &blockingFrameWriter{started: make(chan struct{}), exited: make(chan struct{})}
	conn := New(nil,
		WithEventSink(recorder),
		WithDeliveryQueueSize(1),
		WithDeliveryTimeout(250*time.Millisecond),
	).ConnectFrameWriter(writer)
	done := make(chan struct{})
	go func() {
		conn.emitProjection(validHostCommand(types.HostCommandKindEventsSubscribe), recordedDropProjection(
			types.RealtimeEventTypeDelta,
			types.RealtimeEventTypeDelta,
			types.RealtimeEventTypeDelta,
		))
		close(done)
	}()
	select {
	case <-writer.started:
	case <-time.After(time.Second):
		t.Fatal("projection writer did not start")
	}
	select {
	case <-done:
	case <-time.After(100 * time.Millisecond):
		t.Fatal("recorded projection drop blocked on transport")
	}
	conn.Close()
	select {
	case <-writer.exited:
	case <-time.After(time.Second):
		t.Fatal("blocked writer did not exit after connection close")
	}

	var found bool
	for _, record := range mgr.RecentRuns(8) {
		if record.RunID == "run-1" && record.ProtocolState == "dropped" && record.ProtocolAdmissionReason == HostReasonDeliverySourcePolicyDrop {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("RuntimeRecorder did not persist bounded drop fact: %#v", mgr.RecentRuns(8))
	}
}
