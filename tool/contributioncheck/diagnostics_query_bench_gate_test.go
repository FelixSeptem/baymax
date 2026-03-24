package contributioncheck

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func TestDiagnosticsQueryBenchmarkRegressionScriptsContainFailFastCoverage(t *testing.T) {
	root := repoRoot(t)
	shellPath := filepath.Join(root, "scripts", "check-diagnostics-query-performance-regression.sh")
	psPath := filepath.Join(root, "scripts", "check-diagnostics-query-performance-regression.ps1")

	shellRaw, err := os.ReadFile(shellPath)
	if err != nil {
		t.Fatalf("read shell script: %v", err)
	}
	psRaw, err := os.ReadFile(psPath)
	if err != nil {
		t.Fatalf("read ps script: %v", err)
	}
	shellContent := string(shellRaw)
	psContent := string(psRaw)

	requiredTokens := []string{
		"BenchmarkDiagnosticsQueryRuns",
		"BenchmarkDiagnosticsQueryMailbox",
		"BenchmarkDiagnosticsMailboxAggregates",
		"missing_output_line",
		"missing_required_metric",
		"parse-failure",
		"BAYMAX_DIAGNOSTICS_QUERY_BENCH_COUNT",
		"BAYMAX_DIAGNOSTICS_QUERY_BENCH_MAX_NS_DEGRADATION_PCT",
		"BAYMAX_DIAGNOSTICS_QUERY_BENCH_MAX_P95_DEGRADATION_PCT",
		"BAYMAX_DIAGNOSTICS_QUERY_BENCH_MAX_ALLOCS_DEGRADATION_PCT",
		"ns/op",
		"p95-ns/op",
		"allocs/op",
	}
	for _, token := range requiredTokens {
		if !strings.Contains(shellContent, token) {
			t.Fatalf("shell script missing required fail-fast/parsing token %q", token)
		}
		if !strings.Contains(psContent, token) {
			t.Fatalf("powershell script missing required fail-fast/parsing token %q", token)
		}
	}
}

func TestDiagnosticsQueryBenchmarkBaselineEnvKeysAndValues(t *testing.T) {
	root := repoRoot(t)
	path := filepath.Join(root, "scripts", "diagnostics-query-benchmark-baseline.env")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read baseline env: %v", err)
	}
	values := parseSimpleEnvKV(t, string(raw))

	required := []string{
		"BAYMAX_DIAGNOSTICS_QUERY_BENCH_ENABLED",
		"BAYMAX_DIAGNOSTICS_QUERY_BENCH_BENCHTIME",
		"BAYMAX_DIAGNOSTICS_QUERY_BENCH_COUNT",
		"BAYMAX_DIAGNOSTICS_QUERY_BENCH_MAX_NS_DEGRADATION_PCT",
		"BAYMAX_DIAGNOSTICS_QUERY_BENCH_MAX_P95_DEGRADATION_PCT",
		"BAYMAX_DIAGNOSTICS_QUERY_BENCH_MAX_ALLOCS_DEGRADATION_PCT",
		"BAYMAX_DIAGNOSTICS_QUERY_BENCH_BASELINE_QUERY_RUNS_NS_OP",
		"BAYMAX_DIAGNOSTICS_QUERY_BENCH_BASELINE_QUERY_RUNS_P95_NS_OP",
		"BAYMAX_DIAGNOSTICS_QUERY_BENCH_BASELINE_QUERY_RUNS_ALLOCS_OP",
		"BAYMAX_DIAGNOSTICS_QUERY_BENCH_BASELINE_QUERY_MAILBOX_NS_OP",
		"BAYMAX_DIAGNOSTICS_QUERY_BENCH_BASELINE_QUERY_MAILBOX_P95_NS_OP",
		"BAYMAX_DIAGNOSTICS_QUERY_BENCH_BASELINE_QUERY_MAILBOX_ALLOCS_OP",
		"BAYMAX_DIAGNOSTICS_QUERY_BENCH_BASELINE_MAILBOX_AGGREGATES_NS_OP",
		"BAYMAX_DIAGNOSTICS_QUERY_BENCH_BASELINE_MAILBOX_AGGREGATES_P95_NS_OP",
		"BAYMAX_DIAGNOSTICS_QUERY_BENCH_BASELINE_MAILBOX_AGGREGATES_ALLOCS_OP",
	}
	for _, key := range required {
		value, ok := values[key]
		if !ok {
			t.Fatalf("missing required baseline key %q", key)
		}
		if strings.TrimSpace(value) == "" {
			t.Fatalf("baseline key %q must not be empty", key)
		}
	}

	if strings.TrimSpace(values["BAYMAX_DIAGNOSTICS_QUERY_BENCH_ENABLED"]) != "true" {
		t.Fatalf("BAYMAX_DIAGNOSTICS_QUERY_BENCH_ENABLED must default to true")
	}

	count, err := strconv.Atoi(values["BAYMAX_DIAGNOSTICS_QUERY_BENCH_COUNT"])
	if err != nil || count <= 0 {
		t.Fatalf("BAYMAX_DIAGNOSTICS_QUERY_BENCH_COUNT must be positive integer, got %q", values["BAYMAX_DIAGNOSTICS_QUERY_BENCH_COUNT"])
	}

	if strings.TrimSpace(values["BAYMAX_DIAGNOSTICS_QUERY_BENCH_BENCHTIME"]) == "" {
		t.Fatalf("BAYMAX_DIAGNOSTICS_QUERY_BENCH_BENCHTIME must not be empty")
	}

	numericKeys := []string{
		"BAYMAX_DIAGNOSTICS_QUERY_BENCH_MAX_NS_DEGRADATION_PCT",
		"BAYMAX_DIAGNOSTICS_QUERY_BENCH_MAX_P95_DEGRADATION_PCT",
		"BAYMAX_DIAGNOSTICS_QUERY_BENCH_MAX_ALLOCS_DEGRADATION_PCT",
		"BAYMAX_DIAGNOSTICS_QUERY_BENCH_BASELINE_QUERY_RUNS_NS_OP",
		"BAYMAX_DIAGNOSTICS_QUERY_BENCH_BASELINE_QUERY_RUNS_P95_NS_OP",
		"BAYMAX_DIAGNOSTICS_QUERY_BENCH_BASELINE_QUERY_RUNS_ALLOCS_OP",
		"BAYMAX_DIAGNOSTICS_QUERY_BENCH_BASELINE_QUERY_MAILBOX_NS_OP",
		"BAYMAX_DIAGNOSTICS_QUERY_BENCH_BASELINE_QUERY_MAILBOX_P95_NS_OP",
		"BAYMAX_DIAGNOSTICS_QUERY_BENCH_BASELINE_QUERY_MAILBOX_ALLOCS_OP",
		"BAYMAX_DIAGNOSTICS_QUERY_BENCH_BASELINE_MAILBOX_AGGREGATES_NS_OP",
		"BAYMAX_DIAGNOSTICS_QUERY_BENCH_BASELINE_MAILBOX_AGGREGATES_P95_NS_OP",
		"BAYMAX_DIAGNOSTICS_QUERY_BENCH_BASELINE_MAILBOX_AGGREGATES_ALLOCS_OP",
	}
	for _, key := range numericKeys {
		v := strings.TrimSpace(values[key])
		parsed, parseErr := strconv.ParseFloat(v, 64)
		if parseErr != nil || parsed <= 0 {
			t.Fatalf("%s must be positive numeric value, got %q", key, v)
		}
	}
}

func parseSimpleEnvKV(t *testing.T, content string) map[string]string {
	t.Helper()
	out := map[string]string{}
	lines := strings.Split(content, "\n")
	for idx := range lines {
		line := strings.TrimSpace(lines[idx])
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			t.Fatalf("invalid env line %d: %q", idx+1, lines[idx])
		}
		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])
		if key == "" {
			t.Fatalf("empty env key at line %d", idx+1)
		}
		out[key] = val
	}
	return out
}
