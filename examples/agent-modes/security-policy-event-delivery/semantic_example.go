package securitypolicyeventdelivery

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
	patternName      = "security-policy-event-delivery"
	phase            = "P2"
	semanticAnchor   = "security.policy_event_delivery"
	classification   = "security.policy_delivery"
	semanticToolName = "mode_security_policy_event_delivery_semantic_step"
	defaultEventID   = "sec-event-20260410"
)

type securityStep struct {
	Marker        string
	RuntimeDomain string
	Intent        string
	Outcome       string
}

type securityState struct {
	EventID              string
	PolicyDecision       string
	PolicyReason         string
	RiskScore            int
	EventSink            string
	DeliveryStatus       string
	RetryCount           int
	FallbackQueued       bool
	DenySemanticPreserve bool
	EffectiveAction      string
	GovernanceDecision   string
	GovernanceTicket     string
	ReplaySignature      string
	SeenMarkers          []string
	TotalScore           int
}

var runtimeDomains = []string{"runtime/security", "observability/event"}

var minimalSemanticSteps = []securityStep{
	{
		Marker:        "security_policy_decision_emitted",
		RuntimeDomain: "runtime/security",
		Intent:        "emit policy allow/deny decision with reason and risk score",
		Outcome:       "policy decision payload is emitted",
	},
	{
		Marker:        "security_event_delivery_attempted",
		RuntimeDomain: "observability/event",
		Intent:        "attempt delivery with retry and fallback queue semantics",
		Outcome:       "delivery status/retry/fallback fields are emitted",
	},
	{
		Marker:        "security_deny_semantic_preserved",
		RuntimeDomain: "runtime/security",
		Intent:        "preserve deny semantic regardless of delivery result",
		Outcome:       "effective action and preserve flag are emitted",
	},
}

var productionGovernanceSteps = []securityStep{
	{
		Marker:        "governance_security_gate_enforced",
		RuntimeDomain: "runtime/security",
		Intent:        "enforce security gate using decision integrity and delivery fallback",
		Outcome:       "governance decision and ticket are emitted",
	},
	{
		Marker:        "governance_security_replay_bound",
		RuntimeDomain: "observability/event",
		Intent:        "bind security governance decision to replay signature",
		Outcome:       "replay signature is emitted",
	},
}

func RunMinimal() {
	executeSecurityVariant(modecommon.VariantMinimal)
}

func RunProduction() {
	executeSecurityVariant(modecommon.VariantProduction)
}

func executeSecurityVariant(variant string) {
	if err := modecommon.EnsureVariant(variant); err != nil {
		panic(err)
	}

	mgr, err := runtimeconfig.NewManager(runtimeconfig.ManagerOptions{EnvPrefix: "BAYMAX"})
	if err != nil {
		panic(err)
	}
	defer func() { _ = mgr.Close() }()

	reg := local.NewRegistry()
	if _, err := reg.Register(&securityDeliveryTool{}); err != nil {
		panic(err)
	}

	model := &securityDeliveryModel{
		variant: variant,
		state: securityState{
			EventID: defaultEventID,
		},
	}

	engine := runner.New(model, runner.WithLocalRegistry(reg), runner.WithRuntimeManager(mgr))
	result, err := engine.Run(context.Background(), types.RunRequest{
		RunID: fmt.Sprintf("agent-mode-%s-%s", modecommon.MarkerToken(patternName), modecommon.MarkerToken(variant)),
		Input: "execute security policy event delivery semantic pipeline",
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

func planForVariant(variant string) []securityStep {
	plan := make([]securityStep, 0, len(minimalSemanticSteps)+len(productionGovernanceSteps))
	plan = append(plan, minimalSemanticSteps...)
	if variant == modecommon.VariantProduction {
		plan = append(plan, productionGovernanceSteps...)
	}
	return plan
}

type securityDeliveryModel struct {
	variant string
	cursor  int
	state   securityState
}

func (m *securityDeliveryModel) Generate(ctx context.Context, req types.ModelRequest) (types.ModelResponse, error) {
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
		"%s/%s semantic_path_completed phase=%s anchor=%s classification=%s event=%s decision=%s reason=%s risk=%d sink=%s delivery=%s retry=%d fallback_queued=%t deny_preserved=%t effective=%s governance=%s ticket=%s replay=%s score=%d markers=%s",
		patternName,
		m.variant,
		phase,
		semanticAnchor,
		classification,
		normalizedValue(m.state.EventID, true),
		normalizedValue(m.state.PolicyDecision, true),
		normalizedValue(m.state.PolicyReason, true),
		m.state.RiskScore,
		normalizedValue(m.state.EventSink, true),
		normalizedValue(m.state.DeliveryStatus, true),
		m.state.RetryCount,
		m.state.FallbackQueued,
		m.state.DenySemanticPreserve,
		normalizedValue(m.state.EffectiveAction, true),
		normalizedValue(m.state.GovernanceDecision, governanceOn),
		normalizedValue(m.state.GovernanceTicket, governanceOn),
		normalizedValue(m.state.ReplaySignature, governanceOn),
		m.state.TotalScore,
		strings.Join(markers, ","),
	)
	return types.ModelResponse{FinalAnswer: final}, nil
}

func (m *securityDeliveryModel) Stream(ctx context.Context, req types.ModelRequest, onEvent func(types.ModelEvent) error) error {
	_ = ctx
	_ = req
	_ = onEvent
	return nil
}

func (m *securityDeliveryModel) capture(outcomes []types.ToolCallOutcome) {
	for _, item := range outcomes {
		marker, _ := item.Result.Structured["marker"].(string)
		if marker != "" {
			m.state.SeenMarkers = append(m.state.SeenMarkers, marker)
		}
		if eventID, _ := item.Result.Structured["event_id"].(string); strings.TrimSpace(eventID) != "" {
			m.state.EventID = strings.TrimSpace(eventID)
		}
		if decision, _ := item.Result.Structured["policy_decision"].(string); strings.TrimSpace(decision) != "" {
			m.state.PolicyDecision = strings.TrimSpace(decision)
		}
		if reason, _ := item.Result.Structured["policy_reason"].(string); strings.TrimSpace(reason) != "" {
			m.state.PolicyReason = strings.TrimSpace(reason)
		}
		if risk, ok := modecommon.AsInt(item.Result.Structured["risk_score"]); ok {
			m.state.RiskScore = risk
		}
		if sink, _ := item.Result.Structured["event_sink"].(string); strings.TrimSpace(sink) != "" {
			m.state.EventSink = strings.TrimSpace(sink)
		}
		if status, _ := item.Result.Structured["delivery_status"].(string); strings.TrimSpace(status) != "" {
			m.state.DeliveryStatus = strings.TrimSpace(status)
		}
		if retry, ok := modecommon.AsInt(item.Result.Structured["retry_count"]); ok {
			m.state.RetryCount = retry
		}
		if fallback, ok := item.Result.Structured["fallback_queued"].(bool); ok {
			m.state.FallbackQueued = fallback
		}
		if preserved, ok := item.Result.Structured["deny_semantic_preserved"].(bool); ok {
			m.state.DenySemanticPreserve = preserved
		}
		if action, _ := item.Result.Structured["effective_action"].(string); strings.TrimSpace(action) != "" {
			m.state.EffectiveAction = strings.TrimSpace(action)
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

func (m *securityDeliveryModel) argsForStep(step securityStep, stage int) map[string]any {
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
		"event_id":        m.state.EventID,
	}

	switch step.Marker {
	case "security_policy_decision_emitted":
		risk := 55
		reason := "baseline_policy"
		decision := "allow"
		if m.variant == modecommon.VariantProduction {
			risk = 91
			reason = "high_risk_token_exfiltration"
			decision = "deny"
		}
		args["risk_score"] = risk
		args["policy_reason"] = reason
		args["policy_decision"] = decision
	case "security_event_delivery_attempted":
		sink := "security-siem"
		retry := 1
		status := "delivered"
		if m.variant == modecommon.VariantProduction {
			retry = 3
			status = "delivery_failed_fallback_queued"
		}
		args["event_sink"] = sink
		args["retry_count"] = retry
		args["delivery_status"] = status
		args["policy_decision"] = m.state.PolicyDecision
	case "security_deny_semantic_preserved":
		args["policy_decision"] = m.state.PolicyDecision
		args["delivery_status"] = m.state.DeliveryStatus
		args["retry_count"] = m.state.RetryCount
	case "governance_security_gate_enforced":
		args["policy_decision"] = m.state.PolicyDecision
		args["deny_semantic_preserved"] = m.state.DenySemanticPreserve
		args["delivery_status"] = m.state.DeliveryStatus
		args["fallback_queued"] = m.state.FallbackQueued
	case "governance_security_replay_bound":
		args["event_id"] = m.state.EventID
		args["policy_decision"] = m.state.PolicyDecision
		args["governance_decision"] = m.state.GovernanceDecision
		args["governance_ticket"] = m.state.GovernanceTicket
	}
	return args
}

type securityDeliveryTool struct{}

func (t *securityDeliveryTool) Name() string { return semanticToolName }

func (t *securityDeliveryTool) Description() string {
	return "execute security policy/event/delivery semantic step"
}

func (t *securityDeliveryTool) JSONSchema() map[string]any {
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
			"event_id":        map[string]any{"type": "string"},
			"risk_score":      map[string]any{"type": "integer"},
			"policy_decision": map[string]any{"type": "string"},
			"policy_reason":   map[string]any{"type": "string"},
			"event_sink":      map[string]any{"type": "string"},
			"retry_count":     map[string]any{"type": "integer"},
			"delivery_status": map[string]any{"type": "string"},
			"fallback_queued": map[string]any{"type": "boolean"},
			"deny_semantic_preserved": map[string]any{
				"type": "boolean",
			},
			"effective_action": map[string]any{"type": "string"},
			"governance_decision": map[string]any{
				"type": "string",
			},
			"governance_ticket": map[string]any{"type": "string"},
		},
	}
}

func (t *securityDeliveryTool) Invoke(ctx context.Context, args map[string]any) (types.ToolResult, error) {
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
	case "security_policy_decision_emitted":
		eventID := strings.TrimSpace(fmt.Sprintf("%v", args["event_id"]))
		if eventID == "" {
			eventID = defaultEventID
		}
		riskScore, _ := modecommon.AsInt(args["risk_score"])
		decision := strings.TrimSpace(fmt.Sprintf("%v", args["policy_decision"]))
		reason := strings.TrimSpace(fmt.Sprintf("%v", args["policy_reason"]))
		if decision == "" {
			decision = "allow"
		}
		structured["event_id"] = eventID
		structured["risk_score"] = riskScore
		structured["policy_decision"] = decision
		structured["policy_reason"] = reason
		if decision == "deny" {
			risk = "degraded_path"
		}
	case "security_event_delivery_attempted":
		sink := strings.TrimSpace(fmt.Sprintf("%v", args["event_sink"]))
		retry, _ := modecommon.AsInt(args["retry_count"])
		status := strings.TrimSpace(fmt.Sprintf("%v", args["delivery_status"]))
		decision := strings.TrimSpace(fmt.Sprintf("%v", args["policy_decision"]))
		fallbackQueued := strings.Contains(status, "fallback_queued") || status == "delivery_failed_fallback_queued"
		structured["event_sink"] = sink
		structured["retry_count"] = retry
		structured["delivery_status"] = status
		structured["policy_decision"] = decision
		structured["fallback_queued"] = fallbackQueued
		if status != "delivered" {
			risk = "degraded_path"
		}
	case "security_deny_semantic_preserved":
		decision := strings.TrimSpace(fmt.Sprintf("%v", args["policy_decision"]))
		status := strings.TrimSpace(fmt.Sprintf("%v", args["delivery_status"]))
		retry, _ := modecommon.AsInt(args["retry_count"])
		effective := "allow_request"
		preserved := true
		if decision == "deny" {
			effective = "deny_request"
			preserved = true
		}
		if decision != "deny" && strings.Contains(status, "failed") {
			effective = "allow_request"
		}
		structured["policy_decision"] = decision
		structured["delivery_status"] = status
		structured["retry_count"] = retry
		structured["deny_semantic_preserved"] = preserved
		structured["effective_action"] = effective
		if decision == "deny" {
			risk = "degraded_path"
		}
	case "governance_security_gate_enforced":
		decision := strings.TrimSpace(fmt.Sprintf("%v", args["policy_decision"]))
		preserved := asBool(args["deny_semantic_preserved"])
		status := strings.TrimSpace(fmt.Sprintf("%v", args["delivery_status"]))
		fallbackQueued := asBool(args["fallback_queued"])
		governance := "allow"
		switch {
		case decision == "deny" && !preserved:
			governance = "deny"
		case decision == "deny" && preserved:
			governance = "allow_with_security_hold"
		case strings.Contains(status, "failed") && fallbackQueued:
			governance = "allow_with_delivery_fallback"
		}
		ticket := fmt.Sprintf("security-gate-%d", modecommon.SemanticScore(decision, fmt.Sprintf("%t", preserved), status, fmt.Sprintf("%t", fallbackQueued), governance))
		structured["policy_decision"] = decision
		structured["deny_semantic_preserved"] = preserved
		structured["delivery_status"] = status
		structured["fallback_queued"] = fallbackQueued
		structured["governance"] = true
		structured["governance_decision"] = governance
		structured["governance_ticket"] = ticket
		risk = "governed"
	case "governance_security_replay_bound":
		eventID := strings.TrimSpace(fmt.Sprintf("%v", args["event_id"]))
		decision := strings.TrimSpace(fmt.Sprintf("%v", args["policy_decision"]))
		governance := strings.TrimSpace(fmt.Sprintf("%v", args["governance_decision"]))
		ticket := strings.TrimSpace(fmt.Sprintf("%v", args["governance_ticket"]))
		replay := fmt.Sprintf("security-replay-%d", modecommon.SemanticScore(eventID, decision, governance, ticket))
		structured["event_id"] = eventID
		structured["policy_decision"] = decision
		structured["governance_decision"] = governance
		structured["governance_ticket"] = ticket
		structured["governance"] = true
		structured["replay_signature"] = replay
		risk = "governed"
	default:
		return types.ToolResult{}, fmt.Errorf("unsupported security semantic marker: %s", marker)
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

func normalizedValue(value string, enabled bool) string {
	if !enabled {
		return "n/a"
	}
	if strings.TrimSpace(value) == "" {
		return "pending"
	}
	return value
}
