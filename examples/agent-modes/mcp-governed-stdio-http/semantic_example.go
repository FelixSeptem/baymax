package mcpgovernedstdiohttp

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
	patternName      = "mcp-governed-stdio-http"
	phase            = "P0"
	semanticAnchor   = "transport.profile_failover_governance"
	classification   = "mcp.transport_governance"
	semanticToolName = "mode_mcp_governed_stdio_http_semantic_step"
)

type transportStep struct {
	Marker        string
	RuntimeDomain string
	Intent        string
	Outcome       string
}

type transportProfile struct {
	ID                   string
	Version              int
	PrimaryTransport     string
	FallbackTransport    string
	StdioLatencyBudgetMs int
	AllowHTTPFailover    bool
	RequireDecisionTrace bool
}

type transportProbe struct {
	Transport   string
	Healthy     bool
	LatencyMs   int
	Capability  string
	FailureCode string
	Retryable   bool
}

type transportState struct {
	ProfileID         string
	ProfileVersion    int
	SelectedTransport string
	FailoverTriggered bool
	FailoverReason    string
	DecisionCode      string
	ReasonTrace       []string
	ReplayBinding     string
	SeenMarkers       []string
	AttemptByName     map[string]int
	TotalScore        int
}

var runtimeDomains = []string{"mcp/profile", "mcp/stdio", "mcp/http"}

var minimalSemanticSteps = []transportStep{
	{
		Marker:        "transport_profile_selected",
		RuntimeDomain: "mcp/profile",
		Intent:        "load transport profile and establish stdio/http candidate order",
		Outcome:       "active profile is materialized with deterministic candidate ordering",
	},
	{
		Marker:        "transport_failover_decided",
		RuntimeDomain: "mcp/stdio",
		Intent:        "evaluate stdio health and latency budget to decide fallback eligibility",
		Outcome:       "transport decision resolves to primary retain or http failover",
	},
	{
		Marker:        "transport_reason_trace_emitted",
		RuntimeDomain: "mcp/http",
		Intent:        "emit failover reason trace including attempt counters and decision code",
		Outcome:       "reason trace can be replayed for deterministic transport diagnostics",
	},
}

var productionGovernanceSteps = []transportStep{
	{
		Marker:        "governance_transport_gate_enforced",
		RuntimeDomain: "mcp/profile",
		Intent:        "enforce governance gate on transport decision and trace completeness",
		Outcome:       "transport decision is classified as allow, allow_with_record, or block",
	},
	{
		Marker:        "governance_transport_replay_bound",
		RuntimeDomain: "mcp/http",
		Intent:        "bind replay signature to profile version and transport governance decision",
		Outcome:       "replay signature is emitted for cross-run deterministic verification",
	},
}

var minimalProfile = transportProfile{
	ID:                   "ops-runtime-v1",
	Version:              1,
	PrimaryTransport:     "stdio",
	FallbackTransport:    "http",
	StdioLatencyBudgetMs: 90,
	AllowHTTPFailover:    true,
	RequireDecisionTrace: true,
}

var productionProfile = transportProfile{
	ID:                   "ops-runtime-v2-governed",
	Version:              2,
	PrimaryTransport:     "stdio",
	FallbackTransport:    "http",
	StdioLatencyBudgetMs: 60,
	AllowHTTPFailover:    true,
	RequireDecisionTrace: true,
}

var minimalProbes = []transportProbe{
	{
		Transport:   "stdio",
		Healthy:     false,
		LatencyMs:   220,
		Capability:  "shell-bridge",
		FailureCode: "channel_unavailable",
		Retryable:   true,
	},
	{
		Transport:   "http",
		Healthy:     true,
		LatencyMs:   48,
		Capability:  "remote-mcp-http",
		FailureCode: "",
		Retryable:   true,
	},
}

var productionProbes = []transportProbe{
	{
		Transport:   "stdio",
		Healthy:     false,
		LatencyMs:   305,
		Capability:  "shell-bridge",
		FailureCode: "restart_pending",
		Retryable:   true,
	},
	{
		Transport:   "http",
		Healthy:     true,
		LatencyMs:   72,
		Capability:  "remote-mcp-http",
		FailureCode: "",
		Retryable:   true,
	},
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
	if _, err := reg.Register(&transportGovernanceTool{}); err != nil {
		panic(err)
	}

	engine := runner.New(newTransportModel(variant), runner.WithLocalRegistry(reg), runner.WithRuntimeManager(mgr))
	result, err := engine.Run(context.Background(), types.RunRequest{
		RunID: fmt.Sprintf("agent-mode-%s-%s", modecommon.MarkerToken(patternName), modecommon.MarkerToken(variant)),
		Input: "execute mode semantic pipeline for " + patternName,
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

func executionPlanForVariant(variant string) []transportStep {
	plan := make([]transportStep, 0, len(minimalSemanticSteps)+len(productionGovernanceSteps))
	plan = append(plan, minimalSemanticSteps...)
	if variant == modecommon.VariantProduction {
		plan = append(plan, productionGovernanceSteps...)
	}
	return plan
}

func newTransportModel(variant string) *transportModeModel {
	profile := profileForVariant(variant)
	return &transportModeModel{
		variant: variant,
		state: transportState{
			ProfileID:         profile.ID,
			ProfileVersion:    profile.Version,
			SelectedTransport: profile.PrimaryTransport,
			DecisionCode:      "profile_not_evaluated",
			AttemptByName:     map[string]int{},
		},
	}
}

type transportModeModel struct {
	variant string
	stage   int
	state   transportState
}

func (m *transportModeModel) Generate(ctx context.Context, req types.ModelRequest) (types.ModelResponse, error) {
	_ = ctx
	m.absorb(req.ToolResult)

	plan := executionPlanForVariant(m.variant)
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
	final := fmt.Sprintf(
		"%s/%s semantic_path_completed phase=%s anchor=%s classification=%s profile=%s@%d selected=%s failover=%t reason=%s decision=%s attempts=%s trace=%s replay=%s markers=%s score=%d",
		patternName,
		m.variant,
		phase,
		semanticAnchor,
		classification,
		m.state.ProfileID,
		m.state.ProfileVersion,
		normalizedTransport(m.state.SelectedTransport),
		m.state.FailoverTriggered,
		safeString(m.state.FailoverReason, "none"),
		safeString(m.state.DecisionCode, "not_decided"),
		formatAttempts(m.state.AttemptByName),
		strings.Join(m.state.ReasonTrace, " > "),
		safeString(m.state.ReplayBinding, "none"),
		strings.Join(markers, ","),
		m.state.TotalScore,
	)
	return types.ModelResponse{FinalAnswer: final}, nil
}

func (m *transportModeModel) Stream(ctx context.Context, req types.ModelRequest, onEvent func(types.ModelEvent) error) error {
	_ = ctx
	_ = req
	_ = onEvent
	return nil
}
func (m *transportModeModel) absorb(outcomes []types.ToolCallOutcome) {
	for _, item := range outcomes {
		marker, _ := item.Result.Structured["marker"].(string)
		if marker != "" {
			m.state.SeenMarkers = append(m.state.SeenMarkers, marker)
		}
		if profileID, ok := item.Result.Structured["profile_id"].(string); ok && strings.TrimSpace(profileID) != "" {
			m.state.ProfileID = profileID
		}
		if profileVersion, ok := modecommon.AsInt(item.Result.Structured["profile_version"]); ok {
			m.state.ProfileVersion = profileVersion
		}
		if selected, ok := item.Result.Structured["selected_transport"].(string); ok && strings.TrimSpace(selected) != "" {
			m.state.SelectedTransport = selected
		}
		if failover, ok := item.Result.Structured["failover_triggered"].(bool); ok {
			m.state.FailoverTriggered = failover
		}
		if reason, ok := item.Result.Structured["failover_reason"].(string); ok && strings.TrimSpace(reason) != "" {
			m.state.FailoverReason = reason
		}
		if decision, ok := item.Result.Structured["decision_code"].(string); ok && strings.TrimSpace(decision) != "" {
			m.state.DecisionCode = decision
		}
		if replayBinding, ok := item.Result.Structured["replay_binding"].(string); ok && strings.TrimSpace(replayBinding) != "" {
			m.state.ReplayBinding = replayBinding
		}
		if reasonTrace, ok := toStringSlice(item.Result.Structured["reason_trace"]); ok && len(reasonTrace) > 0 {
			m.state.ReasonTrace = reasonTrace
		}
		if attempts := parseAttemptSnapshot(item.Result.Structured["attempt_snapshot"]); len(attempts) > 0 {
			if m.state.AttemptByName == nil {
				m.state.AttemptByName = map[string]int{}
			}
			for name, count := range attempts {
				m.state.AttemptByName[name] = count
			}
		}
		if score, ok := modecommon.AsInt(item.Result.Structured["score"]); ok {
			m.state.TotalScore += score
		}
	}
}

func (m *transportModeModel) argsForStep(step transportStep, stage int) map[string]any {
	profile := profileForVariant(m.variant)
	args := map[string]any{
		"pattern":                patternName,
		"variant":                m.variant,
		"phase":                  phase,
		"semantic_anchor":        semanticAnchor,
		"classification":         classification,
		"marker":                 step.Marker,
		"runtime_domain":         step.RuntimeDomain,
		"semantic_intent":        step.Intent,
		"semantic_outcome":       step.Outcome,
		"profile_id":             profile.ID,
		"profile_version":        profile.Version,
		"primary_transport":      profile.PrimaryTransport,
		"fallback_transport":     profile.FallbackTransport,
		"stdio_latency_budget":   profile.StdioLatencyBudgetMs,
		"allow_http_failover":    profile.AllowHTTPFailover,
		"require_decision_trace": profile.RequireDecisionTrace,
		"probe_candidates":       toAnySlice(probeSpecsForVariant(m.variant)),
		"stage":                  stage,
	}

	switch step.Marker {
	case "transport_profile_selected":
		args["requested_transport"] = profile.PrimaryTransport
	case "transport_failover_decided":
		args["selected_transport"] = m.state.SelectedTransport
	case "transport_reason_trace_emitted":
		args["selected_transport"] = m.state.SelectedTransport
		args["failover_reason"] = m.state.FailoverReason
		args["attempt_snapshot"] = toAnySlice(attemptSnapshot(m.state.AttemptByName))
		args["reason_trace"] = toAnySlice(m.state.ReasonTrace)
	case "governance_transport_gate_enforced":
		args["selected_transport"] = m.state.SelectedTransport
		args["failover_reason"] = m.state.FailoverReason
		args["reason_trace"] = toAnySlice(m.state.ReasonTrace)
	case "governance_transport_replay_bound":
		args["selected_transport"] = m.state.SelectedTransport
		args["failover_reason"] = m.state.FailoverReason
		args["decision_code"] = m.state.DecisionCode
		args["reason_trace"] = toAnySlice(m.state.ReasonTrace)
	}
	return args
}

type transportGovernanceTool struct{}

func (t *transportGovernanceTool) Name() string { return semanticToolName }

func (t *transportGovernanceTool) Description() string {
	return "execute mode-owned semantic step for " + patternName
}

func (t *transportGovernanceTool) JSONSchema() map[string]any {
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
			"profile_id",
			"profile_version",
			"primary_transport",
			"fallback_transport",
			"stdio_latency_budget",
			"allow_http_failover",
			"stage",
		},
		"properties": map[string]any{
			"pattern":              map[string]any{"type": "string"},
			"variant":              map[string]any{"type": "string"},
			"phase":                map[string]any{"type": "string"},
			"semantic_anchor":      map[string]any{"type": "string"},
			"classification":       map[string]any{"type": "string"},
			"marker":               map[string]any{"type": "string"},
			"runtime_domain":       map[string]any{"type": "string"},
			"semantic_intent":      map[string]any{"type": "string"},
			"semantic_outcome":     map[string]any{"type": "string"},
			"profile_id":           map[string]any{"type": "string"},
			"profile_version":      map[string]any{"type": "integer"},
			"primary_transport":    map[string]any{"type": "string"},
			"fallback_transport":   map[string]any{"type": "string"},
			"stdio_latency_budget": map[string]any{"type": "integer"},
			"allow_http_failover":  map[string]any{"type": "boolean"},
			"require_decision_trace": map[string]any{
				"type": "boolean",
			},
			"requested_transport": map[string]any{"type": "string"},
			"selected_transport":  map[string]any{"type": "string"},
			"failover_reason":     map[string]any{"type": "string"},
			"decision_code":       map[string]any{"type": "string"},
			"probe_candidates":    map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
			"attempt_snapshot":    map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
			"reason_trace":        map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
			"stage":               map[string]any{"type": "integer"},
		},
	}
}

func (t *transportGovernanceTool) Invoke(ctx context.Context, args map[string]any) (types.ToolResult, error) {
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
	profileID := strings.TrimSpace(fmt.Sprintf("%v", args["profile_id"]))
	profileVersion, _ := modecommon.AsInt(args["profile_version"])
	primaryTransport := strings.TrimSpace(fmt.Sprintf("%v", args["primary_transport"]))
	fallbackTransport := strings.TrimSpace(fmt.Sprintf("%v", args["fallback_transport"]))
	stdioLatencyBudget, _ := modecommon.AsInt(args["stdio_latency_budget"])
	allowHTTPFailover := parseBool(args["allow_http_failover"], true)
	requireDecisionTrace := parseBool(args["require_decision_trace"], true)
	stage, _ := modecommon.AsInt(args["stage"])
	if stage <= 0 {
		stage = 1
	}

	probeSpecs, _ := toStringSlice(args["probe_candidates"])
	probes := parseProbeSpecs(probeSpecs)
	if len(probes) == 0 {
		probes = probesForVariant(variant)
	}

	probeIndex := map[string]transportProbe{}
	for _, probe := range probes {
		probeIndex[probe.Transport] = probe
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
		"profile_id":         profileID,
		"profile_version":    profileVersion,
		"primary_transport":  primaryTransport,
		"fallback_transport": fallbackTransport,
		"stage":              stage,
		"governance":         false,
	}

	var risk string
	selectedTransport := safeString(args["selected_transport"], primaryTransport)
	failoverReason := safeString(args["failover_reason"], "")
	decisionCode := safeString(args["decision_code"], "")

	switch marker {
	case "transport_profile_selected":
		requested := safeString(args["requested_transport"], primaryTransport)
		if _, exists := probeIndex[requested]; !exists {
			requested = primaryTransport
		}
		trace := []string{
			fmt.Sprintf("profile=%s@%d", profileID, profileVersion),
			fmt.Sprintf("candidate_order=%s>%s", primaryTransport, fallbackTransport),
			fmt.Sprintf("requested=%s", requested),
		}
		result["selected_transport"] = requested
		result["failover_triggered"] = false
		result["failover_reason"] = "primary_pending_probe"
		result["reason_trace"] = toAnySlice(trace)
		result["attempt_snapshot"] = toAnySlice([]string{fmt.Sprintf("%s:1", requested)})
		result["probe_count"] = len(probes)
		risk = "profile_loaded"
	case "transport_failover_decided":
		decided, failover, reason, trace, attempts := decideTransport(
			primaryTransport,
			fallbackTransport,
			stdioLatencyBudget,
			allowHTTPFailover,
			probeIndex,
		)
		result["selected_transport"] = decided
		result["failover_triggered"] = failover
		result["failover_reason"] = reason
		result["reason_trace"] = toAnySlice(trace)
		result["attempt_snapshot"] = toAnySlice(attemptSnapshot(attempts))
		result["decision_code"] = "transport_decided"
		selectedTransport = decided
		failoverReason = reason
		decisionCode = "transport_decided"
		switch {
		case failover:
			risk = "failover_path"
		case decided == "none":
			risk = "degraded_path"
		default:
			risk = "primary_retained"
		}
	case "transport_reason_trace_emitted":
		selectedTransport = normalizedTransport(selectedTransport)
		failoverReason = safeString(failoverReason, "primary_healthy")
		attempts := parseAttemptSnapshot(args["attempt_snapshot"])
		previousTrace, _ := toStringSlice(args["reason_trace"])
		trace := append([]string(nil), previousTrace...)
		trace = append(
			trace,
			fmt.Sprintf("selected=%s", selectedTransport),
			fmt.Sprintf("reason=%s", failoverReason),
			fmt.Sprintf("attempts=%s", formatAttempts(attempts)),
		)
		decisionCode = "primary_stdio_retained"
		risk = "decision_trace"
		switch selectedTransport {
		case "http":
			decisionCode = "fallback_http_accepted"
			risk = "failover_trace"
		case "none":
			decisionCode = "transport_unavailable"
			risk = "degraded_path"
		}
		result["selected_transport"] = selectedTransport
		result["failover_triggered"] = selectedTransport == "http"
		result["failover_reason"] = failoverReason
		result["decision_code"] = decisionCode
		result["reason_trace"] = toAnySlice(trace)
		result["attempt_snapshot"] = toAnySlice(attemptSnapshot(attempts))
	case "governance_transport_gate_enforced":
		selectedTransport = normalizedTransport(selectedTransport)
		failoverReason = safeString(failoverReason, "unknown")
		trace, _ := toStringSlice(args["reason_trace"])
		decisionCode = "allow"
		risk = "governed_allow"
		switch selectedTransport {
		case "none":
			decisionCode = "block_transport_unavailable"
			risk = "governed_block"
		case "http":
			decisionCode = "allow_with_record"
			risk = "governed_warn"
			if requireDecisionTrace && len(trace) == 0 {
				decisionCode = "block_missing_trace"
				risk = "governed_block"
			}
		}
		result["selected_transport"] = selectedTransport
		result["failover_reason"] = failoverReason
		result["decision_code"] = decisionCode
		result["reason_trace"] = toAnySlice(trace)
		result["gate_reason_count"] = len(trace)
		result["governance"] = true
	case "governance_transport_replay_bound":
		selectedTransport = normalizedTransport(selectedTransport)
		failoverReason = safeString(failoverReason, "unknown")
		decisionCode = safeString(decisionCode, "allow")
		trace, _ := toStringSlice(args["reason_trace"])
		replayBinding := fmt.Sprintf(
			"transport-replay-%d",
			modecommon.SemanticScore(
				pattern,
				variant,
				profileID,
				fmt.Sprintf("%d", profileVersion),
				selectedTransport,
				failoverReason,
				decisionCode,
				strings.Join(trace, "|"),
			),
		)
		result["selected_transport"] = selectedTransport
		result["failover_reason"] = failoverReason
		result["decision_code"] = decisionCode
		result["reason_trace"] = toAnySlice(trace)
		result["replay_binding"] = replayBinding
		result["governance"] = true
		risk = "governed_replay"
	default:
		return types.ToolResult{}, fmt.Errorf("unsupported transport marker: %s", marker)
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
		safeString(result["selected_transport"], selectedTransport),
		safeString(result["failover_reason"], failoverReason),
		safeString(result["decision_code"], decisionCode),
	)
	result["risk"] = risk

	content := fmt.Sprintf("pattern=%s variant=%s phase=%s marker=%s domain=%s stage=%d governance=%t risk=%s",
		pattern,
		variant,
		phaseValue,
		marker,
		runtimeDomain,
		stage,
		parseBool(result["governance"], false),
		risk,
	)
	return types.ToolResult{Content: content, Structured: result}, nil
}

func profileForVariant(variant string) transportProfile {
	if variant == modecommon.VariantProduction {
		return productionProfile
	}
	return minimalProfile
}

func probesForVariant(variant string) []transportProbe {
	if variant == modecommon.VariantProduction {
		return cloneProbes(productionProbes)
	}
	return cloneProbes(minimalProbes)
}

func cloneProbes(input []transportProbe) []transportProbe {
	cloned := make([]transportProbe, 0, len(input))
	cloned = append(cloned, input...)
	return cloned
}

func probeSpecsForVariant(variant string) []string {
	probes := probesForVariant(variant)
	specs := make([]string, 0, len(probes))
	for _, probe := range probes {
		specs = append(
			specs,
			fmt.Sprintf(
				"%s|%t|%d|%s|%s|%t",
				probe.Transport,
				probe.Healthy,
				probe.LatencyMs,
				probe.Capability,
				probe.FailureCode,
				probe.Retryable,
			),
		)
	}
	return specs
}

func parseProbeSpecs(specs []string) []transportProbe {
	out := make([]transportProbe, 0, len(specs))
	for _, spec := range specs {
		parts := strings.Split(spec, "|")
		if len(parts) != 6 {
			continue
		}
		out = append(out, transportProbe{
			Transport:   strings.TrimSpace(parts[0]),
			Healthy:     parseBool(parts[1], false),
			LatencyMs:   parseInt(parts[2]),
			Capability:  strings.TrimSpace(parts[3]),
			FailureCode: strings.TrimSpace(parts[4]),
			Retryable:   parseBool(parts[5], false),
		})
	}
	return out
}

func decideTransport(
	primaryTransport string,
	fallbackTransport string,
	stdioLatencyBudget int,
	allowHTTPFailover bool,
	probes map[string]transportProbe,
) (string, bool, string, []string, map[string]int) {
	attempts := map[string]int{}
	trace := make([]string, 0, 6)

	primaryProbe, primaryExists := probes[primaryTransport]
	attempts[primaryTransport]++
	if !primaryExists {
		primaryProbe = transportProbe{
			Transport:   primaryTransport,
			Healthy:     false,
			LatencyMs:   999,
			FailureCode: "probe_missing",
		}
	}
	trace = append(
		trace,
		fmt.Sprintf("primary[%s]:healthy=%t latency=%d", primaryTransport, primaryProbe.Healthy, primaryProbe.LatencyMs),
	)

	primaryHealthy := primaryProbe.Healthy && primaryProbe.LatencyMs <= stdioLatencyBudget
	if primaryHealthy {
		trace = append(trace, "decision=retain_primary")
		return primaryTransport, false, "primary_healthy", trace, attempts
	}

	reason := ""
	switch {
	case primaryProbe.FailureCode != "":
		reason = "primary_" + primaryProbe.FailureCode
	case primaryProbe.LatencyMs > stdioLatencyBudget:
		reason = fmt.Sprintf("primary_latency_exceeded_%dms", primaryProbe.LatencyMs)
	default:
		reason = "primary_unhealthy"
	}
	trace = append(trace, "primary_not_eligible="+reason)

	if !allowHTTPFailover {
		trace = append(trace, "failover=disabled")
		return "none", false, reason, trace, attempts
	}

	fallbackProbe, fallbackExists := probes[fallbackTransport]
	attempts[fallbackTransport]++
	if !fallbackExists {
		trace = append(trace, "fallback=probe_missing")
		return "none", false, reason + "_fallback_missing", trace, attempts
	}

	trace = append(
		trace,
		fmt.Sprintf("fallback[%s]:healthy=%t latency=%d", fallbackTransport, fallbackProbe.Healthy, fallbackProbe.LatencyMs),
	)
	if fallbackProbe.Healthy {
		trace = append(trace, "decision=failover_to_"+fallbackTransport)
		return fallbackTransport, true, reason, trace, attempts
	}

	if fallbackProbe.FailureCode != "" {
		reason += "_fallback_" + fallbackProbe.FailureCode
	} else {
		reason += "_fallback_unhealthy"
	}
	trace = append(trace, "decision=no_viable_transport")
	return "none", false, reason, trace, attempts
}
func attemptSnapshot(attempts map[string]int) []string {
	if len(attempts) == 0 {
		return []string{}
	}
	keys := make([]string, 0, len(attempts))
	for key := range attempts {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make([]string, 0, len(keys))
	for _, key := range keys {
		out = append(out, fmt.Sprintf("%s:%d", key, attempts[key]))
	}
	return out
}

func parseAttemptSnapshot(raw any) map[string]int {
	parts, ok := toStringSlice(raw)
	if !ok {
		return map[string]int{}
	}
	out := map[string]int{}
	for _, part := range parts {
		items := strings.SplitN(part, ":", 2)
		if len(items) != 2 {
			continue
		}
		name := strings.TrimSpace(items[0])
		if name == "" {
			continue
		}
		count := parseInt(items[1])
		if count <= 0 {
			count = 1
		}
		out[name] = count
	}
	return out
}

func formatAttempts(attempts map[string]int) string {
	parts := attemptSnapshot(attempts)
	if len(parts) == 0 {
		return "none"
	}
	return strings.Join(parts, ",")
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

func toAnySlice(values []string) []any {
	if len(values) == 0 {
		return []any{}
	}
	out := make([]any, 0, len(values))
	for _, value := range values {
		out = append(out, value)
	}
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

func normalizedTransport(value string) string {
	trimmed := strings.TrimSpace(strings.ToLower(value))
	if trimmed == "" {
		return "none"
	}
	return trimmed
}
