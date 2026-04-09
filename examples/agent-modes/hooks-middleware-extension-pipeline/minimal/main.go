package main

import (
	"context"
	"fmt"
	"hash/fnv"
	"sort"
	"strings"

	"github.com/FelixSeptem/baymax/core/runner"
	"github.com/FelixSeptem/baymax/core/types"
	runtimeconfig "github.com/FelixSeptem/baymax/runtime/config"
	"github.com/FelixSeptem/baymax/tool/local"
)

const patternName = "hooks-middleware-extension-pipeline"
const variantName = "minimal"

type modeModel struct {
	calls int
}

func (m *modeModel) Generate(ctx context.Context, req types.ModelRequest) (types.ModelResponse, error) {
	_ = ctx
	m.calls++
	if m.calls == 1 {
		toolCalls := []types.ToolCall{
			{
				CallID: "primary",
				Name:   "local.mode_step",
				Args: map[string]any{
					"pattern":  patternName,
					"variant":  variantName,
					"stage":    "primary",
					"keywords": []any{"hooks", "middleware", "extension", "pipeline"},
				},
			},
		}
		if variantName == "production-ish" {
			toolCalls = append(toolCalls, types.ToolCall{
				CallID: "governance",
				Name:   "local.mode_step",
				Args: map[string]any{
					"pattern":  patternName,
					"variant":  variantName,
					"stage":    "governance",
					"keywords": []any{"contract", "gate", "replay"},
				},
			})
		}
		return types.ModelResponse{ToolCalls: toolCalls}, nil
	}

	totalScore := 0
	signals := make([]string, 0, len(req.ToolResult))
	for _, outcome := range req.ToolResult {
		score, ok := asInt(outcome.Result.Structured["score"])
		if ok {
			totalScore += score
		}
		signal, _ := outcome.Result.Structured["signal"].(string)
		signal = strings.TrimSpace(signal)
		if signal != "" {
			signals = append(signals, signal)
		}
	}
	sort.Strings(signals)
	final := fmt.Sprintf("%s/%s runtime_completed operations=%d score=%d signals=%s", patternName, variantName, len(req.ToolResult), totalScore, strings.Join(signals, ","))
	return types.ModelResponse{FinalAnswer: final}, nil
}

func (m *modeModel) Stream(ctx context.Context, req types.ModelRequest, onEvent func(types.ModelEvent) error) error {
	_ = ctx
	_ = req
	_ = onEvent
	return nil
}

type modeStepTool struct{}

func (t *modeStepTool) Name() string        { return "mode_step" }
func (t *modeStepTool) Description() string { return "execute deterministic agent-mode runtime step" }

func (t *modeStepTool) JSONSchema() map[string]any {
	return map[string]any{
		"type":     "object",
		"required": []any{"pattern", "variant", "stage", "keywords"},
		"properties": map[string]any{
			"pattern": map[string]any{"type": "string"},
			"variant": map[string]any{"type": "string"},
			"stage":   map[string]any{"type": "string"},
			"keywords": map[string]any{
				"type": "array",
				"items": map[string]any{
					"type": "string",
				},
			},
		},
	}
}

func (t *modeStepTool) Invoke(ctx context.Context, args map[string]any) (types.ToolResult, error) {
	_ = ctx
	pattern := strings.TrimSpace(fmt.Sprintf("%v", args["pattern"]))
	variant := strings.TrimSpace(fmt.Sprintf("%v", args["variant"]))
	stage := strings.TrimSpace(fmt.Sprintf("%v", args["stage"]))
	if stage == "" {
		stage = "primary"
	}
	keywords := extractKeywords(args)
	joined := strings.Join(keywords, ":")
	score := len(joined) + len(pattern)
	if variant == "production-ish" {
		score += 9
	}
	if stage == "governance" {
		score += 13
	}
	signal := strings.ReplaceAll(pattern, "-", "_") + "." + stage
	content := fmt.Sprintf("pattern=%s stage=%s keyword_count=%d score=%d", pattern, stage, len(keywords), score)
	return types.ToolResult{
		Content: content,
		Structured: map[string]any{
			"pattern":       pattern,
			"variant":       variant,
			"stage":         stage,
			"keyword_count": len(keywords),
			"score":         score,
			"signal":        signal,
		},
	}, nil
}

func extractKeywords(args map[string]any) []string {
	raw, ok := args["keywords"].([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(raw))
	for _, item := range raw {
		text := strings.ToLower(strings.TrimSpace(fmt.Sprintf("%v", item)))
		if text == "" {
			continue
		}
		out = append(out, text)
	}
	sort.Strings(out)
	return out
}

func asInt(value any) (int, bool) {
	switch tv := value.(type) {
	case int:
		return tv, true
	case int64:
		return int(tv), true
	case float64:
		return int(tv), true
	default:
		return 0, false
	}
}

func computeSignature(finalAnswer string, calls []types.ToolCallSummary) uint64 {
	h := fnv.New64a()
	_, _ = h.Write([]byte(finalAnswer))
	for _, call := range calls {
		_, _ = h.Write([]byte("|"))
		_, _ = h.Write([]byte(call.Name))
		if call.Error != nil {
			_, _ = h.Write([]byte(call.Error.Class))
			_, _ = h.Write([]byte(call.Error.Message))
		}
	}
	return h.Sum64()
}

func main() {
	mgr, err := runtimeconfig.NewManager(runtimeconfig.ManagerOptions{EnvPrefix: "BAYMAX"})
	if err != nil {
		panic(err)
	}
	defer func() { _ = mgr.Close() }()

	reg := local.NewRegistry()
	if _, err := reg.Register(&modeStepTool{}); err != nil {
		panic(err)
	}

	engine := runner.New(&modeModel{}, runner.WithLocalRegistry(reg), runner.WithRuntimeManager(mgr))
	result, err := engine.Run(context.Background(), types.RunRequest{
		RunID: fmt.Sprintf("agent-mode-%s-%s", strings.ReplaceAll(patternName, "-", "_"), strings.ReplaceAll(variantName, "-", "_")),
		Input: "execute " + patternName,
	}, nil)
	if err != nil {
		panic(err)
	}

	pathStatus := "failed"
	if len(result.ToolCalls) > 0 {
		pathStatus = "ok"
	}

	fmt.Println("agent-mode example")
	fmt.Printf("pattern=%s\n", patternName)
	fmt.Printf("variant=%s\n", variantName)
	fmt.Printf("runtime.path=core/runner,tool/local,runtime/config\n")
	fmt.Printf("verification.mainline_runtime_path=%s\n", pathStatus)
	fmt.Printf("result.tool_calls=%d\n", len(result.ToolCalls))
	fmt.Printf("result.final_answer=%s\n", result.FinalAnswer)
	fmt.Printf("result.signature=%d\n", computeSignature(result.FinalAnswer, result.ToolCalls))
}
