package config

import (
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestBuildEvalSummaryThresholdBoundaries(t *testing.T) {
	agentCfg := RuntimeEvalAgentConfig{
		TaskSuccessThreshold:           0.80,
		ToolCorrectnessThreshold:       0.75,
		DenyInterceptAccuracyThreshold: 0.90,
		CostBudgetThreshold:            1.0,
		LatencyBudgetThreshold:         2 * time.Second,
	}
	summary := BuildEvalSummary(agentCfg, EvalSummaryInput{
		TaskTotal:       10,
		TaskSuccess:     8,
		ToolTotal:       20,
		ToolCorrect:     15,
		DenyTotal:       10,
		DenyCorrect:     9,
		CostEstimate:    0.98,
		LatencyEstimate: 1900 * time.Millisecond,
	})

	if summary["all_constraints_pass"] != true {
		t.Fatalf("summary all_constraints_pass = %#v, want true", summary["all_constraints_pass"])
	}
	task := summary["task_success"].(EvalMetricSummary)
	tool := summary["tool_correctness"].(EvalMetricSummary)
	deny := summary["deny_intercept"].(EvalMetricSummary)
	costLatency := summary["cost_latency"].(EvalCostLatencySummary)

	if task.Rate != 0.8 || !task.Passed {
		t.Fatalf("task_success summary mismatch: %#v", task)
	}
	if tool.Rate != 0.75 || !tool.Passed {
		t.Fatalf("tool_correctness summary mismatch: %#v", tool)
	}
	if deny.Rate != 0.9 || !deny.Passed {
		t.Fatalf("deny_intercept summary mismatch: %#v", deny)
	}
	if !costLatency.ConstraintPassed {
		t.Fatalf("cost/latency summary mismatch: %#v", costLatency)
	}
}

func TestAggregateEvalShardMetricsLocalAndDistributedEquivalence(t *testing.T) {
	shard := EvalShardMetrics{
		ShardID:             "shard-0",
		Attempt:             1,
		ResumeCount:         0,
		TaskSuccessRate:     0.9,
		TaskWeight:          10,
		ToolCorrectnessRate: 0.8,
		ToolWeight:          20,
		DenyInterceptRate:   0.95,
		DenyWeight:          10,
		CostEstimate:        0.7,
		LatencyEstimate:     1500 * time.Millisecond,
	}

	local := AggregateEvalShardMetrics(RuntimeEvalExecutionConfig{
		Mode: RuntimeEvalExecutionModeLocal,
	}, []EvalShardMetrics{shard})
	distributed := AggregateEvalShardMetrics(RuntimeEvalExecutionConfig{
		Mode:        RuntimeEvalExecutionModeDistributed,
		Aggregation: RuntimeEvalExecutionAggregationWeightedMean,
	}, []EvalShardMetrics{shard})

	if local.ShardTotal != 1 || distributed.ShardTotal != 1 {
		t.Fatalf("unexpected shard totals local=%d distributed=%d", local.ShardTotal, distributed.ShardTotal)
	}
	if local.Summary["task_success_rate"] != distributed.Summary["task_success_rate"] ||
		local.Summary["tool_correctness_rate"] != distributed.Summary["tool_correctness_rate"] ||
		local.Summary["deny_intercept_rate"] != distributed.Summary["deny_intercept_rate"] {
		t.Fatalf("local/distributed single-shard summaries should be equivalent: local=%#v distributed=%#v", local.Summary, distributed.Summary)
	}
}

func TestAggregateEvalShardMetricsResumeIdempotent(t *testing.T) {
	cfg := RuntimeEvalExecutionConfig{
		Mode:        RuntimeEvalExecutionModeDistributed,
		Aggregation: RuntimeEvalExecutionAggregationWeightedMean,
	}
	shards := []EvalShardMetrics{
		{
			ShardID:             "s1",
			Attempt:             1,
			ResumeCount:         0,
			TaskSuccessRate:     0.4,
			TaskWeight:          5,
			ToolCorrectnessRate: 0.5,
			ToolWeight:          5,
			DenyInterceptRate:   0.6,
			DenyWeight:          5,
			CostEstimate:        1.3,
			LatencyEstimate:     3 * time.Second,
		},
		{
			ShardID:             "s1",
			Attempt:             2,
			ResumeCount:         1,
			TaskSuccessRate:     0.8,
			TaskWeight:          5,
			ToolCorrectnessRate: 0.9,
			ToolWeight:          5,
			DenyInterceptRate:   0.95,
			DenyWeight:          5,
			CostEstimate:        0.9,
			LatencyEstimate:     1500 * time.Millisecond,
		},
		{
			ShardID:             "s2",
			Attempt:             1,
			ResumeCount:         0,
			TaskSuccessRate:     1.0,
			TaskWeight:          10,
			ToolCorrectnessRate: 0.7,
			ToolWeight:          10,
			DenyInterceptRate:   0.9,
			DenyWeight:          10,
			CostEstimate:        0.8,
			LatencyEstimate:     time.Second,
		},
	}

	first := AggregateEvalShardMetrics(cfg, shards)
	second := AggregateEvalShardMetrics(cfg, shards)
	if first.ShardTotal != 2 || second.ShardTotal != 2 {
		t.Fatalf("dedup shard total mismatch: first=%d second=%d", first.ShardTotal, second.ShardTotal)
	}
	if first.ResumeCount != 1 || second.ResumeCount != 1 {
		t.Fatalf("resume count mismatch: first=%d second=%d", first.ResumeCount, second.ResumeCount)
	}
	if first.Summary["task_success_rate"] != second.Summary["task_success_rate"] ||
		first.Summary["tool_correctness_rate"] != second.Summary["tool_correctness_rate"] ||
		first.Summary["deny_intercept_rate"] != second.Summary["deny_intercept_rate"] ||
		first.Summary["cost_estimate"] != second.Summary["cost_estimate"] ||
		first.Summary["latency_estimate"] != second.Summary["latency_estimate"] {
		t.Fatalf("aggregation must be deterministic and idempotent: first=%#v second=%#v", first.Summary, second.Summary)
	}
}

func TestRuntimeEvalExecutionConfigBoundaryNoControlPlaneDependency(t *testing.T) {
	typeName := reflect.TypeOf(RuntimeEvalExecutionConfig{})
	bannedTokens := []string{
		"control_plane",
		"scheduler",
		"service",
		"endpoint",
		"hosted",
	}
	for i := 0; i < typeName.NumField(); i++ {
		field := typeName.Field(i)
		jsonTag := strings.ToLower(strings.TrimSpace(strings.Split(field.Tag.Get("json"), ",")[0]))
		name := strings.ToLower(strings.TrimSpace(field.Name))
		for _, token := range bannedTokens {
			if strings.Contains(name, token) || strings.Contains(jsonTag, token) {
				t.Fatalf("runtime.eval.execution.* must stay embedded and control-plane free, got field=%q tag=%q", field.Name, jsonTag)
			}
		}
	}
}
