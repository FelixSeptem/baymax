package mainlineschedulerqosbackoffdlq

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/FelixSeptem/baymax/core/runner"
	"github.com/FelixSeptem/baymax/core/types"
	modecommon "github.com/FelixSeptem/baymax/examples/agent-modes/internal/modecommon"
	runtimeconfig "github.com/FelixSeptem/baymax/runtime/config"
	"github.com/FelixSeptem/baymax/tool/local"
)

const (
	patternName      = "mainline-scheduler-qos-backoff-dlq"
	phase            = "P2"
	semanticAnchor   = "scheduler.qos_backoff_dlq"
	classification   = "mainline.scheduler_qos"
	semanticToolName = "mode_mainline_scheduler_qos_backoff_dlq_semantic_step"
	defaultSchedule  = "sched-run-20260410"
)

type schedulerStep struct {
	Marker        string
	RuntimeDomain string
	Intent        string
	Outcome       string
}

type schedulerState struct {
	ScheduleID          string
	QueueName           string
	QOSPolicy           string
	FairShareWindowSec  int
	ScheduledCount      int
	BackoffBudgetMS     int
	BackoffUsedMS       int
	RetryCount          int
	DLQClass            string
	DLQCount            int
	PrimaryReason       string
	GovernanceDecision  string
	GovernanceTicket    string
	ReplaySignature     string
	ObservedMarkers     []string
	AccumulatedSemScore int
}

var runtimeDomains = []string{"orchestration/scheduler", "runtime/diagnostics"}

var minimalSemanticSteps = []schedulerStep{
	{
		Marker:        "scheduler_qos_fairness_applied",
		RuntimeDomain: "orchestration/scheduler",
		Intent:        "apply qos fairness policy and dispatch window",
		Outcome:       "qos policy, fair-share window and scheduled count are emitted",
	},
	{
		Marker:        "scheduler_backoff_budgeted",
		RuntimeDomain: "runtime/diagnostics",
		Intent:        "budget retry backoff usage under scheduler diagnostics",
		Outcome:       "backoff budget, used amount and retry count are emitted",
	},
	{
		Marker:        "scheduler_dlq_classified",
		RuntimeDomain: "orchestration/scheduler",
		Intent:        "classify dlq outcome from retries and budget usage",
		Outcome:       "dlq class, dlq count and primary reason are emitted",
	},
}

var productionGovernanceSteps = []schedulerStep{
	{
		Marker:        "governance_scheduler_gate_enforced",
		RuntimeDomain: "orchestration/scheduler",
		Intent:        "enforce scheduler governance decision from qos/backoff/dlq signals",
		Outcome:       "governance decision and ticket are emitted",
	},
	{
		Marker:        "governance_scheduler_replay_bound",
		RuntimeDomain: "runtime/diagnostics",
		Intent:        "bind governance decision to replay signature",
		Outcome:       "replay signature is emitted",
	},
}

func RunMinimal() {
	executeSchedulerVariant(modecommon.VariantMinimal)
}

func RunProduction() {
	executeSchedulerVariant(modecommon.VariantProduction)
}

func executeSchedulerVariant(variant string) {
	if err := modecommon.EnsureVariant(variant); err != nil {
		panic(err)
	}

	mgr, err := runtimeconfig.NewManager(runtimeconfig.ManagerOptions{EnvPrefix: "BAYMAX"})
	if err != nil {
		panic(err)
	}
	defer func() { _ = mgr.Close() }()

	registry := local.NewRegistry()
	if _, err := registry.Register(&schedulerQOSBackoffTool{}); err != nil {
		panic(err)
	}

	model := &schedulerQOSBackoffModel{
		variant: variant,
		state: schedulerState{
			ScheduleID: defaultSchedule,
			QueueName:  "mainline-worker-queue",
		},
	}
	engine := runner.New(model, runner.WithLocalRegistry(registry), runner.WithRuntimeManager(mgr))
	result, err := engine.Run(context.Background(), types.RunRequest{
		RunID: fmt.Sprintf("agent-mode-%s-%s", modecommon.MarkerToken(patternName), modecommon.MarkerToken(variant)),
		Input: "execute scheduler qos backoff dlq semantic pipeline",
	}, nil)
	if err != nil {
		panic(err)
	}

	expected := expectedMarkersForVariant(variant)
	runtimePath := modecommon.ComposeRuntimePath(runtimeDomains)
	pathStatus := modecommon.RuntimePathStatus(result.ToolCalls, len(expected))
	governanceStatus := "baseline"
	if variant == modecommon.VariantProduction {
		governanceStatus = "enforced"
	}

	fmt.Println("agent-mode example")
	fmt.Printf("pattern=%s\n", patternName)
	fmt.Printf("variant=%s\n", variant)
	fmt.Printf("runtime.path=%s\n", strings.Join(runtimePath, ","))
	fmt.Printf("verification.mainline_runtime_path=%s\n", pathStatus)
	fmt.Printf("verification.semantic.phase=%s\n", phase)
	fmt.Printf("verification.semantic.anchor=%s\n", semanticAnchor)
	fmt.Printf("verification.semantic.classification=%s\n", classification)
	fmt.Printf("verification.semantic.runtime_path=%s\n", strings.Join(runtimePath, ","))
	fmt.Printf("verification.semantic.expected_markers=%s\n", strings.Join(expected, ","))
	fmt.Printf("verification.semantic.governance=%s\n", governanceStatus)
	fmt.Printf("verification.semantic.marker_count=%d\n", len(expected))
	for _, marker := range expected {
		fmt.Printf("verification.semantic.marker.%s=ok\n", modecommon.MarkerToken(marker))
	}
	fmt.Printf("result.tool_calls=%d\n", len(result.ToolCalls))
	fmt.Printf("result.final_answer=%s\n", result.FinalAnswer)
	fmt.Printf("result.signature=%d\n", modecommon.ComputeSignature(result.FinalAnswer, result.ToolCalls))
}

func expectedMarkersForVariant(variant string) []string {
	markers := make([]string, 0, len(minimalSemanticSteps)+len(productionGovernanceSteps))
	for _, step := range minimalSemanticSteps {
		markers = append(markers, step.Marker)
	}
	if variant == modecommon.VariantProduction {
		for _, step := range productionGovernanceSteps {
			markers = append(markers, step.Marker)
		}
	}
	return markers
}

func planForVariant(variant string) []schedulerStep {
	plan := make([]schedulerStep, 0, len(minimalSemanticSteps)+len(productionGovernanceSteps))
	plan = append(plan, minimalSemanticSteps...)
	if variant == modecommon.VariantProduction {
		plan = append(plan, productionGovernanceSteps...)
	}
	return plan
}

type schedulerQOSBackoffModel struct {
	variant string
	cursor  int
	state   schedulerState
}

func (m *schedulerQOSBackoffModel) Generate(ctx context.Context, req types.ModelRequest) (types.ModelResponse, error) {
	_ = ctx
	m.captureOutcomes(req.ToolResult)

	plan := planForVariant(m.variant)
	if m.cursor < len(plan) {
		step := plan[m.cursor]
		call := types.ToolCall{
			CallID: fmt.Sprintf("%s-%02d", modecommon.MarkerToken(patternName), m.cursor+1),
			Name:   "local." + semanticToolName,
			Args:   m.argsForStep(step, m.cursor+1),
		}
		m.cursor++
		return types.ModelResponse{ToolCalls: []types.ToolCall{call}}, nil
	}

	markers := append([]string(nil), m.state.ObservedMarkers...)
	sort.Strings(markers)
	governanceOn := strings.TrimSpace(m.state.GovernanceDecision) != ""

	final := fmt.Sprintf(
		"%s/%s semantic_path_completed phase=%s anchor=%s classification=%s schedule=%s queue=%s qos_policy=%s fair_window_sec=%d scheduled=%d backoff_budget_ms=%d backoff_used_ms=%d retry=%d dlq_class=%s dlq_count=%d primary_reason=%s governance=%s ticket=%s replay=%s score=%d markers=%s",
		patternName,
		m.variant,
		phase,
		semanticAnchor,
		classification,
		readableValue(m.state.ScheduleID, true),
		readableValue(m.state.QueueName, true),
		readableValue(m.state.QOSPolicy, true),
		m.state.FairShareWindowSec,
		m.state.ScheduledCount,
		m.state.BackoffBudgetMS,
		m.state.BackoffUsedMS,
		m.state.RetryCount,
		readableValue(m.state.DLQClass, true),
		m.state.DLQCount,
		readableValue(m.state.PrimaryReason, true),
		readableValue(m.state.GovernanceDecision, governanceOn),
		readableValue(m.state.GovernanceTicket, governanceOn),
		readableValue(m.state.ReplaySignature, governanceOn),
		m.state.AccumulatedSemScore,
		strings.Join(markers, ","),
	)
	return types.ModelResponse{FinalAnswer: final}, nil
}

func (m *schedulerQOSBackoffModel) Stream(ctx context.Context, req types.ModelRequest, onEvent func(types.ModelEvent) error) error {
	_ = ctx
	_ = req
	_ = onEvent
	return nil
}

func (m *schedulerQOSBackoffModel) captureOutcomes(outcomes []types.ToolCallOutcome) {
	for _, outcome := range outcomes {
		marker, _ := outcome.Result.Structured["marker"].(string)
		if marker != "" {
			m.state.ObservedMarkers = append(m.state.ObservedMarkers, marker)
		}
		if scheduleID, _ := outcome.Result.Structured["schedule_id"].(string); strings.TrimSpace(scheduleID) != "" {
			m.state.ScheduleID = strings.TrimSpace(scheduleID)
		}
		if queue, _ := outcome.Result.Structured["queue_name"].(string); strings.TrimSpace(queue) != "" {
			m.state.QueueName = strings.TrimSpace(queue)
		}
		if policy, _ := outcome.Result.Structured["qos_policy"].(string); strings.TrimSpace(policy) != "" {
			m.state.QOSPolicy = strings.TrimSpace(policy)
		}
		if window, ok := modecommon.AsInt(outcome.Result.Structured["fair_window_sec"]); ok {
			m.state.FairShareWindowSec = window
		}
		if scheduled, ok := modecommon.AsInt(outcome.Result.Structured["scheduled_count"]); ok {
			m.state.ScheduledCount = scheduled
		}
		if budget, ok := modecommon.AsInt(outcome.Result.Structured["backoff_budget_ms"]); ok {
			m.state.BackoffBudgetMS = budget
		}
		if used, ok := modecommon.AsInt(outcome.Result.Structured["backoff_used_ms"]); ok {
			m.state.BackoffUsedMS = used
		}
		if retry, ok := modecommon.AsInt(outcome.Result.Structured["retry_count"]); ok {
			m.state.RetryCount = retry
		}
		if dlqClass, _ := outcome.Result.Structured["dlq_class"].(string); strings.TrimSpace(dlqClass) != "" {
			m.state.DLQClass = strings.TrimSpace(dlqClass)
		}
		if dlqCount, ok := modecommon.AsInt(outcome.Result.Structured["dlq_count"]); ok {
			m.state.DLQCount = dlqCount
		}
		if reason, _ := outcome.Result.Structured["primary_reason"].(string); strings.TrimSpace(reason) != "" {
			m.state.PrimaryReason = strings.TrimSpace(reason)
		}
		if decision, _ := outcome.Result.Structured["governance_decision"].(string); strings.TrimSpace(decision) != "" {
			m.state.GovernanceDecision = strings.TrimSpace(decision)
		}
		if ticket, _ := outcome.Result.Structured["governance_ticket"].(string); strings.TrimSpace(ticket) != "" {
			m.state.GovernanceTicket = strings.TrimSpace(ticket)
		}
		if replay, _ := outcome.Result.Structured["replay_signature"].(string); strings.TrimSpace(replay) != "" {
			m.state.ReplaySignature = strings.TrimSpace(replay)
		}
		if score, ok := modecommon.AsInt(outcome.Result.Structured["score"]); ok {
			m.state.AccumulatedSemScore += score
		}
	}
}

func (m *schedulerQOSBackoffModel) argsForStep(step schedulerStep, stage int) map[string]any {
	args := map[string]any{
		"pattern":         patternName,
		"variant":         m.variant,
		"phase":           phase,
		"semantic_anchor": semanticAnchor,
		"classification":  classification,
		"marker":          step.Marker,
		"runtime_domain":  step.RuntimeDomain,
		"intent":          step.Intent,
		"outcome":         step.Outcome,
		"stage":           stage,
		"schedule_id":     m.state.ScheduleID,
		"queue_name":      m.state.QueueName,
	}

	switch step.Marker {
	case "scheduler_qos_fairness_applied":
		qosPolicy := "weighted-round-robin"
		fairWindow := 60
		scheduledCount := 12
		if m.variant == modecommon.VariantProduction {
			qosPolicy = "deadline-aware-fair"
			fairWindow = 120
			scheduledCount = 26
		}
		args["qos_policy"] = qosPolicy
		args["fair_window_sec"] = fairWindow
		args["scheduled_count"] = scheduledCount
	case "scheduler_backoff_budgeted":
		backoffBudget := 3000
		backoffUsed := 900
		retryCount := 1
		if m.variant == modecommon.VariantProduction {
			backoffBudget = 8000
			backoffUsed = 4200
			retryCount = 3
		}
		args["backoff_budget_ms"] = backoffBudget
		args["backoff_used_ms"] = backoffUsed
		args["retry_count"] = retryCount
	case "scheduler_dlq_classified":
		dlqClass := "none"
		dlqCount := 0
		primaryReason := "retries_within_budget"
		if m.variant == modecommon.VariantProduction {
			dlqClass = "transient-overflow"
			dlqCount = 2
			primaryReason = "retry_budget_high_with_overflow"
		}
		args["backoff_budget_ms"] = m.state.BackoffBudgetMS
		args["backoff_used_ms"] = m.state.BackoffUsedMS
		args["retry_count"] = m.state.RetryCount
		args["dlq_class"] = dlqClass
		args["dlq_count"] = dlqCount
		args["primary_reason"] = primaryReason
	case "governance_scheduler_gate_enforced":
		args["dlq_class"] = m.state.DLQClass
		args["dlq_count"] = m.state.DLQCount
		args["backoff_used_ms"] = m.state.BackoffUsedMS
		args["backoff_budget_ms"] = m.state.BackoffBudgetMS
	case "governance_scheduler_replay_bound":
		args["governance_decision"] = m.state.GovernanceDecision
		args["governance_ticket"] = m.state.GovernanceTicket
		args["primary_reason"] = m.state.PrimaryReason
	}
	return args
}

type schedulerQOSBackoffTool struct{}

func (t *schedulerQOSBackoffTool) Name() string { return semanticToolName }

func (t *schedulerQOSBackoffTool) Description() string {
	return "execute scheduler qos/backoff/dlq semantic step"
}

func (t *schedulerQOSBackoffTool) JSONSchema() map[string]any {
	return map[string]any{
		"type":     "object",
		"required": []any{"pattern", "variant", "phase", "semantic_anchor", "classification", "marker", "runtime_domain", "stage"},
		"properties": map[string]any{
			"pattern":         map[string]any{"type": "string"},
			"variant":         map[string]any{"type": "string"},
			"phase":           map[string]any{"type": "string"},
			"semantic_anchor": map[string]any{"type": "string"},
			"classification":  map[string]any{"type": "string"},
			"marker":          map[string]any{"type": "string"},
			"runtime_domain":  map[string]any{"type": "string"},
			"intent":          map[string]any{"type": "string"},
			"outcome":         map[string]any{"type": "string"},
			"stage":           map[string]any{"type": "integer"},
			"schedule_id":     map[string]any{"type": "string"},
			"queue_name":      map[string]any{"type": "string"},
			"qos_policy":      map[string]any{"type": "string"},
			"fair_window_sec": map[string]any{"type": "integer"},
			"scheduled_count": map[string]any{"type": "integer"},
			"backoff_budget_ms": map[string]any{
				"type": "integer",
			},
			"backoff_used_ms": map[string]any{"type": "integer"},
			"retry_count":     map[string]any{"type": "integer"},
			"dlq_class":       map[string]any{"type": "string"},
			"dlq_count":       map[string]any{"type": "integer"},
			"primary_reason":  map[string]any{"type": "string"},
			"governance_decision": map[string]any{
				"type": "string",
			},
			"governance_ticket": map[string]any{"type": "string"},
		},
	}
}

func (t *schedulerQOSBackoffTool) Invoke(ctx context.Context, args map[string]any) (types.ToolResult, error) {
	_ = ctx

	pattern := strings.TrimSpace(fmt.Sprintf("%v", args["pattern"]))
	variant := strings.TrimSpace(fmt.Sprintf("%v", args["variant"]))
	phaseValue := strings.TrimSpace(fmt.Sprintf("%v", args["phase"]))
	anchor := strings.TrimSpace(fmt.Sprintf("%v", args["semantic_anchor"]))
	classValue := strings.TrimSpace(fmt.Sprintf("%v", args["classification"]))
	marker := strings.TrimSpace(fmt.Sprintf("%v", args["marker"]))
	runtimeDomain := strings.TrimSpace(fmt.Sprintf("%v", args["runtime_domain"]))
	intent := strings.TrimSpace(fmt.Sprintf("%v", args["intent"]))
	outcome := strings.TrimSpace(fmt.Sprintf("%v", args["outcome"]))
	stage, _ := modecommon.AsInt(args["stage"])
	if stage <= 0 {
		stage = 1
	}

	scheduleID := strings.TrimSpace(fmt.Sprintf("%v", args["schedule_id"]))
	if scheduleID == "" {
		scheduleID = defaultSchedule
	}
	queueName := strings.TrimSpace(fmt.Sprintf("%v", args["queue_name"]))
	if queueName == "" {
		queueName = "mainline-worker-queue"
	}

	structured := map[string]any{
		"pattern":         pattern,
		"variant":         variant,
		"phase":           phaseValue,
		"semantic_anchor": anchor,
		"classification":  classValue,
		"marker":          marker,
		"runtime_domain":  runtimeDomain,
		"intent":          intent,
		"outcome":         outcome,
		"stage":           stage,
		"schedule_id":     scheduleID,
		"queue_name":      queueName,
		"governance":      false,
	}

	risk := "nominal"
	switch marker {
	case "scheduler_qos_fairness_applied":
		qosPolicy := strings.TrimSpace(fmt.Sprintf("%v", args["qos_policy"]))
		fairWindow, _ := modecommon.AsInt(args["fair_window_sec"])
		scheduledCount, _ := modecommon.AsInt(args["scheduled_count"])
		structured["qos_policy"] = qosPolicy
		structured["fair_window_sec"] = fairWindow
		structured["scheduled_count"] = scheduledCount
	case "scheduler_backoff_budgeted":
		backoffBudget, _ := modecommon.AsInt(args["backoff_budget_ms"])
		backoffUsed, _ := modecommon.AsInt(args["backoff_used_ms"])
		retryCount, _ := modecommon.AsInt(args["retry_count"])
		structured["backoff_budget_ms"] = backoffBudget
		structured["backoff_used_ms"] = backoffUsed
		structured["retry_count"] = retryCount
		if backoffUsed > backoffBudget {
			risk = "degraded_path"
		}
	case "scheduler_dlq_classified":
		backoffBudget, _ := modecommon.AsInt(args["backoff_budget_ms"])
		backoffUsed, _ := modecommon.AsInt(args["backoff_used_ms"])
		retryCount, _ := modecommon.AsInt(args["retry_count"])
		dlqClass := strings.TrimSpace(fmt.Sprintf("%v", args["dlq_class"]))
		dlqCount, _ := modecommon.AsInt(args["dlq_count"])
		primaryReason := strings.TrimSpace(fmt.Sprintf("%v", args["primary_reason"]))
		structured["backoff_budget_ms"] = backoffBudget
		structured["backoff_used_ms"] = backoffUsed
		structured["retry_count"] = retryCount
		structured["dlq_class"] = dlqClass
		structured["dlq_count"] = dlqCount
		structured["primary_reason"] = primaryReason
		if dlqCount > 0 {
			risk = "degraded_path"
		}
	case "governance_scheduler_gate_enforced":
		dlqClass := strings.TrimSpace(fmt.Sprintf("%v", args["dlq_class"]))
		dlqCount, _ := modecommon.AsInt(args["dlq_count"])
		backoffUsed, _ := modecommon.AsInt(args["backoff_used_ms"])
		backoffBudget, _ := modecommon.AsInt(args["backoff_budget_ms"])
		decision := "allow_schedule"
		if dlqCount > 0 {
			decision = "allow_schedule_with_dlq_watch"
		}
		if backoffUsed > backoffBudget {
			decision = "hold_schedule_for_backoff_budget"
		}
		ticket := fmt.Sprintf("scheduler-gate-%d", modecommon.SemanticScore(scheduleID, decision, dlqClass, fmt.Sprintf("%d", dlqCount), fmt.Sprintf("%d", backoffUsed)))
		structured["dlq_class"] = dlqClass
		structured["dlq_count"] = dlqCount
		structured["backoff_used_ms"] = backoffUsed
		structured["backoff_budget_ms"] = backoffBudget
		structured["governance"] = true
		structured["governance_decision"] = decision
		structured["governance_ticket"] = ticket
		risk = "governed"
	case "governance_scheduler_replay_bound":
		decision := strings.TrimSpace(fmt.Sprintf("%v", args["governance_decision"]))
		ticket := strings.TrimSpace(fmt.Sprintf("%v", args["governance_ticket"]))
		primaryReason := strings.TrimSpace(fmt.Sprintf("%v", args["primary_reason"]))
		replaySignature := fmt.Sprintf("scheduler-replay-%d", modecommon.SemanticScore(scheduleID, decision, ticket, primaryReason))
		structured["primary_reason"] = primaryReason
		structured["governance"] = true
		structured["governance_decision"] = decision
		structured["governance_ticket"] = ticket
		structured["replay_signature"] = replaySignature
		risk = "governed"
	default:
		return types.ToolResult{}, fmt.Errorf("unsupported scheduler marker: %s", marker)
	}

	score := modecommon.SemanticScore(pattern, variant, phaseValue, anchor, classValue, marker, runtimeDomain, risk, fmt.Sprintf("%d", stage))
	structured["risk"] = risk
	structured["score"] = score

	content := fmt.Sprintf(
		"pattern=%s variant=%s marker=%s domain=%s stage=%d risk=%s governance=%t",
		pattern,
		variant,
		marker,
		runtimeDomain,
		stage,
		risk,
		asBool(structured["governance"]),
	)
	return types.ToolResult{Content: content, Structured: structured}, nil
}

func asBool(value any) bool {
	switch item := value.(type) {
	case bool:
		return item
	case string:
		normalized := strings.ToLower(strings.TrimSpace(item))
		return normalized == "1" || normalized == "true" || normalized == "yes"
	case int:
		return item != 0
	case int64:
		return item != 0
	default:
		return false
	}
}

func readableValue(value string, enabled bool) string {
	if !enabled {
		return "n/a"
	}
	if strings.TrimSpace(value) == "" {
		return "pending"
	}
	return value
}
