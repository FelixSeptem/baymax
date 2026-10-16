package integration

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/FelixSeptem/baymax/adapter/modelcapability"
	"github.com/FelixSeptem/baymax/core/runner"
	"github.com/FelixSeptem/baymax/core/types"
	"github.com/FelixSeptem/baymax/host"
	"github.com/FelixSeptem/baymax/tool/local"
)

type resumeHostFixture struct {
	modelCalls, toolCalls, verifications, effects int
	revision, authority, effect                   string
	current                                       types.HostResumeFacts
	binding                                       types.HostResumeBinding
	onContinuation                                func()
	continuationError                             error
}

func (f *resumeHostFixture) Generate(_ context.Context, r types.ModelRequest) (types.ModelResponse, error) {
	f.modelCalls++
	if len(r.ToolResult) == 0 {
		return types.ModelResponse{ToolCalls: []types.ToolCall{{CallID: "prepare-call", Name: "local.prepare"}}}, nil
	}
	if r.ToolResult[0].Result.PendingAction == nil {
		return types.ModelResponse{}, fmt.Errorf("checkpoint tool result missing")
	}
	if f.onContinuation != nil {
		f.onContinuation()
	}
	if f.continuationError != nil {
		return types.ModelResponse{}, f.continuationError
	}
	// An independent host CAS guards the business effect after admission.
	if f.revision == f.binding.BusinessRevision && f.authority == f.binding.AuthorityDigest && f.effect == f.binding.EffectReference {
		f.effects++
		f.effect = "consumed"
	}
	return types.ModelResponse{FinalAnswer: "runtime completed"}, nil
}
func (f *resumeHostFixture) Stream(ctx context.Context, r types.ModelRequest, emit func(types.ModelEvent) error) error {
	x, err := f.Generate(ctx, r)
	if err != nil {
		return err
	}
	if len(x.ToolCalls) > 0 {
		return emit(types.ModelEvent{Type: types.ModelEventTypeToolCall, ToolCall: &x.ToolCalls[0]})
	}
	return emit(types.ModelEvent{Type: types.ModelEventTypeFinalAnswer, TextDelta: x.FinalAnswer})
}

type resumePrepareTool struct{ fixture *resumeHostFixture }

func (*resumePrepareTool) Name() string               { return "prepare" }
func (*resumePrepareTool) Description() string        { return "offline pending action" }
func (*resumePrepareTool) JSONSchema() map[string]any { return map[string]any{"type": "object"} }
func (t *resumePrepareTool) Invoke(context.Context, map[string]any) (types.ToolResult, error) {
	t.fixture.toolCalls++
	return types.ToolResult{PendingAction: &types.DynamicActionReference{Token: "action", Kind: "prepare", Resumable: true, RunID: "resume-run", SessionID: "resume-session", Iteration: 1, CallID: "prepare-call", Source: "fixture", Digest: "action-digest"}}, nil
}

func newResumeHostFixture(t *testing.T, stream bool) (*runner.Engine, *resumeHostFixture, types.DynamicActionDecision) {
	t.Helper()
	f := &resumeHostFixture{revision: "revision", authority: "authority", effect: "effect"}
	f.current = types.HostResumeFacts{Plan: types.HostResumePlan{ID: "plan", Version: 1, Digest: "plan-digest"}, BusinessRevision: f.revision, AuthorityDigest: f.authority, ToolCapabilityDigest: "tools", WriteScopeDigest: "scope", ConfirmationReference: "confirmation", EffectReference: f.effect, DecisionAuthorized: true, ExecutionAuthorized: true, CancellationAuthorized: true, ConfirmationValid: true, EffectEligible: true}
	m, err := modelcapability.Wrap(f, modelcapability.Config{Provider: "offline", Capabilities: []types.ModelCapability{types.ModelCapabilityStreaming, types.ModelCapabilityToolCall}})
	if err != nil {
		t.Fatal(err)
	}
	reg := local.NewRegistry()
	if _, err = reg.Register(&resumePrepareTool{f}); err != nil {
		t.Fatal(err)
	}
	v := types.HostResumeVerifierFunc(func(context.Context, types.HostResumeRequest) (types.HostResumeResult, error) {
		f.verifications++
		return types.HostResumeResult{Version: types.HostResumeAdmissionVersion, Status: types.RunResumeAdmissionAccepted, Binding: f.binding, Current: f.current}, nil
	})
	e := runner.New(m, runner.WithLocalRegistry(reg), runner.WithHostResumeVerifier(v))
	req := types.RunRequest{RunID: "resume-run", SessionID: "resume-session", Input: "prepare"}
	var paused types.RunResult
	if stream {
		paused, err = e.Stream(context.Background(), req, nil)
	} else {
		paused, err = e.Run(context.Background(), req, nil)
	}
	if err != nil || paused.TerminalOutcome == nil || paused.TerminalOutcome.State != types.RunStateInputRequired {
		t.Fatalf("pause=%+v err=%v", paused, err)
	}
	cp := paused.TerminalOutcome
	// Persist trusted application references at the pause boundary; never take
	// plan/revision/authority from a later command or callback request.
	f.binding = types.HostResumeBinding{Version: types.HostResumeAdmissionVersion, Checkpoint: types.RunCheckpoint{Version: cp.CheckpointVersion, CheckpointID: cp.CheckpointID, RunID: req.RunID, SessionID: req.SessionID, State: types.RunStateInputRequired, Mode: map[bool]string{true: "stream", false: "run"}[stream], Iteration: paused.Iterations, PendingAction: cp.PendingAction, Digest: cp.CheckpointDigest}, Plan: f.current.Plan, BusinessRevision: f.revision, AuthorityDigest: f.authority, ToolCapabilityDigest: "tools", WriteScopeDigest: "scope", ConfirmationReference: "confirmation", EffectReference: f.effect}
	d := types.DynamicActionDecision{Decision: types.DynamicActionDecisionConfirm, Token: cp.PendingAction.Token, RunID: req.RunID, SessionID: req.SessionID, CheckpointID: cp.CheckpointID, CheckpointVersion: cp.CheckpointVersion, CheckpointDigest: cp.CheckpointDigest, IdempotencyKey: "resume-key", Reference: "confirmation"}
	return e, f, d
}

type resumeAuthorizer struct {
	calls   int
	allowed bool
}

func (a *resumeAuthorizer) AuthorizeHostCommand(context.Context, types.HostCommandEnvelope) error {
	a.calls++
	if !a.allowed {
		return errors.New("fixture denied result access")
	}
	return nil
}

func TestHostResumeNativeSourceGuardAndResponseBarrier(t *testing.T) {
	for _, stream := range []bool{false, true} {
		t.Run(fmt.Sprint(stream), func(t *testing.T) {
			e, f, d := newResumeHostFixture(t, stream)
			auth := &resumeAuthorizer{allowed: true}
			var frames []any
			conn := host.New(e, host.WithAuthorization(auth)).Connect(func(frame any) error { frames = append(frames, frame); return nil })
			cmd := types.HostCommandEnvelope{Version: types.HostProtocolVersionV1, MessageID: "denied-message", RequestID: "resume-request", Kind: types.HostCommandKindActionResume, Time: time.Now().UTC(), RunID: d.RunID, SessionID: d.SessionID, Payload: map[string]any{"decision": string(d.Decision), "token": d.Token, "checkpoint_id": d.CheckpointID, "checkpoint_version": d.CheckpointVersion, "checkpoint_digest": d.CheckpointDigest, "idempotency_key": d.IdempotencyKey, "reference": d.Reference, "stream": stream}}
			f.current.DecisionAuthorized = false
			response, err := conn.HandleCommand(context.Background(), cmd)
			if err != nil || response.Status != types.HostAdmissionStatusRejected || response.ReasonCode != string(types.HostResumeAuthorityDenied) || f.modelCalls != 1 || f.toolCalls != 1 || f.effects != 0 {
				t.Fatalf("response=%+v err=%v counts=%d/%d/%d", response, err, f.modelCalls, f.toolCalls, f.effects)
			}
			f.current.DecisionAuthorized = true
			cmd.MessageID = "accepted-message"
			frames = nil
			response, err = conn.HandleCommand(context.Background(), cmd)
			if err != nil || response.Status != types.HostAdmissionStatusAccepted || f.effects != 1 || f.modelCalls != 2 || f.toolCalls != 1 {
				t.Fatalf("response=%+v err=%v", response, err)
			}
			if len(frames) < 2 {
				t.Fatal("source events missing")
			}
			if _, ok := frames[0].(types.HostCommandResponse); !ok {
				t.Fatalf("first frame=%T want response", frames[0])
			}
			before := len(frames)
			cmd.MessageID = "duplicate-message"
			if _, err = conn.HandleCommand(context.Background(), cmd); err != nil {
				t.Fatal(err)
			}
			if len(frames) != before+1 || f.verifications != 2 || f.effects != 1 {
				t.Fatal("different-message duplicate added source events/effects")
			}
			auth.allowed = false
			cmd.MessageID = "unauthorized-result-read"
			response, err = conn.HandleCommand(context.Background(), cmd)
			if response.Status != types.HostAdmissionStatusRejected || f.verifications != 2 || auth.calls != 4 {
				t.Fatalf("duplicate bypassed authorization: %+v %v", response, err)
			}
		})
	}
}

func TestHostResumeContinuationFailureKeepsAdmissionAccepted(t *testing.T) {
	for _, stream := range []bool{false, true} {
		for _, failure := range []error{errors.New("private provider error body"), context.Canceled} {
			t.Run(fmt.Sprintf("stream=%v/%v", stream, failure), func(t *testing.T) {
				e, f, d := newResumeHostFixture(t, stream)
				f.continuationError = failure
				var frames []any
				conn := host.New(e).Connect(func(frame any) error { frames = append(frames, frame); return nil })
				cmd := types.HostCommandEnvelope{Version: types.HostProtocolVersionV1, MessageID: "failed-continuation", RequestID: "request", Kind: types.HostCommandKindActionResume, Time: time.Now().UTC(), RunID: d.RunID, SessionID: d.SessionID, Payload: map[string]any{"decision": string(d.Decision), "token": d.Token, "checkpoint_id": d.CheckpointID, "checkpoint_version": d.CheckpointVersion, "checkpoint_digest": d.CheckpointDigest, "idempotency_key": d.IdempotencyKey, "reference": d.Reference, "stream": stream}}
				response, err := conn.HandleCommand(context.Background(), cmd)
				if err != nil || response.Status != types.HostAdmissionStatusAccepted || response.ReasonCode != "dynamic_action.resume_admitted" {
					t.Fatalf("accepted decision misreported: response=%+v err=%v", response, err)
				}
				if len(frames) < 2 {
					t.Fatal("runtime events missing")
				}
				if _, ok := frames[0].(types.HostCommandResponse); !ok {
					t.Fatalf("first frame=%T", frames[0])
				}
				before := len(frames)
				cmd.MessageID = "failed-continuation-duplicate"
				response, err = conn.HandleCommand(context.Background(), cmd)
				if err != nil || response.Status != types.HostAdmissionStatusAccepted || response.ReasonCode != "dynamic_action.resume_admitted" || len(frames) != before+1 || f.modelCalls != 2 || f.verifications != 1 || f.effects != 0 {
					t.Fatalf("retained failure duplicate drift: response=%+v err=%v frames=%d calls=%d/%d effects=%d", response, err, len(frames), f.modelCalls, f.verifications, f.effects)
				}
			})
		}
	}
}

func TestHostResumeBusinessEffectStillRequiresCAS(t *testing.T) {
	for _, stream := range []bool{false, true} {
		for _, change := range []string{"revision", "authority", "effect"} {
			t.Run(fmt.Sprintf("%v/%s", stream, change), func(t *testing.T) {
				e, f, d := newResumeHostFixture(t, stream)
				f.onContinuation = func() {
					switch change {
					case "revision":
						f.revision = "changed"
					case "authority":
						f.authority = "revoked"
					case "effect":
						f.effect = "conflict"
					}
				}
				r, err := e.ResumeDynamicAction(context.Background(), d, nil, stream)
				if err != nil || r.TerminalOutcome.State != types.RunStateCompleted || f.effects != 0 {
					t.Fatalf("runtime completion must not imply business commit: %+v err=%v effects=%d", r, err, f.effects)
				}
			})
		}
	}
}
