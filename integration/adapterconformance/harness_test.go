package adapterconformance

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/FelixSeptem/baymax/core/types"
)

func TestAdapterConformanceMinimumMatrixCoverage(t *testing.T) {
	if err := ValidateMinimumMatrix(MinimumMatrix); err != nil {
		t.Fatalf("minimum conformance matrix invalid: %v", err)
	}
}

func TestAdapterConformanceMCPNormalizationAndFailFast(t *testing.T) {
	client := newOfflineMCPClient()
	defer func() { _ = client.Close() }()

	okRes, err := client.CallTool(context.Background(), "echo", map[string]any{"input": "hello"})
	if err != nil {
		t.Fatalf("mcp call failed: %v", err)
	}
	if okRes.Content != "tool=echo input=hello" {
		t.Fatalf("unexpected normalized mcp response: %s", okRes.Content)
	}
	if err := ValidateReasonCode("mcp.response.normalized"); err != nil {
		t.Fatalf("unexpected reason taxonomy failure: %v", err)
	}

	badRes1, badErr1 := client.CallTool(context.Background(), "echo", map[string]any{})
	if badErr1 != nil {
		t.Fatalf("expected deterministic result classification path, got transport error: %v", badErr1)
	}
	if badRes1.Error == nil {
		t.Fatal("expected fail-fast classification for invalid mandatory input")
	}
	c1 := ClassifyAdapterResult(badRes1, badErr1, types.ErrMCP, "mcp.invoke.execution_failed")
	if c1.Class != types.ErrMCP || c1.ReasonCode != "mcp.validation.missing_required_input" {
		t.Fatalf("unexpected classification: %+v", c1)
	}
	if err := ValidateReasonCode(c1.ReasonCode); err != nil {
		t.Fatalf("invalid reason taxonomy for mcp fail-fast: %v", err)
	}

	badRes2, badErr2 := client.CallTool(context.Background(), "echo", map[string]any{})
	c2 := ClassifyAdapterResult(badRes2, badErr2, types.ErrMCP, "mcp.invoke.execution_failed")
	if !IsDeterministicClassification(c1, c2) {
		t.Fatalf("non-deterministic classification: first=%+v second=%+v", c1, c2)
	}
}

func TestAdapterConformanceModelRunStreamAndDowngrade(t *testing.T) {
	runAnswer, streamAnswer, err := runAndStreamFinalAnswer(context.Background(), equivalentModelAdapter{}, "ping")
	if err != nil {
		t.Fatalf("run/stream execution failed: %v", err)
	}
	if NormalizeSemanticText(runAnswer) != NormalizeSemanticText(streamAnswer) {
		t.Fatalf("run/stream semantic drift: run=%q stream=%q", runAnswer, streamAnswer)
	}
	if err := ValidateReasonCode("model.run_stream.semantic_equivalent"); err != nil {
		t.Fatalf("invalid model equivalence reason taxonomy: %v", err)
	}

	dg := EvaluateOptionalTokenCount(false)
	if !dg.Downgraded || dg.ReasonCode != "model.capability.token_count_unsupported_downgrade" {
		t.Fatalf("unexpected downgrade semantics: %+v", dg)
	}
	if err := ValidateReasonCode(dg.ReasonCode); err != nil {
		t.Fatalf("invalid downgrade reason taxonomy: %v", err)
	}

	malformedResp, err := malformedModelAdapter{}.Generate(context.Background(), types.ModelRequest{Input: "malformed"})
	if err != nil {
		t.Fatalf("unexpected malformed fixture generate error: %v", err)
	}
	classifiedErr1 := ValidateMandatoryModelResponse(malformedResp)
	if classifiedErr1 == nil {
		t.Fatal("expected fail-fast malformed model response classification")
	}
	c1 := ClassifyAdapterResult(types.ToolResult{}, classifiedErr1, types.ErrModel, "model.response.invalid")
	if c1.Class != types.ErrModel || c1.ReasonCode != "model.response.malformed" {
		t.Fatalf("unexpected model malformed classification: %+v", c1)
	}
	classifiedErr2 := ValidateMandatoryModelResponse(malformedResp)
	c2 := ClassifyAdapterResult(types.ToolResult{}, classifiedErr2, types.ErrModel, "model.response.invalid")
	if !IsDeterministicClassification(c1, c2) {
		t.Fatalf("non-deterministic model classification: first=%+v second=%+v", c1, c2)
	}
}

func TestAdapterConformanceToolInvocationAndFailFast(t *testing.T) {
	tool := requiredInputToolAdapter{}

	okRes, err := tool.Invoke(context.Background(), map[string]any{"input": "hello-tool"})
	if err != nil {
		t.Fatalf("tool invoke failed: %v", err)
	}
	if okRes.Content != "echo=hello-tool" {
		t.Fatalf("unexpected tool response: %s", okRes.Content)
	}
	if err := ValidateReasonCode("tool.invoke.contract_satisfied"); err != nil {
		t.Fatalf("invalid tool success reason taxonomy: %v", err)
	}

	_, invokeErr1 := tool.Invoke(context.Background(), map[string]any{})
	if invokeErr1 == nil {
		t.Fatal("expected fail-fast for missing tool mandatory input")
	}
	c1 := ClassifyAdapterResult(types.ToolResult{}, invokeErr1, types.ErrTool, "tool.invoke.execution_failed")
	if c1.Class != types.ErrTool || c1.ReasonCode != "tool.validation.missing_required_input" {
		t.Fatalf("unexpected tool fail-fast classification: %+v", c1)
	}

	_, invokeErr2 := tool.Invoke(context.Background(), map[string]any{})
	c2 := ClassifyAdapterResult(types.ToolResult{}, invokeErr2, types.ErrTool, "tool.invoke.execution_failed")
	if !IsDeterministicClassification(c1, c2) {
		t.Fatalf("non-deterministic tool classification: first=%+v second=%+v", c1, c2)
	}
}

func TestAdapterConformanceReasonTaxonomyAndFailureClassification(t *testing.T) {
	reasons := []string{
		"mcp.response.normalized",
		"mcp.validation.missing_required_input",
		"model.run_stream.semantic_equivalent",
		"model.response.malformed",
		"tool.validation.missing_required_input",
	}
	for _, reason := range reasons {
		if err := ValidateReasonCode(reason); err != nil {
			t.Fatalf("reason taxonomy validation failed for %s: %v", reason, err)
		}
	}
	if err := ValidateReasonCode("BAD_REASON_CODE"); err == nil {
		t.Fatal("expected invalid reason taxonomy to fail")
	}
}

func TestAdapterConformanceTemplateTraceabilityAndDriftGuard(t *testing.T) {
	root := repoRoot(t)
	indexDoc := mustRead(t, filepath.Join(root, "docs", "external-adapter-template-index.md"))
	mappingDoc := mustRead(t, filepath.Join(root, "docs", "adapter-migration-mapping.md"))

	for _, scenario := range MinimumMatrix {
		templateAbs := filepath.Join(root, filepath.FromSlash(scenario.TemplatePath))
		if _, err := os.Stat(templateAbs); err != nil {
			t.Fatalf("template path does not exist for scenario %s: %s (%v)", scenario.ID, scenario.TemplatePath, err)
		}
		if !containsTemplatePath(indexDoc, scenario.TemplatePath) {
			t.Fatalf("template index drift: missing path %s for scenario %s", scenario.TemplatePath, scenario.ID)
		}
		if !containsTemplatePath(mappingDoc, "additive + nullable + default + fail-fast") {
			t.Fatalf("mapping doc drift: missing unified compatibility boundary marker")
		}
	}
}

func mustRead(t *testing.T, path string) string {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(raw)
}

func repoRoot(t *testing.T) string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve runtime caller failed")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(thisFile), "..", ".."))
}
