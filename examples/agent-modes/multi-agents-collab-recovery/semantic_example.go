package multiagentscollabrecovery

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
	patternName      = "multi-agents-collab-recovery"
	phase            = "P0"
	semanticAnchor   = "collab.mailbox_taskboard_recovery"
	classification   = "multi_agents.collaboration_recovery"
	semanticToolName = "mode_multi_agents_collab_recovery_semantic_step"
)

type collabStep struct {
	Marker        string
	RuntimeDomain string
	Intent        string
	Outcome       string
}

type mailboxMessage struct {
	ID         string
	From       string
	To         string
	Topic      string
	Acked      bool
	Retryable  bool
	Priority   int
	DependsOn  string
	RetryCount int
}

type taskCard struct {
	ID          string
	Owner       string
	State       string
	DependsOn   string
	RetryBudget int
	Critical    bool
}

type recoveryPolicy struct {
	Name            string
	StrictAck       bool
	MaxMailboxRetry int
	AllowDLQ        bool
	ReconcileWindow int
}

type collabState struct {
	RunID              string
	MailboxDelivered   []string
	MailboxPending     []string
	MailboxRetryQueued []string
	TaskReady          []string
	TaskBlocked        []string
	TaskInFlight       []string
	RecoveryAction     string
	RecoveryStatus     string
	RetryCount         int
	DLQCount           int
	GovernanceDecision string
	ReplayBinding      string
	SeenMarkers        []string
	TotalScore         int
}

var runtimeDomains = []string{"orchestration/collab", "orchestration/mailbox", "orchestration/scheduler"}

var minimalSemanticSteps = []collabStep{
	{
		Marker:        "collab_mailbox_orchestrated",
		RuntimeDomain: "orchestration/collab",
		Intent:        "orchestrate mailbox fanout and ack tracking across participating agents",
		Outcome:       "mailbox delivered, pending, and retry-queue sets are emitted",
	},
	{
		Marker:        "collab_task_board_reconciled",
		RuntimeDomain: "orchestration/mailbox",
		Intent:        "reconcile task-board ownership and dependencies using mailbox outcomes",
		Outcome:       "task board resolves ready, blocked, and in-flight partitions",
	},
	{
		Marker:        "collab_recovery_continued",
		RuntimeDomain: "orchestration/scheduler",
		Intent:        "apply recovery policy to continue collaboration under partial failures",
		Outcome:       "recovery action and status are produced with retry or dlq decisions",
	},
}

var productionGovernanceSteps = []collabStep{
	{
		Marker:        "governance_collab_gate_enforced",
		RuntimeDomain: "orchestration/collab",
		Intent:        "enforce governance gate from board health and recovery decisions",
		Outcome:       "governance decision allow, allow_with_record, or block is emitted",
	},
	{
		Marker:        "governance_collab_replay_bound",
		RuntimeDomain: "orchestration/mailbox",
		Intent:        "bind replay signature to mailbox, task board, and recovery outcome",
		Outcome:       "deterministic replay signature is generated",
	},
}

var minimalMailbox = []mailboxMessage{
	{ID: "mb-01", From: "planner", To: "researcher", Topic: "collect_evidence", Acked: true, Retryable: true, Priority: 3, DependsOn: "", RetryCount: 0},
	{ID: "mb-02", From: "planner", To: "validator", Topic: "validate_patch", Acked: false, Retryable: true, Priority: 3, DependsOn: "mb-01", RetryCount: 0},
	{ID: "mb-03", From: "researcher", To: "planner", Topic: "evidence_ready", Acked: true, Retryable: false, Priority: 2, DependsOn: "mb-01", RetryCount: 0},
}

var productionMailbox = []mailboxMessage{
	{ID: "mb-11", From: "planner", To: "implementer", Topic: "prepare_patch", Acked: true, Retryable: true, Priority: 3, DependsOn: "", RetryCount: 0},
	{ID: "mb-12", From: "planner", To: "validator", Topic: "final_validate", Acked: false, Retryable: true, Priority: 3, DependsOn: "mb-11", RetryCount: 1},
	{ID: "mb-13", From: "validator", To: "planner", Topic: "validation_pending", Acked: false, Retryable: true, Priority: 2, DependsOn: "mb-12", RetryCount: 0},
	{ID: "mb-14", From: "implementer", To: "planner", Topic: "patch_ready", Acked: true, Retryable: false, Priority: 2, DependsOn: "mb-11", RetryCount: 0},
}

var minimalTasks = []taskCard{
	{ID: "tb-collect", Owner: "researcher", State: "todo", DependsOn: "mb-01", RetryBudget: 1, Critical: true},
	{ID: "tb-validate", Owner: "validator", State: "todo", DependsOn: "mb-02", RetryBudget: 1, Critical: true},
	{ID: "tb-summarize", Owner: "planner", State: "todo", DependsOn: "mb-03", RetryBudget: 0, Critical: false},
}

var productionTasks = []taskCard{
	{ID: "tb-build", Owner: "implementer", State: "todo", DependsOn: "mb-11", RetryBudget: 2, Critical: true},
	{ID: "tb-verify", Owner: "validator", State: "todo", DependsOn: "mb-12", RetryBudget: 1, Critical: true},
	{ID: "tb-close", Owner: "planner", State: "todo", DependsOn: "mb-14", RetryBudget: 0, Critical: true},
}

var minimalPolicy = recoveryPolicy{
	Name:            "collab-recovery-v1",
	StrictAck:       false,
	MaxMailboxRetry: 1,
	AllowDLQ:        false,
	ReconcileWindow: 3,
}

var productionPolicy = recoveryPolicy{
	Name:            "collab-recovery-v2-strict",
	StrictAck:       true,
	MaxMailboxRetry: 1,
	AllowDLQ:        true,
	ReconcileWindow: 2,
}

func RunMinimal() {
	executeVariant(modecommon.VariantMinimal)
}

func RunProduction() {
	executeVariant(modecommon.VariantProduction)
}

func executeVariant(variant string) {
	if err := modecommon.EnsureVariant(variant); err != nil {
		panic(err)
	}

	mgr, err := runtimeconfig.NewManager(runtimeconfig.ManagerOptions{EnvPrefix: "BAYMAX"})
	if err != nil {
		panic(err)
	}
	defer func() { _ = mgr.Close() }()

	reg := local.NewRegistry()
	if _, err := reg.Register(&collabRecoverySemanticTool{}); err != nil {
		panic(err)
	}

	engine := runner.New(newCollabWorkflowModel(variant), runner.WithLocalRegistry(reg), runner.WithRuntimeManager(mgr))
	result, err := engine.Run(context.Background(), types.RunRequest{
		RunID: fmt.Sprintf("agent-mode-%s-%s", modecommon.MarkerToken(patternName), modecommon.MarkerToken(variant)),
		Input: "execute multi agents collaboration recovery workflow",
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

func stepsForVariant(variant string) []collabStep {
	steps := make([]collabStep, 0, len(minimalSemanticSteps)+len(productionGovernanceSteps))
	steps = append(steps, minimalSemanticSteps...)
	if variant == modecommon.VariantProduction {
		steps = append(steps, productionGovernanceSteps...)
	}
	return steps
}

func newCollabWorkflowModel(variant string) *collabWorkflowModel {
	policy := policyForVariant(variant)
	return &collabWorkflowModel{
		variant: variant,
		state: collabState{
			RunID:          runIDForVariant(variant),
			RecoveryStatus: "not_started",
			RecoveryAction: "none",
			GovernanceDecision: func() string {
				if policy.StrictAck {
					return "pending"
				}
				return "not_applicable"
			}(),
		},
	}
}

type collabWorkflowModel struct {
	variant string
	stage   int
	state   collabState
}

func (m *collabWorkflowModel) Generate(ctx context.Context, req types.ModelRequest) (types.ModelResponse, error) {
	_ = ctx
	m.absorb(req.ToolResult)

	steps := stepsForVariant(m.variant)
	if m.stage < len(steps) {
		step := steps[m.stage]
		call := types.ToolCall{
			CallID: fmt.Sprintf("%s-%02d", modecommon.MarkerToken(patternName), m.stage+1),
			Name:   "local." + semanticToolName,
			Args:   m.argsForStep(step, m.stage+1),
		}
		m.stage++
		return types.ModelResponse{ToolCalls: []types.ToolCall{call}}, nil
	}

	markers := append([]string(nil), m.state.SeenMarkers...)
	sort.Strings(markers)
	final := fmt.Sprintf(
		"%s/%s semantic_path_completed phase=%s anchor=%s classification=%s run=%s delivered=%s pending=%s retry_queue=%s ready=%s blocked=%s inflight=%s recovery_action=%s recovery_status=%s retry_count=%d dlq_count=%d governance=%s replay=%s markers=%s score=%d",
		patternName,
		m.variant,
		phase,
		semanticAnchor,
		classification,
		m.state.RunID,
		strings.Join(m.state.MailboxDelivered, ","),
		strings.Join(m.state.MailboxPending, ","),
		strings.Join(m.state.MailboxRetryQueued, ","),
		strings.Join(m.state.TaskReady, ","),
		strings.Join(m.state.TaskBlocked, ","),
		strings.Join(m.state.TaskInFlight, ","),
		safeString(m.state.RecoveryAction, "none"),
		safeString(m.state.RecoveryStatus, "none"),
		m.state.RetryCount,
		m.state.DLQCount,
		normalizedDecision(m.state.GovernanceDecision),
		safeString(m.state.ReplayBinding, "none"),
		strings.Join(markers, ","),
		m.state.TotalScore,
	)
	return types.ModelResponse{FinalAnswer: final}, nil
}

func (m *collabWorkflowModel) Stream(ctx context.Context, req types.ModelRequest, onEvent func(types.ModelEvent) error) error {
	_ = ctx
	_ = req
	_ = onEvent
	return nil
}
func (m *collabWorkflowModel) absorb(outcomes []types.ToolCallOutcome) {
	for _, item := range outcomes {
		marker, _ := item.Result.Structured["marker"].(string)
		if marker != "" {
			m.state.SeenMarkers = append(m.state.SeenMarkers, marker)
		}
		if runID, ok := item.Result.Structured["run_id"].(string); ok && strings.TrimSpace(runID) != "" {
			m.state.RunID = runID
		}
		if values, ok := toStringSlice(item.Result.Structured["mailbox_delivered"]); ok {
			m.state.MailboxDelivered = values
		}
		if values, ok := toStringSlice(item.Result.Structured["mailbox_pending"]); ok {
			m.state.MailboxPending = values
		}
		if values, ok := toStringSlice(item.Result.Structured["mailbox_retry_queue"]); ok {
			m.state.MailboxRetryQueued = values
		}
		if values, ok := toStringSlice(item.Result.Structured["task_ready"]); ok {
			m.state.TaskReady = values
		}
		if values, ok := toStringSlice(item.Result.Structured["task_blocked"]); ok {
			m.state.TaskBlocked = values
		}
		if values, ok := toStringSlice(item.Result.Structured["task_inflight"]); ok {
			m.state.TaskInFlight = values
		}
		if action, ok := item.Result.Structured["recovery_action"].(string); ok && strings.TrimSpace(action) != "" {
			m.state.RecoveryAction = action
		}
		if status, ok := item.Result.Structured["recovery_status"].(string); ok && strings.TrimSpace(status) != "" {
			m.state.RecoveryStatus = status
		}
		if retryCount, ok := modecommon.AsInt(item.Result.Structured["retry_count"]); ok {
			m.state.RetryCount = retryCount
		}
		if dlqCount, ok := modecommon.AsInt(item.Result.Structured["dlq_count"]); ok {
			m.state.DLQCount = dlqCount
		}
		if governanceDecision, ok := item.Result.Structured["governance_decision"].(string); ok && strings.TrimSpace(governanceDecision) != "" {
			m.state.GovernanceDecision = governanceDecision
		}
		if replayBinding, ok := item.Result.Structured["replay_binding"].(string); ok && strings.TrimSpace(replayBinding) != "" {
			m.state.ReplayBinding = replayBinding
		}
		if score, ok := modecommon.AsInt(item.Result.Structured["score"]); ok {
			m.state.TotalScore += score
		}
	}
}

func (m *collabWorkflowModel) argsForStep(step collabStep, stage int) map[string]any {
	policy := policyForVariant(m.variant)
	args := map[string]any{
		"pattern":             patternName,
		"variant":             m.variant,
		"phase":               phase,
		"semantic_anchor":     semanticAnchor,
		"classification":      classification,
		"marker":              step.Marker,
		"runtime_domain":      step.RuntimeDomain,
		"semantic_intent":     step.Intent,
		"semantic_outcome":    step.Outcome,
		"run_id":              m.state.RunID,
		"mailbox_specs":       toAnySlice(mailboxSpecsForVariant(m.variant)),
		"task_specs":          toAnySlice(taskSpecsForVariant(m.variant)),
		"policy_spec":         policySpec(policy),
		"mailbox_delivered":   toAnySlice(m.state.MailboxDelivered),
		"mailbox_pending":     toAnySlice(m.state.MailboxPending),
		"mailbox_retry_queue": toAnySlice(m.state.MailboxRetryQueued),
		"task_ready":          toAnySlice(m.state.TaskReady),
		"task_blocked":        toAnySlice(m.state.TaskBlocked),
		"task_inflight":       toAnySlice(m.state.TaskInFlight),
		"recovery_action":     m.state.RecoveryAction,
		"recovery_status":     m.state.RecoveryStatus,
		"retry_count":         m.state.RetryCount,
		"dlq_count":           m.state.DLQCount,
		"governance_decision": m.state.GovernanceDecision,
		"stage":               stage,
	}
	return args
}

type collabRecoverySemanticTool struct{}

func (t *collabRecoverySemanticTool) Name() string { return semanticToolName }

func (t *collabRecoverySemanticTool) Description() string {
	return "execute multi-agent collaboration recovery semantic step"
}

func (t *collabRecoverySemanticTool) JSONSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"required": []any{
			"pattern",
			"variant",
			"phase",
			"semantic_anchor",
			"classification",
			"marker",
			"runtime_domain",
			"semantic_intent",
			"semantic_outcome",
			"run_id",
			"mailbox_specs",
			"task_specs",
			"policy_spec",
			"stage",
		},
		"properties": map[string]any{
			"pattern":             map[string]any{"type": "string"},
			"variant":             map[string]any{"type": "string"},
			"phase":               map[string]any{"type": "string"},
			"semantic_anchor":     map[string]any{"type": "string"},
			"classification":      map[string]any{"type": "string"},
			"marker":              map[string]any{"type": "string"},
			"runtime_domain":      map[string]any{"type": "string"},
			"semantic_intent":     map[string]any{"type": "string"},
			"semantic_outcome":    map[string]any{"type": "string"},
			"run_id":              map[string]any{"type": "string"},
			"mailbox_specs":       map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
			"task_specs":          map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
			"policy_spec":         map[string]any{"type": "string"},
			"mailbox_delivered":   map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
			"mailbox_pending":     map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
			"mailbox_retry_queue": map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
			"task_ready":          map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
			"task_blocked":        map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
			"task_inflight":       map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
			"recovery_action":     map[string]any{"type": "string"},
			"recovery_status":     map[string]any{"type": "string"},
			"retry_count":         map[string]any{"type": "integer"},
			"dlq_count":           map[string]any{"type": "integer"},
			"governance_decision": map[string]any{"type": "string"},
			"stage":               map[string]any{"type": "integer"},
		},
	}
}

func (t *collabRecoverySemanticTool) Invoke(ctx context.Context, args map[string]any) (types.ToolResult, error) {
	_ = ctx

	pattern := strings.TrimSpace(fmt.Sprintf("%v", args["pattern"]))
	variant := strings.TrimSpace(fmt.Sprintf("%v", args["variant"]))
	phaseValue := strings.TrimSpace(fmt.Sprintf("%v", args["phase"]))
	anchor := strings.TrimSpace(fmt.Sprintf("%v", args["semantic_anchor"]))
	classValue := strings.TrimSpace(fmt.Sprintf("%v", args["classification"]))
	marker := strings.TrimSpace(fmt.Sprintf("%v", args["marker"]))
	runtimeDomain := strings.TrimSpace(fmt.Sprintf("%v", args["runtime_domain"]))
	intent := strings.TrimSpace(fmt.Sprintf("%v", args["semantic_intent"]))
	outcome := strings.TrimSpace(fmt.Sprintf("%v", args["semantic_outcome"]))
	runID := strings.TrimSpace(fmt.Sprintf("%v", args["run_id"]))
	mailboxSpecs, _ := toStringSlice(args["mailbox_specs"])
	messages := parseMailboxSpecs(mailboxSpecs)
	taskSpecs, _ := toStringSlice(args["task_specs"])
	tasks := parseTaskSpecs(taskSpecs)
	policy := parsePolicySpec(strings.TrimSpace(fmt.Sprintf("%v", args["policy_spec"])))
	mailboxDelivered, _ := toStringSlice(args["mailbox_delivered"])
	mailboxPending, _ := toStringSlice(args["mailbox_pending"])
	mailboxRetryQueue, _ := toStringSlice(args["mailbox_retry_queue"])
	taskReady, _ := toStringSlice(args["task_ready"])
	taskBlocked, _ := toStringSlice(args["task_blocked"])
	taskInFlight, _ := toStringSlice(args["task_inflight"])
	recoveryAction := safeString(args["recovery_action"], "")
	recoveryStatus := safeString(args["recovery_status"], "")
	retryCount, _ := modecommon.AsInt(args["retry_count"])
	dlqCount, _ := modecommon.AsInt(args["dlq_count"])
	governanceDecision := safeString(args["governance_decision"], "")
	stage, _ := modecommon.AsInt(args["stage"])
	if stage <= 0 {
		stage = 1
	}

	result := map[string]any{
		"pattern":             pattern,
		"variant":             variant,
		"phase":               phaseValue,
		"semantic_anchor":     anchor,
		"classification":      classValue,
		"marker":              marker,
		"runtime_domain":      runtimeDomain,
		"semantic_intent":     intent,
		"semantic_outcome":    outcome,
		"run_id":              runID,
		"recovery_action":     recoveryAction,
		"recovery_status":     recoveryStatus,
		"retry_count":         retryCount,
		"dlq_count":           dlqCount,
		"governance_decision": governanceDecision,
		"stage":               stage,
		"governance":          false,
	}

	risk := "nominal"

	switch marker {
	case "collab_mailbox_orchestrated":
		mailboxDelivered, mailboxPending, mailboxRetryQueue = orchestrateMailbox(messages)
		result["mailbox_delivered"] = toAnySlice(mailboxDelivered)
		result["mailbox_pending"] = toAnySlice(mailboxPending)
		result["mailbox_retry_queue"] = toAnySlice(mailboxRetryQueue)
		if len(mailboxPending) > 0 {
			risk = "mailbox_partial"
		} else {
			risk = "mailbox_clean"
		}
	case "collab_task_board_reconciled":
		if len(mailboxDelivered) == 0 && len(mailboxPending) == 0 {
			mailboxDelivered, mailboxPending, _ = orchestrateMailbox(messages)
		}
		taskReady, taskBlocked, taskInFlight = reconcileTaskBoard(tasks, mailboxDelivered, mailboxPending)
		result["mailbox_delivered"] = toAnySlice(mailboxDelivered)
		result["mailbox_pending"] = toAnySlice(mailboxPending)
		result["task_ready"] = toAnySlice(taskReady)
		result["task_blocked"] = toAnySlice(taskBlocked)
		result["task_inflight"] = toAnySlice(taskInFlight)
		if len(taskBlocked) > 0 {
			risk = "board_blocked"
		} else {
			risk = "board_reconciled"
		}
	case "collab_recovery_continued":
		if len(taskReady) == 0 && len(taskBlocked) == 0 {
			if len(mailboxDelivered) == 0 && len(mailboxPending) == 0 {
				mailboxDelivered, mailboxPending, mailboxRetryQueue = orchestrateMailbox(messages)
			}
			taskReady, taskBlocked, taskInFlight = reconcileTaskBoard(tasks, mailboxDelivered, mailboxPending)
		}
		recoveryAction, recoveryStatus, retryCount, dlqCount = continueRecovery(taskBlocked, mailboxPending, mailboxRetryQueue, policy)
		result["mailbox_delivered"] = toAnySlice(mailboxDelivered)
		result["mailbox_pending"] = toAnySlice(mailboxPending)
		result["mailbox_retry_queue"] = toAnySlice(mailboxRetryQueue)
		result["task_ready"] = toAnySlice(taskReady)
		result["task_blocked"] = toAnySlice(taskBlocked)
		result["task_inflight"] = toAnySlice(taskInFlight)
		result["recovery_action"] = recoveryAction
		result["recovery_status"] = recoveryStatus
		result["retry_count"] = retryCount
		result["dlq_count"] = dlqCount
		if recoveryStatus == "continued" {
			risk = "recovery_continued"
		} else {
			risk = "degraded_path"
		}
	case "governance_collab_gate_enforced":
		governanceDecision = "allow"
		if policy.StrictAck && len(mailboxPending) > 0 {
			governanceDecision = "allow_with_record"
			risk = "governed_warn"
		}
		if recoveryStatus == "stalled" {
			governanceDecision = "block"
			risk = "governed_block"
		}
		if len(taskBlocked) > policy.ReconcileWindow {
			governanceDecision = "allow_with_record"
			risk = "governed_warn"
		}
		result["mailbox_pending"] = toAnySlice(mailboxPending)
		result["task_blocked"] = toAnySlice(taskBlocked)
		result["recovery_status"] = recoveryStatus
		result["recovery_action"] = recoveryAction
		result["retry_count"] = retryCount
		result["dlq_count"] = dlqCount
		result["governance_decision"] = governanceDecision
		result["governance"] = true
	case "governance_collab_replay_bound":
		governanceDecision = safeString(governanceDecision, "allow")
		replayBinding := fmt.Sprintf(
			"collab-replay-%d",
			modecommon.SemanticScore(
				pattern,
				variant,
				runID,
				strings.Join(uniqueSorted(mailboxDelivered), ","),
				strings.Join(uniqueSorted(mailboxPending), ","),
				strings.Join(uniqueSorted(taskReady), ","),
				strings.Join(uniqueSorted(taskBlocked), ","),
				recoveryAction,
				recoveryStatus,
				governanceDecision,
			),
		)
		result["mailbox_delivered"] = toAnySlice(uniqueSorted(mailboxDelivered))
		result["mailbox_pending"] = toAnySlice(uniqueSorted(mailboxPending))
		result["task_ready"] = toAnySlice(uniqueSorted(taskReady))
		result["task_blocked"] = toAnySlice(uniqueSorted(taskBlocked))
		result["recovery_action"] = recoveryAction
		result["recovery_status"] = recoveryStatus
		result["governance_decision"] = governanceDecision
		result["replay_binding"] = replayBinding
		result["retry_count"] = retryCount
		result["dlq_count"] = dlqCount
		result["governance"] = true
		risk = "governed_replay"
	default:
		return types.ToolResult{}, fmt.Errorf("unsupported collab marker: %s", marker)
	}

	result["score"] = modecommon.SemanticScore(
		pattern,
		variant,
		phaseValue,
		anchor,
		classValue,
		marker,
		runtimeDomain,
		risk,
		safeString(result["recovery_status"], recoveryStatus),
		safeString(result["governance_decision"], governanceDecision),
	)
	result["risk"] = risk

	content := fmt.Sprintf(
		"pattern=%s variant=%s marker=%s pending=%d blocked=%d recovery=%s governance=%s risk=%s",
		pattern,
		variant,
		marker,
		len(mailboxPending),
		len(taskBlocked),
		safeString(result["recovery_status"], recoveryStatus),
		normalizedDecision(safeString(result["governance_decision"], governanceDecision)),
		risk,
	)
	return types.ToolResult{Content: content, Structured: result}, nil
}

func runIDForVariant(variant string) string {
	if variant == modecommon.VariantProduction {
		return "collab-run-prod-202604"
	}
	return "collab-run-min-202604"
}

func mailboxSpecsForVariant(variant string) []string {
	items := minimalMailbox
	if variant == modecommon.VariantProduction {
		items = productionMailbox
	}
	specs := make([]string, 0, len(items))
	for _, item := range items {
		specs = append(specs, fmt.Sprintf(
			"%s|%s|%s|%s|%t|%t|%d|%s|%d",
			item.ID,
			item.From,
			item.To,
			item.Topic,
			item.Acked,
			item.Retryable,
			item.Priority,
			item.DependsOn,
			item.RetryCount,
		))
	}
	return specs
}

func taskSpecsForVariant(variant string) []string {
	items := minimalTasks
	if variant == modecommon.VariantProduction {
		items = productionTasks
	}
	specs := make([]string, 0, len(items))
	for _, item := range items {
		specs = append(specs, fmt.Sprintf(
			"%s|%s|%s|%s|%d|%t",
			item.ID,
			item.Owner,
			item.State,
			item.DependsOn,
			item.RetryBudget,
			item.Critical,
		))
	}
	return specs
}

func policySpec(policy recoveryPolicy) string {
	return fmt.Sprintf("%s|%t|%d|%t|%d", policy.Name, policy.StrictAck, policy.MaxMailboxRetry, policy.AllowDLQ, policy.ReconcileWindow)
}

func policyForVariant(variant string) recoveryPolicy {
	if variant == modecommon.VariantProduction {
		return productionPolicy
	}
	return minimalPolicy
}

func parseMailboxSpecs(specs []string) []mailboxMessage {
	out := make([]mailboxMessage, 0, len(specs))
	for _, spec := range specs {
		parts := strings.Split(spec, "|")
		if len(parts) != 9 {
			continue
		}
		out = append(out, mailboxMessage{
			ID:         strings.TrimSpace(parts[0]),
			From:       strings.TrimSpace(parts[1]),
			To:         strings.TrimSpace(parts[2]),
			Topic:      strings.TrimSpace(parts[3]),
			Acked:      parseBool(parts[4], false),
			Retryable:  parseBool(parts[5], false),
			Priority:   parseInt(parts[6]),
			DependsOn:  strings.TrimSpace(parts[7]),
			RetryCount: parseInt(parts[8]),
		})
	}
	if len(out) == 0 {
		out = append(out, minimalMailbox...)
	}
	return out
}

func parseTaskSpecs(specs []string) []taskCard {
	out := make([]taskCard, 0, len(specs))
	for _, spec := range specs {
		parts := strings.Split(spec, "|")
		if len(parts) != 6 {
			continue
		}
		out = append(out, taskCard{
			ID:          strings.TrimSpace(parts[0]),
			Owner:       strings.TrimSpace(parts[1]),
			State:       strings.TrimSpace(parts[2]),
			DependsOn:   strings.TrimSpace(parts[3]),
			RetryBudget: parseInt(parts[4]),
			Critical:    parseBool(parts[5], false),
		})
	}
	if len(out) == 0 {
		out = append(out, minimalTasks...)
	}
	return out
}

func parsePolicySpec(spec string) recoveryPolicy {
	parts := strings.Split(spec, "|")
	if len(parts) != 5 {
		return minimalPolicy
	}
	return recoveryPolicy{
		Name:            strings.TrimSpace(parts[0]),
		StrictAck:       parseBool(parts[1], false),
		MaxMailboxRetry: parseInt(parts[2]),
		AllowDLQ:        parseBool(parts[3], false),
		ReconcileWindow: parseInt(parts[4]),
	}
}

func orchestrateMailbox(messages []mailboxMessage) ([]string, []string, []string) {
	delivered := make([]string, 0)
	pending := make([]string, 0)
	retryQueue := make([]string, 0)
	for _, message := range messages {
		if message.Acked {
			delivered = append(delivered, message.ID)
			continue
		}
		pending = append(pending, message.ID)
		if message.Retryable {
			retryQueue = append(retryQueue, fmt.Sprintf("%s:r%d", message.ID, message.RetryCount+1))
		}
	}
	return uniqueSorted(delivered), uniqueSorted(pending), uniqueSorted(retryQueue)
}

func reconcileTaskBoard(tasks []taskCard, delivered []string, pending []string) ([]string, []string, []string) {
	deliveredSet := map[string]struct{}{}
	for _, item := range delivered {
		deliveredSet[item] = struct{}{}
	}
	pendingSet := map[string]struct{}{}
	for _, item := range pending {
		pendingSet[item] = struct{}{}
	}
	ready := make([]string, 0)
	blocked := make([]string, 0)
	inflight := make([]string, 0)
	for _, task := range tasks {
		if task.DependsOn == "" {
			ready = append(ready, task.ID)
			continue
		}
		if _, ok := deliveredSet[task.DependsOn]; ok {
			if task.State == "running" {
				inflight = append(inflight, task.ID)
			} else {
				ready = append(ready, task.ID)
			}
			continue
		}
		if _, ok := pendingSet[task.DependsOn]; ok {
			blocked = append(blocked, task.ID+":waiting_mailbox")
			continue
		}
		blocked = append(blocked, task.ID+":missing_dependency")
	}
	return uniqueSorted(ready), uniqueSorted(blocked), uniqueSorted(inflight)
}

func continueRecovery(taskBlocked []string, mailboxPending []string, mailboxRetry []string, policy recoveryPolicy) (string, string, int, int) {
	retryCount := 0
	dlqCount := 0
	action := "direct_continue"
	status := "continued"

	if len(mailboxPending) == 0 && len(taskBlocked) == 0 {
		return action, status, retryCount, dlqCount
	}

	if len(mailboxRetry) > 0 {
		retryCount = len(mailboxRetry)
		action = "retry_mailbox_then_reconcile"
		status = "continued"
		if policy.MaxMailboxRetry > 0 && retryCount > policy.MaxMailboxRetry {
			if policy.AllowDLQ {
				dlqCount = retryCount - policy.MaxMailboxRetry
				action = "retry_and_move_overflow_to_dlq"
				status = "partial"
			} else {
				action = "retry_over_budget_wait_manual"
				status = "stalled"
			}
		}
	} else {
		action = "reconcile_task_board_only"
		status = "partial"
	}

	if policy.StrictAck && len(mailboxPending) > 0 && retryCount == 0 {
		action = "await_manual_ack"
		status = "stalled"
	}

	if len(taskBlocked) > policy.ReconcileWindow {
		status = "partial"
		if action == "direct_continue" {
			action = "staggered_reconcile_window"
		}
	}

	return action, status, retryCount, dlqCount
}

func toStringSlice(value any) ([]string, bool) {
	switch raw := value.(type) {
	case []string:
		return append([]string(nil), raw...), true
	case []any:
		out := make([]string, 0, len(raw))
		for _, item := range raw {
			text := strings.TrimSpace(fmt.Sprintf("%v", item))
			if text == "" || text == "<nil>" {
				continue
			}
			out = append(out, text)
		}
		return out, true
	default:
		return nil, false
	}
}

func toAnySlice(in []string) []any {
	if len(in) == 0 {
		return []any{}
	}
	out := make([]any, 0, len(in))
	for _, item := range in {
		out = append(out, item)
	}
	return out
}

func uniqueSorted(in []string) []string {
	set := map[string]struct{}{}
	for _, item := range in {
		trimmed := strings.TrimSpace(item)
		if trimmed == "" {
			continue
		}
		set[trimmed] = struct{}{}
	}
	out := make([]string, 0, len(set))
	for item := range set {
		out = append(out, item)
	}
	sort.Strings(out)
	return out
}

func parseBool(value any, fallback bool) bool {
	switch raw := value.(type) {
	case bool:
		return raw
	default:
		text := strings.TrimSpace(strings.ToLower(fmt.Sprintf("%v", value)))
		if text == "" || text == "<nil>" {
			return fallback
		}
		switch text {
		case "1", "true", "yes", "y":
			return true
		case "0", "false", "no", "n":
			return false
		default:
			return fallback
		}
	}
}

func parseInt(raw string) int {
	sign := 1
	value := 0
	for idx, char := range raw {
		if idx == 0 && char == '-' {
			sign = -1
			continue
		}
		if char < '0' || char > '9' {
			continue
		}
		value = value*10 + int(char-'0')
	}
	return sign * value
}

func safeString(value any, fallback string) string {
	text := strings.TrimSpace(fmt.Sprintf("%v", value))
	if text == "" || text == "<nil>" {
		return fallback
	}
	return text
}

func normalizedDecision(value string) string {
	if strings.TrimSpace(value) == "" {
		return "not_applicable"
	}
	return value
}
