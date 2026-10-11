package diagnosticsreplay

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"reflect"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/FelixSeptem/baymax/core/types"
)

const (
	ReasonNativeExecutionReplayDrift   = "native_execution.replay_drift"
	ReasonNativeExecutionParityDrift   = "native_execution.parity_drift"
	ReasonNativeExecutionPauseTerminal = "native_execution.pause_as_terminal"
)

type NativeExecutionFixture struct {
	Version string                       `json:"version"`
	Cases   []NativeExecutionFixtureCase `json:"cases"`
}
type NativeExecutionFixtureCase struct {
	CaseID                 string                              `json:"case_id"`
	Options                types.NativeExecutionOptions        `json:"options"`
	Events                 []types.Event                       `json:"events,omitempty"`
	Binding                *types.EventStreamBindingProjection `json:"binding,omitempty"`
	Observation            *NativeExecutionDeliveryEvidence    `json:"observation,omitempty"`
	ClaimTerminal          bool                                `json:"claim_terminal,omitempty"`
	ExpectedReason         string                              `json:"expected_reason"`
	ExpectedEvents         int                                 `json:"expected_events"`
	ExpectedTerminals      int                                 `json:"expected_terminals"`
	ExpectedSharedIdentity bool                                `json:"expected_shared_identity,omitempty"`
}
type NativeExecutionReplayCase struct {
	CaseID    string `json:"case_id"`
	Reason    string `json:"reason"`
	Events    int    `json:"events"`
	Terminals int    `json:"terminals"`
	Digest    string `json:"digest"`
}

// NativeExecutionDeliveryEvidence is a bounded offline receipt projection.
// Replay checks recorded evidence; it never calls a live observer or sink.
type NativeExecutionDeliveryEvidence struct {
	State             string `json:"state"`
	Reason            string `json:"reason,omitempty"`
	Received          uint64 `json:"received"`
	Delivered         uint64 `json:"delivered"`
	Unconfirmed       uint64 `json:"unconfirmed"`
	Skipped           uint64 `json:"skipped"`
	TerminalDelivered uint64 `json:"terminal_delivered"`
	Incomplete        bool   `json:"incomplete"`
}
type NativeExecutionTranscript struct {
	Events      []types.NativeExecutionEvent    `json:"events"`
	Observation NativeExecutionDeliveryEvidence `json:"observation"`
}

func nativeReplayError(reason string) error {
	return &ValidationError{Code: reason, Message: "native execution conformance failed"}
}

func ReplayNativeExecutionFixtureJSON(raw []byte) ([]NativeExecutionReplayCase, error) {
	if len(raw) > 4*1024*1024 || !utf8.Valid(raw) {
		return nil, nativeReplayError(ReasonNativeExecutionReplayDrift)
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	decoder.UseNumber()
	var fixture NativeExecutionFixture
	if err := decoder.Decode(&fixture); err != nil {
		return nil, nativeReplayError(ReasonNativeExecutionReplayDrift)
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		return nil, nativeReplayError(ReasonNativeExecutionReplayDrift)
	}
	if fixture.Version != types.NativeExecutionEventVersionV1 || len(fixture.Cases) == 0 || len(fixture.Cases) > 256 {
		return nil, nativeReplayError(ReasonNativeExecutionReplayDrift)
	}
	seen := make(map[string]bool, len(fixture.Cases))
	results := make([]NativeExecutionReplayCase, 0, len(fixture.Cases))
	for _, item := range fixture.Cases {
		if item.CaseID == "" || len(item.CaseID) > 256 || seen[item.CaseID] || len(item.Events) > 256 {
			return nil, nativeReplayError(ReasonNativeExecutionReplayDrift)
		}
		seen[item.CaseID] = true
		first, err := replayNativeExecutionCase(item)
		if err != nil {
			return nil, err
		}
		second, err := replayNativeExecutionCase(item)
		if err != nil || first != second {
			return nil, nativeReplayError(ReasonNativeExecutionReplayDrift)
		}
		results = append(results, first)
	}
	return results, nil
}

func replayNativeExecutionCase(item NativeExecutionFixtureCase) (NativeExecutionReplayCase, error) {
	p, err := types.NewNativeExecutionProjector(item.Options)
	var projected []types.NativeExecutionEvent
	if err == nil {
		if item.Binding != nil {
			if len(item.Events) != 0 {
				return NativeExecutionReplayCase{}, nativeReplayError(ReasonNativeExecutionReplayDrift)
			}
			var binding types.NativeExecutionBinding
			binding, err = p.ProjectRealtimeBinding(*item.Binding)
			projected = binding.Events
		} else {
			for _, event := range item.Events {
				var e types.NativeExecutionEvent
				e, err = p.ProjectEvent(event)
				if err != nil {
					break
				}
				projected = append(projected, e)
			}
		}
	}
	reason := types.NativeExecutionReasonOf(err)
	if err != nil {
		// Preserve existing source binding error classifications without copying
		// its correlation/body-bearing error message into the replay result.
		for _, code := range []string{types.ProtocolReasonEventStreamSequenceGap, types.ProtocolReasonEventStreamInvalidSubscription, types.ProtocolReasonEventStreamIncompatibleDeliveryOutcome} {
			if strings.Contains(err.Error(), code) {
				reason = code
				break
			}
		}
	}
	terminals := 0
	shared := false
	for i, e := range projected {
		if e.IsTerminal() {
			terminals++
		}
		if item.ClaimTerminal && e.SourceState() == "input_required" {
			reason = ReasonNativeExecutionPauseTerminal
		}
		for j := 0; j < i; j++ {
			if projected[j].Event.EventID == e.Event.EventID {
				shared = true
			}
		}
	}
	if item.Observation != nil {
		if err := validateNativeEvidence(*item.Observation); err != nil {
			return NativeExecutionReplayCase{}, err
		}
		if reason == "" {
			reason = item.Observation.Reason
		}
	}
	if reason != item.ExpectedReason || len(projected) != item.ExpectedEvents || terminals != item.ExpectedTerminals || item.ExpectedSharedIdentity && !shared {
		return NativeExecutionReplayCase{}, nativeReplayError(ReasonNativeExecutionReplayDrift)
	}
	encoded, err := json.Marshal(struct {
		Events      []types.NativeExecutionEvent
		Binding     *types.EventStreamBindingProjection
		Observation *NativeExecutionDeliveryEvidence
	}{projected, item.Binding, item.Observation})
	if err != nil {
		return NativeExecutionReplayCase{}, nativeReplayError(ReasonNativeExecutionReplayDrift)
	}
	sum := sha256.Sum256(encoded)
	return NativeExecutionReplayCase{CaseID: item.CaseID, Reason: reason, Events: len(projected), Terminals: terminals, Digest: hex.EncodeToString(sum[:])}, nil
}

func validateNativeEvidence(e NativeExecutionDeliveryEvidence) error {
	if e.State != "healthy" && e.State != "failed" && e.State != "closed" {
		return nativeReplayError(ReasonNativeExecutionReplayDrift)
	}
	if e.TerminalDelivered > e.Delivered || e.Delivered > e.Received || e.Unconfirmed > e.Received-e.Delivered || e.Skipped > e.Received-e.Delivered-e.Unconfirmed {
		return nativeReplayError(ReasonNativeExecutionReplayDrift)
	}
	if e.Incomplete != (e.Reason != "" || e.TerminalDelivered == 0) || e.State == "failed" && e.Reason == "" || e.State == "healthy" && e.Reason != "" {
		return nativeReplayError(ReasonNativeExecutionReplayDrift)
	}
	if e.Reason != "" {
		switch e.Reason {
		case "observation.sink_error", "observation.timeout", "observation.concurrent", "observation.closed", "observation.context_canceled", types.NativeExecutionReasonInvalid, types.NativeExecutionReasonLimit, types.NativeExecutionReasonCorrelation, types.NativeExecutionReasonSerialization:
		default:
			return nativeReplayError(ReasonNativeExecutionReplayDrift)
		}
	}
	return nil
}

// CompareNativeExecutionParity normalizes generated references by equality
// class, timestamps/timing measurements, and model.delta fragmentation only.
// Fallback mapped IDs are verified against the existing mapper before removing
// their clock-dependent value. Native lifecycle/order, source, payload facts,
// failure families and receipt failures remain observable.
func CompareNativeExecutionParity(run, stream NativeExecutionTranscript) error {
	a, err := normalizeNativeTranscript(run)
	if err != nil {
		return err
	}
	b, err := normalizeNativeTranscript(stream)
	if err != nil {
		return err
	}
	if !reflect.DeepEqual(a, b) {
		return &ValidationError{Code: ReasonNativeExecutionParityDrift, Message: "transcript differs at " + nativeDifferencePath(a, b, "root")}
	}
	return nil
}

func nativeDifferencePath(a, b any, path string) string {
	if reflect.DeepEqual(a, b) {
		return ""
	}
	if left, ok := a.(map[string]any); ok {
		if right, ok := b.(map[string]any); ok {
			keys := make([]string, 0, len(left))
			for key := range left {
				keys = append(keys, key)
			}
			sort.Strings(keys)
			for key := range right {
				if _, ok := left[key]; !ok {
					keys = append(keys, key)
				}
			}
			sort.Strings(keys)
			for _, key := range keys {
				if _, ok := right[key]; !ok {
					return path + "." + key + ".presence"
				}
				if _, ok := left[key]; !ok {
					return path + "." + key + ".presence"
				}
				if result := nativeDifferencePath(left[key], right[key], path+"."+key); result != "" {
					return result
				}
			}
		}
	}
	if left, ok := a.([]any); ok {
		if right, ok := b.([]any); ok {
			if len(left) != len(right) {
				return path + ".length"
			}
			for i := range left {
				if result := nativeDifferencePath(left[i], right[i], fmt.Sprintf("%s[%d]", path, i)); result != "" {
					return result
				}
			}
		}
	}
	if len(path) > 256 {
		return path[:256]
	}
	return path
}

func normalizeNativeTranscript(input NativeExecutionTranscript) (map[string]any, error) {
	if len(input.Events) > 4096 {
		return nil, nativeReplayError(ReasonNativeExecutionParityDrift)
	}
	if err := validateNativeEvidence(input.Observation); err != nil {
		return nil, err
	}
	if input.Observation.Delivered != uint64(len(input.Events)) {
		return nil, nativeReplayError(ReasonNativeExecutionParityDrift)
	}
	ids := make(map[string]map[string]string)
	var events []any
	var deltas uint64
	for _, event := range input.Events {
		if event.Version != types.NativeExecutionEventVersionV1 {
			return nil, nativeReplayError(ReasonNativeExecutionParityDrift)
		}
		value := event
		if value.Capabilities.Identity == "mapping_reference" {
			mapped, err := types.MapEventToProtocol(types.Event{Type: value.NativeType, RunID: value.Event.RunID, CallID: value.Event.StepID, TraceID: value.TraceID, Time: value.Event.Time}, value.Event.Source)
			if err != nil || mapped.EventID != value.Event.EventID {
				return nil, nativeReplayError(ReasonNativeExecutionParityDrift)
			}
			if value.TraceID == "" {
				value.Event.EventID = ""
			}
		}
		if nativeFragmentOnly(value) {
			// Fragment callbacks must still belong to an observed model step.
			matched := false
			for _, source := range input.Events {
				if source.NativeType == "model.requested" && source.Event.Source == value.Event.Source && source.Event.RunID == value.Event.RunID && source.Event.SessionID == value.Event.SessionID && source.Event.StepID == value.Event.StepID && source.Iteration == value.Iteration && source.NativeVersion == value.NativeVersion && source.Capabilities == value.Capabilities && reflect.DeepEqual(source.HostReferences, value.HostReferences) {
					matched = true
					break
				}
			}
			if !matched {
				return nil, nativeReplayError(ReasonNativeExecutionParityDrift)
			}
			deltas++
			continue
		}
		encoded, err := json.Marshal(value)
		if err != nil {
			return nil, nativeReplayError(ReasonNativeExecutionParityDrift)
		}
		var obj map[string]any
		d := json.NewDecoder(bytes.NewReader(encoded))
		d.UseNumber()
		if err := d.Decode(&obj); err != nil {
			return nil, nativeReplayError(ReasonNativeExecutionParityDrift)
		}
		normalizeNativeValue(obj, ids, "")
		events = append(events, obj)
	}
	evidence := input.Observation
	if evidence.Received < deltas || evidence.Delivered < deltas {
		return nil, nativeReplayError(ReasonNativeExecutionParityDrift)
	}
	evidence.Received -= deltas
	evidence.Delivered -= deltas
	return map[string]any{"events": events, "observation": evidence}, nil
}

// Only known fragment representations may be removed. Error, clarification,
// metadata and unknown callbacks remain facts and must compare exactly.
func nativeFragmentOnly(event types.NativeExecutionEvent) bool {
	if event.NativeType != "model.delta" {
		return false
	}
	payload := event.Event.Payload
	for key := range payload {
		if key != "event_type" && key != "delta" && key != "tool_call" {
			return false
		}
	}
	kind, _ := payload["event_type"].(string)
	switch kind {
	case types.ModelEventTypeFinalAnswer, types.ModelEventTypeOutputTextDelta, types.ModelEventTypeOutputTextDone, types.ModelEventTypeToolCall, types.ModelEventTypeFunctionArgsDelta, types.ModelEventTypeFunctionArgsDone:
		return true
	default:
		return false
	}
}

func normalizeNativeValue(value any, ids map[string]map[string]string, path string) {
	switch object := value.(type) {
	case map[string]any:
		// Fixed traversal order is required for deterministic equality labels.
		keys := make([]string, 0, len(object))
		for key := range object {
			keys = append(keys, key)
		}
		// Sorting keys is independent of event order.
		sort.Strings(keys)
		for _, key := range keys {
			v := object[key]
			childPath := path + "." + key
			// Host references and arbitrary nested business values are literal.
			// Normalize only schema-owned envelope/native payload fields.
			allowed := path == "" || path == ".event" || path == ".event.payload" || path == ".event.payload.terminal_outcome" || path == ".event.payload.checkpoint"
			namespace := ""
			if allowed {
				switch key {
				case "run_id", "parent_run_id":
					namespace = "run"
				case "session_id":
					namespace = "session"
				case "step_id", "call_id", "parent_step_id":
					namespace = "step"
				case "trace_id", "event_id":
					namespace = "event"
				case "span_id":
					namespace = "span"
				case "causation_id", "terminal_causation_id":
					namespace = "causation"
				case "checkpoint_id":
					namespace = "checkpoint"
				case "checkpoint_digest":
					namespace = "checkpoint_digest"
				case "time", "ts", "latency_ms", "assemble_latency_ms":
					delete(object, key)
					continue
				}
			}
			if namespace != "" {
				if text, ok := v.(string); ok && text != "" {
					m := ids[namespace]
					if m == nil {
						m = make(map[string]string)
						ids[namespace] = m
					}
					label, exists := m[text]
					if !exists {
						label = fmt.Sprintf("%s:%d", namespace, len(m)+1)
						m[text] = label
					}
					object[key] = label
				}
			}
			normalizeNativeValue(object[key], ids, childPath)
		}
	case []any:
		for _, v := range object {
			normalizeNativeValue(v, ids, path+"[]")
		}
	}
}
