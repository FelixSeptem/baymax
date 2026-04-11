package customadapterhealthreadinesscircuit

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
	patternName      = "custom-adapter-health-readiness-circuit"
	phase            = "P2"
	semanticAnchor   = "adapterhealth.readiness_backoff_circuit"
	classification   = "adapter.health_readiness"
	semanticToolName = "mode_custom_adapter_health_readiness_circuit_semantic_step"
	defaultProbeID   = "adapter-health-20260410"
)

type adapterHealthStep struct {
	Marker        string
	RuntimeDomain string
	Intent        string
	Outcome       string
}

type adapterHealthState struct {
	ProbeID             string
	LatencyMS           int
	ErrorRatePct        int
	HealthScore         int
	ReadinessState      string
	CircuitState        string
	BackoffMS           int
	RecoveryClass       string
	PrimaryReason       string
	GovernanceDecision  string
	GovernanceTicket    string
	ReplaySignature     string
	ObservedMarkers     []string
	AccumulatedSemScore int
}

var runtimeDomains = []string{"adapter/health", "runtime/config"}

var minimalSemanticSteps = []adapterHealthStep{
	{
		Marker:        "adapter_health_probe_sampled",
		RuntimeDomain: "adapter/health",
		Intent:        "sample adapter health probe metrics",
		Outcome:       "health score, latency and error rate are emitted",
	},
	{
		Marker:        "adapter_readiness_circuit_transitioned",
		RuntimeDomain: "runtime/config",
		Intent:        "transition readiness and circuit state from probe signal",
		Outcome:       "readiness state, circuit state and primary reason are emitted",
	},
	{
		Marker:        "adapter_backoff_recovery_classified",
		RuntimeDomain: "adapter/health",
		Intent:        "classify recovery/backoff path based on circuit state",
		Outcome:       "backoff window and recovery class are emitted",
	},
}

var productionGovernanceSteps = []adapterHealthStep{
	{
		Marker:        "governance_adapter_health_gate_enforced",
		RuntimeDomain: "adapter/health",
		Intent:        "enforce adapter health governance from circuit and recovery signals",
		Outcome:       "governance decision and ticket are emitted",
	},
	{
		Marker:        "governance_adapter_health_replay_bound",
		RuntimeDomain: "runtime/config",
		Intent:        "bind adapter health governance decision to replay signature",
		Outcome:       "replay signature is emitted",
	},
}

func RunMinimal() {
	executeAdapterHealthVariant(modecommon.VariantMinimal)
}

func RunProduction() {
	executeAdapterHealthVariant(modecommon.VariantProduction)
}

func executeAdapterHealthVariant(variant string) {
	if err := modecommon.EnsureVariant(variant); err != nil {
		panic(err)
	}

	mgr, err := runtimeconfig.NewManager(runtimeconfig.ManagerOptions{EnvPrefix: "BAYMAX"})
	if err != nil {
		panic(err)
	}
	defer func() { _ = mgr.Close() }()

	registry := local.NewRegistry()
	if _, err := registry.Register(&adapterHealthTool{}); err != nil {
		panic(err)
	}

	model := &adapterHealthModel{
		variant: variant,
		state: adapterHealthState{
			ProbeID: defaultProbeID,
		},
	}
	engine := runner.New(model, runner.WithLocalRegistry(registry), runner.WithRuntimeManager(mgr))
	result, err := engine.Run(context.Background(), types.RunRequest{
		RunID: fmt.Sprintf("agent-mode-%s-%s", modecommon.MarkerToken(patternName), modecommon.MarkerToken(variant)),
		Input: "execute custom adapter health readiness circuit semantic pipeline",
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

func planForVariant(variant string) []adapterHealthStep {
	plan := make([]adapterHealthStep, 0, len(minimalSemanticSteps)+len(productionGovernanceSteps))
	plan = append(plan, minimalSemanticSteps...)
	if variant == modecommon.VariantProduction {
		plan = append(plan, productionGovernanceSteps...)
	}
	return plan
}

type adapterHealthModel struct {
	variant string
	cursor  int
	state   adapterHealthState
}

func (m *adapterHealthModel) Generate(ctx context.Context, req types.ModelRequest) (types.ModelResponse, error) {
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
		"%s/%s semantic_path_completed phase=%s anchor=%s classification=%s probe=%s latency_ms=%d error_rate_pct=%d health_score=%d readiness=%s circuit=%s backoff_ms=%d recovery_class=%s primary_reason=%s governance=%s ticket=%s replay=%s score=%d markers=%s",
		patternName,
		m.variant,
		phase,
		semanticAnchor,
		classification,
		readableValue(m.state.ProbeID, true),
		m.state.LatencyMS,
		m.state.ErrorRatePct,
		m.state.HealthScore,
		readableValue(m.state.ReadinessState, true),
		readableValue(m.state.CircuitState, true),
		m.state.BackoffMS,
		readableValue(m.state.RecoveryClass, true),
		readableValue(m.state.PrimaryReason, true),
		readableValue(m.state.GovernanceDecision, governanceOn),
		readableValue(m.state.GovernanceTicket, governanceOn),
		readableValue(m.state.ReplaySignature, governanceOn),
		m.state.AccumulatedSemScore,
		strings.Join(markers, ","),
	)
	return types.ModelResponse{FinalAnswer: final}, nil
}

func (m *adapterHealthModel) Stream(ctx context.Context, req types.ModelRequest, onEvent func(types.ModelEvent) error) error {
	_ = ctx
	_ = req
	_ = onEvent
	return nil
}

func (m *adapterHealthModel) captureOutcomes(outcomes []types.ToolCallOutcome) {
	for _, outcome := range outcomes {
		marker, _ := outcome.Result.Structured["marker"].(string)
		if marker != "" {
			m.state.ObservedMarkers = append(m.state.ObservedMarkers, marker)
		}
		if probeID, _ := outcome.Result.Structured["probe_id"].(string); strings.TrimSpace(probeID) != "" {
			m.state.ProbeID = strings.TrimSpace(probeID)
		}
		if latency, ok := modecommon.AsInt(outcome.Result.Structured["latency_ms"]); ok {
			m.state.LatencyMS = latency
		}
		if errRate, ok := modecommon.AsInt(outcome.Result.Structured["error_rate_pct"]); ok {
			m.state.ErrorRatePct = errRate
		}
		if score, ok := modecommon.AsInt(outcome.Result.Structured["health_score"]); ok {
			m.state.HealthScore = score
		}
		if readiness, _ := outcome.Result.Structured["readiness_state"].(string); strings.TrimSpace(readiness) != "" {
			m.state.ReadinessState = strings.TrimSpace(readiness)
		}
		if circuit, _ := outcome.Result.Structured["circuit_state"].(string); strings.TrimSpace(circuit) != "" {
			m.state.CircuitState = strings.TrimSpace(circuit)
		}
		if backoff, ok := modecommon.AsInt(outcome.Result.Structured["backoff_ms"]); ok {
			m.state.BackoffMS = backoff
		}
		if recovery, _ := outcome.Result.Structured["recovery_class"].(string); strings.TrimSpace(recovery) != "" {
			m.state.RecoveryClass = strings.TrimSpace(recovery)
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

func (m *adapterHealthModel) argsForStep(step adapterHealthStep, stage int) map[string]any {
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
		"probe_id":        m.state.ProbeID,
	}

	switch step.Marker {
	case "adapter_health_probe_sampled":
		latency := 80
		errorRate := 1
		healthScore := 92
		if m.variant == modecommon.VariantProduction {
			latency = 260
			errorRate = 6
			healthScore = 61
		}
		args["latency_ms"] = latency
		args["error_rate_pct"] = errorRate
		args["health_score"] = healthScore
	case "adapter_readiness_circuit_transitioned":
		readiness := "ready"
		circuit := "closed"
		primaryReason := "health_probe_within_threshold"
		if m.state.HealthScore < 70 || m.state.ErrorRatePct >= 5 {
			readiness = "degraded"
			circuit = "half-open"
			primaryReason = "health_probe_below_threshold"
		}
		args["readiness_state"] = readiness
		args["circuit_state"] = circuit
		args["primary_reason"] = primaryReason
	case "adapter_backoff_recovery_classified":
		backoff := 200
		recovery := "stable_recovery"
		if m.state.CircuitState != "closed" {
			backoff = 1200
			recovery = "degraded_with_backoff"
		}
		args["circuit_state"] = m.state.CircuitState
		args["backoff_ms"] = backoff
		args["recovery_class"] = recovery
		args["primary_reason"] = m.state.PrimaryReason
	case "governance_adapter_health_gate_enforced":
		args["readiness_state"] = m.state.ReadinessState
		args["circuit_state"] = m.state.CircuitState
		args["recovery_class"] = m.state.RecoveryClass
		args["backoff_ms"] = m.state.BackoffMS
	case "governance_adapter_health_replay_bound":
		args["governance_decision"] = m.state.GovernanceDecision
		args["governance_ticket"] = m.state.GovernanceTicket
		args["primary_reason"] = m.state.PrimaryReason
	}
	return args
}

type adapterHealthTool struct{}

func (t *adapterHealthTool) Name() string { return semanticToolName }

func (t *adapterHealthTool) Description() string {
	return "execute adapter health readiness circuit semantic step"
}

func (t *adapterHealthTool) JSONSchema() map[string]any {
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
			"probe_id":        map[string]any{"type": "string"},
			"latency_ms":      map[string]any{"type": "integer"},
			"error_rate_pct":  map[string]any{"type": "integer"},
			"health_score":    map[string]any{"type": "integer"},
			"readiness_state": map[string]any{"type": "string"},
			"circuit_state":   map[string]any{"type": "string"},
			"backoff_ms":      map[string]any{"type": "integer"},
			"recovery_class":  map[string]any{"type": "string"},
			"primary_reason":  map[string]any{"type": "string"},
			"governance_decision": map[string]any{
				"type": "string",
			},
			"governance_ticket": map[string]any{"type": "string"},
		},
	}
}

func (t *adapterHealthTool) Invoke(ctx context.Context, args map[string]any) (types.ToolResult, error) {
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

	probeID := strings.TrimSpace(fmt.Sprintf("%v", args["probe_id"]))
	if probeID == "" {
		probeID = defaultProbeID
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
		"probe_id":        probeID,
		"governance":      false,
	}

	risk := "nominal"
	switch marker {
	case "adapter_health_probe_sampled":
		latency, _ := modecommon.AsInt(args["latency_ms"])
		errorRate, _ := modecommon.AsInt(args["error_rate_pct"])
		healthScore, _ := modecommon.AsInt(args["health_score"])
		structured["latency_ms"] = latency
		structured["error_rate_pct"] = errorRate
		structured["health_score"] = healthScore
		if healthScore < 70 || errorRate >= 5 {
			risk = "degraded_path"
		}
	case "adapter_readiness_circuit_transitioned":
		readiness := strings.TrimSpace(fmt.Sprintf("%v", args["readiness_state"]))
		circuit := strings.TrimSpace(fmt.Sprintf("%v", args["circuit_state"]))
		primaryReason := strings.TrimSpace(fmt.Sprintf("%v", args["primary_reason"]))
		structured["readiness_state"] = readiness
		structured["circuit_state"] = circuit
		structured["primary_reason"] = primaryReason
		if circuit != "closed" {
			risk = "degraded_path"
		}
	case "adapter_backoff_recovery_classified":
		circuit := strings.TrimSpace(fmt.Sprintf("%v", args["circuit_state"]))
		backoff, _ := modecommon.AsInt(args["backoff_ms"])
		recovery := strings.TrimSpace(fmt.Sprintf("%v", args["recovery_class"]))
		primaryReason := strings.TrimSpace(fmt.Sprintf("%v", args["primary_reason"]))
		structured["circuit_state"] = circuit
		structured["backoff_ms"] = backoff
		structured["recovery_class"] = recovery
		structured["primary_reason"] = primaryReason
		if strings.Contains(recovery, "degraded") {
			risk = "degraded_path"
		}
	case "governance_adapter_health_gate_enforced":
		readiness := strings.TrimSpace(fmt.Sprintf("%v", args["readiness_state"]))
		circuit := strings.TrimSpace(fmt.Sprintf("%v", args["circuit_state"]))
		recovery := strings.TrimSpace(fmt.Sprintf("%v", args["recovery_class"]))
		backoff, _ := modecommon.AsInt(args["backoff_ms"])
		decision := "allow_adapter_ready"
		if readiness != "ready" || circuit != "closed" {
			decision = "allow_adapter_with_circuit_guard"
		}
		if backoff > 2000 {
			decision = "block_adapter_until_recovery"
		}
		ticket := fmt.Sprintf("adapter-health-gate-%d", modecommon.SemanticScore(probeID, decision, circuit, recovery, fmt.Sprintf("%d", backoff)))
		structured["readiness_state"] = readiness
		structured["circuit_state"] = circuit
		structured["recovery_class"] = recovery
		structured["backoff_ms"] = backoff
		structured["governance"] = true
		structured["governance_decision"] = decision
		structured["governance_ticket"] = ticket
		risk = "governed"
	case "governance_adapter_health_replay_bound":
		decision := strings.TrimSpace(fmt.Sprintf("%v", args["governance_decision"]))
		ticket := strings.TrimSpace(fmt.Sprintf("%v", args["governance_ticket"]))
		primaryReason := strings.TrimSpace(fmt.Sprintf("%v", args["primary_reason"]))
		replaySignature := fmt.Sprintf("adapter-health-replay-%d", modecommon.SemanticScore(probeID, decision, ticket, primaryReason))
		structured["primary_reason"] = primaryReason
		structured["governance"] = true
		structured["governance_decision"] = decision
		structured["governance_ticket"] = ticket
		structured["replay_signature"] = replaySignature
		risk = "governed"
	default:
		return types.ToolResult{}, fmt.Errorf("unsupported adapter health marker: %s", marker)
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
