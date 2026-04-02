package integration

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/FelixSeptem/baymax/core/types"
	"github.com/FelixSeptem/baymax/integration/fakes"
	"github.com/FelixSeptem/baymax/observability/event"
	"github.com/FelixSeptem/baymax/orchestration/composer"
	runtimeconfig "github.com/FelixSeptem/baymax/runtime/config"
	runtimediag "github.com/FelixSeptem/baymax/runtime/diagnostics"
)

func TestRuntimeBudgetAdmissionContractMixedCostLatencyRunStreamDegradeParity(t *testing.T) {
	cfgPath := filepath.Join(t.TempDir(), "runtime-a60-budget-degrade.yaml")
	writeRuntimeBudgetAdmissionConfig(t, cfgPath, 0.75, 1.2, "800ms", "2s")

	mgr, err := runtimeconfig.NewManager(runtimeconfig.ManagerOptions{
		FilePath:  cfgPath,
		EnvPrefix: "BAYMAX_A60_TEST",
	})
	if err != nil {
		t.Fatalf("new runtime manager: %v", err)
	}
	defer func() { _ = mgr.Close() }()

	model := fakes.NewModel([]fakes.ModelStep{{Response: types.ModelResponse{FinalAnswer: "ok"}}})
	dispatcher := event.NewDispatcher(event.NewRuntimeRecorder(mgr))
	comp, err := composer.NewBuilder(model).
		WithRuntimeManager(mgr).
		WithEventHandler(dispatcherHandler{dispatcher: dispatcher}).
		Build()
	if err != nil {
		t.Fatalf("new composer: %v", err)
	}

	runReq := budgetAdmissionRunRequest("run-a60-budget-degrade-run")
	runRes, runErr := comp.Run(context.Background(), runReq, nil)
	if runErr != nil || runRes.Error != nil {
		t.Fatalf("budget degrade run should pass with degraded decision, err=%v result=%#v", runErr, runRes)
	}
	streamReq := budgetAdmissionRunRequest("run-a60-budget-degrade-stream")
	streamRes, streamErr := comp.Stream(context.Background(), streamReq, nil)
	if streamErr != nil || streamRes.Error != nil {
		t.Fatalf("budget degrade stream should pass with degraded decision, err=%v result=%#v", streamErr, streamRes)
	}

	runRecord := mustFindRunRecordByID(t, mgr, runReq.RunID)
	streamRecord := mustFindRunRecordByID(t, mgr, streamReq.RunID)
	if runRecord.BudgetDecision != string(runtimeconfig.RuntimeAdmissionBudgetDecisionDegrade) ||
		streamRecord.BudgetDecision != string(runtimeconfig.RuntimeAdmissionBudgetDecisionDegrade) {
		t.Fatalf("budget_decision mismatch run=%q stream=%q", runRecord.BudgetDecision, streamRecord.BudgetDecision)
	}
	if runRecord.DegradeAction != runtimeconfig.RuntimeAdmissionDegradeActionReduceToolCallLimit ||
		streamRecord.DegradeAction != runtimeconfig.RuntimeAdmissionDegradeActionReduceToolCallLimit {
		t.Fatalf("degrade_action mismatch run=%q stream=%q", runRecord.DegradeAction, streamRecord.DegradeAction)
	}
	assertBudgetSnapshotMixedContribution(t, runRecord.BudgetSnapshot)
	assertBudgetSnapshotMixedContribution(t, streamRecord.BudgetSnapshot)
}

func TestRuntimeBudgetAdmissionContractHardThresholdDenyRunStreamNoSideEffects(t *testing.T) {
	cfgPath := filepath.Join(t.TempDir(), "runtime-a60-budget-deny.yaml")
	writeRuntimeBudgetAdmissionConfig(t, cfgPath, 0.75, 0.8, "800ms", "840ms")

	mgr, err := runtimeconfig.NewManager(runtimeconfig.ManagerOptions{
		FilePath:  cfgPath,
		EnvPrefix: "BAYMAX_A60_TEST",
	})
	if err != nil {
		t.Fatalf("new runtime manager: %v", err)
	}
	defer func() { _ = mgr.Close() }()

	model := fakes.NewModel([]fakes.ModelStep{{Response: types.ModelResponse{FinalAnswer: "ok"}}})
	dispatcher := event.NewDispatcher(event.NewRuntimeRecorder(mgr))
	comp, err := composer.NewBuilder(model).
		WithRuntimeManager(mgr).
		WithEventHandler(dispatcherHandler{dispatcher: dispatcher}).
		Build()
	if err != nil {
		t.Fatalf("new composer: %v", err)
	}

	before, err := comp.SchedulerStats(context.Background())
	if err != nil {
		t.Fatalf("scheduler stats before deny failed: %v", err)
	}
	mailboxBefore := len(mgr.RecentMailbox(10))

	runReq := budgetAdmissionRunRequest("run-a60-budget-deny-run")
	runRes, runErr := comp.Run(context.Background(), runReq, nil)
	if runErr == nil {
		t.Fatal("budget hard-threshold run should be denied")
	}
	assertAdmissionContractDeniedResult(t, runRes, runtimeconfig.ReadinessAdmissionCodeBudgetHardDeny)

	streamReq := budgetAdmissionRunRequest("run-a60-budget-deny-stream")
	streamRes, streamErr := comp.Stream(context.Background(), streamReq, nil)
	if streamErr == nil {
		t.Fatal("budget hard-threshold stream should be denied")
	}
	assertAdmissionContractDeniedResult(t, streamRes, runtimeconfig.ReadinessAdmissionCodeBudgetHardDeny)

	runDecision, _ := runRes.Error.Details["budget_decision"].(string)
	streamDecision, _ := streamRes.Error.Details["budget_decision"].(string)
	if runDecision != string(runtimeconfig.RuntimeAdmissionBudgetDecisionDeny) ||
		streamDecision != string(runtimeconfig.RuntimeAdmissionBudgetDecisionDeny) {
		t.Fatalf("deny details budget_decision mismatch run=%q stream=%q", runDecision, streamDecision)
	}
	runDegradeAction, _ := runRes.Error.Details["degrade_action"].(string)
	streamDegradeAction, _ := streamRes.Error.Details["degrade_action"].(string)
	if strings.TrimSpace(runDegradeAction) != "" || strings.TrimSpace(streamDegradeAction) != "" {
		t.Fatalf("deny details degrade_action should be empty run=%q stream=%q", runDegradeAction, streamDegradeAction)
	}
	runSnapshot := runtimeBudgetSnapshotFromErrorDetails(t, runRes.Error.Details["budget_snapshot"])
	streamSnapshot := runtimeBudgetSnapshotFromErrorDetails(t, streamRes.Error.Details["budget_snapshot"])
	assertBudgetSnapshotMixedContribution(t, runSnapshot)
	assertBudgetSnapshotMixedContribution(t, streamSnapshot)

	after, err := comp.SchedulerStats(context.Background())
	if err != nil {
		t.Fatalf("scheduler stats after deny failed: %v", err)
	}
	if before.QueueTotal != after.QueueTotal || before.ClaimTotal != after.ClaimTotal || before.ReclaimTotal != after.ReclaimTotal {
		t.Fatalf("deny path should be side-effect free, before=%#v after=%#v", before, after)
	}
	if len(mgr.RecentMailbox(10)) != mailboxBefore {
		t.Fatalf("deny path should not mutate mailbox diagnostics: before=%d after=%d", mailboxBefore, len(mgr.RecentMailbox(10)))
	}

	runRecord := mustFindRunRecordByID(t, mgr, runReq.RunID)
	streamRecord := mustFindRunRecordByID(t, mgr, streamReq.RunID)
	if runRecord.BudgetDecision != string(runtimeconfig.RuntimeAdmissionBudgetDecisionDeny) ||
		streamRecord.BudgetDecision != string(runtimeconfig.RuntimeAdmissionBudgetDecisionDeny) {
		t.Fatalf("run diagnostics budget_decision mismatch run=%q stream=%q", runRecord.BudgetDecision, streamRecord.BudgetDecision)
	}
	if strings.TrimSpace(runRecord.DegradeAction) != "" || strings.TrimSpace(streamRecord.DegradeAction) != "" {
		t.Fatalf("run diagnostics degrade_action should be empty run=%q stream=%q", runRecord.DegradeAction, streamRecord.DegradeAction)
	}
	assertBudgetSnapshotMixedContribution(t, runRecord.BudgetSnapshot)
	assertBudgetSnapshotMixedContribution(t, streamRecord.BudgetSnapshot)
}

func budgetAdmissionRunRequest(runID string) types.RunRequest {
	return types.RunRequest{
		RunID: runID,
		Input: strings.Repeat("a", 1600),
		Messages: []types.Message{
			{Role: "user", Content: "msg-1"},
			{Role: "assistant", Content: "msg-2"},
			{Role: "user", Content: "msg-3"},
		},
		Capabilities: types.CapabilityRequirements{
			Required: []types.ModelCapability{types.ModelCapabilityToolCall},
		},
	}
}

func writeRuntimeBudgetAdmissionConfig(t *testing.T, path string, costDegrade, costHard float64, latencyDegrade, latencyHard string) {
	t.Helper()
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
		fmt.Sprintf("        degrade_threshold: %.2f", costDegrade),
		fmt.Sprintf("        hard_threshold: %.2f", costHard),
		"      latency:",
		"        degrade_threshold: " + strings.TrimSpace(latencyDegrade),
		"        hard_threshold: " + strings.TrimSpace(latencyHard),
		"    degrade_policy:",
		"      enabled: true",
		"      action_order: [reduce_tool_call_limit, trim_memory_context, sandbox_throttle]",
		"      conflict_policy: first_action",
		"security:",
		"  sandbox:",
		"    enabled: true",
		"    required: false",
		"    mode: observe",
		"    policy:",
		"      default_action: host",
		"      profile: default",
		"      fallback_action: allow_and_record",
		"reload:",
		"  enabled: false",
		"  debounce: 20ms",
		"",
	}, "\n")
	if err := os.WriteFile(path, []byte(cfg), 0o600); err != nil {
		t.Fatalf("write runtime config %q: %v", path, err)
	}
}

func mustFindRunRecordByID(t *testing.T, mgr *runtimeconfig.Manager, runID string) runtimediag.RunRecord {
	t.Helper()
	items := mgr.RecentRuns(50)
	for i := range items {
		if strings.TrimSpace(items[i].RunID) == strings.TrimSpace(runID) {
			return items[i]
		}
	}
	t.Fatalf("run record %q not found in %#v", runID, items)
	return runtimediag.RunRecord{}
}

func runtimeBudgetSnapshotFromErrorDetails(t *testing.T, raw any) map[string]any {
	t.Helper()
	switch value := raw.(type) {
	case map[string]any:
		return value
	case *runtimeconfig.RuntimeAdmissionBudgetSnapshot:
		if value == nil {
			return nil
		}
		return map[string]any{
			"version": value.Version,
			"cost_estimate": map[string]any{
				"token":   value.CostEstimate.Token,
				"tool":    value.CostEstimate.Tool,
				"sandbox": value.CostEstimate.Sandbox,
				"memory":  value.CostEstimate.Memory,
				"total":   value.CostEstimate.Total,
			},
			"latency_estimate": map[string]any{
				"token_ms":   value.LatencyEstimate.TokenMs,
				"tool_ms":    value.LatencyEstimate.ToolMs,
				"sandbox_ms": value.LatencyEstimate.SandboxMs,
				"memory_ms":  value.LatencyEstimate.MemoryMs,
				"total_ms":   value.LatencyEstimate.TotalMs,
			},
		}
	default:
		t.Fatalf("unexpected budget_snapshot detail type: %T", raw)
	}
	return nil
}

func assertBudgetSnapshotMixedContribution(t *testing.T, snapshot map[string]any) {
	t.Helper()
	if len(snapshot) == 0 {
		t.Fatalf("budget_snapshot should not be empty: %#v", snapshot)
	}
	version, _ := snapshot["version"].(string)
	if strings.TrimSpace(version) != runtimeconfig.RuntimeAdmissionBudgetSnapshotVersionV1 {
		t.Fatalf("budget_snapshot.version=%q, want %q", version, runtimeconfig.RuntimeAdmissionBudgetSnapshotVersionV1)
	}
	costEstimate, ok := snapshot["cost_estimate"].(map[string]any)
	if !ok {
		t.Fatalf("budget_snapshot.cost_estimate missing or invalid: %#v", snapshot["cost_estimate"])
	}
	latencyEstimate, ok := snapshot["latency_estimate"].(map[string]any)
	if !ok {
		t.Fatalf("budget_snapshot.latency_estimate missing or invalid: %#v", snapshot["latency_estimate"])
	}
	requiredCostKeys := []string{"token", "tool", "sandbox", "memory", "total"}
	for i := range requiredCostKeys {
		key := requiredCostKeys[i]
		value, ok := toFloat64(costEstimate[key])
		if !ok || value <= 0 {
			t.Fatalf("budget_snapshot.cost_estimate.%s must be >0, got %#v", key, costEstimate[key])
		}
	}
	requiredLatencyKeys := []string{"token_ms", "tool_ms", "sandbox_ms", "memory_ms", "total_ms"}
	for i := range requiredLatencyKeys {
		key := requiredLatencyKeys[i]
		value, ok := toInt64(latencyEstimate[key])
		if !ok || value <= 0 {
			t.Fatalf("budget_snapshot.latency_estimate.%s must be >0, got %#v", key, latencyEstimate[key])
		}
	}
}

func toFloat64(raw any) (float64, bool) {
	switch value := raw.(type) {
	case float64:
		return value, true
	case float32:
		return float64(value), true
	case int:
		return float64(value), true
	case int64:
		return float64(value), true
	case int32:
		return float64(value), true
	default:
		return 0, false
	}
}

func toInt64(raw any) (int64, bool) {
	switch value := raw.(type) {
	case int64:
		return value, true
	case int32:
		return int64(value), true
	case int:
		return int64(value), true
	case float64:
		if float64(int64(value)) != value {
			return 0, false
		}
		return int64(value), true
	default:
		return 0, false
	}
}
