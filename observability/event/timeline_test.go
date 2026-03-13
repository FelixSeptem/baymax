package event

import (
	"testing"
	"time"

	"github.com/FelixSeptem/baymax/core/types"
)

func TestParseActionTimelineSuccess(t *testing.T) {
	ev := types.Event{
		Type:      types.EventTypeActionTimeline,
		RunID:     "run-1",
		Iteration: 2,
		Time:      time.Now(),
		Payload: map[string]any{
			"phase":    "model",
			"status":   "running",
			"sequence": int64(3),
			"reason":   "retrying",
		},
	}
	out, ok := ParseActionTimeline(ev)
	if !ok {
		t.Fatal("ParseActionTimeline should parse a valid timeline event")
	}
	if out.Phase != types.ActionPhaseModel || out.Status != types.ActionStatusRunning {
		t.Fatalf("unexpected phase/status: %#v", out)
	}
	if out.Sequence != 3 || out.Reason != "retrying" {
		t.Fatalf("unexpected sequence/reason: %#v", out)
	}
}

func TestParseActionTimelineRejectsInvalidPayload(t *testing.T) {
	ev := types.Event{
		Type:  types.EventTypeActionTimeline,
		RunID: "run-1",
		Payload: map[string]any{
			"phase":    "unknown",
			"status":   "running",
			"sequence": int64(1),
		},
	}
	if _, ok := ParseActionTimeline(ev); ok {
		t.Fatal("ParseActionTimeline should reject unknown phase")
	}

	ev.Payload = map[string]any{
		"phase":    "run",
		"status":   "unknown",
		"sequence": int64(1),
	}
	if _, ok := ParseActionTimeline(ev); ok {
		t.Fatal("ParseActionTimeline should reject unknown status")
	}

	ev.Payload = map[string]any{
		"phase":    "run",
		"status":   "pending",
		"sequence": int64(0),
	}
	if _, ok := ParseActionTimeline(ev); ok {
		t.Fatal("ParseActionTimeline should reject non-positive sequence")
	}
}
