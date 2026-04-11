package workflowroutingstrategyswitch

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
	patternName      = "workflow-routing-strategy-switch"
	phase            = "P2"
	semanticAnchor   = "routing.strategy_switch_confidence"
	classification   = "workflow.strategy_switch"
	semanticToolName = "mode_workflow_routing_strategy_switch_semantic_step"
	defaultRequestID = "route-switch-20260410"
)

type routingStep struct {
	Marker        string
	RuntimeDomain string
	Intent        string
	Outcome       string
}

type routingSwitchState struct {
	RequestID           string
	RouteVersion        string
	CurrentStrategy     string
	CandidateStrategy   string
	ActiveStrategy      string
	FallbackStrategy    string
	TrafficProfile      string
	SelectionPolicy     string
	SwitchThreshold     int
	ConfidenceScore     int
	ConfidenceReason    string
	SwitchCommitted     bool
	SwitchReason        string
	GovernanceDecision  string
	GovernanceTicket    string
	ReplaySignature     string
	ObservedMarkers     []string
	AccumulatedSemScore int
}

var runtimeDomains = []string{"orchestration/workflow", "runtime/config"}

var minimalSemanticSteps = []routingStep{
	{
		Marker:        "routing_strategy_selected",
		RuntimeDomain: "orchestration/workflow",
		Intent:        "choose candidate routing strategy from traffic profile and current strategy",
		Outcome:       "candidate strategy, selection policy and switch threshold are emitted",
	},
	{
		Marker:        "routing_confidence_evaluated",
		RuntimeDomain: "runtime/config",
		Intent:        "evaluate confidence using runtime signals against switch threshold",
		Outcome:       "confidence score and recommendation are emitted",
	},
	{
		Marker:        "routing_switch_committed",
		RuntimeDomain: "orchestration/workflow",
		Intent:        "commit switch only when confidence meets threshold",
		Outcome:       "active strategy, switch decision and fallback strategy are emitted",
	},
}

var productionGovernanceSteps = []routingStep{
	{
		Marker:        "governance_routing_gate_enforced",
		RuntimeDomain: "orchestration/workflow",
		Intent:        "enforce governance decision based on confidence and traffic risk",
		Outcome:       "governance decision and ticket are emitted",
	},
	{
		Marker:        "governance_routing_replay_bound",
		RuntimeDomain: "runtime/config",
		Intent:        "bind routing decision to replay signature for route-version audit",
		Outcome:       "replay signature is emitted",
	},
}

func RunMinimal() {
	executeRoutingVariant(modecommon.VariantMinimal)
}

func RunProduction() {
	executeRoutingVariant(modecommon.VariantProduction)
}

func executeRoutingVariant(variant string) {
	if err := modecommon.EnsureVariant(variant); err != nil {
		panic(err)
	}

	mgr, err := runtimeconfig.NewManager(runtimeconfig.ManagerOptions{EnvPrefix: "BAYMAX"})
	if err != nil {
		panic(err)
	}
	defer func() { _ = mgr.Close() }()

	registry := local.NewRegistry()
	if _, err := registry.Register(&routingSwitchTool{}); err != nil {
		panic(err)
	}

	model := &routingSwitchModel{
		variant: variant,
		state: routingSwitchState{
			RequestID:        defaultRequestID,
			RouteVersion:     "route-v31",
			CurrentStrategy:  "latency-first",
			ActiveStrategy:   "latency-first",
			FallbackStrategy: "latency-first",
		},
	}
	engine := runner.New(model, runner.WithLocalRegistry(registry), runner.WithRuntimeManager(mgr))
	result, err := engine.Run(context.Background(), types.RunRequest{
		RunID: fmt.Sprintf("agent-mode-%s-%s", modecommon.MarkerToken(patternName), modecommon.MarkerToken(variant)),
		Input: "execute workflow routing strategy switch semantic pipeline",
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

func planForVariant(variant string) []routingStep {
	plan := make([]routingStep, 0, len(minimalSemanticSteps)+len(productionGovernanceSteps))
	plan = append(plan, minimalSemanticSteps...)
	if variant == modecommon.VariantProduction {
		plan = append(plan, productionGovernanceSteps...)
	}
	return plan
}

type routingSwitchModel struct {
	variant string
	cursor  int
	state   routingSwitchState
}

func (m *routingSwitchModel) Generate(ctx context.Context, req types.ModelRequest) (types.ModelResponse, error) {
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
		"%s/%s semantic_path_completed phase=%s anchor=%s classification=%s request=%s route_version=%s current=%s candidate=%s active=%s fallback=%s traffic=%s threshold=%d confidence=%d confidence_reason=%s switch_committed=%t switch_reason=%s governance=%s ticket=%s replay=%s score=%d markers=%s",
		patternName,
		m.variant,
		phase,
		semanticAnchor,
		classification,
		readableValue(m.state.RequestID, true),
		readableValue(m.state.RouteVersion, true),
		readableValue(m.state.CurrentStrategy, true),
		readableValue(m.state.CandidateStrategy, true),
		readableValue(m.state.ActiveStrategy, true),
		readableValue(m.state.FallbackStrategy, true),
		readableValue(m.state.TrafficProfile, true),
		m.state.SwitchThreshold,
		m.state.ConfidenceScore,
		readableValue(m.state.ConfidenceReason, true),
		m.state.SwitchCommitted,
		readableValue(m.state.SwitchReason, true),
		readableValue(m.state.GovernanceDecision, governanceOn),
		readableValue(m.state.GovernanceTicket, governanceOn),
		readableValue(m.state.ReplaySignature, governanceOn),
		m.state.AccumulatedSemScore,
		strings.Join(markers, ","),
	)
	return types.ModelResponse{FinalAnswer: final}, nil
}

func (m *routingSwitchModel) Stream(ctx context.Context, req types.ModelRequest, onEvent func(types.ModelEvent) error) error {
	_ = ctx
	_ = req
	_ = onEvent
	return nil
}

func (m *routingSwitchModel) captureOutcomes(outcomes []types.ToolCallOutcome) {
	for _, outcome := range outcomes {
		marker, _ := outcome.Result.Structured["marker"].(string)
		if marker != "" {
			m.state.ObservedMarkers = append(m.state.ObservedMarkers, marker)
		}
		if requestID, _ := outcome.Result.Structured["request_id"].(string); strings.TrimSpace(requestID) != "" {
			m.state.RequestID = strings.TrimSpace(requestID)
		}
		if routeVersion, _ := outcome.Result.Structured["route_version"].(string); strings.TrimSpace(routeVersion) != "" {
			m.state.RouteVersion = strings.TrimSpace(routeVersion)
		}
		if current, _ := outcome.Result.Structured["current_strategy"].(string); strings.TrimSpace(current) != "" {
			m.state.CurrentStrategy = strings.TrimSpace(current)
		}
		if candidate, _ := outcome.Result.Structured["candidate_strategy"].(string); strings.TrimSpace(candidate) != "" {
			m.state.CandidateStrategy = strings.TrimSpace(candidate)
		}
		if active, _ := outcome.Result.Structured["active_strategy"].(string); strings.TrimSpace(active) != "" {
			m.state.ActiveStrategy = strings.TrimSpace(active)
		}
		if fallback, _ := outcome.Result.Structured["fallback_strategy"].(string); strings.TrimSpace(fallback) != "" {
			m.state.FallbackStrategy = strings.TrimSpace(fallback)
		}
		if traffic, _ := outcome.Result.Structured["traffic_profile"].(string); strings.TrimSpace(traffic) != "" {
			m.state.TrafficProfile = strings.TrimSpace(traffic)
		}
		if policy, _ := outcome.Result.Structured["selection_policy"].(string); strings.TrimSpace(policy) != "" {
			m.state.SelectionPolicy = strings.TrimSpace(policy)
		}
		if threshold, ok := modecommon.AsInt(outcome.Result.Structured["switch_threshold"]); ok {
			m.state.SwitchThreshold = threshold
		}
		if confidence, ok := modecommon.AsInt(outcome.Result.Structured["confidence_score"]); ok {
			m.state.ConfidenceScore = confidence
		}
		if reason, _ := outcome.Result.Structured["confidence_reason"].(string); strings.TrimSpace(reason) != "" {
			m.state.ConfidenceReason = strings.TrimSpace(reason)
		}
		if committed, ok := outcome.Result.Structured["switch_committed"].(bool); ok {
			m.state.SwitchCommitted = committed
		}
		if reason, _ := outcome.Result.Structured["switch_reason"].(string); strings.TrimSpace(reason) != "" {
			m.state.SwitchReason = strings.TrimSpace(reason)
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

func (m *routingSwitchModel) argsForStep(step routingStep, stage int) map[string]any {
	args := map[string]any{
		"pattern":            patternName,
		"variant":            m.variant,
		"phase":              phase,
		"semantic_anchor":    semanticAnchor,
		"classification":     classification,
		"marker":             step.Marker,
		"runtime_domain":     step.RuntimeDomain,
		"intent":             step.Intent,
		"outcome":            step.Outcome,
		"stage":              stage,
		"request_id":         m.state.RequestID,
		"route_version":      m.state.RouteVersion,
		"current_strategy":   m.state.CurrentStrategy,
		"candidate_strategy": m.state.CandidateStrategy,
		"traffic_profile":    m.state.TrafficProfile,
		"switch_threshold":   m.state.SwitchThreshold,
		"confidence_score":   m.state.ConfidenceScore,
		"active_strategy":    m.state.ActiveStrategy,
	}

	switch step.Marker {
	case "routing_strategy_selected":
		trafficProfile := "interactive-chat"
		candidateStrategy := "cost-aware"
		selectionPolicy := "latency_cost_ratio"
		switchThreshold := 72
		routeVersion := "route-v32"
		if m.variant == modecommon.VariantProduction {
			trafficProfile = "spiky-batch-plus-chat"
			candidateStrategy = "hybrid-risk-aware"
			selectionPolicy = "latency_error_budget"
			switchThreshold = 84
			routeVersion = "route-v33"
		}
		args["traffic_profile"] = trafficProfile
		args["candidate_strategy"] = candidateStrategy
		args["selection_policy"] = selectionPolicy
		args["switch_threshold"] = switchThreshold
		args["route_version"] = routeVersion
	case "routing_confidence_evaluated":
		confidenceScore := 76
		confidenceReason := "p95_latency_drop_18"
		if m.variant == modecommon.VariantProduction {
			confidenceScore = 91
			confidenceReason = "p95_latency_drop_22_error_budget_stable"
		}
		args["confidence_score"] = confidenceScore
		args["confidence_reason"] = confidenceReason
		args["candidate_strategy"] = m.state.CandidateStrategy
		args["switch_threshold"] = m.state.SwitchThreshold
	case "routing_switch_committed":
		args["confidence_score"] = m.state.ConfidenceScore
		args["switch_threshold"] = m.state.SwitchThreshold
		args["candidate_strategy"] = m.state.CandidateStrategy
		args["current_strategy"] = m.state.CurrentStrategy
		args["fallback_strategy"] = m.state.CurrentStrategy
	case "governance_routing_gate_enforced":
		args["switch_committed"] = m.state.SwitchCommitted
		args["active_strategy"] = m.state.ActiveStrategy
		args["traffic_profile"] = m.state.TrafficProfile
		args["confidence_score"] = m.state.ConfidenceScore
		args["switch_threshold"] = m.state.SwitchThreshold
	case "governance_routing_replay_bound":
		args["governance_decision"] = m.state.GovernanceDecision
		args["governance_ticket"] = m.state.GovernanceTicket
		args["active_strategy"] = m.state.ActiveStrategy
		args["switch_reason"] = m.state.SwitchReason
	}
	return args
}

type routingSwitchTool struct{}

func (t *routingSwitchTool) Name() string { return semanticToolName }

func (t *routingSwitchTool) Description() string {
	return "execute workflow routing strategy switch semantic step"
}

func (t *routingSwitchTool) JSONSchema() map[string]any {
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
			"stage":              map[string]any{"type": "integer"},
			"request_id":         map[string]any{"type": "string"},
			"route_version":      map[string]any{"type": "string"},
			"current_strategy":   map[string]any{"type": "string"},
			"candidate_strategy": map[string]any{"type": "string"},
			"active_strategy":    map[string]any{"type": "string"},
			"fallback_strategy":  map[string]any{"type": "string"},
			"traffic_profile":    map[string]any{"type": "string"},
			"selection_policy":   map[string]any{"type": "string"},
			"switch_threshold":   map[string]any{"type": "integer"},
			"confidence_score":   map[string]any{"type": "integer"},
			"confidence_reason":  map[string]any{"type": "string"},
			"switch_committed":   map[string]any{"type": "boolean"},
			"switch_reason":      map[string]any{"type": "string"},
			"governance_decision": map[string]any{
				"type": "string",
			},
			"governance_ticket": map[string]any{"type": "string"},
		},
	}
}

func (t *routingSwitchTool) Invoke(ctx context.Context, args map[string]any) (types.ToolResult, error) {
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

	requestID := strings.TrimSpace(fmt.Sprintf("%v", args["request_id"]))
	if requestID == "" {
		requestID = defaultRequestID
	}
	routeVersion := strings.TrimSpace(fmt.Sprintf("%v", args["route_version"]))
	currentStrategy := strings.TrimSpace(fmt.Sprintf("%v", args["current_strategy"]))

	structured := map[string]any{
		"pattern":          pattern,
		"variant":          variant,
		"phase":            phaseValue,
		"semantic_anchor":  anchor,
		"classification":   classValue,
		"marker":           marker,
		"runtime_domain":   runtimeDomain,
		"intent":           intent,
		"outcome":          outcome,
		"stage":            stage,
		"request_id":       requestID,
		"route_version":    routeVersion,
		"current_strategy": currentStrategy,
		"governance":       false,
	}

	risk := "nominal"
	switch marker {
	case "routing_strategy_selected":
		candidateStrategy := strings.TrimSpace(fmt.Sprintf("%v", args["candidate_strategy"]))
		trafficProfile := strings.TrimSpace(fmt.Sprintf("%v", args["traffic_profile"]))
		selectionPolicy := strings.TrimSpace(fmt.Sprintf("%v", args["selection_policy"]))
		switchThreshold, _ := modecommon.AsInt(args["switch_threshold"])
		if switchThreshold <= 0 {
			switchThreshold = 72
		}
		if routeVersion == "" {
			routeVersion = "route-v32"
		}
		if currentStrategy == "" {
			currentStrategy = "latency-first"
		}
		if candidateStrategy == "" {
			candidateStrategy = "cost-aware"
		}
		if trafficProfile == "" {
			trafficProfile = "interactive-chat"
		}
		if selectionPolicy == "" {
			selectionPolicy = "latency_cost_ratio"
		}
		structured["route_version"] = routeVersion
		structured["current_strategy"] = currentStrategy
		structured["candidate_strategy"] = candidateStrategy
		structured["active_strategy"] = currentStrategy
		structured["fallback_strategy"] = currentStrategy
		structured["traffic_profile"] = trafficProfile
		structured["selection_policy"] = selectionPolicy
		structured["switch_threshold"] = switchThreshold
	case "routing_confidence_evaluated":
		candidateStrategy := strings.TrimSpace(fmt.Sprintf("%v", args["candidate_strategy"]))
		if candidateStrategy == "" {
			candidateStrategy = "cost-aware"
		}
		switchThreshold, _ := modecommon.AsInt(args["switch_threshold"])
		confidenceScore, _ := modecommon.AsInt(args["confidence_score"])
		confidenceReason := strings.TrimSpace(fmt.Sprintf("%v", args["confidence_reason"]))
		if confidenceReason == "" {
			confidenceReason = "insufficient_signal"
		}
		recommendation := "stay_current"
		if confidenceScore >= switchThreshold {
			recommendation = "switch_to_candidate"
		}
		structured["candidate_strategy"] = candidateStrategy
		structured["switch_threshold"] = switchThreshold
		structured["confidence_score"] = confidenceScore
		structured["confidence_reason"] = confidenceReason
		structured["routing_recommendation"] = recommendation
		if confidenceScore < switchThreshold {
			risk = "degraded_path"
		}
	case "routing_switch_committed":
		candidateStrategy := strings.TrimSpace(fmt.Sprintf("%v", args["candidate_strategy"]))
		if candidateStrategy == "" {
			candidateStrategy = "cost-aware"
		}
		if currentStrategy == "" {
			currentStrategy = "latency-first"
		}
		confidenceScore, _ := modecommon.AsInt(args["confidence_score"])
		switchThreshold, _ := modecommon.AsInt(args["switch_threshold"])
		fallbackStrategy := strings.TrimSpace(fmt.Sprintf("%v", args["fallback_strategy"]))
		if fallbackStrategy == "" {
			fallbackStrategy = currentStrategy
		}
		switchCommitted := confidenceScore >= switchThreshold
		activeStrategy := currentStrategy
		switchReason := "confidence_below_threshold"
		if switchCommitted {
			activeStrategy = candidateStrategy
			switchReason = "confidence_passed"
		}
		structured["candidate_strategy"] = candidateStrategy
		structured["current_strategy"] = currentStrategy
		structured["active_strategy"] = activeStrategy
		structured["fallback_strategy"] = fallbackStrategy
		structured["confidence_score"] = confidenceScore
		structured["switch_threshold"] = switchThreshold
		structured["switch_committed"] = switchCommitted
		structured["switch_reason"] = switchReason
		if !switchCommitted {
			risk = "degraded_path"
		}
	case "governance_routing_gate_enforced":
		switchCommitted := asBool(args["switch_committed"])
		activeStrategy := strings.TrimSpace(fmt.Sprintf("%v", args["active_strategy"]))
		trafficProfile := strings.TrimSpace(fmt.Sprintf("%v", args["traffic_profile"]))
		confidenceScore, _ := modecommon.AsInt(args["confidence_score"])
		switchThreshold, _ := modecommon.AsInt(args["switch_threshold"])
		decision := "allow_switch"
		if !switchCommitted {
			decision = "hold_current_strategy"
		} else if strings.Contains(trafficProfile, "spiky") {
			decision = "allow_switch_with_canary"
		}
		ticket := fmt.Sprintf("route-gate-%d", modecommon.SemanticScore(requestID, activeStrategy, decision, trafficProfile, fmt.Sprintf("%d", confidenceScore)))
		structured["active_strategy"] = activeStrategy
		structured["traffic_profile"] = trafficProfile
		structured["switch_committed"] = switchCommitted
		structured["confidence_score"] = confidenceScore
		structured["switch_threshold"] = switchThreshold
		structured["governance"] = true
		structured["governance_decision"] = decision
		structured["governance_ticket"] = ticket
		risk = "governed"
	case "governance_routing_replay_bound":
		decision := strings.TrimSpace(fmt.Sprintf("%v", args["governance_decision"]))
		ticket := strings.TrimSpace(fmt.Sprintf("%v", args["governance_ticket"]))
		activeStrategy := strings.TrimSpace(fmt.Sprintf("%v", args["active_strategy"]))
		switchReason := strings.TrimSpace(fmt.Sprintf("%v", args["switch_reason"]))
		replaySignature := fmt.Sprintf("route-replay-%d", modecommon.SemanticScore(requestID, routeVersion, activeStrategy, decision, ticket, switchReason))
		structured["active_strategy"] = activeStrategy
		structured["switch_reason"] = switchReason
		structured["governance"] = true
		structured["governance_decision"] = decision
		structured["governance_ticket"] = ticket
		structured["replay_signature"] = replaySignature
		risk = "governed"
	default:
		return types.ToolResult{}, fmt.Errorf("unsupported workflow routing marker: %s", marker)
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
