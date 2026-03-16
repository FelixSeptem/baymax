package integration

import (
	"context"
	"errors"
	"runtime"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/FelixSeptem/baymax/context/assembler"
	"github.com/FelixSeptem/baymax/core/runner"
	"github.com/FelixSeptem/baymax/core/types"
	"github.com/FelixSeptem/baymax/integration/fakes"
	httpmcp "github.com/FelixSeptem/baymax/mcp/http"
	mcpprofile "github.com/FelixSeptem/baymax/mcp/profile"
	runtimeconfig "github.com/FelixSeptem/baymax/runtime/config"
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

func BenchmarkToolFanOutHighConcurrency(b *testing.B) {
	reg := local.NewRegistry()
	_, _ = reg.Register(&fakes.Tool{NameValue: "echo", InvokeFn: func(ctx context.Context, args map[string]any) (types.ToolResult, error) {
		return types.ToolResult{Content: "ok"}, nil
	}})
	dispatcher := local.NewDispatcher(reg)
	calls := make([]types.ToolCall, 64)
	for i := range calls {
		calls[i] = types.ToolCall{CallID: "c", Name: "local.echo"}
	}
	cfg := local.DispatchConfig{MaxCalls: 64, Concurrency: 32, QueueSize: 32, Backpressure: types.BackpressureBlock}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = dispatcher.Dispatch(context.Background(), calls, cfg)
	}
}

func BenchmarkToolFanOutSlowCall(b *testing.B) {
	reg := local.NewRegistry()
	_, _ = reg.Register(&fakes.Tool{NameValue: "slow", InvokeFn: func(ctx context.Context, args map[string]any) (types.ToolResult, error) {
		time.Sleep(100 * time.Microsecond)
		return types.ToolResult{Content: "ok"}, nil
	}})
	dispatcher := local.NewDispatcher(reg)
	calls := make([]types.ToolCall, 32)
	for i := range calls {
		calls[i] = types.ToolCall{CallID: "s", Name: "local.slow"}
	}
	cfg := local.DispatchConfig{MaxCalls: 32, Concurrency: 16, QueueSize: 16, Backpressure: types.BackpressureBlock}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = dispatcher.Dispatch(context.Background(), calls, cfg)
	}
}

func BenchmarkToolFanOutCancelStorm(b *testing.B) {
	reg := local.NewRegistry()
	_, _ = reg.Register(&fakes.Tool{NameValue: "slow", InvokeFn: func(ctx context.Context, args map[string]any) (types.ToolResult, error) {
		select {
		case <-ctx.Done():
			return types.ToolResult{}, ctx.Err()
		case <-time.After(2 * time.Millisecond):
			return types.ToolResult{Content: "ok"}, nil
		}
	}})
	dispatcher := local.NewDispatcher(reg)
	calls := make([]types.ToolCall, 16)
	for i := range calls {
		calls[i] = types.ToolCall{CallID: "k", Name: "local.slow"}
	}
	cfg := local.DispatchConfig{MaxCalls: 16, Concurrency: 8, QueueSize: 8, Backpressure: types.BackpressureBlock}
	durations := make([]int64, 0, b.N)
	goroutinePeak := 0
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		start := time.Now()
		ctx, cancel := context.WithTimeout(context.Background(), 200*time.Microsecond)
		_, _ = dispatcher.Dispatch(ctx, calls, cfg)
		cancel()
		durations = append(durations, time.Since(start).Nanoseconds())
		if n := runtime.NumGoroutine(); n > goroutinePeak {
			goroutinePeak = n
		}
	}
	b.StopTimer()
	if p95 := percentileNs(durations, 95); p95 > 0 {
		b.ReportMetric(float64(p95), "p95-ns/op")
	}
	if goroutinePeak > 0 {
		b.ReportMetric(float64(goroutinePeak), "goroutine-peak")
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

func BenchmarkMCPProfileDefaultUnderFailure(b *testing.B) {
	attempt := 0
	connector := func(ctx context.Context) (httpmcp.Session, error) {
		attempt++
		if attempt%3 != 0 {
			return &fakeHTTPSession{callErr: errors.New("flaky")}, nil
		}
		return &fakeHTTPSession{}, nil
	}
	client := httpmcp.NewClient(httpmcp.Config{
		Connect: connector,
		Profile: mcpprofile.Default,
	})
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = client.CallTool(context.Background(), "tool", nil)
	}
}

func BenchmarkMCPProfileHighReliabilityUnderFailure(b *testing.B) {
	attempt := 0
	connector := func(ctx context.Context) (httpmcp.Session, error) {
		attempt++
		if attempt%4 != 0 {
			return &fakeHTTPSession{callErr: errors.New("flaky")}, nil
		}
		return &fakeHTTPSession{}, nil
	}
	client := httpmcp.NewClient(httpmcp.Config{
		Connect: connector,
		Profile: mcpprofile.HighReliab,
	})
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = client.CallTool(context.Background(), "tool", nil)
	}
}

func BenchmarkCA4PressureEvaluation(b *testing.B) {
	cfg := runtimeconfig.DefaultConfig().ContextAssembler
	cfg.Enabled = true
	cfg.CA3.Enabled = true
	cfg.CA3.Tokenizer.Mode = "estimate_only"
	cfg.CA3.MaxContextTokens = 4096
	cfg.CA3.PercentThresholds = runtimeconfig.ContextAssemblerCA3Thresholds{
		Safe: 20, Comfort: 40, Warning: 60, Danger: 75, Emergency: 90,
	}
	cfg.CA3.AbsoluteThresholds = runtimeconfig.ContextAssemblerCA3Thresholds{
		Safe: 512, Comfort: 1024, Warning: 2048, Danger: 3072, Emergency: 3584,
	}
	a := assembler.New(func() runtimeconfig.ContextAssemblerConfig { return cfg })
	req := types.ContextAssembleRequest{
		RunID:         "bench-ca4",
		SessionID:     "bench-session",
		PrefixVersion: "ca1",
		Messages: []types.Message{
			{Role: "system", Content: "stable system prompt"},
			{Role: "user", Content: strings.Repeat("payload ", 80)},
		},
	}
	modelReq := types.ModelRequest{
		RunID:    req.RunID,
		Model:    "gpt-4.1-mini",
		Input:    strings.Repeat("bench input ", 160),
		Messages: append([]types.Message(nil), req.Messages...),
	}

	durations := make([]int64, 0, b.N)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		start := time.Now()
		if _, _, err := a.Assemble(context.Background(), req, modelReq); err != nil {
			b.Fatalf("assemble failed: %v", err)
		}
		durations = append(durations, time.Since(start).Nanoseconds())
	}
	b.StopTimer()
	if p95 := percentileNs(durations, 95); p95 > 0 {
		b.ReportMetric(float64(p95), "p95-ns/op")
	}
}

func percentileNs(values []int64, percentile int) int64 {
	if len(values) == 0 || percentile <= 0 {
		return 0
	}
	copyVals := append([]int64(nil), values...)
	sort.Slice(copyVals, func(i, j int) bool { return copyVals[i] < copyVals[j] })
	idx := (len(copyVals)*percentile + 99) / 100
	if idx <= 0 {
		idx = 1
	}
	if idx > len(copyVals) {
		idx = len(copyVals)
	}
	return copyVals[idx-1]
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
