package offline

import (
	"context"
	"fmt"
	"strings"

	"github.com/FelixSeptem/baymax/core/types"
)

type OfflineToolAdapter struct{}

func (OfflineToolAdapter) Name() string { return "local.offline" }

func (OfflineToolAdapter) Description() string {
	return "generated tool adapter placeholder"
}

func (OfflineToolAdapter) JSONSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"input": map[string]any{"type": "string"},
		},
		"required": []any{"input"},
	}
}

func (OfflineToolAdapter) Invoke(_ context.Context, args map[string]any) (types.ToolResult, error) {
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
