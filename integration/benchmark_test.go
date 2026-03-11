package integration

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/FelixSeptem/baymax/core/runner"
	"github.com/FelixSeptem/baymax/core/types"
	"github.com/FelixSeptem/baymax/integration/fakes"
	httpmcp "github.com/FelixSeptem/baymax/mcp/http"
	"github.com/FelixSeptem/baymax/tool/local"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func BenchmarkIterationLatency(b *testing.B) {
	model := fakes.NewModel([]fakes.ModelStep{{Response: types.ModelResponse{FinalAnswer: "ok"}}})
	eng := runner.New(model)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = eng.Run(context.Background(), types.RunRequest{Input: "x"}, nil)
	}
}

func BenchmarkToolFanOut(b *testing.B) {
	reg := local.NewRegistry()
	_, _ = reg.Register(&fakes.Tool{NameValue: "echo", InvokeFn: func(ctx context.Context, args map[string]any) (types.ToolResult, error) {
		return types.ToolResult{Content: "ok"}, nil
	}})
	dispatcher := local.NewDispatcher(reg)
	calls := make([]types.ToolCall, 8)
	for i := range calls {
		calls[i] = types.ToolCall{CallID: "c", Name: "local.echo"}
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = dispatcher.Dispatch(context.Background(), calls, local.DispatchConfig{MaxCalls: 8, Concurrency: 8})
	}
}

func BenchmarkMCPReconnectOverhead(b *testing.B) {
	attempt := 0
	connector := func(ctx context.Context) (httpmcp.Session, error) {
		attempt++
		if attempt%2 == 1 {
			return &fakeHTTPSession{callErr: errors.New("down")}, nil
		}
		return &fakeHTTPSession{}, nil
	}
	client := httpmcp.NewClient(httpmcp.Config{Connect: connector, Retry: 1, Backoff: time.Microsecond})
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = client.CallTool(context.Background(), "tool", nil)
	}
}

type fakeHTTPSession struct{ callErr error }

func (s *fakeHTTPSession) ListTools(ctx context.Context, p *mcp.ListToolsParams) (*mcp.ListToolsResult, error) {
	return &mcp.ListToolsResult{}, nil
}
func (s *fakeHTTPSession) CallTool(ctx context.Context, p *mcp.CallToolParams) (*mcp.CallToolResult, error) {
	if s.callErr != nil {
		return nil, s.callErr
	}
	return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: "ok"}}}, nil
}
func (s *fakeHTTPSession) Close() error { return nil }
