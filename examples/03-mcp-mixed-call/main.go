package main

import (
	"context"
	"fmt"

	"github.com/FelixSeptem/baymax/core/runner"
	"github.com/FelixSeptem/baymax/core/types"
	"github.com/FelixSeptem/baymax/tool/local"
)

type mixedModel struct{ calls int }

func (m *mixedModel) Generate(ctx context.Context, req types.ModelRequest) (types.ModelResponse, error) {
	m.calls++
	if m.calls == 1 {
		return types.ModelResponse{ToolCalls: []types.ToolCall{{CallID: "c1", Name: "local.mcp_proxy", Args: map[string]any{"tool": "fake.mcp"}}}}, nil
	}
	return types.ModelResponse{FinalAnswer: "mixed local+mcp done"}, nil
}

func (m *mixedModel) Stream(ctx context.Context, req types.ModelRequest, onEvent func(types.ModelEvent) error) error {
	return nil
}

type mcpProxyTool struct{}

func (t *mcpProxyTool) Name() string        { return "mcp_proxy" }
func (t *mcpProxyTool) Description() string { return "proxy to fake MCP" }
func (t *mcpProxyTool) JSONSchema() map[string]any {
	return map[string]any{"type": "object", "required": []any{"tool"}, "properties": map[string]any{"tool": map[string]any{"type": "string"}}}
}
func (t *mcpProxyTool) Invoke(ctx context.Context, args map[string]any) (types.ToolResult, error) {
	return types.ToolResult{Content: fmt.Sprintf("mcp-call=%v", args["tool"])}, nil
}

func main() {
	reg := local.NewRegistry()
	_, _ = reg.Register(&mcpProxyTool{})
	eng := runner.New(&mixedModel{}, runner.WithLocalRegistry(reg))
	res, err := eng.Run(context.Background(), types.RunRequest{Input: "run mixed"}, nil)
	if err != nil {
		panic(err)
	}
	fmt.Println(res.FinalAnswer)
}
