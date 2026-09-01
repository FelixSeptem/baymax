package diagnostics

import (
	"encoding/json"
	"testing"
	"time"
)

func TestRunRecordRoundTripsNullableEventStreamBindingFields(t *testing.T) {
	record := RunRecord{
		Time:                          time.Unix(10, 0).UTC(),
		RunID:                         "run-binding",
		StreamSubscriptionID:          "sub-1",
		StreamBindingPhase:            "live",
		StreamBindingDecision:         "accepted",
		StreamBindingReason:           "realtime.binding.live",
		StreamBindingCursorMode:       "after_cursor",
		StreamBindingSequenceBoundary: 42,
	}
	encoded, err := json.Marshal(record)
	if err != nil {
		t.Fatalf("marshalRunRecord() error = %v", err)
	}
	var decoded RunRecord
	err = json.Unmarshal(encoded, &decoded)
	if err != nil {
		t.Fatalf("unmarshalRunRecord() error = %v", err)
	}
	if decoded.StreamSubscriptionID != record.StreamSubscriptionID ||
		decoded.StreamBindingPhase != record.StreamBindingPhase ||
		decoded.StreamBindingDecision != record.StreamBindingDecision ||
		decoded.StreamBindingReason != record.StreamBindingReason ||
		decoded.StreamBindingCursorMode != record.StreamBindingCursorMode ||
		decoded.StreamBindingSequenceBoundary != record.StreamBindingSequenceBoundary {
		t.Fatalf("binding fields mismatch: %#v", decoded)
	}
}

func TestRunRecordRoundTripsNullableEventStreamTerminalRecoveryFields(t *testing.T) {
	record := RunRecord{Time: time.Unix(10, 0).UTC(), RunID: "run-recovery", StreamRecoveryPhase: "terminal_available", StreamRecoveryReason: "realtime.binding.live", StreamRecoveryTerminalState: "completed", StreamRecoveryTerminalFailureFamily: "none", StreamRecoveryRetainedEventTotal: 2, StreamRecoveryRetainedToolCallTotal: 1, StreamRecoveryTerminalConflictRecorded: true}
	encoded, err := json.Marshal(record)
	if err != nil {
		t.Fatalf("marshal RunRecord() error = %v", err)
	}
	var decoded RunRecord
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("unmarshal RunRecord() error = %v", err)
	}
	if decoded.StreamRecoveryPhase != record.StreamRecoveryPhase || decoded.StreamRecoveryRetainedEventTotal != 2 || !decoded.StreamRecoveryTerminalConflictRecorded {
		t.Fatalf("recovery fields mismatch: %#v", decoded)
	}
}
