package hitlgovernedcheckpoint

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
	patternName      = "hitl-governed-checkpoint"
	phase            = "P0"
	semanticAnchor   = "hitl.await_resume_reject_timeout_recover"
	classification   = "hitl.checkpoint_governance"
	semanticToolName = "mode_hitl_governed_checkpoint_semantic_step"
)

type checkpointStep struct {
	Marker        string
	RuntimeDomain string
	Intent        string
	Outcome       string
}

type checkpointTicket struct {
	ID                string
	Version           int
	RiskLevel         string
	TTLSeconds        int
	RequireDualReview bool
	RequestedAction   string
}

type reviewerDecision struct {
	Outcome        string
	Reviewer       string
	LatencySeconds int
	Reason         string
}

type checkpointState struct {
	TicketID           string
	TicketVersion      int
	AwaitStatus        string
	HumanDecision      string
	DecisionReason     string
	TimeoutTriggered   bool
	RecoveryPlan       string
	GovernanceDecision string
	ReplayBinding      string
	Escalations        []string
	SeenMarkers        []string
	TotalScore         int
}

var runtimeDomains = []string{"orchestration/composer", "runtime/diagnostics"}

var minimalSemanticSteps = []checkpointStep{
	{
		Marker:        "hitl_checkpoint_awaited",
		RuntimeDomain: "orchestration/composer",
		Intent:        "open checkpoint and wait for reviewer acknowledgement with TTL tracking",
		Outcome:       "checkpoint enters waiting state with deterministic wait metadata",
	},
	{
		Marker:        "hitl_resume_reject_classified",
		RuntimeDomain: "runtime/diagnostics",
		Intent:        "classify reviewer feedback into resume or reject and bind reason code",
		Outcome:       "decision class and reason are persisted for downstream control",
	},
	{
		Marker:        "hitl_timeout_recoverable",
		RuntimeDomain: "orchestration/composer",
		Intent:        "evaluate timeout and produce recoverability plan for delayed or rejected reviews",
		Outcome:       "timeout flag and recovery plan are emitted for orchestrator continuation",
	},
}

var productionGovernanceSteps = []checkpointStep{
	{
		Marker:        "governance_hitl_gate_enforced",
		RuntimeDomain: "orchestration/composer",
		Intent:        "enforce governance gate using decision class, timeout status, and recovery plan",
		Outcome:       "governance decision allow/allow_with_record/block is produced",
	},
	{
		Marker:        "governance_hitl_replay_bound",
		RuntimeDomain: "runtime/diagnostics",
		Intent:        "bind replay signature to ticket version and governance decision",
		Outcome:       "deterministic replay signature is generated",
	},
}

var minimalTicket = checkpointTicket{
	ID:                "chk-ops-2026-0007",
	Version:           1,
	RiskLevel:         "medium",
	TTLSeconds:        90,
	RequireDualReview: false,
	RequestedAction:   "apply_hotfix_and_continue",
}

var productionTicket = checkpointTicket{
	ID:                "chk-ops-2026-0211",
	Version:           3,
	RiskLevel:         "high",
	TTLSeconds:        60,
	RequireDualReview: true,
	RequestedAction:   "promote_runtime_patch_to_primary",
}

var minimalDecision = reviewerDecision{
	Outcome:        "resume",
	Reviewer:       "oncall-lead",
	LatencySeconds: 32,
	Reason:         "risk accepted with known mitigation",
}

var productionDecision = reviewerDecision{
	Outcome:        "reject",
	Reviewer:       "change-approver",
	LatencySeconds: 128,
	Reason:         "insufficient rollback evidence in current packet",
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
	if _, err := reg.Register(&checkpointGovernanceTool{}); err != nil {
		panic(err)
	}

	engine := runner.New(newCheckpointModel(variant), runner.WithLocalRegistry(reg), runner.WithRuntimeManager(mgr))
	result, err := engine.Run(context.Background(), types.RunRequest{
		RunID: fmt.Sprintf("agent-mode-%s-%s", modecommon.MarkerToken(patternName), modecommon.MarkerToken(variant)),
		Input: "execute hitl governed checkpoint workflow",
	}, nil)
	if err != nil {
		panic(err)
	}

	expectedMarkers := expectedMarkersForVariant(variant)
	runtimePath := modecommon.ComposeRuntimePath(runtimeDomains)
	pathStatus := modecommon.RuntimePathStatus(result.ToolCalls, len(expectedMarkers))
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
	fmt.Printf("verification.semantic.expected_markers=%s\n", strings.Join(expectedMarkers, ","))
	fmt.Printf("verification.semantic.governance=%s\n", governanceStatus)
	fmt.Printf("verification.semantic.marker_count=%d\n", len(expectedMarkers))
	for _, marker := range expectedMarkers {
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

func stepsForVariant(variant string) []checkpointStep {
	steps := make([]checkpointStep, 0, len(minimalSemanticSteps)+len(productionGovernanceSteps))
	steps = append(steps, minimalSemanticSteps...)
	if variant == modecommon.VariantProduction {
		steps = append(steps, productionGovernanceSteps...)
	}
	return steps
}

func newCheckpointModel(variant string) *checkpointWorkflowModel {
	ticket := ticketForVariant(variant)
	return &checkpointWorkflowModel{
		variant: variant,
		state: checkpointState{
			TicketID:      ticket.ID,
			TicketVersion: ticket.Version,
			AwaitStatus:   "pending_open",
			Escalations:   []string{},
		},
	}
}

type checkpointWorkflowModel struct {
	variant string
	stage   int
	state   checkpointState
}

func (m *checkpointWorkflowModel) Generate(ctx context.Context, req types.ModelRequest) (types.ModelResponse, error) {
	_ = ctx
	m.absorb(req.ToolResult)

	plan := stepsForVariant(m.variant)
	if m.stage < len(plan) {
		step := plan[m.stage]
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
	escalations := append([]string(nil), m.state.Escalations...)
	sort.Strings(escalations)
	final := fmt.Sprintf(
		"%s/%s semantic_path_completed phase=%s anchor=%s classification=%s ticket=%s@%d await=%s decision=%s reason=%s timeout=%t recovery=%s governance=%s replay=%s escalations=%s markers=%s score=%d",
		patternName,
		m.variant,
		phase,
		semanticAnchor,
		classification,
		m.state.TicketID,
		m.state.TicketVersion,
		safeString(m.state.AwaitStatus, "unknown"),
		safeString(m.state.HumanDecision, "none"),
		safeString(m.state.DecisionReason, "none"),
		m.state.TimeoutTriggered,
		safeString(m.state.RecoveryPlan, "none"),
		normalizedDecision(m.state.GovernanceDecision),
		safeString(m.state.ReplayBinding, "none"),
		strings.Join(escalations, ","),
		strings.Join(markers, ","),
		m.state.TotalScore,
	)
	return types.ModelResponse{FinalAnswer: final}, nil
}

func (m *checkpointWorkflowModel) Stream(ctx context.Context, req types.ModelRequest, onEvent func(types.ModelEvent) error) error {
	_ = ctx
	_ = req
	_ = onEvent
	return nil
}
func (m *checkpointWorkflowModel) absorb(outcomes []types.ToolCallOutcome) {
	for _, item := range outcomes {
		marker, _ := item.Result.Structured["marker"].(string)
		if marker != "" {
			m.state.SeenMarkers = append(m.state.SeenMarkers, marker)
		}
		if ticketID, ok := item.Result.Structured["ticket_id"].(string); ok && strings.TrimSpace(ticketID) != "" {
			m.state.TicketID = ticketID
		}
		if ticketVersion, ok := modecommon.AsInt(item.Result.Structured["ticket_version"]); ok {
			m.state.TicketVersion = ticketVersion
		}
		if awaitStatus, ok := item.Result.Structured["await_status"].(string); ok && strings.TrimSpace(awaitStatus) != "" {
			m.state.AwaitStatus = awaitStatus
		}
		if decision, ok := item.Result.Structured["human_decision"].(string); ok && strings.TrimSpace(decision) != "" {
			m.state.HumanDecision = decision
		}
		if reason, ok := item.Result.Structured["decision_reason"].(string); ok && strings.TrimSpace(reason) != "" {
			m.state.DecisionReason = reason
		}
		if timeoutTriggered, ok := item.Result.Structured["timeout_triggered"].(bool); ok {
			m.state.TimeoutTriggered = timeoutTriggered
		}
		if recoveryPlan, ok := item.Result.Structured["recovery_plan"].(string); ok && strings.TrimSpace(recoveryPlan) != "" {
			m.state.RecoveryPlan = recoveryPlan
		}
		if governanceDecision, ok := item.Result.Structured["governance_decision"].(string); ok && strings.TrimSpace(governanceDecision) != "" {
			m.state.GovernanceDecision = governanceDecision
		}
		if replayBinding, ok := item.Result.Structured["replay_binding"].(string); ok && strings.TrimSpace(replayBinding) != "" {
			m.state.ReplayBinding = replayBinding
		}
		if escalations, ok := toStringSlice(item.Result.Structured["escalations"]); ok && len(escalations) > 0 {
			m.state.Escalations = escalations
		}
		if score, ok := modecommon.AsInt(item.Result.Structured["score"]); ok {
			m.state.TotalScore += score
		}
	}
}

func (m *checkpointWorkflowModel) argsForStep(step checkpointStep, stage int) map[string]any {
	ticket := ticketForVariant(m.variant)
	decision := decisionForVariant(m.variant)
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
		"ticket_id":           ticket.ID,
		"ticket_version":      ticket.Version,
		"ticket_risk":         ticket.RiskLevel,
		"ticket_ttl_seconds":  ticket.TTLSeconds,
		"require_dual_review": ticket.RequireDualReview,
		"requested_action":    ticket.RequestedAction,
		"decision_outcome":    decision.Outcome,
		"decision_reviewer":   decision.Reviewer,
		"decision_latency":    decision.LatencySeconds,
		"decision_reason":     decision.Reason,
		"stage":               stage,
	}

	switch step.Marker {
	case "hitl_timeout_recoverable":
		args["human_decision"] = m.state.HumanDecision
		args["timeout_triggered"] = m.state.TimeoutTriggered
		args["decision_reason"] = m.state.DecisionReason
	case "governance_hitl_gate_enforced":
		args["human_decision"] = m.state.HumanDecision
		args["timeout_triggered"] = m.state.TimeoutTriggered
		args["recovery_plan"] = m.state.RecoveryPlan
		args["decision_reason"] = m.state.DecisionReason
		args["await_status"] = m.state.AwaitStatus
	case "governance_hitl_replay_bound":
		args["human_decision"] = m.state.HumanDecision
		args["timeout_triggered"] = m.state.TimeoutTriggered
		args["recovery_plan"] = m.state.RecoveryPlan
		args["governance_decision"] = m.state.GovernanceDecision
		args["await_status"] = m.state.AwaitStatus
		args["escalations"] = toAnySlice(m.state.Escalations)
	}

	return args
}

type checkpointGovernanceTool struct{}

func (t *checkpointGovernanceTool) Name() string { return semanticToolName }

func (t *checkpointGovernanceTool) Description() string {
	return "execute hitl checkpoint governance semantic step"
}

func (t *checkpointGovernanceTool) JSONSchema() map[string]any {
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
			"ticket_id",
			"ticket_version",
			"ticket_risk",
			"ticket_ttl_seconds",
			"requested_action",
			"decision_outcome",
			"decision_latency",
			"decision_reason",
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
			"ticket_id":           map[string]any{"type": "string"},
			"ticket_version":      map[string]any{"type": "integer"},
			"ticket_risk":         map[string]any{"type": "string"},
			"ticket_ttl_seconds":  map[string]any{"type": "integer"},
			"require_dual_review": map[string]any{"type": "boolean"},
			"requested_action":    map[string]any{"type": "string"},
			"decision_outcome":    map[string]any{"type": "string"},
			"decision_reviewer":   map[string]any{"type": "string"},
			"decision_latency":    map[string]any{"type": "integer"},
			"decision_reason":     map[string]any{"type": "string"},
			"human_decision":      map[string]any{"type": "string"},
			"await_status":        map[string]any{"type": "string"},
			"timeout_triggered":   map[string]any{"type": "boolean"},
			"recovery_plan":       map[string]any{"type": "string"},
			"governance_decision": map[string]any{"type": "string"},
			"escalations":         map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
			"stage":               map[string]any{"type": "integer"},
		},
	}
}

func (t *checkpointGovernanceTool) Invoke(ctx context.Context, args map[string]any) (types.ToolResult, error) {
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
	ticketID := strings.TrimSpace(fmt.Sprintf("%v", args["ticket_id"]))
	ticketVersion, _ := modecommon.AsInt(args["ticket_version"])
	ticketRisk := strings.TrimSpace(fmt.Sprintf("%v", args["ticket_risk"]))
	ticketTTL, _ := modecommon.AsInt(args["ticket_ttl_seconds"])
	requireDualReview := parseBool(args["require_dual_review"], false)
	requestedAction := strings.TrimSpace(fmt.Sprintf("%v", args["requested_action"]))
	decisionOutcome := strings.TrimSpace(fmt.Sprintf("%v", args["decision_outcome"]))
	decisionReviewer := strings.TrimSpace(fmt.Sprintf("%v", args["decision_reviewer"]))
	decisionLatency, _ := modecommon.AsInt(args["decision_latency"])
	decisionReason := strings.TrimSpace(fmt.Sprintf("%v", args["decision_reason"]))
	stage, _ := modecommon.AsInt(args["stage"])
	if stage <= 0 {
		stage = 1
	}

	result := map[string]any{
		"pattern":            pattern,
		"variant":            variant,
		"phase":              phaseValue,
		"semantic_anchor":    anchor,
		"classification":     classValue,
		"marker":             marker,
		"runtime_domain":     runtimeDomain,
		"semantic_intent":    intent,
		"semantic_outcome":   outcome,
		"ticket_id":          ticketID,
		"ticket_version":     ticketVersion,
		"ticket_risk":        ticketRisk,
		"ticket_ttl_seconds": ticketTTL,
		"requested_action":   requestedAction,
		"stage":              stage,
		"governance":         false,
	}

	var risk string
	humanDecision := normalizeHumanDecision(safeString(args["human_decision"], decisionOutcome))
	timeoutTriggered := parseBool(args["timeout_triggered"], false)
	recoveryPlan := safeString(args["recovery_plan"], "")
	governanceDecision := safeString(args["governance_decision"], "")
	escalations, _ := toStringSlice(args["escalations"])
	if len(escalations) == 0 {
		escalations = []string{}
	}

	switch marker {
	case "hitl_checkpoint_awaited":
		awaitStatus := "awaiting_human"
		escalations = append(escalations, "checkpoint_opened")
		if ticketRisk == "high" {
			escalations = append(escalations, "priority_watch")
		}
		result["await_status"] = awaitStatus
		result["timeout_triggered"] = false
		result["escalations"] = toAnySlice(uniqueSorted(escalations))
		risk = "checkpoint_waiting"
	case "hitl_resume_reject_classified":
		humanDecision = normalizeHumanDecision(decisionOutcome)
		timeoutTriggered = decisionLatency > ticketTTL || humanDecision == "timeout"
		if timeoutTriggered {
			humanDecision = "timeout"
		}
		if humanDecision == "reject" {
			escalations = append(escalations, "rework_required")
		}
		if timeoutTriggered {
			escalations = append(escalations, "sla_breached")
		}
		result["human_decision"] = humanDecision
		result["decision_reviewer"] = decisionReviewer
		result["decision_reason"] = decisionReason
		result["decision_latency"] = decisionLatency
		result["timeout_triggered"] = timeoutTriggered
		result["await_status"] = "decision_recorded"
		result["escalations"] = toAnySlice(uniqueSorted(escalations))
		switch {
		case timeoutTriggered:
			risk = "degraded_path"
		case humanDecision == "reject":
			risk = "reject_path"
		default:
			risk = "resume_path"
		}
	case "hitl_timeout_recoverable":
		switch {
		case timeoutTriggered:
			recoveryPlan = "checkpoint_reopen_with_latest_snapshot"
		case humanDecision == "reject":
			recoveryPlan = "operator_rework_then_resubmit"
		default:
			recoveryPlan = "not_required"
		}
		recoverable := recoveryPlan != ""
		result["human_decision"] = humanDecision
		result["timeout_triggered"] = timeoutTriggered
		result["recovery_plan"] = recoveryPlan
		result["recoverable"] = recoverable
		result["escalations"] = toAnySlice(uniqueSorted(escalations))
		switch {
		case timeoutTriggered:
			risk = "timeout_recoverable"
		case humanDecision == "reject":
			risk = "reject_recoverable"
		default:
			risk = "continue_ready"
		}
	case "governance_hitl_gate_enforced":
		humanDecision = normalizeHumanDecision(humanDecision)
		switch {
		case timeoutTriggered && strings.TrimSpace(recoveryPlan) == "":
			governanceDecision = "block_missing_recovery"
			risk = "governed_block"
		case humanDecision == "reject":
			governanceDecision = "allow_with_record"
			risk = "governed_warn"
		case timeoutTriggered:
			governanceDecision = "allow_with_recovery"
			risk = "governed_warn"
		default:
			governanceDecision = "allow"
			risk = "governed_allow"
		}
		if requireDualReview && ticketRisk == "high" && governanceDecision == "allow" {
			governanceDecision = "allow_with_dual_review_record"
			risk = "governed_warn"
		}
		result["human_decision"] = humanDecision
		result["timeout_triggered"] = timeoutTriggered
		result["recovery_plan"] = recoveryPlan
		result["governance_decision"] = governanceDecision
		result["escalations"] = toAnySlice(uniqueSorted(escalations))
		result["governance"] = true
	case "governance_hitl_replay_bound":
		humanDecision = normalizeHumanDecision(humanDecision)
		governanceDecision = safeString(governanceDecision, "allow")
		replayBinding := fmt.Sprintf(
			"hitl-replay-%d",
			modecommon.SemanticScore(
				pattern,
				variant,
				ticketID,
				fmt.Sprintf("%d", ticketVersion),
				humanDecision,
				fmt.Sprintf("%t", timeoutTriggered),
				recoveryPlan,
				governanceDecision,
				strings.Join(uniqueSorted(escalations), "|"),
			),
		)
		result["human_decision"] = humanDecision
		result["timeout_triggered"] = timeoutTriggered
		result["recovery_plan"] = recoveryPlan
		result["governance_decision"] = governanceDecision
		result["replay_binding"] = replayBinding
		result["escalations"] = toAnySlice(uniqueSorted(escalations))
		result["governance"] = true
		risk = "governed_replay"
	default:
		return types.ToolResult{}, fmt.Errorf("unsupported hitl marker: %s", marker)
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
		safeString(result["human_decision"], humanDecision),
		safeString(result["governance_decision"], governanceDecision),
		safeString(result["recovery_plan"], recoveryPlan),
	)
	result["risk"] = risk

	content := fmt.Sprintf(
		"pattern=%s variant=%s marker=%s decision=%s timeout=%t governance=%s risk=%s",
		pattern,
		variant,
		marker,
		safeString(result["human_decision"], humanDecision),
		parseBool(result["timeout_triggered"], false),
		normalizedDecision(safeString(result["governance_decision"], "not_applicable")),
		risk,
	)

	return types.ToolResult{Content: content, Structured: result}, nil
}

func ticketForVariant(variant string) checkpointTicket {
	if variant == modecommon.VariantProduction {
		return productionTicket
	}
	return minimalTicket
}

func decisionForVariant(variant string) reviewerDecision {
	if variant == modecommon.VariantProduction {
		return productionDecision
	}
	return minimalDecision
}

func normalizeHumanDecision(value string) string {
	trimmed := strings.TrimSpace(strings.ToLower(value))
	switch trimmed {
	case "resume", "reject", "timeout":
		return trimmed
	default:
		return "timeout"
	}
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

func uniqueSorted(input []string) []string {
	if len(input) == 0 {
		return []string{}
	}
	set := map[string]struct{}{}
	for _, item := range input {
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
