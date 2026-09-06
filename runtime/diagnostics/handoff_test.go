package diagnostics

import (
	"encoding/json"
	"testing"
	"time"
)

func TestRunRecordRoundTripsHandoffFields(t *testing.T) {
	record := RunRecord{Time: time.Unix(1, 0).UTC(), RunID: "run-1", ContextHandoffVersion: "handoff.v1", ContextHandoffCut: "checkpoint", ContextHandoffQualityScore: 0.8, ContextHandoffFallbackReason: "handoff_quality_below_threshold", ContextHandoffRestoreReady: true}
	raw, err := json.Marshal(record)
	if err != nil {
		t.Fatalf("marshalRunRecord() error = %v", err)
	}
	var got RunRecord
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("unmarshalRunRecord() error = %v", err)
	}
	if got.ContextHandoffVersion != record.ContextHandoffVersion || got.ContextHandoffCut != record.ContextHandoffCut || got.ContextHandoffFallbackReason != record.ContextHandoffFallbackReason || !got.ContextHandoffRestoreReady {
		t.Fatalf("handoff fields = %+v", got)
	}
}
