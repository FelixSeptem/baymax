package integration

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/FelixSeptem/baymax/a2a"
	"github.com/FelixSeptem/baymax/core/types"
	"github.com/FelixSeptem/baymax/integration/fakes"
	"github.com/FelixSeptem/baymax/observability/event"
	"github.com/FelixSeptem/baymax/orchestration/collab"
	"github.com/FelixSeptem/baymax/orchestration/composer"
	"github.com/FelixSeptem/baymax/orchestration/invoke"
	"github.com/FelixSeptem/baymax/orchestration/scheduler"
	runtimeconfig "github.com/FelixSeptem/baymax/runtime/config"
)

type retrySyncClient struct {
	attempts int
	errText  string
}

func (c *retrySyncClient) Submit(_ context.Context, req a2a.TaskRequest) (a2a.TaskRecord, error) {
	return a2a.TaskRecord{TaskID: req.TaskID, Status: a2a.StatusSubmitted}, nil
}

func (c *retrySyncClient) WaitResult(
	_ context.Context,
	taskID string,
	_ time.Duration,
	_ func(context.Context, a2a.TaskRecord) error,
) (a2a.TaskRecord, error) {
	c.attempts++
	if strings.TrimSpace(c.errText) != "" {
		return a2a.TaskRecord{}, errors.New(c.errText)
	}
	if c.attempts == 1 {
		return a2a.TaskRecord{}, errors.New("connection reset by peer")
	}
	return a2a.TaskRecord{TaskID: taskID, Status: a2a.StatusSucceeded, Result: map[string]any{"ok": true}}, nil
}

func TestCollaborationRetryContractDisabledDefaultNoRetry(t *testing.T) {
	client := &retrySyncClient{}
	out, err := collab.DelegateSync(context.Background(), client, invoke.Request{TaskID: "a33-default"})
	if err == nil {
		t.Fatal("default-disabled retry should not swallow first transport failure")
	}
	if out.Retryable != true {
		t.Fatalf("retryable marker should preserve transport classification, got %#v", out)
	}
	if client.attempts != 1 {
		t.Fatalf("primitive retry disabled should execute once, attempts=%d", client.attempts)
	}
}

func TestCollaborationRetryContractEnabledConvergenceAndTransportOnly(t *testing.T) {
	client := &retrySyncClient{}
	events := make([]collab.RetryEvent, 0, 2)
	out, err := collab.DelegateSyncWithRetry(context.Background(), client, invoke.Request{TaskID: "a33-enabled"}, collab.RetryConfig{
		Enabled:        true,
		MaxAttempts:    3,
		BackoffInitial: time.Millisecond,
		BackoffMax:     2 * time.Millisecond,
		Multiplier:     2,
		JitterRatio:    0,
		RetryOn:        collab.RetryOnTransportOnly,
	}, func(ev collab.RetryEvent) {
		events = append(events, ev)
	})
	if err != nil {
		t.Fatalf("retry-enabled sync delegation should converge: %v", err)
	}
	if out.Status != collab.StatusSucceeded {
		t.Fatalf("final status=%q, want succeeded", out.Status)
	}
	if client.attempts != 2 {
		t.Fatalf("transport retry should execute twice, attempts=%d", client.attempts)
	}
	if out.Payload["collab_retry_attempts"] != 1 {
		t.Fatalf("collab_retry_attempts=%v, want 1", out.Payload["collab_retry_attempts"])
	}
	if len(events) != 2 || events[0].Type != collab.RetryEventAttempt || events[1].Type != collab.RetryEventSuccess {
		t.Fatalf("retry timeline markers mismatch: %#v", events)
	}

	protocolClient := &retrySyncClient{errText: "invalid request payload"}
	_, err = collab.DelegateSyncWithRetry(context.Background(), protocolClient, invoke.Request{TaskID: "a33-transport-only"}, collab.RetryConfig{
		Enabled:        true,
		MaxAttempts:    3,
		BackoffInitial: time.Millisecond,
		BackoffMax:     2 * time.Millisecond,
		Multiplier:     2,
		JitterRatio:    0,
		RetryOn:        collab.RetryOnTransportOnly,
	}, nil)
	if err == nil {
		t.Fatal("protocol/semantic failure should not be retried by default")
	}
	if protocolClient.attempts != 1 {
		t.Fatalf("transport_only should skip protocol retries, attempts=%d", protocolClient.attempts)
	}
}

func TestCollaborationRetryContractSchedulerNoDoubleRetry(t *testing.T) {
	s, err := scheduler.New(scheduler.NewMemoryStore(), scheduler.WithLeaseTimeout(500*time.Millisecond))
	if err != nil {
		t.Fatalf("new scheduler: %v", err)
	}
	ctx := context.Background()
	if _, err := s.Enqueue(ctx, scheduler.Task{
		TaskID:      "task-a33-single-owner",
		RunID:       "run-a33-single-owner",
		MaxAttempts: 2,
		AgentID:     "agent-a33",
		PeerID:      "peer-a33",
	}); err != nil {
		t.Fatalf("enqueue: %v", err)
	}
	client := &sequencedA2AClient{}

	claimed1, ok, err := s.Claim(ctx, "worker-a")
	if err != nil || !ok {
		t.Fatalf("claim #1 failed: ok=%v err=%v", ok, err)
	}
	exec1, err := scheduler.ExecuteClaimWithA2A(ctx, client, claimed1, 5*time.Millisecond)
	if err == nil {
		t.Fatal("first execution should fail")
	}
	if _, err := s.Requeue(ctx, exec1.Commit.TaskID, "transport_retryable"); err != nil {
		t.Fatalf("requeue #1: %v", err)
	}

	var claimed2 scheduler.ClaimedTask
	ok = false
	deadline := time.Now().Add(500 * time.Millisecond)
	for time.Now().Before(deadline) {
		claimed2, ok, err = s.Claim(ctx, "worker-b")
		if err != nil {
			t.Fatalf("claim #2 error: %v", err)
		}
		if ok {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if !ok {
		t.Fatal("claim #2 timed out")
	}
	exec2, err := scheduler.ExecuteClaimWithA2A(ctx, client, claimed2, 5*time.Millisecond)
	if err != nil {
		t.Fatalf("second execution should succeed: %v", err)
	}
	if _, err := s.Complete(ctx, exec2.Commit); err != nil {
		t.Fatalf("complete #2: %v", err)
	}
	if client.submitCount != 2 {
		t.Fatalf("scheduler-managed path should keep single retry owner, submit_count=%d want 2", client.submitCount)
	}
}

func TestCollaborationRetryContractRunStreamAndReplayIdempotent(t *testing.T) {
	cfgPath := filepath.Join(t.TempDir(), "runtime-a33.yaml")
	cfg := strings.Join([]string{
		"composer:",
		"  collab:",
		"    enabled: true",
		"    retry:",
		"      enabled: true",
		"      max_attempts: 3",
		"      backoff_initial: 100ms",
		"      backoff_max: 2s",
		"      multiplier: 2",
		"      jitter_ratio: 0.2",
		"      retry_on: transport_only",
		"",
	}, "\n")
	if err := os.WriteFile(cfgPath, []byte(cfg), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	mgr, err := runtimeconfig.NewManager(runtimeconfig.ManagerOptions{FilePath: cfgPath, EnvPrefix: "BAYMAX_A33"})
	if err != nil {
		t.Fatalf("new manager: %v", err)
	}
	defer func() { _ = mgr.Close() }()

	dispatcher := event.NewDispatcher(event.NewRuntimeRecorder(mgr))
	model := fakes.NewModel([]fakes.ModelStep{{Response: types.ModelResponse{FinalAnswer: "ok"}}})
	model.SetStream([]types.ModelEvent{{Type: types.ModelEventTypeFinalAnswer, TextDelta: "ok"}}, nil)

	comp, err := composer.NewBuilder(model).
		WithRuntimeManager(mgr).
		WithEventHandler(dispatcherHandler{dispatcher: dispatcher}).
		Build()
	if err != nil {
		t.Fatalf("build composer: %v", err)
	}

	if _, err := comp.Run(context.Background(), types.RunRequest{RunID: "run-a33-run", Input: "ok"}, nil); err != nil {
		t.Fatalf("run failed: %v", err)
	}
	if _, err := comp.Stream(context.Background(), types.RunRequest{RunID: "run-a33-stream", Input: "ok"}, nil); err != nil {
		t.Fatalf("stream failed: %v", err)
	}

	runRec := findRunRecord(t, mgr.RecentRuns(10), "run-a33-run")
	streamRec := findRunRecord(t, mgr.RecentRuns(10), "run-a33-stream")
	if runRec.CollabRetryTotal != streamRec.CollabRetryTotal ||
		runRec.CollabRetrySuccessTotal != streamRec.CollabRetrySuccessTotal ||
		runRec.CollabRetryExhaustedTotal != streamRec.CollabRetryExhaustedTotal {
		t.Fatalf("run/stream retry summary mismatch run=%#v stream=%#v", runRec, streamRec)
	}

	recorder := event.NewRuntimeRecorder(mgr)
	recEvent := types.Event{
		Version: types.EventSchemaVersionV1,
		Type:    "run.finished",
		RunID:   "run-a33-replay",
		Time:    time.Now(),
		Payload: map[string]any{
			"status":                       "failed",
			"collab_retry_total":           2,
			"collab_retry_success_total":   1,
			"collab_retry_exhausted_total": 1,
		},
	}
	recorder.OnEvent(context.Background(), recEvent)
	recorder.OnEvent(context.Background(), recEvent)
	got := findRunRecord(t, mgr.RecentRuns(20), "run-a33-replay")
	if got.CollabRetryTotal != 2 || got.CollabRetrySuccessTotal != 1 || got.CollabRetryExhaustedTotal != 1 {
		t.Fatalf("replay-idempotent retry aggregates mismatch: %#v", got)
	}
}
