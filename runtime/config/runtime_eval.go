package config

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

const (
	RuntimeEvalExecutionModeLocal       = "local"
	RuntimeEvalExecutionModeDistributed = "distributed"
)

const (
	RuntimeEvalExecutionAggregationWeightedMean = "weighted_mean"
	RuntimeEvalExecutionAggregationWorstCase    = "worst_case"
)

type RuntimeEvalConfig struct {
	Agent     RuntimeEvalAgentConfig     `json:"agent"`
	Execution RuntimeEvalExecutionConfig `json:"execution"`
}

type RuntimeEvalAgentConfig struct {
	Enabled                        bool          `json:"enabled"`
	SuiteID                        string        `json:"suite_id"`
	TaskSuccessThreshold           float64       `json:"task_success_threshold"`
	ToolCorrectnessThreshold       float64       `json:"tool_correctness_threshold"`
	DenyInterceptAccuracyThreshold float64       `json:"deny_intercept_accuracy_threshold"`
	CostBudgetThreshold            float64       `json:"cost_budget_threshold"`
	LatencyBudgetThreshold         time.Duration `json:"latency_budget_threshold"`
}

type RuntimeEvalExecutionConfig struct {
	Mode        string                           `json:"mode"`
	Shard       RuntimeEvalExecutionShardConfig  `json:"shard"`
	Retry       RuntimeEvalExecutionRetryConfig  `json:"retry"`
	Resume      RuntimeEvalExecutionResumeConfig `json:"resume"`
	Aggregation string                           `json:"aggregation"`
}

type RuntimeEvalExecutionShardConfig struct {
	Total int `json:"total"`
}

type RuntimeEvalExecutionRetryConfig struct {
	MaxAttempts int `json:"max_attempts"`
}

type RuntimeEvalExecutionResumeConfig struct {
	Enabled  bool `json:"enabled"`
	MaxCount int  `json:"max_count"`
}

type EvalSummaryInput struct {
	TaskTotal       int
	TaskSuccess     int
	ToolTotal       int
	ToolCorrect     int
	DenyTotal       int
	DenyCorrect     int
	CostEstimate    float64
	LatencyEstimate time.Duration
}

type EvalMetricSummary struct {
	Rate      float64 `json:"rate"`
	Threshold float64 `json:"threshold"`
	Passed    bool    `json:"passed"`
	Total     int     `json:"total"`
}

type EvalCostLatencySummary struct {
	CostEstimate     float64       `json:"cost_estimate"`
	CostThreshold    float64       `json:"cost_threshold"`
	CostWithinBudget bool          `json:"cost_within_budget"`
	LatencyEstimate  time.Duration `json:"latency_estimate"`
	LatencyThreshold time.Duration `json:"latency_threshold"`
	LatencyWithinSLO bool          `json:"latency_within_slo"`
	ConstraintPassed bool          `json:"constraint_passed"`
}

type EvalShardMetrics struct {
	ShardID             string
	Attempt             int
	ResumeCount         int
	TaskSuccessRate     float64
	TaskWeight          int
	ToolCorrectnessRate float64
	ToolWeight          int
	DenyInterceptRate   float64
	DenyWeight          int
	CostEstimate        float64
	LatencyEstimate     time.Duration
}

type EvalAggregationResult struct {
	Mode        string         `json:"mode"`
	ShardTotal  int            `json:"shard_total"`
	ResumeCount int            `json:"resume_count"`
	Summary     map[string]any `json:"summary"`
}

func normalizeRuntimeEvalConfig(in RuntimeEvalConfig) RuntimeEvalConfig {
	base := DefaultConfig().Runtime.Eval
	out := in

	out.Agent.SuiteID = strings.TrimSpace(out.Agent.SuiteID)
	if out.Agent.SuiteID == "" {
		out.Agent.SuiteID = strings.TrimSpace(base.Agent.SuiteID)
	}
	out.Execution.Mode = strings.ToLower(strings.TrimSpace(out.Execution.Mode))
	if out.Execution.Mode == "" {
		out.Execution.Mode = strings.ToLower(strings.TrimSpace(base.Execution.Mode))
	}
	out.Execution.Aggregation = strings.ToLower(strings.TrimSpace(out.Execution.Aggregation))
	if out.Execution.Aggregation == "" {
		out.Execution.Aggregation = strings.ToLower(strings.TrimSpace(base.Execution.Aggregation))
	}
	return out
}

func ValidateRuntimeEvalConfig(cfg RuntimeEvalConfig) error {
	normalized := normalizeRuntimeEvalConfig(cfg)

	if strings.TrimSpace(normalized.Agent.SuiteID) == "" {
		return fmt.Errorf("runtime.eval.agent.suite_id is required")
	}
	if normalized.Agent.TaskSuccessThreshold < 0 || normalized.Agent.TaskSuccessThreshold > 1 {
		return fmt.Errorf("runtime.eval.agent.task_success_threshold must be in [0,1], got %v", cfg.Agent.TaskSuccessThreshold)
	}
	if normalized.Agent.ToolCorrectnessThreshold < 0 || normalized.Agent.ToolCorrectnessThreshold > 1 {
		return fmt.Errorf("runtime.eval.agent.tool_correctness_threshold must be in [0,1], got %v", cfg.Agent.ToolCorrectnessThreshold)
	}
	if normalized.Agent.DenyInterceptAccuracyThreshold < 0 || normalized.Agent.DenyInterceptAccuracyThreshold > 1 {
		return fmt.Errorf("runtime.eval.agent.deny_intercept_accuracy_threshold must be in [0,1], got %v", cfg.Agent.DenyInterceptAccuracyThreshold)
	}
	if normalized.Agent.CostBudgetThreshold <= 0 {
		return fmt.Errorf("runtime.eval.agent.cost_budget_threshold must be > 0")
	}
	if normalized.Agent.LatencyBudgetThreshold <= 0 {
		return fmt.Errorf("runtime.eval.agent.latency_budget_threshold must be > 0")
	}

	switch normalized.Execution.Mode {
	case RuntimeEvalExecutionModeLocal, RuntimeEvalExecutionModeDistributed:
	default:
		return fmt.Errorf(
			"runtime.eval.execution.mode must be one of [%s,%s], got %q",
			RuntimeEvalExecutionModeLocal,
			RuntimeEvalExecutionModeDistributed,
			cfg.Execution.Mode,
		)
	}
	if normalized.Execution.Shard.Total <= 0 {
		return fmt.Errorf("runtime.eval.execution.shard.total must be > 0")
	}
	if normalized.Execution.Retry.MaxAttempts <= 0 {
		return fmt.Errorf("runtime.eval.execution.retry.max_attempts must be > 0")
	}
	if normalized.Execution.Resume.MaxCount < 0 {
		return fmt.Errorf("runtime.eval.execution.resume.max_count must be >= 0")
	}
	switch normalized.Execution.Aggregation {
	case RuntimeEvalExecutionAggregationWeightedMean, RuntimeEvalExecutionAggregationWorstCase:
	default:
		return fmt.Errorf(
			"runtime.eval.execution.aggregation must be one of [%s,%s], got %q",
			RuntimeEvalExecutionAggregationWeightedMean,
			RuntimeEvalExecutionAggregationWorstCase,
			cfg.Execution.Aggregation,
		)
	}
	return nil
}

func BuildEvalSummary(cfg RuntimeEvalAgentConfig, input EvalSummaryInput) map[string]any {
	taskSuccess := buildEvalRateSummary(input.TaskSuccess, input.TaskTotal, cfg.TaskSuccessThreshold)
	toolCorrectness := buildEvalRateSummary(input.ToolCorrect, input.ToolTotal, cfg.ToolCorrectnessThreshold)
	denyIntercept := buildEvalRateSummary(input.DenyCorrect, input.DenyTotal, cfg.DenyInterceptAccuracyThreshold)

	costWithinBudget := input.CostEstimate <= cfg.CostBudgetThreshold
	latencyWithinSLO := input.LatencyEstimate <= cfg.LatencyBudgetThreshold
	costLatency := EvalCostLatencySummary{
		CostEstimate:     input.CostEstimate,
		CostThreshold:    cfg.CostBudgetThreshold,
		CostWithinBudget: costWithinBudget,
		LatencyEstimate:  input.LatencyEstimate,
		LatencyThreshold: cfg.LatencyBudgetThreshold,
		LatencyWithinSLO: latencyWithinSLO,
		ConstraintPassed: costWithinBudget && latencyWithinSLO,
	}

	return map[string]any{
		"version":              "agent_eval.v1",
		"task_success":         taskSuccess,
		"tool_correctness":     toolCorrectness,
		"deny_intercept":       denyIntercept,
		"cost_latency":         costLatency,
		"all_constraints_pass": taskSuccess.Passed && toolCorrectness.Passed && denyIntercept.Passed && costLatency.ConstraintPassed,
	}
}

func AggregateEvalShardMetrics(cfg RuntimeEvalExecutionConfig, shards []EvalShardMetrics) EvalAggregationResult {
	mode := strings.ToLower(strings.TrimSpace(cfg.Mode))
	if mode == "" {
		mode = RuntimeEvalExecutionModeLocal
	}
	deduped := dedupeEvalShardMetrics(shards)
	sort.Slice(deduped, func(i, j int) bool {
		return strings.Compare(deduped[i].ShardID, deduped[j].ShardID) < 0
	})

	if mode == RuntimeEvalExecutionModeLocal {
		if len(deduped) == 0 {
			return EvalAggregationResult{
				Mode:        mode,
				ShardTotal:  0,
				ResumeCount: 0,
				Summary: map[string]any{
					"version": "agent_eval.v1",
				},
			}
		}
		only := deduped[0]
		return EvalAggregationResult{
			Mode:        mode,
			ShardTotal:  1,
			ResumeCount: maxInt(0, only.ResumeCount),
			Summary: map[string]any{
				"version":               "agent_eval.v1",
				"task_success_rate":     clampRate(only.TaskSuccessRate),
				"tool_correctness_rate": clampRate(only.ToolCorrectnessRate),
				"deny_intercept_rate":   clampRate(only.DenyInterceptRate),
				"cost_estimate":         only.CostEstimate,
				"latency_estimate":      only.LatencyEstimate,
			},
		}
	}

	totalTaskWeight := 0
	totalToolWeight := 0
	totalDenyWeight := 0
	taskWeighted := 0.0
	toolWeighted := 0.0
	denyWeighted := 0.0
	resumeCount := 0
	maxCost := 0.0
	maxLatency := time.Duration(0)
	sumCost := 0.0
	sumLatency := time.Duration(0)
	for _, shard := range deduped {
		taskWeight := maxInt(1, shard.TaskWeight)
		toolWeight := maxInt(1, shard.ToolWeight)
		denyWeight := maxInt(1, shard.DenyWeight)
		totalTaskWeight += taskWeight
		totalToolWeight += toolWeight
		totalDenyWeight += denyWeight
		taskWeighted += clampRate(shard.TaskSuccessRate) * float64(taskWeight)
		toolWeighted += clampRate(shard.ToolCorrectnessRate) * float64(toolWeight)
		denyWeighted += clampRate(shard.DenyInterceptRate) * float64(denyWeight)
		resumeCount += maxInt(0, shard.ResumeCount)
		sumCost += shard.CostEstimate
		sumLatency += shard.LatencyEstimate
		if shard.CostEstimate > maxCost {
			maxCost = shard.CostEstimate
		}
		if shard.LatencyEstimate > maxLatency {
			maxLatency = shard.LatencyEstimate
		}
	}

	aggregation := strings.ToLower(strings.TrimSpace(cfg.Aggregation))
	if aggregation == "" {
		aggregation = RuntimeEvalExecutionAggregationWeightedMean
	}
	costEstimate := 0.0
	latencyEstimate := time.Duration(0)
	switch aggregation {
	case RuntimeEvalExecutionAggregationWorstCase:
		costEstimate = maxCost
		latencyEstimate = maxLatency
	default:
		if len(deduped) > 0 {
			costEstimate = sumCost / float64(len(deduped))
			latencyEstimate = sumLatency / time.Duration(len(deduped))
		}
	}

	return EvalAggregationResult{
		Mode:        mode,
		ShardTotal:  len(deduped),
		ResumeCount: resumeCount,
		Summary: map[string]any{
			"version":               "agent_eval_distributed.v1",
			"task_success_rate":     safeWeightedRate(taskWeighted, totalTaskWeight),
			"tool_correctness_rate": safeWeightedRate(toolWeighted, totalToolWeight),
			"deny_intercept_rate":   safeWeightedRate(denyWeighted, totalDenyWeight),
			"cost_estimate":         costEstimate,
			"latency_estimate":      latencyEstimate,
			"aggregation":           aggregation,
		},
	}
}

func buildEvalRateSummary(passed, total int, threshold float64) EvalMetricSummary {
	rate := 1.0
	if total > 0 {
		rate = clampRate(float64(maxInt(0, passed)) / float64(total))
	}
	return EvalMetricSummary{
		Rate:      rate,
		Threshold: threshold,
		Passed:    rate >= threshold,
		Total:     maxInt(0, total),
	}
}

func dedupeEvalShardMetrics(in []EvalShardMetrics) []EvalShardMetrics {
	if len(in) == 0 {
		return nil
	}
	best := map[string]EvalShardMetrics{}
	for _, shard := range in {
		shardID := strings.TrimSpace(shard.ShardID)
		if shardID == "" {
			shardID = fmt.Sprintf("shard-%03d", len(best))
		}
		shard.ShardID = shardID
		prev, exists := best[shardID]
		if !exists || shard.Attempt > prev.Attempt || (shard.Attempt == prev.Attempt && shard.ResumeCount > prev.ResumeCount) {
			best[shardID] = shard
		}
	}
	out := make([]EvalShardMetrics, 0, len(best))
	for _, shard := range best {
		out = append(out, shard)
	}
	return out
}

func safeWeightedRate(weightedTotal float64, totalWeight int) float64 {
	if totalWeight <= 0 {
		return 1.0
	}
	return clampRate(weightedTotal / float64(totalWeight))
}

func clampRate(value float64) float64 {
	switch {
	case value < 0:
		return 0
	case value > 1:
		return 1
	default:
		return value
	}
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
