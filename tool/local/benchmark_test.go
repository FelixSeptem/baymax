package local

import (
	"testing"

	"github.com/FelixSeptem/baymax/core/types"
	runtimeconfig "github.com/FelixSeptem/baymax/runtime/config"
)

func benchmarkDropLowPriorityPolicy() DropLowPriorityPolicy {
	return DropLowPriorityPolicy{
		PriorityByTool: map[string]string{
			"local.deploy": runtimeconfig.DropPriorityHigh,
		},
		PriorityByKeyword: map[string]string{
			"critical": runtimeconfig.DropPriorityHigh,
			"cache":    runtimeconfig.DropPriorityLow,
			"warmup":   runtimeconfig.DropPriorityLow,
			"report":   runtimeconfig.DropPriorityNormal,
		},
		DroppablePriorities: []string{runtimeconfig.DropPriorityLow},
	}
}

func BenchmarkLocalDispatchPriorityClassifyWarmCache(b *testing.B) {
	policy := benchmarkDropLowPriorityPolicy()
	classifier := compileDropLowPriorityClassifier(policy)
	call := types.ToolCall{
		Name: "local.search",
		Args: map[string]any{
			"q": "cache warmup task",
		},
	}

	_ = classifyPriority(call, policy, classifier)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = classifyPriority(call, policy, classifier)
	}
}

func BenchmarkLocalDispatchPriorityClassifyMixedCalls(b *testing.B) {
	policy := benchmarkDropLowPriorityPolicy()
	classifier := compileDropLowPriorityClassifier(policy)
	calls := []types.ToolCall{
		{Name: "local.search", Args: map[string]any{"q": "cache warmup task"}},
		{Name: "local.search", Args: map[string]any{"q": "critical failure"}},
		{Name: "local.report", Args: map[string]any{"q": "daily report"}},
		{Name: "local.deploy", Args: map[string]any{"q": "release"}},
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = classifyPriority(calls[i%len(calls)], policy, classifier)
	}
}
