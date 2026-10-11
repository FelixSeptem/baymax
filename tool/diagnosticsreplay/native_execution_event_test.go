package diagnosticsreplay

import (
	"encoding/json"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/FelixSeptem/baymax/core/types"
)

func replayNativeFact(kind, state string) types.NativeExecutionEvent {
	p, _ := types.NewNativeExecutionProjector(types.NativeExecutionOptions{Source: types.ProtocolSourceRunner})
	e, _ := p.ProjectEvent(types.Event{Type: kind, RunID: "fake-run", TraceID: "fake-trace", Time: time.Unix(100, 0), Payload: map[string]any{"state": state}})
	return e
}
func TestNativeExecutionReplayFixtures(t *testing.T) {
	raw, err := os.ReadFile("testdata/native_execution_event.v1.json")
	if err != nil {
		t.Fatal(err)
	}
	first, err := ReplayNativeExecutionFixtureJSON(raw)
	if err != nil {
		var cases NativeExecutionFixture
		if decodeErr := json.Unmarshal(raw, &cases); decodeErr != nil {
			t.Fatal(decodeErr)
		}
		for _, item := range cases.Cases {
			if _, caseErr := replayNativeExecutionCase(item); caseErr != nil {
				t.Errorf("case %s: %v", item.CaseID, caseErr)
			}
		}
		t.Fatal(err)
	}
	second, err := ReplayNativeExecutionFixtureJSON(raw)
	if err != nil || !reflect.DeepEqual(first, second) {
		t.Fatal("nondeterministic", err)
	}
	var fixture NativeExecutionFixture
	if err := json.Unmarshal(raw, &fixture); err != nil {
		t.Fatal(err)
	}
	fixture.Cases[0].ExpectedReason = "unexpected"
	raw, _ = json.Marshal(fixture)
	if _, err := ReplayNativeExecutionFixtureJSON(raw); err == nil {
		t.Fatal("negative fixture accepted")
	}
	for _, raw := range []string{`{}`, `{"version":"wrong","cases":[]}`, `{"version":"native_execution_event.v1","unknown":true}`, `{"version":"native_execution_event.v1","cases":[{"case_id":"bad","options":{"HostReferences":{"run_id":"overwrite"}}}]}`} {
		if _, err := ReplayNativeExecutionFixtureJSON([]byte(raw)); err == nil {
			t.Fatal("invalid fixture accepted")
		}
	}
}
func TestNativeExecutionParityDetectsDrift(t *testing.T) {
	e := replayNativeFact("run.finished", "completed")
	run := NativeExecutionTranscript{Events: []types.NativeExecutionEvent{e}, Observation: NativeExecutionDeliveryEvidence{State: "healthy", Received: 1, Delivered: 1, TerminalDelivered: 1}}
	stream := run
	stream.Events = append([]types.NativeExecutionEvent(nil), run.Events...)
	stream.Events[0].Event.Time = time.Unix(200, 0)
	stream.Events[0].Event.RunID = "generated-other"
	stream.Events[0].TraceID = "other-trace"
	stream.Events[0].Event.EventID = "other-trace"
	if err := CompareNativeExecutionParity(run, stream); err != nil {
		t.Fatal("permitted generation difference", err)
	}
	for _, change := range []func(*NativeExecutionTranscript){
		func(v *NativeExecutionTranscript) { v.Events[0] = replayNativeFact("run.finished", "failed") },
		func(v *NativeExecutionTranscript) {
			v.Events = append(v.Events, replayNativeFact("run.dynamic_action.resolved", ""))
		},
		func(v *NativeExecutionTranscript) { v.Events[0].Event.Source = types.ProtocolSourceTeams },
		func(v *NativeExecutionTranscript) {
			v.Observation.Reason = "observation.sink_error"
			v.Observation.Incomplete = true
		},
		func(v *NativeExecutionTranscript) { v.Events[0].Event.Kind = types.ProtocolEventKindError },
	} {
		bad := run
		bad.Events = append([]types.NativeExecutionEvent(nil), run.Events...)
		change(&bad)
		if err := CompareNativeExecutionParity(run, bad); err == nil {
			t.Fatal("drift normalized away")
		}
	}
	// Equality classes survive ID normalization; two callbacks sharing an ID
	// cannot be collapsed or changed into independent source identities.
	run.Events = append(run.Events, replayNativeFact("tool.dispatch.started", ""))
	run.Observation.Received = 2
	run.Observation.Delivered = 2
	bad := run
	bad.Events = append([]types.NativeExecutionEvent(nil), run.Events...)
	bad.Events[1].Event.EventID = "distinct"
	if err := CompareNativeExecutionParity(run, bad); err == nil {
		t.Fatal("identity relationship lost")
	}
}

func TestNativeExecutionParityRetainsDeltaFactsAndBusinessFields(t *testing.T) {
	e := replayNativeFact("run.finished", "completed")
	base := NativeExecutionTranscript{Events: []types.NativeExecutionEvent{e}, Observation: NativeExecutionDeliveryEvidence{State: "healthy", Received: 1, Delivered: 1, TerminalDelivered: 1}}
	for _, payload := range []map[string]any{
		{"event_type": "error", "delta": ""},
		{"event_type": "output_text_delta", "delta": "", "meta": map[string]any{"failure": true}},
		{"event_type": "clarification_request", "clarification_request": map[string]any{"request_id": "q"}},
	} {
		changed := base
		delta := replayNativeFact("model.delta", "")
		delta.Event.Payload = payload
		changed.Events = append([]types.NativeExecutionEvent{delta}, e)
		changed.Observation.Received++
		changed.Observation.Delivered++
		if err := CompareNativeExecutionParity(base, changed); err == nil {
			t.Fatal("semantic delta drift hidden")
		}
	}
	for _, key := range []string{"time", "run_id", "checkpoint_digest"} {
		left, right := base, base
		left.Events = []types.NativeExecutionEvent{e}
		right.Events = []types.NativeExecutionEvent{e}
		left.Events[0].Event.Payload = map[string]any{"business": map[string]any{key: "first"}}
		right.Events[0].Event.Payload = map[string]any{"business": map[string]any{key: "second"}}
		if err := CompareNativeExecutionParity(left, right); err == nil {
			t.Fatalf("nested business %s normalized away", key)
		}
	}
	requested := replayNativeFact("model.requested", "")
	delta := replayNativeFact("model.delta", "")
	delta.Event.Payload = map[string]any{"event_type": types.ModelEventTypeOutputTextDelta, "delta": "fragment"}
	base.Events = []types.NativeExecutionEvent{requested, e}
	base.Observation.Received, base.Observation.Delivered = 2, 2
	fragmented := base
	fragmented.Events = []types.NativeExecutionEvent{requested, delta, e}
	fragmented.Observation.Received, fragmented.Observation.Delivered = 3, 3
	if err := CompareNativeExecutionParity(base, fragmented); err != nil {
		t.Fatal("legitimate fragment rejected", err)
	}
	for _, mutate := range []func(*types.NativeExecutionEvent){
		func(v *types.NativeExecutionEvent) { v.Event.RunID = "foreign-run" },
		func(v *types.NativeExecutionEvent) { v.Event.StepID = "foreign-step" },
		func(v *types.NativeExecutionEvent) { v.HostReferences.TaskID = "foreign-task" },
		func(v *types.NativeExecutionEvent) { v.Capabilities.Cursor = true },
	} {
		bad := fragmented
		bad.Events = append([]types.NativeExecutionEvent(nil), fragmented.Events...)
		mutate(&bad.Events[1])
		if err := CompareNativeExecutionParity(base, bad); err == nil {
			t.Fatal("fragment correlation/profile drift hidden")
		}
	}
}
