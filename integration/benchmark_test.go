package integration

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/FelixSeptem/baymax/a2a"
	"github.com/FelixSeptem/baymax/context/assembler"
	"github.com/FelixSeptem/baymax/core/runner"
	"github.com/FelixSeptem/baymax/core/types"
	"github.com/FelixSeptem/baymax/integration/fakes"
	httpmcp "github.com/FelixSeptem/baymax/mcp/http"
	mcpprofile "github.com/FelixSeptem/baymax/mcp/profile"
	"github.com/FelixSeptem/baymax/orchestration/scheduler"
	"github.com/FelixSeptem/baymax/orchestration/workflow"
	runtimeconfig "github.com/FelixSeptem/baymax/runtime/config"
	runtimediag "github.com/FelixSeptem/baymax/runtime/diagnostics"
	"github.com/FelixSeptem/baymax/tool/local"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func BenchmarkIterationLatency(b *testing.B) {
	model := fakes.NewModel([]fakes.ModelStep{{Response: types.ModelResponse{FinalAnswer: "ok"}}})
	eng := runner.New(model)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = eng.Run(context.Background(), types.RunRequest{Input: "x"}, nil)
	}
}

func BenchmarkRuntimeBudgetAdmissionDecisionStability(b *testing.B) {
	cfgPath := filepath.Join(b.TempDir(), "runtime-a60-budget-bench.yaml")
	cfg := strings.Join([]string{
		"runtime:",
		"  readiness:",
		"    enabled: true",
		"    strict: false",
		"    remote_probe_enabled: false",
		"    admission:",
		"      enabled: true",
		"      mode: fail_fast",
		"      block_on: blocked_only",
		"      degraded_policy: allow_and_record",
		"  admission:",
		"    budget:",
		"      cost:",
		"        degrade_threshold: 0.75",
		"        hard_threshold: 1.20",
		"      latency:",
		"        degrade_threshold: 1s",
		"        hard_threshold: 2200ms",
		"    degrade_policy:",
		"      enabled: true",
		"      action_order: [reduce_tool_call_limit, trim_memory_context, sandbox_throttle]",
		"      conflict_policy: first_action",
		"reload:",
		"  enabled: false",
		"  debounce: 20ms",
		"",
	}, "\n")
	if err := os.WriteFile(cfgPath, []byte(cfg), 0o600); err != nil {
		b.Fatalf("write runtime config: %v", err)
	}
	mgr, err := runtimeconfig.NewManager(runtimeconfig.ManagerOptions{
		FilePath:  cfgPath,
		EnvPrefix: "BAYMAX_A60_BENCH",
	})
	if err != nil {
		b.Fatalf("new runtime manager: %v", err)
	}
	defer func() { _ = mgr.Close() }()

	degradeReq := runtimeconfig.ReadinessAdmissionRequest{
		ExplicitBudgetEstimate: &runtimeconfig.RuntimeAdmissionBudgetEstimate{
			TokenCost:      0.35,
			ToolCost:       0.21,
			SandboxCost:    0.18,
			MemoryCost:     0.15,
			TokenLatency:   420 * time.Millisecond,
			ToolLatency:    320 * time.Millisecond,
			SandboxLatency: 260 * time.Millisecond,
			MemoryLatency:  180 * time.Millisecond,
		},
	}
	denyReq := runtimeconfig.ReadinessAdmissionRequest{
		ExplicitBudgetEstimate: &runtimeconfig.RuntimeAdmissionBudgetEstimate{
			TokenCost:      0.55,
			ToolCost:       0.40,
			SandboxCost:    0.30,
			MemoryCost:     0.20,
			TokenLatency:   700 * time.Millisecond,
			ToolLatency:    620 * time.Millisecond,
			SandboxLatency: 540 * time.Millisecond,
			MemoryLatency:  420 * time.Millisecond,
		},
	}

	degradeDurations := make([]int64, 0, b.N)
	denyDurations := make([]int64, 0, b.N)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		startDegrade := time.Now()
		degradeDecision := mgr.EvaluateReadinessAdmissionWithBudgetRequest("", degradeReq)
		degradeDurations = append(degradeDurations, time.Since(startDegrade).Nanoseconds())
		if degradeDecision.BudgetDecision != runtimeconfig.RuntimeAdmissionBudgetDecisionDegrade {
			b.Fatalf("degrade decision drift at iter=%d: %#v", i, degradeDecision)
		}
		if degradeDecision.ReasonCode != runtimeconfig.ReadinessAdmissionCodeBudgetDegradeAllow {
			b.Fatalf("degrade reason drift at iter=%d: %#v", i, degradeDecision)
		}
		if degradeDecision.DegradeAction != runtimeconfig.RuntimeAdmissionDegradeActionReduceToolCallLimit {
			b.Fatalf("degrade action drift at iter=%d: %#v", i, degradeDecision)
		}

		startDeny := time.Now()
		denyDecision := mgr.EvaluateReadinessAdmissionWithBudgetRequest("", denyReq)
		denyDurations = append(denyDurations, time.Since(startDeny).Nanoseconds())
		if denyDecision.BudgetDecision != runtimeconfig.RuntimeAdmissionBudgetDecisionDeny {
			b.Fatalf("deny decision drift at iter=%d: %#v", i, denyDecision)
		}
		if denyDecision.ReasonCode != runtimeconfig.ReadinessAdmissionCodeBudgetHardDeny {
			b.Fatalf("deny reason drift at iter=%d: %#v", i, denyDecision)
		}
		if strings.TrimSpace(denyDecision.DegradeAction) != "" {
			b.Fatalf("deny action must be empty at iter=%d: %#v", i, denyDecision)
		}
	}
	b.StopTimer()

	if p95 := percentileNs(degradeDurations, 95); p95 > 0 {
		b.ReportMetric(float64(p95), "a60-degrade-p95-ns/op")
	}
	if p99 := percentileNs(degradeDurations, 99); p99 > 0 {
		b.ReportMetric(float64(p99), "a60-degrade-p99-ns/op")
	}
	if p95 := percentileNs(denyDurations, 95); p95 > 0 {
		b.ReportMetric(float64(p95), "a60-deny-p95-ns/op")
	}
	if p99 := percentileNs(denyDurations, 99); p99 > 0 {
		b.ReportMetric(float64(p99), "a60-deny-p99-ns/op")
	}
}

func BenchmarkToolFanOut(b *testing.B) {
	reg := local.NewRegistry()
	_, _ = reg.Register(&fakes.Tool{NameValue: "echo", InvokeFn: func(ctx context.Context, args map[string]any) (types.ToolResult, error) {
		return types.ToolResult{Content: "ok"}, nil
	}})
	dispatcher := local.NewDispatcher(reg)
	calls := make([]types.ToolCall, 8)
	for i := range calls {
		calls[i] = types.ToolCall{CallID: "c", Name: "local.echo"}
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = dispatcher.Dispatch(context.Background(), calls, local.DispatchConfig{MaxCalls: 8, Concurrency: 8})
	}
}

func BenchmarkToolFanOutHighConcurrency(b *testing.B) {
	reg := local.NewRegistry()
	_, _ = reg.Register(&fakes.Tool{NameValue: "echo", InvokeFn: func(ctx context.Context, args map[string]any) (types.ToolResult, error) {
		return types.ToolResult{Content: "ok"}, nil
	}})
	dispatcher := local.NewDispatcher(reg)
	calls := make([]types.ToolCall, 64)
	for i := range calls {
		calls[i] = types.ToolCall{CallID: "c", Name: "local.echo"}
	}
	cfg := local.DispatchConfig{MaxCalls: 64, Concurrency: 32, QueueSize: 32, Backpressure: types.BackpressureBlock}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = dispatcher.Dispatch(context.Background(), calls, cfg)
	}
}

func BenchmarkToolFanOutSlowCall(b *testing.B) {
	reg := local.NewRegistry()
	_, _ = reg.Register(&fakes.Tool{NameValue: "slow", InvokeFn: func(ctx context.Context, args map[string]any) (types.ToolResult, error) {
		time.Sleep(100 * time.Microsecond)
		return types.ToolResult{Content: "ok"}, nil
	}})
	dispatcher := local.NewDispatcher(reg)
	calls := make([]types.ToolCall, 32)
	for i := range calls {
		calls[i] = types.ToolCall{CallID: "s", Name: "local.slow"}
	}
	cfg := local.DispatchConfig{MaxCalls: 32, Concurrency: 16, QueueSize: 16, Backpressure: types.BackpressureBlock}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = dispatcher.Dispatch(context.Background(), calls, cfg)
	}
}

func BenchmarkToolFanOutCancelStorm(b *testing.B) {
	reg := local.NewRegistry()
	_, _ = reg.Register(&fakes.Tool{NameValue: "slow", InvokeFn: func(ctx context.Context, args map[string]any) (types.ToolResult, error) {
		select {
		case <-ctx.Done():
			return types.ToolResult{}, ctx.Err()
		case <-time.After(2 * time.Millisecond):
			return types.ToolResult{Content: "ok"}, nil
		}
	}})
	dispatcher := local.NewDispatcher(reg)
	calls := make([]types.ToolCall, 16)
	for i := range calls {
		calls[i] = types.ToolCall{CallID: "k", Name: "local.slow"}
	}
	cfg := local.DispatchConfig{MaxCalls: 16, Concurrency: 8, QueueSize: 8, Backpressure: types.BackpressureBlock}
	durations := make([]int64, 0, b.N)
	goroutinePeak := 0
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		start := time.Now()
		ctx, cancel := context.WithTimeout(context.Background(), 200*time.Microsecond)
		_, _ = dispatcher.Dispatch(ctx, calls, cfg)
		cancel()
		elapsed := time.Since(start).Nanoseconds()
		if elapsed <= 0 {
			elapsed = 1000
		}
		durations = append(durations, elapsed)
		if n := runtime.NumGoroutine(); n > goroutinePeak {
			goroutinePeak = n
		}
	}
	b.StopTimer()
	if p95 := percentileNs(durations, 95); p95 > 0 {
		b.ReportMetric(float64(p95), "p95-ns/op")
	}
	if goroutinePeak > 0 {
		b.ReportMetric(float64(goroutinePeak), "goroutine-peak")
	}
}

func BenchmarkToolFanOutDropLowPriority(b *testing.B) {
	reg := local.NewRegistry()
	_, _ = reg.Register(&fakes.Tool{NameValue: "slow", InvokeFn: func(ctx context.Context, args map[string]any) (types.ToolResult, error) {
		time.Sleep(250 * time.Microsecond)
		return types.ToolResult{Content: "ok"}, nil
	}})
	dispatcher := local.NewDispatcher(reg)
	calls := make([]types.ToolCall, 24)
	for i := range calls {
		calls[i] = types.ToolCall{
			CallID: "d",
			Name:   "local.slow",
			Args:   map[string]any{"q": "cache warmup"},
		}
	}
	cfg := local.DispatchConfig{
		MaxCalls:     24,
		Concurrency:  2,
		QueueSize:    2,
		Backpressure: types.BackpressureDropLowPriority,
		DropLowPriority: local.DropLowPriorityPolicy{
			PriorityByKeyword:   map[string]string{"cache": runtimeconfig.DropPriorityLow},
			DroppablePriorities: []string{runtimeconfig.DropPriorityLow},
		},
	}
	durations := make([]int64, 0, b.N)
	goroutinePeak := 0
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		start := time.Now()
		_, _ = dispatcher.Dispatch(context.Background(), calls, cfg)
		elapsed := time.Since(start).Nanoseconds()
		if elapsed <= 0 {
			elapsed = 1000
		}
		durations = append(durations, elapsed)
		if n := runtime.NumGoroutine(); n > goroutinePeak {
			goroutinePeak = n
		}
	}
	b.StopTimer()
	if p95 := percentileNs(durations, 95); p95 > 0 {
		b.ReportMetric(float64(p95), "p95-ns/op")
	}
	if goroutinePeak > 0 {
		b.ReportMetric(float64(goroutinePeak), "goroutine-peak")
	}
}

func BenchmarkToolFanOutDropLowPriorityMCPAndSkill(b *testing.B) {
	reg := local.NewRegistry()
	_, _ = reg.Register(&fakes.Tool{NameValue: "mcp_proxy", InvokeFn: func(ctx context.Context, args map[string]any) (types.ToolResult, error) {
		time.Sleep(250 * time.Microsecond)
		return types.ToolResult{Content: "ok"}, nil
	}})
	_, _ = reg.Register(&fakes.Tool{NameValue: "skill_router", InvokeFn: func(ctx context.Context, args map[string]any) (types.ToolResult, error) {
		time.Sleep(250 * time.Microsecond)
		return types.ToolResult{Content: "ok"}, nil
	}})
	dispatcher := local.NewDispatcher(reg)
	calls := make([]types.ToolCall, 24)
	for i := range calls {
		if i%2 == 0 {
			calls[i] = types.ToolCall{CallID: "m", Name: "local.mcp_proxy", Args: map[string]any{"q": "cache warmup"}}
			continue
		}
		calls[i] = types.ToolCall{CallID: "s", Name: "local.skill_router", Args: map[string]any{"q": "cache warmup"}}
	}
	cfg := local.DispatchConfig{
		MaxCalls:     24,
		Concurrency:  2,
		QueueSize:    2,
		Backpressure: types.BackpressureDropLowPriority,
		DropLowPriority: local.DropLowPriorityPolicy{
			PriorityByKeyword:   map[string]string{"cache": runtimeconfig.DropPriorityLow},
			DroppablePriorities: []string{runtimeconfig.DropPriorityLow},
		},
	}
	durations := make([]int64, 0, b.N)
	goroutinePeak := 0
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		start := time.Now()
		_, _ = dispatcher.Dispatch(context.Background(), calls, cfg)
		elapsed := time.Since(start).Nanoseconds()
		if elapsed <= 0 {
			elapsed = 1000
		}
		durations = append(durations, elapsed)
		if n := runtime.NumGoroutine(); n > goroutinePeak {
			goroutinePeak = n
		}
	}
	b.StopTimer()
	if p95 := percentileNs(durations, 95); p95 > 0 {
		b.ReportMetric(float64(p95), "p95-ns/op")
	}
	if goroutinePeak > 0 {
		b.ReportMetric(float64(goroutinePeak), "goroutine-peak")
	}
}

func BenchmarkMultiAgentMainlineSyncInvocation(b *testing.B) {
	server := a2a.NewInMemoryServer(a2a.HandlerFunc(func(_ context.Context, req a2a.TaskRequest) (map[string]any, error) {
		return map[string]any{
			"ok":          true,
			"workflow_id": req.WorkflowID,
			"team_id":     req.TeamID,
		}, nil
	}), nil)
	client := a2a.NewClient(server, []a2a.AgentCard{
		{
			AgentID:                "bench-agent-remote",
			PeerID:                 "bench-peer-remote",
			SchemaVersion:          "a2a.v1.0",
			SupportedDeliveryModes: []string{a2a.DeliveryModeCallback},
		},
	}, a2a.DeterministicRouter{RequireAll: true}, a2a.ClientPolicy{
		Timeout:            400 * time.Millisecond,
		RequestMaxAttempts: 1,
	}, nil)
	workflowEngine := workflow.New(
		workflow.WithStepAdapter(workflow.DispatchAdapter{
			A2A: workflow.NewA2AStepAdapter(client, workflow.A2AStepAdapterOptions{
				Method:       "workflow.delegate",
				PollInterval: 5 * time.Millisecond,
			}),
		}),
	)

	durations := make([]int64, 0, b.N)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		start := time.Now()
		res, err := workflowEngine.Run(context.Background(), workflow.RunRequest{
			RunID: fmt.Sprintf("bench-a19-sync-%d", i),
			DSL: workflow.Definition{
				WorkflowID: "bench-a19-sync-workflow",
				Steps: []workflow.Step{
					{
						StepID:  "bench-remote-step",
						TaskID:  fmt.Sprintf("bench-sync-task-%d", i),
						Kind:    workflow.StepKindA2A,
						TeamID:  "bench-sync-team",
						AgentID: "bench-agent-main",
						PeerID:  "bench-peer-remote",
					},
				},
			},
		})
		if err != nil {
			b.Fatalf("sync workflow run failed: %v", err)
		}
		if res.WorkflowStatus != "succeeded" || res.WorkflowRemoteTotal != 1 || res.WorkflowRemoteFailed != 0 {
			b.Fatalf("sync workflow aggregate mismatch: %#v", res)
		}
		elapsed := time.Since(start).Nanoseconds()
		if elapsed <= 0 {
			elapsed = 1000
		}
		durations = append(durations, elapsed)
	}
	b.StopTimer()
	p95 := percentileNs(durations, 95)
	if p95 <= 0 && b.N > 0 {
		p95 = b.Elapsed().Nanoseconds() / int64(b.N)
	}
	if p95 <= 0 {
		p95 = 1
	}
	b.ReportMetric(float64(p95), "p95-ns/op")
}

func BenchmarkMultiAgentMainlineAsyncReporting(b *testing.B) {
	server := a2a.NewInMemoryServer(a2a.HandlerFunc(func(_ context.Context, _ a2a.TaskRequest) (map[string]any, error) {
		return map[string]any{"ok": true}, nil
	}), nil)
	client := a2a.NewClient(server, nil, nil, a2a.ClientPolicy{
		Timeout:            300 * time.Millisecond,
		RequestMaxAttempts: 1,
		AsyncReporting: a2a.AsyncReportingPolicy{
			Enabled: true,
			Retry: a2a.AsyncReportingRetryPolicy{
				MaxAttempts:    2,
				BackoffInitial: time.Millisecond,
				BackoffMax:     2 * time.Millisecond,
			},
			JitterRatio: 0,
		},
	}, nil)

	durations := make([]int64, 0, b.N)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sink := a2a.NewChannelReportSink(1)
		start := time.Now()
		ack, err := client.SubmitAsync(context.Background(), a2a.TaskRequest{
			TaskID:  fmt.Sprintf("bench-a19-async-task-%d", i),
			AgentID: "bench-agent-async",
			PeerID:  "bench-peer-async",
			Method:  "bench.async",
		}, sink)
		if err != nil {
			b.Fatalf("async submit failed: %v", err)
		}

		select {
		case report := <-sink.Channel():
			if report.TaskID != ack.TaskID || report.Status != a2a.StatusSucceeded {
				b.Fatalf("unexpected async report: %#v", report)
			}
		case <-time.After(2 * time.Second):
			b.Fatal("timed out waiting for async report")
		}

		durations = append(durations, time.Since(start).Nanoseconds())
	}
	b.StopTimer()
	p95 := percentileNs(durations, 95)
	if p95 <= 0 && b.N > 0 {
		p95 = b.Elapsed().Nanoseconds() / int64(b.N)
	}
	if p95 <= 0 {
		p95 = 1
	}
	b.ReportMetric(float64(p95), "p95-ns/op")
}

func BenchmarkMultiAgentMainlineDelayedDispatch(b *testing.B) {
	ctx := context.Background()
	durations := make([]int64, 0, b.N)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		start := time.Now()
		s, err := scheduler.New(scheduler.NewMemoryStore(), scheduler.WithLeaseTimeout(500*time.Millisecond))
		if err != nil {
			b.Fatalf("new scheduler: %v", err)
		}

		taskID := fmt.Sprintf("bench-a19-delayed-task-%d", i)
		if _, err := s.Enqueue(ctx, scheduler.Task{
			TaskID:    taskID,
			RunID:     "bench-a19-delayed-run",
			NotBefore: time.Now().Add(120 * time.Microsecond),
		}); err != nil {
			b.Fatalf("enqueue delayed task failed: %v", err)
		}
		claimed, ok, err := s.Claim(ctx, "bench-delayed-worker")
		if err != nil {
			b.Fatalf("claim delayed task failed: %v", err)
		}
		if !ok {
			time.Sleep(180 * time.Microsecond)
			claimed, ok, err = s.Claim(ctx, "bench-delayed-worker")
			if err != nil || !ok {
				b.Fatalf("claim delayed task after wait failed: ok=%v err=%v", ok, err)
			}
		}
		if _, err := s.Complete(ctx, scheduler.TerminalCommit{
			TaskID:      claimed.Record.Task.TaskID,
			AttemptID:   claimed.Attempt.AttemptID,
			Status:      scheduler.TaskStateSucceeded,
			CommittedAt: time.Now(),
			Result:      map[string]any{"ok": true},
		}); err != nil {
			b.Fatalf("complete delayed task failed: %v", err)
		}
		durations = append(durations, time.Since(start).Nanoseconds())
	}
	b.StopTimer()
	p95 := percentileNs(durations, 95)
	if p95 <= 0 && b.N > 0 {
		p95 = b.Elapsed().Nanoseconds() / int64(b.N)
	}
	if p95 <= 0 {
		p95 = 1
	}
	b.ReportMetric(float64(p95), "p95-ns/op")
}

func BenchmarkMultiAgentMainlineRecoveryReplay(b *testing.B) {
	ctx := context.Background()
	setupScheduler, err := scheduler.New(scheduler.NewMemoryStore(), scheduler.WithLeaseTimeout(500*time.Millisecond))
	if err != nil {
		b.Fatalf("new setup scheduler: %v", err)
	}
	if _, err := setupScheduler.Enqueue(ctx, scheduler.Task{
		TaskID: "bench-a19-recovery-task",
		RunID:  "bench-a19-recovery-run",
	}); err != nil {
		b.Fatalf("enqueue setup task failed: %v", err)
	}
	claimed, ok, err := setupScheduler.Claim(ctx, "bench-recovery-worker")
	if err != nil || !ok {
		b.Fatalf("claim setup task failed: ok=%v err=%v", ok, err)
	}
	commit := scheduler.TerminalCommit{
		TaskID:      claimed.Record.Task.TaskID,
		AttemptID:   claimed.Attempt.AttemptID,
		Status:      scheduler.TaskStateSucceeded,
		CommittedAt: time.Now(),
		Result:      map[string]any{"ok": true},
	}
	if _, err := setupScheduler.Complete(ctx, commit); err != nil {
		b.Fatalf("complete setup task failed: %v", err)
	}
	snapshot, err := setupScheduler.Snapshot(ctx)
	if err != nil {
		b.Fatalf("snapshot setup scheduler failed: %v", err)
	}

	durations := make([]int64, 0, b.N)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		start := time.Now()
		replayedScheduler, err := scheduler.New(scheduler.NewMemoryStore(), scheduler.WithLeaseTimeout(500*time.Millisecond))
		if err != nil {
			b.Fatalf("new replay scheduler: %v", err)
		}
		if err := replayedScheduler.Restore(ctx, snapshot); err != nil {
			b.Fatalf("restore snapshot failed: %v", err)
		}
		before, err := replayedScheduler.Stats(ctx)
		if err != nil {
			b.Fatalf("stats before replay failed: %v", err)
		}
		replayResult, err := replayedScheduler.Complete(ctx, commit)
		if err != nil {
			b.Fatalf("replay complete failed: %v", err)
		}
		after, err := replayedScheduler.Stats(ctx)
		if err != nil {
			b.Fatalf("stats after replay failed: %v", err)
		}
		if !replayResult.Duplicate {
			b.Fatalf("replay complete should be duplicate: %#v", replayResult)
		}
		if before.CompleteTotal != after.CompleteTotal {
			b.Fatalf("replay should not inflate complete_total before=%d after=%d", before.CompleteTotal, after.CompleteTotal)
		}
		durations = append(durations, time.Since(start).Nanoseconds())
	}
	b.StopTimer()
	p95 := percentileNs(durations, 95)
	if p95 <= 0 && b.N > 0 {
		p95 = b.Elapsed().Nanoseconds() / int64(b.N)
	}
	if p95 <= 0 {
		p95 = 1
	}
	b.ReportMetric(float64(p95), "p95-ns/op")
}

func BenchmarkMCPReconnectOverhead(b *testing.B) {
	attempt := 0
	connector := func(ctx context.Context) (httpmcp.Session, error) {
		attempt++
		if attempt%2 == 1 {
			return &fakeHTTPSession{callErr: errors.New("down")}, nil
		}
		return &fakeHTTPSession{}, nil
	}
	client := httpmcp.NewClient(httpmcp.Config{Connect: connector, Retry: 1, Backoff: time.Microsecond})
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = client.CallTool(context.Background(), "tool", nil)
	}
}

func BenchmarkMCPProfileDefaultUnderFailure(b *testing.B) {
	attempt := 0
	connector := func(ctx context.Context) (httpmcp.Session, error) {
		attempt++
		if attempt%3 != 0 {
			return &fakeHTTPSession{callErr: errors.New("flaky")}, nil
		}
		return &fakeHTTPSession{}, nil
	}
	client := httpmcp.NewClient(httpmcp.Config{
		Connect: connector,
		Profile: mcpprofile.Default,
	})
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = client.CallTool(context.Background(), "tool", nil)
	}
}

func BenchmarkMCPProfileHighReliabilityUnderFailure(b *testing.B) {
	attempt := 0
	connector := func(ctx context.Context) (httpmcp.Session, error) {
		attempt++
		if attempt%4 != 0 {
			return &fakeHTTPSession{callErr: errors.New("flaky")}, nil
		}
		return &fakeHTTPSession{}, nil
	}
	client := httpmcp.NewClient(httpmcp.Config{
		Connect: connector,
		Profile: mcpprofile.HighReliab,
	})
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = client.CallTool(context.Background(), "tool", nil)
	}
}

func BenchmarkContextProductionHardeningPressureEvaluation(b *testing.B) {
	cfg := runtimeconfig.DefaultConfig().ContextAssembler
	cfg.Enabled = true
	cfg.CA3.Enabled = true
	cfg.CA3.Tokenizer.Mode = "estimate_only"
	cfg.CA3.MaxContextTokens = 4096
	cfg.CA3.PercentThresholds = runtimeconfig.ContextAssemblerCA3Thresholds{
		Safe: 20, Comfort: 40, Warning: 60, Danger: 75, Emergency: 90,
	}
	cfg.CA3.AbsoluteThresholds = runtimeconfig.ContextAssemblerCA3Thresholds{
		Safe: 512, Comfort: 1024, Warning: 2048, Danger: 3072, Emergency: 3584,
	}
	a := assembler.New(func() runtimeconfig.ContextAssemblerConfig { return cfg })
	req := types.ContextAssembleRequest{
		RunID:         "bench-context-production-hardening",
		SessionID:     "bench-session",
		PrefixVersion: "ca1",
		Messages: []types.Message{
			{Role: "system", Content: "stable system prompt"},
			{Role: "user", Content: strings.Repeat("payload ", 80)},
		},
	}
	modelReq := types.ModelRequest{
		RunID:    req.RunID,
		Model:    "gpt-4.1-mini",
		Input:    strings.Repeat("bench input ", 160),
		Messages: append([]types.Message(nil), req.Messages...),
	}

	durations := make([]int64, 0, b.N)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		start := time.Now()
		if _, _, err := a.Assemble(context.Background(), req, modelReq); err != nil {
			b.Fatalf("assemble failed: %v", err)
		}
		durations = append(durations, time.Since(start).Nanoseconds())
	}
	b.StopTimer()
	if p95 := percentileNs(durations, 95); p95 > 0 {
		b.ReportMetric(float64(p95), "p95-ns/op")
	}
}

func BenchmarkContextPressureSemanticCompactionLatency(b *testing.B) {
	cfg := runtimeconfig.DefaultConfig().ContextAssembler
	cfg.Enabled = true
	cfg.CA3.Enabled = true
	cfg.CA3.MaxContextTokens = 4096
	cfg.CA3.Compaction.Mode = "semantic"
	cfg.CA3.Compaction.Quality.Threshold = 0.2
	cfg.CA3.Compaction.Evidence.Keywords = []string{"decision", "risk"}
	cfg.CA3.Tokenizer.Mode = "estimate_only"
	a := assembler.New(func() runtimeconfig.ContextAssemblerConfig { return cfg })

	model := fakes.NewModel([]fakes.ModelStep{
		{Response: types.ModelResponse{FinalAnswer: "decision preserved with compact risk summary"}},
	})
	req := types.ContextAssembleRequest{
		RunID:         "bench-context-pressure-semantic",
		SessionID:     "bench-session",
		PrefixVersion: "ca1",
		Input:         strings.Repeat("input payload ", 80),
		Messages: []types.Message{
			{Role: "system", Content: "stable system prompt"},
			{Role: "user", Content: strings.Repeat("decision and risk details. ", 80)},
		},
		ModelClient: model,
	}
	modelReq := types.ModelRequest{
		RunID:    req.RunID,
		Model:    "gpt-4.1-mini",
		Input:    req.Input,
		Messages: append([]types.Message(nil), req.Messages...),
	}

	durations := make([]int64, 0, b.N)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		start := time.Now()
		if _, _, err := a.Assemble(context.Background(), req, modelReq); err != nil {
			b.Fatalf("assemble semantic compaction failed: %v", err)
		}
		durations = append(durations, time.Since(start).Nanoseconds())
	}
	b.StopTimer()
	if p95 := percentileNs(durations, 95); p95 > 0 {
		b.ReportMetric(float64(p95), "p95-ns/op")
	}
}

type benchEmbeddingScorer func(ctx context.Context, req assembler.SemanticEmbeddingScoreRequest) (float64, error)

func (f benchEmbeddingScorer) Score(ctx context.Context, req assembler.SemanticEmbeddingScoreRequest) (float64, error) {
	return f(ctx, req)
}

type benchReranker func(ctx context.Context, req assembler.SemanticRerankRequest) (assembler.SemanticRerankResult, error)

func (f benchReranker) Rerank(ctx context.Context, req assembler.SemanticRerankRequest) (assembler.SemanticRerankResult, error) {
	return f(ctx, req)
}

func BenchmarkContextPressureSemanticCompactionLatencyEmbeddingEnabled(b *testing.B) {
	cfg := runtimeconfig.DefaultConfig().ContextAssembler
	cfg.Enabled = true
	cfg.CA3.Enabled = true
	cfg.CA3.MaxContextTokens = 4096
	cfg.CA3.Compaction.Mode = "semantic"
	cfg.CA3.Compaction.Quality.Threshold = 0.2
	cfg.CA3.Compaction.Evidence.Keywords = []string{"decision", "risk"}
	cfg.CA3.Tokenizer.Mode = "estimate_only"
	cfg.CA3.Compaction.Embedding.Enabled = true
	cfg.CA3.Compaction.Embedding.Selector = "bench"
	cfg.CA3.Compaction.Embedding.Provider = "openai"
	cfg.CA3.Compaction.Embedding.Model = "text-embedding-3-small"
	cfg.CA3.Compaction.Embedding.Timeout = 200 * time.Millisecond
	cfg.CA3.Compaction.Embedding.SimilarityMetric = "cosine"
	cfg.CA3.Compaction.Embedding.RuleWeight = 0.7
	cfg.CA3.Compaction.Embedding.EmbeddingWeight = 0.3

	scorer := benchEmbeddingScorer(func(ctx context.Context, req assembler.SemanticEmbeddingScoreRequest) (float64, error) {
		_ = ctx
		_ = req
		return 0.82, nil
	})
	a := assembler.New(func() runtimeconfig.ContextAssemblerConfig { return cfg }, assembler.WithSemanticEmbeddingScorer("", scorer))

	model := fakes.NewModel([]fakes.ModelStep{
		{Response: types.ModelResponse{FinalAnswer: "decision preserved with compact risk summary"}},
	})
	req := types.ContextAssembleRequest{
		RunID:         "bench-context-pressure-semantic-embedding",
		SessionID:     "bench-session",
		PrefixVersion: "ca1",
		Input:         strings.Repeat("input payload ", 80),
		Messages: []types.Message{
			{Role: "system", Content: "stable system prompt"},
			{Role: "user", Content: strings.Repeat("decision and risk details. ", 80)},
		},
		ModelClient: model,
	}
	modelReq := types.ModelRequest{
		RunID:    req.RunID,
		Model:    "gpt-4.1-mini",
		Input:    req.Input,
		Messages: append([]types.Message(nil), req.Messages...),
	}

	durations := make([]int64, 0, b.N)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		start := time.Now()
		if _, _, err := a.Assemble(context.Background(), req, modelReq); err != nil {
			b.Fatalf("assemble semantic compaction with embedding failed: %v", err)
		}
		durations = append(durations, time.Since(start).Nanoseconds())
	}
	b.StopTimer()
	if p95 := percentileNs(durations, 95); p95 > 0 {
		b.ReportMetric(float64(p95), "p95-ns/op")
	}
}

func BenchmarkContextPressureSemanticCompactionLatencyRerankerGovernanceEnabled(b *testing.B) {
	cfg := runtimeconfig.DefaultConfig().ContextAssembler
	cfg.Enabled = true
	cfg.CA3.Enabled = true
	cfg.CA3.MaxContextTokens = 4096
	cfg.CA3.Compaction.Mode = "semantic"
	cfg.CA3.Compaction.Quality.Threshold = 0.2
	cfg.CA3.Compaction.Evidence.Keywords = []string{"decision", "risk"}
	cfg.CA3.Tokenizer.Mode = "estimate_only"
	cfg.CA3.Compaction.Embedding.Enabled = true
	cfg.CA3.Compaction.Embedding.Selector = "bench"
	cfg.CA3.Compaction.Embedding.Provider = "anthropic"
	cfg.CA3.Compaction.Embedding.Model = "claude-3-haiku"
	cfg.CA3.Compaction.Embedding.Timeout = 200 * time.Millisecond
	cfg.CA3.Compaction.Reranker.Enabled = true
	cfg.CA3.Compaction.Reranker.Timeout = 120 * time.Millisecond
	cfg.CA3.Compaction.Reranker.MaxRetries = 0
	cfg.CA3.Compaction.Reranker.ThresholdProfiles = map[string]float64{
		"anthropic:claude-3-haiku": 0.68,
	}
	cfg.CA3.Compaction.Reranker.Governance.Mode = runtimeconfig.CA3RerankerGovernanceModeEnforce
	cfg.CA3.Compaction.Reranker.Governance.ProfileVersion = "bench-v1"
	cfg.CA3.Compaction.Reranker.Governance.RolloutProviderModels = []string{"anthropic:claude-3-haiku"}

	reranker := benchReranker(func(ctx context.Context, req assembler.SemanticRerankRequest) (assembler.SemanticRerankResult, error) {
		_ = ctx
		return assembler.SemanticRerankResult{Score: req.CurrentScore}, nil
	})
	a := assembler.New(
		func() runtimeconfig.ContextAssemblerConfig { return cfg },
		assembler.WithSemanticReranker("anthropic", reranker),
	)
	model := fakes.NewModel([]fakes.ModelStep{
		{Response: types.ModelResponse{FinalAnswer: "decision preserved with compact risk summary"}},
	})
	req := types.ContextAssembleRequest{
		RunID:         "bench-context-pressure-semantic-reranker-governance",
		SessionID:     "bench-session",
		PrefixVersion: "ca1",
		Input:         strings.Repeat("input payload ", 80),
		Messages: []types.Message{
			{Role: "system", Content: "stable system prompt"},
			{Role: "user", Content: strings.Repeat("decision and risk details. ", 80)},
		},
		ModelClient: model,
	}
	modelReq := types.ModelRequest{
		RunID:    req.RunID,
		Model:    "claude-3-haiku",
		Input:    req.Input,
		Messages: append([]types.Message(nil), req.Messages...),
	}

	durations := make([]int64, 0, b.N)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		start := time.Now()
		if _, _, err := a.Assemble(context.Background(), req, modelReq); err != nil {
			b.Fatalf("assemble semantic compaction with reranker governance failed: %v", err)
		}
		durations = append(durations, time.Since(start).Nanoseconds())
	}
	b.StopTimer()
	if p95 := percentileNs(durations, 95); p95 > 0 {
		b.ReportMetric(float64(p95), "p95-ns/op")
	}
}

func BenchmarkDiagnosticsTimelineTrendQuery(b *testing.B) {
	store := runtimediag.NewStore(64, 512, 16, 32, runtimediag.TimelineTrendConfig{
		Enabled:    true,
		LastNRuns:  100,
		TimeWindow: 15 * time.Minute,
	}, runtimediag.ContextStage2ExternalTrendConfig{Enabled: true, Window: 15 * time.Minute})
	now := time.Now()
	for i := 0; i < 400; i++ {
		runID := fmt.Sprintf("bench-run-%d", i)
		start := now.Add(time.Duration(i) * time.Millisecond)
		end := start.Add(time.Duration((i%9)+1) * 3 * time.Millisecond)
		store.AddTimelineEvent(runID, "model", "running", int64(i*2+1), start)
		store.AddTimelineEvent(runID, "model", "succeeded", int64(i*2+2), end)
		store.AddRun(runtimediag.RunRecord{
			Time:      end,
			RunID:     runID,
			Status:    "success",
			LatencyMs: end.Sub(start).Milliseconds(),
		})
	}
	durations := make([]int64, 0, b.N)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		start := time.Now()
		items := store.TimelineTrends(runtimediag.TimelineTrendQuery{
			Mode:      runtimediag.TimelineTrendModeLastNRuns,
			LastNRuns: 100,
		})
		if len(items) == 0 {
			b.Fatalf("trend query returned empty result")
		}
		for _, item := range items {
			if item.LatencyP95Ms < 0 {
				b.Fatalf("invalid latency_p95_ms: %#v", item)
			}
		}
		durations = append(durations, time.Since(start).Nanoseconds())
	}
	b.StopTimer()
	if p95 := percentileNs(durations, 95); p95 > 0 {
		b.ReportMetric(float64(p95), "p95-ns/op")
	}
}

type diagnosticsQueryBenchmarkFixture struct {
	store               *runtimediag.Store
	runsRequest         runtimediag.UnifiedRunQueryRequest
	sandboxRunsRequest  runtimediag.UnifiedRunQueryRequest
	mailboxRequest      runtimediag.MailboxQueryRequest
	aggregateRequest    runtimediag.MailboxAggregateRequest
	expectedRunsPage    int
	expectedSandboxPage int
	expectedMailboxPage int
}

var (
	diagnosticsQueryBenchFixtureOnce sync.Once
	diagnosticsQueryBenchFixture     diagnosticsQueryBenchmarkFixture
)

func newDiagnosticsQueryBenchmarkFixture() diagnosticsQueryBenchmarkFixture {
	const (
		diagnosticsBenchRunTotal     = 1200
		diagnosticsBenchSandboxTotal = 900
		diagnosticsBenchMailboxTotal = 4800
	)
	base := time.Date(2026, time.January, 2, 12, 0, 0, 0, time.UTC)
	store := runtimediag.NewStore(64, diagnosticsBenchMailboxTotal+500, 16, 32, runtimediag.TimelineTrendConfig{
		Enabled:    true,
		LastNRuns:  100,
		TimeWindow: 15 * time.Minute,
	}, runtimediag.ContextStage2ExternalTrendConfig{
		Enabled: true,
		Window:  15 * time.Minute,
	})

	teams := []string{"team-a", "team-b", "team-c", "team-d"}
	workflows := []string{"wf-alpha", "wf-beta", "wf-gamma"}
	statuses := []string{"success", "success", "success", "success", "success", "success", "failed"}

	for i := 0; i < diagnosticsBenchRunTotal; i++ {
		teamID := teams[i%len(teams)]
		workflowID := workflows[i%len(workflows)]
		store.AddRun(runtimediag.RunRecord{
			Time:       base.Add(time.Duration(i) * time.Millisecond),
			RunID:      fmt.Sprintf("diag-run-%05d", i),
			Status:     statuses[i%len(statuses)],
			Iterations: (i % 5) + 1,
			ToolCalls:  (i % 9) + 1,
			LatencyMs:  int64((i%30)+10) * 3,
			TeamID:     teamID,
			WorkflowID: workflowID,
			TaskID:     fmt.Sprintf("task-%03d", i%360),
		})
	}

	sandboxDecisions := []string{"deny", "sandbox", "sandbox", "sandbox", "deny"}
	sandboxReasons := []string{"sandbox.timeout", "sandbox.runtime_launch_failed", "sandbox.capability_mismatch"}
	sandboxRolloutPhases := []string{"observe", "canary", "baseline", "full", "frozen"}
	sandboxCapacityActions := []string{"allow", "throttle", "deny"}
	sandboxHealthBudgetStatuses := []string{"within_budget", "near_budget", "breached"}
	for i := 0; i < diagnosticsBenchSandboxTotal; i++ {
		status := "success"
		decision := sandboxDecisions[i%len(sandboxDecisions)]
		reason := sandboxReasons[i%len(sandboxReasons)]
		if decision == "deny" {
			status = "failed"
		}
		rolloutPhase := sandboxRolloutPhases[i%len(sandboxRolloutPhases)]
		capacityAction := sandboxCapacityActions[(i+1)%len(sandboxCapacityActions)]
		healthBudgetStatus := sandboxHealthBudgetStatuses[(i+2)%len(sandboxHealthBudgetStatuses)]
		rolloutEffectiveRatio := map[string]float64{
			"observe":  0.0,
			"canary":   0.1,
			"baseline": 0.5,
			"full":     1.0,
			"frozen":   0.0,
		}[rolloutPhase]
		healthBudgetBreachTotal := 0
		switch healthBudgetStatus {
		case "near_budget":
			healthBudgetBreachTotal = 1
		case "breached":
			healthBudgetBreachTotal = 3
		}
		freezeState := rolloutPhase == "frozen" || healthBudgetStatus == "breached"
		freezeReasonCode := ""
		if freezeState {
			freezeReasonCode = "sandbox.rollout.health_budget_breached"
		}
		store.AddRun(runtimediag.RunRecord{
			Time:                              base.Add(20*time.Second + time.Duration(i)*2*time.Millisecond),
			RunID:                             fmt.Sprintf("diag-sandbox-run-%05d", i),
			Status:                            status,
			Iterations:                        (i % 4) + 1,
			ToolCalls:                         (i % 6) + 1,
			LatencyMs:                         int64((i%45)+20) * 2,
			TeamID:                            "team-sandbox",
			WorkflowID:                        "wf-sandbox",
			TaskID:                            fmt.Sprintf("sandbox-task-%03d", i%400),
			PolicyKind:                        "sandbox",
			Decision:                          "deny",
			ReasonCode:                        reason,
			AlertDispatchStatus:               "succeeded",
			AlertDeliveryMode:                 "sync",
			AlertRetryCount:                   0,
			AlertCircuitState:                 "closed",
			SandboxMode:                       "enforce",
			SandboxBackend:                    "windows_job",
			SandboxProfile:                    "default",
			SandboxSessionMode:                "per_call",
			SandboxRequiredCapabilities:       []string{"stdout_stderr_capture", "network_off"},
			SandboxDecision:                   decision,
			SandboxReasonCode:                 reason,
			SandboxFallbackUsed:               i%11 == 0,
			SandboxTimeoutTotal:               i % 2,
			SandboxLaunchFailedTotal:          i % 3,
			SandboxCapabilityMismatchTotal:    i % 4,
			SandboxQueueWaitMsP95:             int64((i % 17) + 1),
			SandboxExecLatencyMsP95:           int64((i % 23) + 3),
			SandboxExitCodeLast:               124,
			SandboxOOMTotal:                   i % 2,
			SandboxResourceCPUMsTotal:         int64((i % 57) + 20),
			SandboxResourceMemoryPeakBytesP95: int64((i%4096)+2048) * 4,
			SandboxRolloutPhase:               rolloutPhase,
			SandboxRolloutEffectiveRatio:      rolloutEffectiveRatio,
			SandboxHealthBudgetStatus:         healthBudgetStatus,
			SandboxHealthBudgetBreachTotal:    healthBudgetBreachTotal,
			SandboxFreezeState:                freezeState,
			SandboxFreezeReasonCode:           freezeReasonCode,
			SandboxCapacityAction:             capacityAction,
			SandboxCapacityQueueDepth:         40 + (i % 90),
			SandboxCapacityInflight:           8 + (i % 24),
			RuntimePrimaryDomain:              "scheduler",
			RuntimePrimaryCode:                "scheduler.backend.fallback",
			RuntimePrimarySource:              "runtime.readiness",
			RuntimePrimaryConflictTotal:       0,
			RuntimeSecondaryReasonCodes:       []string{},
			RuntimeSecondaryReasonCount:       0,
			RuntimeArbitrationRuleVersion:     runtimeconfig.RuntimeArbitrationRuleVersionExplainabilityV1,
			RuntimeRemediationHintCode:        "scheduler.recover_backend",
			RuntimeRemediationHintDomain:      "scheduler",
		})
	}

	kinds := []string{"command", "event", "result"}
	states := []string{"queued", "in_flight", "acked", "nacked", "dead_letter", "expired"}
	reasons := []string{"retry_exhausted", "handler_error", "consumer_mismatch", "message_not_found", "timeout", "expired"}
	for i := 0; i < diagnosticsBenchMailboxTotal; i++ {
		runIdx := i % diagnosticsBenchRunTotal
		teamID := teams[runIdx%len(teams)]
		workflowID := workflows[runIdx%len(workflows)]
		store.AddMailbox(runtimediag.MailboxRecord{
			Time:                  base.Add(20*time.Millisecond + time.Duration(i)*500*time.Microsecond),
			MessageID:             fmt.Sprintf("mailbox-msg-%05d", i%8000),
			IdempotencyKey:        fmt.Sprintf("mailbox-key-%05d", i%12000),
			CorrelationID:         fmt.Sprintf("corr-%03d", runIdx%400),
			Kind:                  kinds[i%len(kinds)],
			State:                 states[i%len(states)],
			FromAgent:             fmt.Sprintf("agent-%d", i%17),
			ToAgent:               fmt.Sprintf("agent-%d", (i+3)%17),
			RunID:                 fmt.Sprintf("diag-run-%05d", runIdx),
			TaskID:                fmt.Sprintf("task-%03d", runIdx%360),
			WorkflowID:            workflowID,
			TeamID:                teamID,
			Attempt:               (i % 3) + 1,
			ConsumerID:            fmt.Sprintf("consumer-%d", i%7),
			ReasonCode:            reasons[(i/2)%len(reasons)],
			Backend:               "memory",
			ConfiguredBackend:     "memory",
			BackendFallback:       false,
			BackendFallbackReason: "",
			PublishPath:           "runtime",
			Reclaimed:             i%11 == 0,
			PanicRecovered:        i%97 == 0,
		})
	}

	runPageSize := 40
	sandboxPageSize := 32
	mailboxPageSize := 30
	return diagnosticsQueryBenchmarkFixture{
		store: store,
		runsRequest: runtimediag.UnifiedRunQueryRequest{
			TeamID:     "team-b",
			WorkflowID: "wf-beta",
			Status:     "success",
			TimeRange: &runtimediag.UnifiedQueryTimeRange{
				Start: base.Add(500 * time.Millisecond),
				End:   base.Add(4500 * time.Millisecond),
			},
			PageSize: &runPageSize,
			Sort:     runtimediag.UnifiedQuerySort{Field: "time", Order: "asc"},
		},
		sandboxRunsRequest: runtimediag.UnifiedRunQueryRequest{
			TeamID:     "team-sandbox",
			WorkflowID: "wf-sandbox",
			TimeRange: &runtimediag.UnifiedQueryTimeRange{
				Start: base.Add(20*time.Second + 10*time.Millisecond),
				End:   base.Add(22*time.Second + 500*time.Millisecond),
			},
			PageSize: &sandboxPageSize,
			Sort:     runtimediag.UnifiedQuerySort{Field: "time", Order: "asc"},
		},
		mailboxRequest: runtimediag.MailboxQueryRequest{
			Kind:       "command",
			State:      "queued",
			TeamID:     "team-c",
			WorkflowID: "wf-alpha",
			TimeRange: &runtimediag.MailboxQueryTimeRange{
				Start: base.Add(1 * time.Second),
				End:   base.Add(10 * time.Second),
			},
			PageSize: &mailboxPageSize,
			Sort:     runtimediag.MailboxQuerySort{Field: "time", Order: "desc"},
		},
		aggregateRequest: runtimediag.MailboxAggregateRequest{
			TeamID:     "team-c",
			WorkflowID: "wf-alpha",
			TimeRange: &runtimediag.MailboxQueryTimeRange{
				Start: base.Add(1 * time.Second),
				End:   base.Add(10 * time.Second),
			},
		},
		expectedRunsPage:    runPageSize,
		expectedSandboxPage: sandboxPageSize,
		expectedMailboxPage: mailboxPageSize,
	}
}

func diagnosticsQueryBenchmarkFixtureSnapshot() diagnosticsQueryBenchmarkFixture {
	diagnosticsQueryBenchFixtureOnce.Do(func() {
		diagnosticsQueryBenchFixture = newDiagnosticsQueryBenchmarkFixture()
	})
	return diagnosticsQueryBenchFixture
}

func BenchmarkDiagnosticsQueryRuns(b *testing.B) {
	fixture := diagnosticsQueryBenchmarkFixtureSnapshot()
	store := fixture.store
	durations := make([]int64, 0, b.N)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		start := time.Now()
		first, err := store.QueryRuns(fixture.runsRequest)
		if err != nil {
			b.Fatalf("query runs first page failed: %v", err)
		}
		if len(first.Items) != fixture.expectedRunsPage {
			b.Fatalf("query runs first page size mismatch: want=%d got=%d", fixture.expectedRunsPage, len(first.Items))
		}
		if first.NextCursor == "" {
			b.Fatalf("query runs first page should include next cursor")
		}
		if first.SortOrder != "asc" {
			b.Fatalf("query runs sort order mismatch: %s", first.SortOrder)
		}
		if first.Items[0].Time.After(first.Items[len(first.Items)-1].Time) {
			b.Fatalf("query runs first page not sorted ascending")
		}
		nextReq := fixture.runsRequest
		nextReq.Cursor = first.NextCursor
		second, err := store.QueryRuns(nextReq)
		if err != nil {
			b.Fatalf("query runs second page failed: %v", err)
		}
		if len(second.Items) == 0 {
			b.Fatalf("query runs second page should not be empty")
		}
		if !second.Items[0].Time.After(first.Items[len(first.Items)-1].Time) {
			b.Fatalf("query runs second page should advance cursor window")
		}
		durations = append(durations, time.Since(start).Nanoseconds())
	}
	b.StopTimer()
	if p95 := percentileNs(durations, 95); p95 > 0 {
		b.ReportMetric(float64(p95), "p95-ns/op")
	}
}

func BenchmarkDiagnosticsQueryRunsSandboxEnriched(b *testing.B) {
	fixture := diagnosticsQueryBenchmarkFixtureSnapshot()
	store := fixture.store
	durations := make([]int64, 0, b.N)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		start := time.Now()
		first, err := store.QueryRuns(fixture.sandboxRunsRequest)
		if err != nil {
			b.Fatalf("query sandbox runs first page failed: %v", err)
		}
		if len(first.Items) != fixture.expectedSandboxPage {
			b.Fatalf("query sandbox runs first page size mismatch: want=%d got=%d", fixture.expectedSandboxPage, len(first.Items))
		}
		if first.NextCursor == "" {
			b.Fatalf("query sandbox runs first page should include next cursor")
		}
		if first.Items[0].Time.After(first.Items[len(first.Items)-1].Time) {
			b.Fatalf("query sandbox runs first page not sorted ascending")
		}
		phaseSeen := map[string]struct{}{}
		capacityActionSeen := map[string]struct{}{}
		healthBudgetSeen := map[string]struct{}{}
		for _, item := range first.Items {
			if item.PolicyKind != "sandbox" || item.SandboxMode != "enforce" || len(item.SandboxRequiredCapabilities) == 0 {
				b.Fatalf("sandbox-enriched record missing mandatory fields: %#v", item)
			}
			if item.SandboxRolloutPhase == "" || item.SandboxCapacityAction == "" || item.SandboxHealthBudgetStatus == "" {
				b.Fatalf("sandbox rollout governance record missing additive fields: %#v", item)
			}
			phaseSeen[item.SandboxRolloutPhase] = struct{}{}
			capacityActionSeen[item.SandboxCapacityAction] = struct{}{}
			healthBudgetSeen[item.SandboxHealthBudgetStatus] = struct{}{}
			if item.SandboxFreezeState && item.SandboxFreezeReasonCode == "" {
				b.Fatalf("sandbox freeze state set without freeze reason code: %#v", item)
			}
		}
		if len(phaseSeen) != 5 {
			b.Fatalf("sandbox-enriched page must include mixed rollout phases, got=%d", len(phaseSeen))
		}
		if len(capacityActionSeen) != 3 {
			b.Fatalf("sandbox-enriched page must include mixed capacity actions, got=%d", len(capacityActionSeen))
		}
		if len(healthBudgetSeen) != 3 {
			b.Fatalf("sandbox-enriched page must include mixed health budget states, got=%d", len(healthBudgetSeen))
		}
		nextReq := fixture.sandboxRunsRequest
		nextReq.Cursor = first.NextCursor
		second, err := store.QueryRuns(nextReq)
		if err != nil {
			b.Fatalf("query sandbox runs second page failed: %v", err)
		}
		if len(second.Items) == 0 {
			b.Fatalf("query sandbox runs second page should not be empty")
		}
		if !second.Items[0].Time.After(first.Items[len(first.Items)-1].Time) {
			b.Fatalf("query sandbox runs second page should advance cursor window")
		}
		durations = append(durations, time.Since(start).Nanoseconds())
	}
	b.StopTimer()
	if p95 := percentileNs(durations, 95); p95 > 0 {
		b.ReportMetric(float64(p95), "p95-ns/op")
	}
}

func BenchmarkDiagnosticsQueryMailbox(b *testing.B) {
	fixture := diagnosticsQueryBenchmarkFixtureSnapshot()
	store := fixture.store
	durations := make([]int64, 0, b.N)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		start := time.Now()
		first, err := store.QueryMailbox(fixture.mailboxRequest)
		if err != nil {
			b.Fatalf("query mailbox first page failed: %v", err)
		}
		if len(first.Items) != fixture.expectedMailboxPage {
			b.Fatalf("query mailbox first page size mismatch: want=%d got=%d", fixture.expectedMailboxPage, len(first.Items))
		}
		if first.NextCursor == "" {
			b.Fatalf("query mailbox first page should include next cursor")
		}
		if first.SortOrder != "desc" {
			b.Fatalf("query mailbox sort order mismatch: %s", first.SortOrder)
		}
		if first.Items[0].Time.Before(first.Items[len(first.Items)-1].Time) {
			b.Fatalf("query mailbox first page not sorted descending")
		}
		nextReq := fixture.mailboxRequest
		nextReq.Cursor = first.NextCursor
		second, err := store.QueryMailbox(nextReq)
		if err != nil {
			b.Fatalf("query mailbox second page failed: %v", err)
		}
		if len(second.Items) == 0 {
			b.Fatalf("query mailbox second page should not be empty")
		}
		if second.Items[0].Time.After(first.Items[len(first.Items)-1].Time) {
			b.Fatalf("query mailbox second page should advance cursor window")
		}
		durations = append(durations, time.Since(start).Nanoseconds())
	}
	b.StopTimer()
	if p95 := percentileNs(durations, 95); p95 > 0 {
		b.ReportMetric(float64(p95), "p95-ns/op")
	}
}

func BenchmarkDiagnosticsMailboxAggregates(b *testing.B) {
	fixture := diagnosticsQueryBenchmarkFixtureSnapshot()
	store := fixture.store
	durations := make([]int64, 0, b.N)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		start := time.Now()
		agg := store.MailboxAggregates(fixture.aggregateRequest)
		if agg.TotalRecords == 0 {
			b.Fatalf("mailbox aggregate should include records")
		}
		if agg.TotalMessages == 0 {
			b.Fatalf("mailbox aggregate should include messages")
		}
		if len(agg.ByKind) == 0 || len(agg.ByState) == 0 {
			b.Fatalf("mailbox aggregate should include kind/state distributions")
		}
		if len(agg.ReasonCodeTotals) == 0 {
			b.Fatalf("mailbox aggregate should include reason code totals")
		}
		durations = append(durations, time.Since(start).Nanoseconds())
	}
	b.StopTimer()
	if p95 := percentileNs(durations, 95); p95 > 0 {
		b.ReportMetric(float64(p95), "p95-ns/op")
	}
}

func BenchmarkContextStage2ExternalRetrieverTrendAggregation(b *testing.B) {
	store := runtimediag.NewStore(64, 1024, 16, 32,
		runtimediag.TimelineTrendConfig{Enabled: true, LastNRuns: 100, TimeWindow: 15 * time.Minute},
		runtimediag.ContextStage2ExternalTrendConfig{
			Enabled: true,
			Window:  15 * time.Minute,
			Thresholds: runtimediag.ContextStage2ExternalThresholds{
				P95LatencyMs: 1000,
				ErrorRate:    0.15,
				HitRate:      0.30,
			},
		},
	)
	now := time.Now()
	for i := 0; i < 600; i++ {
		provider := "http"
		if i%2 == 0 {
			provider = "rag"
		}
		code := "ok"
		layer := ""
		hitCount := 1
		if i%5 == 0 {
			code = "timeout"
			layer = "transport"
			hitCount = 0
		}
		store.AddRun(runtimediag.RunRecord{
			Time:             now.Add(time.Duration(i) * time.Millisecond),
			RunID:            fmt.Sprintf("context-stage2-run-%d", i),
			Stage2Provider:   provider,
			Stage2LatencyMs:  int64((i%11)+1) * 35,
			Stage2HitCount:   hitCount,
			Stage2ReasonCode: code,
			Stage2ErrorLayer: layer,
		})
	}
	durations := make([]int64, 0, b.N)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		start := time.Now()
		items := store.ContextStage2ExternalTrends(runtimediag.ContextStage2ExternalTrendQuery{})
		if len(items) == 0 {
			b.Fatalf("context stage2 external trend result is empty")
		}
		for _, item := range items {
			if item.P95LatencyMs < 0 || item.ErrorRate < 0 || item.HitRate < 0 {
				b.Fatalf("invalid trend record: %#v", item)
			}
		}
		durations = append(durations, time.Since(start).Nanoseconds())
	}
	b.StopTimer()
	if p95 := percentileNs(durations, 95); p95 > 0 {
		b.ReportMetric(float64(p95), "p95-ns/op")
	}
}

func BenchmarkContextStage2HintTemplateResolution(b *testing.B) {
	cfg := runtimeconfig.DefaultConfig().ContextAssembler.CA2.Stage2.External
	cfg.Profile = runtimeconfig.ContextStage2ExternalProfileRAGFlowLike
	cfg.Endpoint = "http://127.0.0.1:8080/retrieve"
	cfg.Mapping.Request.QueryField = "payload.query"
	cfg.Hints.Enabled = true
	cfg.Hints.Capabilities = []string{"metadata_filter", "rerank_metadata"}

	durations := make([]int64, 0, b.N)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		start := time.Now()
		result := runtimeconfig.PrecheckStage2External(runtimeconfig.ContextStage2ProviderHTTP, cfg)
		if err := result.FirstError(); err != nil {
			b.Fatalf("precheck failed: %v", err)
		}
		if result.Normalized.TemplateResolutionSource == "" {
			b.Fatalf("template_resolution_source should not be empty: %#v", result.Normalized)
		}
		durations = append(durations, time.Since(start).Nanoseconds())
	}
	b.StopTimer()
	if p95 := percentileNs(durations, 95); p95 > 0 {
		b.ReportMetric(float64(p95), "p95-ns/op")
	}
}

func percentileNs(values []int64, percentile int) int64 {
	if len(values) == 0 || percentile <= 0 {
		return 0
	}
	copyVals := append([]int64(nil), values...)
	sort.Slice(copyVals, func(i, j int) bool { return copyVals[i] < copyVals[j] })
	idx := (len(copyVals)*percentile + 99) / 100
	if idx <= 0 {
		idx = 1
	}
	if idx > len(copyVals) {
		idx = len(copyVals)
	}
	return copyVals[idx-1]
}

type fakeHTTPSession struct{ callErr error }

func (s *fakeHTTPSession) ListTools(ctx context.Context, p *mcp.ListToolsParams) (*mcp.ListToolsResult, error) {
	return &mcp.ListToolsResult{}, nil
}
func (s *fakeHTTPSession) CallTool(ctx context.Context, p *mcp.CallToolParams) (*mcp.CallToolResult, error) {
	if s.callErr != nil {
		return nil, s.callErr
	}
	return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: "ok"}}}, nil
}
func (s *fakeHTTPSession) Close() error { return nil }
