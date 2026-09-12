package host

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/FelixSeptem/baymax/core/types"
)

type fakeRunner struct {
	mu     sync.Mutex
	called bool
}

func (f *fakeRunner) Run(ctx context.Context, req types.RunRequest, h types.EventHandler) (types.RunResult, error) {
	f.mu.Lock()
	f.called = true
	f.mu.Unlock()
	h.OnEvent(ctx, types.Event{Version: "v1", Type: "run.started", RunID: req.RunID, Time: time.Now(), Payload: map[string]any{"accepted": true}})
	h.OnEvent(ctx, types.Event{Version: "v1", Type: "run.finished", RunID: req.RunID, Time: time.Now(), Payload: map[string]any{"state": "completed"}})
	outcome := &types.TerminalOutcome{RunID: req.RunID, SessionID: req.SessionID, State: types.RunStateCompleted, FailureFamily: types.FailureFamilyNone, Phase: types.ExecutionPhasePostStart}
	return types.RunResult{RunID: req.RunID, TerminalOutcome: outcome}, nil
}
func (f *fakeRunner) Stream(context.Context, types.RunRequest, types.EventHandler) (types.RunResult, error) {
	return types.RunResult{}, nil
}

func (f *fakeRunner) AdmitHostRun(ctx context.Context, req types.RunRequest, mode types.HostRunExecutionMode) (types.HostRunStartAdmission, error) {
	return types.HostRunStartAdmission{Status: types.HostAdmissionStatusAccepted, RunID: req.RunID, Execution: runnerExecution{runner: f, ctx: context.WithoutCancel(ctx), req: req, mode: mode}}, nil
}

func TestConnectionAcceptsRunStartAndEmitsAsyncEvents(t *testing.T) {
	r := &fakeRunner{}
	var mu sync.Mutex
	frames := make([]any, 0, 2)
	c := New(r).Connect(func(frame any) error { mu.Lock(); defer mu.Unlock(); frames = append(frames, frame); return nil })
	cmd := types.HostCommandEnvelope{Version: types.HostProtocolVersionV1, MessageID: "m1", Kind: types.HostCommandKindRunStart, Time: time.Now(), RequestID: "req1", SessionID: "s1", RunID: "r1", Payload: map[string]any{"input": "hello"}}
	resp, err := c.HandleCommand(context.Background(), cmd)
	if err != nil || resp.Status != types.HostAdmissionStatusAccepted {
		t.Fatalf("resp=%#v err=%v", resp, err)
	}
	time.Sleep(10 * time.Millisecond)
	mu.Lock()
	defer mu.Unlock()
	if len(frames) < 2 {
		t.Fatalf("frames=%d want >=2", len(frames))
	}
}

func TestConnectionRejectsMalformedCommandBeforeRunner(t *testing.T) {
	r := &fakeRunner{}
	c := New(r).Connect(nil)
	cmd := types.HostCommandEnvelope{Version: "unknown", MessageID: "m1", Kind: types.HostCommandKindRunStart, Time: time.Now(), RequestID: "req1", SessionID: "s1"}
	if _, err := c.HandleCommand(context.Background(), cmd); err == nil {
		t.Fatal("expected validation error")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.called {
		t.Fatal("runner called for malformed command")
	}
}
