package main

import (
	"context"
	"fmt"
	"time"

	"github.com/FelixSeptem/baymax/core/types"
	mcpstdio "github.com/FelixSeptem/baymax/mcp/stdio"
)

// fakeTransport is a template placeholder for external MCP transport integration.
type fakeTransport struct{}

func (fakeTransport) Initialize(context.Context) error { return nil }

func (fakeTransport) ListTools(context.Context) ([]types.MCPToolMeta, error) {
	return []types.MCPToolMeta{
		{
			Name:        "echo",
			Description: "echo user input",
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

func (fakeTransport) CallTool(_ context.Context, name string, args map[string]any) (mcpstdio.Response, error) {
	return mcpstdio.Response{
		Content: fmt.Sprintf("tool=%s input=%v", name, args["input"]),
		Structured: map[string]any{
			"tool":  name,
			"input": args["input"],
		},
	}, nil
}

func (fakeTransport) Close() error { return nil }

func main() {
	client := mcpstdio.NewClient(fakeTransport{}, mcpstdio.Config{
		ReadPoolSize:  1,
		WritePoolSize: 1,
		CallTimeout:   2 * time.Second,
		Retry:         0,
		Backoff:       time.Millisecond,
		QueueSize:     16,
		Backpressure:  types.BackpressureBlock,
	})
	defer func() { _ = client.Close() }()

	res, err := client.CallTool(context.Background(), "echo", map[string]any{"input": "hello-adapter"})
	if err != nil {
		panic(err)
	}
	fmt.Println(res.Content)
}
