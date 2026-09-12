package host

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/FelixSeptem/baymax/core/types"
)

type contractControl struct {
	mu         sync.Mutex
	actions    int
	onAction   func(types.HostCommandEnvelope)
	onRealtime func(types.HostCommandEnvelope)
	status     types.HostAdmissionStatus
	reason     string
	err        error
}

func (s *contractControl) ExecuteAction(_ context.Context, cmd types.HostCommandEnvelope, _ types.ProtocolAction) (types.HostAdmissionStatus, string, error) {
	s.mu.Lock()
	s.actions++
	onAction := s.onAction
	s.mu.Unlock()
	if onAction != nil {
		onAction(cmd)
	}
	if s.status != "" || s.reason != "" || s.err != nil {
		return s.status, s.reason, s.err
	}
	return types.HostAdmissionStatusAccepted, "source.accepted", nil
}

func (s *contractControl) IngestRealtime(_ context.Context, cmd types.HostCommandEnvelope, _ types.RealtimeEventEnvelope) (types.HostAdmissionStatus, string, error) {
	if s.onRealtime != nil {
		s.onRealtime(cmd)
	}
	return types.HostAdmissionStatusAccepted, "source.accepted", nil
}

type contractReadiness struct{ err error }

func (r contractReadiness) CheckHostReadiness(context.Context, types.HostCommandEnvelope) error {
	return r.err
}

type contractAuthorization struct{ err error }

func (a contractAuthorization) AuthorizeHostCommand(context.Context, types.HostCommandEnvelope) error {
	return a.err
}

type contractTerminal struct {
	terminal  *types.TerminalOutcome
	sessionID string
	runID     string
}

func (q *contractTerminal) QueryTerminal(_ context.Context, sessionID, runID string) (*types.TerminalOutcome, error) {
	q.sessionID = sessionID
	q.runID = runID
	return q.terminal, nil
}

type contractSubscription struct {
	projection types.EventStreamBindingProjection
	called     bool
	received   types.EventStreamSubscription
}

type contractRunStarter struct {
	admission types.HostRunStartAdmission
	called    chan struct{}
}

func (s *contractRunStarter) AdmitHostRun(context.Context, types.RunRequest, types.HostRunExecutionMode) (types.HostRunStartAdmission, error) {
	if s.called != nil {
		close(s.called)
	}
	return s.admission, nil
}

type contractRunExecution struct {
	executed chan struct{}
	released chan struct{}
}

type profileParityRunner struct {
	modes chan types.HostRunExecutionMode
}

func (r *profileParityRunner) Run(ctx context.Context, req types.RunRequest, h types.EventHandler) (types.RunResult, error) {
	r.modes <- types.HostRunExecutionModeRun
	return profileParityResult(ctx, req, h)
}

func (r *profileParityRunner) Stream(ctx context.Context, req types.RunRequest, h types.EventHandler) (types.RunResult, error) {
	r.modes <- types.HostRunExecutionModeStream
	return profileParityResult(ctx, req, h)
}

func (r *profileParityRunner) AdmitHostRun(ctx context.Context, req types.RunRequest, mode types.HostRunExecutionMode) (types.HostRunStartAdmission, error) {
	return types.HostRunStartAdmission{
		Status: types.HostAdmissionStatusAccepted, RunID: req.RunID,
		Execution: runnerExecution{runner: r, ctx: context.WithoutCancel(ctx), req: req, mode: mode},
	}, nil
}

func profileParityResult(ctx context.Context, req types.RunRequest, h types.EventHandler) (types.RunResult, error) {
	terminal := types.TerminalOutcome{
		RunID: req.RunID, SessionID: req.SessionID, State: types.RunStateCompleted,
		FailureFamily: types.FailureFamilyNone, Phase: types.ExecutionPhasePostStart,
		SourceReason: "source.completed",
	}
	h.OnEvent(ctx, types.Event{
		Version: "v1", Type: "run.finished", RunID: req.RunID, Time: time.Now().UTC(),
		Payload: map[string]any{"terminal_outcome": terminal},
	})
	return types.RunResult{RunID: req.RunID, TerminalOutcome: &terminal}, nil
}

type blockingFrameWriter struct {
	started chan struct{}
	exited  chan struct{}
	mu      sync.Mutex
	visible []any
}

type panicOnRuntimeEventFrameWriter struct {
	eventStarted chan struct{}
	mu           sync.Mutex
	panicked     bool
}

func (w *panicOnRuntimeEventFrameWriter) WriteFrame(_ context.Context, frame any) error {
	if _, ok := frame.(types.HostRuntimeEventEnvelope); ok {
		w.mu.Lock()
		first := !w.panicked
		w.panicked = true
		w.mu.Unlock()
		if first {
			close(w.eventStarted)
			panic("runtime event writer exploded")
		}
	}
	return nil
}

func (*panicOnRuntimeEventFrameWriter) Close() error { return nil }

type delayedCancelFrameWriter struct {
	started     chan struct{}
	canceled    chan struct{}
	allowReturn chan struct{}
	exited      chan struct{}
}

func (w *delayedCancelFrameWriter) WriteFrame(ctx context.Context, _ any) error {
	close(w.started)
	<-ctx.Done()
	close(w.canceled)
	<-w.allowReturn
	close(w.exited)
	return ctx.Err()
}

func (*delayedCancelFrameWriter) Close() error { return nil }

type visibleAfterCancelFrameWriter struct {
	started chan struct{}
	visible chan any
}

func (w *visibleAfterCancelFrameWriter) WriteFrame(ctx context.Context, frame any) error {
	select {
	case w.started <- struct{}{}:
	default:
	}
	<-ctx.Done()
	w.visible <- frame
	return nil
}

func (*visibleAfterCancelFrameWriter) Close() error { return nil }

type firstWriteBlockingFrameWriter struct {
	firstStarted chan struct{}
	releaseFirst chan struct{}
	mu           sync.Mutex
	writes       int
}

type contextCapturingFrameWriter struct {
	contexts chan context.Context
}

func (w *contextCapturingFrameWriter) WriteFrame(ctx context.Context, _ any) error {
	w.contexts <- ctx
	return nil
}

func (*contextCapturingFrameWriter) Close() error { return nil }

func (w *firstWriteBlockingFrameWriter) WriteFrame(ctx context.Context, _ any) error {
	w.mu.Lock()
	w.writes++
	writeNumber := w.writes
	w.mu.Unlock()
	if writeNumber == 1 {
		close(w.firstStarted)
		select {
		case <-w.releaseFirst:
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return nil
}

func (w *firstWriteBlockingFrameWriter) Close() error { return nil }

func (w *blockingFrameWriter) WriteFrame(ctx context.Context, frame any) error {
	select {
	case <-w.started:
	default:
		close(w.started)
	}
	defer close(w.exited)
	<-ctx.Done()
	return ctx.Err()
}

func (*blockingFrameWriter) Close() error { return nil }

func (w *blockingFrameWriter) VisibleCount() int {
	w.mu.Lock()
	defer w.mu.Unlock()
	return len(w.visible)
}

func (e *contractRunExecution) Execute(h types.EventHandler) (types.RunResult, error) {
	if e.executed != nil {
		close(e.executed)
	}
	h.OnEvent(context.Background(), types.Event{Version: "v1", Type: "run.started", RunID: "run-1", Time: time.Now().UTC()})
	return types.RunResult{RunID: "run-1"}, nil
}

func (e *contractRunExecution) Release() {
	if e.released != nil {
		close(e.released)
	}
}

func (s *contractSubscription) SubscribeHostEvents(_ context.Context, subscription types.EventStreamSubscription) (types.EventStreamBindingProjection, error) {
	s.called = true
	s.received = subscription
	return s.projection, nil
}

type contractRaceControl struct {
	terminalPublished <-chan struct{}
}

func (c contractRaceControl) ExecuteAction(context.Context, types.HostCommandEnvelope, types.ProtocolAction) (types.HostAdmissionStatus, string, error) {
	select {
	case <-c.terminalPublished:
		return types.HostAdmissionStatusRejected, "source.already_terminal", nil
	default:
		return types.HostAdmissionStatusAccepted, "source.cancel_admitted", nil
	}
}

func (contractRaceControl) IngestRealtime(context.Context, types.HostCommandEnvelope, types.RealtimeEventEnvelope) (types.HostAdmissionStatus, string, error) {
	return types.HostAdmissionStatusRejected, "source.realtime_unsupported", nil
}

func validHostCommand(kind types.HostCommandKind) types.HostCommandEnvelope {
	return types.HostCommandEnvelope{
		Version: types.HostProtocolVersionV1, MessageID: "message-1", Kind: kind, Time: time.Now().UTC(),
		RequestID: "request-1", SessionID: "session-1", RunID: "run-1",
	}
}

func TestCoordinatorChecksReadinessAndAuthorizationBeforeSourceMutation(t *testing.T) {
	control := &contractControl{}
	coord := New(nil,
		WithSourceControl(control),
		WithReadiness(contractReadiness{}),
		WithAuthorization(contractAuthorization{err: errors.New("policy.denied")}),
	)
	cmd := validHostCommand(types.HostCommandKindAction)
	cmd.Payload = map[string]any{"action": "cancel"}
	resp, err := coord.Connect(func(any) error { return nil }).HandleCommand(context.Background(), cmd)
	if err != nil {
		t.Fatal(err)
	}
	if resp.Status != types.HostAdmissionStatusRejected || resp.ReasonCode != "policy.denied" {
		t.Fatalf("response = %#v", resp)
	}
	control.mu.Lock()
	defer control.mu.Unlock()
	if control.actions != 0 {
		t.Fatalf("source mutations = %d, want 0", control.actions)
	}
}

func TestCoordinatorPreservesSourceActionRejectionReasonWithoutTerminalEvent(t *testing.T) {
	const sourceReason = "source.cancel_session_mismatch"
	control := &contractControl{status: types.HostAdmissionStatusRejected, reason: sourceReason}
	frames := make(chan any, 2)
	conn := New(nil, WithSourceControl(control)).Connect(func(frame any) error {
		frames <- frame
		return nil
	})
	cmd := validHostCommand(types.HostCommandKindAction)
	cmd.Payload = map[string]any{"action": "cancel"}

	response, err := conn.HandleCommand(context.Background(), cmd)
	if err != nil {
		t.Fatal(err)
	}
	if response.Status != types.HostAdmissionStatusRejected || response.ReasonCode != sourceReason {
		t.Fatalf("response=%#v, want rejected reason %q", response, sourceReason)
	}
	if frame := <-frames; frame.(types.HostCommandResponse).ReasonCode != sourceReason {
		t.Fatalf("written response=%#v, want source reason %q", frame, sourceReason)
	}
	select {
	case frame := <-frames:
		t.Fatalf("rejected source action emitted synthetic frame %#v", frame)
	case <-time.After(20 * time.Millisecond):
	}
}

func TestCoordinatorCancelCompletionRacePreservesAdmissionAndAuthoritativeTerminal(t *testing.T) {
	for _, terminalFirst := range []bool{false, true} {
		name := "cancel-admitted"
		if terminalFirst {
			name = "completion-wins"
		}
		t.Run(name, func(t *testing.T) {
			terminalPublished := make(chan struct{})
			if terminalFirst {
				close(terminalPublished)
			}
			control := contractRaceControl{terminalPublished: terminalPublished}
			frames := make(chan any, 3)
			conn := New(nil, WithSourceControl(control)).Connect(func(frame any) error {
				frames <- frame
				return nil
			})
			cmd := validHostCommand(types.HostCommandKindAction)
			cmd.Payload = map[string]any{"action": "cancel"}

			response, err := conn.HandleCommand(context.Background(), cmd)
			if err != nil {
				t.Fatal(err)
			}
			wantStatus := types.HostAdmissionStatusAccepted
			wantReason := "source.cancel_admitted"
			if terminalFirst {
				wantStatus = types.HostAdmissionStatusRejected
				wantReason = "source.already_terminal"
			}
			if response.Status != wantStatus || response.ReasonCode != wantReason || response.Terminal != nil {
				t.Fatalf("response=%#v, want status=%q reason=%q and no terminal", response, wantStatus, wantReason)
			}
			if frame := <-frames; frame.(types.HostCommandResponse).ReasonCode != wantReason {
				t.Fatalf("written admission=%#v", frame)
			}
			select {
			case frame := <-frames:
				t.Fatalf("cancel admission synthesized terminal frame %#v", frame)
			case <-time.After(20 * time.Millisecond):
			}
		})
	}
}

func TestConnectionBoundsCommandCorrelationWithoutSourceMutation(t *testing.T) {
	control := &contractControl{}
	conn := New(nil, WithSourceControl(control), WithCommandCorrelationLimit(1)).Connect(func(any) error { return nil })
	first := validHostCommand(types.HostCommandKindAction)
	first.Payload = map[string]any{"action": "cancel"}
	if response, err := conn.HandleCommand(context.Background(), first); err != nil || response.Status != types.HostAdmissionStatusAccepted {
		t.Fatalf("first response=%#v err=%v", response, err)
	}
	second := first
	second.MessageID = "message-2"
	second.RequestID = "request-2"
	response, err := conn.HandleCommand(context.Background(), second)
	if err != nil {
		t.Fatal(err)
	}
	if response.Status != types.HostAdmissionStatusRejected || response.ReasonCode != HostReasonCorrelationLimit {
		t.Fatalf("second response=%#v", response)
	}
	control.mu.Lock()
	defer control.mu.Unlock()
	if control.actions != 1 {
		t.Fatalf("source mutations=%d, want 1", control.actions)
	}
}

func TestEveryClassifiedCommandOutcomeIsWrittenExactlyOnce(t *testing.T) {
	tests := []struct {
		name       string
		configure  func(*contractControl) []Option
		commands   []types.HostCommandEnvelope
		want       types.HostAdmissionStatus
		wantReason string
	}{
		{
			name:      "duplicate",
			configure: func(control *contractControl) []Option { return []Option{WithSourceControl(control)} },
			commands: func() []types.HostCommandEnvelope {
				cmd := validHostCommand(types.HostCommandKindAction)
				cmd.Payload = map[string]any{"action": "cancel"}
				return []types.HostCommandEnvelope{cmd, cmd}
			}(),
			want:       types.HostAdmissionStatusDuplicate,
			wantReason: types.HostReasonDuplicate,
		},
		{
			name: "correlation limit",
			configure: func(control *contractControl) []Option {
				return []Option{WithSourceControl(control), WithCommandCorrelationLimit(1)}
			},
			commands: func() []types.HostCommandEnvelope {
				first := validHostCommand(types.HostCommandKindAction)
				first.Payload = map[string]any{"action": "cancel"}
				second := first
				second.MessageID = "message-2"
				second.RequestID = "request-2"
				return []types.HostCommandEnvelope{first, second}
			}(),
			want:       types.HostAdmissionStatusRejected,
			wantReason: HostReasonCorrelationLimit,
		},
		{
			name: "readiness rejection",
			configure: func(control *contractControl) []Option {
				return []Option{WithSourceControl(control), WithReadiness(contractReadiness{err: errors.New("runtime.not_ready")})}
			},
			commands: func() []types.HostCommandEnvelope {
				cmd := validHostCommand(types.HostCommandKindAction)
				cmd.Payload = map[string]any{"action": "cancel"}
				return []types.HostCommandEnvelope{cmd}
			}(),
			want:       types.HostAdmissionStatusRejected,
			wantReason: "runtime.not_ready",
		},
		{
			name: "authorization rejection",
			configure: func(control *contractControl) []Option {
				return []Option{WithSourceControl(control), WithAuthorization(contractAuthorization{err: errors.New("policy.denied")})}
			},
			commands: func() []types.HostCommandEnvelope {
				cmd := validHostCommand(types.HostCommandKindAction)
				cmd.Payload = map[string]any{"action": "cancel"}
				return []types.HostCommandEnvelope{cmd}
			}(),
			want:       types.HostAdmissionStatusRejected,
			wantReason: "policy.denied",
		},
		{
			name:      "representable malformed envelope",
			configure: func(*contractControl) []Option { return nil },
			commands: func() []types.HostCommandEnvelope {
				cmd := validHostCommand(types.HostCommandKindAction)
				cmd.RunID = ""
				cmd.Payload = map[string]any{"action": "cancel"}
				return []types.HostCommandEnvelope{cmd}
			}(),
			want:       types.HostAdmissionStatusRejected,
			wantReason: types.HostReasonInvalidEnvelope,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			control := &contractControl{}
			var mu sync.Mutex
			var frames []any
			conn := New(nil, tt.configure(control)...).Connect(func(frame any) error {
				mu.Lock()
				frames = append(frames, frame)
				mu.Unlock()
				return nil
			})
			var response types.HostCommandResponse
			for _, cmd := range tt.commands {
				var err error
				response, err = conn.HandleCommand(context.Background(), cmd)
				if err != nil {
					t.Fatalf("HandleCommand error = %v", err)
				}
			}
			if response.Status != tt.want || response.ReasonCode != tt.wantReason {
				t.Fatalf("response = %#v, want status=%q reason=%q", response, tt.want, tt.wantReason)
			}
			mu.Lock()
			defer mu.Unlock()
			responses := 0
			for _, frame := range frames {
				if written, ok := frame.(types.HostCommandResponse); ok && written.MessageID == response.MessageID && written.Status == response.Status {
					responses++
				}
			}
			if responses != 1 {
				t.Fatalf("matching command responses = %d, frames=%#v", responses, frames)
			}
		})
	}
}

func TestSourceMutationCannotPublishEventBeforeCommandResponse(t *testing.T) {
	for _, kind := range []types.HostCommandKind{types.HostCommandKindAction, types.HostCommandKindRealtimeInterrupt} {
		t.Run(string(kind), func(t *testing.T) {
			control := &contractControl{}
			frames := make(chan any, 2)
			conn := New(nil, WithSourceControl(control)).Connect(func(frame any) error {
				frames <- frame
				return nil
			})
			publish := func(cmd types.HostCommandEnvelope) {
				_ = conn.emitEvent(context.Background(), cmd, types.Event{
					Version: "v1", Type: "run.progress", RunID: cmd.RunID, Time: time.Now().UTC(),
				})
			}
			control.onAction = publish
			control.onRealtime = publish

			cmd := validHostCommand(kind)
			if kind == types.HostCommandKindAction {
				cmd.Payload = map[string]any{"action": "cancel"}
			} else {
				cmd.Payload = map[string]any{"seq": float64(1)}
			}
			if response, err := conn.HandleCommand(context.Background(), cmd); err != nil || response.Status != types.HostAdmissionStatusAccepted {
				t.Fatalf("response=%#v err=%v", response, err)
			}
			if first := <-frames; first == nil {
				t.Fatal("missing first frame")
			} else if _, ok := first.(types.HostCommandResponse); !ok {
				t.Fatalf("first frame = %T, want command response", first)
			}
			if second := <-frames; second == nil {
				t.Fatal("missing source event")
			} else if _, ok := second.(types.HostRuntimeEventEnvelope); !ok {
				t.Fatalf("second frame = %T, want runtime event", second)
			}
		})
	}
}

func TestHITLContinuationCannotResumeBeforeCommandResponse(t *testing.T) {
	responseWriteStarted := make(chan struct{})
	allowResponseWrite := make(chan struct{})
	hostRequestWritten := make(chan struct{})
	conn := New(nil, WithDeliveryTimeout(time.Second)).Connect(func(frame any) error {
		switch frame.(type) {
		case types.HostRequestEnvelope:
			close(hostRequestWritten)
		case types.HostCommandResponse:
			close(responseWriteStarted)
			<-allowResponseWrite
		}
		return nil
	})
	resolverDone := make(chan bool, 1)
	go func() {
		allowed, _ := conn.Confirm(context.Background(), types.ActionGateConfirmRequest{
			Check: types.ActionGateCheck{RunID: "run-1", SessionID: "session-1", CallID: "gate-1"}, Timeout: time.Second,
		})
		resolverDone <- allowed
	}()
	<-hostRequestWritten
	cmd := validHostCommand(types.HostCommandKindHITLRespond)
	cmd.RequestID = "command-request-1"
	cmd.Payload = map[string]any{"request_id": "gate-1", "accepted": true}
	handleDone := make(chan error, 1)
	go func() {
		_, err := conn.HandleCommand(context.Background(), cmd)
		handleDone <- err
	}()
	<-responseWriteStarted
	select {
	case <-resolverDone:
		t.Fatal("HITL continuation resumed before its command response was visible")
	default:
	}
	close(allowResponseWrite)
	if err := <-handleDone; err != nil {
		t.Fatal(err)
	}
	if allowed := <-resolverDone; !allowed {
		t.Fatal("action gate did not resume after command response")
	}
}

func TestQueuedAcceptedRunResponseIgnoresLaterCommandCancellation(t *testing.T) {
	responseWriteStarted := make(chan struct{})
	allowResponseWrite := make(chan struct{})
	released := make(chan struct{}, 1)
	executed := make(chan struct{})
	execution := &contractRunExecution{executed: executed, released: released}
	starter := &contractRunStarter{admission: types.HostRunStartAdmission{
		Status: types.HostAdmissionStatusAccepted, Execution: execution,
	}}
	conn := New(nil, WithRunStarter(starter), WithDeliveryTimeout(time.Second)).Connect(func(frame any) error {
		if _, ok := frame.(types.HostCommandResponse); ok {
			close(responseWriteStarted)
			<-allowResponseWrite
		}
		return nil
	})
	ctx, cancel := context.WithCancel(context.Background())
	handleDone := make(chan error, 1)
	go func() {
		_, err := conn.HandleCommand(ctx, validHostCommand(types.HostCommandKindRunStart))
		handleDone <- err
	}()
	<-responseWriteStarted
	cancel()
	select {
	case <-released:
		t.Fatal("queued accepted response released its Run reservation after command cancellation")
	case <-time.After(20 * time.Millisecond):
	}
	select {
	case err := <-handleDone:
		t.Fatalf("HandleCommand returned before queued response completed: %v", err)
	default:
	}
	close(allowResponseWrite)
	if err := <-handleDone; err != nil {
		t.Fatalf("HandleCommand error = %v", err)
	}
	select {
	case <-executed:
	case <-time.After(time.Second):
		t.Fatal("accepted Run execution did not start after response became visible")
	}
	select {
	case <-released:
		t.Fatal("successful admission released its Run reservation")
	default:
	}
}

func TestDeliveryTimeoutCancelsWriteBeforeRunAdmissionRollback(t *testing.T) {
	writer := &blockingFrameWriter{started: make(chan struct{}), exited: make(chan struct{})}
	released := make(chan struct{}, 1)
	executed := make(chan struct{})
	execution := &contractRunExecution{executed: executed, released: released}
	conn := New(nil, WithRunStarter(&contractRunStarter{admission: types.HostRunStartAdmission{
		Status: types.HostAdmissionStatusAccepted, Execution: execution,
	}}), WithDeliveryTimeout(20*time.Millisecond)).ConnectFrameWriter(writer)

	_, err := conn.HandleCommand(context.Background(), validHostCommand(types.HostCommandKindRunStart))
	if !errors.Is(err, ErrDeliveryTimeout) {
		t.Fatalf("HandleCommand error = %v, want delivery timeout", err)
	}
	select {
	case <-writer.exited:
	case <-time.After(time.Second):
		t.Fatal("timed-out write goroutine did not exit")
	}
	if writer.VisibleCount() != 0 {
		t.Fatal("timed-out response became visible")
	}
	select {
	case <-released:
	case <-time.After(time.Second):
		t.Fatal("Run reservation was not released after failed response delivery")
	}
	select {
	case <-executed:
		t.Fatal("Run executed without a visible accepted response")
	default:
	}
}

func TestCloseCancelsBlockedHITLRequestWriteAndSettlesPending(t *testing.T) {
	writer := &blockingFrameWriter{started: make(chan struct{}), exited: make(chan struct{})}
	conn := New(nil, WithDeliveryTimeout(time.Second)).ConnectFrameWriter(writer)
	done := make(chan error, 1)
	go func() {
		_, err := conn.Confirm(context.Background(), types.ActionGateConfirmRequest{
			Check: types.ActionGateCheck{RunID: "run-1", SessionID: "session-1", CallID: "gate-close"}, Timeout: time.Second,
		})
		done <- err
	}()
	<-writer.started
	conn.Close()
	select {
	case <-writer.exited:
	case <-time.After(time.Second):
		t.Fatal("closed connection left write goroutine blocked")
	}
	if err := <-done; !errors.Is(err, ErrConnectionClosed) {
		t.Fatalf("Confirm error = %v, want connection closed", err)
	}
	if conn.PendingCount() != 0 || writer.VisibleCount() != 0 {
		t.Fatalf("pending=%d visible=%d", conn.PendingCount(), writer.VisibleCount())
	}
}

func TestCloseWaitsForActiveProductionWriteOutcomeBeforeRunRollback(t *testing.T) {
	writer := &delayedCancelFrameWriter{
		started: make(chan struct{}), canceled: make(chan struct{}),
		allowReturn: make(chan struct{}), exited: make(chan struct{}),
	}
	released := make(chan struct{}, 1)
	execution := &contractRunExecution{executed: make(chan struct{}), released: released}
	conn := New(nil, WithRunStarter(&contractRunStarter{admission: types.HostRunStartAdmission{
		Status: types.HostAdmissionStatusAccepted, Execution: execution,
	}}), WithDeliveryTimeout(time.Second)).ConnectFrameWriter(writer)
	handleDone := make(chan error, 1)
	go func() {
		_, err := conn.HandleCommand(context.Background(), validHostCommand(types.HostCommandKindRunStart))
		handleDone <- err
	}()
	<-writer.started
	conn.Close()
	<-writer.canceled
	select {
	case err := <-handleDone:
		t.Fatalf("HandleCommand returned before writer outcome: %v", err)
	case <-released:
		t.Fatal("Run reservation rolled back before writer outcome")
	default:
	}
	close(writer.allowReturn)
	if err := <-handleDone; !errors.Is(err, ErrConnectionClosed) {
		t.Fatalf("HandleCommand error = %v, want connection closed", err)
	}
	select {
	case <-writer.exited:
	default:
		t.Fatal("writer had not exited before HandleCommand returned")
	}
	select {
	case <-released:
	case <-time.After(time.Second):
		t.Fatal("Run reservation was not released after write cancellation")
	}
}

func TestCommandResponseBarrierOverflowClosesConnection(t *testing.T) {
	control := &contractControl{}
	conn := New(nil, WithSourceControl(control), WithDeliveryQueueSize(2), WithDeliveryTimeout(time.Second)).ConnectFrameWriter(FrameWriterFunc(func(context.Context, any) error { return nil }))
	control.onAction = func(cmd types.HostCommandEnvelope) {
		for i := 0; i < 3; i++ {
			_ = conn.emitEvent(context.Background(), cmd, types.Event{
				Version: "v1", Type: "run.progress", RunID: cmd.RunID, Time: time.Now().UTC(),
				Payload: map[string]any{"index": i},
			})
		}
	}
	cmd := validHostCommand(types.HostCommandKindAction)
	cmd.Payload = map[string]any{"action": "cancel"}
	if _, err := conn.HandleCommand(context.Background(), cmd); !errors.Is(err, ErrConnectionClosed) && !errors.Is(err, ErrDeliveryFull) {
		t.Fatalf("HandleCommand error = %v, want bounded barrier failure", err)
	}
	if !conn.isClosed() {
		t.Fatal("barrier overflow did not close connection")
	}
}

func TestVisibleAcceptedRunResponseCommitsDespiteConcurrentClose(t *testing.T) {
	writer := &visibleAfterCancelFrameWriter{started: make(chan struct{}, 1), visible: make(chan any, 1)}
	executed := make(chan struct{})
	released := make(chan struct{}, 1)
	conn := New(nil, WithRunStarter(&contractRunStarter{admission: types.HostRunStartAdmission{
		Status:    types.HostAdmissionStatusAccepted,
		Execution: &contractRunExecution{executed: executed, released: released},
	}}), WithDeliveryTimeout(time.Second)).ConnectFrameWriter(writer)
	done := make(chan error, 1)
	go func() {
		_, err := conn.HandleCommand(context.Background(), validHostCommand(types.HostCommandKindRunStart))
		done <- err
	}()
	<-writer.started
	conn.Close()
	if frame := <-writer.visible; frame == nil {
		t.Fatal("writer did not make accepted response visible")
	}
	if err := <-done; err != nil {
		t.Fatalf("visible response was treated as failed: %v", err)
	}
	select {
	case <-executed:
	case <-time.After(time.Second):
		t.Fatal("visible accepted response did not commit Run execution")
	}
	select {
	case <-released:
		t.Fatal("visible accepted response released Run reservation")
	default:
	}
}

func TestVisibleHITLResponseCommitsDespiteDeliveryTimeout(t *testing.T) {
	visible := make(chan any, 4)
	requestWritten := make(chan struct{})
	writer := FrameWriterFunc(func(ctx context.Context, frame any) error {
		visible <- frame
		if _, ok := frame.(types.HostRequestEnvelope); ok {
			close(requestWritten)
			return nil
		}
		<-ctx.Done()
		visible <- frame
		return nil
	})
	conn := New(nil, WithDeliveryTimeout(10*time.Millisecond)).ConnectFrameWriter(writer)
	resolved := make(chan bool, 1)
	go func() {
		allowed, _ := conn.Confirm(context.Background(), types.ActionGateConfirmRequest{
			Check: types.ActionGateCheck{RunID: "run-1", SessionID: "session-1", CallID: "gate-visible"}, Timeout: time.Second,
		})
		resolved <- allowed
	}()
	<-requestWritten
	cmd := validHostCommand(types.HostCommandKindHITLRespond)
	cmd.RequestID = "hitl-command-visible"
	cmd.Payload = map[string]any{"request_id": "gate-visible", "accepted": true}
	if _, err := conn.HandleCommand(context.Background(), cmd); err != nil {
		t.Fatalf("visible HITL response was treated as failed: %v", err)
	}
	if allowed := <-resolved; !allowed {
		t.Fatal("visible HITL response did not commit continuation")
	}
}

func TestCanceledCommandContextCannotLeaveMutatedSourceWithoutResponseOrDisconnect(t *testing.T) {
	for _, kind := range []types.HostCommandKind{types.HostCommandKindAction, types.HostCommandKindRealtimeInterrupt} {
		t.Run(string(kind), func(t *testing.T) {
			writer := &firstWriteBlockingFrameWriter{firstStarted: make(chan struct{}), releaseFirst: make(chan struct{})}
			control := &contractControl{}
			mutated := make(chan struct{})
			if kind == types.HostCommandKindAction {
				control.onAction = func(types.HostCommandEnvelope) { close(mutated) }
			} else {
				control.onRealtime = func(types.HostCommandEnvelope) { close(mutated) }
			}
			conn := New(nil, WithSourceControl(control), WithDeliveryQueueSize(1), WithDeliveryTimeout(time.Second)).ConnectFrameWriter(writer)
			firstDone := make(chan error, 1)
			go func() { firstDone <- conn.deliver(context.Background(), "occupy-writer", true) }()
			<-writer.firstStarted
			queuedDone := make(chan error, 1)
			go func() { queuedDone <- conn.deliver(context.Background(), "occupy-queue", true) }()
			time.Sleep(10 * time.Millisecond)

			cmd := validHostCommand(kind)
			if kind == types.HostCommandKindAction {
				cmd.Payload = map[string]any{"action": "cancel"}
			} else {
				cmd.Payload = map[string]any{"seq": float64(1)}
			}
			ctx, cancel := context.WithCancel(context.Background())
			handleDone := make(chan error, 1)
			go func() {
				_, err := conn.HandleCommand(ctx, cmd)
				handleDone <- err
			}()
			<-mutated
			cancel()
			select {
			case err := <-handleDone:
				if errors.Is(err, context.Canceled) && !conn.isClosed() {
					t.Fatalf("mutated source lost its response while connection stayed open: %v", err)
				}
			case <-time.After(20 * time.Millisecond):
				// Desired behavior: caller cancellation cannot interrupt critical
				// response enqueue; the connection delivery deadline still owns it.
			}
			close(writer.releaseFirst)
			if err := <-firstDone; err != nil {
				t.Fatal(err)
			}
			if err := <-queuedDone; err != nil {
				t.Fatal(err)
			}
			if err := <-handleDone; err != nil {
				t.Fatalf("command response delivery error = %v", err)
			}
		})
	}
}

func TestCoordinatorDispatchesTerminalQueryWithoutSynthesizingTerminal(t *testing.T) {
	terminal := &types.TerminalOutcome{
		RunID: "run-1", SessionID: "session-1", State: types.RunStateCompleted,
		FailureFamily: types.FailureFamilyNone, Phase: types.ExecutionPhasePostStart,
		SourceReason: "source.completed", Attempt: 2, CausationID: "cause-source-1",
	}
	query := &contractTerminal{terminal: terminal}
	coord := New(nil, WithTerminalQuery(query))
	frames := make(chan any, 2)
	resp, err := coord.Connect(func(frame any) error { frames <- frame; return nil }).HandleCommand(context.Background(), validHostCommand(types.HostCommandKindRunGet))
	if err != nil {
		t.Fatal(err)
	}
	if resp.Status != types.HostAdmissionStatusAccepted || resp.Terminal != nil {
		t.Fatalf("response = %#v", resp)
	}
	if query.sessionID != "session-1" || query.runID != "run-1" {
		t.Fatalf("terminal query correlation=(%q,%q)", query.sessionID, query.runID)
	}
	if raw := <-frames; raw.(types.HostCommandResponse).Status != types.HostAdmissionStatusAccepted {
		t.Fatalf("written response = %#v", raw)
	}
	select {
	case raw := <-frames:
		event, ok := raw.(types.HostRuntimeEventEnvelope)
		if !ok || event.RequestID != "request-1" || event.Event.Kind != types.ProtocolEventKindState {
			t.Fatalf("terminal event = %#v", raw)
		}
		projected, ok := event.Event.Payload["terminal_outcome"].(types.TerminalOutcome)
		if !ok || projected != *terminal {
			t.Fatalf("projected terminal=%#v, want authoritative %#v", event.Event.Payload["terminal_outcome"], *terminal)
		}
	case <-time.After(time.Second):
		t.Fatal("authoritative terminal event was not delivered")
	}
}

func TestCoordinatorProjectsSourceSubscriptionCatchUp(t *testing.T) {
	subscription := types.EventStreamSubscription{
		Version: types.DurableEventStreamBindingVersionV1, SubscriptionID: "subscription-recovery-1",
		Source: types.ProtocolSourceRealtime, SessionID: "session-1", RunID: "run-1",
		StartMode: types.EventStreamStartAfterCursor, Cursor: types.EventStreamCursor{Value: "cursor-2", Sequence: 2},
		DeliveryPolicy: types.EventStreamDeliveryPolicyReject, MaxBatchSize: 8,
	}
	history := types.RealtimeEventEnvelope{
		EventID: "event-3", SessionID: "session-1", RunID: "run-1", Seq: 3,
		Type: types.RealtimeEventTypeAck, TS: time.Now().UTC(), Payload: map[string]any{"origin": "catch-up"},
	}
	projection, err := types.ProjectEventStreamBinding(subscription, types.EventStreamBindingOutcome{
		SubscriptionID: subscription.SubscriptionID, Phase: types.EventStreamBindingPhaseLive,
		ReasonCode: "realtime.binding.live", SourceOutcomeDeclared: true,
	}, []types.RealtimeEventEnvelope{history}, nil)
	if err != nil {
		t.Fatal(err)
	}
	source := &contractSubscription{projection: projection}
	frames := make(chan any, 2)
	conn := New(nil, WithEventSubscription(source)).Connect(func(frame any) error {
		frames <- frame
		return nil
	})
	cmd := validHostCommand(types.HostCommandKindEventsSubscribe)
	cmd.Payload = map[string]any{
		"version": subscription.Version, "subscription_id": subscription.SubscriptionID,
		"source": string(subscription.Source), "start_mode": string(subscription.StartMode),
		"cursor":          map[string]any{"value": subscription.Cursor.Value, "sequence": subscription.Cursor.Sequence},
		"delivery_policy": string(subscription.DeliveryPolicy), "max_batch_size": subscription.MaxBatchSize,
	}

	response, err := conn.HandleCommand(context.Background(), cmd)
	if err != nil || response.Status != types.HostAdmissionStatusAccepted {
		t.Fatalf("response=%#v err=%v", response, err)
	}
	if !source.called || source.received.SessionID != cmd.SessionID || source.received.RunID != cmd.RunID || source.received.Cursor != subscription.Cursor {
		t.Fatalf("source subscription=%#v", source.received)
	}
	if frame := <-frames; frame.(types.HostCommandResponse).Status != types.HostAdmissionStatusAccepted {
		t.Fatalf("first frame=%#v", frame)
	}
	select {
	case frame := <-frames:
		event, ok := frame.(types.HostRuntimeEventEnvelope)
		if !ok || event.Event.EventID != history.EventID || event.Event.Sequence != history.Seq || event.Event.StreamBinding == nil {
			t.Fatalf("catch-up frame=%#v", frame)
		}
		binding := event.Event.StreamBinding
		if binding.SubscriptionID != subscription.SubscriptionID || binding.CursorMode != types.EventStreamStartAfterCursor || binding.SequenceBoundary != history.Seq {
			t.Fatalf("catch-up binding=%#v", binding)
		}
	case <-time.After(time.Second):
		t.Fatal("source catch-up projection was not delivered")
	}
}

func TestRunStartUsesSourceAdmissionBeforeExecution(t *testing.T) {
	executed := make(chan struct{})
	starter := &contractRunStarter{admission: types.HostRunStartAdmission{
		Status:     types.HostAdmissionStatusRejected,
		ReasonCode: "source.run_duplicate",
		Execution:  &contractRunExecution{executed: executed},
	}}
	frames := make(chan any, 1)
	conn := New(nil, WithRunStarter(starter)).Connect(func(frame any) error {
		frames <- frame
		return nil
	})

	response, err := conn.HandleCommand(context.Background(), validHostCommand(types.HostCommandKindRunStart))
	if err != nil {
		t.Fatal(err)
	}
	if response.Status != types.HostAdmissionStatusRejected || response.ReasonCode != "source.run_duplicate" {
		t.Fatalf("response=%#v", response)
	}
	select {
	case <-executed:
		t.Fatal("rejected source admission started execution")
	default:
	}
}

func TestAcceptedRunResponseIsWrittenBeforeSourceEvents(t *testing.T) {
	execution := &contractRunExecution{executed: make(chan struct{})}
	starter := &contractRunStarter{admission: types.HostRunStartAdmission{
		Status:    types.HostAdmissionStatusAccepted,
		Execution: execution,
	}}
	frames := make(chan any, 2)
	conn := New(nil, WithRunStarter(starter)).Connect(func(frame any) error {
		frames <- frame
		return nil
	})

	response, err := conn.HandleCommand(context.Background(), validHostCommand(types.HostCommandKindRunStart))
	if err != nil {
		t.Fatal(err)
	}
	if response.Status != types.HostAdmissionStatusAccepted {
		t.Fatalf("response=%#v", response)
	}
	first := <-frames
	if _, ok := first.(types.HostCommandResponse); !ok {
		t.Fatalf("first frame=%T, want command response", first)
	}
	second := <-frames
	if _, ok := second.(types.HostRuntimeEventEnvelope); !ok {
		t.Fatalf("second frame=%T, want runtime event", second)
	}
}

func TestFirstProfileRunAndStreamStartHaveEquivalentAdmissionAndTerminalProjection(t *testing.T) {
	type normalized struct {
		status         types.HostAdmissionStatus
		terminalState  types.RunState
		failureFamily  types.FailureFamily
		sourceReason   string
		frameOrderOkay bool
	}
	results := make(map[types.HostRunExecutionMode]normalized, 2)
	for _, mode := range []types.HostRunExecutionMode{types.HostRunExecutionModeRun, types.HostRunExecutionModeStream} {
		t.Run(string(mode), func(t *testing.T) {
			runner := &profileParityRunner{modes: make(chan types.HostRunExecutionMode, 1)}
			frames := make(chan any, 2)
			conn := New(runner).Connect(func(frame any) error {
				frames <- frame
				return nil
			})
			cmd := validHostCommand(types.HostCommandKindRunStart)
			cmd.MessageID = "message-" + string(mode)
			cmd.RequestID = "request-" + string(mode)
			cmd.Payload = map[string]any{"input": "hello", "stream": mode == types.HostRunExecutionModeStream}

			response, err := conn.HandleCommand(context.Background(), cmd)
			if err != nil {
				t.Fatal(err)
			}
			if invoked := <-runner.modes; invoked != mode {
				t.Fatalf("source mode=%q, want %q", invoked, mode)
			}
			first := <-frames
			writtenResponse, ok := first.(types.HostCommandResponse)
			if !ok || writtenResponse.Status != types.HostAdmissionStatusAccepted || writtenResponse.Terminal != nil {
				t.Fatalf("first frame=%#v, want terminal-free accepted response", first)
			}
			var event types.HostRuntimeEventEnvelope
			select {
			case frame := <-frames:
				var eventOK bool
				event, eventOK = frame.(types.HostRuntimeEventEnvelope)
				if !eventOK {
					t.Fatalf("second frame=%#v, want source event", frame)
				}
			case <-time.After(time.Second):
				t.Fatal("source terminal projection was not delivered")
			}
			terminal, ok := event.Event.Payload["terminal_outcome"].(types.TerminalOutcome)
			if !ok {
				t.Fatalf("terminal projection=%#v", event.Event.Payload["terminal_outcome"])
			}
			results[mode] = normalized{
				status: response.Status, terminalState: terminal.State,
				failureFamily: terminal.FailureFamily, sourceReason: terminal.SourceReason,
				frameOrderOkay: true,
			}
		})
	}
	if results[types.HostRunExecutionModeRun] != results[types.HostRunExecutionModeStream] {
		t.Fatalf("Run/Stream normalized outcomes diverged: %#v", results)
	}
}

func TestConnectionPendingClosesExactlyOnceAcrossResponseAndDisconnect(t *testing.T) {
	conn := New(nil).Connect(func(any) error { return nil })
	responses, err := conn.RegisterPending("request-1")
	if err != nil {
		t.Fatal(err)
	}
	resp := types.HostResponseEnvelope{Version: types.HostProtocolVersionV1, MessageID: "response-1", Kind: types.HostEnvelopeKindHostResponse, Time: time.Now().UTC(), RequestID: "request-1", Accepted: true}
	if err := conn.Respond(resp); err != nil {
		t.Fatal(err)
	}
	conn.Close()
	got, ok := <-responses
	if !ok || got.MessageID != resp.MessageID {
		t.Fatalf("response = %#v, ok=%v", got, ok)
	}
	if _, open := <-responses; open {
		t.Fatal("pending channel remained open")
	}
	if conn.PendingCount() != 0 {
		t.Fatalf("pending = %d", conn.PendingCount())
	}
	if err := conn.Respond(resp); !errors.Is(err, ErrPendingNotFound) {
		t.Fatalf("late response error = %v", err)
	}
}

func TestConnectionRejectsMismatchedPendingResponseCorrelation(t *testing.T) {
	requestFrames := make(chan types.HostRequestEnvelope, 1)
	conn := New(nil, WithDeliveryTimeout(time.Second)).Connect(func(frame any) error {
		if req, ok := frame.(types.HostRequestEnvelope); ok {
			requestFrames <- req
		}
		return nil
	})
	finished := make(chan error, 1)
	go func() {
		_, err := conn.Resolve(context.Background(), types.ClarificationResolveRequest{
			RunID: "run-1", SessionID: "session-1", Timeout: time.Second,
			Request: types.ClarificationRequest{RequestID: "clarify-mismatch", Questions: []string{"continue?"}},
		})
		finished <- err
	}()
	<-requestFrames
	err := conn.Respond(types.HostResponseEnvelope{Version: types.HostProtocolVersionV1, MessageID: "wrong-run", Kind: types.HostEnvelopeKindHostResponse, Time: time.Now().UTC(), RequestID: "clarify-mismatch", SessionID: "session-1", RunID: "run-other", Accepted: true})
	if !errors.Is(err, ErrPendingCorrelation) {
		t.Fatalf("mismatch error = %v", err)
	}
	if conn.PendingCount() != 1 {
		t.Fatalf("pending after mismatch = %d, want 1", conn.PendingCount())
	}
	conn.Close()
	if err := <-finished; !errors.Is(err, ErrConnectionClosed) {
		t.Fatalf("resolver error = %v", err)
	}
}

func TestWriteFailureClosesConnectionAndSettlesPending(t *testing.T) {
	writeErr := errors.New("pipe closed")
	conn := New(nil, WithDeliveryTimeout(time.Second)).Connect(func(any) error { return writeErr })
	_, err := conn.Resolve(context.Background(), types.ClarificationResolveRequest{
		RunID: "run-1", SessionID: "session-1", Timeout: time.Second,
		Request: types.ClarificationRequest{RequestID: "clarify-write", Questions: []string{"continue?"}},
	})
	if !errors.Is(err, writeErr) && !errors.Is(err, ErrConnectionClosed) {
		t.Fatalf("resolver error = %v", err)
	}
	if conn.PendingCount() != 0 {
		t.Fatalf("pending = %d", conn.PendingCount())
	}
	if _, err := conn.RegisterPending("late"); !errors.Is(err, ErrConnectionClosed) {
		t.Fatalf("register after write failure = %v", err)
	}
}

func TestClarificationResolverPreservesRequestIDAndResponse(t *testing.T) {
	requestFrames := make(chan types.HostRequestEnvelope, 1)
	conn := New(nil, WithDeliveryTimeout(100*time.Millisecond)).Connect(func(frame any) error {
		if req, ok := frame.(types.HostRequestEnvelope); ok {
			requestFrames <- req
		}
		return nil
	})
	result := make(chan types.ClarificationResponse, 1)
	go func() {
		got, _ := conn.Resolve(context.Background(), types.ClarificationResolveRequest{
			RunID: "run-1", SessionID: "session-1", Timeout: time.Second,
			Request: types.ClarificationRequest{RequestID: "clarify-1", Questions: []string{"continue?"}},
		})
		result <- got
	}()
	req := <-requestFrames
	if req.RequestID != "clarify-1" || req.RequestType != HostRequestTypeClarification {
		t.Fatalf("host request = %#v", req)
	}
	if err := conn.Respond(types.HostResponseEnvelope{Version: types.HostProtocolVersionV1, MessageID: "answer-1", Kind: types.HostEnvelopeKindHostResponse, Time: time.Now().UTC(), RequestID: "clarify-1", SessionID: "session-1", RunID: "run-1", Accepted: true, Payload: map[string]any{"answers": []any{"yes"}}}); err != nil {
		t.Fatal(err)
	}
	got := <-result
	if got.RequestID != "clarify-1" || len(got.Answers) != 1 || got.Answers[0] != "yes" {
		t.Fatalf("clarification = %#v", got)
	}
}

func TestActionGateDisconnectFailsClosedAndClearsPending(t *testing.T) {
	started := make(chan struct{})
	conn := New(nil, WithDeliveryTimeout(time.Second)).Connect(func(frame any) error {
		if _, ok := frame.(types.HostRequestEnvelope); ok {
			close(started)
		}
		return nil
	})
	result := make(chan bool, 1)
	go func() {
		allowed, _ := conn.Confirm(context.Background(), types.ActionGateConfirmRequest{
			Check: types.ActionGateCheck{RunID: "run-1", SessionID: "session-1", CallID: "call-1"}, Timeout: time.Second,
		})
		result <- allowed
	}()
	<-started
	conn.Close()
	if <-result {
		t.Fatal("action gate allowed after disconnect")
	}
	if conn.PendingCount() != 0 {
		t.Fatalf("pending = %d", conn.PendingCount())
	}
}

func TestRunnerCallbackIsBoundedWhenWriterBlocks(t *testing.T) {
	blocked := make(chan struct{})
	runnerReturned := make(chan struct{})
	r := runnerFunc(func(ctx context.Context, req types.RunRequest, h types.EventHandler) (types.RunResult, error) {
		h.OnEvent(ctx, types.Event{Version: "v1", Type: "run.progress", RunID: req.RunID, Time: time.Now().UTC()})
		close(runnerReturned)
		return types.RunResult{RunID: req.RunID}, nil
	})
	conn := New(r, WithDeliveryQueueSize(1), WithDeliveryTimeout(20*time.Millisecond)).Connect(func(frame any) error {
		if _, ok := frame.(types.HostRuntimeEventEnvelope); ok {
			<-blocked
		}
		return nil
	})
	cmd := validHostCommand(types.HostCommandKindRunStart)
	cmd.Payload = map[string]any{"input": "hello"}
	if resp, err := conn.HandleCommand(context.Background(), cmd); err != nil || resp.Status != types.HostAdmissionStatusAccepted {
		t.Fatalf("response=%#v err=%v", resp, err)
	}
	select {
	case <-runnerReturned:
	case <-time.After(100 * time.Millisecond):
		t.Fatal("runner callback blocked on writer")
	}
	close(blocked)
}

func TestRunnerCallbackReturnsWhenWriterPanicsAndConnectionFailsDeterministically(t *testing.T) {
	runnerReturned := make(chan struct{})
	r := runnerFunc(func(ctx context.Context, req types.RunRequest, h types.EventHandler) (types.RunResult, error) {
		h.OnEvent(ctx, types.Event{Version: "v1", Type: "run.progress", RunID: req.RunID, Time: time.Now().UTC()})
		close(runnerReturned)
		return types.RunResult{RunID: req.RunID}, nil
	})
	writer := &panicOnRuntimeEventFrameWriter{eventStarted: make(chan struct{})}
	conn := New(r, WithDeliveryTimeout(100*time.Millisecond)).ConnectFrameWriter(writer)
	cmd := validHostCommand(types.HostCommandKindRunStart)
	if resp, err := conn.HandleCommand(context.Background(), cmd); err != nil || resp.Status != types.HostAdmissionStatusAccepted {
		t.Fatalf("response=%#v err=%v", resp, err)
	}
	select {
	case <-runnerReturned:
	case <-time.After(time.Second):
		t.Fatal("Runner callback remained blocked after writer panic")
	}
	select {
	case <-writer.eventStarted:
	case <-time.After(time.Second):
		t.Fatal("panicking writer was not exercised")
	}
	deadline := time.Now().Add(time.Second)
	for !conn.isClosed() && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if !conn.isClosed() {
		t.Fatal("writer panic did not close the connection")
	}
	if _, err := conn.HandleCommand(context.Background(), validHostCommand(types.HostCommandKindRunGet)); !errors.Is(err, ErrConnectionClosed) {
		t.Fatalf("post-panic command error=%v, want connection closed", err)
	}
}

func TestRunnerLowPriorityDeliveryWithoutSourcePolicyFailsInsteadOfDropping(t *testing.T) {
	// Runner callbacks expose only types.EventHandler; they carry no source-owned
	// delivery policy, so low-priority drop-with-record is intentionally
	// unsupported at this adapter boundary. Such events remain critical and a
	// saturated connection fails deterministically rather than silently dropping.
	writer := &blockingFrameWriter{started: make(chan struct{}), exited: make(chan struct{})}
	conn := New(nil, WithDeliveryQueueSize(1), WithDeliveryTimeout(20*time.Millisecond)).ConnectFrameWriter(writer)
	event := types.Event{Version: "v1", Type: "run.progress", RunID: "run-1", Time: time.Now().UTC()}
	firstDone := make(chan error, 1)
	go func() {
		firstDone <- conn.emitEvent(context.Background(), validHostCommand(types.HostCommandKindRunStart), event)
	}()
	<-writer.started
	if err := conn.emitEvent(context.Background(), validHostCommand(types.HostCommandKindRunStart), event); !errors.Is(err, ErrDeliveryTimeout) {
		t.Fatalf("second low-priority event error=%v, want delivery timeout", err)
	}
	if !conn.isClosed() {
		t.Fatal("saturated critical delivery did not close the connection")
	}
	if err := <-firstDone; !errors.Is(err, ErrDeliveryTimeout) {
		t.Fatalf("first event error=%v, want delivery timeout after saturation", err)
	}
}

func TestProjectionCompleteRemainsCriticalUnderDropWithRecord(t *testing.T) {
	collector := &factCollector{}
	writer := &blockingFrameWriter{started: make(chan struct{}), exited: make(chan struct{})}
	conn := New(nil,
		WithEventSink(collector),
		WithDeliveryQueueSize(1),
		WithDeliveryTimeout(20*time.Millisecond),
	).ConnectFrameWriter(writer)
	done := make(chan struct{})
	go func() {
		conn.emitProjection(validHostCommand(types.HostCommandKindEventsSubscribe), recordedDropProjection(types.RealtimeEventTypeComplete))
		close(done)
	}()
	select {
	case <-writer.started:
	case <-time.After(time.Second):
		t.Fatal("complete event writer did not start")
	}
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("critical complete delivery did not reach its bounded failure")
	}
	if !conn.isClosed() {
		t.Fatal("failed critical complete delivery did not close the connection")
	}
	for _, event := range collector.snapshot() {
		if event.Type == types.EventTypeHostDelivery && event.Payload["fact"] == "dropped" {
			t.Fatalf("critical complete event was recorded as dropped: %#v", event)
		}
	}
	select {
	case <-writer.exited:
	case <-time.After(time.Second):
		t.Fatal("critical writer did not exit after delivery timeout")
	}
}

func TestNonCriticalDeliveryReleasesDeadlineAfterWrite(t *testing.T) {
	writer := &contextCapturingFrameWriter{contexts: make(chan context.Context, 1)}
	conn := New(nil, WithDeliveryTimeout(time.Second)).ConnectFrameWriter(writer)
	if err := conn.deliver(context.Background(), "low-priority", false); err != nil {
		t.Fatalf("non-critical delivery error = %v", err)
	}
	var writeCtx context.Context
	select {
	case writeCtx = <-writer.contexts:
	case <-time.After(time.Second):
		t.Fatal("non-critical delivery was not written")
	}
	select {
	case <-writeCtx.Done():
	case <-time.After(100 * time.Millisecond):
		t.Fatal("successful non-critical delivery retained its deadline")
	}
	conn.Close()
}

func TestAcceptedRunIsNotCanceledByCommandContextOrDisconnect(t *testing.T) {
	runStarted := make(chan struct{})
	runCanceled := make(chan struct{}, 1)
	release := make(chan struct{})
	r := runnerFunc(func(ctx context.Context, req types.RunRequest, h types.EventHandler) (types.RunResult, error) {
		close(runStarted)
		select {
		case <-ctx.Done():
			runCanceled <- struct{}{}
		case <-release:
		}
		return types.RunResult{RunID: req.RunID}, nil
	})
	conn := New(r).Connect(func(any) error { return nil })
	commandCtx, cancelCommand := context.WithCancel(context.Background())
	cmd := validHostCommand(types.HostCommandKindRunStart)
	if resp, err := conn.HandleCommand(commandCtx, cmd); err != nil || resp.Status != types.HostAdmissionStatusAccepted {
		t.Fatalf("response=%#v err=%v", resp, err)
	}
	<-runStarted
	cancelCommand()
	conn.Close()
	select {
	case <-runCanceled:
		t.Fatal("command transport cancellation reached business Run")
	case <-time.After(20 * time.Millisecond):
	}
	close(release)
}

type runnerFunc func(context.Context, types.RunRequest, types.EventHandler) (types.RunResult, error)

type runnerExecution struct {
	runner types.Runner
	ctx    context.Context
	req    types.RunRequest
	mode   types.HostRunExecutionMode
}

func (e runnerExecution) Execute(h types.EventHandler) (types.RunResult, error) {
	if e.mode == types.HostRunExecutionModeStream {
		return e.runner.Stream(e.ctx, e.req, h)
	}
	return e.runner.Run(e.ctx, e.req, h)
}

func (runnerExecution) Release() {}

func (f runnerFunc) Run(ctx context.Context, req types.RunRequest, h types.EventHandler) (types.RunResult, error) {
	return f(ctx, req, h)
}

func (f runnerFunc) Stream(ctx context.Context, req types.RunRequest, h types.EventHandler) (types.RunResult, error) {
	return f(ctx, req, h)
}

func (f runnerFunc) AdmitHostRun(ctx context.Context, req types.RunRequest, mode types.HostRunExecutionMode) (types.HostRunStartAdmission, error) {
	return types.HostRunStartAdmission{Status: types.HostAdmissionStatusAccepted, RunID: req.RunID, Execution: runnerExecution{runner: f, ctx: context.WithoutCancel(ctx), req: req, mode: mode}}, nil
}
