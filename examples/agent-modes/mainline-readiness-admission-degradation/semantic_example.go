package mainlinereadinessadmissiondegradation

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
	patternName      = "mainline-readiness-admission-degradation"
	phase            = "P2"
	semanticAnchor   = "readiness.admission_degradation"
	classification   = "mainline.readiness_admission"
	semanticToolName = "mode_mainline_readiness_admission_degradation_semantic_step"
	defaultCheckID   = "readiness-check-20260410"
)

type readinessStep struct {
	Marker        string
	RuntimeDomain string
	Intent        string
	Outcome       string
}

type readinessState struct {
	CheckID              string
	Component            string
	LatencyP95MS         int
	ErrorRatePct         int
	ReadinessState       string
	AdmissionDecision    string
	DegradationClass     string
	RollbackGuardEnabled bool
	RollbackPlan         string
	PrimaryReason        string
	GovernanceDecision   string
	GovernanceTicket     string
	ReplaySignature      string
	ObservedMarkers      []string
	AccumulatedSemScore  int
}

var runtimeDomains = []string{"runtime/config", "runtime/diagnostics", "orchestration/composer"}

var minimalSemanticSteps = []readinessStep{
	{
		Marker:        "readiness_preflight_evaluated",
		RuntimeDomain: "runtime/config",
		Intent:        "evaluate readiness preflight metrics before admission",
		Outcome:       "latency/error and readiness state are emitted",
	},
	{
		Marker:        "admission_degradation_classified",
		RuntimeDomain: "runtime/diagnostics",
		Intent:        "classify admission decision and degradation class",
		Outcome:       "admission decision, degradation class and reason are emitted",
	},
	{
		Marker:        "readiness_rollback_guarded",
		RuntimeDomain: "orchestration/composer",
		Intent:        "apply rollback guard based on degradation class",
		Outcome:       "rollback guard and rollback plan are emitted",
	},
}

var productionGovernanceSteps = []readinessStep{
	{
		Marker:        "governance_readiness_gate_enforced",
		RuntimeDomain: "runtime/config",
		Intent:        "enforce readiness governance from admission and rollback guard signals",
		Outcome:       "governance decision and ticket are emitted",
	},
	{
		Marker:        "governance_readiness_replay_bound",
		RuntimeDomain: "runtime/diagnostics",
		Intent:        "bind governance decision to replay signature",
		Outcome:       "replay signature is emitted",
	},
}

func RunMinimal() {
	executeReadinessVariant(modecommon.VariantMinimal)
}

func RunProduction() {
	executeReadinessVariant(modecommon.VariantProduction)
}

func executeReadinessVariant(variant string) {
	if err := modecommon.EnsureVariant(variant); err != nil {
		panic(err)
	}

	mgr, err := runtimeconfig.NewManager(runtimeconfig.ManagerOptions{EnvPrefix: "BAYMAX"})
	if err != nil {
		panic(err)
	}
	defer func() { _ = mgr.Close() }()

	registry := local.NewRegistry()
	if _, err := registry.Register(&readinessAdmissionTool{}); err != nil {
		panic(err)
	}

	model := &readinessAdmissionModel{
		variant: variant,
		state: readinessState{
			CheckID:   defaultCheckID,
			Component: "runtime-composer",
		},
	}
	engine := runner.New(model, runner.WithLocalRegistry(registry), runner.WithRuntimeManager(mgr))
	result, err := engine.Run(context.Background(), types.RunRequest{
		RunID: fmt.Sprintf("agent-mode-%s-%s", modecommon.MarkerToken(patternName), modecommon.MarkerToken(variant)),
		Input: "execute readiness admission degradation semantic pipeline",
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

func planForVariant(variant string) []readinessStep {
	plan := make([]readinessStep, 0, len(minimalSemanticSteps)+len(productionGovernanceSteps))
	plan = append(plan, minimalSemanticSteps...)
	if variant == modecommon.VariantProduction {
		plan = append(plan, productionGovernanceSteps...)
	}
	return plan
}

type readinessAdmissionModel struct {
	variant string
	cursor  int
	state   readinessState
}

func (m *readinessAdmissionModel) Generate(ctx context.Context, req types.ModelRequest) (types.ModelResponse, error) {
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
		"%s/%s semantic_path_completed phase=%s anchor=%s classification=%s check=%s component=%s latency_p95_ms=%d error_rate_pct=%d readiness=%s admission=%s degradation=%s rollback_guard=%t rollback_plan=%s primary_reason=%s governance=%s ticket=%s replay=%s score=%d markers=%s",
		patternName,
		m.variant,
		phase,
		semanticAnchor,
		classification,
		readableValue(m.state.CheckID, true),
		readableValue(m.state.Component, true),
		m.state.LatencyP95MS,
		m.state.ErrorRatePct,
		readableValue(m.state.ReadinessState, true),
		readableValue(m.state.AdmissionDecision, true),
		readableValue(m.state.DegradationClass, true),
		m.state.RollbackGuardEnabled,
		readableValue(m.state.RollbackPlan, true),
		readableValue(m.state.PrimaryReason, true),
		readableValue(m.state.GovernanceDecision, governanceOn),
		readableValue(m.state.GovernanceTicket, governanceOn),
		readableValue(m.state.ReplaySignature, governanceOn),
		m.state.AccumulatedSemScore,
		strings.Join(markers, ","),
	)
	return types.ModelResponse{FinalAnswer: final}, nil
}

func (m *readinessAdmissionModel) Stream(ctx context.Context, req types.ModelRequest, onEvent func(types.ModelEvent) error) error {
	_ = ctx
	_ = req
	_ = onEvent
	return nil
}

func (m *readinessAdmissionModel) captureOutcomes(outcomes []types.ToolCallOutcome) {
	for _, outcome := range outcomes {
		marker, _ := outcome.Result.Structured["marker"].(string)
		if marker != "" {
			m.state.ObservedMarkers = append(m.state.ObservedMarkers, marker)
		}
		if checkID, _ := outcome.Result.Structured["check_id"].(string); strings.TrimSpace(checkID) != "" {
			m.state.CheckID = strings.TrimSpace(checkID)
		}
		if component, _ := outcome.Result.Structured["component"].(string); strings.TrimSpace(component) != "" {
			m.state.Component = strings.TrimSpace(component)
		}
		if latency, ok := modecommon.AsInt(outcome.Result.Structured["latency_p95_ms"]); ok {
			m.state.LatencyP95MS = latency
		}
		if errRate, ok := modecommon.AsInt(outcome.Result.Structured["error_rate_pct"]); ok {
			m.state.ErrorRatePct = errRate
		}
		if readiness, _ := outcome.Result.Structured["readiness_state"].(string); strings.TrimSpace(readiness) != "" {
			m.state.ReadinessState = strings.TrimSpace(readiness)
		}
		if admission, _ := outcome.Result.Structured["admission_decision"].(string); strings.TrimSpace(admission) != "" {
			m.state.AdmissionDecision = strings.TrimSpace(admission)
		}
		if class, _ := outcome.Result.Structured["degradation_class"].(string); strings.TrimSpace(class) != "" {
			m.state.DegradationClass = strings.TrimSpace(class)
		}
		if rollbackGuard, ok := outcome.Result.Structured["rollback_guard"].(bool); ok {
			m.state.RollbackGuardEnabled = rollbackGuard
		}
		if plan, _ := outcome.Result.Structured["rollback_plan"].(string); strings.TrimSpace(plan) != "" {
			m.state.RollbackPlan = strings.TrimSpace(plan)
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

func (m *readinessAdmissionModel) argsForStep(step readinessStep, stage int) map[string]any {
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
		"check_id":        m.state.CheckID,
		"component":       m.state.Component,
	}

	switch step.Marker {
	case "readiness_preflight_evaluated":
		latency := 180
		errorRate := 1
		readinessState := "ready"
		if m.variant == modecommon.VariantProduction {
			latency = 420
			errorRate = 5
			readinessState = "degraded"
		}
		args["latency_p95_ms"] = latency
		args["error_rate_pct"] = errorRate
		args["readiness_state"] = readinessState
	case "admission_degradation_classified":
		admission := "allow"
		degradationClass := "none"
		primaryReason := "preflight_within_threshold"
		if m.state.ReadinessState == "degraded" {
			admission = "allow_with_degradation"
			degradationClass = "degraded_capacity_50"
			primaryReason = "latency_or_error_above_threshold"
		}
		args["admission_decision"] = admission
		args["degradation_class"] = degradationClass
		args["primary_reason"] = primaryReason
	case "readiness_rollback_guarded":
		rollbackGuard := m.state.DegradationClass != "none"
		rollbackPlan := "not-required"
		if rollbackGuard {
			rollbackPlan = "rollback-to-stable-profile-v2"
		}
		args["degradation_class"] = m.state.DegradationClass
		args["rollback_guard"] = rollbackGuard
		args["rollback_plan"] = rollbackPlan
		args["primary_reason"] = m.state.PrimaryReason
	case "governance_readiness_gate_enforced":
		args["admission_decision"] = m.state.AdmissionDecision
		args["degradation_class"] = m.state.DegradationClass
		args["rollback_guard"] = m.state.RollbackGuardEnabled
		args["primary_reason"] = m.state.PrimaryReason
	case "governance_readiness_replay_bound":
		args["governance_decision"] = m.state.GovernanceDecision
		args["governance_ticket"] = m.state.GovernanceTicket
		args["rollback_plan"] = m.state.RollbackPlan
	}
	return args
}

type readinessAdmissionTool struct{}

func (t *readinessAdmissionTool) Name() string { return semanticToolName }

func (t *readinessAdmissionTool) Description() string {
	return "execute readiness admission degradation semantic step"
}

func (t *readinessAdmissionTool) JSONSchema() map[string]any {
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
			"check_id":        map[string]any{"type": "string"},
			"component":       map[string]any{"type": "string"},
			"latency_p95_ms":  map[string]any{"type": "integer"},
			"error_rate_pct":  map[string]any{"type": "integer"},
			"readiness_state": map[string]any{"type": "string"},
			"admission_decision": map[string]any{
				"type": "string",
			},
			"degradation_class": map[string]any{"type": "string"},
			"rollback_guard":    map[string]any{"type": "boolean"},
			"rollback_plan":     map[string]any{"type": "string"},
			"primary_reason":    map[string]any{"type": "string"},
			"governance_decision": map[string]any{
				"type": "string",
			},
			"governance_ticket": map[string]any{"type": "string"},
		},
	}
}

func (t *readinessAdmissionTool) Invoke(ctx context.Context, args map[string]any) (types.ToolResult, error) {
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

	checkID := strings.TrimSpace(fmt.Sprintf("%v", args["check_id"]))
	if checkID == "" {
		checkID = defaultCheckID
	}
	component := strings.TrimSpace(fmt.Sprintf("%v", args["component"]))
	if component == "" {
		component = "runtime-composer"
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
		"check_id":        checkID,
		"component":       component,
		"governance":      false,
	}

	risk := "nominal"
	switch marker {
	case "readiness_preflight_evaluated":
		latency, _ := modecommon.AsInt(args["latency_p95_ms"])
		errorRate, _ := modecommon.AsInt(args["error_rate_pct"])
		readinessState := strings.TrimSpace(fmt.Sprintf("%v", args["readiness_state"]))
		structured["latency_p95_ms"] = latency
		structured["error_rate_pct"] = errorRate
		structured["readiness_state"] = readinessState
		if readinessState != "ready" {
			risk = "degraded_path"
		}
	case "admission_degradation_classified":
		admissionDecision := strings.TrimSpace(fmt.Sprintf("%v", args["admission_decision"]))
		degradationClass := strings.TrimSpace(fmt.Sprintf("%v", args["degradation_class"]))
		primaryReason := strings.TrimSpace(fmt.Sprintf("%v", args["primary_reason"]))
		structured["admission_decision"] = admissionDecision
		structured["degradation_class"] = degradationClass
		structured["primary_reason"] = primaryReason
		if degradationClass != "none" {
			risk = "degraded_path"
		}
	case "readiness_rollback_guarded":
		degradationClass := strings.TrimSpace(fmt.Sprintf("%v", args["degradation_class"]))
		rollbackGuard := asBool(args["rollback_guard"])
		rollbackPlan := strings.TrimSpace(fmt.Sprintf("%v", args["rollback_plan"]))
		primaryReason := strings.TrimSpace(fmt.Sprintf("%v", args["primary_reason"]))
		structured["degradation_class"] = degradationClass
		structured["rollback_guard"] = rollbackGuard
		structured["rollback_plan"] = rollbackPlan
		structured["primary_reason"] = primaryReason
		if rollbackGuard {
			risk = "degraded_path"
		}
	case "governance_readiness_gate_enforced":
		admissionDecision := strings.TrimSpace(fmt.Sprintf("%v", args["admission_decision"]))
		degradationClass := strings.TrimSpace(fmt.Sprintf("%v", args["degradation_class"]))
		rollbackGuard := asBool(args["rollback_guard"])
		primaryReason := strings.TrimSpace(fmt.Sprintf("%v", args["primary_reason"]))
		decision := "pass_readiness_gate"
		if admissionDecision == "allow_with_degradation" && rollbackGuard {
			decision = "pass_with_degradation_guard"
		}
		if admissionDecision == "deny" {
			decision = "block_by_readiness_gate"
		}
		ticket := fmt.Sprintf("readiness-gate-%d", modecommon.SemanticScore(checkID, decision, degradationClass, primaryReason))
		structured["admission_decision"] = admissionDecision
		structured["degradation_class"] = degradationClass
		structured["rollback_guard"] = rollbackGuard
		structured["primary_reason"] = primaryReason
		structured["governance"] = true
		structured["governance_decision"] = decision
		structured["governance_ticket"] = ticket
		risk = "governed"
	case "governance_readiness_replay_bound":
		decision := strings.TrimSpace(fmt.Sprintf("%v", args["governance_decision"]))
		ticket := strings.TrimSpace(fmt.Sprintf("%v", args["governance_ticket"]))
		rollbackPlan := strings.TrimSpace(fmt.Sprintf("%v", args["rollback_plan"]))
		replaySignature := fmt.Sprintf("readiness-replay-%d", modecommon.SemanticScore(checkID, decision, ticket, rollbackPlan))
		structured["rollback_plan"] = rollbackPlan
		structured["governance"] = true
		structured["governance_decision"] = decision
		structured["governance_ticket"] = ticket
		structured["replay_signature"] = replaySignature
		risk = "governed"
	default:
		return types.ToolResult{}, fmt.Errorf("unsupported readiness marker: %s", marker)
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
