package fixture

import (
	"context"
	"fmt"
	"strings"

	"github.com/FelixSeptem/baymax/core/types"
)

type FixtureToolAdapter struct{}

func (FixtureToolAdapter) Name() string { return "local.fixture" }

func (FixtureToolAdapter) Description() string {
	return "generated tool adapter placeholder"
}

func (FixtureToolAdapter) JSONSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"input": map[string]any{"type": "string"},
		},
		"required": []any{"input"},
	}
}

func (FixtureToolAdapter) Invoke(_ context.Context, args map[string]any) (types.ToolResult, error) {
	raw, ok := args["input"]
	if !ok {
		return types.ToolResult{}, fmt.Errorf("missing required input")
	}
	input, ok := raw.(string)
	if !ok || strings.TrimSpace(input) == "" {
		return types.ToolResult{}, fmt.Errorf("missing required input")
	}
	return types.ToolResult{
		Content: fmt.Sprintf("echo=%s", input),
	}, nil
}
