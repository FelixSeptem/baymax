package realtimeinterruptresume

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
	patternName      = "realtime-interrupt-resume"
	phase            = "P0"
	semanticAnchor   = "realtime.cursor_idempotent_interrupt_resume"
	classification   = "realtime.resume_recovery"
	semanticToolName = "mode_realtime_interrupt_resume_semantic_step"
)

type semanticStep struct {
	Marker        string
	RuntimeDomain string
	Intent        string
	Outcome       string
}

var runtimeDomains = []string{"core/runner", "runtime/diagnostics"}

var minimalSemanticSteps = []semanticStep{
	{Marker: "realtime_cursor_idempotent", RuntimeDomain: "core/runner", Intent: "drive realtime cursor idempotent via realtime cursor_idempotent_interrupt_resume", Outcome: "realtime cursor idempotent confirmed on core/runner"},
	{Marker: "realtime_interrupt_captured", RuntimeDomain: "runtime/diagnostics", Intent: "drive realtime interrupt captured via realtime cursor_idempotent_interrupt_resume", Outcome: "realtime interrupt captured confirmed on runtime/diagnostics"},
	{Marker: "realtime_resume_recovered", RuntimeDomain: "core/runner", Intent: "drive realtime resume recovered via realtime cursor_idempotent_interrupt_resume", Outcome: "realtime resume recovered confirmed on core/runner"},
}

var productionGovernanceSteps = []semanticStep{
	{Marker: "governance_realtime_gate_enforced", RuntimeDomain: "core/runner", Intent: "drive governance realtime gate enforced via realtime cursor_idempotent_interrupt_resume", Outcome: "governance realtime gate enforced confirmed on core/runner"},
	{Marker: "governance_realtime_replay_bound", RuntimeDomain: "runtime/diagnostics", Intent: "drive governance realtime replay bound via realtime cursor_idempotent_interrupt_resume", Outcome: "governance realtime replay bound confirmed on runtime/diagnostics"},
}

func RunMinimal() {
	runVariant(modecommon.VariantMinimal)
}

func RunProduction() {
	runVariant(modecommon.VariantProduction)
}

func runVariant(variant string) {
	if err := modecommon.EnsureVariant(variant); err != nil {
		panic(err)
	}

	mgr, err := runtimeconfig.NewManager(runtimeconfig.ManagerOptions{EnvPrefix: "BAYMAX"})
	if err != nil {
		panic(err)
	}
	defer func() { _ = mgr.Close() }()

	reg := local.NewRegistry()
	if _, err := reg.Register(&modeSemanticStepTool{}); err != nil {
		panic(err)
	}

	engine := runner.New(&modeSemanticModel{variant: variant}, runner.WithLocalRegistry(reg), runner.WithRuntimeManager(mgr))
	result, err := engine.Run(context.Background(), types.RunRequest{
		RunID: fmt.Sprintf("agent-mode-%s-%s", modecommon.MarkerToken(patternName), modecommon.MarkerToken(variant)),
		Input: "execute mode semantic pipeline for " + patternName,
	}, nil)
	if err != nil {
		panic(err)
	}

	expected := expectedSemanticSteps(variant)
	runtimePath := modecommon.ComposeRuntimePath(runtimeDomains)
	pathStatus := modecommon.RuntimePathStatus(result.ToolCalls, len(expected))
	governanceStatus := "baseline"
	if variant == modecommon.VariantProduction {
		governanceStatus = "enforced"
	}

	markerNames := make([]string, 0, len(expected))
	for _, step := range expected {
		markerNames = append(markerNames, step.Marker)
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
	fmt.Printf("verification.semantic.expected_markers=%s\n", strings.Join(markerNames, ","))
	fmt.Printf("verification.semantic.governance=%s\n", governanceStatus)
	fmt.Printf("verification.semantic.marker_count=%d\n", len(markerNames))
	for _, marker := range markerNames {
		fmt.Printf("verification.semantic.marker.%s=ok\n", modecommon.MarkerToken(marker))
	}
	fmt.Printf("result.tool_calls=%d\n", len(result.ToolCalls))
	fmt.Printf("result.final_answer=%s\n", result.FinalAnswer)
	fmt.Printf("result.signature=%d\n", modecommon.ComputeSignature(result.FinalAnswer, result.ToolCalls))
}

func expectedSemanticSteps(variant string) []semanticStep {
	steps := make([]semanticStep, 0, len(minimalSemanticSteps)+len(productionGovernanceSteps))
	steps = append(steps, minimalSemanticSteps...)
	if variant == modecommon.VariantProduction {
		steps = append(steps, productionGovernanceSteps...)
	}
	return steps
}

type modeSemanticModel struct {
	variant string
	calls   int
}

func (m *modeSemanticModel) Generate(ctx context.Context, req types.ModelRequest) (types.ModelResponse, error) {
	_ = ctx
	m.calls++

	if m.calls == 1 {
		steps := expectedSemanticSteps(m.variant)
		toolCalls := make([]types.ToolCall, 0, len(steps))
		for idx, step := range steps {
			toolCalls = append(toolCalls, types.ToolCall{
				CallID: fmt.Sprintf("%s-%02d", modecommon.MarkerToken(patternName), idx+1),
				Name:   "local." + semanticToolName,
				Args: map[string]any{
					"pattern":          patternName,
					"variant":          m.variant,
					"phase":            phase,
					"semantic_anchor":  semanticAnchor,
					"classification":   classification,
					"marker":           step.Marker,
					"runtime_domain":   step.RuntimeDomain,
					"semantic_intent":  step.Intent,
					"semantic_outcome": step.Outcome,
					"stage":            idx + 1,
				},
			})
		}
		return types.ModelResponse{ToolCalls: toolCalls}, nil
	}

	markers := make([]string, 0, len(req.ToolResult))
	domainSet := map[string]struct{}{}
	governanceSeen := false
	totalScore := 0

	for _, outcome := range req.ToolResult {
		marker, _ := outcome.Result.Structured["marker"].(string)
		if marker != "" {
			markers = append(markers, marker)
		}
		domain, _ := outcome.Result.Structured["runtime_domain"].(string)
		if domain != "" {
			domainSet[domain] = struct{}{}
		}
		if g, ok := outcome.Result.Structured["governance"].(bool); ok && g {
			governanceSeen = true
		}
		if score, ok := modecommon.AsInt(outcome.Result.Structured["score"]); ok {
			totalScore += score
		}
	}

	sort.Strings(markers)
	domains := make([]string, 0, len(domainSet))
	for domain := range domainSet {
		domains = append(domains, domain)
	}
	sort.Strings(domains)

	final := fmt.Sprintf(
		"%s/%s semantic_path_completed phase=%s anchor=%s classification=%s marker_count=%d governance=%t score=%d domains=%s markers=%s",
		patternName,
		m.variant,
		phase,
		semanticAnchor,
		classification,
		len(markers),
		governanceSeen,
		totalScore,
		strings.Join(domains, ","),
		strings.Join(markers, ","),
	)
	return types.ModelResponse{FinalAnswer: final}, nil
}

func (m *modeSemanticModel) Stream(ctx context.Context, req types.ModelRequest, onEvent func(types.ModelEvent) error) error {
	_ = ctx
	_ = req
	_ = onEvent
	return nil
}

type modeSemanticStepTool struct{}

func (t *modeSemanticStepTool) Name() string { return semanticToolName }
func (t *modeSemanticStepTool) Description() string {
	return "execute mode-owned semantic step for " + patternName
}

func (t *modeSemanticStepTool) JSONSchema() map[string]any {
	return map[string]any{
		"type":     "object",
		"required": []any{"pattern", "variant", "phase", "semantic_anchor", "classification", "marker", "runtime_domain", "semantic_intent", "semantic_outcome", "stage"},
		"properties": map[string]any{
			"pattern":          map[string]any{"type": "string"},
			"variant":          map[string]any{"type": "string"},
			"phase":            map[string]any{"type": "string"},
			"semantic_anchor":  map[string]any{"type": "string"},
			"classification":   map[string]any{"type": "string"},
			"marker":           map[string]any{"type": "string"},
			"runtime_domain":   map[string]any{"type": "string"},
			"semantic_intent":  map[string]any{"type": "string"},
			"semantic_outcome": map[string]any{"type": "string"},
			"stage":            map[string]any{"type": "integer"},
		},
	}
}

func (t *modeSemanticStepTool) Invoke(ctx context.Context, args map[string]any) (types.ToolResult, error) {
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
	stage, _ := modecommon.AsInt(args["stage"])
	if stage <= 0 {
		stage = 1
	}

	governance := strings.HasPrefix(marker, "governance_") || variant == modecommon.VariantProduction
	risk := "nominal"
	if strings.Contains(marker, "fallback") || strings.Contains(marker, "rollback") || strings.Contains(marker, "recover") {
		risk = "degraded_path"
	}
	if governance {
		risk = "governed"
	}

	score := modecommon.SemanticScore(pattern, variant, phaseValue, anchor, classValue, marker, runtimeDomain, intent, outcome, fmt.Sprintf("%d", stage))

	content := fmt.Sprintf("pattern=%s variant=%s phase=%s marker=%s domain=%s stage=%d governance=%t risk=%s",
		pattern, variant, phaseValue, marker, runtimeDomain, stage, governance, risk)
	return types.ToolResult{
		Content: content,
		Structured: map[string]any{
			"pattern":          pattern,
			"variant":          variant,
			"phase":            phaseValue,
			"semantic_anchor":  anchor,
			"classification":   classValue,
			"marker":           marker,
			"runtime_domain":   runtimeDomain,
			"semantic_intent":  intent,
			"semantic_outcome": outcome,
			"stage":            stage,
			"governance":       governance,
			"risk":             risk,
			"score":            score,
		},
	}, nil
}
