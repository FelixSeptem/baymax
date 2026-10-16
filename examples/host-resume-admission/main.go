// This offline example uses a real runner and synthetic host references.
package main

import (
	"context"
	"errors"
	"fmt"

	"github.com/FelixSeptem/baymax/adapter/modelcapability"
	"github.com/FelixSeptem/baymax/core/runner"
	"github.com/FelixSeptem/baymax/core/types"
	"github.com/FelixSeptem/baymax/tool/local"
)

type fixture struct {
	modelCalls, toolCalls, verifierCalls, effects, resolved, canceled int
	current                                                           types.HostResumeFacts
	binding                                                           types.HostResumeBinding
	revision, authority, effect                                       string
	beforeCommit                                                      func()
}

func newFixture() *fixture {
	f := &fixture{revision: "revision", authority: "authority", effect: "effect"}
	f.current = types.HostResumeFacts{Plan: types.HostResumePlan{ID: "offline-plan", Version: 1, Digest: "plan-digest"}, BusinessRevision: f.revision, AuthorityDigest: f.authority, ToolCapabilityDigest: "tools", WriteScopeDigest: "scope", ConfirmationReference: "confirmation", EffectReference: f.effect, DecisionAuthorized: true, ExecutionAuthorized: true, CancellationAuthorized: true, ConfirmationValid: true, EffectEligible: true}
	return f
}

func (f *fixture) Generate(_ context.Context, r types.ModelRequest) (types.ModelResponse, error) {
	f.modelCalls++
	if len(r.ToolResult) == 0 {
		return types.ModelResponse{ToolCalls: []types.ToolCall{{CallID: "prepare-call", Name: "local.prepare"}}}, nil
	}
	if r.ToolResult[0].Result.PendingAction == nil {
		return types.ModelResponse{}, errors.New("saved tool result missing")
	}
	if f.beforeCommit != nil {
		f.beforeCommit()
	}
	// Admission cannot replace a host CAS at the actual write boundary.
	if f.revision == f.binding.BusinessRevision && f.authority == f.binding.AuthorityDigest && f.effect == f.binding.EffectReference {
		f.effects++
		f.effect = "consumed"
	}
	return types.ModelResponse{FinalAnswer: "offline completion"}, nil
}
func (f *fixture) Stream(ctx context.Context, r types.ModelRequest, emit func(types.ModelEvent) error) error {
	x, err := f.Generate(ctx, r)
	if err != nil {
		return err
	}
	if len(x.ToolCalls) > 0 {
		return emit(types.ModelEvent{Type: types.ModelEventTypeToolCall, ToolCall: &x.ToolCalls[0]})
	}
	return emit(types.ModelEvent{Type: types.ModelEventTypeFinalAnswer, TextDelta: x.FinalAnswer})
}
func (f *fixture) OnEvent(_ context.Context, e types.Event) {
	if e.Type == "run.dynamic_action.resolved" {
		f.resolved++
	}
	if e.Type == "run.finished" && e.Payload["state"] == string(types.RunStateCanceled) {
		f.canceled++
	}
}

type prepareTool struct{ fixture *fixture }

func (*prepareTool) Name() string               { return "prepare" }
func (*prepareTool) Description() string        { return "offline PendingAction" }
func (*prepareTool) JSONSchema() map[string]any { return map[string]any{"type": "object"} }
func (t *prepareTool) Invoke(context.Context, map[string]any) (types.ToolResult, error) {
	t.fixture.toolCalls++
	return types.ToolResult{PendingAction: &types.DynamicActionReference{Token: "action", Kind: "prepare", Resumable: true, RunID: "offline-run", SessionID: "offline-session", Iteration: 1, CallID: "prepare-call", Source: "fixture", Digest: "action-digest"}}, nil
}

func pause(f *fixture, stream bool) (*runner.Engine, types.DynamicActionDecision, error) {
	m, err := modelcapability.Wrap(f, modelcapability.Config{Provider: "offline", Capabilities: []types.ModelCapability{types.ModelCapabilityStreaming, types.ModelCapabilityToolCall}})
	if err != nil {
		return nil, types.DynamicActionDecision{}, err
	}
	reg := local.NewRegistry()
	if _, err = reg.Register(&prepareTool{f}); err != nil {
		return nil, types.DynamicActionDecision{}, err
	}
	v := types.HostResumeVerifierFunc(func(context.Context, types.HostResumeRequest) (types.HostResumeResult, error) {
		f.verifierCalls++
		// Read the previously saved trusted binding and current facts. Do not
		// adopt the caller's checkpoint/digests as application authorization.
		return types.HostResumeResult{Version: types.HostResumeAdmissionVersion, Status: types.RunResumeAdmissionAccepted, Binding: f.binding, Current: f.current}, nil
	})
	e := runner.New(m, runner.WithLocalRegistry(reg), runner.WithHostResumeVerifier(v))
	req := types.RunRequest{RunID: "offline-run", SessionID: "offline-session", Input: "prepare"}
	var r types.RunResult
	if stream {
		r, err = e.Stream(context.Background(), req, f)
	} else {
		r, err = e.Run(context.Background(), req, f)
	}
	if err != nil {
		return nil, types.DynamicActionDecision{}, err
	}
	if r.TerminalOutcome == nil || r.TerminalOutcome.State != types.RunStateInputRequired {
		return nil, types.DynamicActionDecision{}, errors.New("native pause missing")
	}
	cp := r.TerminalOutcome
	c := f.current
	f.binding = types.HostResumeBinding{Version: types.HostResumeAdmissionVersion, Checkpoint: types.RunCheckpoint{Version: cp.CheckpointVersion, CheckpointID: cp.CheckpointID, RunID: req.RunID, SessionID: req.SessionID, State: types.RunStateInputRequired, Mode: map[bool]string{true: "stream", false: "run"}[stream], Iteration: r.Iterations, PendingAction: cp.PendingAction, Digest: cp.CheckpointDigest}, Plan: c.Plan, BusinessRevision: c.BusinessRevision, AuthorityDigest: c.AuthorityDigest, ToolCapabilityDigest: c.ToolCapabilityDigest, WriteScopeDigest: c.WriteScopeDigest, ConfirmationReference: c.ConfirmationReference, EffectReference: c.EffectReference}
	d := types.DynamicActionDecision{Decision: types.DynamicActionDecisionConfirm, Token: cp.PendingAction.Token, RunID: req.RunID, SessionID: req.SessionID, CheckpointID: cp.CheckpointID, CheckpointVersion: cp.CheckpointVersion, CheckpointDigest: cp.CheckpointDigest, IdempotencyKey: "resume-key", Reference: "confirmation"}
	return e, d, nil
}

func exercise(stream bool) ([]string, error) {
	ctx := context.Background()
	var markers []string
	f := newFixture()
	e, d, err := pause(f, stream)
	if err != nil {
		return nil, err
	}
	f.current.DecisionAuthorized = false
	if _, err = e.ResumeDynamicAction(ctx, d, f, stream); !errors.Is(err, types.HostResumeAuthorityDenied) || f.modelCalls != 1 || f.toolCalls != 1 || f.effects != 0 || f.resolved != 0 {
		return nil, errors.New("authority rejection changed execution")
	}
	markers = append(markers, "authority_denied")
	f.current.DecisionAuthorized = true
	entered, release := make(chan struct{}), make(chan struct{})
	f.beforeCommit = func() { close(entered); <-release }
	type outcome struct {
		result types.RunResult
		err    error
	}
	done := make(chan outcome, 1)
	go func() { r, err := e.ResumeDynamicAction(ctx, d, f, stream); done <- outcome{r, err} }()
	<-entered
	_, duplicateErr := e.ResumeDynamicAction(ctx, d, nil, stream)
	close(release)
	completed := <-done
	if !errors.Is(duplicateErr, runner.ErrDynamicActionInProgress) {
		return nil, errors.New("in-flight duplicate returned success")
	}
	markers = append(markers, "resume_in_progress")
	if completed.err != nil || completed.result.TerminalOutcome.State != types.RunStateCompleted || completed.result.Iterations != 2 || f.effects != 1 || f.resolved != 1 || f.canceled != 0 {
		return nil, errors.New("native continuation did not complete exactly once")
	}
	markers = append(markers, "resume_accepted", "completed")
	if _, err = e.ResumeDynamicAction(ctx, d, f, stream); err != nil || f.verifierCalls != 2 || f.modelCalls != 2 || f.toolCalls != 1 || f.effects != 1 || f.resolved != 1 {
		return nil, errors.New("duplicate activated source")
	}
	markers = append(markers, "duplicate_no_effect")
	g := newFixture()
	other, decision, err := pause(g, stream)
	if err != nil {
		return nil, err
	}
	g.current.BusinessRevision = "new-revision"
	if _, err = other.ResumeDynamicAction(ctx, decision, g, stream); !errors.Is(err, types.HostResumeRevisionDrift) || g.modelCalls != 1 || g.effects != 0 || g.resolved != 0 {
		return nil, errors.New("revision drift activated source")
	}
	markers = append(markers, "revision_drift")
	g.current.BusinessRevision = "revision"
	g.beforeCommit = func() { g.revision = "changed-after-admission" }
	r, err := other.ResumeDynamicAction(ctx, decision, g, stream)
	if err != nil || r.TerminalOutcome.State != types.RunStateCompleted || g.effects != 0 {
		return nil, errors.New("business CAS failed to reject stale commit")
	}
	markers = append(markers, "effect_cas_rejected")
	return markers, nil
}

func main() {
	run, err := exercise(false)
	if err != nil {
		panic(err)
	}
	stream, err := exercise(true)
	if err != nil {
		panic(err)
	}
	if fmt.Sprint(run) != fmt.Sprint(stream) {
		panic("Run/Stream semantics differ")
	}
	for _, marker := range run {
		fmt.Println(marker)
	}
	fmt.Println("run_stream_parity")
}
