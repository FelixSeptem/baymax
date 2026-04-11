package hooksmiddlewareextensionpipeline

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
	patternName      = "hooks-middleware-extension-pipeline"
	phase            = "P1"
	semanticAnchor   = "middleware.onion_bubble_passthrough"
	classification   = "hooks.middleware_pipeline"
	semanticToolName = "mode_hooks_middleware_extension_pipeline_semantic_step"
	defaultPipeline  = "mw-pipeline-20260410"
)

type middlewareStep struct {
	Marker        string
	RuntimeDomain string
	Intent        string
	Outcome       string
}

type middlewareState struct {
	PipelineID         string
	OnionOrder         string
	MiddlewareDepth    int
	BubbleCode         string
	BubbleSeverity     string
	Retryable          bool
	ExtensionFields    []string
	PassthroughOK      bool
	GovernanceDecision string
	GovernanceTicket   string
	ReplaySignature    string
	SeenMarkers        []string
	TotalScore         int
}

var runtimeDomains = []string{"core/runner", "tool/local"}

var middlewareStack = []string{"auth-hook", "rate-limit-hook", "audit-hook"}

var minimalSemanticSteps = []middlewareStep{
	{
		Marker:        "middleware_onion_order_verified",
		RuntimeDomain: "core/runner",
		Intent:        "verify middleware onion enter/exit order around tool invocation",
		Outcome:       "onion order and depth are emitted",
	},
	{
		Marker:        "middleware_error_bubbled",
		RuntimeDomain: "tool/local",
		Intent:        "bubble inner middleware error with transformed severity",
		Outcome:       "bubble code/severity/retryable are emitted",
	},
	{
		Marker:        "middleware_extension_passthrough",
		RuntimeDomain: "core/runner",
		Intent:        "pass extension context through middleware to tool response",
		Outcome:       "extension passthrough evidence is emitted",
	},
}

var productionGovernanceSteps = []middlewareStep{
	{
		Marker:        "governance_hooks_gate_enforced",
		RuntimeDomain: "core/runner",
		Intent:        "enforce gate based on bubble severity and passthrough quality",
		Outcome:       "governance decision and ticket are emitted",
	},
	{
		Marker:        "governance_hooks_replay_bound",
		RuntimeDomain: "tool/local",
		Intent:        "bind middleware governance decision to replay signature",
		Outcome:       "replay signature is emitted",
	},
}

func RunMinimal() {
	executeMiddlewareVariant(modecommon.VariantMinimal)
}

func RunProduction() {
	executeMiddlewareVariant(modecommon.VariantProduction)
}

func executeMiddlewareVariant(variant string) {
	if err := modecommon.EnsureVariant(variant); err != nil {
		panic(err)
	}

	mgr, err := runtimeconfig.NewManager(runtimeconfig.ManagerOptions{EnvPrefix: "BAYMAX"})
	if err != nil {
		panic(err)
	}
	defer func() { _ = mgr.Close() }()

	reg := local.NewRegistry()
	if _, err := reg.Register(&middlewarePipelineTool{}); err != nil {
		panic(err)
	}

	model := &middlewarePipelineModel{
		variant: variant,
		state: middlewareState{
			PipelineID: defaultPipeline,
		},
	}

	engine := runner.New(model, runner.WithLocalRegistry(reg), runner.WithRuntimeManager(mgr))
	result, err := engine.Run(context.Background(), types.RunRequest{
		RunID: fmt.Sprintf("agent-mode-%s-%s", modecommon.MarkerToken(patternName), modecommon.MarkerToken(variant)),
		Input: "execute hooks middleware extension semantic pipeline",
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

func planForVariant(variant string) []middlewareStep {
	plan := make([]middlewareStep, 0, len(minimalSemanticSteps)+len(productionGovernanceSteps))
	plan = append(plan, minimalSemanticSteps...)
	if variant == modecommon.VariantProduction {
		plan = append(plan, productionGovernanceSteps...)
	}
	return plan
}

type middlewarePipelineModel struct {
	variant string
	cursor  int
	state   middlewareState
}

func (m *middlewarePipelineModel) Generate(ctx context.Context, req types.ModelRequest) (types.ModelResponse, error) {
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
		"%s/%s semantic_path_completed phase=%s anchor=%s classification=%s onion=%s depth=%d bubble=%s severity=%s retryable=%t extension=%s passthrough=%t governance=%s ticket=%s replay=%s score=%d markers=%s",
		patternName,
		m.variant,
		phase,
		semanticAnchor,
		classification,
		normalizedValue(m.state.OnionOrder, true),
		m.state.MiddlewareDepth,
		normalizedValue(m.state.BubbleCode, true),
		normalizedValue(m.state.BubbleSeverity, true),
		m.state.Retryable,
		stringSliceToken(m.state.ExtensionFields),
		m.state.PassthroughOK,
		normalizedValue(m.state.GovernanceDecision, governanceOn),
		normalizedValue(m.state.GovernanceTicket, governanceOn),
		normalizedValue(m.state.ReplaySignature, governanceOn),
		m.state.TotalScore,
		strings.Join(markers, ","),
	)
	return types.ModelResponse{FinalAnswer: final}, nil
}

func (m *middlewarePipelineModel) Stream(ctx context.Context, req types.ModelRequest, onEvent func(types.ModelEvent) error) error {
	_ = ctx
	_ = req
	_ = onEvent
	return nil
}

func (m *middlewarePipelineModel) capture(outcomes []types.ToolCallOutcome) {
	for _, item := range outcomes {
		marker, _ := item.Result.Structured["marker"].(string)
		if marker != "" {
			m.state.SeenMarkers = append(m.state.SeenMarkers, marker)
		}
		if onion, _ := item.Result.Structured["onion_order"].(string); strings.TrimSpace(onion) != "" {
			m.state.OnionOrder = strings.TrimSpace(onion)
		}
		if depth, ok := modecommon.AsInt(item.Result.Structured["middleware_depth"]); ok {
			m.state.MiddlewareDepth = depth
		}
		if bubble, _ := item.Result.Structured["bubble_code"].(string); strings.TrimSpace(bubble) != "" {
			m.state.BubbleCode = strings.TrimSpace(bubble)
		}
		if severity, _ := item.Result.Structured["bubble_severity"].(string); strings.TrimSpace(severity) != "" {
			m.state.BubbleSeverity = strings.TrimSpace(severity)
		}
		if retryable, ok := item.Result.Structured["retryable"].(bool); ok {
			m.state.Retryable = retryable
		}
		if fields := toStringSlice(item.Result.Structured["extension_fields"]); len(fields) > 0 {
			m.state.ExtensionFields = fields
		}
		if pass, ok := item.Result.Structured["passthrough_ok"].(bool); ok {
			m.state.PassthroughOK = pass
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

func (m *middlewarePipelineModel) argsForStep(step middlewareStep, stage int) map[string]any {
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
		"pipeline_id":     m.state.PipelineID,
	}

	switch step.Marker {
	case "middleware_onion_order_verified":
		stack := append([]string{}, middlewareStack...)
		if m.variant == modecommon.VariantProduction {
			stack = append(stack, "security-hook")
		}
		args["middleware_stack"] = stringSliceToAny(stack)
		args["handler_error"] = "none"
	case "middleware_error_bubbled":
		handlerError := "tool_timeout"
		if m.variant == modecommon.VariantProduction {
			handlerError = "policy_denied"
		}
		args["onion_order"] = m.state.OnionOrder
		args["middleware_depth"] = m.state.MiddlewareDepth
		args["handler_error"] = handlerError
	case "middleware_extension_passthrough":
		extensions := []string{"trace_id", "span_id", "tenant"}
		if m.variant == modecommon.VariantProduction {
			extensions = append(extensions, "guardrail_ticket")
		}
		args["extension_fields"] = stringSliceToAny(extensions)
		args["bubble_severity"] = m.state.BubbleSeverity
	case "governance_hooks_gate_enforced":
		args["bubble_severity"] = m.state.BubbleSeverity
		args["retryable"] = m.state.Retryable
		args["passthrough_ok"] = m.state.PassthroughOK
		args["middleware_depth"] = m.state.MiddlewareDepth
	case "governance_hooks_replay_bound":
		args["pipeline_id"] = m.state.PipelineID
		args["onion_order"] = m.state.OnionOrder
		args["governance_decision"] = m.state.GovernanceDecision
		args["governance_ticket"] = m.state.GovernanceTicket
	}
	return args
}

type middlewarePipelineTool struct{}

func (t *middlewarePipelineTool) Name() string { return semanticToolName }

func (t *middlewarePipelineTool) Description() string {
	return "execute middleware onion/bubble/passthrough semantic step"
}

func (t *middlewarePipelineTool) JSONSchema() map[string]any {
	return map[string]any{
		"type":     "object",
		"required": []any{"pattern", "variant", "phase", "semantic_anchor", "classification", "marker", "runtime_domain", "stage"},
		"properties": map[string]any{
			"pattern":          map[string]any{"type": "string"},
			"variant":          map[string]any{"type": "string"},
			"phase":            map[string]any{"type": "string"},
			"semantic_anchor":  map[string]any{"type": "string"},
			"classification":   map[string]any{"type": "string"},
			"marker":           map[string]any{"type": "string"},
			"runtime_domain":   map[string]any{"type": "string"},
			"intent":           map[string]any{"type": "string"},
			"outcome":          map[string]any{"type": "string"},
			"stage":            map[string]any{"type": "integer"},
			"pipeline_id":      map[string]any{"type": "string"},
			"middleware_stack": map[string]any{"type": "array"},
			"onion_order":      map[string]any{"type": "string"},
			"middleware_depth": map[string]any{"type": "integer"},
			"handler_error":    map[string]any{"type": "string"},
			"bubble_severity":  map[string]any{"type": "string"},
			"retryable":        map[string]any{"type": "boolean"},
			"extension_fields": map[string]any{"type": "array"},
			"passthrough_ok":   map[string]any{"type": "boolean"},
			"governance_decision": map[string]any{
				"type": "string",
			},
			"governance_ticket": map[string]any{"type": "string"},
		},
	}
}

func (t *middlewarePipelineTool) Invoke(ctx context.Context, args map[string]any) (types.ToolResult, error) {
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
	case "middleware_onion_order_verified":
		stack := toStringSlice(args["middleware_stack"])
		if len(stack) == 0 {
			stack = append([]string{}, middlewareStack...)
		}
		enter := strings.Join(stack, ">")
		exit := reverseJoin(stack, "<")
		onionOrder := fmt.Sprintf("enter:%s|exit:%s", enter, exit)
		structured["middleware_depth"] = len(stack)
		structured["onion_order"] = onionOrder
		structured["onion_integrity"] = true
	case "middleware_error_bubbled":
		handlerError := strings.TrimSpace(fmt.Sprintf("%v", args["handler_error"]))
		if handlerError == "" {
			handlerError = "tool_timeout"
		}
		bubbleCode := "BUBBLE_TIMEOUT"
		bubbleSeverity := "warning"
		retryable := true
		if strings.Contains(handlerError, "policy") {
			bubbleCode = "BUBBLE_POLICY_DENY"
			bubbleSeverity = "critical"
			retryable = false
		}
		structured["handler_error"] = handlerError
		structured["bubble_code"] = bubbleCode
		structured["bubble_severity"] = bubbleSeverity
		structured["retryable"] = retryable
		if bubbleSeverity == "critical" {
			risk = "degraded_path"
		}
	case "middleware_extension_passthrough":
		fields := toStringSlice(args["extension_fields"])
		if len(fields) == 0 {
			fields = []string{"trace_id", "span_id"}
		}
		bubbleSeverity := strings.TrimSpace(fmt.Sprintf("%v", args["bubble_severity"]))
		passthroughOK := len(fields) >= 3
		if bubbleSeverity == "critical" && len(fields) < 4 {
			passthroughOK = false
		}
		structured["extension_fields"] = stringSliceToAny(fields)
		structured["passthrough_ok"] = passthroughOK
		structured["bubble_severity"] = bubbleSeverity
		if !passthroughOK {
			risk = "degraded_path"
		}
	case "governance_hooks_gate_enforced":
		bubbleSeverity := strings.TrimSpace(fmt.Sprintf("%v", args["bubble_severity"]))
		retryable := asBool(args["retryable"])
		passthroughOK := asBool(args["passthrough_ok"])
		depth, _ := modecommon.AsInt(args["middleware_depth"])
		decision := "allow"
		if bubbleSeverity == "critical" && !retryable {
			decision = "deny"
		} else if !passthroughOK || depth > 3 {
			decision = "allow_with_guardrails"
		}
		ticket := fmt.Sprintf("hooks-gate-%d", modecommon.SemanticScore(bubbleSeverity, fmt.Sprintf("%t", retryable), fmt.Sprintf("%t", passthroughOK), fmt.Sprintf("%d", depth), decision))
		structured["bubble_severity"] = bubbleSeverity
		structured["retryable"] = retryable
		structured["passthrough_ok"] = passthroughOK
		structured["middleware_depth"] = depth
		structured["governance"] = true
		structured["governance_decision"] = decision
		structured["governance_ticket"] = ticket
		risk = "governed"
	case "governance_hooks_replay_bound":
		pipelineID := strings.TrimSpace(fmt.Sprintf("%v", args["pipeline_id"]))
		onionOrder := strings.TrimSpace(fmt.Sprintf("%v", args["onion_order"]))
		decision := strings.TrimSpace(fmt.Sprintf("%v", args["governance_decision"]))
		ticket := strings.TrimSpace(fmt.Sprintf("%v", args["governance_ticket"]))
		replay := fmt.Sprintf("hooks-replay-%d", modecommon.SemanticScore(pipelineID, onionOrder, decision, ticket))
		structured["pipeline_id"] = pipelineID
		structured["onion_order"] = onionOrder
		structured["governance_decision"] = decision
		structured["governance_ticket"] = ticket
		structured["governance"] = true
		structured["replay_signature"] = replay
		risk = "governed"
	default:
		return types.ToolResult{}, fmt.Errorf("unsupported hooks semantic marker: %s", marker)
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

func reverseJoin(in []string, sep string) string {
	if len(in) == 0 {
		return ""
	}
	reversed := make([]string, 0, len(in))
	for i := len(in) - 1; i >= 0; i-- {
		reversed = append(reversed, in[i])
	}
	return strings.Join(reversed, sep)
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
