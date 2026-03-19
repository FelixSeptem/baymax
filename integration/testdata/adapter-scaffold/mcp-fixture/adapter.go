package fixture

import (
	"context"
	"fmt"
	"time"

	"github.com/FelixSeptem/baymax/core/types"
	mcpstdio "github.com/FelixSeptem/baymax/mcp/stdio"
)

type FixtureMcpAdapterTransport struct{}

func (FixtureMcpAdapterTransport) Initialize(context.Context) error { return nil }

func (FixtureMcpAdapterTransport) ListTools(context.Context) ([]types.MCPToolMeta, error) {
	return []types.MCPToolMeta{
		{
			Name:        "echo",
			Description: "generated mcp adapter placeholder",
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

func (FixtureMcpAdapterTransport) CallTool(_ context.Context, name string, args map[string]any) (mcpstdio.Response, error) {
	raw, ok := args["input"]
	if !ok {
		return mcpstdio.Response{Error: "missing required input"}, nil
	}
	input, ok := raw.(string)
	if !ok {
		return mcpstdio.Response{Error: "missing required input"}, nil
	}
	return mcpstdio.Response{
		Content: fmt.Sprintf("tool=%s input=%s", name, input),
		Structured: map[string]any{
			"tool":  name,
			"input": input,
		},
	}, nil
}

func (FixtureMcpAdapterTransport) Close() error { return nil }

func NewFixtureMcpAdapterClient() *mcpstdio.Client {
	return mcpstdio.NewClient(FixtureMcpAdapterTransport{}, mcpstdio.Config{
		ReadPoolSize:  1,
		WritePoolSize: 1,
		CallTimeout:   2 * time.Second,
		Retry:         0,
		Backoff:       time.Millisecond,
		QueueSize:     8,
		Backpressure:  types.BackpressureBlock,
	})
}
