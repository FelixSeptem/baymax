package budgetprojection

import (
	"crypto/sha256"
	"encoding/hex"
	"sort"
	"strconv"
	"strings"
)

// Benchmark step outcomes. Only these four values are accepted.
const (
	OutcomeProgress          = "progress"
	OutcomeRepeat            = "repeat"
	OutcomeTerminalSuccess   = "terminal_success"
	OutcomeTerminalExhausted = "terminal_exhausted"
)

// BudgetBenchmarkOutcomes is the canonical ordered set of step outcomes.
var BudgetBenchmarkOutcomes = []string{
	OutcomeProgress,
	OutcomeRepeat,
	OutcomeTerminalSuccess,
	OutcomeTerminalExhausted,
}

// BudgetBenchmarkStep is one decision step of a replayed trace. Facts are a
// read-only snapshot; no raw reasoning, transcript or payload is carried.
type BudgetBenchmarkStep struct {
	Outcome string      `json:"outcome"`
	Facts   BudgetFacts `json:"facts"`
}

// BudgetBenchmarkTrace is a bounded decision trace.
type BudgetBenchmarkTrace struct {
	TraceID string                `json:"trace_id"`
	Steps   []BudgetBenchmarkStep `json:"steps"`
}

// BudgetUtilizationSample aggregates one dimension's usage across a benchmark.
type BudgetUtilizationSample struct {
	Dimension string  `json:"dimension"`
	Ratio     float64 `json:"ratio"`
	Samples   int     `json:"samples"`
}

// BudgetBenchmarkMetrics is the deterministic, recomputable benchmark result.
type BudgetBenchmarkMetrics struct {
	TraceCount       int                       `json:"trace_count"`
	StepCount        int                       `json:"step_count"`
	CompletionRate   float64                   `json:"completion_rate"`
	RepeatedLoopRate float64                   `json:"repeated_loop_rate"`
	SaturationRate   float64                   `json:"saturation_rate"`
	Utilization      []BudgetUtilizationSample `json:"utilization"`
	Digest           string                    `json:"digest"`
}

// benchmarkDimensions is the stable ordering used for utilization output.
var benchmarkDimensions = []string{"iteration", "tool_call", "time_ms", "cost_headroom"}

// EvaluateBudgetBenchmark recomputes the budget utilization metrics from a
// bounded set of traces. It is deterministic, offline and side-effect free: no
// model call, no clock, no network, no runtime state.
func EvaluateBudgetBenchmark(traces []BudgetBenchmarkTrace) (BudgetBenchmarkMetrics, error) {
	if len(traces) == 0 {
		return BudgetBenchmarkMetrics{}, newError(CodeBenchmarkSchemaDrift, "benchmark requires at least one trace")
	}
	if len(traces) > MaxTraces {
		return BudgetBenchmarkMetrics{}, newError(
			CodeBenchmarkOverflowDrift, "trace count %d exceeds limit %d", len(traces), MaxTraces,
		)
	}

	metrics := BudgetBenchmarkMetrics{
		TraceCount:  len(traces),
		Utilization: make([]BudgetUtilizationSample, 0, len(benchmarkDimensions)),
	}
	ratioTotals := map[string]float64{}
	ratioCounts := map[string]int{}
	completed := 0
	saturated := 0
	repeated := 0

	for _, trace := range traces {
		if strings.TrimSpace(trace.TraceID) == "" {
			return BudgetBenchmarkMetrics{}, newError(CodeBenchmarkSchemaDrift, "trace id must not be empty")
		}
		if len(trace.Steps) == 0 {
			return BudgetBenchmarkMetrics{}, newError(
				CodeBenchmarkSchemaDrift, "trace %q must contain at least one step", trace.TraceID,
			)
		}
		if len(trace.Steps) > MaxStepsPerTrace {
			return BudgetBenchmarkMetrics{}, newError(
				CodeBenchmarkOverflowDrift,
				"trace %q step count %d exceeds limit %d",
				trace.TraceID, len(trace.Steps), MaxStepsPerTrace,
			)
		}
		for index, step := range trace.Steps {
			if !isKnownOutcome(step.Outcome) {
				return BudgetBenchmarkMetrics{}, newError(
					CodeBenchmarkSchemaDrift,
					"trace %q step %d has unknown outcome %q",
					trace.TraceID, index, step.Outcome,
				)
			}
			metrics.StepCount++
			switch step.Outcome {
			case OutcomeRepeat:
				repeated++
			case OutcomeTerminalSuccess:
				completed++
			case OutcomeTerminalExhausted:
				saturated++
			}
			for _, sample := range utilizationSamples(step.Facts) {
				ratioTotals[sample.Dimension] += sample.Ratio
				ratioCounts[sample.Dimension]++
			}
		}
	}

	metrics.CompletionRate = float64(completed) / float64(metrics.TraceCount)
	metrics.RepeatedLoopRate = float64(repeated) / float64(metrics.StepCount)
	metrics.SaturationRate = float64(saturated) / float64(metrics.TraceCount)

	dimensions := append([]string(nil), benchmarkDimensions...)
	sort.Strings(dimensions)
	for _, dimension := range dimensions {
		count, ok := ratioCounts[dimension]
		if !ok || count == 0 {
			metrics.Utilization = append(metrics.Utilization, BudgetUtilizationSample{
				Dimension: dimension,
			})
			continue
		}
		metrics.Utilization = append(metrics.Utilization, BudgetUtilizationSample{
			Dimension: dimension,
			Ratio:     ratioTotals[dimension] / float64(count),
			Samples:   count,
		})
	}

	metrics.Digest = BudgetBenchmarkDigest(metrics)
	return metrics, nil
}

func utilizationSamples(facts BudgetFacts) []BudgetUtilizationSample {
	projection := DeriveBudgetProjection(facts)
	samples := make([]BudgetUtilizationSample, 0, len(benchmarkDimensions))
	for _, entry := range []struct {
		name      string
		dimension BudgetDimension
	}{
		{"iteration", projection.Remaining.Iteration},
		{"tool_call", projection.Remaining.ToolCall},
		{"time_ms", projection.Remaining.TimeMS},
		{"cost_headroom", projection.Remaining.CostHeadroom},
	} {
		if !entry.dimension.Available {
			continue
		}
		samples = append(samples, BudgetUtilizationSample{
			Dimension: entry.name,
			Ratio:     float64Value(entry.dimension.Ratio),
		})
	}
	return samples
}

// BudgetBenchmarkDigest returns the SHA-256 digest of the canonical benchmark
// serialization. The Digest field itself is excluded.
func BudgetBenchmarkDigest(metrics BudgetBenchmarkMetrics) string {
	sum := sha256.Sum256([]byte(canonicalBudgetBenchmark(metrics)))
	return hex.EncodeToString(sum[:])
}

func canonicalBudgetBenchmark(metrics BudgetBenchmarkMetrics) string {
	var builder strings.Builder
	builder.WriteString("budget_benchmark.v1")
	builder.WriteString("|traces=")
	builder.WriteString(strconv.Itoa(metrics.TraceCount))
	builder.WriteString("|steps=")
	builder.WriteString(strconv.Itoa(metrics.StepCount))
	builder.WriteString("|completion=")
	builder.WriteString(formatFloat(metrics.CompletionRate))
	builder.WriteString("|repeat=")
	builder.WriteString(formatFloat(metrics.RepeatedLoopRate))
	builder.WriteString("|saturation=")
	builder.WriteString(formatFloat(metrics.SaturationRate))
	builder.WriteString("|utilization=")
	builder.WriteString(strconv.Itoa(len(metrics.Utilization)))
	for _, sample := range metrics.Utilization {
		builder.WriteString(":")
		builder.WriteString(sample.Dimension)
		builder.WriteString("=")
		builder.WriteString(formatFloat(sample.Ratio))
		builder.WriteString("/")
		builder.WriteString(strconv.Itoa(sample.Samples))
	}
	return builder.String()
}

// ValidateBudgetBenchmark compares recomputed metrics against a fixture
// expectation and returns the deterministic drift codes. A second evaluation of
// the same traces MUST reproduce the same digest; otherwise recovery recompute
// drift is reported.
func ValidateBudgetBenchmark(traces []BudgetBenchmarkTrace, expected BudgetBenchmarkMetrics) []string {
	first, err := EvaluateBudgetBenchmark(traces)
	if err != nil {
		if typed, ok := err.(*Error); ok {
			return []string{typed.Code}
		}
		return []string{CodeBenchmarkSchemaDrift}
	}
	second, err := EvaluateBudgetBenchmark(traces)
	if err != nil {
		if typed, ok := err.(*Error); ok {
			return []string{typed.Code}
		}
		return []string{CodeBenchmarkSchemaDrift}
	}
	codes := make([]string, 0, 3)
	if first.Digest != second.Digest {
		codes = append(codes, CodeBenchmarkRecoveryRecompute)
	}
	if expected.Digest != "" && expected.Digest != first.Digest {
		codes = append(codes, CodeBenchmarkMetricMismatch)
	}
	if expected.TraceCount != 0 && expected.TraceCount != first.TraceCount {
		codes = append(codes, CodeBenchmarkMetricMismatch)
	}
	if expected.StepCount != 0 && expected.StepCount != first.StepCount {
		codes = append(codes, CodeBenchmarkMetricMismatch)
	}
	return dedupeStable(codes)
}

func isKnownOutcome(outcome string) bool {
	for _, candidate := range BudgetBenchmarkOutcomes {
		if candidate == outcome {
			return true
		}
	}
	return false
}
