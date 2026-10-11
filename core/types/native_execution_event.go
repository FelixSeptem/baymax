package types

import (
	"bytes"
	"encoding"
	"encoding/json"
	"errors"
	"io"
	"reflect"
	"strings"
	"time"
	"unicode/utf8"
)

const NativeExecutionEventVersionV1 = "native_execution_event.v1"

const (
	NativeExecutionReasonInvalid       = "native_execution.invalid"
	NativeExecutionReasonLimit         = "native_execution.limit"
	NativeExecutionReasonCorrelation   = "native_execution.correlation"
	NativeExecutionReasonUnsupported   = "native_execution.unsupported"
	NativeExecutionReasonSerialization = "native_execution.serialization"
)

type nativeExecutionError struct{ reason string }

func (e *nativeExecutionError) Error() string    { return e.reason }
func nativeExecutionFailure(reason string) error { return &nativeExecutionError{reason: reason} }

// NativeExecutionReasonOf returns a bounded classification, never an event body.
func NativeExecutionReasonOf(err error) string {
	var failure *nativeExecutionError
	if errors.As(err, &failure) {
		return failure.reason
	}
	if err != nil {
		return NativeExecutionReasonInvalid
	}
	return ""
}

// NativeExecutionHostReferences are application references, not execution or
// authorization facts. Unknown JSON keys are rejected.
type NativeExecutionHostReferences struct {
	TaskID                       string   `json:"task_id,omitempty"`
	ScheduleOccurrenceReference  string   `json:"schedule_occurrence_reference,omitempty"`
	PlanVersion                  string   `json:"plan_version,omitempty"`
	PlanDigest                   string   `json:"plan_digest,omitempty"`
	ProjectionDigest             string   `json:"projection_digest,omitempty"`
	KnowledgeGenerationReference string   `json:"knowledge_generation_reference,omitempty"`
	CitationReferences           []string `json:"citation_references,omitempty"`
}

func (r *NativeExecutionHostReferences) UnmarshalJSON(raw []byte) error {
	if !utf8.Valid(raw) {
		return nativeExecutionFailure(NativeExecutionReasonInvalid)
	}
	type plain NativeExecutionHostReferences
	var value plain
	d := json.NewDecoder(bytes.NewReader(raw))
	d.DisallowUnknownFields()
	if err := d.Decode(&value); err != nil {
		return nativeExecutionFailure(NativeExecutionReasonInvalid)
	}
	if err := d.Decode(new(any)); err != io.EOF {
		return nativeExecutionFailure(NativeExecutionReasonInvalid)
	}
	// encoding/json accepts null for string fields and string slice elements.
	// References require the declared JSON types, including for optional fields.
	var fields map[string]any
	if err := json.Unmarshal(raw, &fields); err != nil || fields == nil {
		return nativeExecutionFailure(NativeExecutionReasonInvalid)
	}
	for key, field := range fields {
		if key == "citation_references" {
			items, ok := field.([]any)
			if !ok {
				return nativeExecutionFailure(NativeExecutionReasonInvalid)
			}
			for _, item := range items {
				if _, ok := item.(string); !ok {
					return nativeExecutionFailure(NativeExecutionReasonInvalid)
				}
			}
		} else if _, ok := field.(string); !ok {
			return nativeExecutionFailure(NativeExecutionReasonInvalid)
		}
	}
	*r = NativeExecutionHostReferences(value)
	return nil
}

// NativeExecutionLimits apply to UTF-8 bytes and serialized JSON bytes. Explicit
// limits must all be positive; these are constructor values, not runtime keys.
type NativeExecutionLimits struct {
	ScalarReferenceBytes int `json:"scalar_reference_bytes"`
	CitationCount        int `json:"citation_count"`
	HostReferenceBytes   int `json:"host_reference_bytes"`
	EventBytes           int `json:"event_bytes"`
}

func DefaultNativeExecutionLimits() NativeExecutionLimits {
	return NativeExecutionLimits{1024, 16, 8 * 1024, 64 * 1024}
}

type NativeExecutionCapabilities struct {
	Identity        string `json:"identity"`
	Ordering        string `json:"ordering"`
	Cursor          bool   `json:"cursor"`
	DurableHistory  bool   `json:"durable_history"`
	Deduplication   bool   `json:"deduplication"`
	GapVerification bool   `json:"gap_verification"`
}

type NativeExecutionEvent struct {
	Version        string                        `json:"version"`
	Event          EventEnvelope                 `json:"event"`
	NativeType     string                        `json:"native_type"`
	NativeVersion  string                        `json:"native_version,omitempty"`
	Iteration      int                           `json:"iteration,omitempty"`
	TraceID        string                        `json:"trace_id,omitempty"`
	SpanID         string                        `json:"span_id,omitempty"`
	Capabilities   NativeExecutionCapabilities   `json:"capabilities"`
	HostReferences NativeExecutionHostReferences `json:"host_references"`
}

// IsTerminal classifies a source fact. It has no cross-event lifecycle state.
// input_required and dynamic action resolution are never terminal.
func (e NativeExecutionEvent) IsTerminal() bool {
	if e.Event.Source == ProtocolSourceRealtime {
		return e.NativeType == string(RealtimeEventTypeComplete)
	}
	if e.NativeType != "run.finished" {
		return false
	}
	state := e.SourceState()
	return state == "completed" || state == "failed" || state == "canceled"
}

// SourceState reads the source's terminal projection or explicit pause state.
// It does not infer an outcome from status, acknowledgement, or event names.
func (e NativeExecutionEvent) SourceState() string {
	if outcome, ok := e.Event.Payload["terminal_outcome"].(map[string]any); ok {
		if state, ok := outcome["state"].(string); ok {
			return state
		}
	}
	if state, ok := e.Event.Payload["terminal_state"].(string); ok {
		return state
	}
	state, _ := e.Event.Payload["state"].(string)
	return state
}

type NativeExecutionOptions struct {
	Source               ProtocolSource                `json:"source"`
	ExpectedRunID        string                        `json:"expected_run_id,omitempty"`
	SessionID            string                        `json:"session_id,omitempty"`
	HostReferences       NativeExecutionHostReferences `json:"host_references,omitempty"`
	Limits               *NativeExecutionLimits        `json:"limits,omitempty"`
	RequiredCapabilities []string                      `json:"required_capabilities,omitempty"`
}

// NativeExecutionProjector is immutable after construction and owns no source,
// delivery, cursor, history, deduplication, or terminal state.
type NativeExecutionProjector struct {
	source           ProtocolSource
	runID, sessionID string
	references       NativeExecutionHostReferences
	limits           NativeExecutionLimits
	capabilities     NativeExecutionCapabilities
}

func NewNativeExecutionProjector(opts NativeExecutionOptions) (*NativeExecutionProjector, error) {
	caps := NativeExecutionCapabilities{Identity: "mapping_reference", Ordering: "callback_order"}
	switch opts.Source {
	case ProtocolSourceRunner, ProtocolSourceWorkflow, ProtocolSourceTeams, ProtocolSourceComposer, ProtocolSourceScheduler, ProtocolSourceA2A, ProtocolSourceTimeline:
	case ProtocolSourceRealtime:
		caps = NativeExecutionCapabilities{Identity: "source_owned", Ordering: "source_sequence", Cursor: true, DurableHistory: true, Deduplication: true, GapVerification: true}
	default:
		return nil, nativeExecutionFailure(NativeExecutionReasonUnsupported)
	}
	for _, required := range opts.RequiredCapabilities {
		supported := false
		switch required {
		case "cursor":
			supported = caps.Cursor
		case "durable_history":
			supported = caps.DurableHistory
		case "deduplication":
			supported = caps.Deduplication
		case "gap_verification":
			supported = caps.GapVerification
		case "callback_order":
			supported = caps.Ordering == required
		case "source_sequence":
			supported = caps.Ordering == required
		case "source_owned":
			supported = caps.Identity == required
		case "mapping_reference":
			supported = caps.Identity == required
		}
		if !supported {
			return nil, nativeExecutionFailure(NativeExecutionReasonUnsupported)
		}
	}
	limits := DefaultNativeExecutionLimits()
	if opts.Limits != nil {
		limits = *opts.Limits
	}
	if limits.ScalarReferenceBytes <= 0 || limits.CitationCount <= 0 || limits.HostReferenceBytes <= 0 || limits.EventBytes <= 0 {
		return nil, nativeExecutionFailure(NativeExecutionReasonInvalid)
	}
	refs := opts.HostReferences
	if len(refs.CitationReferences) > limits.CitationCount {
		return nil, nativeExecutionFailure(NativeExecutionReasonLimit)
	}
	values := make([]string, 0, 6+len(refs.CitationReferences))
	values = append(values, refs.TaskID, refs.ScheduleOccurrenceReference, refs.PlanVersion, refs.PlanDigest, refs.ProjectionDigest, refs.KnowledgeGenerationReference)
	values = append(values, refs.CitationReferences...)
	for _, v := range values {
		if !utf8.ValidString(v) {
			return nil, nativeExecutionFailure(NativeExecutionReasonInvalid)
		}
		if len(v) > limits.ScalarReferenceBytes {
			return nil, nativeExecutionFailure(NativeExecutionReasonLimit)
		}
	}
	if !utf8.ValidString(opts.ExpectedRunID) || !utf8.ValidString(opts.SessionID) {
		return nil, nativeExecutionFailure(NativeExecutionReasonInvalid)
	}
	encoded, err := json.Marshal(refs)
	if err != nil {
		return nil, nativeExecutionFailure(NativeExecutionReasonSerialization)
	}
	if len(encoded) > limits.HostReferenceBytes {
		return nil, nativeExecutionFailure(NativeExecutionReasonLimit)
	}
	refs.CitationReferences = append([]string(nil), refs.CitationReferences...)
	return &NativeExecutionProjector{source: opts.Source, runID: strings.TrimSpace(opts.ExpectedRunID), sessionID: strings.TrimSpace(opts.SessionID), references: refs, limits: limits, capabilities: caps}, nil
}

func (p *NativeExecutionProjector) checkCorrelation(runID, sessionID string) error {
	if p.runID != "" && strings.TrimSpace(runID) != p.runID || p.sessionID != "" && sessionID != "" && strings.TrimSpace(sessionID) != p.sessionID {
		return nativeExecutionFailure(NativeExecutionReasonCorrelation)
	}
	return nil
}

func (p *NativeExecutionProjector) ProjectEvent(ev Event) (NativeExecutionEvent, error) {
	if p.source == ProtocolSourceRealtime {
		return NativeExecutionEvent{}, nativeExecutionFailure(NativeExecutionReasonUnsupported)
	}
	sessionID := ""
	if value, ok := ev.Payload["session_id"]; ok {
		var valid bool
		sessionID, valid = value.(string)
		if !valid {
			return NativeExecutionEvent{}, nativeExecutionFailure(NativeExecutionReasonCorrelation)
		}
	}
	if err := p.checkCorrelation(ev.RunID, sessionID); err != nil {
		return NativeExecutionEvent{}, err
	}
	// Validate before the mapper can trim strings or JSON can replace bad UTF-8.
	budget := p.limits.EventBytes
	if err := nativeExecutionInspect(reflect.ValueOf(ev), 0, &budget); err != nil {
		return NativeExecutionEvent{}, err
	}
	mapped, err := MapEventToProtocol(ev, p.source)
	if err != nil {
		return NativeExecutionEvent{}, nativeExecutionFailure(NativeExecutionReasonInvalid)
	}
	mapped.SessionID = sessionID
	if mapped.SessionID == "" {
		mapped.SessionID = p.sessionID
	}
	return p.snapshot(NativeExecutionEvent{Version: NativeExecutionEventVersionV1, Event: mapped, NativeType: ev.Type, NativeVersion: ev.Version, Iteration: ev.Iteration, TraceID: ev.TraceID, SpanID: ev.SpanID, Capabilities: p.capabilities, HostReferences: p.references})
}

type NativeExecutionBinding struct {
	Subscription EventStreamSubscription   `json:"subscription"`
	Outcome      EventStreamBindingOutcome `json:"outcome"`
	Events       []NativeExecutionEvent    `json:"events,omitempty"`
}

func (p *NativeExecutionProjector) ProjectRealtimeBinding(binding EventStreamBindingProjection) (NativeExecutionBinding, error) {
	if p.source != ProtocolSourceRealtime {
		return NativeExecutionBinding{}, nativeExecutionFailure(NativeExecutionReasonUnsupported)
	}
	if err := p.checkCorrelation(binding.Subscription.RunID, binding.Subscription.SessionID); err != nil {
		return NativeExecutionBinding{}, err
	}
	// Empty/expired batches still carry source metadata. Bound and validate it
	// before the source normalizer can trim or encode these strings.
	budget := p.limits.EventBytes
	metadata := NativeExecutionBinding{Subscription: binding.Subscription, Outcome: binding.Outcome}
	if err := nativeExecutionInspect(reflect.ValueOf(metadata), 0, &budget); err != nil {
		return NativeExecutionBinding{}, err
	}
	encoded, err := json.Marshal(metadata)
	if err != nil {
		return NativeExecutionBinding{}, nativeExecutionFailure(NativeExecutionReasonSerialization)
	}
	if len(encoded) > p.limits.EventBytes {
		return NativeExecutionBinding{}, nativeExecutionFailure(NativeExecutionReasonLimit)
	}
	// Inspect before source cloning (which assumes acyclic JSON-compatible input).
	for _, event := range binding.Events {
		budget := p.limits.EventBytes
		if err := nativeExecutionInspect(reflect.ValueOf(event), 0, &budget); err != nil {
			return NativeExecutionBinding{}, err
		}
	}
	normalized, err := ProjectEventStreamBinding(binding.Subscription, binding.Outcome, binding.Events, nil)
	if err != nil {
		return NativeExecutionBinding{}, err
	}
	mapped, err := MapEventStreamBindingToProtocol(normalized)
	if err != nil {
		return NativeExecutionBinding{}, err
	}
	result := NativeExecutionBinding{Subscription: normalized.Subscription, Outcome: normalized.Outcome, Events: make([]NativeExecutionEvent, 0, len(mapped))}
	for i, event := range mapped {
		projected, err := p.snapshot(NativeExecutionEvent{Version: NativeExecutionEventVersionV1, Event: event, NativeType: string(normalized.Events[i].Type), Capabilities: p.capabilities, HostReferences: p.references})
		if err != nil {
			return NativeExecutionBinding{}, err
		}
		result.Events = append(result.Events, projected)
	}
	return result, nil
}

func (p *NativeExecutionProjector) snapshot(value NativeExecutionEvent) (NativeExecutionEvent, error) {
	encoded, err := json.Marshal(value)
	if err != nil {
		return NativeExecutionEvent{}, nativeExecutionFailure(NativeExecutionReasonSerialization)
	}
	if len(encoded) > p.limits.EventBytes {
		return NativeExecutionEvent{}, nativeExecutionFailure(NativeExecutionReasonLimit)
	}
	var result NativeExecutionEvent
	decoder := json.NewDecoder(bytes.NewReader(encoded))
	decoder.UseNumber()
	if err := decoder.Decode(&result); err != nil {
		return NativeExecutionEvent{}, nativeExecutionFailure(NativeExecutionReasonSerialization)
	}
	return result, nil
}

// Inspection bounds traversal before JSON encoding and rejects cycles/deep
// values. The byte budget is a conservative lower bound; final JSON size is
// checked separately. Custom serialization is rejected before invocation;
// time.Time is the sole supported standard-library marshaler.
func nativeExecutionInspect(v reflect.Value, depth int, budget *int) error {
	if depth > 128 {
		return nativeExecutionFailure(NativeExecutionReasonSerialization)
	}
	if !v.IsValid() {
		return nil
	}
	if v.CanInterface() && v.Type() != reflect.TypeOf(time.Time{}) && v.Type() != reflect.TypeOf((*time.Time)(nil)) {
		if _, ok := v.Interface().(json.Marshaler); ok {
			return nativeExecutionFailure(NativeExecutionReasonSerialization)
		}
		if _, ok := v.Interface().(encoding.TextMarshaler); ok {
			return nativeExecutionFailure(NativeExecutionReasonSerialization)
		}
		if v.CanAddr() && v.Addr().CanInterface() {
			if _, ok := v.Addr().Interface().(json.Marshaler); ok {
				return nativeExecutionFailure(NativeExecutionReasonSerialization)
			}
			if _, ok := v.Addr().Interface().(encoding.TextMarshaler); ok {
				return nativeExecutionFailure(NativeExecutionReasonSerialization)
			}
		}
	}
	*budget--
	if *budget < 0 {
		return nativeExecutionFailure(NativeExecutionReasonLimit)
	}
	switch v.Kind() {
	case reflect.Interface, reflect.Pointer:
		if !v.IsNil() {
			return nativeExecutionInspect(v.Elem(), depth+1, budget)
		}
	case reflect.String:
		if !utf8.ValidString(v.String()) {
			return nativeExecutionFailure(NativeExecutionReasonInvalid)
		}
		*budget -= len(v.String())
		if *budget < 0 {
			return nativeExecutionFailure(NativeExecutionReasonLimit)
		}
	case reflect.Map:
		iter := v.MapRange()
		for iter.Next() {
			if err := nativeExecutionInspect(iter.Key(), depth+1, budget); err != nil {
				return err
			}
			if err := nativeExecutionInspect(iter.Value(), depth+1, budget); err != nil {
				return err
			}
		}
	case reflect.Slice, reflect.Array:
		for i := 0; i < v.Len(); i++ {
			if err := nativeExecutionInspect(v.Index(i), depth+1, budget); err != nil {
				return err
			}
		}
	case reflect.Struct:
		for i := 0; i < v.NumField(); i++ {
			if v.Type().Field(i).IsExported() {
				if err := nativeExecutionInspect(v.Field(i), depth+1, budget); err != nil {
					return err
				}
			}
		}
	case reflect.Chan, reflect.Func, reflect.Complex64, reflect.Complex128, reflect.UnsafePointer:
		return nativeExecutionFailure(NativeExecutionReasonSerialization)
	}
	return nil
}
