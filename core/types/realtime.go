package types

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

const RealtimeEventProtocolVersionV1 = "realtime_event_protocol.v1"

type RealtimeEventType string

const (
	RealtimeEventTypeRequest   RealtimeEventType = "request"
	RealtimeEventTypeDelta     RealtimeEventType = "delta"
	RealtimeEventTypeInterrupt RealtimeEventType = "interrupt"
	RealtimeEventTypeResume    RealtimeEventType = "resume"
	RealtimeEventTypeAck       RealtimeEventType = "ack"
	RealtimeEventTypeError     RealtimeEventType = "error"
	RealtimeEventTypeComplete  RealtimeEventType = "complete"
)

type RealtimeEventEnvelope struct {
	EventID   string            `json:"event_id"`
	SessionID string            `json:"session_id"`
	RunID     string            `json:"run_id"`
	Seq       int64             `json:"seq"`
	Type      RealtimeEventType `json:"type"`
	TS        time.Time         `json:"ts"`
	Payload   map[string]any    `json:"payload"`
}

type RealtimeRunRequest struct {
	SessionID string                  `json:"session_id,omitempty"`
	Events    []RealtimeEventEnvelope `json:"events,omitempty"`
}

func ParseRealtimeEventEnvelope(raw []byte) (RealtimeEventEnvelope, error) {
	var out RealtimeEventEnvelope
	if err := json.Unmarshal(raw, &out); err != nil {
		return RealtimeEventEnvelope{}, fmt.Errorf("unmarshal realtime event envelope: %w", err)
	}
	if err := ValidateRealtimeEventEnvelope(out); err != nil {
		return RealtimeEventEnvelope{}, err
	}
	return out, nil
}

func (e RealtimeEventEnvelope) Validate() error {
	return ValidateRealtimeEventEnvelope(e)
}

func ValidateRealtimeEventEnvelope(e RealtimeEventEnvelope) error {
	if strings.TrimSpace(e.EventID) == "" {
		return fmt.Errorf("realtime event envelope: event_id is required")
	}
	if strings.TrimSpace(e.SessionID) == "" {
		return fmt.Errorf("realtime event envelope: session_id is required")
	}
	if strings.TrimSpace(e.RunID) == "" {
		return fmt.Errorf("realtime event envelope: run_id is required")
	}
	if e.Seq <= 0 {
		return fmt.Errorf("realtime event envelope: seq must be > 0")
	}
	switch normalizeRealtimeEventType(e.Type) {
	case RealtimeEventTypeRequest,
		RealtimeEventTypeDelta,
		RealtimeEventTypeInterrupt,
		RealtimeEventTypeResume,
		RealtimeEventTypeAck,
		RealtimeEventTypeError,
		RealtimeEventTypeComplete:
	default:
		return fmt.Errorf("realtime event envelope: unsupported type %q", strings.TrimSpace(string(e.Type)))
	}
	if e.TS.IsZero() {
		return fmt.Errorf("realtime event envelope: ts is required")
	}
	if e.Payload == nil {
		return fmt.Errorf("realtime event envelope: payload is required")
	}
	return nil
}

func (e RealtimeEventEnvelope) CanonicalPayload() map[string]any {
	payload := map[string]any{
		"event_id":   strings.TrimSpace(e.EventID),
		"session_id": strings.TrimSpace(e.SessionID),
		"run_id":     strings.TrimSpace(e.RunID),
		"seq":        e.Seq,
		"type":       string(normalizeRealtimeEventType(e.Type)),
		"ts":         e.TS.UTC().Format(time.RFC3339Nano),
		"payload":    cloneAnyPayloadMap(e.Payload),
	}
	return payload
}

func (e RealtimeEventEnvelope) DedupKey() string {
	if key := strings.TrimSpace(payloadString(e.Payload, "dedup_key")); key != "" {
		return key
	}
	if normalizeRealtimeEventType(e.Type) == RealtimeEventTypeInterrupt {
		return "interrupt:" + strings.TrimSpace(e.SessionID) + ":" + strings.TrimSpace(e.RunID)
	}
	if normalizeRealtimeEventType(e.Type) == RealtimeEventTypeResume {
		cursor := strings.TrimSpace(e.ResumeCursor())
		if cursor != "" {
			return "resume:" + strings.TrimSpace(e.SessionID) + ":" + cursor
		}
	}
	return strings.TrimSpace(e.EventID)
}

func (e RealtimeEventEnvelope) ResumeCursor() string {
	cursor := strings.TrimSpace(payloadString(e.Payload, "resume_cursor"))
	if cursor != "" {
		return cursor
	}
	return strings.TrimSpace(payloadString(e.Payload, "cursor"))
}

func normalizeRealtimeEventType(in RealtimeEventType) RealtimeEventType {
	return RealtimeEventType(strings.ToLower(strings.TrimSpace(string(in))))
}

func cloneAnyPayloadMap(in map[string]any) map[string]any {
	if len(in) == 0 {
		return map[string]any{}
	}
	out := make(map[string]any, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func payloadString(payload map[string]any, key string) string {
	if len(payload) == 0 {
		return ""
	}
	raw, ok := payload[key]
	if !ok {
		return ""
	}
	typed, _ := raw.(string)
	return typed
}
