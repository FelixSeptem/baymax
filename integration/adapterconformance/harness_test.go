package adapterconformance

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	adaptermanifest "github.com/FelixSeptem/baymax/adapter/manifest"
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

func TestAdapterConformanceManifestActivationSuccess(t *testing.T) {
	root := repoRoot(t)
	path := filepath.Join(root, "integration", "testdata", "adapter-scaffold", "model-fixture", "adapter-manifest.json")
	result, err := ActivateAdapterManifest(path, "0.26.0-rc.2", "model-run-stream-downgrade", []string{
		"model.run_stream.semantic_equivalent",
		"model.response.mandatory_fields",
		"model.capability.token_count",
	})
	if err != nil {
		t.Fatalf("activate adapter manifest: %v", err)
	}
	if len(result.OptionalDowngrades) != 0 {
		t.Fatalf("unexpected downgrades: %#v", result.OptionalDowngrades)
	}
}

func TestAdapterConformanceManifestCompatibilityFailFast(t *testing.T) {
	root := repoRoot(t)
	path := filepath.Join(root, "integration", "testdata", "adapter-scaffold", "mcp-fixture", "adapter-manifest.json")
	_, err := ActivateAdapterManifest(path, "0.27.0", "mcp-normalization-fail-fast", []string{
		"mcp.invoke.required_input",
		"mcp.response.normalized",
	})
	if err == nil {
		t.Fatal("expected compatibility mismatch")
	}
	ce := contractErr(t, err)
	if ce.Code != adaptermanifest.CodeCompatibilityMismatch {
		t.Fatalf("unexpected compatibility error classification: %#v", ce)
	}
}

func TestAdapterConformanceManifestRequiredCapabilityFailFast(t *testing.T) {
	root := repoRoot(t)
	path := filepath.Join(root, "integration", "testdata", "adapter-scaffold", "tool-fixture", "adapter-manifest.json")
	_, err := ActivateAdapterManifest(path, "0.26.0-rc.2", "tool-invoke-fail-fast", []string{
		"tool.schema.rich_validation",
	})
	if err == nil {
		t.Fatal("expected required capability mismatch")
	}
	ce := contractErr(t, err)
	if ce.Code != adaptermanifest.CodeRequiredCapabilityMissing {
		t.Fatalf("unexpected required capability classification: %#v", ce)
	}
}

func TestAdapterConformanceManifestOptionalCapabilityDowngradeDeterministic(t *testing.T) {
	root := repoRoot(t)
	path := filepath.Join(root, "integration", "testdata", "adapter-scaffold", "model-fixture", "adapter-manifest.json")
	res1, err1 := ActivateAdapterManifest(path, "0.26.0-rc.2", "model-run-stream-downgrade", []string{
		"model.run_stream.semantic_equivalent",
		"model.response.mandatory_fields",
	})
	res2, err2 := ActivateAdapterManifest(path, "0.26.0-rc.2", "model-run-stream-downgrade", []string{
		"model.run_stream.semantic_equivalent",
		"model.response.mandatory_fields",
	})
	if err1 != nil || err2 != nil {
		t.Fatalf("unexpected activation error err1=%v err2=%v", err1, err2)
	}
	if len(res1.OptionalDowngrades) != 1 || len(res2.OptionalDowngrades) != 1 {
		t.Fatalf("expected single optional downgrade, got %#v %#v", res1.OptionalDowngrades, res2.OptionalDowngrades)
	}
	if res1.OptionalDowngrades[0] != res2.OptionalDowngrades[0] {
		t.Fatalf("non-deterministic optional downgrade reason: %#v vs %#v", res1.OptionalDowngrades[0], res2.OptionalDowngrades[0])
	}
	if !strings.HasPrefix(res1.OptionalDowngrades[0].ReasonCode, "adapter.manifest.capability.optional_missing.") {
		t.Fatalf("unexpected downgrade reason code: %s", res1.OptionalDowngrades[0].ReasonCode)
	}
}

func TestAdapterConformanceManifestProfileAlignmentForFixtures(t *testing.T) {
	root := repoRoot(t)
	testCases := []struct {
		fixtureDir string
		scenarioID string
	}{
		{fixtureDir: "mcp-fixture", scenarioID: "mcp-normalization-fail-fast"},
		{fixtureDir: "model-fixture", scenarioID: "model-run-stream-downgrade"},
		{fixtureDir: "tool-fixture", scenarioID: "tool-invoke-fail-fast"},
	}
	for _, tc := range testCases {
		tc := tc
		t.Run(tc.fixtureDir, func(t *testing.T) {
			dir := filepath.Join(root, "integration", "testdata", "adapter-scaffold", tc.fixtureDir)
			if err := ValidateManifestProfileAlignmentForScaffold(dir, tc.scenarioID); err != nil {
				t.Fatalf("fixture manifest alignment failed: %v", err)
			}
		})
	}
}

func contractErr(t *testing.T, err error) *adaptermanifest.ContractError {
	t.Helper()
	ce := &adaptermanifest.ContractError{}
	if !errors.As(err, &ce) {
		t.Fatalf("expected manifest ContractError, got %T (%v)", err, err)
	}
	return ce
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
