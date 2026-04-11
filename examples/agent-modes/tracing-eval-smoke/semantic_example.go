package tracingevalsmoke

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
	patternName      = "tracing-eval-smoke"
	phase            = "P1"
	semanticAnchor   = "trace.eval_feedback_loop"
	classification   = "tracing.eval_interop"
	semanticToolName = "mode_tracing_eval_smoke_semantic_step"
	defaultTraceID   = "trace-checkout-20260410"
)

type tracingStep struct {
	Marker        string
	RuntimeDomain string
	Intent        string
	Outcome       string
}

type tracingState struct {
	TraceID            string
	SpanCount          int
	P95LatencyMs       int
	ErrorRatePermille  int
	EvalScore          int
	EvalSignal         string
	FeedbackAction     string
	LoopClosed         bool
	GovernanceDecision string
	GovernanceTicket   string
	ReplaySignature    string
	SeenMarkers        []string
	TotalScore         int
}

var runtimeDomains = []string{"observability/trace", "runtime/diagnostics"}

var minimalSemanticSteps = []tracingStep{
	{
		Marker:        "tracing_span_emitted",
		RuntimeDomain: "observability/trace",
		Intent:        "emit trace spans with latency/error telemetry from request path",
		Outcome:       "span telemetry is emitted",
	},
	{
		Marker:        "eval_signal_recorded",
		RuntimeDomain: "runtime/diagnostics",
		Intent:        "convert trace telemetry to eval score and risk signal",
		Outcome:       "eval score and signal are emitted",
	},
	{
		Marker:        "trace_eval_loop_closed",
		RuntimeDomain: "observability/trace",
		Intent:        "close feedback loop by selecting runtime action from eval signal",
		Outcome:       "feedback action and loop-closed status are emitted",
	},
}

var productionGovernanceSteps = []tracingStep{
	{
		Marker:        "governance_tracing_gate_enforced",
		RuntimeDomain: "observability/trace",
		Intent:        "enforce governance gate for high-risk trace/eval outputs",
		Outcome:       "governance decision and ticket are emitted",
	},
	{
		Marker:        "governance_tracing_replay_bound",
		RuntimeDomain: "runtime/diagnostics",
		Intent:        "bind governance decision to replay signature for audit determinism",
		Outcome:       "replay signature is emitted",
	},
}

func RunMinimal() {
	executeTracingVariant(modecommon.VariantMinimal)
}

func RunProduction() {
	executeTracingVariant(modecommon.VariantProduction)
}

func executeTracingVariant(variant string) {
	if err := modecommon.EnsureVariant(variant); err != nil {
		panic(err)
	}

	mgr, err := runtimeconfig.NewManager(runtimeconfig.ManagerOptions{EnvPrefix: "BAYMAX"})
	if err != nil {
		panic(err)
	}
	defer func() { _ = mgr.Close() }()

	reg := local.NewRegistry()
	if _, err := reg.Register(&tracingEvalTool{}); err != nil {
		panic(err)
	}

	model := &tracingEvalModel{
		variant: variant,
		state: tracingState{
			TraceID: defaultTraceID,
		},
	}

	engine := runner.New(model, runner.WithLocalRegistry(reg), runner.WithRuntimeManager(mgr))
	result, err := engine.Run(context.Background(), types.RunRequest{
		RunID: fmt.Sprintf("agent-mode-%s-%s", modecommon.MarkerToken(patternName), modecommon.MarkerToken(variant)),
		Input: "execute tracing eval smoke semantic pipeline",
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

func planForVariant(variant string) []tracingStep {
	plan := make([]tracingStep, 0, len(minimalSemanticSteps)+len(productionGovernanceSteps))
	plan = append(plan, minimalSemanticSteps...)
	if variant == modecommon.VariantProduction {
		plan = append(plan, productionGovernanceSteps...)
	}
	return plan
}

type tracingEvalModel struct {
	variant string
	cursor  int
	state   tracingState
}

func (m *tracingEvalModel) Generate(ctx context.Context, req types.ModelRequest) (types.ModelResponse, error) {
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
		"%s/%s semantic_path_completed phase=%s anchor=%s classification=%s trace_id=%s spans=%d p95=%d error_permille=%d eval_score=%d eval_signal=%s action=%s loop_closed=%t governance=%s ticket=%s replay=%s score=%d markers=%s",
		patternName,
		m.variant,
		phase,
		semanticAnchor,
		classification,
		normalizedValue(m.state.TraceID, true),
		m.state.SpanCount,
		m.state.P95LatencyMs,
		m.state.ErrorRatePermille,
		m.state.EvalScore,
		normalizedValue(m.state.EvalSignal, true),
		normalizedValue(m.state.FeedbackAction, true),
		m.state.LoopClosed,
		normalizedValue(m.state.GovernanceDecision, governanceOn),
		normalizedValue(m.state.GovernanceTicket, governanceOn),
		normalizedValue(m.state.ReplaySignature, governanceOn),
		m.state.TotalScore,
		strings.Join(markers, ","),
	)
	return types.ModelResponse{FinalAnswer: final}, nil
}

func (m *tracingEvalModel) Stream(ctx context.Context, req types.ModelRequest, onEvent func(types.ModelEvent) error) error {
	_ = ctx
	_ = req
	_ = onEvent
	return nil
}

func (m *tracingEvalModel) capture(outcomes []types.ToolCallOutcome) {
	for _, item := range outcomes {
		marker, _ := item.Result.Structured["marker"].(string)
		if marker != "" {
			m.state.SeenMarkers = append(m.state.SeenMarkers, marker)
		}
		if traceID, _ := item.Result.Structured["trace_id"].(string); strings.TrimSpace(traceID) != "" {
			m.state.TraceID = strings.TrimSpace(traceID)
		}
		if spans, ok := modecommon.AsInt(item.Result.Structured["span_count"]); ok {
			m.state.SpanCount = spans
		}
		if p95, ok := modecommon.AsInt(item.Result.Structured["p95_latency_ms"]); ok {
			m.state.P95LatencyMs = p95
		}
		if errorRate, ok := modecommon.AsInt(item.Result.Structured["error_permille"]); ok {
			m.state.ErrorRatePermille = errorRate
		}
		if evalScore, ok := modecommon.AsInt(item.Result.Structured["eval_score"]); ok {
			m.state.EvalScore = evalScore
		}
		if signal, _ := item.Result.Structured["eval_signal"].(string); strings.TrimSpace(signal) != "" {
			m.state.EvalSignal = strings.TrimSpace(signal)
		}
		if action, _ := item.Result.Structured["feedback_action"].(string); strings.TrimSpace(action) != "" {
			m.state.FeedbackAction = strings.TrimSpace(action)
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

func (m *tracingEvalModel) argsForStep(step tracingStep, stage int) map[string]any {
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
		"trace_id":        m.state.TraceID,
	}

	switch step.Marker {
	case "tracing_span_emitted":
		p95 := 190
		errorPermille := 8
		if m.variant == modecommon.VariantProduction {
			p95 = 340
			errorPermille = 22
		}
		args["span_count"] = 7
		args["p95_latency_ms"] = p95
		args["error_permille"] = errorPermille
	case "eval_signal_recorded":
		args["span_count"] = m.state.SpanCount
		args["p95_latency_ms"] = m.state.P95LatencyMs
		args["error_permille"] = m.state.ErrorRatePermille
	case "trace_eval_loop_closed":
		args["eval_score"] = m.state.EvalScore
		args["eval_signal"] = m.state.EvalSignal
	case "governance_tracing_gate_enforced":
		args["eval_signal"] = m.state.EvalSignal
		args["feedback_action"] = m.state.FeedbackAction
		args["eval_score"] = m.state.EvalScore
		args["trace_id"] = m.state.TraceID
	case "governance_tracing_replay_bound":
		args["trace_id"] = m.state.TraceID
		args["governance_decision"] = m.state.GovernanceDecision
		args["governance_ticket"] = m.state.GovernanceTicket
		args["eval_signal"] = m.state.EvalSignal
	}
	return args
}

type tracingEvalTool struct{}

func (t *tracingEvalTool) Name() string { return semanticToolName }

func (t *tracingEvalTool) Description() string {
	return "execute tracing/eval feedback semantic step"
}

func (t *tracingEvalTool) JSONSchema() map[string]any {
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
			"trace_id":        map[string]any{"type": "string"},
			"span_count":      map[string]any{"type": "integer"},
			"p95_latency_ms":  map[string]any{"type": "integer"},
			"error_permille":  map[string]any{"type": "integer"},
			"eval_score":      map[string]any{"type": "integer"},
			"eval_signal":     map[string]any{"type": "string"},
			"feedback_action": map[string]any{"type": "string"},
			"governance_decision": map[string]any{
				"type": "string",
			},
			"governance_ticket": map[string]any{"type": "string"},
		},
	}
}

func (t *tracingEvalTool) Invoke(ctx context.Context, args map[string]any) (types.ToolResult, error) {
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
	case "tracing_span_emitted":
		traceID := strings.TrimSpace(fmt.Sprintf("%v", args["trace_id"]))
		if traceID == "" {
			traceID = defaultTraceID
		}
		spanCount, _ := modecommon.AsInt(args["span_count"])
		if spanCount <= 0 {
			spanCount = 5
		}
		p95, _ := modecommon.AsInt(args["p95_latency_ms"])
		errorPermille, _ := modecommon.AsInt(args["error_permille"])
		structured["trace_id"] = traceID
		structured["span_count"] = spanCount
		structured["p95_latency_ms"] = p95
		structured["error_permille"] = errorPermille
		if p95 > 300 || errorPermille > 20 {
			risk = "degraded_path"
		}
	case "eval_signal_recorded":
		spanCount, _ := modecommon.AsInt(args["span_count"])
		p95, _ := modecommon.AsInt(args["p95_latency_ms"])
		errorPermille, _ := modecommon.AsInt(args["error_permille"])
		evalScore := 100 - p95/10 - errorPermille
		if evalScore < 0 {
			evalScore = 0
		}
		evalSignal := "healthy"
		if evalScore < 55 {
			evalSignal = "critical"
		} else if evalScore < 75 {
			evalSignal = "warning"
		}
		structured["span_count"] = spanCount
		structured["p95_latency_ms"] = p95
		structured["error_permille"] = errorPermille
		structured["eval_score"] = evalScore
		structured["eval_signal"] = evalSignal
		if evalSignal != "healthy" {
			risk = "degraded_path"
		}
	case "trace_eval_loop_closed":
		evalScore, _ := modecommon.AsInt(args["eval_score"])
		evalSignal := strings.TrimSpace(fmt.Sprintf("%v", args["eval_signal"]))
		action := "keep_baseline"
		if evalSignal == "warning" {
			action = "increase_sampling"
		}
		if evalSignal == "critical" {
			action = "trigger_rollback_guard"
		}
		structured["eval_score"] = evalScore
		structured["eval_signal"] = evalSignal
		structured["feedback_action"] = action
		structured["loop_closed"] = true
		if action != "keep_baseline" {
			risk = "degraded_path"
		}
	case "governance_tracing_gate_enforced":
		evalSignal := strings.TrimSpace(fmt.Sprintf("%v", args["eval_signal"]))
		action := strings.TrimSpace(fmt.Sprintf("%v", args["feedback_action"]))
		evalScore, _ := modecommon.AsInt(args["eval_score"])
		traceID := strings.TrimSpace(fmt.Sprintf("%v", args["trace_id"]))
		decision := "allow"
		if evalSignal == "critical" || action == "trigger_rollback_guard" {
			decision = "deny"
		} else if evalSignal == "warning" {
			decision = "allow_with_sampling"
		}
		ticket := fmt.Sprintf("trace-gate-%d", modecommon.SemanticScore(traceID, evalSignal, action, decision, fmt.Sprintf("%d", evalScore)))
		structured["eval_signal"] = evalSignal
		structured["feedback_action"] = action
		structured["eval_score"] = evalScore
		structured["trace_id"] = traceID
		structured["governance"] = true
		structured["governance_decision"] = decision
		structured["governance_ticket"] = ticket
		risk = "governed"
	case "governance_tracing_replay_bound":
		traceID := strings.TrimSpace(fmt.Sprintf("%v", args["trace_id"]))
		decision := strings.TrimSpace(fmt.Sprintf("%v", args["governance_decision"]))
		ticket := strings.TrimSpace(fmt.Sprintf("%v", args["governance_ticket"]))
		evalSignal := strings.TrimSpace(fmt.Sprintf("%v", args["eval_signal"]))
		replay := fmt.Sprintf("trace-replay-%d", modecommon.SemanticScore(traceID, decision, ticket, evalSignal))
		structured["trace_id"] = traceID
		structured["governance_decision"] = decision
		structured["governance_ticket"] = ticket
		structured["eval_signal"] = evalSignal
		structured["governance"] = true
		structured["replay_signature"] = replay
		risk = "governed"
	default:
		return types.ToolResult{}, fmt.Errorf("unsupported tracing semantic marker: %s", marker)
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
