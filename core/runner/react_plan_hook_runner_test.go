package runner

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/FelixSeptem/baymax/core/types"
	runtimeconfig "github.com/FelixSeptem/baymax/runtime/config"
	"github.com/FelixSeptem/baymax/tool/local"
)

func TestReactPlanHookRunStreamParityAndContext(t *testing.T) {
	mgr := newReactPlanHookRuntimeManager(t, runtimeconfig.RuntimeReactPlanChangeHookFailModeFailFast, 200)
	defer func() { _ = mgr.Close() }()

	reg := local.NewRegistry()
	if _, err := reg.Register(&fakeTool{
		name: "echo",
		invoke: func(ctx context.Context, args map[string]any) (types.ToolResult, error) {
			_ = ctx
			_ = args
			return types.ToolResult{Content: "ok"}, nil
		},
	}); err != nil {
		t.Fatalf("register tool: %v", err)
	}

	runPhases := make([]string, 0, 6)
	runHook := types.AgentLifecycleHookFunc(func(ctx context.Context, in types.AgentLifecycleHookContext) error {
		_ = ctx
		if strings.TrimSpace(in.PlanID) == "" {
			t.Fatalf("run hook missing plan_id in phase=%s", in.Phase)
		}
		runPhases = append(runPhases, fmt.Sprintf(
			"%d:%s:%d:%s:%s",
			in.Iteration,
			in.Phase,
			in.PlanVersion,
			in.PlanAction,
			in.PlanReason,
		))
		return nil
	})
	streamPhases := make([]string, 0, 6)
	streamHook := types.AgentLifecycleHookFunc(func(ctx context.Context, in types.AgentLifecycleHookContext) error {
		_ = ctx
		if strings.TrimSpace(in.PlanID) == "" {
			t.Fatalf("stream hook missing plan_id in phase=%s", in.Phase)
		}
		streamPhases = append(streamPhases, fmt.Sprintf(
			"%d:%s:%d:%s:%s",
			in.Iteration,
			in.Phase,
			in.PlanVersion,
			in.PlanAction,
			in.PlanReason,
		))
		return nil
	})

	runTurns := 0
	runEngine := New(&fakeModel{
		generate: func(ctx context.Context, req types.ModelRequest) (types.ModelResponse, error) {
			_ = ctx
			_ = req
			runTurns++
			if runTurns == 1 {
				return types.ModelResponse{ToolCalls: []types.ToolCall{{CallID: "r1", Name: "local.echo"}}}, nil
			}
			return types.ModelResponse{FinalAnswer: "done"}, nil
		},
	}, WithRuntimeManager(mgr), WithLocalRegistry(reg), WithLifecycleHooks(runHook))
	streamTurns := 0
	streamEngine := New(&fakeModel{
		stream: func(ctx context.Context, req types.ModelRequest, onEvent func(types.ModelEvent) error) error {
			_ = ctx
			_ = req
			streamTurns++
			if streamTurns == 1 {
				return onEvent(types.ModelEvent{
					Type:     types.ModelEventTypeToolCall,
					ToolCall: &types.ToolCall{CallID: "s1", Name: "local.echo"},
				})
			}
			return onEvent(types.ModelEvent{Type: types.ModelEventTypeOutputTextDelta, TextDelta: "done"})
		},
	}, WithRuntimeManager(mgr), WithLocalRegistry(reg), WithLifecycleHooks(streamHook))

	req := types.RunRequest{RunID: "run-react-plan-parity", Input: "x"}
	runRes, runErr := runEngine.Run(context.Background(), req, nil)
	streamRes, streamErr := streamEngine.Stream(context.Background(), req, nil)
	if runErr != nil || streamErr != nil {
		t.Fatalf("run/stream should both succeed, runErr=%v streamErr=%v", runErr, streamErr)
	}
	if runRes.FinalAnswer != "done" || streamRes.FinalAnswer != "done" {
		t.Fatalf("final answer mismatch run=%q stream=%q", runRes.FinalAnswer, streamRes.FinalAnswer)
	}
	runJoined := strings.Join(runPhases, ",")
	streamJoined := strings.Join(streamPhases, ",")
	if runJoined != streamJoined {
		t.Fatalf("plan hook order mismatch run=%q stream=%q", runJoined, streamJoined)
	}
	want := strings.Join([]string{
		"1:before_plan_change:1:create:initial_plan",
		"1:after_plan_change:1:create:initial_plan",
		"2:before_plan_change:2:revise:react_iteration_boundary",
		"2:after_plan_change:2:revise:react_iteration_boundary",
		"2:before_plan_change:3:complete:run_completed",
		"2:after_plan_change:3:complete:run_completed",
	}, ",")
	if runJoined != want {
		t.Fatalf("unexpected plan hook sequence=%q, want %q", runJoined, want)
	}
}

func TestReactPlanHookFailFastStopsRunAndStream(t *testing.T) {
	mgr := newReactPlanHookRuntimeManager(t, runtimeconfig.RuntimeReactPlanChangeHookFailModeFailFast, 200)
	defer func() { _ = mgr.Close() }()

	runModelCalled := 0
	streamModelCalled := 0
	hook := types.AgentLifecycleHookFunc(func(ctx context.Context, in types.AgentLifecycleHookContext) error {
		_ = ctx
		_ = in
		return errors.New("plan hook boom")
	})

	runCollector := &eventCollector{}
	runEngine := New(&fakeModel{
		generate: func(ctx context.Context, req types.ModelRequest) (types.ModelResponse, error) {
			_ = ctx
			_ = req
			runModelCalled++
			return types.ModelResponse{FinalAnswer: "should-not-reach"}, nil
		},
	}, WithRuntimeManager(mgr), WithLifecycleHooks(hook))
	streamCollector := &eventCollector{}
	streamEngine := New(&fakeModel{
		stream: func(ctx context.Context, req types.ModelRequest, onEvent func(types.ModelEvent) error) error {
			_ = ctx
			_ = req
			_ = onEvent
			streamModelCalled++
			return nil
		},
	}, WithRuntimeManager(mgr), WithLifecycleHooks(hook))

	runRes, runErr := runEngine.Run(context.Background(), types.RunRequest{RunID: "run-react-plan-failfast-run", Input: "x"}, runCollector)
	streamRes, streamErr := streamEngine.Stream(context.Background(), types.RunRequest{RunID: "run-react-plan-failfast-stream", Input: "x"}, streamCollector)
	if runErr == nil || streamErr == nil {
		t.Fatalf("expected run/stream fail-fast errors, runErr=%v streamErr=%v", runErr, streamErr)
	}
	if runRes.Error == nil || streamRes.Error == nil {
		t.Fatalf("missing classified errors run=%#v stream=%#v", runRes.Error, streamRes.Error)
	}
	if runRes.Error.Class != types.ErrContext || streamRes.Error.Class != types.ErrContext {
		t.Fatalf("error class mismatch run=%#v stream=%#v", runRes.Error, streamRes.Error)
	}
	if runModelCalled != 0 || streamModelCalled != 0 {
		t.Fatalf("model should not run after before_plan_change fail-fast, run=%d stream=%d", runModelCalled, streamModelCalled)
	}
	if runRes.Error.Details["phase"] != string(types.AgentLifecyclePhaseBeforePlanChange) ||
		streamRes.Error.Details["phase"] != string(types.AgentLifecyclePhaseBeforePlanChange) {
		t.Fatalf("phase mismatch run=%#v stream=%#v", runRes.Error.Details, streamRes.Error.Details)
	}
	runPayload := mustLastRunFinishedPayload(t, runCollector)
	streamPayload := mustLastRunFinishedPayload(t, streamCollector)
	if runPayload["react_plan_hook_status"] != reactPlanHookStatusFailed ||
		streamPayload["react_plan_hook_status"] != reactPlanHookStatusFailed {
		t.Fatalf("hook status mismatch run=%#v stream=%#v", runPayload["react_plan_hook_status"], streamPayload["react_plan_hook_status"])
	}
}

func TestReactPlanHookDegradeSkipsMutationAndContinuesRunStream(t *testing.T) {
	mgr := newReactPlanHookRuntimeManager(t, runtimeconfig.RuntimeReactPlanChangeHookFailModeDegrade, 200)
	defer func() { _ = mgr.Close() }()

	hook := types.AgentLifecycleHookFunc(func(ctx context.Context, in types.AgentLifecycleHookContext) error {
		_ = ctx
		_ = in
		return errors.New("degrade")
	})

	runCollector := &eventCollector{}
	runEngine := New(&fakeModel{
		generate: func(ctx context.Context, req types.ModelRequest) (types.ModelResponse, error) {
			_ = ctx
			_ = req
			return types.ModelResponse{FinalAnswer: "ok"}, nil
		},
	}, WithRuntimeManager(mgr), WithLifecycleHooks(hook))
	streamCollector := &eventCollector{}
	streamEngine := New(&fakeModel{
		stream: func(ctx context.Context, req types.ModelRequest, onEvent func(types.ModelEvent) error) error {
			_ = ctx
			_ = req
			return onEvent(types.ModelEvent{Type: types.ModelEventTypeOutputTextDelta, TextDelta: "ok"})
		},
	}, WithRuntimeManager(mgr), WithLifecycleHooks(hook))

	runRes, runErr := runEngine.Run(context.Background(), types.RunRequest{RunID: "run-react-plan-degrade-run", Input: "x"}, runCollector)
	streamRes, streamErr := streamEngine.Stream(context.Background(), types.RunRequest{RunID: "run-react-plan-degrade-stream", Input: "x"}, streamCollector)
	if runErr != nil || streamErr != nil {
		t.Fatalf("degrade should continue run/stream, runErr=%v streamErr=%v", runErr, streamErr)
	}
	if runRes.FinalAnswer != "ok" || streamRes.FinalAnswer != "ok" {
		t.Fatalf("final answer mismatch run=%q stream=%q", runRes.FinalAnswer, streamRes.FinalAnswer)
	}
	if len(runRes.Warnings) == 0 || len(streamRes.Warnings) == 0 {
		t.Fatalf("expected degrade warnings run=%#v stream=%#v", runRes.Warnings, streamRes.Warnings)
	}
	runPayload := mustLastRunFinishedPayload(t, runCollector)
	streamPayload := mustLastRunFinishedPayload(t, streamCollector)
	if runPayload["react_plan_hook_status"] != reactPlanHookStatusDegraded ||
		streamPayload["react_plan_hook_status"] != reactPlanHookStatusDegraded {
		t.Fatalf("hook status mismatch run=%#v stream=%#v", runPayload["react_plan_hook_status"], streamPayload["react_plan_hook_status"])
	}
	if runPayload["react_plan_change_total"] != 0 || streamPayload["react_plan_change_total"] != 0 {
		t.Fatalf("degrade should skip mutations run=%#v stream=%#v", runPayload["react_plan_change_total"], streamPayload["react_plan_change_total"])
	}
}

func TestReactPlanHookTimeoutClassifiedAsPolicyTimeout(t *testing.T) {
	mgr := newReactPlanHookRuntimeManager(t, runtimeconfig.RuntimeReactPlanChangeHookFailModeFailFast, 20)
	defer func() { _ = mgr.Close() }()

	runModelCalled := 0
	streamModelCalled := 0
	hook := types.AgentLifecycleHookFunc(func(ctx context.Context, in types.AgentLifecycleHookContext) error {
		_ = in
		<-ctx.Done()
		return ctx.Err()
	})

	runEngine := New(&fakeModel{
		generate: func(ctx context.Context, req types.ModelRequest) (types.ModelResponse, error) {
			_ = ctx
			_ = req
			runModelCalled++
			return types.ModelResponse{FinalAnswer: "should-not-reach"}, nil
		},
	}, WithRuntimeManager(mgr), WithLifecycleHooks(hook))
	streamEngine := New(&fakeModel{
		stream: func(ctx context.Context, req types.ModelRequest, onEvent func(types.ModelEvent) error) error {
			_ = ctx
			_ = req
			_ = onEvent
			streamModelCalled++
			return nil
		},
	}, WithRuntimeManager(mgr), WithLifecycleHooks(hook))

	runRes, runErr := runEngine.Run(context.Background(), types.RunRequest{RunID: "run-react-plan-timeout-run", Input: "x"}, nil)
	streamRes, streamErr := streamEngine.Stream(context.Background(), types.RunRequest{RunID: "run-react-plan-timeout-stream", Input: "x"}, nil)
	if runErr == nil || streamErr == nil {
		t.Fatalf("expected run/stream timeout errors, runErr=%v streamErr=%v", runErr, streamErr)
	}
	if runRes.Error == nil || streamRes.Error == nil {
		t.Fatalf("missing classified timeout errors run=%#v stream=%#v", runRes.Error, streamRes.Error)
	}
	if runRes.Error.Class != types.ErrPolicyTimeout || streamRes.Error.Class != types.ErrPolicyTimeout {
		t.Fatalf("timeout class mismatch run=%#v stream=%#v", runRes.Error, streamRes.Error)
	}
	if runRes.Error.Details["reason_code"] != "hook_timeout" || streamRes.Error.Details["reason_code"] != "hook_timeout" {
		t.Fatalf("reason_code mismatch run=%#v stream=%#v", runRes.Error.Details, streamRes.Error.Details)
	}
	if runModelCalled != 0 || streamModelCalled != 0 {
		t.Fatalf("model should not run after hook timeout, run=%d stream=%d", runModelCalled, streamModelCalled)
	}
}

func TestReactPlanRecoveryRunStreamParityAcrossBackends(t *testing.T) {
	for _, backend := range []string{"memory", "file"} {
		t.Run(backend, func(t *testing.T) {
			mgr := newReactPlanRecoveryRuntimeManager(t, backend)
			defer func() { _ = mgr.Close() }()

			runCollector := &eventCollector{}
			runEngine := New(&fakeModel{
				generate: func(ctx context.Context, req types.ModelRequest) (types.ModelResponse, error) {
					_ = ctx
					_ = req
					return types.ModelResponse{FinalAnswer: "ok"}, nil
				},
			}, WithRuntimeManager(mgr))
			streamCollector := &eventCollector{}
			streamEngine := New(&fakeModel{
				stream: func(ctx context.Context, req types.ModelRequest, onEvent func(types.ModelEvent) error) error {
					_ = ctx
					_ = req
					return onEvent(types.ModelEvent{Type: types.ModelEventTypeOutputTextDelta, TextDelta: "ok"})
				},
			}, WithRuntimeManager(mgr))

			req := types.RunRequest{
				RunID:     "run-react-plan-recovery-parity",
				SessionID: "session-react-plan-recovery",
				Input:     "resume",
				Messages:  []types.Message{{Role: "user", Content: "resume context"}},
			}
			runRes, runErr := runEngine.Run(context.Background(), req, runCollector)
			streamRes, streamErr := streamEngine.Stream(context.Background(), req, streamCollector)
			if runErr != nil || streamErr != nil {
				t.Fatalf("run/stream should both succeed, runErr=%v streamErr=%v", runErr, streamErr)
			}
			if runRes.FinalAnswer != "ok" || streamRes.FinalAnswer != "ok" {
				t.Fatalf("final answer mismatch run=%q stream=%q", runRes.FinalAnswer, streamRes.FinalAnswer)
			}
			runPayload := mustLastRunFinishedPayload(t, runCollector)
			streamPayload := mustLastRunFinishedPayload(t, streamCollector)
			if runPayload["react_plan_recover_count"] != 1 || streamPayload["react_plan_recover_count"] != 1 {
				t.Fatalf("recover count mismatch run=%#v stream=%#v", runPayload["react_plan_recover_count"], streamPayload["react_plan_recover_count"])
			}
			if runPayload["react_plan_change_total"] != 3 || streamPayload["react_plan_change_total"] != 3 {
				t.Fatalf("change total mismatch run=%#v stream=%#v", runPayload["react_plan_change_total"], streamPayload["react_plan_change_total"])
			}
			if runPayload["react_plan_last_action"] != reactPlanActionComplete || streamPayload["react_plan_last_action"] != reactPlanActionComplete {
				t.Fatalf("last action mismatch run=%#v stream=%#v", runPayload["react_plan_last_action"], streamPayload["react_plan_last_action"])
			}
			if runPayload["react_plan_hook_status"] != reactPlanHookStatusDisabled ||
				streamPayload["react_plan_hook_status"] != reactPlanHookStatusDisabled {
				t.Fatalf("hook status mismatch run=%#v stream=%#v", runPayload["react_plan_hook_status"], streamPayload["react_plan_hook_status"])
			}
		})
	}
}

func TestReactPlanNotebookDoesNotBypassActionGatePrecedence(t *testing.T) {
	mgr := newReactPlanRecoveryRuntimeManager(t, "memory")
	defer func() { _ = mgr.Close() }()

	reg := local.NewRegistry()
	if _, err := reg.Register(&fakeTool{name: "shell"}); err != nil {
		t.Fatalf("register tool: %v", err)
	}
	runModel := &fakeModel{
		generate: func(ctx context.Context, req types.ModelRequest) (types.ModelResponse, error) {
			_ = ctx
			_ = req
			return types.ModelResponse{ToolCalls: []types.ToolCall{{CallID: "r1", Name: "local.shell"}}}, nil
		},
	}
	streamModel := &fakeModel{
		stream: func(ctx context.Context, req types.ModelRequest, onEvent func(types.ModelEvent) error) error {
			_ = ctx
			_ = req
			return onEvent(types.ModelEvent{
				Type:     types.ModelEventTypeToolCall,
				ToolCall: &types.ToolCall{CallID: "s1", Name: "local.shell"},
			})
		},
	}
	denyMatcher := &fakeGateMatcher{
		evaluate: func(ctx context.Context, check types.ActionGateCheck) (types.ActionGateDecision, error) {
			_ = ctx
			_ = check
			return types.ActionGateDecisionDeny, nil
		},
	}

	runCollector := &eventCollector{}
	runEngine := New(runModel, WithRuntimeManager(mgr), WithLocalRegistry(reg), WithActionGateMatcher(denyMatcher))
	streamCollector := &eventCollector{}
	streamEngine := New(streamModel, WithRuntimeManager(mgr), WithActionGateMatcher(denyMatcher))

	runRes, runErr := runEngine.Run(context.Background(), types.RunRequest{RunID: "run-react-plan-policy-precedence-boundary-run", Input: "danger"}, runCollector)
	streamRes, streamErr := streamEngine.Stream(context.Background(), types.RunRequest{RunID: "run-react-plan-policy-precedence-boundary-stream", Input: "danger"}, streamCollector)
	if runErr == nil || streamErr == nil {
		t.Fatalf("run/stream should both be denied by action gate, runErr=%v streamErr=%v", runErr, streamErr)
	}
	if runRes.Error == nil || streamRes.Error == nil {
		t.Fatalf("missing classified errors run=%#v stream=%#v", runRes.Error, streamRes.Error)
	}
	if runRes.Error.Details["winner_stage"] != runtimeconfig.RuntimePolicyStageActionGate ||
		streamRes.Error.Details["winner_stage"] != runtimeconfig.RuntimePolicyStageActionGate {
		t.Fatalf("winner_stage mismatch run=%#v stream=%#v", runRes.Error.Details, streamRes.Error.Details)
	}
	runPayload := mustLastRunFinishedPayload(t, runCollector)
	streamPayload := mustLastRunFinishedPayload(t, streamCollector)
	for _, key := range []string{"reason_code", "policy_precedence_version", "winner_stage", "deny_source"} {
		if runPayload[key] != streamPayload[key] {
			t.Fatalf("run/stream precedence payload mismatch key=%s run=%#v stream=%#v", key, runPayload[key], streamPayload[key])
		}
	}
	if runPayload["react_plan_change_total"] == 0 || streamPayload["react_plan_change_total"] == 0 {
		t.Fatalf("react plan should still record lifecycle before deny run=%#v stream=%#v", runPayload["react_plan_change_total"], streamPayload["react_plan_change_total"])
	}
}

func TestReactPlanNotebookDoesNotBypassSandboxSecurityChain(t *testing.T) {
	mgr := newReactPlanSecurityRuntimeManager(t)
	defer func() { _ = mgr.Close() }()

	reg := local.NewRegistry()
	toolInvoke := 0
	if _, err := reg.Register(&fakeTool{
		name: "search",
		invoke: func(ctx context.Context, args map[string]any) (types.ToolResult, error) {
			_ = ctx
			_ = args
			toolInvoke++
			return types.ToolResult{Content: "should-not-run"}, nil
		},
	}); err != nil {
		t.Fatalf("register tool: %v", err)
	}
	runModel := &fakeModel{
		generate: func(ctx context.Context, req types.ModelRequest) (types.ModelResponse, error) {
			_ = ctx
			_ = req
			return types.ModelResponse{
				ToolCalls: []types.ToolCall{{
					CallID: "r-egress",
					Name:   "local.search",
					Args:   map[string]any{"url": "https://api.example.com/v1/search"},
				}},
			}, nil
		},
	}
	streamModel := &fakeModel{
		stream: func(ctx context.Context, req types.ModelRequest, onEvent func(types.ModelEvent) error) error {
			_ = ctx
			_ = req
			return onEvent(types.ModelEvent{
				Type: types.ModelEventTypeToolCall,
				ToolCall: &types.ToolCall{
					CallID: "s-egress",
					Name:   "local.search",
					Args:   map[string]any{"url": "https://api.example.com/v1/search"},
				},
			})
		},
	}

	runCollector := &eventCollector{}
	runEngine := New(runModel, WithRuntimeManager(mgr), WithLocalRegistry(reg))
	streamCollector := &eventCollector{}
	streamEngine := New(streamModel, WithRuntimeManager(mgr), WithLocalRegistry(reg))

	runRes, runErr := runEngine.Run(context.Background(), types.RunRequest{RunID: "run-react-plan-sandbox-egress-boundary-run", Input: "trigger egress deny"}, runCollector)
	streamRes, streamErr := streamEngine.Stream(context.Background(), types.RunRequest{RunID: "run-react-plan-sandbox-egress-boundary-stream", Input: "trigger egress deny"}, streamCollector)
	if runErr == nil || streamErr == nil {
		t.Fatalf("run/stream should both be denied by sandbox egress, runErr=%v streamErr=%v", runErr, streamErr)
	}
	if runRes.Error == nil || streamRes.Error == nil {
		t.Fatalf("missing classified errors run=%#v stream=%#v", runRes.Error, streamRes.Error)
	}
	if runRes.Error.Class != types.ErrSecurity || streamRes.Error.Class != types.ErrSecurity {
		t.Fatalf("security class mismatch run=%#v stream=%#v", runRes.Error, streamRes.Error)
	}
	if runRes.Error.Details["reason_code"] != "sandbox.egress_deny" ||
		streamRes.Error.Details["reason_code"] != "sandbox.egress_deny" {
		t.Fatalf("reason_code mismatch run=%#v stream=%#v", runRes.Error.Details, streamRes.Error.Details)
	}
	if toolInvoke != 0 {
		t.Fatalf("tool should not execute when egress denied, invoke=%d", toolInvoke)
	}
	runPayload := mustLastRunFinishedPayload(t, runCollector)
	streamPayload := mustLastRunFinishedPayload(t, streamCollector)
	if runPayload["sandbox_egress_action"] != runtimeconfig.SecuritySandboxEgressActionDeny ||
		streamPayload["sandbox_egress_action"] != runtimeconfig.SecuritySandboxEgressActionDeny {
		t.Fatalf("sandbox egress action mismatch run=%#v stream=%#v", runPayload["sandbox_egress_action"], streamPayload["sandbox_egress_action"])
	}
	if runPayload["react_plan_change_total"] == 0 || streamPayload["react_plan_change_total"] == 0 {
		t.Fatalf("react plan should still record lifecycle before sandbox deny run=%#v stream=%#v", runPayload["react_plan_change_total"], streamPayload["react_plan_change_total"])
	}
}

func mustLastRunFinishedPayload(t *testing.T, collector *eventCollector) map[string]any {
	t.Helper()
	ev, ok := collector.lastNonTimelineEvent()
	if !ok {
		t.Fatal("missing events")
	}
	if ev.Type != "run.finished" {
		t.Fatalf("last event=%q, want run.finished", ev.Type)
	}
	if ev.Payload == nil {
		t.Fatal("run.finished payload is nil")
	}
	return ev.Payload
}

func newReactPlanHookRuntimeManager(t *testing.T, failMode string, timeoutMs int) *runtimeconfig.Manager {
	t.Helper()
	cfgPath := filepath.Join(t.TempDir(), "runtime-react-plan-hook.yaml")
	cfg := fmt.Sprintf(`
runtime:
  hooks:
    enabled: false
  react:
    enabled: true
    stream_tool_dispatch_enabled: true
    plan_notebook:
      enabled: true
      max_history: 16
      on_recover_conflict: reject
    plan_change_hook:
      enabled: true
      fail_mode: %s
      timeout_ms: %d
`, strings.TrimSpace(failMode), timeoutMs)
	if err := os.WriteFile(cfgPath, []byte(strings.TrimSpace(cfg)), 0o600); err != nil {
		t.Fatalf("write config failed: %v", err)
	}
	mgr, err := runtimeconfig.NewManager(runtimeconfig.ManagerOptions{
		FilePath:  cfgPath,
		EnvPrefix: "BAYMAX_RUNTIME_REACT_PLAN_HOOK_TEST",
	})
	if err != nil {
		t.Fatalf("new runtime manager failed: %v", err)
	}
	return mgr
}

func newReactPlanRecoveryRuntimeManager(t *testing.T, backend string) *runtimeconfig.Manager {
	t.Helper()
	cfgPath := filepath.Join(t.TempDir(), "runtime-react-plan-recovery.yaml")
	recoveryPath := ""
	normalizedBackend := strings.ToLower(strings.TrimSpace(backend))
	if normalizedBackend == "file" {
		recoveryPath = filepath.Join(t.TempDir(), "recovery-store.json")
	}
	extraPath := ""
	if recoveryPath != "" {
		extraPath = fmt.Sprintf("\n  path: %s", strings.ReplaceAll(recoveryPath, "\\", "/"))
	}
	cfg := fmt.Sprintf(`
runtime:
  react:
    enabled: true
    stream_tool_dispatch_enabled: true
    plan_notebook:
      enabled: true
      max_history: 16
      on_recover_conflict: reject
    plan_change_hook:
      enabled: false
      fail_mode: fail_fast
      timeout_ms: 2000
recovery:
  backend: %s%s
  resume_boundary: next_attempt_only
  timeout_reentry_policy: single_reentry_then_fail
  timeout_reentry_max_per_task: 1
`, normalizedBackend, extraPath)
	if err := os.WriteFile(cfgPath, []byte(strings.TrimSpace(cfg)), 0o600); err != nil {
		t.Fatalf("write config failed: %v", err)
	}
	mgr, err := runtimeconfig.NewManager(runtimeconfig.ManagerOptions{
		FilePath:  cfgPath,
		EnvPrefix: "BAYMAX_RUNTIME_REACT_PLAN_RECOVERY_TEST",
	})
	if err != nil {
		t.Fatalf("new runtime manager failed: %v", err)
	}
	return mgr
}

func newReactPlanSecurityRuntimeManager(t *testing.T) *runtimeconfig.Manager {
	t.Helper()
	cfgPath := filepath.Join(t.TempDir(), "runtime-react-plan-security.yaml")
	cfg := `
runtime:
  react:
    enabled: true
    stream_tool_dispatch_enabled: true
    plan_notebook:
      enabled: true
      max_history: 16
      on_recover_conflict: reject
    plan_change_hook:
      enabled: false
      fail_mode: fail_fast
      timeout_ms: 2000
security:
  sandbox:
    enabled: true
    mode: enforce
    required: false
    policy:
      default_action: host
      profile: default
      fallback_action: deny
    egress:
      enabled: true
      default_action: deny
      on_violation: deny
`
	if err := os.WriteFile(cfgPath, []byte(strings.TrimSpace(cfg)), 0o600); err != nil {
		t.Fatalf("write config failed: %v", err)
	}
	mgr, err := runtimeconfig.NewManager(runtimeconfig.ManagerOptions{
		FilePath:  cfgPath,
		EnvPrefix: "BAYMAX_RUNTIME_REACT_PLAN_SECURITY_TEST",
	})
	if err != nil {
		t.Fatalf("new runtime manager failed: %v", err)
	}
	return mgr
}
