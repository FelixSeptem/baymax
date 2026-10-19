package types

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
	"time"
)

func nativeTestEvent() Event {
	return Event{Version: EventSchemaVersionV1, Type: "run.finished", RunID: "run-native", TraceID: "shared-trace", Iteration: 2, Time: time.Unix(100, 0), Payload: map[string]any{"state": "input_required", "nested": map[string]any{"values": []any{"before", int64(9007199254740993)}}}}
}

type nativeCustomJSON struct{ called *bool }

func (v nativeCustomJSON) MarshalJSON() ([]byte, error) {
	*v.called = true
	return []byte(`"custom"`), nil
}

type nativePointerJSON struct{ called *bool }

func (v *nativePointerJSON) MarshalJSON() ([]byte, error) {
	*v.called = true
	return []byte(`"custom"`), nil
}

func TestNativeExecutionRejectsAddressableCustomSerialization(t *testing.T) {
	p, _ := NewNativeExecutionProjector(NativeExecutionOptions{Source: ProtocolSourceRunner})
	called := false
	ev := nativeTestEvent()
	// encoding/json can invoke a pointer-receiver marshaler on slice elements.
	ev.Payload = map[string]any{"value": []nativePointerJSON{{called: &called}}}
	_, err := p.ProjectEvent(ev)
	if NativeExecutionReasonOf(err) != NativeExecutionReasonSerialization || called {
		t.Fatalf("addressable custom serializer invoked=%v, error=%v", called, err)
	}
}

func TestNativeExecutionRejectsCustomSerializationBeforeInvocation(t *testing.T) {
	p, _ := NewNativeExecutionProjector(NativeExecutionOptions{Source: ProtocolSourceRunner})
	called := false
	ev := nativeTestEvent()
	ev.Payload = map[string]any{"value": nativeCustomJSON{called: &called}}
	_, err := p.ProjectEvent(ev)
	if NativeExecutionReasonOf(err) != NativeExecutionReasonSerialization || called {
		t.Fatalf("custom serializer invoked=%v, error=%v", called, err)
	}
}

func TestNativeExecutionRealtimeMetadataLimits(t *testing.T) {
	p, _ := NewNativeExecutionProjector(NativeExecutionOptions{Source: ProtocolSourceRealtime})
	binding := EventStreamBindingProjection{
		Subscription: EventStreamSubscription{Version: DurableEventStreamBindingVersionV1, SubscriptionID: "sub", Source: ProtocolSourceRealtime, SessionID: "session", RunID: "run", StartMode: EventStreamStartAfterCursor, Cursor: EventStreamCursor{Value: "cursor", Sequence: 1}, MaxBatchSize: 4, DeliveryPolicy: EventStreamDeliveryPolicyReject},
		Outcome:      EventStreamBindingOutcome{SubscriptionID: "sub", Phase: EventStreamBindingPhaseExpired, ReasonCode: "source.expired", LastSequence: 1},
	}
	binding.Outcome.ReasonCode = string([]byte{0xff})
	if _, err := p.ProjectRealtimeBinding(binding); NativeExecutionReasonOf(err) != NativeExecutionReasonInvalid {
		t.Fatalf("invalid metadata UTF-8: %v", err)
	}
	binding.Outcome.ReasonCode = strings.Repeat("x", DefaultNativeExecutionLimits().EventBytes+1)
	if _, err := p.ProjectRealtimeBinding(binding); NativeExecutionReasonOf(err) != NativeExecutionReasonLimit {
		t.Fatalf("unbounded empty binding metadata: %v", err)
	}
}
func TestNativeExecutionMapperBaseline(t *testing.T) {
	ev := nativeTestEvent()
	first, err := MapEventToProtocol(ev, ProtocolSourceRunner)
	if err != nil {
		t.Fatal(err)
	}
	ev.Type = "tool.finished"
	second, err := MapEventToProtocol(ev, ProtocolSourceRunner)
	if err != nil {
		t.Fatal(err)
	}
	if first.EventID != second.EventID || first.Sequence != 0 {
		t.Fatal("mapper baseline changed")
	}
	ev.TraceID = ""
	first, _ = MapEventToProtocol(ev, ProtocolSourceRunner)
	ev.Payload = map[string]any{"changed": true}
	second, _ = MapEventToProtocol(ev, ProtocolSourceRunner)
	if first.EventID != second.EventID {
		t.Fatal("fallback identity baseline changed")
	}
}
func TestNativeExecutionProjectionSnapshotAndCapabilities(t *testing.T) {
	refs := NativeExecutionHostReferences{TaskID: "task", CitationReferences: []string{"citation"}}
	p, err := NewNativeExecutionProjector(NativeExecutionOptions{Source: ProtocolSourceRunner, ExpectedRunID: "run-native", SessionID: "session", HostReferences: refs})
	if err != nil {
		t.Fatal(err)
	}
	ev := nativeTestEvent()
	got, err := p.ProjectEvent(ev)
	if err != nil {
		t.Fatal(err)
	}
	ev.Payload["nested"].(map[string]any)["values"].([]any)[0] = "after"
	refs.CitationReferences[0] = "after"
	if got.Version != NativeExecutionEventVersionV1 || got.NativeType != "run.finished" || got.Event.SessionID != "session" || got.Capabilities.Identity != "mapping_reference" || got.Capabilities.Cursor || got.Capabilities.Deduplication || got.Event.Sequence != 0 || got.IsTerminal() {
		t.Fatalf("projection=%#v", got)
	}
	if got.Event.Payload["nested"].(map[string]any)["values"].([]any)[0] != "before" || got.HostReferences.CitationReferences[0] != "citation" {
		t.Fatal("snapshot aliased")
	}
	if got.Event.Payload["nested"].(map[string]any)["values"].([]any)[1].(json.Number).String() != "9007199254740993" {
		t.Fatal("integer lost precision")
	}
	for _, state := range []string{"completed", "failed", "canceled"} {
		ev.Payload["state"] = state
		got, err = p.ProjectEvent(ev)
		if err != nil || !got.IsTerminal() {
			t.Fatalf("terminal %s %v", state, err)
		}
	}
	ev.Type = "run.dynamic_action.resolved"
	got, err = p.ProjectEvent(ev)
	if err != nil || got.IsTerminal() {
		t.Fatal("resolution implies terminal")
	}
}
func TestNativeExecutionProjectionRejectsInvalidInput(t *testing.T) {
	for _, source := range []ProtocolSource{ProtocolSourceRunner, ProtocolSourceWorkflow, ProtocolSourceTeams, ProtocolSourceScheduler, ProtocolSourceComposer, ProtocolSourceA2A, ProtocolSourceTimeline} {
		p, err := NewNativeExecutionProjector(NativeExecutionOptions{Source: source, ExpectedRunID: "run-native"})
		if err != nil {
			t.Fatal(err)
		}
		got, err := p.ProjectEvent(nativeTestEvent())
		if err != nil || got.Event.Source != source {
			t.Fatalf("source %s: %v", source, err)
		}
	}
	for _, opts := range []NativeExecutionOptions{
		{Source: "unknown"}, {Source: ProtocolSourceRunner, RequiredCapabilities: []string{"cursor"}},
		{Source: ProtocolSourceRunner, Limits: &NativeExecutionLimits{}},
		{Source: ProtocolSourceRunner, HostReferences: NativeExecutionHostReferences{TaskID: string([]byte{255})}},
		{Source: ProtocolSourceRunner, HostReferences: NativeExecutionHostReferences{TaskID: strings.Repeat("a", 1025)}},
		{Source: ProtocolSourceRunner, HostReferences: NativeExecutionHostReferences{CitationReferences: make([]string, 17)}},
	} {
		if _, err := NewNativeExecutionProjector(opts); err == nil {
			t.Fatalf("accepted invalid options %#v", opts)
		}
	}
	var refs NativeExecutionHostReferences
	for _, raw := range []string{`{"run_id":"overwrite"}`, `{"task_id":1}`, `null`, `{"task_id":null}`, `{"citation_references":null}`, `{"citation_references":[null]}`} {
		if json.Unmarshal([]byte(raw), &refs) == nil {
			t.Fatal("accepted invalid references")
		}
	}
	p, _ := NewNativeExecutionProjector(NativeExecutionOptions{Source: ProtocolSourceRunner, SessionID: "expected", ExpectedRunID: "run-native"})
	for _, mutate := range []func(*Event){
		func(e *Event) { e.RunID = "other" }, func(e *Event) { e.Payload["session_id"] = "other" },
		func(e *Event) { e.Time = time.Time{} }, func(e *Event) { e.Payload["invalid"] = make(chan int) },
		func(e *Event) { e.Payload["invalid"] = string([]byte{255}) }, func(e *Event) { e.Payload["huge"] = strings.Repeat("x", 65536) },
		func(e *Event) { e.Payload["cycle"] = e.Payload },
	} {
		ev := nativeTestEvent()
		mutate(&ev)
		if _, err := p.ProjectEvent(ev); err == nil {
			t.Fatal("accepted invalid event")
		}
	}
}
func TestNativeExecutionExactLimits(t *testing.T) {
	opts := NativeExecutionOptions{Source: ProtocolSourceRunner, HostReferences: NativeExecutionHostReferences{TaskID: strings.Repeat("a", 1024)}}
	p, err := NewNativeExecutionProjector(opts)
	if err != nil {
		t.Fatal(err)
	}
	event := nativeTestEvent()
	first, err := p.ProjectEvent(event)
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(first)
	limits := DefaultNativeExecutionLimits()
	limits.EventBytes = len(raw)
	opts.Limits = &limits
	p, err = NewNativeExecutionProjector(opts)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = p.ProjectEvent(event); err != nil {
		t.Fatal("exact limit", err)
	}
	limits.EventBytes--
	p, err = NewNativeExecutionProjector(opts)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = p.ProjectEvent(event); NativeExecutionReasonOf(err) != NativeExecutionReasonLimit {
		t.Fatal("over limit", err)
	}
	refsRaw, _ := json.Marshal(opts.HostReferences)
	limits = DefaultNativeExecutionLimits()
	limits.HostReferenceBytes = len(refsRaw)
	opts.Limits = &limits
	if _, err = NewNativeExecutionProjector(opts); err != nil {
		t.Fatal("exact reference limit", err)
	}
	limits.HostReferenceBytes--
	if _, err = NewNativeExecutionProjector(opts); NativeExecutionReasonOf(err) != NativeExecutionReasonLimit {
		t.Fatal("reference limit", err)
	}
}
func TestNativeExecutionRealtimeBinding(t *testing.T) {
	sub := EventStreamSubscription{Version: DurableEventStreamBindingVersionV1, SubscriptionID: "sub", Source: ProtocolSourceRealtime, SessionID: "session", RunID: "run-native", StartMode: EventStreamStartAfterCursor, Cursor: EventStreamCursor{Value: "cursor", Sequence: 1}, MaxBatchSize: 4, DeliveryPolicy: EventStreamDeliveryPolicyReject}
	out := EventStreamBindingOutcome{SubscriptionID: "sub", Phase: EventStreamBindingPhaseLive, ReasonCode: "source.live", LastSequence: 2}
	ev := RealtimeEventEnvelope{EventID: "event", SessionID: "session", RunID: "run-native", Seq: 2, Type: RealtimeEventTypeComplete, TS: time.Unix(100, 0), Payload: map[string]any{"state": "completed"}}
	normalized, err := ProjectEventStreamBinding(sub, out, []RealtimeEventEnvelope{ev}, []RealtimeEventEnvelope{ev})
	if err != nil {
		t.Fatal(err)
	}
	p, err := NewNativeExecutionProjector(NativeExecutionOptions{Source: ProtocolSourceRealtime})
	if err != nil {
		t.Fatal(err)
	}
	got, err := p.ProjectRealtimeBinding(normalized)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Events) != 1 || got.Events[0].Event.Sequence != 2 || !got.Events[0].Capabilities.Cursor || !reflect.DeepEqual(got.Outcome, normalized.Outcome) {
		t.Fatal("binding facts changed")
	}
	normalized.Events[0].Seq = 4
	if _, err = p.ProjectRealtimeBinding(normalized); err == nil {
		t.Fatal("gap accepted")
	}
	normalized.Events = nil
	normalized.Outcome.Phase = EventStreamBindingPhaseExpired
	if got, err = p.ProjectRealtimeBinding(normalized); err != nil || got.Outcome.Phase != EventStreamBindingPhaseExpired {
		t.Fatal("expired disguised", err)
	}
	normalized.Subscription.Source = ProtocolSourceRunner
	if _, err = p.ProjectRealtimeBinding(normalized); err == nil {
		t.Fatal("nonRealtime accepted")
	}
}

func TestNativeExecutionTerminalUsesNativeOutcome(t *testing.T) {
	p, _ := NewNativeExecutionProjector(NativeExecutionOptions{Source: ProtocolSourceRunner})
	for _, payload := range []map[string]any{
		{"terminal_state": "completed", "status": "success"},
		{"terminal_outcome": &TerminalOutcome{State: RunStateFailed}},
		{"terminal_state": "canceled"},
	} {
		ev := nativeTestEvent()
		ev.Payload = payload
		got, err := p.ProjectEvent(ev)
		if err != nil || !got.IsTerminal() {
			t.Fatalf("native terminal lost %#v %v", got, err)
		}
	}
}

func TestNativeExecutionTerminalUsesMailboxHistoryOutcome(t *testing.T) {
	source := SourceRef{Kind: HistorySourceMailbox, Namespace: "mailbox.history.v1", ID: "mailbox"}
	for _, outcome := range []string{"acked", "dead_letter", "expired"} {
		event := NativeExecutionEvent{
			History:    &NativeHistoryIdentity{Source: source, Sequence: 1, EventID: "event", TerminalOutcome: outcome},
			NativeType: "mailbox." + outcome,
		}
		if !event.IsTerminal() || event.SourceState() != outcome {
			t.Errorf("mailbox terminal outcome %q was not recognized: state=%q", outcome, event.SourceState())
		}
	}
	for _, event := range []NativeExecutionEvent{
		{History: &NativeHistoryIdentity{Source: source, Sequence: 1, EventID: "event"}, NativeType: "mailbox.consume"},
		{History: &NativeHistoryIdentity{Source: source, Sequence: 1, EventID: "event", TerminalOutcome: "business_committed"}, NativeType: "mailbox.ack"},
		{History: &NativeHistoryIdentity{Source: SourceRef{Kind: HistorySourceRunner, Namespace: "runner", ID: "run"}, Sequence: 1, EventID: "event", TerminalOutcome: "acked"}, NativeType: "runner.event"},
	} {
		if event.IsTerminal() {
			t.Errorf("non-mailbox/unknown lifecycle fact classified terminal: %+v", event)
		}
	}
}

func BenchmarkNativeExecutionProjection(b *testing.B) {
	p, err := NewNativeExecutionProjector(NativeExecutionOptions{Source: ProtocolSourceRunner})
	if err != nil {
		b.Fatal(err)
	}
	ev := nativeTestEvent()
	ev.Payload["text"] = strings.Repeat("x", 4096)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := p.ProjectEvent(ev); err != nil {
			b.Fatal(err)
		}
	}
}
