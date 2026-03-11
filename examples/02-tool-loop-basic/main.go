package main

import (
	"context"
	"fmt"

	"github.com/FelixSeptem/baymax/core/runner"
	"github.com/FelixSeptem/baymax/core/types"
	"github.com/FelixSeptem/baymax/tool/local"
)

type toolLoopModel struct{ calls int }

func (m *toolLoopModel) Generate(ctx context.Context, req types.ModelRequest) (types.ModelResponse, error) {
	m.calls++
	if m.calls == 1 {
		return types.ModelResponse{ToolCalls: []types.ToolCall{{CallID: "c1", Name: "local.echo", Args: map[string]any{"q": "tool-loop"}}}}, nil
	}
	return types.ModelResponse{FinalAnswer: "tool result received"}, nil
}

func (m *toolLoopModel) Stream(ctx context.Context, req types.ModelRequest, onEvent func(types.ModelEvent) error) error {
	return nil
}

type echoTool struct{}

func (t *echoTool) Name() string        { return "echo" }
func (t *echoTool) Description() string { return "echo text" }
func (t *echoTool) JSONSchema() map[string]any {
	return map[string]any{"type": "object", "required": []any{"q"}, "properties": map[string]any{"q": map[string]any{"type": "string"}}}
}
func (t *echoTool) Invoke(ctx context.Context, args map[string]any) (types.ToolResult, error) {
	return types.ToolResult{Content: fmt.Sprintf("echo=%v", args["q"])}, nil
}

func main() {
	reg := local.NewRegistry()
	_, _ = reg.Register(&echoTool{})
	eng := runner.New(&toolLoopModel{}, runner.WithLocalRegistry(reg))
	res, err := eng.Run(context.Background(), types.RunRequest{Input: "run tool"}, nil)
	if err != nil {
		panic(err)
	}
	fmt.Println(res.FinalAnswer)
}
