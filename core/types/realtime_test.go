package types

import (
	"strings"
	"testing"
	"time"
)

func TestParseRealtimeEventEnvelope(t *testing.T) {
	raw := []byte(`{
		"event_id":"evt-1",
		"session_id":"sess-1",
		"run_id":"run-1",
		"seq":1,
		"type":"request",
		"ts":"2026-04-03T12:00:00Z",
		"payload":{"input":"hello"}
	}`)
	ev, err := ParseRealtimeEventEnvelope(raw)
	if err != nil {
		t.Fatalf("ParseRealtimeEventEnvelope failed: %v", err)
	}
	if ev.EventID != "evt-1" || ev.Seq != 1 || ev.Type != RealtimeEventTypeRequest {
		t.Fatalf("unexpected envelope parsed: %#v", ev)
	}
}

func TestParseRealtimeEventEnvelopeRejectsSchemaViolations(t *testing.T) {
	t.Run("missing_event_id", func(t *testing.T) {
		raw := []byte(`{
			"session_id":"sess-1",
			"run_id":"run-1",
			"seq":1,
			"type":"request",
			"ts":"2026-04-03T12:00:00Z",
			"payload":{}
		}`)
		_, err := ParseRealtimeEventEnvelope(raw)
		if err == nil || !strings.Contains(err.Error(), "event_id") {
			t.Fatalf("expected event_id validation error, got %v", err)
		}
	})

	t.Run("wrong_seq_type", func(t *testing.T) {
		raw := []byte(`{
			"event_id":"evt-1",
			"session_id":"sess-1",
			"run_id":"run-1",
			"seq":"bad",
			"type":"request",
			"ts":"2026-04-03T12:00:00Z",
			"payload":{}
		}`)
		_, err := ParseRealtimeEventEnvelope(raw)
		if err == nil || !strings.Contains(err.Error(), "unmarshal realtime event envelope") {
			t.Fatalf("expected unmarshal validation error, got %v", err)
		}
	})

	t.Run("unsupported_type", func(t *testing.T) {
		ev := RealtimeEventEnvelope{
			EventID:   "evt-1",
			SessionID: "sess-1",
			RunID:     "run-1",
			Seq:       1,
			Type:      RealtimeEventType("unknown"),
			TS:        time.Now(),
			Payload:   map[string]any{},
		}
		if err := ev.Validate(); err == nil || !strings.Contains(err.Error(), "unsupported type") {
			t.Fatalf("expected unsupported type validation error, got %v", err)
		}
	})

	t.Run("illegal_seq", func(t *testing.T) {
		ev := RealtimeEventEnvelope{
			EventID:   "evt-1",
			SessionID: "sess-1",
			RunID:     "run-1",
			Seq:       0,
			Type:      RealtimeEventTypeRequest,
			TS:        time.Now(),
			Payload:   map[string]any{},
		}
		if err := ev.Validate(); err == nil || !strings.Contains(err.Error(), "seq must be > 0") {
			t.Fatalf("expected seq validation error, got %v", err)
		}
	})
}

func TestRealtimeEventEnvelopeDedupKey(t *testing.T) {
	base := RealtimeEventEnvelope{
		EventID:   "evt-1",
		SessionID: "sess-1",
		RunID:     "run-1",
		Seq:       1,
		TS:        time.Now(),
		Payload:   map[string]any{},
	}

	ev := base
	ev.Type = RealtimeEventTypeInterrupt
	if got := ev.DedupKey(); got != "interrupt:sess-1:run-1" {
		t.Fatalf("interrupt dedup key = %q, want %q", got, "interrupt:sess-1:run-1")
	}

	ev = base
	ev.Type = RealtimeEventTypeResume
	ev.Payload = map[string]any{"cursor": "cursor-1"}
	if got := ev.DedupKey(); got != "resume:sess-1:cursor-1" {
		t.Fatalf("resume dedup key = %q, want %q", got, "resume:sess-1:cursor-1")
	}

	ev = base
	ev.Type = RealtimeEventTypeDelta
	ev.Payload = map[string]any{"dedup_key": "custom-key"}
	if got := ev.DedupKey(); got != "custom-key" {
		t.Fatalf("custom dedup key = %q, want custom-key", got)
	}
}

func TestRealtimeEventEnvelopeCanonicalPayload(t *testing.T) {
	ts := time.Date(2026, 4, 3, 12, 30, 0, 0, time.UTC)
	ev := RealtimeEventEnvelope{
		EventID:   "evt-1",
		SessionID: "sess-1",
		RunID:     "run-1",
		Seq:       8,
		Type:      RealtimeEventTypeDelta,
		TS:        ts,
		Payload: map[string]any{
			"delta": "hello",
		},
	}
	payload := ev.CanonicalPayload()
	if payload["event_id"] != "evt-1" ||
		payload["session_id"] != "sess-1" ||
		payload["run_id"] != "run-1" ||
		payload["seq"] != int64(8) ||
		payload["type"] != "delta" {
		t.Fatalf("unexpected canonical payload: %#v", payload)
	}
	if payload["ts"] != "2026-04-03T12:30:00Z" {
		t.Fatalf("unexpected ts payload: %#v", payload["ts"])
	}
	inner, ok := payload["payload"].(map[string]any)
	if !ok || inner["delta"] != "hello" {
		t.Fatalf("unexpected nested payload: %#v", payload["payload"])
	}
}
