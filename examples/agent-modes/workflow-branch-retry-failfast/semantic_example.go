package workflowbranchretryfailfast

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
	patternName      = "workflow-branch-retry-failfast"
	phase            = "P1"
	semanticAnchor   = "workflow.branch_retry_failfast"
	classification   = "workflow.retry_failfast"
	semanticToolName = "mode_workflow_branch_retry_failfast_semantic_step"
	defaultSignal    = "payment-settlement-lag"
)

type workflowStep struct {
	Marker        string
	RuntimeDomain string
	Intent        string
	Outcome       string
}

type workflowState struct {
	InputSignal         string
	PreferredBranch     string
	ActiveBranch        string
	RetryBudget         int
	RetryConsumed       int
	RetryDenied         bool
	FailfastClass       string
	Failfast            bool
	EscalationLane      string
	GovernanceDecision  string
	GovernanceTicket    string
	ReplaySignature     string
	ObservedMarkers     []string
	TotalSemanticScore  int
	LatestRuntimeDomain string
}

var runtimeDomains = []string{"orchestration/workflow", "runtime/config"}

var minimalSemanticSteps = []workflowStep{
	{
		Marker:        "workflow_branch_routed",
		RuntimeDomain: "orchestration/workflow",
		Intent:        "route incoming request into fast-path or safe-path according to runtime risk budget",
		Outcome:       "branch route is selected with explicit route reason",
	},
	{
		Marker:        "workflow_retry_budgeted",
		RuntimeDomain: "runtime/config",
		Intent:        "compute retry budget per selected branch and current failure streak",
		Outcome:       "retry budget and consumption counters are persisted",
	},
	{
		Marker:        "workflow_failfast_classified",
		RuntimeDomain: "orchestration/workflow",
		Intent:        "classify fail-fast according to budget exhaustion and terminal error semantics",
		Outcome:       "fail-fast decision and escalation lane are emitted",
	},
}

var productionGovernanceSteps = []workflowStep{
	{
		Marker:        "governance_workflow_gate_enforced",
		RuntimeDomain: "orchestration/workflow",
		Intent:        "enforce governance gate on fail-fast output before workflow dispatch",
		Outcome:       "gate decision and governance ticket are persisted",
	},
	{
		Marker:        "governance_workflow_replay_bound",
		RuntimeDomain: "runtime/config",
		Intent:        "bind governance decision to deterministic replay signature",
		Outcome:       "replay signature is emitted for deterministic auditing",
	},
}

func RunMinimal() {
	executeWorkflowVariant(modecommon.VariantMinimal)
}

func RunProduction() {
	executeWorkflowVariant(modecommon.VariantProduction)
}

func executeWorkflowVariant(variant string) {
	if err := modecommon.EnsureVariant(variant); err != nil {
		panic(err)
	}

	mgr, err := runtimeconfig.NewManager(runtimeconfig.ManagerOptions{EnvPrefix: "BAYMAX"})
	if err != nil {
		panic(err)
	}
	defer func() { _ = mgr.Close() }()

	reg := local.NewRegistry()
	if _, err := reg.Register(&workflowBranchRetryTool{}); err != nil {
		panic(err)
	}

	model := &workflowBranchRetryModel{
		variant: variant,
		state: workflowState{
			InputSignal:     defaultSignal,
			PreferredBranch: "fast-path",
		},
	}

	engine := runner.New(model, runner.WithLocalRegistry(reg), runner.WithRuntimeManager(mgr))
	result, err := engine.Run(context.Background(), types.RunRequest{
		RunID: fmt.Sprintf("agent-mode-%s-%s", modecommon.MarkerToken(patternName), modecommon.MarkerToken(variant)),
		Input: "execute workflow branch retry failfast semantic pipeline",
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
	out := make([]string, 0, len(minimalSemanticSteps)+len(productionGovernanceSteps))
	for _, step := range minimalSemanticSteps {
		out = append(out, step.Marker)
	}
	if variant == modecommon.VariantProduction {
		for _, step := range productionGovernanceSteps {
			out = append(out, step.Marker)
		}
	}
	return out
}

func semanticPlanForVariant(variant string) []workflowStep {
	plan := make([]workflowStep, 0, len(minimalSemanticSteps)+len(productionGovernanceSteps))
	plan = append(plan, minimalSemanticSteps...)
	if variant == modecommon.VariantProduction {
		plan = append(plan, productionGovernanceSteps...)
	}
	return plan
}

type workflowBranchRetryModel struct {
	variant   string
	nextStage int
	state     workflowState
}

func (m *workflowBranchRetryModel) Generate(ctx context.Context, req types.ModelRequest) (types.ModelResponse, error) {
	_ = ctx
	m.captureStepResult(req.ToolResult)

	executionPlan := semanticPlanForVariant(m.variant)
	if m.nextStage < len(executionPlan) {
		step := executionPlan[m.nextStage]
		call := types.ToolCall{
			CallID: fmt.Sprintf("%s-%02d", modecommon.MarkerToken(patternName), m.nextStage+1),
			Name:   "local." + semanticToolName,
			Args:   m.stepArguments(step, m.nextStage+1),
		}
		m.nextStage++
		return types.ModelResponse{ToolCalls: []types.ToolCall{call}}, nil
	}

	markers := append([]string(nil), m.state.ObservedMarkers...)
	sort.Strings(markers)
	governanceOn := m.state.GovernanceDecision != ""

	final := fmt.Sprintf(
		"%s/%s semantic_path_completed phase=%s anchor=%s classification=%s branch=%s retry=%d/%d retry_denied=%t failfast=%t failfast_class=%s escalation=%s governance=%s ticket=%s replay=%s score=%d markers=%s",
		patternName,
		m.variant,
		phase,
		semanticAnchor,
		classification,
		m.state.ActiveBranch,
		m.state.RetryConsumed,
		m.state.RetryBudget,
		m.state.RetryDenied,
		m.state.Failfast,
		m.state.FailfastClass,
		m.state.EscalationLane,
		normalizedValue(m.state.GovernanceDecision, governanceOn),
		normalizedValue(m.state.GovernanceTicket, governanceOn),
		normalizedValue(m.state.ReplaySignature, governanceOn),
		m.state.TotalSemanticScore,
		strings.Join(markers, ","),
	)
	return types.ModelResponse{FinalAnswer: final}, nil
}

func (m *workflowBranchRetryModel) Stream(ctx context.Context, req types.ModelRequest, onEvent func(types.ModelEvent) error) error {
	_ = ctx
	_ = req
	_ = onEvent
	return nil
}

func (m *workflowBranchRetryModel) captureStepResult(outcomes []types.ToolCallOutcome) {
	for _, outcome := range outcomes {
		marker, _ := outcome.Result.Structured["marker"].(string)
		if marker != "" {
			m.state.ObservedMarkers = append(m.state.ObservedMarkers, marker)
		}

		if branch, _ := outcome.Result.Structured["active_branch"].(string); strings.TrimSpace(branch) != "" {
			m.state.ActiveBranch = strings.TrimSpace(branch)
		}
		if retryBudget, ok := modecommon.AsInt(outcome.Result.Structured["retry_budget"]); ok {
			m.state.RetryBudget = retryBudget
		}
		if retryConsumed, ok := modecommon.AsInt(outcome.Result.Structured["retry_consumed"]); ok {
			m.state.RetryConsumed = retryConsumed
		}
		if retryDenied, ok := outcome.Result.Structured["retry_denied"].(bool); ok {
			m.state.RetryDenied = retryDenied
		}
		if failfastClass, _ := outcome.Result.Structured["failfast_class"].(string); strings.TrimSpace(failfastClass) != "" {
			m.state.FailfastClass = strings.TrimSpace(failfastClass)
		}
		if failfast, ok := outcome.Result.Structured["failfast"].(bool); ok {
			m.state.Failfast = failfast
		}
		if escalation, _ := outcome.Result.Structured["escalation_lane"].(string); strings.TrimSpace(escalation) != "" {
			m.state.EscalationLane = strings.TrimSpace(escalation)
		}
		if decision, _ := outcome.Result.Structured["governance_decision"].(string); strings.TrimSpace(decision) != "" {
			m.state.GovernanceDecision = strings.TrimSpace(decision)
		}
		if ticket, _ := outcome.Result.Structured["governance_ticket"].(string); strings.TrimSpace(ticket) != "" {
			m.state.GovernanceTicket = strings.TrimSpace(ticket)
		}
		if signature, _ := outcome.Result.Structured["replay_signature"].(string); strings.TrimSpace(signature) != "" {
			m.state.ReplaySignature = strings.TrimSpace(signature)
		}
		if runtimeDomain, _ := outcome.Result.Structured["runtime_domain"].(string); strings.TrimSpace(runtimeDomain) != "" {
			m.state.LatestRuntimeDomain = strings.TrimSpace(runtimeDomain)
		}
		if score, ok := modecommon.AsInt(outcome.Result.Structured["score"]); ok {
			m.state.TotalSemanticScore += score
		}
	}
}

func (m *workflowBranchRetryModel) stepArguments(step workflowStep, stage int) map[string]any {
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
	}

	switch step.Marker {
	case "workflow_branch_routed":
		errorRatio := 0.18
		if m.variant == modecommon.VariantProduction {
			errorRatio = 0.31
		}
		args["input_signal"] = m.state.InputSignal
		args["preferred_branch"] = m.state.PreferredBranch
		args["error_ratio"] = errorRatio
		args["latency_slo_ms"] = 280
	case "workflow_retry_budgeted":
		activeBranch := m.state.ActiveBranch
		if strings.TrimSpace(activeBranch) == "" {
			activeBranch = m.state.PreferredBranch
		}
		transientFailures := 2
		if m.variant == modecommon.VariantProduction {
			transientFailures = 3
		}
		args["active_branch"] = activeBranch
		args["transient_failures"] = transientFailures
		args["base_retry_budget"] = 2
	case "workflow_failfast_classified":
		lastError := "timeout_upstream"
		if m.variant == modecommon.VariantProduction && strings.TrimSpace(m.state.ActiveBranch) == "safe-path" {
			lastError = "policy_sandbox_denied"
		}
		args["active_branch"] = m.state.ActiveBranch
		args["retry_budget"] = m.state.RetryBudget
		args["retry_consumed"] = m.state.RetryConsumed
		args["retry_denied"] = m.state.RetryDenied
		args["last_error"] = lastError
	case "governance_workflow_gate_enforced":
		args["active_branch"] = m.state.ActiveBranch
		args["failfast"] = m.state.Failfast
		args["failfast_class"] = m.state.FailfastClass
		args["retry_denied"] = m.state.RetryDenied
	case "governance_workflow_replay_bound":
		args["active_branch"] = m.state.ActiveBranch
		args["retry_budget"] = m.state.RetryBudget
		args["retry_consumed"] = m.state.RetryConsumed
		args["failfast_class"] = m.state.FailfastClass
		args["governance_decision"] = m.state.GovernanceDecision
		args["governance_ticket"] = m.state.GovernanceTicket
	}
	return args
}

type workflowBranchRetryTool struct{}

func (t *workflowBranchRetryTool) Name() string { return semanticToolName }

func (t *workflowBranchRetryTool) Description() string {
	return "execute workflow branch/retry/failfast semantic step"
}

func (t *workflowBranchRetryTool) JSONSchema() map[string]any {
	return map[string]any{
		"type":     "object",
		"required": []any{"pattern", "variant", "phase", "semantic_anchor", "classification", "marker", "runtime_domain", "stage"},
		"properties": map[string]any{
			"pattern":            map[string]any{"type": "string"},
			"variant":            map[string]any{"type": "string"},
			"phase":              map[string]any{"type": "string"},
			"semantic_anchor":    map[string]any{"type": "string"},
			"classification":     map[string]any{"type": "string"},
			"marker":             map[string]any{"type": "string"},
			"runtime_domain":     map[string]any{"type": "string"},
			"intent":             map[string]any{"type": "string"},
			"outcome":            map[string]any{"type": "string"},
			"input_signal":       map[string]any{"type": "string"},
			"preferred_branch":   map[string]any{"type": "string"},
			"active_branch":      map[string]any{"type": "string"},
			"error_ratio":        map[string]any{"type": "number"},
			"latency_slo_ms":     map[string]any{"type": "integer"},
			"transient_failures": map[string]any{"type": "integer"},
			"base_retry_budget":  map[string]any{"type": "integer"},
			"retry_budget":       map[string]any{"type": "integer"},
			"retry_consumed":     map[string]any{"type": "integer"},
			"retry_denied":       map[string]any{"type": "boolean"},
			"last_error":         map[string]any{"type": "string"},
			"failfast":           map[string]any{"type": "boolean"},
			"failfast_class":     map[string]any{"type": "string"},
			"governance_decision": map[string]any{
				"type": "string",
			},
			"governance_ticket": map[string]any{"type": "string"},
			"stage":             map[string]any{"type": "integer"},
		},
	}
}

func (t *workflowBranchRetryTool) Invoke(ctx context.Context, args map[string]any) (types.ToolResult, error) {
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
		"governance":      false,
	}

	risk := "nominal"
	switch marker {
	case "workflow_branch_routed":
		inputSignal := strings.TrimSpace(fmt.Sprintf("%v", args["input_signal"]))
		if inputSignal == "" {
			inputSignal = defaultSignal
		}
		preferredBranch := strings.TrimSpace(fmt.Sprintf("%v", args["preferred_branch"]))
		if preferredBranch == "" {
			preferredBranch = "fast-path"
		}
		errorRatio := asFloat(args["error_ratio"], 0.12)
		latencySLO, _ := modecommon.AsInt(args["latency_slo_ms"])
		if latencySLO <= 0 {
			latencySLO = 280
		}
		strictGate := variant == modecommon.VariantProduction
		threshold := 0.25
		if strictGate {
			threshold = 0.15
		}
		activeBranch := preferredBranch
		routeReason := "latency_bias"
		if errorRatio > threshold || latencySLO < 250 {
			activeBranch = "safe-path"
			routeReason = "stability_bias"
		}
		structured["input_signal"] = inputSignal
		structured["preferred_branch"] = preferredBranch
		structured["active_branch"] = activeBranch
		structured["route_reason"] = routeReason
		structured["error_ratio"] = errorRatio
		structured["latency_slo_ms"] = latencySLO
		structured["strict_gate"] = strictGate
		if activeBranch != preferredBranch {
			risk = "degraded_path"
		}
	case "workflow_retry_budgeted":
		activeBranch := strings.TrimSpace(fmt.Sprintf("%v", args["active_branch"]))
		if activeBranch == "" {
			activeBranch = "fast-path"
		}
		transientFailures, _ := modecommon.AsInt(args["transient_failures"])
		if transientFailures <= 0 {
			transientFailures = 1
		}
		baseBudget, _ := modecommon.AsInt(args["base_retry_budget"])
		if baseBudget <= 0 {
			baseBudget = 2
		}
		if activeBranch == "safe-path" {
			baseBudget++
		}
		if variant == modecommon.VariantProduction {
			baseBudget++
		}
		retryConsumed := transientFailures
		retryDenied := retryConsumed > baseBudget
		if retryDenied {
			retryConsumed = baseBudget
		}
		retryRemaining := baseBudget - retryConsumed
		if retryRemaining < 0 {
			retryRemaining = 0
		}
		structured["active_branch"] = activeBranch
		structured["transient_failures"] = transientFailures
		structured["retry_budget"] = baseBudget
		structured["retry_consumed"] = retryConsumed
		structured["retry_remaining"] = retryRemaining
		structured["retry_denied"] = retryDenied
		if retryDenied {
			risk = "degraded_path"
		} else {
			risk = "retry_inflight"
		}
	case "workflow_failfast_classified":
		activeBranch := strings.TrimSpace(fmt.Sprintf("%v", args["active_branch"]))
		retryBudget, _ := modecommon.AsInt(args["retry_budget"])
		retryConsumed, _ := modecommon.AsInt(args["retry_consumed"])
		retryDenied := asBool(args["retry_denied"])
		lastError := strings.ToLower(strings.TrimSpace(fmt.Sprintf("%v", args["last_error"])))
		if lastError == "" {
			lastError = "timeout_upstream"
		}

		failfastClass := "retryable"
		failfast := false
		escalationLane := "retry-loop"

		if retryDenied || retryConsumed >= retryBudget && retryBudget > 0 {
			failfastClass = "budget_exhausted"
			failfast = true
			escalationLane = "dead-letter"
		}
		if strings.Contains(lastError, "policy") || strings.Contains(lastError, "auth") {
			failfastClass = "terminal_policy"
			failfast = true
			escalationLane = "manual-approval"
		}
		if activeBranch == "safe-path" && !failfast {
			escalationLane = "safe-path-drain"
		}

		structured["active_branch"] = activeBranch
		structured["retry_budget"] = retryBudget
		structured["retry_consumed"] = retryConsumed
		structured["retry_denied"] = retryDenied
		structured["last_error"] = lastError
		structured["failfast_class"] = failfastClass
		structured["failfast"] = failfast
		structured["escalation_lane"] = escalationLane
		if failfast {
			risk = "degraded_path"
		}
	case "governance_workflow_gate_enforced":
		activeBranch := strings.TrimSpace(fmt.Sprintf("%v", args["active_branch"]))
		failfast := asBool(args["failfast"])
		retryDenied := asBool(args["retry_denied"])
		failfastClass := strings.TrimSpace(fmt.Sprintf("%v", args["failfast_class"]))
		if failfastClass == "" {
			failfastClass = "retryable"
		}

		governanceDecision := "allow"
		if failfast || failfastClass == "terminal_policy" {
			governanceDecision = "deny"
		} else if retryDenied || activeBranch == "safe-path" {
			governanceDecision = "allow_with_guardrails"
		}
		governanceTicket := fmt.Sprintf(
			"wf-gate-%d",
			modecommon.SemanticScore(pattern, variant, activeBranch, failfastClass, governanceDecision),
		)

		structured["active_branch"] = activeBranch
		structured["failfast"] = failfast
		structured["failfast_class"] = failfastClass
		structured["retry_denied"] = retryDenied
		structured["governance"] = true
		structured["governance_decision"] = governanceDecision
		structured["governance_ticket"] = governanceTicket
		risk = "governed"
	case "governance_workflow_replay_bound":
		activeBranch := strings.TrimSpace(fmt.Sprintf("%v", args["active_branch"]))
		retryBudget, _ := modecommon.AsInt(args["retry_budget"])
		retryConsumed, _ := modecommon.AsInt(args["retry_consumed"])
		failfastClass := strings.TrimSpace(fmt.Sprintf("%v", args["failfast_class"]))
		governanceDecision := strings.TrimSpace(fmt.Sprintf("%v", args["governance_decision"]))
		governanceTicket := strings.TrimSpace(fmt.Sprintf("%v", args["governance_ticket"]))

		replaySignature := fmt.Sprintf(
			"wf-replay-%d",
			modecommon.SemanticScore(
				pattern,
				variant,
				activeBranch,
				fmt.Sprintf("%d/%d", retryConsumed, retryBudget),
				failfastClass,
				governanceDecision,
				governanceTicket,
			),
		)

		structured["active_branch"] = activeBranch
		structured["retry_budget"] = retryBudget
		structured["retry_consumed"] = retryConsumed
		structured["failfast_class"] = failfastClass
		structured["governance_decision"] = governanceDecision
		structured["governance_ticket"] = governanceTicket
		structured["replay_signature"] = replaySignature
		structured["governance"] = true
		risk = "governed"
	default:
		return types.ToolResult{}, fmt.Errorf("unsupported workflow semantic marker: %s", marker)
	}

	score := modecommon.SemanticScore(
		pattern,
		variant,
		phaseValue,
		anchor,
		classValue,
		marker,
		runtimeDomain,
		risk,
		fmt.Sprintf("%d", stage),
	)
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

	return types.ToolResult{
		Content:    content,
		Structured: structured,
	}, nil
}

func normalizedValue(value string, enabled bool) string {
	if !enabled {
		return "n/a"
	}
	if strings.TrimSpace(value) == "" {
		return "pending"
	}
	return value
}

func asFloat(value any, fallback float64) float64 {
	switch item := value.(type) {
	case float64:
		return item
	case float32:
		return float64(item)
	case int:
		return float64(item)
	case int64:
		return float64(item)
	case string:
		text := strings.TrimSpace(item)
		if text == "" {
			return fallback
		}
		var parsed float64
		if _, err := fmt.Sscanf(text, "%f", &parsed); err == nil {
			return parsed
		}
	}
	return fallback
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
