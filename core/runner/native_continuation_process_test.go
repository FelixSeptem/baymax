package runner

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"sync/atomic"
	"testing"
	"time"

	"github.com/FelixSeptem/baymax/core/checkpoint"
	"github.com/FelixSeptem/baymax/core/types"
	"github.com/FelixSeptem/baymax/tool/local"
)

func TestNativeContinuationAdmissionAndConcurrentClaim(t *testing.T) {
	store, err := checkpoint.NewMemoryStore(checkpoint.Options{Namespace: "runner", Capacity: 8})
	if err != nil {
		t.Fatal(err)
	}
	binding := NativeContinuationOptions{Store: store, OwnerNamespace: "runner", DependencyProfile: "offline", DependencyRevision: "v1"}
	e, _, _ := newDynamicActionResumeEventScenario(t, false, WithNativeContinuation(binding))
	ref, err := e.NativeCheckpoint(context.Background(), "run-events")
	if err != nil {
		t.Fatal(err)
	}
	request := types.HostResumeRequest{NativeCheckpoint: &ref, Decision: types.DynamicActionDecision{Decision: types.DynamicActionDecisionConfirm, Token: "opaque-events", RunID: ref.RunID, SessionID: ref.SessionID, CheckpointID: ref.CheckpointID, CheckpointVersion: ref.SchemaVersion, CheckpointDigest: ref.Digest, IdempotencyKey: "shared"}}
	denied := New(&fakeModel{}, WithNativeContinuation(binding), WithHostResumeVerifier(types.HostResumeVerifierFunc(func(context.Context, types.HostResumeRequest) (types.HostResumeResult, error) {
		return types.HostResumeResult{}, errors.New("deny")
	})))
	if _, err := denied.ResumeFromCheckpoint(context.Background(), request, nil, false); !errors.Is(err, types.HostResumeRevalidationFailed) {
		t.Fatalf("denial=%v", err)
	}
	// Denial was outside source locks and did not consume the resume identity.
	if _, err := store.Load(context.Background(), ref); err != nil {
		t.Fatal(err)
	}
	var modelCalls atomic.Int32
	model := &fakeModel{generate: func(_ context.Context, r types.ModelRequest) (types.ModelResponse, error) {
		modelCalls.Add(1)
		return types.ModelResponse{FinalAnswer: "ok"}, nil
	}}
	fresh := New(model, WithNativeContinuation(binding))
	start := make(chan struct{})
	results := make(chan error, 2)
	for range 2 {
		go func() {
			<-start
			_, err := fresh.ResumeFromCheckpoint(context.Background(), request, nil, false)
			results <- err
		}()
	}
	close(start)
	first, second := <-results, <-results
	if first != nil && second != nil {
		t.Fatalf("both resumes failed: %v / %v", first, second)
	}
	if modelCalls.Load() != 1 {
		t.Fatalf("duplicate executed model %d times", modelCalls.Load())
	}
	conflict := request
	conflict.Decision.IdempotencyKey = "other"
	if _, err := fresh.ResumeFromCheckpoint(context.Background(), conflict, nil, false); !errors.Is(err, types.ErrCheckpointConflict) {
		t.Fatalf("conflict=%v", err)
	}
}

func TestNativeContinuationUnavailableAndExpiredFailClosed(t *testing.T) {
	for _, mode := range []string{"missing", "expired"} {
		t.Run(mode, func(t *testing.T) {
			now := time.Now()
			store, err := checkpoint.NewMemoryStore(checkpoint.Options{Namespace: "runner", Capacity: 8, Now: func() time.Time { return now }})
			if err != nil {
				t.Fatal(err)
			}
			binding := NativeContinuationOptions{Store: store, OwnerNamespace: "runner", DependencyProfile: "offline", DependencyRevision: "v1"}
			e, _, _ := newDynamicActionResumeEventScenario(t, false, WithNativeContinuation(binding))
			ref, err := e.NativeCheckpoint(context.Background(), "run-events")
			if err != nil {
				t.Fatal(err)
			}
			if mode == "expired" {
				now = now.Add(48 * time.Hour)
			}
			calls := 0
			fresh := New(&fakeModel{generate: func(context.Context, types.ModelRequest) (types.ModelResponse, error) {
				calls++
				return types.ModelResponse{}, nil
			}}, WithNativeContinuation(binding))
			bad := ref
			if mode == "missing" {
				bad.CheckpointID = "missing"
			}
			request := types.HostResumeRequest{NativeCheckpoint: &bad, Decision: types.DynamicActionDecision{Decision: types.DynamicActionDecisionConfirm, Token: "opaque-events", RunID: ref.RunID, SessionID: ref.SessionID, CheckpointID: bad.CheckpointID, CheckpointVersion: bad.SchemaVersion, CheckpointDigest: bad.Digest, IdempotencyKey: "x"}}
			_, err = fresh.ResumeFromCheckpoint(context.Background(), request, nil, false)
			if calls != 0 || err == nil {
				t.Fatalf("calls=%d err=%v", calls, err)
			}
		})
	}
}

// Separate OS processes are essential here: a fresh Engine in the same test
// process cannot prove that no process-owned continuation was used.
func TestNativeContinuationCrossProcess(t *testing.T) {
	if os.Getenv("BAYMAX_CONTINUATION_HELPER") == "1" {
		nativeContinuationProcess(t)
		return
	}
	for _, reason := range []types.PauseReason{types.PauseReasonInputRequired, types.PauseReasonApprovalRequired, types.PauseReasonBudgetExhausted} {
		for _, stream := range []bool{false, true} {
			t.Run(string(reason)+"/"+strconv.FormatBool(stream), func(t *testing.T) {
				dir := t.TempDir()
				for _, stage := range []string{"pause", "restore", "duplicate"} {
					cmd := exec.Command(os.Args[0], "-test.run=^TestNativeContinuationCrossProcess$", "-test.v")
					cmd.Env = append(os.Environ(), "BAYMAX_CONTINUATION_HELPER=1", "BAYMAX_CONTINUATION_DIR="+dir, "BAYMAX_CONTINUATION_REASON="+string(reason), "BAYMAX_CONTINUATION_STREAM="+strconv.FormatBool(stream), "BAYMAX_CONTINUATION_STAGE="+stage)
					if output, err := cmd.CombinedOutput(); err != nil {
						t.Fatalf("%s: %v\n%s", stage, err, output)
					}
				}
			})
		}
	}
}

func nativeContinuationProcess(t *testing.T) {
	dir := os.Getenv("BAYMAX_CONTINUATION_DIR")
	stage := os.Getenv("BAYMAX_CONTINUATION_STAGE")
	reason := types.PauseReason(os.Getenv("BAYMAX_CONTINUATION_REASON"))
	stream, _ := strconv.ParseBool(os.Getenv("BAYMAX_CONTINUATION_STREAM"))
	store, err := checkpoint.OpenFileStore(filepath.Join(dir, "source"), checkpoint.Options{Namespace: "runner", Capacity: 16})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = store.Close() }()
	var tools, models atomic.Int32
	reg := local.NewRegistry()
	_, err = reg.Register(&fakeTool{name: "pending", invoke: func(_ context.Context, args map[string]any) (types.ToolResult, error) {
		tools.Add(1)
		if stage == "duplicate" || reason == types.PauseReasonInputRequired && stage != "pause" {
			t.Error("completed tool reexecuted")
		}
		if reason == types.PauseReasonInputRequired {
			return types.ToolResult{PendingAction: &types.DynamicActionReference{Token: "opaque", Kind: "input", Resumable: true, RunID: "run", SessionID: "session", Iteration: 1, CallID: "one", Source: "offline", Digest: "action"}}, nil
		}
		return types.ToolResult{Content: "done"}, nil
	}})
	if err != nil {
		t.Fatal(err)
	}
	calls := []types.ToolCall{{CallID: "one", Name: "local.pending"}}
	if reason == types.PauseReasonBudgetExhausted {
		calls = append(calls, types.ToolCall{CallID: "two", Name: "local.pending"})
	}
	model := &fakeModel{generate: func(_ context.Context, r types.ModelRequest) (types.ModelResponse, error) {
		models.Add(1)
		if stage == "duplicate" {
			t.Error("duplicate called model")
		}
		if r.RunID != "run" {
			t.Error("run identity changed")
		}
		if stage == "pause" {
			return types.ModelResponse{ToolCalls: calls}, nil
		}
		if len(r.ToolResult) == 0 {
			t.Error("missing retained outcomes")
		}
		return types.ModelResponse{FinalAnswer: "restored"}, nil
	}, stream: func(_ context.Context, r types.ModelRequest, emit func(types.ModelEvent) error) error {
		models.Add(1)
		if stage == "duplicate" {
			t.Error("duplicate streamed model")
		}
		if r.RunID != "run" {
			t.Error("run identity changed")
		}
		if stage == "pause" {
			for _, call := range calls {
				if err := emit(types.ModelEvent{Type: types.ModelEventTypeToolCall, ToolCall: &call}); err != nil {
					return err
				}
			}
			return nil
		}
		if len(r.ToolResult) == 0 {
			t.Error("missing retained outcomes")
		}
		return emit(types.ModelEvent{Type: types.ModelEventTypeFinalAnswer, TextDelta: "restored"})
	}}
	binding := NativeContinuationOptions{Store: store, OwnerNamespace: "runner", DependencyProfile: "offline", DependencyRevision: "v1", Approval: true, Budget: true, Binding: types.NativeCheckpointRef{ParentRunID: "parent", HostCorrelation: "host", AuthorityDigest: "authority", PlanDigest: "plan", WriteScopeDigest: "scope", OccurrenceID: "occurrence", EffectNamespace: "effects"}, BudgetOwner: func(_ context.Context, _ types.NativeCheckpointRef, _ types.DurableBudget, b types.DurableBudget) (types.DurableBudget, error) {
		return b, nil
	}}
	opts := []Option{WithNativeContinuation(binding), WithLocalRegistry(reg)}
	if reason == types.PauseReasonApprovalRequired {
		opts = append(opts, WithActionGateMatcher(alwaysRequireConfirmation{}))
	}
	e := New(model, opts...)
	path := filepath.Join(dir, "request.json")
	if stage == "pause" {
		policy := types.DefaultLoopPolicy()
		if reason == types.PauseReasonBudgetExhausted {
			policy.ToolCallLimit = 1
		}
		req := types.RunRequest{RunID: "run", SessionID: "session", Input: "pause", Policy: &policy}
		var r types.RunResult
		if stream {
			r, err = e.Stream(context.Background(), req, nil)
		} else {
			r, err = e.Run(context.Background(), req, nil)
		}
		if err != nil || r.NativeCheckpoint == nil || r.NativeCheckpoint.PauseReason != reason {
			t.Fatalf("pause=%+v err=%v", r, err)
		}
		if _, err := e.ContinuationCapabilities(context.Background()); err != nil {
			t.Fatalf("complete source capability unavailable: %v", err)
		}
		if r.NativeCheckpoint.ParentRunID != "parent" || r.NativeCheckpoint.EffectNamespace != "effects" {
			t.Fatal("lost causal binding")
		}
		request := nativeResumeRequest(r, "same-request")
		if reason == types.PauseReasonBudgetExhausted {
			p, err := store.Load(context.Background(), *r.NativeCheckpoint)
			if err != nil {
				t.Fatal(err)
			}
			b := p.Budget
			b.Revision = "budget-v2"
			b.ToolLimit = 4
			request.NativeBudget = &b
		}
		data, err := json.Marshal(request)
		if err != nil {
			t.Fatal(err)
		}
		if err = os.WriteFile(path, data, 0600); err != nil {
			t.Fatal(err)
		}
		want := int32(0)
		if reason == types.PauseReasonInputRequired {
			want = 1
		}
		if models.Load() != 1 || tools.Load() != want {
			t.Fatalf("pause models=%d tools=%d", models.Load(), tools.Load())
		}
		return
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var request types.HostResumeRequest
	if err = json.Unmarshal(data, &request); err != nil {
		t.Fatal(err)
	}
	r, err := e.ResumeFromCheckpoint(context.Background(), request, nil, stream)
	if err != nil || r.RunID != "run" || r.Iterations != 2 || r.FinalAnswer != "restored" || r.TerminalOutcome == nil || r.TerminalOutcome.State != types.RunStateCompleted {
		t.Fatalf("restore=%+v err=%v", r, err)
	}
	wantTools, wantModels := int32(0), int32(0)
	if stage == "restore" {
		wantModels = 1
		if reason != types.PauseReasonInputRequired {
			wantTools = int32(len(calls))
		}
	}
	if models.Load() != wantModels || tools.Load() != wantTools {
		t.Fatalf("models=%d tools=%d", models.Load(), tools.Load())
	}
	if stage == "duplicate" {
		if _, err := e.ContinuationCapabilities(context.Background()); err != nil {
			t.Fatal(err)
		}
	}
}

func nativeResumeRequest(r types.RunResult, key string) types.HostResumeRequest {
	ref := *r.NativeCheckpoint
	return types.HostResumeRequest{NativeCheckpoint: &ref, Decision: types.DynamicActionDecision{Decision: types.DynamicActionDecisionConfirm, Token: r.TerminalOutcome.PendingAction.Token, RunID: ref.RunID, SessionID: ref.SessionID, CheckpointID: ref.CheckpointID, CheckpointVersion: ref.SchemaVersion, CheckpointDigest: ref.Digest, IdempotencyKey: key}}
}
