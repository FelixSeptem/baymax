// This example uses synthetic execution data and a local single-writer store.
package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync/atomic"

	"github.com/FelixSeptem/baymax/adapter/modelcapability"
	"github.com/FelixSeptem/baymax/core/checkpoint"
	"github.com/FelixSeptem/baymax/core/runner"
	"github.com/FelixSeptem/baymax/core/types"
	"github.com/FelixSeptem/baymax/orchestration/planbridge"
	"github.com/FelixSeptem/baymax/tool/local"
)

type fixture struct {
	reason types.PauseReason
	models atomic.Int32
	tools  atomic.Int32
}

func (f *fixture) Generate(_ context.Context, r types.ModelRequest) (types.ModelResponse, error) {
	f.models.Add(1)
	if r.RunID != "original-run" {
		return types.ModelResponse{}, errors.New("run identity changed")
	}
	if len(r.ToolResult) != 0 {
		return types.ModelResponse{FinalAnswer: "restored"}, nil
	}
	calls := []types.ToolCall{{CallID: "one", Name: "local.prepare"}}
	if f.reason == types.PauseReasonBudgetExhausted {
		calls = append(calls, types.ToolCall{CallID: "two", Name: "local.prepare"})
	}
	return types.ModelResponse{ToolCalls: calls}, nil
}

func (f *fixture) Stream(ctx context.Context, r types.ModelRequest, emit func(types.ModelEvent) error) error {
	response, err := f.Generate(ctx, r)
	if err != nil {
		return err
	}
	for _, call := range response.ToolCalls {
		if err := emit(types.ModelEvent{Type: types.ModelEventTypeToolCall, ToolCall: &call}); err != nil {
			return err
		}
	}
	if response.FinalAnswer != "" {
		return emit(types.ModelEvent{Type: types.ModelEventTypeFinalAnswer, TextDelta: response.FinalAnswer})
	}
	return nil
}

type prepareTool struct{ f *fixture }

func (*prepareTool) Name() string               { return "prepare" }
func (*prepareTool) Description() string        { return "synthetic continuation boundary" }
func (*prepareTool) JSONSchema() map[string]any { return map[string]any{"type": "object"} }
func (t *prepareTool) Invoke(context.Context, map[string]any) (types.ToolResult, error) {
	t.f.tools.Add(1)
	if t.f.reason == types.PauseReasonInputRequired {
		return types.ToolResult{PendingAction: &types.DynamicActionReference{Token: "opaque-action", Kind: "input", Resumable: true, RunID: "original-run", SessionID: "original-session", Iteration: 1, CallID: "one", Source: "offline", Digest: "action-digest"}}, nil
	}
	return types.ToolResult{Content: "retained outcome"}, nil
}

type approvalGate struct{}

func (approvalGate) Evaluate(context.Context, types.ActionGateCheck) (types.ActionGateDecision, error) {
	return types.ActionGateDecisionRequireConfirm, nil
}

func engine(f *fixture, store *checkpoint.Store) (*runner.Engine, error) {
	model, err := modelcapability.Wrap(f, modelcapability.Config{Provider: "offline", Capabilities: []types.ModelCapability{types.ModelCapabilityStreaming, types.ModelCapabilityToolCall}})
	if err != nil {
		return nil, err
	}
	registry := local.NewRegistry()
	if _, err := registry.Register(&prepareTool{f}); err != nil {
		return nil, err
	}
	options := runner.NativeContinuationOptions{
		Store: store, OwnerNamespace: "offline-runner", DependencyProfile: "offline", DependencyRevision: "v1", Approval: true, Budget: true,
		Binding: types.NativeCheckpointRef{ParentRunID: "parent", HostCorrelation: "host", AuthorityDigest: "authority", PlanDigest: "plan", WriteScopeDigest: "scope", OccurrenceID: "occurrence", EffectNamespace: "effects"},
		BudgetOwner: func(_ context.Context, _ types.NativeCheckpointRef, _ types.DurableBudget, proposed types.DurableBudget) (types.DurableBudget, error) {
			// A real source owner checks current authoritative budget here.
			return proposed, nil
		},
	}
	opts := []runner.Option{runner.WithLocalRegistry(registry), runner.WithNativeContinuation(options)}
	if f.reason == types.PauseReasonApprovalRequired {
		opts = append(opts, runner.WithActionGateMatcher(approvalGate{}))
	}
	return runner.New(model, opts...), nil
}

func exercise(reason types.PauseReason, stream bool) error {
	ctx := context.Background()
	dir, err := os.MkdirTemp("", "baymax-continuation-")
	if err != nil {
		return err
	}
	defer func() { _ = os.RemoveAll(dir) }()
	store, err := checkpoint.OpenFileStore(dir, checkpoint.Options{Namespace: "offline-runner", Capacity: 8})
	if err != nil {
		return err
	}
	defer func() { _ = store.Close() }()
	f := &fixture{reason: reason}
	e, err := engine(f, store)
	if err != nil {
		return err
	}
	policy := types.DefaultLoopPolicy()
	if reason == types.PauseReasonBudgetExhausted {
		policy.ToolCallLimit = 1
	}
	req := types.RunRequest{RunID: "original-run", SessionID: "original-session", Input: "pause", Policy: &policy}
	var paused types.RunResult
	if stream {
		paused, err = e.Stream(ctx, req, nil)
	} else {
		paused, err = e.Run(ctx, req, nil)
	}
	if err != nil {
		return err
	}
	if paused.NativeCheckpoint == nil || paused.NativeCheckpoint.PauseReason != reason || paused.TerminalOutcome == nil || paused.TerminalOutcome.PendingAction == nil {
		return errors.New("source pause missing")
	}
	ref := *paused.NativeCheckpoint
	request := types.HostResumeRequest{NativeCheckpoint: &ref, Decision: types.DynamicActionDecision{Decision: types.DynamicActionDecisionConfirm, Token: paused.TerminalOutcome.PendingAction.Token, RunID: ref.RunID, SessionID: ref.SessionID, CheckpointID: ref.CheckpointID, CheckpointVersion: ref.SchemaVersion, CheckpointDigest: ref.Digest, IdempotencyKey: "same-request"}}
	if reason == types.PauseReasonBudgetExhausted {
		p, err := store.Load(ctx, ref)
		if err != nil {
			return err
		}
		budget := p.Budget
		budget.Revision, budget.ToolLimit = "budget-v2", 4
		request.NativeBudget = &budget
	}
	if err = store.Close(); err != nil {
		return err
	}
	if _, err = e.ResumeFromCheckpoint(ctx, request, nil, stream); !errors.Is(err, types.ErrCheckpointUnavailable) || f.models.Load() != 1 {
		return errors.New("closed source did not fail before execution")
	}
	store, err = checkpoint.OpenFileStore(dir, checkpoint.Options{Namespace: "offline-runner", Capacity: 8})
	if err != nil {
		return err
	}
	fresh, err := engine(f, store)
	if err != nil {
		return err
	}
	foreign := request.Detached()
	foreign.NativeCheckpoint.OwnerNamespace = "foreign"
	if _, err := fresh.ResumeFromCheckpoint(ctx, foreign, nil, stream); !errors.Is(err, types.ErrCheckpointForeign) {
		return fmt.Errorf("foreign checkpoint: %w", err)
	}
	r, err := fresh.ResumeFromCheckpoint(ctx, request, nil, stream)
	if err != nil {
		return err
	}
	wantTools := int32(1)
	if reason == types.PauseReasonBudgetExhausted {
		wantTools = 2
	}
	if r.RunID != ref.RunID || r.TerminalOutcome == nil || r.TerminalOutcome.SessionID != ref.SessionID || r.TerminalOutcome.State != types.RunStateCompleted || r.FinalAnswer != "restored" || ref.ParentRunID != "parent" || ref.EffectNamespace != "effects" || f.models.Load() != 2 || f.tools.Load() != wantTools {
		return errors.New("restore lost source identity or reexecuted completed work")
	}
	if _, err = fresh.ResumeFromCheckpoint(ctx, request, nil, stream); err != nil || f.models.Load() != 2 || f.tools.Load() != wantTools {
		return errors.New("duplicate reexecuted source")
	}
	conflict := request.Detached()
	conflict.Decision.IdempotencyKey = "conflict"
	if _, err := fresh.ResumeFromCheckpoint(ctx, conflict, nil, stream); !errors.Is(err, types.ErrCheckpointConflict) {
		return errors.New("conflicting request was admitted")
	}
	manifest, err := planbridge.CapabilityManifestWithContinuation(ctx, fresh)
	if err != nil {
		return err
	}
	for _, form := range manifest.Forms {
		if form.Form != planbridge.ExecutionFormDirect && (form.CrossProcessRestore != planbridge.CapabilityUnsupported || form.InputRequired != planbridge.CapabilityUnsupported || form.ApprovalRequired != planbridge.CapabilityUnsupported || form.BudgetExhausted != planbridge.CapabilityUnsupported) {
			return errors.New("composite restore was promoted")
		}
	}
	mode := map[bool]string{false: "run", true: "stream"}[stream]
	fmt.Printf("%s_%s_restored\n", reason, mode)
	return nil
}

func main() {
	for _, reason := range []types.PauseReason{types.PauseReasonInputRequired, types.PauseReasonApprovalRequired, types.PauseReasonBudgetExhausted} {
		for _, stream := range []bool{false, true} {
			if err := exercise(reason, stream); err != nil {
				panic(err)
			}
		}
	}
	for _, marker := range []string{"duplicate_projection", "conflict_rejected", "foreign_rejected", "store_unavailable", "composite_unsupported", "run_stream_parity"} {
		fmt.Println(marker)
	}
}
