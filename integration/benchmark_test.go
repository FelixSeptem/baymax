package integration

import (
	"context"
	"errors"
	"fmt"
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
	runtimediag "github.com/FelixSeptem/baymax/runtime/diagnostics"
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

func BenchmarkToolFanOutDropLowPriority(b *testing.B) {
	reg := local.NewRegistry()
	_, _ = reg.Register(&fakes.Tool{NameValue: "slow", InvokeFn: func(ctx context.Context, args map[string]any) (types.ToolResult, error) {
		time.Sleep(250 * time.Microsecond)
		return types.ToolResult{Content: "ok"}, nil
	}})
	dispatcher := local.NewDispatcher(reg)
	calls := make([]types.ToolCall, 24)
	for i := range calls {
		calls[i] = types.ToolCall{
			CallID: "d",
			Name:   "local.slow",
			Args:   map[string]any{"q": "cache warmup"},
		}
	}
	cfg := local.DispatchConfig{
		MaxCalls:     24,
		Concurrency:  2,
		QueueSize:    2,
		Backpressure: types.BackpressureDropLowPriority,
		DropLowPriority: local.DropLowPriorityPolicy{
			PriorityByKeyword:   map[string]string{"cache": runtimeconfig.DropPriorityLow},
			DroppablePriorities: []string{runtimeconfig.DropPriorityLow},
		},
	}
	durations := make([]int64, 0, b.N)
	goroutinePeak := 0
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		start := time.Now()
		_, _ = dispatcher.Dispatch(context.Background(), calls, cfg)
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

func BenchmarkToolFanOutDropLowPriorityMCPAndSkill(b *testing.B) {
	reg := local.NewRegistry()
	_, _ = reg.Register(&fakes.Tool{NameValue: "mcp_proxy", InvokeFn: func(ctx context.Context, args map[string]any) (types.ToolResult, error) {
		time.Sleep(250 * time.Microsecond)
		return types.ToolResult{Content: "ok"}, nil
	}})
	_, _ = reg.Register(&fakes.Tool{NameValue: "skill_router", InvokeFn: func(ctx context.Context, args map[string]any) (types.ToolResult, error) {
		time.Sleep(250 * time.Microsecond)
		return types.ToolResult{Content: "ok"}, nil
	}})
	dispatcher := local.NewDispatcher(reg)
	calls := make([]types.ToolCall, 24)
	for i := range calls {
		if i%2 == 0 {
			calls[i] = types.ToolCall{CallID: "m", Name: "local.mcp_proxy", Args: map[string]any{"q": "cache warmup"}}
			continue
		}
		calls[i] = types.ToolCall{CallID: "s", Name: "local.skill_router", Args: map[string]any{"q": "cache warmup"}}
	}
	cfg := local.DispatchConfig{
		MaxCalls:     24,
		Concurrency:  2,
		QueueSize:    2,
		Backpressure: types.BackpressureDropLowPriority,
		DropLowPriority: local.DropLowPriorityPolicy{
			PriorityByKeyword:   map[string]string{"cache": runtimeconfig.DropPriorityLow},
			DroppablePriorities: []string{runtimeconfig.DropPriorityLow},
		},
	}
	durations := make([]int64, 0, b.N)
	goroutinePeak := 0
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		start := time.Now()
		_, _ = dispatcher.Dispatch(context.Background(), calls, cfg)
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

func BenchmarkCA3SemanticCompactionLatency(b *testing.B) {
	cfg := runtimeconfig.DefaultConfig().ContextAssembler
	cfg.Enabled = true
	cfg.CA3.Enabled = true
	cfg.CA3.MaxContextTokens = 4096
	cfg.CA3.Compaction.Mode = "semantic"
	cfg.CA3.Compaction.Quality.Threshold = 0.2
	cfg.CA3.Compaction.Evidence.Keywords = []string{"decision", "risk"}
	cfg.CA3.Tokenizer.Mode = "estimate_only"
	a := assembler.New(func() runtimeconfig.ContextAssemblerConfig { return cfg })

	model := fakes.NewModel([]fakes.ModelStep{
		{Response: types.ModelResponse{FinalAnswer: "decision preserved with compact risk summary"}},
	})
	req := types.ContextAssembleRequest{
		RunID:         "bench-ca3-semantic",
		SessionID:     "bench-session",
		PrefixVersion: "ca1",
		Input:         strings.Repeat("input payload ", 80),
		Messages: []types.Message{
			{Role: "system", Content: "stable system prompt"},
			{Role: "user", Content: strings.Repeat("decision and risk details. ", 80)},
		},
		ModelClient: model,
	}
	modelReq := types.ModelRequest{
		RunID:    req.RunID,
		Model:    "gpt-4.1-mini",
		Input:    req.Input,
		Messages: append([]types.Message(nil), req.Messages...),
	}

	durations := make([]int64, 0, b.N)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		start := time.Now()
		if _, _, err := a.Assemble(context.Background(), req, modelReq); err != nil {
			b.Fatalf("assemble semantic compaction failed: %v", err)
		}
		durations = append(durations, time.Since(start).Nanoseconds())
	}
	b.StopTimer()
	if p95 := percentileNs(durations, 95); p95 > 0 {
		b.ReportMetric(float64(p95), "p95-ns/op")
	}
}

type benchEmbeddingScorer func(ctx context.Context, req assembler.SemanticEmbeddingScoreRequest) (float64, error)

func (f benchEmbeddingScorer) Score(ctx context.Context, req assembler.SemanticEmbeddingScoreRequest) (float64, error) {
	return f(ctx, req)
}

type benchReranker func(ctx context.Context, req assembler.SemanticRerankRequest) (assembler.SemanticRerankResult, error)

func (f benchReranker) Rerank(ctx context.Context, req assembler.SemanticRerankRequest) (assembler.SemanticRerankResult, error) {
	return f(ctx, req)
}

func BenchmarkCA3SemanticCompactionLatencyEmbeddingEnabled(b *testing.B) {
	cfg := runtimeconfig.DefaultConfig().ContextAssembler
	cfg.Enabled = true
	cfg.CA3.Enabled = true
	cfg.CA3.MaxContextTokens = 4096
	cfg.CA3.Compaction.Mode = "semantic"
	cfg.CA3.Compaction.Quality.Threshold = 0.2
	cfg.CA3.Compaction.Evidence.Keywords = []string{"decision", "risk"}
	cfg.CA3.Tokenizer.Mode = "estimate_only"
	cfg.CA3.Compaction.Embedding.Enabled = true
	cfg.CA3.Compaction.Embedding.Selector = "bench"
	cfg.CA3.Compaction.Embedding.Provider = "openai"
	cfg.CA3.Compaction.Embedding.Model = "text-embedding-3-small"
	cfg.CA3.Compaction.Embedding.Timeout = 200 * time.Millisecond
	cfg.CA3.Compaction.Embedding.SimilarityMetric = "cosine"
	cfg.CA3.Compaction.Embedding.RuleWeight = 0.7
	cfg.CA3.Compaction.Embedding.EmbeddingWeight = 0.3

	scorer := benchEmbeddingScorer(func(ctx context.Context, req assembler.SemanticEmbeddingScoreRequest) (float64, error) {
		_ = ctx
		_ = req
		return 0.82, nil
	})
	a := assembler.New(func() runtimeconfig.ContextAssemblerConfig { return cfg }, assembler.WithSemanticEmbeddingScorer("", scorer))

	model := fakes.NewModel([]fakes.ModelStep{
		{Response: types.ModelResponse{FinalAnswer: "decision preserved with compact risk summary"}},
	})
	req := types.ContextAssembleRequest{
		RunID:         "bench-ca3-semantic-embedding",
		SessionID:     "bench-session",
		PrefixVersion: "ca1",
		Input:         strings.Repeat("input payload ", 80),
		Messages: []types.Message{
			{Role: "system", Content: "stable system prompt"},
			{Role: "user", Content: strings.Repeat("decision and risk details. ", 80)},
		},
		ModelClient: model,
	}
	modelReq := types.ModelRequest{
		RunID:    req.RunID,
		Model:    "gpt-4.1-mini",
		Input:    req.Input,
		Messages: append([]types.Message(nil), req.Messages...),
	}

	durations := make([]int64, 0, b.N)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		start := time.Now()
		if _, _, err := a.Assemble(context.Background(), req, modelReq); err != nil {
			b.Fatalf("assemble semantic compaction with embedding failed: %v", err)
		}
		durations = append(durations, time.Since(start).Nanoseconds())
	}
	b.StopTimer()
	if p95 := percentileNs(durations, 95); p95 > 0 {
		b.ReportMetric(float64(p95), "p95-ns/op")
	}
}

func BenchmarkCA3SemanticCompactionLatencyRerankerGovernanceEnabled(b *testing.B) {
	cfg := runtimeconfig.DefaultConfig().ContextAssembler
	cfg.Enabled = true
	cfg.CA3.Enabled = true
	cfg.CA3.MaxContextTokens = 4096
	cfg.CA3.Compaction.Mode = "semantic"
	cfg.CA3.Compaction.Quality.Threshold = 0.2
	cfg.CA3.Compaction.Evidence.Keywords = []string{"decision", "risk"}
	cfg.CA3.Tokenizer.Mode = "estimate_only"
	cfg.CA3.Compaction.Embedding.Enabled = true
	cfg.CA3.Compaction.Embedding.Selector = "bench"
	cfg.CA3.Compaction.Embedding.Provider = "anthropic"
	cfg.CA3.Compaction.Embedding.Model = "claude-3-haiku"
	cfg.CA3.Compaction.Embedding.Timeout = 200 * time.Millisecond
	cfg.CA3.Compaction.Reranker.Enabled = true
	cfg.CA3.Compaction.Reranker.Timeout = 120 * time.Millisecond
	cfg.CA3.Compaction.Reranker.MaxRetries = 0
	cfg.CA3.Compaction.Reranker.ThresholdProfiles = map[string]float64{
		"anthropic:claude-3-haiku": 0.68,
	}
	cfg.CA3.Compaction.Reranker.Governance.Mode = runtimeconfig.CA3RerankerGovernanceModeEnforce
	cfg.CA3.Compaction.Reranker.Governance.ProfileVersion = "bench-v1"
	cfg.CA3.Compaction.Reranker.Governance.RolloutProviderModels = []string{"anthropic:claude-3-haiku"}

	reranker := benchReranker(func(ctx context.Context, req assembler.SemanticRerankRequest) (assembler.SemanticRerankResult, error) {
		_ = ctx
		return assembler.SemanticRerankResult{Score: req.CurrentScore}, nil
	})
	a := assembler.New(
		func() runtimeconfig.ContextAssemblerConfig { return cfg },
		assembler.WithSemanticReranker("anthropic", reranker),
	)
	model := fakes.NewModel([]fakes.ModelStep{
		{Response: types.ModelResponse{FinalAnswer: "decision preserved with compact risk summary"}},
	})
	req := types.ContextAssembleRequest{
		RunID:         "bench-ca3-semantic-reranker-governance",
		SessionID:     "bench-session",
		PrefixVersion: "ca1",
		Input:         strings.Repeat("input payload ", 80),
		Messages: []types.Message{
			{Role: "system", Content: "stable system prompt"},
			{Role: "user", Content: strings.Repeat("decision and risk details. ", 80)},
		},
		ModelClient: model,
	}
	modelReq := types.ModelRequest{
		RunID:    req.RunID,
		Model:    "claude-3-haiku",
		Input:    req.Input,
		Messages: append([]types.Message(nil), req.Messages...),
	}

	durations := make([]int64, 0, b.N)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		start := time.Now()
		if _, _, err := a.Assemble(context.Background(), req, modelReq); err != nil {
			b.Fatalf("assemble semantic compaction with reranker governance failed: %v", err)
		}
		durations = append(durations, time.Since(start).Nanoseconds())
	}
	b.StopTimer()
	if p95 := percentileNs(durations, 95); p95 > 0 {
		b.ReportMetric(float64(p95), "p95-ns/op")
	}
}

func BenchmarkDiagnosticsTimelineTrendQuery(b *testing.B) {
	store := runtimediag.NewStore(64, 512, 16, 32, runtimediag.TimelineTrendConfig{
		Enabled:    true,
		LastNRuns:  100,
		TimeWindow: 15 * time.Minute,
	}, runtimediag.CA2ExternalTrendConfig{Enabled: true, Window: 15 * time.Minute})
	now := time.Now()
	for i := 0; i < 400; i++ {
		runID := fmt.Sprintf("bench-run-%d", i)
		start := now.Add(time.Duration(i) * time.Millisecond)
		end := start.Add(time.Duration((i%9)+1) * 3 * time.Millisecond)
		store.AddTimelineEvent(runID, "model", "running", int64(i*2+1), start)
		store.AddTimelineEvent(runID, "model", "succeeded", int64(i*2+2), end)
		store.AddRun(runtimediag.RunRecord{
			Time:      end,
			RunID:     runID,
			Status:    "success",
			LatencyMs: end.Sub(start).Milliseconds(),
		})
	}
	durations := make([]int64, 0, b.N)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		start := time.Now()
		items := store.TimelineTrends(runtimediag.TimelineTrendQuery{
			Mode:      runtimediag.TimelineTrendModeLastNRuns,
			LastNRuns: 100,
		})
		if len(items) == 0 {
			b.Fatalf("trend query returned empty result")
		}
		for _, item := range items {
			if item.LatencyP95Ms < 0 {
				b.Fatalf("invalid latency_p95_ms: %#v", item)
			}
		}
		durations = append(durations, time.Since(start).Nanoseconds())
	}
	b.StopTimer()
	if p95 := percentileNs(durations, 95); p95 > 0 {
		b.ReportMetric(float64(p95), "p95-ns/op")
	}
}

func BenchmarkCA2ExternalRetrieverTrendAggregation(b *testing.B) {
	store := runtimediag.NewStore(64, 1024, 16, 32,
		runtimediag.TimelineTrendConfig{Enabled: true, LastNRuns: 100, TimeWindow: 15 * time.Minute},
		runtimediag.CA2ExternalTrendConfig{
			Enabled: true,
			Window:  15 * time.Minute,
			Thresholds: runtimediag.CA2ExternalThresholds{
				P95LatencyMs: 1000,
				ErrorRate:    0.15,
				HitRate:      0.30,
			},
		},
	)
	now := time.Now()
	for i := 0; i < 600; i++ {
		provider := "http"
		if i%2 == 0 {
			provider = "rag"
		}
		code := "ok"
		layer := ""
		hitCount := 1
		if i%5 == 0 {
			code = "timeout"
			layer = "transport"
			hitCount = 0
		}
		store.AddRun(runtimediag.RunRecord{
			Time:             now.Add(time.Duration(i) * time.Millisecond),
			RunID:            fmt.Sprintf("ca2-run-%d", i),
			Stage2Provider:   provider,
			Stage2LatencyMs:  int64((i%11)+1) * 35,
			Stage2HitCount:   hitCount,
			Stage2ReasonCode: code,
			Stage2ErrorLayer: layer,
		})
	}
	durations := make([]int64, 0, b.N)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		start := time.Now()
		items := store.CA2ExternalTrends(runtimediag.CA2ExternalTrendQuery{})
		if len(items) == 0 {
			b.Fatalf("ca2 external trend result is empty")
		}
		for _, item := range items {
			if item.P95LatencyMs < 0 || item.ErrorRate < 0 || item.HitRate < 0 {
				b.Fatalf("invalid trend record: %#v", item)
			}
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
