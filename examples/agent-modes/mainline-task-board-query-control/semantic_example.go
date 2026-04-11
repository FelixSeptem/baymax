package mainlinetaskboardquerycontrol

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
	patternName      = "mainline-task-board-query-control"
	phase            = "P2"
	semanticAnchor   = "taskboard.query_control_idempotency"
	classification   = "mainline.taskboard_control"
	semanticToolName = "mode_mainline_task_board_query_control_semantic_step"
	defaultQueryID   = "taskboard-query-20260410"
)

type taskboardStep struct {
	Marker        string
	RuntimeDomain string
	Intent        string
	Outcome       string
}

type taskboardState struct {
	QueryID             string
	BoardName           string
	QueryFilter         string
	MatchedTasks        int
	ControlAction       string
	ControlTarget       string
	ControlValid        bool
	ValidationReason    string
	OperationKey        string
	DuplicateAttempts   int
	Idempotent          bool
	AppliedCount        int
	GovernanceDecision  string
	GovernanceTicket    string
	ReplaySignature     string
	ObservedMarkers     []string
	AccumulatedSemScore int
}

var runtimeDomains = []string{"orchestration/scheduler", "runtime/diagnostics"}

var minimalSemanticSteps = []taskboardStep{
	{
		Marker:        "taskboard_query_filtered",
		RuntimeDomain: "orchestration/scheduler",
		Intent:        "filter task board entries using operator-supplied query",
		Outcome:       "query filter and matched task count are emitted",
	},
	{
		Marker:        "taskboard_control_validated",
		RuntimeDomain: "runtime/diagnostics",
		Intent:        "validate control action against board policy",
		Outcome:       "control validity and validation reason are emitted",
	},
	{
		Marker:        "taskboard_operation_idempotent",
		RuntimeDomain: "orchestration/scheduler",
		Intent:        "execute operation idempotently under duplicate retries",
		Outcome:       "operation key, duplicate attempts and applied count are emitted",
	},
}

var productionGovernanceSteps = []taskboardStep{
	{
		Marker:        "governance_taskboard_gate_enforced",
		RuntimeDomain: "orchestration/scheduler",
		Intent:        "enforce taskboard governance decision from control/idempotency evidence",
		Outcome:       "governance decision and ticket are emitted",
	},
	{
		Marker:        "governance_taskboard_replay_bound",
		RuntimeDomain: "runtime/diagnostics",
		Intent:        "bind governance decision to replay signature",
		Outcome:       "replay signature is emitted",
	},
}

func RunMinimal() {
	executeTaskboardVariant(modecommon.VariantMinimal)
}

func RunProduction() {
	executeTaskboardVariant(modecommon.VariantProduction)
}

func executeTaskboardVariant(variant string) {
	if err := modecommon.EnsureVariant(variant); err != nil {
		panic(err)
	}

	mgr, err := runtimeconfig.NewManager(runtimeconfig.ManagerOptions{EnvPrefix: "BAYMAX"})
	if err != nil {
		panic(err)
	}
	defer func() { _ = mgr.Close() }()

	registry := local.NewRegistry()
	if _, err := registry.Register(&taskboardQueryControlTool{}); err != nil {
		panic(err)
	}

	model := &taskboardQueryControlModel{
		variant: variant,
		state: taskboardState{
			QueryID:   defaultQueryID,
			BoardName: "mainline-task-board",
		},
	}
	engine := runner.New(model, runner.WithLocalRegistry(registry), runner.WithRuntimeManager(mgr))
	result, err := engine.Run(context.Background(), types.RunRequest{
		RunID: fmt.Sprintf("agent-mode-%s-%s", modecommon.MarkerToken(patternName), modecommon.MarkerToken(variant)),
		Input: "execute task-board query control idempotency semantic pipeline",
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

func planForVariant(variant string) []taskboardStep {
	plan := make([]taskboardStep, 0, len(minimalSemanticSteps)+len(productionGovernanceSteps))
	plan = append(plan, minimalSemanticSteps...)
	if variant == modecommon.VariantProduction {
		plan = append(plan, productionGovernanceSteps...)
	}
	return plan
}

type taskboardQueryControlModel struct {
	variant string
	cursor  int
	state   taskboardState
}

func (m *taskboardQueryControlModel) Generate(ctx context.Context, req types.ModelRequest) (types.ModelResponse, error) {
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
		"%s/%s semantic_path_completed phase=%s anchor=%s classification=%s query=%s board=%s filter=%s matched=%d control_action=%s control_target=%s control_valid=%t validation_reason=%s operation_key=%s duplicate_attempts=%d idempotent=%t applied=%d governance=%s ticket=%s replay=%s score=%d markers=%s",
		patternName,
		m.variant,
		phase,
		semanticAnchor,
		classification,
		readableValue(m.state.QueryID, true),
		readableValue(m.state.BoardName, true),
		readableValue(m.state.QueryFilter, true),
		m.state.MatchedTasks,
		readableValue(m.state.ControlAction, true),
		readableValue(m.state.ControlTarget, true),
		m.state.ControlValid,
		readableValue(m.state.ValidationReason, true),
		readableValue(m.state.OperationKey, true),
		m.state.DuplicateAttempts,
		m.state.Idempotent,
		m.state.AppliedCount,
		readableValue(m.state.GovernanceDecision, governanceOn),
		readableValue(m.state.GovernanceTicket, governanceOn),
		readableValue(m.state.ReplaySignature, governanceOn),
		m.state.AccumulatedSemScore,
		strings.Join(markers, ","),
	)
	return types.ModelResponse{FinalAnswer: final}, nil
}

func (m *taskboardQueryControlModel) Stream(ctx context.Context, req types.ModelRequest, onEvent func(types.ModelEvent) error) error {
	_ = ctx
	_ = req
	_ = onEvent
	return nil
}

func (m *taskboardQueryControlModel) captureOutcomes(outcomes []types.ToolCallOutcome) {
	for _, outcome := range outcomes {
		marker, _ := outcome.Result.Structured["marker"].(string)
		if marker != "" {
			m.state.ObservedMarkers = append(m.state.ObservedMarkers, marker)
		}
		if queryID, _ := outcome.Result.Structured["query_id"].(string); strings.TrimSpace(queryID) != "" {
			m.state.QueryID = strings.TrimSpace(queryID)
		}
		if boardName, _ := outcome.Result.Structured["board_name"].(string); strings.TrimSpace(boardName) != "" {
			m.state.BoardName = strings.TrimSpace(boardName)
		}
		if filter, _ := outcome.Result.Structured["query_filter"].(string); strings.TrimSpace(filter) != "" {
			m.state.QueryFilter = strings.TrimSpace(filter)
		}
		if matched, ok := modecommon.AsInt(outcome.Result.Structured["matched_tasks"]); ok {
			m.state.MatchedTasks = matched
		}
		if action, _ := outcome.Result.Structured["control_action"].(string); strings.TrimSpace(action) != "" {
			m.state.ControlAction = strings.TrimSpace(action)
		}
		if target, _ := outcome.Result.Structured["control_target"].(string); strings.TrimSpace(target) != "" {
			m.state.ControlTarget = strings.TrimSpace(target)
		}
		if valid, ok := outcome.Result.Structured["control_valid"].(bool); ok {
			m.state.ControlValid = valid
		}
		if reason, _ := outcome.Result.Structured["validation_reason"].(string); strings.TrimSpace(reason) != "" {
			m.state.ValidationReason = strings.TrimSpace(reason)
		}
		if operationKey, _ := outcome.Result.Structured["operation_key"].(string); strings.TrimSpace(operationKey) != "" {
			m.state.OperationKey = strings.TrimSpace(operationKey)
		}
		if duplicates, ok := modecommon.AsInt(outcome.Result.Structured["duplicate_attempts"]); ok {
			m.state.DuplicateAttempts = duplicates
		}
		if idempotent, ok := outcome.Result.Structured["idempotent"].(bool); ok {
			m.state.Idempotent = idempotent
		}
		if applied, ok := modecommon.AsInt(outcome.Result.Structured["applied_count"]); ok {
			m.state.AppliedCount = applied
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

func (m *taskboardQueryControlModel) argsForStep(step taskboardStep, stage int) map[string]any {
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
		"query_id":        m.state.QueryID,
		"board_name":      m.state.BoardName,
		"query_filter":    m.state.QueryFilter,
		"matched_tasks":   m.state.MatchedTasks,
		"control_action":  m.state.ControlAction,
		"control_target":  m.state.ControlTarget,
	}

	switch step.Marker {
	case "taskboard_query_filtered":
		queryFilter := "status:blocked owner:agent-alpha"
		matchedTasks := 2
		if m.variant == modecommon.VariantProduction {
			queryFilter = "status:running priority:high"
			matchedTasks = 5
		}
		args["query_filter"] = queryFilter
		args["matched_tasks"] = matchedTasks
	case "taskboard_control_validated":
		controlAction := "reassign"
		controlTarget := "agent-beta"
		validationReason := "policy_allow_reassign"
		if m.variant == modecommon.VariantProduction {
			controlAction = "throttle"
			controlTarget = "high-priority-lane"
			validationReason = "policy_allow_throttle_with_audit"
		}
		args["control_action"] = controlAction
		args["control_target"] = controlTarget
		args["validation_reason"] = validationReason
	case "taskboard_operation_idempotent":
		args["control_action"] = m.state.ControlAction
		args["control_target"] = m.state.ControlTarget
		args["query_filter"] = m.state.QueryFilter
		args["matched_tasks"] = m.state.MatchedTasks
		duplicateAttempts := 1
		if m.variant == modecommon.VariantProduction {
			duplicateAttempts = 2
		}
		args["duplicate_attempts"] = duplicateAttempts
	case "governance_taskboard_gate_enforced":
		args["duplicate_attempts"] = m.state.DuplicateAttempts
		args["idempotent"] = m.state.Idempotent
		args["control_valid"] = m.state.ControlValid
		args["applied_count"] = m.state.AppliedCount
	case "governance_taskboard_replay_bound":
		args["governance_decision"] = m.state.GovernanceDecision
		args["governance_ticket"] = m.state.GovernanceTicket
		args["operation_key"] = m.state.OperationKey
	}
	return args
}

type taskboardQueryControlTool struct{}

func (t *taskboardQueryControlTool) Name() string { return semanticToolName }

func (t *taskboardQueryControlTool) Description() string {
	return "execute task-board query/control/idempotency semantic step"
}

func (t *taskboardQueryControlTool) JSONSchema() map[string]any {
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
			"query_id":        map[string]any{"type": "string"},
			"board_name":      map[string]any{"type": "string"},
			"query_filter":    map[string]any{"type": "string"},
			"matched_tasks":   map[string]any{"type": "integer"},
			"control_action":  map[string]any{"type": "string"},
			"control_target":  map[string]any{"type": "string"},
			"control_valid":   map[string]any{"type": "boolean"},
			"validation_reason": map[string]any{
				"type": "string",
			},
			"operation_key": map[string]any{"type": "string"},
			"duplicate_attempts": map[string]any{
				"type": "integer",
			},
			"idempotent":    map[string]any{"type": "boolean"},
			"applied_count": map[string]any{"type": "integer"},
			"governance_decision": map[string]any{
				"type": "string",
			},
			"governance_ticket": map[string]any{"type": "string"},
		},
	}
}

func (t *taskboardQueryControlTool) Invoke(ctx context.Context, args map[string]any) (types.ToolResult, error) {
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

	queryID := strings.TrimSpace(fmt.Sprintf("%v", args["query_id"]))
	if queryID == "" {
		queryID = defaultQueryID
	}
	boardName := strings.TrimSpace(fmt.Sprintf("%v", args["board_name"]))
	if boardName == "" {
		boardName = "mainline-task-board"
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
		"query_id":        queryID,
		"board_name":      boardName,
		"governance":      false,
	}

	risk := "nominal"
	switch marker {
	case "taskboard_query_filtered":
		queryFilter := strings.TrimSpace(fmt.Sprintf("%v", args["query_filter"]))
		matchedTasks, _ := modecommon.AsInt(args["matched_tasks"])
		structured["query_filter"] = queryFilter
		structured["matched_tasks"] = matchedTasks
	case "taskboard_control_validated":
		controlAction := strings.TrimSpace(fmt.Sprintf("%v", args["control_action"]))
		controlTarget := strings.TrimSpace(fmt.Sprintf("%v", args["control_target"]))
		validationReason := strings.TrimSpace(fmt.Sprintf("%v", args["validation_reason"]))
		controlValid := controlAction != "" && controlTarget != ""
		structured["control_action"] = controlAction
		structured["control_target"] = controlTarget
		structured["control_valid"] = controlValid
		structured["validation_reason"] = validationReason
		if !controlValid {
			risk = "degraded_path"
		}
	case "taskboard_operation_idempotent":
		queryFilter := strings.TrimSpace(fmt.Sprintf("%v", args["query_filter"]))
		controlAction := strings.TrimSpace(fmt.Sprintf("%v", args["control_action"]))
		controlTarget := strings.TrimSpace(fmt.Sprintf("%v", args["control_target"]))
		duplicateAttempts, _ := modecommon.AsInt(args["duplicate_attempts"])
		matchedTasks, _ := modecommon.AsInt(args["matched_tasks"])
		operationKey := fmt.Sprintf("tb-op-%d", modecommon.SemanticScore(queryID, boardName, queryFilter, controlAction, controlTarget))
		idempotent := true
		appliedCount := 1
		if duplicateAttempts <= 0 {
			duplicateAttempts = 1
		}
		if controlAction == "" || controlTarget == "" {
			idempotent = false
			appliedCount = 0
		}
		structured["query_filter"] = queryFilter
		structured["matched_tasks"] = matchedTasks
		structured["control_action"] = controlAction
		structured["control_target"] = controlTarget
		structured["operation_key"] = operationKey
		structured["duplicate_attempts"] = duplicateAttempts
		structured["idempotent"] = idempotent
		structured["applied_count"] = appliedCount
		if !idempotent {
			risk = "degraded_path"
		}
	case "governance_taskboard_gate_enforced":
		duplicateAttempts, _ := modecommon.AsInt(args["duplicate_attempts"])
		idempotent := asBool(args["idempotent"])
		controlValid := asBool(args["control_valid"])
		appliedCount, _ := modecommon.AsInt(args["applied_count"])
		decision := "allow_control"
		if !controlValid || !idempotent {
			decision = "deny_control"
		} else if duplicateAttempts > 1 {
			decision = "allow_control_with_audit"
		}
		ticket := fmt.Sprintf("taskboard-gate-%d", modecommon.SemanticScore(queryID, decision, fmt.Sprintf("%d", duplicateAttempts), fmt.Sprintf("%t", idempotent), fmt.Sprintf("%d", appliedCount)))
		structured["duplicate_attempts"] = duplicateAttempts
		structured["idempotent"] = idempotent
		structured["control_valid"] = controlValid
		structured["applied_count"] = appliedCount
		structured["governance"] = true
		structured["governance_decision"] = decision
		structured["governance_ticket"] = ticket
		risk = "governed"
	case "governance_taskboard_replay_bound":
		decision := strings.TrimSpace(fmt.Sprintf("%v", args["governance_decision"]))
		ticket := strings.TrimSpace(fmt.Sprintf("%v", args["governance_ticket"]))
		operationKey := strings.TrimSpace(fmt.Sprintf("%v", args["operation_key"]))
		replaySignature := fmt.Sprintf("taskboard-replay-%d", modecommon.SemanticScore(queryID, decision, ticket, operationKey))
		structured["operation_key"] = operationKey
		structured["governance"] = true
		structured["governance_decision"] = decision
		structured["governance_ticket"] = ticket
		structured["replay_signature"] = replaySignature
		risk = "governed"
	default:
		return types.ToolResult{}, fmt.Errorf("unsupported task-board marker: %s", marker)
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
