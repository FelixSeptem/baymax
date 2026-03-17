package diagnosticsreplay

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"
)

const (
	ReasonCodeInvalidJSON          = "invalid_json"
	ReasonCodeInvalidJSONShape     = "invalid_json_shape"
	ReasonCodeMissingTimelineArray = "missing_timeline_events"
	ReasonCodeMissingRequiredField = "missing_required_field"
	ReasonCodeInvalidFieldType     = "invalid_field_type"
	ReasonCodeInvalidTimestamp     = "invalid_timestamp"
)

// MinimalEvent is the normalized replay output record for diagnostics timeline review.
type MinimalEvent struct {
	RunID     string    `json:"run_id"`
	Sequence  int64     `json:"sequence"`
	Phase     string    `json:"phase"`
	Status    string    `json:"status"`
	Reason    string    `json:"reason,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

// ReplayOutput is the stable JSON envelope emitted by replay tooling.
type ReplayOutput struct {
	Events []MinimalEvent `json:"events"`
}

// ValidationError is a deterministic replay validation failure with machine-readable code.
type ValidationError struct {
	Code    string
	Message string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// ParseMinimalReplayJSON parses diagnostics JSON into normalized minimal replay events.
func ParseMinimalReplayJSON(raw []byte) (ReplayOutput, error) {
	var root any
	if err := json.Unmarshal(raw, &root); err != nil {
		return ReplayOutput{}, &ValidationError{
			Code:    ReasonCodeInvalidJSON,
			Message: err.Error(),
		}
	}

	events, err := parseFromRoot(root)
	if err != nil {
		return ReplayOutput{}, err
	}
	sortMinimalEvents(events)
	return ReplayOutput{Events: events}, nil
}

func parseFromRoot(root any) ([]MinimalEvent, error) {
	switch tv := root.(type) {
	case []any:
		return parseTimelineArray(tv, "root")
	case map[string]any:
		if timeline, ok := tv["timeline_events"]; ok {
			arr, ok := timeline.([]any)
			if !ok {
				return nil, &ValidationError{
					Code:    ReasonCodeInvalidFieldType,
					Message: "timeline_events must be array",
				}
			}
			return parseTimelineArray(arr, "timeline_events")
		}
		if eventsRaw, ok := tv["events"]; ok {
			arr, ok := eventsRaw.([]any)
			if !ok {
				return nil, &ValidationError{
					Code:    ReasonCodeInvalidFieldType,
					Message: "events must be array",
				}
			}
			return parseEventEnvelopeArray(arr)
		}
		return nil, &ValidationError{
			Code:    ReasonCodeMissingTimelineArray,
			Message: "missing timeline_events/events array",
		}
	default:
		return nil, &ValidationError{
			Code:    ReasonCodeInvalidJSONShape,
			Message: "root must be object or array",
		}
	}
}

func parseTimelineArray(items []any, source string) ([]MinimalEvent, error) {
	if len(items) == 0 {
		return nil, &ValidationError{
			Code:    ReasonCodeMissingTimelineArray,
			Message: source + " array is empty",
		}
	}
	out := make([]MinimalEvent, 0, len(items))
	for i, raw := range items {
		item, ok := raw.(map[string]any)
		if !ok {
			return nil, &ValidationError{
				Code:    ReasonCodeInvalidFieldType,
				Message: fmt.Sprintf("%s[%d] must be object", source, i),
			}
		}
		ev, err := parseMinimalEvent(item, fmt.Sprintf("%s[%d]", source, i))
		if err != nil {
			return nil, err
		}
		out = append(out, ev)
	}
	return out, nil
}

func parseEventEnvelopeArray(items []any) ([]MinimalEvent, error) {
	if len(items) == 0 {
		return nil, &ValidationError{
			Code:    ReasonCodeMissingTimelineArray,
			Message: "events array is empty",
		}
	}
	out := make([]MinimalEvent, 0, len(items))
	for i, raw := range items {
		item, ok := raw.(map[string]any)
		if !ok {
			return nil, &ValidationError{
				Code:    ReasonCodeInvalidFieldType,
				Message: fmt.Sprintf("events[%d] must be object", i),
			}
		}
		typ := strings.TrimSpace(toString(item["type"]))
		if typ != "action.timeline" {
			continue
		}
		runID := strings.TrimSpace(toString(item["run_id"]))
		if runID == "" {
			return nil, missingFieldError(fmt.Sprintf("events[%d].run_id", i))
		}
		timestamp, err := parseTimestamp(item, fmt.Sprintf("events[%d]", i))
		if err != nil {
			return nil, err
		}
		payloadRaw, ok := item["payload"]
		if !ok {
			return nil, missingFieldError(fmt.Sprintf("events[%d].payload", i))
		}
		payload, ok := payloadRaw.(map[string]any)
		if !ok {
			return nil, &ValidationError{
				Code:    ReasonCodeInvalidFieldType,
				Message: fmt.Sprintf("events[%d].payload must be object", i),
			}
		}
		sequence, err := requiredPositiveInt64(payload, "sequence", fmt.Sprintf("events[%d].payload", i))
		if err != nil {
			return nil, err
		}
		phase := strings.TrimSpace(toString(payload["phase"]))
		if phase == "" {
			return nil, missingFieldError(fmt.Sprintf("events[%d].payload.phase", i))
		}
		status := strings.TrimSpace(toString(payload["status"]))
		if status == "" {
			return nil, missingFieldError(fmt.Sprintf("events[%d].payload.status", i))
		}
		out = append(out, MinimalEvent{
			RunID:     runID,
			Sequence:  sequence,
			Phase:     phase,
			Status:    status,
			Reason:    strings.TrimSpace(toString(payload["reason"])),
			Timestamp: timestamp,
		})
	}
	if len(out) == 0 {
		return nil, &ValidationError{
			Code:    ReasonCodeMissingTimelineArray,
			Message: "events array contains no action.timeline items",
		}
	}
	return out, nil
}

func parseMinimalEvent(item map[string]any, path string) (MinimalEvent, error) {
	runID := strings.TrimSpace(toString(item["run_id"]))
	if runID == "" {
		return MinimalEvent{}, missingFieldError(path + ".run_id")
	}
	sequence, err := requiredPositiveInt64(item, "sequence", path)
	if err != nil {
		return MinimalEvent{}, err
	}
	phase := strings.TrimSpace(toString(item["phase"]))
	if phase == "" {
		return MinimalEvent{}, missingFieldError(path + ".phase")
	}
	status := strings.TrimSpace(toString(item["status"]))
	if status == "" {
		return MinimalEvent{}, missingFieldError(path + ".status")
	}
	ts, err := parseTimestamp(item, path)
	if err != nil {
		return MinimalEvent{}, err
	}
	return MinimalEvent{
		RunID:     runID,
		Sequence:  sequence,
		Phase:     phase,
		Status:    status,
		Reason:    strings.TrimSpace(toString(item["reason"])),
		Timestamp: ts,
	}, nil
}

func requiredPositiveInt64(item map[string]any, key, path string) (int64, error) {
	raw, ok := item[key]
	if !ok {
		return 0, missingFieldError(path + "." + key)
	}
	value, ok := toInt64(raw)
	if !ok {
		return 0, &ValidationError{
			Code:    ReasonCodeInvalidFieldType,
			Message: fmt.Sprintf("%s.%s must be integer", path, key),
		}
	}
	if value <= 0 {
		return 0, &ValidationError{
			Code:    ReasonCodeMissingRequiredField,
			Message: fmt.Sprintf("%s.%s must be > 0", path, key),
		}
	}
	return value, nil
}

func parseTimestamp(item map[string]any, path string) (time.Time, error) {
	raw, ok := item["timestamp"]
	if !ok {
		raw, ok = item["time"]
	}
	if !ok {
		return time.Time{}, missingFieldError(path + ".timestamp")
	}
	value := strings.TrimSpace(toString(raw))
	if value == "" {
		return time.Time{}, missingFieldError(path + ".timestamp")
	}
	ts, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		return time.Time{}, &ValidationError{
			Code:    ReasonCodeInvalidTimestamp,
			Message: fmt.Sprintf("%s timestamp parse failed: %v", path, err),
		}
	}
	return ts, nil
}

func sortMinimalEvents(events []MinimalEvent) {
	sort.SliceStable(events, func(i, j int) bool {
		if events[i].Sequence != events[j].Sequence {
			return events[i].Sequence < events[j].Sequence
		}
		if !events[i].Timestamp.Equal(events[j].Timestamp) {
			return events[i].Timestamp.Before(events[j].Timestamp)
		}
		if events[i].RunID != events[j].RunID {
			return events[i].RunID < events[j].RunID
		}
		if events[i].Phase != events[j].Phase {
			return events[i].Phase < events[j].Phase
		}
		return events[i].Status < events[j].Status
	})
}

func missingFieldError(path string) *ValidationError {
	return &ValidationError{
		Code:    ReasonCodeMissingRequiredField,
		Message: path + " is required",
	}
}

func toString(raw any) string {
	switch tv := raw.(type) {
	case string:
		return tv
	default:
		return ""
	}
}

func toInt64(raw any) (int64, bool) {
	switch tv := raw.(type) {
	case int:
		return int64(tv), true
	case int8:
		return int64(tv), true
	case int16:
		return int64(tv), true
	case int32:
		return int64(tv), true
	case int64:
		return tv, true
	case float64:
		if float64(int64(tv)) != tv {
			return 0, false
		}
		return int64(tv), true
	default:
		return 0, false
	}
}
