package main

import (
	"context"
	"fmt"

	"github.com/FelixSeptem/baymax/core/runner"
	"github.com/FelixSeptem/baymax/core/types"
	"github.com/FelixSeptem/baymax/tool/local"
)

type toolFirstModel struct {
	toolCalled bool
}

func (m *toolFirstModel) Generate(context.Context, types.ModelRequest) (types.ModelResponse, error) {
	if !m.toolCalled {
		m.toolCalled = true
		return types.ModelResponse{
			ToolCalls: []types.ToolCall{
				{
					CallID: "tool-template-call-1",
					Name:   "local.adapter_echo",
					Args:   map[string]any{"input": "hello-tool-adapter"},
				},
			},
		}, nil
	}
	return types.ModelResponse{FinalAnswer: "tool-adapter-template: done"}, nil
}

func (m *toolFirstModel) Stream(context.Context, types.ModelRequest, func(types.ModelEvent) error) error {
	return nil
}

type adapterEchoTool struct{}

func (adapterEchoTool) Name() string        { return "adapter_echo" }
func (adapterEchoTool) Description() string { return "echo input for adapter template" }
func (adapterEchoTool) JSONSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"input": map[string]any{"type": "string"},
		},
		"required": []any{"input"},
	}
}

func (adapterEchoTool) Invoke(_ context.Context, args map[string]any) (types.ToolResult, error) {
	return types.ToolResult{
		Content: fmt.Sprintf("echo=%v", args["input"]),
	}, nil
}

func main() {
	reg := local.NewRegistry()
	if _, err := reg.Register(adapterEchoTool{}); err != nil {
		panic(err)
	}
	eng := runner.New(&toolFirstModel{}, runner.WithLocalRegistry(reg))
	res, err := eng.Run(context.Background(), types.RunRequest{
		Input: "run tool adapter template",
	}, nil)
	if err != nil {
		panic(err)
	}
	fmt.Println(res.FinalAnswer)
}
