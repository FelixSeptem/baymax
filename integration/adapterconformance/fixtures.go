package adapterconformance

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/FelixSeptem/baymax/core/runner"
	"github.com/FelixSeptem/baymax/core/types"
	mcpstdio "github.com/FelixSeptem/baymax/mcp/stdio"
)

type fixtureMCPTransport struct{}

func (fixtureMCPTransport) Initialize(context.Context) error { return nil }

func (fixtureMCPTransport) ListTools(context.Context) ([]types.MCPToolMeta, error) {
	return []types.MCPToolMeta{
		{
			Name:        "echo",
			Description: "adapter conformance fixture tool",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"input": map[string]any{"type": "string"},
				},
				"required": []any{"input"},
			},
		},
	}, nil
}

func (fixtureMCPTransport) CallTool(_ context.Context, name string, args map[string]any) (mcpstdio.Response, error) {
	input, err := MustRequireStringArg(args, "input", types.ErrMCP, "mcp.validation.missing_required_input")
	if err != nil {
		return mcpstdio.Response{Error: err.Error()}, nil
	}
	return mcpstdio.Response{
		Content: fmt.Sprintf("tool=%s input=%s", name, input),
		Structured: map[string]any{
			"tool":  name,
			"input": input,
		},
	}, nil
}

func (fixtureMCPTransport) Close() error { return nil }

func newOfflineMCPClient() *mcpstdio.Client {
	return mcpstdio.NewClient(fixtureMCPTransport{}, mcpstdio.Config{
		ReadPoolSize:  1,
		WritePoolSize: 1,
		CallTimeout:   2 * time.Second,
		Retry:         0,
		Backoff:       time.Millisecond,
		QueueSize:     8,
		Backpressure:  types.BackpressureBlock,
	})
}

type equivalentModelAdapter struct{}

func (equivalentModelAdapter) Generate(context.Context, types.ModelRequest) (types.ModelResponse, error) {
	return types.ModelResponse{
		FinalAnswer: "adapter conformance semantic result",
	}, nil
}

func (equivalentModelAdapter) Stream(_ context.Context, _ types.ModelRequest, onEvent func(types.ModelEvent) error) error {
	return onEvent(types.ModelEvent{
		Type:      types.ModelEventTypeFinalAnswer,
		TextDelta: "adapter conformance semantic result",
	})
}

type malformedModelAdapter struct{}

func (malformedModelAdapter) Generate(context.Context, types.ModelRequest) (types.ModelResponse, error) {
	return types.ModelResponse{}, nil
}

func (malformedModelAdapter) Stream(context.Context, types.ModelRequest, func(types.ModelEvent) error) error {
	return nil
}

type requiredInputToolAdapter struct{}

func (requiredInputToolAdapter) Name() string        { return "adapter_required_input" }
func (requiredInputToolAdapter) Description() string { return "tool fixture for adapter conformance" }
func (requiredInputToolAdapter) JSONSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"input": map[string]any{"type": "string"},
		},
		"required": []any{"input"},
	}
}

func (requiredInputToolAdapter) Invoke(_ context.Context, args map[string]any) (types.ToolResult, error) {
	input, err := MustRequireStringArg(args, "input", types.ErrTool, "tool.validation.missing_required_input")
	if err != nil {
		return types.ToolResult{}, err
	}
	return types.ToolResult{
		Content: fmt.Sprintf("echo=%s", input),
	}, nil
}

func runAndStreamFinalAnswer(ctx context.Context, model types.ModelClient, input string) (string, string, error) {
	eng := runner.New(model)
	runRes, err := eng.Run(ctx, types.RunRequest{Input: input}, nil)
	if err != nil {
		return "", "", err
	}
	streamRes, err := eng.Stream(ctx, types.RunRequest{Input: input}, nil)
	if err != nil {
		return "", "", err
	}
	return runRes.FinalAnswer, streamRes.FinalAnswer, nil
}

func containsTemplatePath(doc string, path string) bool {
	return strings.Contains(doc, path)
}
