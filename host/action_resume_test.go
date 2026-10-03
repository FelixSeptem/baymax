package host

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/FelixSeptem/baymax/core/types"
)

type dynamicHostRunner struct {
	fakeRunner
	mu        sync.Mutex
	decisions []types.DynamicActionDecision
}

func (r *dynamicHostRunner) ResumeDynamicAction(_ context.Context, decision types.DynamicActionDecision, h types.EventHandler, _ bool) (types.RunResult, error) {
	r.mu.Lock()
	r.decisions = append(r.decisions, decision)
	r.mu.Unlock()
	if h != nil {
		h.OnEvent(context.Background(), types.Event{Version: types.EventSchemaVersionV1, Type: "run.resumed", RunID: decision.RunID, Time: time.Now().UTC()})
	}
	return types.RunResult{RunID: decision.RunID, TerminalOutcome: &types.TerminalOutcome{RunID: decision.RunID, SessionID: decision.SessionID, State: types.RunStateCompleted, FailureFamily: types.FailureFamilyNone, Phase: types.ExecutionPhasePostStart}}, nil
}

func TestConnectionAdmitsDynamicActionResumeAndProjectsSourceEvent(t *testing.T) {
	runner := &dynamicHostRunner{}
	var mu sync.Mutex
	frames := make([]any, 0, 2)
	conn := New(runner).Connect(func(frame any) error {
		mu.Lock()
		defer mu.Unlock()
		frames = append(frames, frame)
		return nil
	})
	cmd := types.HostCommandEnvelope{
		Version: types.HostProtocolVersionV1, MessageID: "resume-message", Kind: types.HostCommandKindActionResume,
		Time: time.Now().UTC(), RequestID: "resume-request", SessionID: "session-1", RunID: "run-1",
		Payload: map[string]any{
			"decision": "confirm", "token": "opaque-token", "checkpoint_id": "checkpoint-1",
			"checkpoint_version": types.DynamicActionResumeProtocolVersion, "checkpoint_digest": "digest-1", "idempotency_key": "idem-1",
		},
	}
	response, err := conn.HandleCommand(context.Background(), cmd)
	if err != nil || response.Status != types.HostAdmissionStatusAccepted || response.ReasonCode != "dynamic_action.resume_admitted" {
		t.Fatalf("response=%#v err=%v", response, err)
	}
	runner.mu.Lock()
	if len(runner.decisions) != 1 || runner.decisions[0].Token != "opaque-token" {
		runner.mu.Unlock()
		t.Fatalf("decisions=%#v", runner.decisions)
	}
	runner.mu.Unlock()
	mu.Lock()
	defer mu.Unlock()
	if len(frames) < 2 {
		t.Fatalf("frames=%d, want admission and source event", len(frames))
	}
	var foundEvent bool
	for _, frame := range frames {
		if event, ok := frame.(types.HostRuntimeEventEnvelope); ok && event.Event.Kind == "progress" {
			foundEvent = true
		}
	}
	if !foundEvent {
		t.Fatalf("frames=%#v, missing source runtime event", frames)
	}
}

func TestConnectionRejectsDynamicActionResumeBeforeSourceMutation(t *testing.T) {
	runner := &dynamicHostRunner{}
	conn := New(runner).Connect(func(any) error { return nil })
	cmd := types.HostCommandEnvelope{
		Version: types.HostProtocolVersionV1, MessageID: "bad-resume-message", Kind: types.HostCommandKindActionResume,
		Time: time.Now().UTC(), RequestID: "bad-resume-request", SessionID: "session-1", RunID: "run-1",
		Payload: map[string]any{"decision": "confirm", "token": "", "checkpoint_id": "checkpoint-1", "checkpoint_version": types.DynamicActionResumeProtocolVersion, "checkpoint_digest": "digest-1", "idempotency_key": "idem-1"},
	}
	response, err := conn.HandleCommand(context.Background(), cmd)
	if response.Status != types.HostAdmissionStatusRejected {
		t.Fatalf("response=%#v err=%v", response, err)
	}
	runner.mu.Lock()
	defer runner.mu.Unlock()
	if len(runner.decisions) != 0 {
		t.Fatalf("source mutated for invalid command: %#v", runner.decisions)
	}
}
