package runner

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/FelixSeptem/baymax/core/types"
	runtimeconfig "github.com/FelixSeptem/baymax/runtime/config"
)

func TestRetryRunCreatesDistinctCausalRunWithoutMutatingTerminalSource(t *testing.T) {
	for _, stream := range []bool{false, true} {
		stream := stream
		mode := "run"
		if stream {
			mode = "stream"
		}
		t.Run(mode, func(t *testing.T) {
			previous := types.RunRef{
				RunID:     "run-failed-source-" + mode,
				SessionID: "session-retry-" + mode,
				State:     types.RunStateFailed,
				Source:    types.ProtocolSourceRunner,
			}
			original := previous
			var modelRunID string
			model := &fakeModel{
				generate: func(_ context.Context, req types.ModelRequest) (types.ModelResponse, error) {
					modelRunID = req.RunID
					return types.ModelResponse{FinalAnswer: "recovered"}, nil
				},
				stream: func(_ context.Context, req types.ModelRequest, onEvent func(types.ModelEvent) error) error {
					modelRunID = req.RunID
					return onEvent(types.ModelEvent{Type: types.ModelEventTypeFinalAnswer, TextDelta: "recovered"})
				},
			}
			collector := &eventCollector{}
			retryID := "run-retry-attempt-" + mode

			result, err := New(model).RetryRun(context.Background(), previous, types.RunRequest{
				RunID: retryID,
				Input: "retry",
			}, collector, stream)
			if err != nil {
				t.Fatalf("RetryRun() error = %v", err)
			}
			if result.RunID != retryID || modelRunID != result.RunID {
				t.Fatalf("retry identity result=%q model=%q", result.RunID, modelRunID)
			}
			if result.RunID == previous.RunID {
				t.Fatal("retry reused terminal source run_id")
			}
			if result.TerminalOutcome == nil || result.TerminalOutcome.CausationID != previous.RunID {
				t.Fatalf("retry terminal causation = %#v, want %q", result.TerminalOutcome, previous.RunID)
			}
			finished, ok := collector.lastNonTimelineEvent()
			if !ok || finished.Type != "run.finished" || finished.Payload["terminal_causation_id"] != previous.RunID {
				t.Fatalf("retry terminal event causation = %#v, want %q", finished, previous.RunID)
			}
			if !reflect.DeepEqual(previous, original) {
				t.Fatalf("terminal source mutated: got %#v want %#v", previous, original)
			}
		})
	}
}

func TestRetryRunRejectsInvalidSourceBeforeModelMutation(t *testing.T) {
	tests := []struct {
		name     string
		previous types.RunRef
		retryID  string
	}{
		{
			name:     "invalid source reference",
			previous: types.RunRef{State: types.RunStateFailed, Source: types.ProtocolSourceRunner},
			retryID:  "run-retry",
		},
		{
			name:     "nonterminal source",
			previous: types.RunRef{RunID: "run-working", State: types.RunStateWorking, Source: types.ProtocolSourceRunner},
			retryID:  "run-retry",
		},
		{
			name:     "same run id",
			previous: types.RunRef{RunID: "run-failed", State: types.RunStateFailed, Source: types.ProtocolSourceRunner},
			retryID:  "run-failed",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var calls atomic.Int32
			model := &fakeModel{generate: func(context.Context, types.ModelRequest) (types.ModelResponse, error) {
				calls.Add(1)
				return types.ModelResponse{FinalAnswer: "unexpected"}, nil
			}}
			result, err := New(model).RetryRun(context.Background(), tt.previous, types.RunRequest{
				RunID: tt.retryID,
				Input: "must not execute",
			}, nil, false)
			if err == nil {
				t.Fatalf("RetryRun() result = %#v, want rejection", result)
			}
			if got := calls.Load(); got != 0 {
				t.Fatalf("model mutations = %d, want 0", got)
			}
		})
	}
}

func TestActiveRunControlIsAdditiveForPlainRunAndStream(t *testing.T) {
	for _, stream := range []bool{false, true} {
		mode := "run"
		if stream {
			mode = "stream"
		}
		t.Run(mode, func(t *testing.T) {
			e := New(nil)
			e.activeRunLimit = 1

			ctrl, gotCtx, err := e.activateActiveRun(context.Background(), "plain-"+mode, "session-plain", stream)
			if err != nil {
				t.Fatalf("plain %s activation: %v", mode, err)
			}
			if ctrl != nil {
				t.Fatalf("plain %s unexpectedly exposed host control", mode)
			}
			if gotCtx == nil {
				t.Fatalf("plain %s returned nil context", mode)
			}
			if got := len(e.ActiveRuns()); got != 0 {
				t.Fatalf("plain %s active controls=%d, want 0", mode, got)
			}
		})
	}
}

func TestActiveRunControlCanBeExplicitlyEnabled(t *testing.T) {
	e := New(nil, WithActiveRunControlLimit(1))
	ctrl, _, err := e.activateActiveRun(context.Background(), "controlled", "session-controlled", false)
	if err != nil {
		t.Fatal(err)
	}
	if ctrl == nil {
		t.Fatal("explicit control option did not expose active control")
	}
	e.finishActiveRun(ctrl)
}

func TestPlainRunAndStreamAreNotSubjectToHostControlLimit(t *testing.T) {
	for _, stream := range []bool{false, true} {
		mode := "run"
		if stream {
			mode = "stream"
		}
		t.Run(mode, func(t *testing.T) {
			started := make(chan struct{}, 2)
			release := make(chan struct{})
			model := &fakeModel{
				generate: func(context.Context, types.ModelRequest) (types.ModelResponse, error) {
					started <- struct{}{}
					<-release
					return types.ModelResponse{FinalAnswer: "ok"}, nil
				},
				stream: func(_ context.Context, _ types.ModelRequest, onEvent func(types.ModelEvent) error) error {
					started <- struct{}{}
					<-release
					return onEvent(types.ModelEvent{Type: types.ModelEventTypeOutputTextDelta, TextDelta: "ok"})
				},
			}
			e := New(model)
			e.activeRunLimit = 1
			var wg sync.WaitGroup
			errs := make(chan error, 2)
			for i := 0; i < 2; i++ {
				i := i
				wg.Add(1)
				go func() {
					defer wg.Done()
					req := types.RunRequest{RunID: fmt.Sprintf("plain-%s-%d", mode, i), Input: "x"}
					var err error
					if stream {
						_, err = e.Stream(context.Background(), req, nil)
					} else {
						_, err = e.Run(context.Background(), req, nil)
					}
					errs <- err
				}()
			}
			<-started
			<-started
			if got := len(e.ActiveRuns()); got != 0 {
				t.Fatalf("plain %s exposed %d host controls", mode, got)
			}
			close(release)
			wg.Wait()
			close(errs)
			for err := range errs {
				if err != nil {
					t.Fatalf("plain %s failed under host control limit: %v", mode, err)
				}
			}
		})
	}
}

func TestActiveRunRegistryDuplicateCleanupAndIsolation(t *testing.T) {
	e1 := New(nil)
	e2 := New(nil)
	ctrl, ctx, err := e1.beginActiveRun(context.Background(), "run-1", "sess-1", false)
	if err != nil || ctrl == nil || ctx == nil {
		t.Fatalf("beginActiveRun: ctrl=%v ctx=%v err=%v", ctrl, ctx, err)
	}
	if _, _, err := e1.beginActiveRun(context.Background(), "run-1", "sess-1", false); err != ErrActiveRunDuplicate {
		t.Fatalf("duplicate error=%v, want %v", err, ErrActiveRunDuplicate)
	}
	if _, ok := e2.ActiveRun("run-1"); ok {
		t.Fatal("active runs leaked across engines")
	}
	if len(e1.ActiveRuns()) != 1 {
		t.Fatalf("active snapshot count=%d, want 1", len(e1.ActiveRuns()))
	}
	e1.finishActiveRun(ctrl)
	if _, ok := e1.ActiveRun("run-1"); ok {
		t.Fatal("run still active after cleanup")
	}
}

func TestActiveRunCancelAndRealtimeIngressAdmission(t *testing.T) {
	e := New(nil, WithActiveRunIngressBuffer(1))
	ctrl, _, err := e.beginActiveRun(context.Background(), "run-c", "sess-c", false)
	if err != nil {
		t.Fatal(err)
	}
	defer e.finishActiveRun(ctrl)
	ctrl.attachRealtime(newControlTestRealtimeRuntime(e, ctrl.runID, ctrl.sessionID))
	status, err := e.CancelRun("run-c", "sess-c")
	if err != nil || status != types.HostAdmissionStatusAccepted {
		t.Fatalf("cancel status=%v err=%v", status, err)
	}
	status, err = e.CancelRun("run-c", "sess-c")
	if err != nil || status != types.HostAdmissionStatusDuplicate {
		t.Fatalf("duplicate cancel status=%v err=%v", status, err)
	}
	ev := types.RealtimeEventEnvelope{EventID: "e1", SessionID: "sess-c", RunID: "run-c", Seq: 1, Type: types.RealtimeEventTypeInterrupt, TS: time.Now().UTC(), Payload: map[string]any{}}
	status, err = e.IngestRealtime("run-c", "sess-c", ev)
	if err != nil || status != types.HostAdmissionStatusAccepted {
		t.Fatalf("ingress status=%v err=%v", status, err)
	}
	status, err = e.IngestRealtime("run-c", "sess-c", ev)
	if err != nil || status != types.HostAdmissionStatusDuplicate {
		t.Fatalf("duplicate ingress status=%v err=%v", status, err)
	}
	if _, err = e.IngestRealtime("run-c", "other", ev); err == nil {
		t.Fatal("session mismatch accepted")
	}
}

func TestIngestRealtimeRejectsSourceStateBeforeAdmission(t *testing.T) {
	e := New(nil, WithActiveRunIngressBuffer(4))
	ctrl, _, err := e.beginActiveRun(context.Background(), "run-realtime", "session-realtime", false)
	if err != nil {
		t.Fatal(err)
	}
	defer e.finishActiveRun(ctrl)
	runtime := newControlTestRealtimeRuntime(e, ctrl.runID, ctrl.sessionID)
	ctrl.attachRealtime(runtime)

	interrupt := controlTestRealtimeEvent("interrupt", ctrl, 1, types.RealtimeEventTypeInterrupt, nil)
	status, err := e.IngestRealtime(ctrl.runID, ctrl.sessionID, interrupt)
	if err != nil || status != types.HostAdmissionStatusAccepted {
		t.Fatalf("interrupt admission status=%v err=%v", status, err)
	}
	if err := ctrl.drainRealtime(context.Background(), &eventCollector{}, 0); err != nil {
		t.Fatalf("drain interrupt: %v", err)
	}

	tests := []struct {
		name  string
		event types.RealtimeEventEnvelope
		code  string
	}{
		{
			name:  "unsupported_control_type",
			event: controlTestRealtimeEvent("delta", ctrl, 2, types.RealtimeEventTypeDelta, map[string]any{"delta": "not host control"}),
			code:  realtimeReasonUnsupportedEventType,
		},
		{
			name:  "sequence_gap",
			event: controlTestRealtimeEvent("gap", ctrl, 3, types.RealtimeEventTypeResume, map[string]any{"resume_cursor": runtime.cursor}),
			code:  realtimeReasonSequenceGap,
		},
		{
			name:  "invalid_cursor",
			event: controlTestRealtimeEvent("bad-cursor", ctrl, 2, types.RealtimeEventTypeResume, map[string]any{"resume_cursor": "wrong"}),
			code:  realtimeReasonInvalidResumeCursor,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status, err := e.IngestRealtime(ctrl.runID, ctrl.sessionID, tt.event)
			if status != types.HostAdmissionStatusRejected || err == nil {
				t.Fatalf("status=%v err=%v, want rejected source validation", status, err)
			}
			var protocolErr *realtimeProtocolError
			if !errors.As(err, &protocolErr) || protocolErr.Code != tt.code {
				t.Fatalf("error=%v, want realtime code %q", err, tt.code)
			}
			if got := len(ctrl.ingress); got != 0 {
				t.Fatalf("invalid event queued: len=%d", got)
			}
			if err := ctrl.drainRealtime(context.Background(), &eventCollector{}, 0); err != nil {
				t.Fatalf("invalid admission later aborted source: %v", err)
			}
			if err := ctrl.ctx.Err(); err != nil {
				t.Fatalf("invalid admission canceled business run: %v", err)
			}
		})
	}
}

func TestHostCancelRunStreamAdmissionAndTerminalParity(t *testing.T) {
	type outcome struct {
		firstStatus   types.HostAdmissionStatus
		duplicate     types.HostAdmissionStatus
		terminalState types.RunState
		errorClass    types.ErrorClass
	}
	outcomes := make(map[types.HostRunExecutionMode]outcome, 2)
	for _, mode := range []types.HostRunExecutionMode{types.HostRunExecutionModeRun, types.HostRunExecutionModeStream} {
		mode := mode
		t.Run(string(mode), func(t *testing.T) {
			started := make(chan struct{})
			model := &fakeModel{
				generate: func(ctx context.Context, _ types.ModelRequest) (types.ModelResponse, error) {
					close(started)
					<-ctx.Done()
					return types.ModelResponse{}, ctx.Err()
				},
				stream: func(ctx context.Context, _ types.ModelRequest, _ func(types.ModelEvent) error) error {
					close(started)
					<-ctx.Done()
					return ctx.Err()
				},
			}
			e := New(model)
			runID := "run-cancel-" + string(mode)
			sessionID := "session-cancel-" + string(mode)
			admission, err := e.AdmitHostRun(context.Background(), types.RunRequest{
				RunID: runID, SessionID: sessionID, Input: "cancel me",
			}, mode)
			if err != nil || admission.Status != types.HostAdmissionStatusAccepted {
				t.Fatalf("AdmitHostRun() status=%v err=%v", admission.Status, err)
			}
			resultCh := make(chan types.RunResult, 1)
			errCh := make(chan error, 1)
			go func() {
				result, runErr := admission.Execution.Execute(nil)
				resultCh <- result
				errCh <- runErr
			}()
			<-started
			firstStatus, err := e.CancelRun(runID, sessionID)
			if err != nil || firstStatus != types.HostAdmissionStatusAccepted {
				t.Fatalf("first cancel status=%v err=%v", firstStatus, err)
			}
			duplicateStatus, err := e.CancelRun(runID, sessionID)
			if err != nil || duplicateStatus != types.HostAdmissionStatusDuplicate {
				t.Fatalf("duplicate cancel status=%v err=%v", duplicateStatus, err)
			}
			result := <-resultCh
			if runErr := <-errCh; !errors.Is(runErr, context.Canceled) {
				t.Fatalf("Execute() error=%v, want context canceled", runErr)
			}
			if result.TerminalOutcome == nil || result.Error == nil {
				t.Fatalf("cancel result missing terminal classification: %#v", result)
			}
			outcomes[mode] = outcome{
				firstStatus: firstStatus, duplicate: duplicateStatus,
				terminalState: result.TerminalOutcome.State, errorClass: result.Error.Class,
			}
		})
	}
	if run, stream := outcomes[types.HostRunExecutionModeRun], outcomes[types.HostRunExecutionModeStream]; run != stream {
		t.Fatalf("cancel parity drift run=%#v stream=%#v", run, stream)
	}
}

func TestIngestRealtimeRejectsResumeOutsideInputRequired(t *testing.T) {
	e := New(nil)
	ctrl, _, err := e.beginActiveRun(context.Background(), "run-working", "session-working", false)
	if err != nil {
		t.Fatal(err)
	}
	defer e.finishActiveRun(ctrl)
	runtime := newControlTestRealtimeRuntime(e, ctrl.runID, ctrl.sessionID)
	runtime.cursor = "session-working:run-working:0"
	ctrl.attachRealtime(runtime)

	resume := controlTestRealtimeEvent("resume-working", ctrl, 1, types.RealtimeEventTypeResume, map[string]any{"resume_cursor": runtime.cursor})
	status, err := e.IngestRealtime(ctrl.runID, ctrl.sessionID, resume)
	if status != types.HostAdmissionStatusRejected || err == nil {
		t.Fatalf("resume outside input_required status=%v err=%v", status, err)
	}
	if got := len(ctrl.ingress); got != 0 {
		t.Fatalf("invalid lifecycle event queued: len=%d", got)
	}
}

func TestIngestRealtimeRejectsUnknownInactiveAndSessionMismatchWithoutMutation(t *testing.T) {
	e := New(nil, WithActiveRunIngressBuffer(2))
	unknown := types.RealtimeEventEnvelope{
		EventID: "unknown-interrupt", SessionID: "session-unknown", RunID: "run-unknown",
		Seq: 1, Type: types.RealtimeEventTypeInterrupt, TS: time.Now().UTC(), Payload: map[string]any{},
	}
	status, err := e.IngestRealtime(unknown.RunID, unknown.SessionID, unknown)
	if status != types.HostAdmissionStatusRejected || !errors.Is(err, ErrActiveRunUnknown) {
		t.Fatalf("unknown ingress status=%v err=%v", status, err)
	}
	if got := len(e.realtimeCursors); got != 0 {
		t.Fatalf("unknown ingress created %d source cursors", got)
	}

	ctrl, _, err := e.beginActiveRun(context.Background(), "run-correlated", "session-correlated", false)
	if err != nil {
		t.Fatal(err)
	}
	runtime := newControlTestRealtimeRuntime(e, ctrl.runID, ctrl.sessionID)
	ctrl.attachRealtime(runtime)
	interrupt := controlTestRealtimeEvent("mismatched-interrupt", ctrl, 1, types.RealtimeEventTypeInterrupt, nil)
	status, err = e.IngestRealtime(ctrl.runID, "session-other", interrupt)
	if status != types.HostAdmissionStatusRejected || err == nil {
		t.Fatalf("session mismatch status=%v err=%v", status, err)
	}
	assertRealtimeControlUnmutated(t, ctrl, runtime)

	e.finishActiveRun(ctrl)
	status, err = e.IngestRealtime(ctrl.runID, ctrl.sessionID, interrupt)
	if status != types.HostAdmissionStatusRejected || !errors.Is(err, ErrActiveRunUnknown) {
		t.Fatalf("inactive ingress status=%v err=%v", status, err)
	}
	assertRealtimeControlUnmutated(t, ctrl, runtime)
}

func TestIngestRealtimeRejectsFullAndClosedIngressWithoutMutation(t *testing.T) {
	e := New(nil, WithActiveRunIngressBuffer(1))
	ctrl, _, err := e.beginActiveRun(context.Background(), "run-boundary", "session-boundary", false)
	if err != nil {
		t.Fatal(err)
	}
	defer e.finishActiveRun(ctrl)
	runtime := newControlTestRealtimeRuntime(e, ctrl.runID, ctrl.sessionID)
	ctrl.attachRealtime(runtime)

	first := controlTestRealtimeEvent("interrupt-first", ctrl, 1, types.RealtimeEventTypeInterrupt, map[string]any{"dedup_key": "interrupt-first"})
	status, err := e.IngestRealtime(ctrl.runID, ctrl.sessionID, first)
	if status != types.HostAdmissionStatusAccepted || err != nil {
		t.Fatalf("first ingress status=%v err=%v", status, err)
	}
	second := controlTestRealtimeEvent("interrupt-second", ctrl, 2, types.RealtimeEventTypeInterrupt, map[string]any{"dedup_key": "interrupt-second"})
	status, err = e.IngestRealtime(ctrl.runID, ctrl.sessionID, second)
	if status != types.HostAdmissionStatusRejected || !errors.Is(err, ErrActiveRunBackpressure) {
		t.Fatalf("full ingress status=%v err=%v", status, err)
	}
	if runtime.interrupted || runtime.lastInSeq != 0 || len(runtime.seenDedup) != 0 {
		t.Fatalf("full ingress mutated source runtime: interrupted=%v seq=%d dedupe=%d", runtime.interrupted, runtime.lastInSeq, len(runtime.seenDedup))
	}
	if _, reserved := runtime.pendingSeq[second.Seq]; reserved {
		t.Fatal("full ingress leaked rejected sequence reservation")
	}

	ctrl.mu.Lock()
	ctrl.closedFlag = true
	ctrl.mu.Unlock()
	status, err = e.IngestRealtime(ctrl.runID, ctrl.sessionID, second)
	if status != types.HostAdmissionStatusRejected || !errors.Is(err, ErrActiveRunClosed) {
		t.Fatalf("closed ingress status=%v err=%v", status, err)
	}
	if got := len(ctrl.ingress); got != 1 {
		t.Fatalf("closed ingress queue length=%d, want original entry only", got)
	}
	if _, reserved := runtime.pendingSeq[second.Seq]; reserved {
		t.Fatal("closed ingress reserved rejected sequence")
	}
}

func TestHostControlledRealtimeRunStreamSafePointParity(t *testing.T) {
	for _, mode := range []types.HostRunExecutionMode{types.HostRunExecutionModeRun, types.HostRunExecutionModeStream} {
		mode := mode
		t.Run(string(mode), func(t *testing.T) {
			modelStarted := make(chan struct{})
			releaseModel := make(chan struct{})
			model := &fakeModel{
				generate: func(context.Context, types.ModelRequest) (types.ModelResponse, error) {
					close(modelStarted)
					<-releaseModel
					return types.ModelResponse{FinalAnswer: "done"}, nil
				},
				stream: func(_ context.Context, _ types.ModelRequest, onEvent func(types.ModelEvent) error) error {
					close(modelStarted)
					<-releaseModel
					return onEvent(types.ModelEvent{Type: types.ModelEventTypeFinalAnswer, TextDelta: "done"})
				},
			}
			e := New(model,
				WithRuntimeManager(newRealtimeRuntimeManagerForTest(t, "BAYMAX_RUNTIME_ACTIVE_INGRESS_PARITY")),
				WithActiveRunIngressBuffer(2),
			)
			runID := "run-active-" + string(mode)
			sessionID := "session-active-" + string(mode)
			admission, err := e.AdmitHostRun(context.Background(), types.RunRequest{
				RunID: runID, SessionID: sessionID, Input: "control me",
			}, mode)
			if err != nil || admission.Status != types.HostAdmissionStatusAccepted {
				t.Fatalf("AdmitHostRun() status=%v err=%v", admission.Status, err)
			}
			resultCh := make(chan types.RunResult, 1)
			errCh := make(chan error, 1)
			collector := &eventCollector{}
			go func() {
				result, runErr := admission.Execution.Execute(collector)
				resultCh <- result
				errCh <- runErr
			}()
			<-modelStarted

			interrupt := types.RealtimeEventEnvelope{
				EventID: "interrupt-" + string(mode), SessionID: sessionID, RunID: runID,
				Seq: 1, Type: types.RealtimeEventTypeInterrupt, TS: time.Now().UTC(), Payload: map[string]any{},
			}
			status, err := e.IngestRealtime(runID, sessionID, interrupt)
			if err != nil || status != types.HostAdmissionStatusAccepted {
				t.Fatalf("interrupt status=%v err=%v", status, err)
			}
			close(releaseModel)

			ctrl, ok := e.ActiveRun(runID)
			if !ok {
				t.Fatal("source run completed before admitting queued interrupt at safe point")
			}
			deadline := time.Now().Add(time.Second)
			for ctrl.Snapshot().State != types.RunStateInputRequired && time.Now().Before(deadline) {
				time.Sleep(time.Millisecond)
			}
			if got := ctrl.Snapshot().State; got != types.RunStateInputRequired {
				t.Fatalf("state after interrupt=%q, want input_required", got)
			}

			resume := types.RealtimeEventEnvelope{
				EventID: "resume-" + string(mode), SessionID: sessionID, RunID: runID,
				Seq: 2, Type: types.RealtimeEventTypeResume, TS: time.Now().UTC(),
				Payload: map[string]any{"resume_cursor": sessionID + ":" + runID + ":1"},
			}
			status, err = e.IngestRealtime(runID, sessionID, resume)
			if err != nil || status != types.HostAdmissionStatusAccepted {
				t.Fatalf("resume status=%v err=%v", status, err)
			}
			select {
			case result := <-resultCh:
				if runErr := <-errCh; runErr != nil {
					t.Fatalf("Execute() error = %v", runErr)
				}
				if result.FinalAnswer != "done" || result.TerminalOutcome == nil || result.TerminalOutcome.State != types.RunStateCompleted {
					t.Fatalf("result after resume = %#v", result)
				}
				payload := lastRunFinishedPayloadFromCollector(t, collector)
				if payload["realtime_interrupt_total"] != 1 || payload["realtime_resume_total"] != 1 {
					t.Fatalf("realtime counters after resume = %#v", payload)
				}
			case <-time.After(time.Second):
				t.Fatal("execution did not complete after valid resume")
			}
		})
	}
}

func assertRealtimeControlUnmutated(t *testing.T, ctrl *ActiveRunControl, runtime *realtimeSessionRuntime) {
	t.Helper()
	if got := len(ctrl.ingress); got != 0 {
		t.Fatalf("rejected ingress queued %d events", got)
	}
	if runtime.interrupted || runtime.lastInSeq != 0 || runtime.seqMax != 0 || len(runtime.seenDedup) != 0 || len(runtime.pendingSeq) != 0 {
		t.Fatalf("rejected ingress mutated source runtime: interrupted=%v last=%d max=%d seen=%d pending=%d",
			runtime.interrupted, runtime.lastInSeq, runtime.seqMax, len(runtime.seenDedup), len(runtime.pendingSeq))
	}
	if got := len(ctrl.engine.realtimeCursors); got != 0 {
		t.Fatalf("rejected ingress created %d cursors", got)
	}
}

func TestCancelRunRechecksClosedStateAtAdmissionLinearization(t *testing.T) {
	e := New(nil)
	ctrl, _, err := e.beginActiveRun(context.Background(), "run-closing", "session-closing", false)
	if err != nil {
		t.Fatal(err)
	}
	defer e.finishActiveRun(ctrl)

	// Model the finish path after it has won the control linearization point but
	// before registry removal/channel close completes.
	ctrl.mu.Lock()
	ctrl.closedFlag = true
	ctrl.mu.Unlock()

	status, err := e.CancelRun(ctrl.runID, ctrl.sessionID)
	if status == types.HostAdmissionStatusAccepted || !errors.Is(err, ErrActiveRunClosed) {
		t.Fatalf("cancel after source close status=%v err=%v", status, err)
	}
}

func TestRealtimeQueuedDedupeIsReleasedAfterSourceConsumption(t *testing.T) {
	e := New(nil, WithActiveRunIngressBuffer(2))
	ctrl, _, err := e.beginActiveRun(context.Background(), "run-dedupe", "session-dedupe", false)
	if err != nil {
		t.Fatal(err)
	}
	defer e.finishActiveRun(ctrl)
	runtime := newControlTestRealtimeRuntime(e, ctrl.runID, ctrl.sessionID)
	ctrl.attachRealtime(runtime)

	ev := controlTestRealtimeEvent("dedupe-event", ctrl, 1, types.RealtimeEventTypeInterrupt, map[string]any{"dedup_key": "source-key"})
	status, err := e.IngestRealtime(ctrl.runID, ctrl.sessionID, ev)
	if err != nil || status != types.HostAdmissionStatusAccepted {
		t.Fatalf("ingest status=%v err=%v", status, err)
	}
	if got := len(ctrl.ingress); got != 1 {
		t.Fatalf("bounded ingress queue entries=%d, want 1 before consumption", got)
	}
	if err := ctrl.drainRealtime(context.Background(), &eventCollector{}, 0); err != nil {
		t.Fatal(err)
	}
	if got := len(ctrl.ingress); got != 0 {
		t.Fatalf("bounded ingress queue entries=%d, want 0 after consumption", got)
	}
	if got := len(runtime.seenDedup); got != 1 {
		t.Fatalf("source dedupe entries=%d, want source-owned history", got)
	}
	status, err = e.IngestRealtime(ctrl.runID, ctrl.sessionID, ev)
	if err != nil || status != types.HostAdmissionStatusDuplicate {
		t.Fatalf("consumed duplicate status=%v err=%v", status, err)
	}
	if got := len(ctrl.ingress); got != 0 {
		t.Fatalf("source duplicate requeued: len=%d", got)
	}
}

func TestRealtimeReservationRemainsDuplicateUntilSourceCommit(t *testing.T) {
	e := New(nil, WithActiveRunIngressBuffer(2))
	ctrl, _, err := e.beginActiveRun(context.Background(), "run-commit-window", "session-commit-window", false)
	if err != nil {
		t.Fatal(err)
	}
	defer e.finishActiveRun(ctrl)
	runtime := newControlTestRealtimeRuntime(e, ctrl.runID, ctrl.sessionID)
	ctrl.attachRealtime(runtime)

	ev := controlTestRealtimeEvent("commit-window", ctrl, 1, types.RealtimeEventTypeInterrupt, map[string]any{"dedup_key": "commit-window-key"})
	status, err := e.IngestRealtime(ctrl.runID, ctrl.sessionID, ev)
	if err != nil || status != types.HostAdmissionStatusAccepted {
		t.Fatalf("first admission status=%v err=%v", status, err)
	}
	dequeued := <-ctrl.ingress
	// Reproduce the old drain order deterministically: dequeue bookkeeping ran,
	// but source sequence/dedup/lifecycle mutation has not committed yet.
	runtime.consumeReservedControlEvent(dequeued)

	status, err = e.IngestRealtime(ctrl.runID, ctrl.sessionID, ev)
	if err != nil || status != types.HostAdmissionStatusDuplicate {
		t.Fatalf("duplicate in dequeue/apply window status=%v err=%v", status, err)
	}
}

type reentrantRealtimeHandler struct {
	onEvent func(types.Event)
}

func (h reentrantRealtimeHandler) OnEvent(_ context.Context, ev types.Event) {
	if h.onEvent != nil {
		h.onEvent(ev)
	}
}

func TestRealtimeSourceCommitDoesNotHoldMutexAcrossHandler(t *testing.T) {
	e := New(nil, WithActiveRunIngressBuffer(2))
	ctrl, _, err := e.beginActiveRun(context.Background(), "run-reentrant", "session-reentrant", false)
	if err != nil {
		t.Fatal(err)
	}
	defer e.finishActiveRun(ctrl)
	runtime := newControlTestRealtimeRuntime(e, ctrl.runID, ctrl.sessionID)
	ctrl.attachRealtime(runtime)

	ev := controlTestRealtimeEvent("reentrant", ctrl, 1, types.RealtimeEventTypeInterrupt, map[string]any{"dedup_key": "reentrant-key"})
	status, err := e.IngestRealtime(ctrl.runID, ctrl.sessionID, ev)
	if err != nil || status != types.HostAdmissionStatusAccepted {
		t.Fatalf("first admission status=%v err=%v", status, err)
	}
	callback := make(chan struct{}, 1)
	h := reentrantRealtimeHandler{onEvent: func(event types.Event) {
		if event.Type != realtimeRunnerEventType {
			return
		}
		status, err := e.IngestRealtime(ctrl.runID, ctrl.sessionID, ev)
		if err != nil || status != types.HostAdmissionStatusDuplicate {
			t.Errorf("reentrant duplicate status=%v err=%v", status, err)
		}
		callback <- struct{}{}
	}}
	done := make(chan error, 1)
	go func() { done <- ctrl.drainRealtime(context.Background(), h, 0) }()
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("source commit held runtime/control mutex across handler callback")
	}
	select {
	case <-callback:
	default:
		t.Fatal("reentrant handler was not invoked")
	}
}

func newControlTestRealtimeRuntime(e *Engine, runID, sessionID string) *realtimeSessionRuntime {
	cfg := runtimeconfig.DefaultConfig().Runtime.Realtime
	cfg.Protocol.Enabled = true
	cfg.Protocol.MaxBufferedEvents = 16
	cfg.InterruptResume.Enabled = true
	cfg.InterruptResume.ResumeCursorTTLMS = 300000
	return &realtimeSessionRuntime{
		engine: e, cfg: cfg, runID: runID, sessionID: sessionID, seenDedup: map[string]struct{}{},
	}
}

func controlTestRealtimeEvent(eventID string, ctrl *ActiveRunControl, seq int64, typ types.RealtimeEventType, payload map[string]any) types.RealtimeEventEnvelope {
	if payload == nil {
		payload = map[string]any{}
	}
	return types.RealtimeEventEnvelope{
		EventID: eventID, SessionID: ctrl.sessionID, RunID: ctrl.runID, Seq: seq,
		Type: typ, TS: time.Now().UTC(), Payload: payload,
	}
}

func TestActiveRunRegistryConcurrentAccess(t *testing.T) {
	e := New(nil, WithActiveRunControlLimit(64))
	var wg sync.WaitGroup
	for i := 0; i < 32; i++ {
		i := i
		wg.Add(1)
		go func() {
			defer wg.Done()
			runID := "run-concurrent-" + string(rune('a'+i))
			ctrl, _, err := e.beginActiveRun(context.Background(), runID, "s", false)
			if err == nil {
				e.ActiveRuns()
				e.finishActiveRun(ctrl)
			}
		}()
	}
	wg.Wait()
	if got := len(e.ActiveRuns()); got != 0 {
		t.Fatalf("active runs after concurrent cleanup=%d", got)
	}
}
