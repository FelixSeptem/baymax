package budgetprojection

import (
	"testing"
)

func benchmarkTraces() []BudgetBenchmarkTrace {
	return []BudgetBenchmarkTrace{
		{
			TraceID: "trace-success",
			Steps: []BudgetBenchmarkStep{
				{Outcome: OutcomeProgress, Facts: fullFacts()},
				{Outcome: OutcomeProgress, Facts: fullFacts()},
				{Outcome: OutcomeTerminalSuccess, Facts: fullFacts()},
			},
		},
		{
			TraceID: "trace-loop",
			Steps: []BudgetBenchmarkStep{
				{Outcome: OutcomeProgress, Facts: fullFacts()},
				{Outcome: OutcomeRepeat, Facts: fullFacts()},
				{Outcome: OutcomeRepeat, Facts: fullFacts()},
				{Outcome: OutcomeTerminalExhausted, Facts: fullFacts()},
			},
		},
	}
}

func TestEvaluateBudgetBenchmarkIsDeterministic(t *testing.T) {
	first, err := EvaluateBudgetBenchmark(benchmarkTraces())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	second, err := EvaluateBudgetBenchmark(benchmarkTraces())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if first.Digest != second.Digest {
		t.Fatalf("benchmark digest not deterministic: %s vs %s", first.Digest, second.Digest)
	}
	if first.TraceCount != 2 {
		t.Fatalf("trace count = %d, want 2", first.TraceCount)
	}
	if first.StepCount != 7 {
		t.Fatalf("step count = %d, want 7", first.StepCount)
	}
	if first.CompletionRate != 0.5 {
		t.Fatalf("completion rate = %v, want 0.5", first.CompletionRate)
	}
	if first.SaturationRate != 0.5 {
		t.Fatalf("saturation rate = %v, want 0.5", first.SaturationRate)
	}
	if got := first.RepeatedLoopRate; got < 0.2857 || got > 0.2858 {
		t.Fatalf("repeated loop rate = %v, want ~0.2857", got)
	}
	if len(first.Utilization) != len(benchmarkDimensions) {
		t.Fatalf("utilization dimensions = %d, want %d", len(first.Utilization), len(benchmarkDimensions))
	}
}

func TestEvaluateBudgetBenchmarkUtilizationIsStablyOrdered(t *testing.T) {
	metrics, err := EvaluateBudgetBenchmark(benchmarkTraces())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []string{"cost_headroom", "iteration", "time_ms", "tool_call"}
	for index, dimension := range want {
		if metrics.Utilization[index].Dimension != dimension {
			t.Fatalf("utilization[%d] = %q, want %q", index, metrics.Utilization[index].Dimension, dimension)
		}
	}
}

func TestEvaluateBudgetBenchmarkRejectsInvalidTraces(t *testing.T) {
	cases := []struct {
		name   string
		traces []BudgetBenchmarkTrace
		want   string
	}{
		{"empty", nil, CodeBenchmarkSchemaDrift},
		{"missing_trace_id", []BudgetBenchmarkTrace{{Steps: []BudgetBenchmarkStep{{Outcome: OutcomeProgress}}}}, CodeBenchmarkSchemaDrift},
		{"no_steps", []BudgetBenchmarkTrace{{TraceID: "t"}}, CodeBenchmarkSchemaDrift},
		{"unknown_outcome", []BudgetBenchmarkTrace{{TraceID: "t", Steps: []BudgetBenchmarkStep{{Outcome: "explode"}}}}, CodeBenchmarkSchemaDrift},
	}
	for _, item := range cases {
		if _, err := EvaluateBudgetBenchmark(item.traces); err == nil {
			t.Fatalf("%s: expected error", item.name)
		} else if typed, ok := err.(*Error); !ok || typed.Code != item.want {
			t.Fatalf("%s: got %v, want %q", item.name, err, item.want)
		}
	}
}

func TestEvaluateBudgetBenchmarkRejectsOverflow(t *testing.T) {
	traces := make([]BudgetBenchmarkTrace, 0, MaxTraces+1)
	for index := 0; index <= MaxTraces; index++ {
		traces = append(traces, BudgetBenchmarkTrace{
			TraceID: "t",
			Steps:   []BudgetBenchmarkStep{{Outcome: OutcomeProgress}},
		})
	}
	if _, err := EvaluateBudgetBenchmark(traces); err == nil {
		t.Fatalf("expected overflow error")
	} else if typed, ok := err.(*Error); !ok || typed.Code != CodeBenchmarkOverflowDrift {
		t.Fatalf("got %v, want %q", err, CodeBenchmarkOverflowDrift)
	}

	steps := make([]BudgetBenchmarkStep, 0, MaxStepsPerTrace+1)
	for index := 0; index <= MaxStepsPerTrace; index++ {
		steps = append(steps, BudgetBenchmarkStep{Outcome: OutcomeProgress})
	}
	if _, err := EvaluateBudgetBenchmark([]BudgetBenchmarkTrace{{TraceID: "t", Steps: steps}}); err == nil {
		t.Fatalf("expected per-trace overflow error")
	} else if typed, ok := err.(*Error); !ok || typed.Code != CodeBenchmarkOverflowDrift {
		t.Fatalf("got %v, want %q", err, CodeBenchmarkOverflowDrift)
	}
}

func TestValidateBudgetBenchmarkDetectsMetricMismatchAndRecoveryDrift(t *testing.T) {
	metrics, err := EvaluateBudgetBenchmark(benchmarkTraces())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if codes := ValidateBudgetBenchmark(benchmarkTraces(), metrics); len(codes) != 0 {
		t.Fatalf("expected no drift, got %v", codes)
	}

	expected := metrics
	expected.Digest = "deadbeef"
	if codes := ValidateBudgetBenchmark(benchmarkTraces(), expected); !containsCode(codes, CodeBenchmarkMetricMismatch) {
		t.Fatalf("expected %q in %v", CodeBenchmarkMetricMismatch, codes)
	}

	expected = metrics
	expected.StepCount = 99
	if codes := ValidateBudgetBenchmark(benchmarkTraces(), expected); !containsCode(codes, CodeBenchmarkMetricMismatch) {
		t.Fatalf("expected %q in %v", CodeBenchmarkMetricMismatch, codes)
	}

	if _, err := EvaluateBudgetBenchmark([]BudgetBenchmarkTrace{}); err == nil {
		t.Fatalf("expected schema drift for empty traces")
	} else if typed, ok := err.(*Error); !ok || typed.Code != CodeBenchmarkSchemaDrift {
		t.Fatalf("got %v, want %q", err, CodeBenchmarkSchemaDrift)
	}
}

func TestValidateBudgetBenchmarkCasePinsDeclaredGaps(t *testing.T) {
	benchmark := BudgetBenchmarkCase{
		BenchmarkID:  "budget-utilization",
		Traces:       benchmarkTraces(),
		Expected:     mustBenchmarkMetrics(t, benchmarkTraces()),
		DeclaredGaps: []string{},
	}
	if codes := ValidateBudgetBenchmarkCase(benchmark); len(codes) != 0 {
		t.Fatalf("expected pinned benchmark to validate, got %v", codes)
	}

	benchmark.Traces = []BudgetBenchmarkTrace{}
	if codes := ValidateBudgetBenchmarkCase(benchmark); !containsCode(codes, CodeBenchmarkSchemaDrift) {
		t.Fatalf("expected %q in %v", CodeBenchmarkSchemaDrift, codes)
	}
}

func mustBenchmarkMetrics(t *testing.T, traces []BudgetBenchmarkTrace) BudgetBenchmarkMetrics {
	t.Helper()
	metrics, err := EvaluateBudgetBenchmark(traces)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	return metrics
}
