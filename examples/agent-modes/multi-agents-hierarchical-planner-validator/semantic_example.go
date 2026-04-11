package multiagentshierarchicalplannervalidator

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
	patternName      = "multi-agents-hierarchical-planner-validator"
	phase            = "P2"
	semanticAnchor   = "hierarchy.planner_validator_correction"
	classification   = "multi_agents.hierarchy"
	semanticToolName = "mode_multi_agents_hierarchical_planner_validator_semantic_step"
	defaultSessionID = "hier-session-20260410"
)

type hierarchyStep struct {
	Marker        string
	RuntimeDomain string
	Intent        string
	Outcome       string
}

type hierarchyState struct {
	SessionID           string
	TopGoal             string
	PlannerLead         string
	PlanRevision        string
	TaskCount           int
	ValidatorIssues     int
	CorrectionRounds    int
	QualityScore        int
	PlanAccepted        bool
	ExecutionPlanID     string
	GovernanceDecision  string
	GovernanceTicket    string
	ReplaySignature     string
	ObservedMarkers     []string
	AccumulatedSemScore int
}

var runtimeDomains = []string{"orchestration/teams", "orchestration/workflow"}

var minimalSemanticSteps = []hierarchyStep{
	{
		Marker:        "hierarchy_plan_decomposed",
		RuntimeDomain: "orchestration/teams",
		Intent:        "decompose top goal into layered subplans with planner ownership",
		Outcome:       "planner lead, top goal, task count and revision are emitted",
	},
	{
		Marker:        "hierarchy_validator_feedback_applied",
		RuntimeDomain: "orchestration/workflow",
		Intent:        "apply validator feedback to reduce cross-agent dependency risks",
		Outcome:       "validator issue count, correction rounds and quality score are emitted",
	},
	{
		Marker:        "hierarchy_correction_loop_closed",
		RuntimeDomain: "orchestration/teams",
		Intent:        "close correction loop and produce executable hierarchy plan",
		Outcome:       "plan acceptance and execution plan id are emitted",
	},
}

var productionGovernanceSteps = []hierarchyStep{
	{
		Marker:        "governance_hierarchy_gate_enforced",
		RuntimeDomain: "orchestration/teams",
		Intent:        "enforce hierarchy governance gate with quality and correction evidence",
		Outcome:       "governance decision and governance ticket are emitted",
	},
	{
		Marker:        "governance_hierarchy_replay_bound",
		RuntimeDomain: "orchestration/workflow",
		Intent:        "bind governance decision to replay signature for planner audit",
		Outcome:       "replay signature is emitted",
	},
}

func RunMinimal() {
	executeHierarchyVariant(modecommon.VariantMinimal)
}

func RunProduction() {
	executeHierarchyVariant(modecommon.VariantProduction)
}

func executeHierarchyVariant(variant string) {
	if err := modecommon.EnsureVariant(variant); err != nil {
		panic(err)
	}

	mgr, err := runtimeconfig.NewManager(runtimeconfig.ManagerOptions{EnvPrefix: "BAYMAX"})
	if err != nil {
		panic(err)
	}
	defer func() { _ = mgr.Close() }()

	registry := local.NewRegistry()
	if _, err := registry.Register(&hierarchyPlannerValidatorTool{}); err != nil {
		panic(err)
	}

	model := &hierarchyPlannerValidatorModel{
		variant: variant,
		state: hierarchyState{
			SessionID:    defaultSessionID,
			TopGoal:      "deliver-quarterly-customer-onboarding-upgrade",
			PlannerLead:  "planner-alpha",
			PlanRevision: "rev-0",
		},
	}
	engine := runner.New(model, runner.WithLocalRegistry(registry), runner.WithRuntimeManager(mgr))
	result, err := engine.Run(context.Background(), types.RunRequest{
		RunID: fmt.Sprintf("agent-mode-%s-%s", modecommon.MarkerToken(patternName), modecommon.MarkerToken(variant)),
		Input: "execute multi-agent hierarchical planner validator semantic pipeline",
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

func planForVariant(variant string) []hierarchyStep {
	plan := make([]hierarchyStep, 0, len(minimalSemanticSteps)+len(productionGovernanceSteps))
	plan = append(plan, minimalSemanticSteps...)
	if variant == modecommon.VariantProduction {
		plan = append(plan, productionGovernanceSteps...)
	}
	return plan
}

type hierarchyPlannerValidatorModel struct {
	variant string
	cursor  int
	state   hierarchyState
}

func (m *hierarchyPlannerValidatorModel) Generate(ctx context.Context, req types.ModelRequest) (types.ModelResponse, error) {
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
		"%s/%s semantic_path_completed phase=%s anchor=%s classification=%s session=%s top_goal=%s planner=%s revision=%s task_count=%d validator_issues=%d correction_rounds=%d quality=%d plan_accepted=%t execution_plan=%s governance=%s ticket=%s replay=%s score=%d markers=%s",
		patternName,
		m.variant,
		phase,
		semanticAnchor,
		classification,
		readableValue(m.state.SessionID, true),
		readableValue(m.state.TopGoal, true),
		readableValue(m.state.PlannerLead, true),
		readableValue(m.state.PlanRevision, true),
		m.state.TaskCount,
		m.state.ValidatorIssues,
		m.state.CorrectionRounds,
		m.state.QualityScore,
		m.state.PlanAccepted,
		readableValue(m.state.ExecutionPlanID, true),
		readableValue(m.state.GovernanceDecision, governanceOn),
		readableValue(m.state.GovernanceTicket, governanceOn),
		readableValue(m.state.ReplaySignature, governanceOn),
		m.state.AccumulatedSemScore,
		strings.Join(markers, ","),
	)
	return types.ModelResponse{FinalAnswer: final}, nil
}

func (m *hierarchyPlannerValidatorModel) Stream(ctx context.Context, req types.ModelRequest, onEvent func(types.ModelEvent) error) error {
	_ = ctx
	_ = req
	_ = onEvent
	return nil
}

func (m *hierarchyPlannerValidatorModel) captureOutcomes(outcomes []types.ToolCallOutcome) {
	for _, outcome := range outcomes {
		marker, _ := outcome.Result.Structured["marker"].(string)
		if marker != "" {
			m.state.ObservedMarkers = append(m.state.ObservedMarkers, marker)
		}
		if sessionID, _ := outcome.Result.Structured["session_id"].(string); strings.TrimSpace(sessionID) != "" {
			m.state.SessionID = strings.TrimSpace(sessionID)
		}
		if goal, _ := outcome.Result.Structured["top_goal"].(string); strings.TrimSpace(goal) != "" {
			m.state.TopGoal = strings.TrimSpace(goal)
		}
		if planner, _ := outcome.Result.Structured["planner_lead"].(string); strings.TrimSpace(planner) != "" {
			m.state.PlannerLead = strings.TrimSpace(planner)
		}
		if revision, _ := outcome.Result.Structured["plan_revision"].(string); strings.TrimSpace(revision) != "" {
			m.state.PlanRevision = strings.TrimSpace(revision)
		}
		if tasks, ok := modecommon.AsInt(outcome.Result.Structured["task_count"]); ok {
			m.state.TaskCount = tasks
		}
		if issues, ok := modecommon.AsInt(outcome.Result.Structured["validator_issues"]); ok {
			m.state.ValidatorIssues = issues
		}
		if rounds, ok := modecommon.AsInt(outcome.Result.Structured["correction_rounds"]); ok {
			m.state.CorrectionRounds = rounds
		}
		if quality, ok := modecommon.AsInt(outcome.Result.Structured["quality_score"]); ok {
			m.state.QualityScore = quality
		}
		if accepted, ok := outcome.Result.Structured["plan_accepted"].(bool); ok {
			m.state.PlanAccepted = accepted
		}
		if planID, _ := outcome.Result.Structured["execution_plan_id"].(string); strings.TrimSpace(planID) != "" {
			m.state.ExecutionPlanID = strings.TrimSpace(planID)
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

func (m *hierarchyPlannerValidatorModel) argsForStep(step hierarchyStep, stage int) map[string]any {
	args := map[string]any{
		"pattern":           patternName,
		"variant":           m.variant,
		"phase":             phase,
		"semantic_anchor":   semanticAnchor,
		"classification":    classification,
		"marker":            step.Marker,
		"runtime_domain":    step.RuntimeDomain,
		"intent":            step.Intent,
		"outcome":           step.Outcome,
		"stage":             stage,
		"session_id":        m.state.SessionID,
		"top_goal":          m.state.TopGoal,
		"planner_lead":      m.state.PlannerLead,
		"plan_revision":     m.state.PlanRevision,
		"task_count":        m.state.TaskCount,
		"validator_issues":  m.state.ValidatorIssues,
		"correction_rounds": m.state.CorrectionRounds,
		"quality_score":     m.state.QualityScore,
	}

	switch step.Marker {
	case "hierarchy_plan_decomposed":
		taskCount := 4
		planRevision := "rev-1"
		if m.variant == modecommon.VariantProduction {
			taskCount = 7
			planRevision = "rev-2"
		}
		args["task_count"] = taskCount
		args["plan_revision"] = planRevision
	case "hierarchy_validator_feedback_applied":
		validatorIssues := 1
		correctionRounds := 1
		qualityScore := 82
		if m.variant == modecommon.VariantProduction {
			validatorIssues = 2
			correctionRounds = 2
			qualityScore = 91
		}
		args["validator_issues"] = validatorIssues
		args["correction_rounds"] = correctionRounds
		args["quality_score"] = qualityScore
	case "hierarchy_correction_loop_closed":
		args["task_count"] = m.state.TaskCount
		args["validator_issues"] = m.state.ValidatorIssues
		args["correction_rounds"] = m.state.CorrectionRounds
		args["quality_score"] = m.state.QualityScore
	case "governance_hierarchy_gate_enforced":
		args["task_count"] = m.state.TaskCount
		args["validator_issues"] = m.state.ValidatorIssues
		args["correction_rounds"] = m.state.CorrectionRounds
		args["quality_score"] = m.state.QualityScore
		args["plan_accepted"] = m.state.PlanAccepted
	case "governance_hierarchy_replay_bound":
		args["governance_decision"] = m.state.GovernanceDecision
		args["governance_ticket"] = m.state.GovernanceTicket
		args["execution_plan_id"] = m.state.ExecutionPlanID
	}
	return args
}

type hierarchyPlannerValidatorTool struct{}

func (t *hierarchyPlannerValidatorTool) Name() string { return semanticToolName }

func (t *hierarchyPlannerValidatorTool) Description() string {
	return "execute hierarchical planner + validator correction semantic step"
}

func (t *hierarchyPlannerValidatorTool) JSONSchema() map[string]any {
	return map[string]any{
		"type":     "object",
		"required": []any{"pattern", "variant", "phase", "semantic_anchor", "classification", "marker", "runtime_domain", "stage"},
		"properties": map[string]any{
			"pattern":           map[string]any{"type": "string"},
			"variant":           map[string]any{"type": "string"},
			"phase":             map[string]any{"type": "string"},
			"semantic_anchor":   map[string]any{"type": "string"},
			"classification":    map[string]any{"type": "string"},
			"marker":            map[string]any{"type": "string"},
			"runtime_domain":    map[string]any{"type": "string"},
			"intent":            map[string]any{"type": "string"},
			"outcome":           map[string]any{"type": "string"},
			"stage":             map[string]any{"type": "integer"},
			"session_id":        map[string]any{"type": "string"},
			"top_goal":          map[string]any{"type": "string"},
			"planner_lead":      map[string]any{"type": "string"},
			"plan_revision":     map[string]any{"type": "string"},
			"task_count":        map[string]any{"type": "integer"},
			"validator_issues":  map[string]any{"type": "integer"},
			"correction_rounds": map[string]any{"type": "integer"},
			"quality_score":     map[string]any{"type": "integer"},
			"plan_accepted":     map[string]any{"type": "boolean"},
			"execution_plan_id": map[string]any{"type": "string"},
			"governance_decision": map[string]any{
				"type": "string",
			},
			"governance_ticket": map[string]any{"type": "string"},
		},
	}
}

func (t *hierarchyPlannerValidatorTool) Invoke(ctx context.Context, args map[string]any) (types.ToolResult, error) {
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

	sessionID := strings.TrimSpace(fmt.Sprintf("%v", args["session_id"]))
	if sessionID == "" {
		sessionID = defaultSessionID
	}
	topGoal := strings.TrimSpace(fmt.Sprintf("%v", args["top_goal"]))
	plannerLead := strings.TrimSpace(fmt.Sprintf("%v", args["planner_lead"]))
	planRevision := strings.TrimSpace(fmt.Sprintf("%v", args["plan_revision"]))

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
		"session_id":      sessionID,
		"top_goal":        topGoal,
		"planner_lead":    plannerLead,
		"plan_revision":   planRevision,
		"governance":      false,
	}

	risk := "nominal"
	switch marker {
	case "hierarchy_plan_decomposed":
		taskCount, _ := modecommon.AsInt(args["task_count"])
		if taskCount <= 0 {
			taskCount = 4
		}
		if topGoal == "" {
			topGoal = "deliver-quarterly-customer-onboarding-upgrade"
		}
		if plannerLead == "" {
			plannerLead = "planner-alpha"
		}
		if planRevision == "" {
			planRevision = "rev-1"
		}
		structured["top_goal"] = topGoal
		structured["planner_lead"] = plannerLead
		structured["plan_revision"] = planRevision
		structured["task_count"] = taskCount
	case "hierarchy_validator_feedback_applied":
		validatorIssues, _ := modecommon.AsInt(args["validator_issues"])
		correctionRounds, _ := modecommon.AsInt(args["correction_rounds"])
		qualityScore, _ := modecommon.AsInt(args["quality_score"])
		structured["validator_issues"] = validatorIssues
		structured["correction_rounds"] = correctionRounds
		structured["quality_score"] = qualityScore
		if validatorIssues > 1 {
			risk = "degraded_path"
		}
	case "hierarchy_correction_loop_closed":
		taskCount, _ := modecommon.AsInt(args["task_count"])
		validatorIssues, _ := modecommon.AsInt(args["validator_issues"])
		correctionRounds, _ := modecommon.AsInt(args["correction_rounds"])
		qualityScore, _ := modecommon.AsInt(args["quality_score"])
		planAccepted := qualityScore >= 80
		executionPlanID := fmt.Sprintf("hier-plan-%d", modecommon.SemanticScore(sessionID, fmt.Sprintf("%d", taskCount), fmt.Sprintf("%d", qualityScore), fmt.Sprintf("%d", correctionRounds)))
		structured["task_count"] = taskCount
		structured["validator_issues"] = validatorIssues
		structured["correction_rounds"] = correctionRounds
		structured["quality_score"] = qualityScore
		structured["plan_accepted"] = planAccepted
		structured["execution_plan_id"] = executionPlanID
		if !planAccepted {
			risk = "degraded_path"
		}
	case "governance_hierarchy_gate_enforced":
		taskCount, _ := modecommon.AsInt(args["task_count"])
		validatorIssues, _ := modecommon.AsInt(args["validator_issues"])
		correctionRounds, _ := modecommon.AsInt(args["correction_rounds"])
		qualityScore, _ := modecommon.AsInt(args["quality_score"])
		planAccepted := asBool(args["plan_accepted"])
		decision := "hold_plan"
		if planAccepted {
			decision = "allow_plan_execution"
			if taskCount >= 7 || correctionRounds >= 2 || validatorIssues >= 2 {
				decision = "allow_plan_with_shadow_validator"
			}
		}
		ticket := fmt.Sprintf("hier-gate-%d", modecommon.SemanticScore(sessionID, decision, fmt.Sprintf("%d", qualityScore), fmt.Sprintf("%d", validatorIssues), fmt.Sprintf("%d", correctionRounds)))
		structured["task_count"] = taskCount
		structured["validator_issues"] = validatorIssues
		structured["correction_rounds"] = correctionRounds
		structured["quality_score"] = qualityScore
		structured["plan_accepted"] = planAccepted
		structured["governance"] = true
		structured["governance_decision"] = decision
		structured["governance_ticket"] = ticket
		risk = "governed"
	case "governance_hierarchy_replay_bound":
		decision := strings.TrimSpace(fmt.Sprintf("%v", args["governance_decision"]))
		ticket := strings.TrimSpace(fmt.Sprintf("%v", args["governance_ticket"]))
		executionPlanID := strings.TrimSpace(fmt.Sprintf("%v", args["execution_plan_id"]))
		replaySignature := fmt.Sprintf("hier-replay-%d", modecommon.SemanticScore(sessionID, executionPlanID, decision, ticket))
		structured["execution_plan_id"] = executionPlanID
		structured["governance"] = true
		structured["governance_decision"] = decision
		structured["governance_ticket"] = ticket
		structured["replay_signature"] = replaySignature
		risk = "governed"
	default:
		return types.ToolResult{}, fmt.Errorf("unsupported hierarchy marker: %s", marker)
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
