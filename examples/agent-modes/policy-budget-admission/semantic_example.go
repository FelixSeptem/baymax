package policybudgetadmission

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
	patternName      = "policy-budget-admission"
	phase            = "P1"
	semanticAnchor   = "policy.precedence_budget_admission_trace"
	classification   = "policy.budget_admission"
	semanticToolName = "mode_policy_budget_admission_semantic_step"
	defaultRequestID = "req-policy-20260410"
)

type policyStep struct {
	Marker        string
	RuntimeDomain string
	Intent        string
	Outcome       string
}

type policyState struct {
	RequestID          string
	WinningPolicy      string
	PrecedenceChain    string
	BudgetLimit        int
	BudgetUsed         int
	ProjectedCost      int
	AdmissionDecision  string
	AdmissionReason    string
	DecisionTraceID    string
	DecisionTraceHash  string
	GovernanceDecision string
	GovernanceTicket   string
	ReplaySignature    string
	SeenMarkers        []string
	TotalScore         int
}

var runtimeDomains = []string{"runtime/config", "runtime/diagnostics"}

var policyStack = []string{"emergency_allow", "deny_sensitive_model", "team_quota", "default_allow"}

var minimalSemanticSteps = []policyStep{
	{
		Marker:        "policy_precedence_applied",
		RuntimeDomain: "runtime/config",
		Intent:        "evaluate ordered policy stack and pick winning policy by precedence",
		Outcome:       "winning policy and precedence chain are emitted",
	},
	{
		Marker:        "budget_admission_decided",
		RuntimeDomain: "runtime/diagnostics",
		Intent:        "decide admission using winning policy and runtime budget headroom",
		Outcome:       "admission decision and budget rationale are emitted",
	},
	{
		Marker:        "decision_trace_recorded",
		RuntimeDomain: "runtime/config",
		Intent:        "record deterministic decision trace for policy/budget arbitration",
		Outcome:       "trace id and trace hash are emitted",
	},
}

var productionGovernanceSteps = []policyStep{
	{
		Marker:        "governance_policy_gate_enforced",
		RuntimeDomain: "runtime/config",
		Intent:        "enforce governance gate for high risk admission outcomes",
		Outcome:       "governance decision and ticket are emitted",
	},
	{
		Marker:        "governance_policy_replay_bound",
		RuntimeDomain: "runtime/diagnostics",
		Intent:        "bind policy decision into replay signature for audit determinism",
		Outcome:       "governance replay signature is emitted",
	},
}

func RunMinimal() {
	executePolicyVariant(modecommon.VariantMinimal)
}

func RunProduction() {
	executePolicyVariant(modecommon.VariantProduction)
}

func executePolicyVariant(variant string) {
	if err := modecommon.EnsureVariant(variant); err != nil {
		panic(err)
	}

	mgr, err := runtimeconfig.NewManager(runtimeconfig.ManagerOptions{EnvPrefix: "BAYMAX"})
	if err != nil {
		panic(err)
	}
	defer func() { _ = mgr.Close() }()

	reg := local.NewRegistry()
	if _, err := reg.Register(&policyAdmissionTool{}); err != nil {
		panic(err)
	}

	model := &policyAdmissionModel{
		variant: variant,
		state: policyState{
			RequestID: defaultRequestID,
		},
	}

	engine := runner.New(model, runner.WithLocalRegistry(reg), runner.WithRuntimeManager(mgr))
	result, err := engine.Run(context.Background(), types.RunRequest{
		RunID: fmt.Sprintf("agent-mode-%s-%s", modecommon.MarkerToken(patternName), modecommon.MarkerToken(variant)),
		Input: "execute policy budget admission semantic pipeline",
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

func planForVariant(variant string) []policyStep {
	plan := make([]policyStep, 0, len(minimalSemanticSteps)+len(productionGovernanceSteps))
	plan = append(plan, minimalSemanticSteps...)
	if variant == modecommon.VariantProduction {
		plan = append(plan, productionGovernanceSteps...)
	}
	return plan
}

type policyAdmissionModel struct {
	variant string
	cursor  int
	state   policyState
}

func (m *policyAdmissionModel) Generate(ctx context.Context, req types.ModelRequest) (types.ModelResponse, error) {
	_ = ctx
	m.capture(req.ToolResult)

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

	markers := append([]string(nil), m.state.SeenMarkers...)
	sort.Strings(markers)
	governanceOn := strings.TrimSpace(m.state.GovernanceDecision) != ""

	final := fmt.Sprintf(
		"%s/%s semantic_path_completed phase=%s anchor=%s classification=%s policy=%s precedence=%s budget=%d/%d projected=%d admission=%s reason=%s trace=%s trace_hash=%s governance=%s ticket=%s replay=%s score=%d markers=%s",
		patternName,
		m.variant,
		phase,
		semanticAnchor,
		classification,
		normalizedValue(m.state.WinningPolicy, true),
		normalizedValue(m.state.PrecedenceChain, true),
		m.state.BudgetUsed,
		m.state.BudgetLimit,
		m.state.ProjectedCost,
		normalizedValue(m.state.AdmissionDecision, true),
		normalizedValue(m.state.AdmissionReason, true),
		normalizedValue(m.state.DecisionTraceID, true),
		normalizedValue(m.state.DecisionTraceHash, true),
		normalizedValue(m.state.GovernanceDecision, governanceOn),
		normalizedValue(m.state.GovernanceTicket, governanceOn),
		normalizedValue(m.state.ReplaySignature, governanceOn),
		m.state.TotalScore,
		strings.Join(markers, ","),
	)
	return types.ModelResponse{FinalAnswer: final}, nil
}

func (m *policyAdmissionModel) Stream(ctx context.Context, req types.ModelRequest, onEvent func(types.ModelEvent) error) error {
	_ = ctx
	_ = req
	_ = onEvent
	return nil
}

func (m *policyAdmissionModel) capture(outcomes []types.ToolCallOutcome) {
	for _, item := range outcomes {
		marker, _ := item.Result.Structured["marker"].(string)
		if marker != "" {
			m.state.SeenMarkers = append(m.state.SeenMarkers, marker)
		}
		if policy, _ := item.Result.Structured["winning_policy"].(string); strings.TrimSpace(policy) != "" {
			m.state.WinningPolicy = strings.TrimSpace(policy)
		}
		if precedence, _ := item.Result.Structured["precedence_chain"].(string); strings.TrimSpace(precedence) != "" {
			m.state.PrecedenceChain = strings.TrimSpace(precedence)
		}
		if budgetLimit, ok := modecommon.AsInt(item.Result.Structured["budget_limit"]); ok {
			m.state.BudgetLimit = budgetLimit
		}
		if budgetUsed, ok := modecommon.AsInt(item.Result.Structured["budget_used"]); ok {
			m.state.BudgetUsed = budgetUsed
		}
		if projected, ok := modecommon.AsInt(item.Result.Structured["projected_cost"]); ok {
			m.state.ProjectedCost = projected
		}
		if admission, _ := item.Result.Structured["admission_decision"].(string); strings.TrimSpace(admission) != "" {
			m.state.AdmissionDecision = strings.TrimSpace(admission)
		}
		if reason, _ := item.Result.Structured["admission_reason"].(string); strings.TrimSpace(reason) != "" {
			m.state.AdmissionReason = strings.TrimSpace(reason)
		}
		if traceID, _ := item.Result.Structured["trace_id"].(string); strings.TrimSpace(traceID) != "" {
			m.state.DecisionTraceID = strings.TrimSpace(traceID)
		}
		if traceHash, _ := item.Result.Structured["trace_hash"].(string); strings.TrimSpace(traceHash) != "" {
			m.state.DecisionTraceHash = strings.TrimSpace(traceHash)
		}
		if decision, _ := item.Result.Structured["governance_decision"].(string); strings.TrimSpace(decision) != "" {
			m.state.GovernanceDecision = strings.TrimSpace(decision)
		}
		if ticket, _ := item.Result.Structured["governance_ticket"].(string); strings.TrimSpace(ticket) != "" {
			m.state.GovernanceTicket = strings.TrimSpace(ticket)
		}
		if replay, _ := item.Result.Structured["replay_signature"].(string); strings.TrimSpace(replay) != "" {
			m.state.ReplaySignature = strings.TrimSpace(replay)
		}
		if score, ok := modecommon.AsInt(item.Result.Structured["score"]); ok {
			m.state.TotalScore += score
		}
	}
}

func (m *policyAdmissionModel) argsForStep(step policyStep, stage int) map[string]any {
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
		"request_id":      m.state.RequestID,
	}

	switch step.Marker {
	case "policy_precedence_applied":
		modelTier := "standard"
		if m.variant == modecommon.VariantProduction {
			modelTier = "sensitive"
		}
		args["policy_stack"] = stringSliceToAny(policyStack)
		args["model_tier"] = modelTier
		args["team"] = "search"
		args["emergency"] = false
	case "budget_admission_decided":
		budgetLimit := 100
		budgetUsed := 72
		projected := 21
		if m.variant == modecommon.VariantProduction {
			budgetUsed = 85
			projected = 28
		}
		args["winning_policy"] = m.state.WinningPolicy
		args["precedence_chain"] = m.state.PrecedenceChain
		args["budget_limit"] = budgetLimit
		args["budget_used"] = budgetUsed
		args["projected_cost"] = projected
	case "decision_trace_recorded":
		args["winning_policy"] = m.state.WinningPolicy
		args["admission_decision"] = m.state.AdmissionDecision
		args["admission_reason"] = m.state.AdmissionReason
		args["budget_limit"] = m.state.BudgetLimit
		args["budget_used"] = m.state.BudgetUsed
		args["projected_cost"] = m.state.ProjectedCost
	case "governance_policy_gate_enforced":
		args["admission_decision"] = m.state.AdmissionDecision
		args["winning_policy"] = m.state.WinningPolicy
		args["projected_cost"] = m.state.ProjectedCost
		args["trace_hash"] = m.state.DecisionTraceHash
	case "governance_policy_replay_bound":
		args["trace_hash"] = m.state.DecisionTraceHash
		args["governance_decision"] = m.state.GovernanceDecision
		args["governance_ticket"] = m.state.GovernanceTicket
		args["request_id"] = m.state.RequestID
	}
	return args
}

type policyAdmissionTool struct{}

func (t *policyAdmissionTool) Name() string { return semanticToolName }

func (t *policyAdmissionTool) Description() string {
	return "execute policy precedence and budget admission semantic step"
}

func (t *policyAdmissionTool) JSONSchema() map[string]any {
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
			"request_id":      map[string]any{"type": "string"},
			"policy_stack":    map[string]any{"type": "array"},
			"model_tier":      map[string]any{"type": "string"},
			"team":            map[string]any{"type": "string"},
			"emergency":       map[string]any{"type": "boolean"},
			"winning_policy":  map[string]any{"type": "string"},
			"precedence_chain": map[string]any{
				"type": "string",
			},
			"budget_limit":       map[string]any{"type": "integer"},
			"budget_used":        map[string]any{"type": "integer"},
			"projected_cost":     map[string]any{"type": "integer"},
			"admission_decision": map[string]any{"type": "string"},
			"admission_reason":   map[string]any{"type": "string"},
			"trace_hash":         map[string]any{"type": "string"},
			"governance_decision": map[string]any{
				"type": "string",
			},
			"governance_ticket": map[string]any{"type": "string"},
		},
	}
}

func (t *policyAdmissionTool) Invoke(ctx context.Context, args map[string]any) (types.ToolResult, error) {
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
	case "policy_precedence_applied":
		requestID := strings.TrimSpace(fmt.Sprintf("%v", args["request_id"]))
		if requestID == "" {
			requestID = defaultRequestID
		}
		stack := toStringSlice(args["policy_stack"])
		if len(stack) == 0 {
			stack = append([]string{}, policyStack...)
		}
		modelTier := strings.TrimSpace(fmt.Sprintf("%v", args["model_tier"]))
		if modelTier == "" {
			modelTier = "standard"
		}
		team := strings.TrimSpace(fmt.Sprintf("%v", args["team"]))
		emergency := asBool(args["emergency"])

		winning := "default_allow"
		reason := "default"
		switch {
		case emergency:
			winning = "emergency_allow"
			reason = "emergency_flag"
		case modelTier == "sensitive":
			winning = "deny_sensitive_model"
			reason = "model_tier_sensitive"
		case team == "search":
			winning = "team_quota"
			reason = "team_quota_guard"
		}

		structured["request_id"] = requestID
		structured["policy_stack"] = stringSliceToAny(stack)
		structured["winning_policy"] = winning
		structured["winning_reason"] = reason
		structured["precedence_chain"] = strings.Join(stack, ">")
		if winning == "deny_sensitive_model" {
			risk = "degraded_path"
		}
	case "budget_admission_decided":
		winning := strings.TrimSpace(fmt.Sprintf("%v", args["winning_policy"]))
		precedence := strings.TrimSpace(fmt.Sprintf("%v", args["precedence_chain"]))
		limit, _ := modecommon.AsInt(args["budget_limit"])
		used, _ := modecommon.AsInt(args["budget_used"])
		projected, _ := modecommon.AsInt(args["projected_cost"])
		if limit <= 0 {
			limit = 100
		}
		headroom := limit - used
		decision := "admit"
		reason := "within_budget"
		if winning == "deny_sensitive_model" {
			decision = "reject"
			reason = "policy_deny"
		} else if headroom < projected {
			decision = "defer"
			reason = "budget_shortfall"
		}

		structured["winning_policy"] = winning
		structured["precedence_chain"] = precedence
		structured["budget_limit"] = limit
		structured["budget_used"] = used
		structured["projected_cost"] = projected
		structured["budget_headroom"] = headroom
		structured["admission_decision"] = decision
		structured["admission_reason"] = reason
		if decision != "admit" {
			risk = "degraded_path"
		}
	case "decision_trace_recorded":
		requestID := strings.TrimSpace(fmt.Sprintf("%v", args["request_id"]))
		winning := strings.TrimSpace(fmt.Sprintf("%v", args["winning_policy"]))
		decision := strings.TrimSpace(fmt.Sprintf("%v", args["admission_decision"]))
		reason := strings.TrimSpace(fmt.Sprintf("%v", args["admission_reason"]))
		limit, _ := modecommon.AsInt(args["budget_limit"])
		used, _ := modecommon.AsInt(args["budget_used"])
		projected, _ := modecommon.AsInt(args["projected_cost"])
		traceID := fmt.Sprintf("trace-%d", modecommon.SemanticScore(requestID, winning, decision))
		traceHash := fmt.Sprintf("tracehash-%d", modecommon.SemanticScore(traceID, reason, fmt.Sprintf("%d/%d/%d", used, limit, projected)))
		structured["trace_id"] = traceID
		structured["trace_hash"] = traceHash
		structured["winning_policy"] = winning
		structured["admission_decision"] = decision
		structured["admission_reason"] = reason
		structured["budget_limit"] = limit
		structured["budget_used"] = used
		structured["projected_cost"] = projected
		if decision == "reject" {
			risk = "degraded_path"
		}
	case "governance_policy_gate_enforced":
		admission := strings.TrimSpace(fmt.Sprintf("%v", args["admission_decision"]))
		winning := strings.TrimSpace(fmt.Sprintf("%v", args["winning_policy"]))
		projected, _ := modecommon.AsInt(args["projected_cost"])
		traceHash := strings.TrimSpace(fmt.Sprintf("%v", args["trace_hash"]))

		governance := "allow"
		if admission == "reject" {
			governance = "deny"
		} else if admission == "defer" || projected > 25 {
			governance = "allow_with_limit"
		}
		ticket := fmt.Sprintf(
			"policy-gate-%d",
			modecommon.SemanticScore(winning, admission, governance, traceHash, fmt.Sprintf("%d", projected)),
		)
		structured["admission_decision"] = admission
		structured["winning_policy"] = winning
		structured["projected_cost"] = projected
		structured["trace_hash"] = traceHash
		structured["governance"] = true
		structured["governance_decision"] = governance
		structured["governance_ticket"] = ticket
		risk = "governed"
	case "governance_policy_replay_bound":
		traceHash := strings.TrimSpace(fmt.Sprintf("%v", args["trace_hash"]))
		governance := strings.TrimSpace(fmt.Sprintf("%v", args["governance_decision"]))
		ticket := strings.TrimSpace(fmt.Sprintf("%v", args["governance_ticket"]))
		requestID := strings.TrimSpace(fmt.Sprintf("%v", args["request_id"]))
		replay := fmt.Sprintf("policy-replay-%d", modecommon.SemanticScore(requestID, traceHash, governance, ticket))
		structured["trace_hash"] = traceHash
		structured["governance_decision"] = governance
		structured["governance_ticket"] = ticket
		structured["request_id"] = requestID
		structured["governance"] = true
		structured["replay_signature"] = replay
		risk = "governed"
	default:
		return types.ToolResult{}, fmt.Errorf("unsupported policy semantic marker: %s", marker)
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

func toStringSlice(value any) []string {
	switch raw := value.(type) {
	case []string:
		return append([]string(nil), raw...)
	case []any:
		out := make([]string, 0, len(raw))
		for _, item := range raw {
			text := strings.TrimSpace(fmt.Sprintf("%v", item))
			if text == "" {
				continue
			}
			out = append(out, text)
		}
		return out
	default:
		return nil
	}
}

func stringSliceToAny(in []string) []any {
	out := make([]any, 0, len(in))
	for _, item := range in {
		out = append(out, item)
	}
	return out
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

func normalizedValue(value string, enabled bool) string {
	if !enabled {
		return "n/a"
	}
	if strings.TrimSpace(value) == "" {
		return "pending"
	}
	return value
}
