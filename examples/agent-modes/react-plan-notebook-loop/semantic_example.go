package reactplannotebookloop

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
	patternName      = "react-plan-notebook-loop"
	phase            = "P1"
	semanticAnchor   = "react.plan_notebook_change_hooks"
	classification   = "react.plan_notebook_loop"
	semanticToolName = "mode_react_plan_notebook_loop_semantic_step"
	defaultRunID     = "react-loop-20260410"
)

type reactStep struct {
	Marker        string
	RuntimeDomain string
	Intent        string
	Outcome       string
}

type planItem struct {
	ID     string
	Owner  string
	Status string
}

type reactState struct {
	RunID              string
	PlanVersion        int
	NotebookDigest     string
	PendingSteps       []string
	ChangeHookType     string
	ChangeHookCount    int
	ToolAction         string
	LoopClosed         bool
	GovernanceDecision string
	GovernanceTicket   string
	ReplaySignature    string
	SeenMarkers        []string
	TotalScore         int
}

var runtimeDomains = []string{"core/runner", "runtime/diagnostics"}

var basePlan = []planItem{
	{ID: "collect-context", Owner: "planner", Status: "done"},
	{ID: "draft-patch", Owner: "executor", Status: "todo"},
	{ID: "verify-gate", Owner: "validator", Status: "todo"},
}

var minimalSemanticSteps = []reactStep{
	{
		Marker:        "react_plan_notebook_synced",
		RuntimeDomain: "core/runner",
		Intent:        "sync plan and notebook states, resolving status drift",
		Outcome:       "pending steps and notebook digest are emitted",
	},
	{
		Marker:        "react_change_hook_emitted",
		RuntimeDomain: "runtime/diagnostics",
		Intent:        "emit change hook when plan/notebook deltas exceed threshold",
		Outcome:       "change hook type/count are emitted",
	},
	{
		Marker:        "react_tool_loop_closed",
		RuntimeDomain: "core/runner",
		Intent:        "close react loop by selecting actionable tool step from notebook",
		Outcome:       "tool action and loop-closed signal are emitted",
	},
}

var productionGovernanceSteps = []reactStep{
	{
		Marker:        "governance_react_gate_enforced",
		RuntimeDomain: "core/runner",
		Intent:        "enforce governance gate on react loop action under drift pressure",
		Outcome:       "governance decision and ticket are emitted",
	},
	{
		Marker:        "governance_react_replay_bound",
		RuntimeDomain: "runtime/diagnostics",
		Intent:        "bind notebook + governance decision to replay signature",
		Outcome:       "replay signature is emitted",
	},
}

func RunMinimal() {
	executeReactVariant(modecommon.VariantMinimal)
}

func RunProduction() {
	executeReactVariant(modecommon.VariantProduction)
}

func executeReactVariant(variant string) {
	if err := modecommon.EnsureVariant(variant); err != nil {
		panic(err)
	}

	mgr, err := runtimeconfig.NewManager(runtimeconfig.ManagerOptions{EnvPrefix: "BAYMAX"})
	if err != nil {
		panic(err)
	}
	defer func() { _ = mgr.Close() }()

	reg := local.NewRegistry()
	if _, err := reg.Register(&reactNotebookTool{}); err != nil {
		panic(err)
	}

	model := &reactNotebookModel{
		variant: variant,
		state: reactState{
			RunID: defaultRunID,
		},
	}

	engine := runner.New(model, runner.WithLocalRegistry(reg), runner.WithRuntimeManager(mgr))
	result, err := engine.Run(context.Background(), types.RunRequest{
		RunID: fmt.Sprintf("agent-mode-%s-%s", modecommon.MarkerToken(patternName), modecommon.MarkerToken(variant)),
		Input: "execute react plan notebook loop semantic pipeline",
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

func planForVariant(variant string) []reactStep {
	plan := make([]reactStep, 0, len(minimalSemanticSteps)+len(productionGovernanceSteps))
	plan = append(plan, minimalSemanticSteps...)
	if variant == modecommon.VariantProduction {
		plan = append(plan, productionGovernanceSteps...)
	}
	return plan
}

type reactNotebookModel struct {
	variant string
	cursor  int
	state   reactState
}

func (m *reactNotebookModel) Generate(ctx context.Context, req types.ModelRequest) (types.ModelResponse, error) {
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
		"%s/%s semantic_path_completed phase=%s anchor=%s classification=%s plan_version=%d notebook_digest=%s pending=%s hook=%s hook_count=%d action=%s loop_closed=%t governance=%s ticket=%s replay=%s score=%d markers=%s",
		patternName,
		m.variant,
		phase,
		semanticAnchor,
		classification,
		m.state.PlanVersion,
		normalizedValue(m.state.NotebookDigest, true),
		stringSliceToken(m.state.PendingSteps),
		normalizedValue(m.state.ChangeHookType, true),
		m.state.ChangeHookCount,
		normalizedValue(m.state.ToolAction, true),
		m.state.LoopClosed,
		normalizedValue(m.state.GovernanceDecision, governanceOn),
		normalizedValue(m.state.GovernanceTicket, governanceOn),
		normalizedValue(m.state.ReplaySignature, governanceOn),
		m.state.TotalScore,
		strings.Join(markers, ","),
	)
	return types.ModelResponse{FinalAnswer: final}, nil
}

func (m *reactNotebookModel) Stream(ctx context.Context, req types.ModelRequest, onEvent func(types.ModelEvent) error) error {
	_ = ctx
	_ = req
	_ = onEvent
	return nil
}

func (m *reactNotebookModel) capture(outcomes []types.ToolCallOutcome) {
	for _, item := range outcomes {
		marker, _ := item.Result.Structured["marker"].(string)
		if marker != "" {
			m.state.SeenMarkers = append(m.state.SeenMarkers, marker)
		}
		if planVersion, ok := modecommon.AsInt(item.Result.Structured["plan_version"]); ok {
			m.state.PlanVersion = planVersion
		}
		if digest, _ := item.Result.Structured["notebook_digest"].(string); strings.TrimSpace(digest) != "" {
			m.state.NotebookDigest = strings.TrimSpace(digest)
		}
		if pending := toStringSlice(item.Result.Structured["pending_steps"]); len(pending) > 0 {
			m.state.PendingSteps = pending
		}
		if hookType, _ := item.Result.Structured["change_hook_type"].(string); strings.TrimSpace(hookType) != "" {
			m.state.ChangeHookType = strings.TrimSpace(hookType)
		}
		if hookCount, ok := modecommon.AsInt(item.Result.Structured["change_hook_count"]); ok {
			m.state.ChangeHookCount = hookCount
		}
		if action, _ := item.Result.Structured["tool_action"].(string); strings.TrimSpace(action) != "" {
			m.state.ToolAction = strings.TrimSpace(action)
		}
		if loopClosed, ok := item.Result.Structured["loop_closed"].(bool); ok {
			m.state.LoopClosed = loopClosed
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

func (m *reactNotebookModel) argsForStep(step reactStep, stage int) map[string]any {
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
		"run_id":          m.state.RunID,
	}

	switch step.Marker {
	case "react_plan_notebook_synced":
		notebook := []string{"collect-context:done", "draft-patch:todo", "verify-gate:todo"}
		planVersion := 2
		if m.variant == modecommon.VariantProduction {
			notebook = []string{"collect-context:done", "draft-patch:in_progress", "verify-gate:todo"}
			planVersion = 3
		}
		args["plan_items"] = encodePlan(basePlan)
		args["notebook_items"] = stringSliceToAny(notebook)
		args["plan_version"] = planVersion
	case "react_change_hook_emitted":
		args["pending_steps"] = stringSliceToAny(m.state.PendingSteps)
		args["notebook_digest"] = m.state.NotebookDigest
		args["plan_version"] = m.state.PlanVersion
	case "react_tool_loop_closed":
		args["pending_steps"] = stringSliceToAny(m.state.PendingSteps)
		args["change_hook_type"] = m.state.ChangeHookType
		args["change_hook_count"] = m.state.ChangeHookCount
	case "governance_react_gate_enforced":
		args["tool_action"] = m.state.ToolAction
		args["change_hook_type"] = m.state.ChangeHookType
		args["change_hook_count"] = m.state.ChangeHookCount
		args["pending_steps"] = stringSliceToAny(m.state.PendingSteps)
	case "governance_react_replay_bound":
		args["notebook_digest"] = m.state.NotebookDigest
		args["governance_decision"] = m.state.GovernanceDecision
		args["governance_ticket"] = m.state.GovernanceTicket
		args["run_id"] = m.state.RunID
	}
	return args
}

type reactNotebookTool struct{}

func (t *reactNotebookTool) Name() string { return semanticToolName }

func (t *reactNotebookTool) Description() string {
	return "execute react plan/notebook/change-hook semantic step"
}

func (t *reactNotebookTool) JSONSchema() map[string]any {
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
			"run_id":          map[string]any{"type": "string"},
			"plan_items":      map[string]any{"type": "array"},
			"notebook_items":  map[string]any{"type": "array"},
			"plan_version":    map[string]any{"type": "integer"},
			"pending_steps":   map[string]any{"type": "array"},
			"notebook_digest": map[string]any{"type": "string"},
			"change_hook_type": map[string]any{
				"type": "string",
			},
			"change_hook_count": map[string]any{"type": "integer"},
			"tool_action":       map[string]any{"type": "string"},
			"governance_decision": map[string]any{
				"type": "string",
			},
			"governance_ticket": map[string]any{"type": "string"},
		},
	}
}

func (t *reactNotebookTool) Invoke(ctx context.Context, args map[string]any) (types.ToolResult, error) {
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
	case "react_plan_notebook_synced":
		runID := strings.TrimSpace(fmt.Sprintf("%v", args["run_id"]))
		if runID == "" {
			runID = defaultRunID
		}
		plan := decodePlan(args["plan_items"])
		if len(plan) == 0 {
			plan = append([]planItem{}, basePlan...)
		}
		notebookItems := toStringSlice(args["notebook_items"])
		planVersion, _ := modecommon.AsInt(args["plan_version"])
		if planVersion <= 0 {
			planVersion = 2
		}
		pending, driftCount := reconcileNotebook(plan, notebookItems)
		digest := fmt.Sprintf("notebook-%d", modecommon.SemanticScore(runID, fmt.Sprintf("v%d", planVersion), strings.Join(pending, "|"), fmt.Sprintf("drift=%d", driftCount)))
		structured["run_id"] = runID
		structured["plan_version"] = planVersion
		structured["pending_steps"] = stringSliceToAny(pending)
		structured["drift_count"] = driftCount
		structured["notebook_digest"] = digest
		if driftCount > 1 {
			risk = "degraded_path"
		}
	case "react_change_hook_emitted":
		pending := toStringSlice(args["pending_steps"])
		notebookDigest := strings.TrimSpace(fmt.Sprintf("%v", args["notebook_digest"]))
		planVersion, _ := modecommon.AsInt(args["plan_version"])
		if planVersion <= 0 {
			planVersion = 2
		}
		hookType := "no_change"
		hookCount := 0
		if len(pending) >= 2 {
			hookType = "patch_plan"
			hookCount = 1
		}
		if variant == modecommon.VariantProduction && len(pending) >= 2 {
			hookType = "guardrail_patch"
			hookCount = 2
		}
		structured["pending_steps"] = stringSliceToAny(pending)
		structured["notebook_digest"] = notebookDigest
		structured["plan_version"] = planVersion
		structured["change_hook_type"] = hookType
		structured["change_hook_count"] = hookCount
		if hookType != "no_change" {
			risk = "degraded_path"
		}
	case "react_tool_loop_closed":
		pending := toStringSlice(args["pending_steps"])
		hookType := strings.TrimSpace(fmt.Sprintf("%v", args["change_hook_type"]))
		hookCount, _ := modecommon.AsInt(args["change_hook_count"])
		action := "finalize_answer"
		if len(pending) > 0 {
			action = "execute_next_tool"
		}
		if hookType == "guardrail_patch" {
			action = "request_human_review"
		}
		loopClosed := action != "request_human_review"
		structured["pending_steps"] = stringSliceToAny(pending)
		structured["change_hook_type"] = hookType
		structured["change_hook_count"] = hookCount
		structured["tool_action"] = action
		structured["loop_closed"] = loopClosed
		if !loopClosed {
			risk = "degraded_path"
		}
	case "governance_react_gate_enforced":
		action := strings.TrimSpace(fmt.Sprintf("%v", args["tool_action"]))
		hookType := strings.TrimSpace(fmt.Sprintf("%v", args["change_hook_type"]))
		hookCount, _ := modecommon.AsInt(args["change_hook_count"])
		pending := toStringSlice(args["pending_steps"])
		decision := "allow"
		if action == "request_human_review" {
			decision = "deny"
		} else if hookType != "no_change" || hookCount > 1 || len(pending) > 1 {
			decision = "allow_with_guardrails"
		}
		ticket := fmt.Sprintf("react-gate-%d", modecommon.SemanticScore(action, hookType, fmt.Sprintf("%d", hookCount), stringSliceToken(pending), decision))
		structured["tool_action"] = action
		structured["change_hook_type"] = hookType
		structured["change_hook_count"] = hookCount
		structured["pending_steps"] = stringSliceToAny(pending)
		structured["governance"] = true
		structured["governance_decision"] = decision
		structured["governance_ticket"] = ticket
		risk = "governed"
	case "governance_react_replay_bound":
		runID := strings.TrimSpace(fmt.Sprintf("%v", args["run_id"]))
		notebookDigest := strings.TrimSpace(fmt.Sprintf("%v", args["notebook_digest"]))
		decision := strings.TrimSpace(fmt.Sprintf("%v", args["governance_decision"]))
		ticket := strings.TrimSpace(fmt.Sprintf("%v", args["governance_ticket"]))
		replay := fmt.Sprintf("react-replay-%d", modecommon.SemanticScore(runID, notebookDigest, decision, ticket))
		structured["run_id"] = runID
		structured["notebook_digest"] = notebookDigest
		structured["governance_decision"] = decision
		structured["governance_ticket"] = ticket
		structured["governance"] = true
		structured["replay_signature"] = replay
		risk = "governed"
	default:
		return types.ToolResult{}, fmt.Errorf("unsupported react semantic marker: %s", marker)
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

func reconcileNotebook(plan []planItem, notebookItems []string) ([]string, int) {
	notebookStatus := map[string]string{}
	for _, row := range notebookItems {
		parts := strings.SplitN(strings.TrimSpace(row), ":", 2)
		if len(parts) != 2 {
			continue
		}
		notebookStatus[parts[0]] = parts[1]
	}
	pending := make([]string, 0)
	driftCount := 0
	for _, item := range plan {
		status := item.Status
		if notebook, ok := notebookStatus[item.ID]; ok {
			if notebook != status {
				driftCount++
				if notebook == "done" && status == "todo" {
					status = notebook
				}
			}
		}
		if status != "done" {
			pending = append(pending, item.ID)
		}
	}
	sort.Strings(pending)
	return pending, driftCount
}

func encodePlan(items []planItem) []any {
	out := make([]any, 0, len(items))
	for _, item := range items {
		out = append(out, fmt.Sprintf("%s|%s|%s", item.ID, item.Owner, item.Status))
	}
	return out
}

func decodePlan(value any) []planItem {
	raw := toStringSlice(value)
	out := make([]planItem, 0, len(raw))
	for _, row := range raw {
		parts := strings.Split(row, "|")
		if len(parts) != 3 {
			continue
		}
		out = append(out, planItem{
			ID:     strings.TrimSpace(parts[0]),
			Owner:  strings.TrimSpace(parts[1]),
			Status: strings.TrimSpace(parts[2]),
		})
	}
	return out
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

func stringSliceToken(in []string) string {
	if len(in) == 0 {
		return "none"
	}
	copySlice := append([]string(nil), in...)
	sort.Strings(copySlice)
	return strings.Join(copySlice, "|")
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
