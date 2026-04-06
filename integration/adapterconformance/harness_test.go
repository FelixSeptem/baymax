package adapterconformance

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
	"time"

	adaptercap "github.com/FelixSeptem/baymax/adapter/capability"
	adapterhealth "github.com/FelixSeptem/baymax/adapter/health"
	adaptermanifest "github.com/FelixSeptem/baymax/adapter/manifest"
	"github.com/FelixSeptem/baymax/core/types"
	runtimeconfig "github.com/FelixSeptem/baymax/runtime/config"
	runtimediag "github.com/FelixSeptem/baymax/runtime/diagnostics"
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
	if result.ContractProfileVersion != "v1alpha1" {
		t.Fatalf("unexpected contract profile version: %#v", result.ContractProfileVersion)
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
	req := adaptercap.Request{
		Required:         []string{"model.run_stream.semantic_equivalent", "model.response.mandatory_fields"},
		Optional:         []string{"model.capability.token_count"},
		StrategyOverride: adaptercap.StrategyBestEffort,
	}
	res1, err1 := ActivateAdapterManifestWithRequest(path, "0.26.0-rc.2", "model-run-stream-downgrade", []string{
		"model.run_stream.semantic_equivalent",
		"model.response.mandatory_fields",
	}, req)
	res2, err2 := ActivateAdapterManifestWithRequest(path, "0.26.0-rc.2", "model-run-stream-downgrade", []string{
		"model.run_stream.semantic_equivalent",
		"model.response.mandatory_fields",
	}, req)
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
	if res1.StrategyApplied != adaptercap.StrategyBestEffort || !res1.StrategyOverride {
		t.Fatalf("unexpected strategy diagnostics: %#v", res1)
	}
	if !containsReason(res1.ReasonCodes, adaptercap.ReasonOptionalDowngraded) || !containsReason(res1.ReasonCodes, adaptercap.ReasonStrategyOverrideApply) {
		t.Fatalf("unexpected negotiation reasons: %#v", res1.ReasonCodes)
	}
	if res1.ContractProfileVersion != "v1alpha1" {
		t.Fatalf("unexpected contract profile version: %#v", res1.ContractProfileVersion)
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

func TestAdapterConformanceManifestNegotiationDefaultFailFast(t *testing.T) {
	root := repoRoot(t)
	path := filepath.Join(root, "integration", "testdata", "adapter-scaffold", "model-fixture", "adapter-manifest.json")
	_, err := ActivateAdapterManifestWithRequest(path, "0.26.0-rc.2", "model-run-stream-downgrade", []string{
		"model.run_stream.semantic_equivalent",
		"model.response.mandatory_fields",
	}, adaptercap.Request{
		Required: []string{"model.run_stream.semantic_equivalent", "model.response.mandatory_fields"},
		Optional: []string{"model.capability.token_count"},
	})
	if err == nil {
		t.Fatal("expected fail_fast to reject missing optional capability request")
	}
	ce := contractErr(t, err)
	if ce.Code != adaptermanifest.CodeRequiredCapabilityMissing {
		t.Fatalf("unexpected error code: %#v", ce)
	}
}

func TestAdapterConformanceManifestNegotiationRunStreamSemanticEquivalent(t *testing.T) {
	root := repoRoot(t)
	path := filepath.Join(root, "integration", "testdata", "adapter-scaffold", "model-fixture", "adapter-manifest.json")

	runAccepted, runErr := ActivateAdapterManifestWithRequest(path, "0.26.0-rc.2", "model-run-stream-downgrade", []string{
		"model.run_stream.semantic_equivalent",
		"model.response.mandatory_fields",
		"model.capability.token_count",
	}, adaptercap.Request{
		Required: []string{"model.run_stream.semantic_equivalent", "model.response.mandatory_fields"},
	})
	streamAccepted, streamErr := ActivateAdapterManifestWithRequest(path, "0.26.0-rc.2", "model-run-stream-downgrade", []string{
		"model.run_stream.semantic_equivalent",
		"model.response.mandatory_fields",
		"model.capability.token_count",
	}, adaptercap.Request{
		Required: []string{"model.run_stream.semantic_equivalent", "model.response.mandatory_fields"},
	})
	if runErr != nil || streamErr != nil {
		t.Fatalf("expected acceptance on both paths runErr=%v streamErr=%v", runErr, streamErr)
	}
	if !reflect.DeepEqual(runAccepted.ReasonCodes, streamAccepted.ReasonCodes) {
		t.Fatalf("accept reason mismatch run=%#v stream=%#v", runAccepted.ReasonCodes, streamAccepted.ReasonCodes)
	}
	if runAccepted.ContractProfileVersion != streamAccepted.ContractProfileVersion {
		t.Fatalf("contract profile mismatch run=%s stream=%s", runAccepted.ContractProfileVersion, streamAccepted.ContractProfileVersion)
	}

	runRejectRes, runRejectErrRaw := ActivateAdapterManifestWithRequest(path, "0.26.0-rc.2", "model-run-stream-downgrade", []string{
		"model.run_stream.semantic_equivalent",
	}, adaptercap.Request{
		Required: []string{"model.run_stream.semantic_equivalent", "model.response.mandatory_fields"},
	})
	runRejectErr := mustActivationError(t, runRejectRes, runRejectErrRaw)
	streamRejectRes, streamRejectErrRaw := ActivateAdapterManifestWithRequest(path, "0.26.0-rc.2", "model-run-stream-downgrade", []string{
		"model.run_stream.semantic_equivalent",
	}, adaptercap.Request{
		Required: []string{"model.run_stream.semantic_equivalent", "model.response.mandatory_fields"},
	})
	streamRejectErr := mustActivationError(t, streamRejectRes, streamRejectErrRaw)
	if runRejectErr.Code != streamRejectErr.Code || runRejectErr.Field != streamRejectErr.Field {
		t.Fatalf("reject classification mismatch run=%#v stream=%#v", runRejectErr, streamRejectErr)
	}

	runDowngrade, runDowngradeErr := ActivateAdapterManifestWithRequest(path, "0.26.0-rc.2", "model-run-stream-downgrade", []string{
		"model.run_stream.semantic_equivalent",
		"model.response.mandatory_fields",
	}, adaptercap.Request{
		Required:         []string{"model.run_stream.semantic_equivalent", "model.response.mandatory_fields"},
		Optional:         []string{"model.capability.token_count"},
		StrategyOverride: adaptercap.StrategyBestEffort,
	})
	streamDowngrade, streamDowngradeErr := ActivateAdapterManifestWithRequest(path, "0.26.0-rc.2", "model-run-stream-downgrade", []string{
		"model.run_stream.semantic_equivalent",
		"model.response.mandatory_fields",
	}, adaptercap.Request{
		Required:         []string{"model.run_stream.semantic_equivalent", "model.response.mandatory_fields"},
		Optional:         []string{"model.capability.token_count"},
		StrategyOverride: adaptercap.StrategyBestEffort,
	})
	if runDowngradeErr != nil || streamDowngradeErr != nil {
		t.Fatalf("downgrade path should succeed runErr=%v streamErr=%v", runDowngradeErr, streamDowngradeErr)
	}
	if !reflect.DeepEqual(runDowngrade.ReasonCodes, streamDowngrade.ReasonCodes) {
		t.Fatalf("downgrade reason mismatch run=%#v stream=%#v", runDowngrade.ReasonCodes, streamDowngrade.ReasonCodes)
	}
}

func TestAdapterConformanceManifestNegotiationInvalidStrategy(t *testing.T) {
	root := repoRoot(t)
	path := filepath.Join(root, "integration", "testdata", "adapter-scaffold", "model-fixture", "adapter-manifest.json")
	_, err := ActivateAdapterManifestWithRequest(path, "0.26.0-rc.2", "model-run-stream-downgrade", []string{
		"model.run_stream.semantic_equivalent",
		"model.response.mandatory_fields",
	}, adaptercap.Request{
		Required:         []string{"model.run_stream.semantic_equivalent", "model.response.mandatory_fields"},
		StrategyOverride: "invalid",
	})
	if err == nil {
		t.Fatal("expected invalid strategy error")
	}
	var ne *adaptercap.NegotiationError
	if !errors.As(err, &ne) {
		t.Fatalf("expected negotiation error, got %T (%v)", err, err)
	}
	if ne.Code != adaptercap.CodeInvalidStrategy {
		t.Fatalf("unexpected negotiation error code: %#v", ne)
	}
}

func TestAdapterConformanceHealthMatrixRequiredUnavailable(t *testing.T) {
	cfgPath := filepath.Join(t.TempDir(), "runtime-a43.yaml")
	writeAdapterHealthConfig(t, cfgPath, true)
	mgr, err := runtimeconfig.NewManager(runtimeconfig.ManagerOptions{FilePath: cfgPath, EnvPrefix: "BAYMAX_A43_CONFORMANCE"})
	if err != nil {
		t.Fatalf("new runtime manager: %v", err)
	}
	defer func() { _ = mgr.Close() }()

	mgr.SetAdapterHealthTargets([]runtimeconfig.AdapterHealthTarget{
		{
			Name:     "required-adapter",
			Required: true,
			Probe: adapterhealth.ProbeFunc(func(context.Context) (adapterhealth.Result, error) {
				return adapterhealth.Result{
					Status:  adapterhealth.StatusUnavailable,
					Code:    adapterhealth.CodeProbeFailed,
					Message: "fixture unavailable",
				}, nil
			}),
		},
	})

	result := mgr.ReadinessPreflight()
	if result.Status != runtimeconfig.ReadinessStatusBlocked {
		t.Fatalf("status=%q, want blocked", result.Status)
	}
	assertReadinessFindingCode(t, result.Findings, runtimeconfig.ReadinessCodeAdapterRequiredUnavailable)
	if err := ValidateReasonCode(runtimeconfig.ReadinessCodeAdapterRequiredUnavailable); err != nil {
		t.Fatalf("required unavailable reason taxonomy invalid: %v", err)
	}
}

func TestAdapterConformanceHealthMatrixOptionalUnavailableDowngradeDeterministic(t *testing.T) {
	cfgPath := filepath.Join(t.TempDir(), "runtime-a43.yaml")
	writeAdapterHealthConfig(t, cfgPath, false)
	mgr, err := runtimeconfig.NewManager(runtimeconfig.ManagerOptions{FilePath: cfgPath, EnvPrefix: "BAYMAX_A43_CONFORMANCE"})
	if err != nil {
		t.Fatalf("new runtime manager: %v", err)
	}
	defer func() { _ = mgr.Close() }()

	mgr.SetAdapterHealthTargets([]runtimeconfig.AdapterHealthTarget{
		{
			Name:     "optional-adapter",
			Required: false,
			Probe: adapterhealth.ProbeFunc(func(context.Context) (adapterhealth.Result, error) {
				return adapterhealth.Result{
					Status:  adapterhealth.StatusUnavailable,
					Code:    adapterhealth.CodeProbeFailed,
					Message: "fixture unavailable optional",
				}, nil
			}),
		},
	})

	first := mgr.ReadinessPreflight()
	second := mgr.ReadinessPreflight()
	if first.Status != runtimeconfig.ReadinessStatusDegraded {
		t.Fatalf("status=%q, want degraded", first.Status)
	}
	assertReadinessFindingCode(t, first.Findings, runtimeconfig.ReadinessCodeAdapterOptionalUnavailable)
	if readinessFingerprint(first) != readinessFingerprint(second) {
		t.Fatalf("optional unavailable downgrade must be deterministic")
	}
}

func TestAdapterConformanceHealthMatrixDegradedVisibility(t *testing.T) {
	cfgPath := filepath.Join(t.TempDir(), "runtime-a43.yaml")
	writeAdapterHealthConfig(t, cfgPath, false)
	mgr, err := runtimeconfig.NewManager(runtimeconfig.ManagerOptions{FilePath: cfgPath, EnvPrefix: "BAYMAX_A43_CONFORMANCE"})
	if err != nil {
		t.Fatalf("new runtime manager: %v", err)
	}
	defer func() { _ = mgr.Close() }()

	mgr.SetAdapterHealthTargets([]runtimeconfig.AdapterHealthTarget{
		{
			Name:     "degraded-adapter",
			Required: false,
			Probe: adapterhealth.ProbeFunc(func(context.Context) (adapterhealth.Result, error) {
				return adapterhealth.Result{
					Status:  adapterhealth.StatusDegraded,
					Code:    adapterhealth.CodeDegraded,
					Message: "fixture degraded",
				}, nil
			}),
		},
	})

	result := mgr.ReadinessPreflight()
	if result.Status != runtimeconfig.ReadinessStatusDegraded {
		t.Fatalf("status=%q, want degraded", result.Status)
	}
	assertReadinessFindingCode(t, result.Findings, runtimeconfig.ReadinessCodeAdapterDegraded)
	summary := result.Summary()
	if summary.AdapterHealthStatus != string(adapterhealth.StatusDegraded) ||
		summary.AdapterHealthProbeTotal != 1 ||
		summary.AdapterHealthDegradedTotal != 1 ||
		summary.AdapterHealthUnavailableTotal != 0 ||
		summary.AdapterHealthPrimaryCode != adapterhealth.CodeDegraded {
		t.Fatalf("adapter health visibility mismatch: %#v", summary)
	}
}

func TestAdapterConformanceHealthGovernanceMatrixStateTransitionDeterministic(t *testing.T) {
	run := func() []string {
		cfgPath := filepath.Join(t.TempDir(), "runtime-a46-state-transition.yaml")
		writeAdapterHealthGovernanceConfig(t, cfgPath, false)
		mgr, err := runtimeconfig.NewManager(runtimeconfig.ManagerOptions{FilePath: cfgPath, EnvPrefix: "BAYMAX_A46_CONFORMANCE"})
		if err != nil {
			t.Fatalf("new runtime manager: %v", err)
		}
		t.Cleanup(func() { _ = mgr.Close() })

		var calls int
		mgr.SetAdapterHealthTargets([]runtimeconfig.AdapterHealthTarget{
			{
				Name:     "required-adapter",
				Required: true,
				Probe: adapterhealth.ProbeFunc(func(context.Context) (adapterhealth.Result, error) {
					calls++
					if calls <= 2 {
						return adapterhealth.Result{
							Status:  adapterhealth.StatusUnavailable,
							Code:    adapterhealth.CodeProbeFailed,
							Message: "fixture unavailable",
						}, nil
					}
					return adapterhealth.Result{
						Status:  adapterhealth.StatusHealthy,
						Code:    adapterhealth.CodeHealthy,
						Message: "fixture recovered",
					}, nil
				}),
			},
		})

		out := make([]string, 0, 5)
		capture := func(res runtimeconfig.ReadinessResult) {
			summary := res.Summary()
			payload := struct {
				Status       runtimeconfig.ReadinessStatus `json:"status"`
				PrimaryCode  string                        `json:"primary_code"`
				CircuitState string                        `json:"circuit_state"`
				OpenTotal    int                           `json:"open_total"`
				RecoverTotal int                           `json:"recover_total"`
			}{
				Status:       res.Status,
				PrimaryCode:  summary.PrimaryCode,
				CircuitState: summary.AdapterHealthCircuitState,
				OpenTotal:    summary.AdapterHealthCircuitOpenTotal,
				RecoverTotal: summary.AdapterHealthCircuitRecoverTotal,
			}
			blob, _ := json.Marshal(payload)
			out = append(out, string(blob))
		}

		capture(mgr.ReadinessPreflight())
		time.Sleep(3 * time.Millisecond)
		capture(mgr.ReadinessPreflight())
		capture(mgr.ReadinessPreflight())
		time.Sleep(35 * time.Millisecond)
		capture(mgr.ReadinessPreflight())
		time.Sleep(3 * time.Millisecond)
		capture(mgr.ReadinessPreflight())
		return out
	}

	first := run()
	second := run()
	if len(first) != len(second) {
		t.Fatalf("state transition sequence length mismatch first=%d second=%d", len(first), len(second))
	}
	for i := range first {
		if first[i] != second[i] {
			t.Fatalf("state transition determinism drift at step=%d first=%s second=%s", i, first[i], second[i])
		}
	}
}

func TestAdapterConformanceHealthGovernanceMatrixHalfOpenRecovery(t *testing.T) {
	cfgPath := filepath.Join(t.TempDir(), "runtime-a46-half-open.yaml")
	writeAdapterHealthGovernanceConfig(t, cfgPath, false)
	mgr, err := runtimeconfig.NewManager(runtimeconfig.ManagerOptions{FilePath: cfgPath, EnvPrefix: "BAYMAX_A46_CONFORMANCE"})
	if err != nil {
		t.Fatalf("new runtime manager: %v", err)
	}
	defer func() { _ = mgr.Close() }()

	var calls int
	mgr.SetAdapterHealthTargets([]runtimeconfig.AdapterHealthTarget{
		{
			Name:     "optional-adapter",
			Required: false,
			Probe: adapterhealth.ProbeFunc(func(context.Context) (adapterhealth.Result, error) {
				calls++
				switch calls {
				case 1, 2:
					return adapterhealth.Result{
						Status:  adapterhealth.StatusUnavailable,
						Code:    adapterhealth.CodeProbeFailed,
						Message: "fixture unavailable",
					}, nil
				case 3:
					return adapterhealth.Result{
						Status:  adapterhealth.StatusDegraded,
						Code:    adapterhealth.CodeDegraded,
						Message: "fixture half-open degraded",
					}, nil
				default:
					return adapterhealth.Result{
						Status:  adapterhealth.StatusHealthy,
						Code:    adapterhealth.CodeHealthy,
						Message: "fixture recovered",
					}, nil
				}
			}),
		},
	})

	_ = mgr.ReadinessPreflight()
	time.Sleep(3 * time.Millisecond)
	_ = mgr.ReadinessPreflight()
	openWindow := mgr.ReadinessPreflight()
	assertReadinessFindingCode(t, openWindow.Findings, runtimeconfig.ReadinessCodeAdapterOptionalCircuitOpen)

	time.Sleep(35 * time.Millisecond)
	halfOpenDegraded := mgr.ReadinessPreflight()
	assertReadinessFindingCode(t, halfOpenDegraded.Findings, runtimeconfig.ReadinessCodeAdapterHalfOpenDegraded)
	summary := halfOpenDegraded.Summary()
	if summary.AdapterHealthCircuitState != string(adapterhealth.CircuitStateHalfOpen) {
		t.Fatalf("half-open state mismatch: %#v", summary)
	}
}

func TestAdapterConformanceHealthGovernanceTaxonomyDriftGuard(t *testing.T) {
	codes := []string{
		runtimeconfig.ReadinessCodeAdapterRequiredUnavailable,
		runtimeconfig.ReadinessCodeAdapterOptionalUnavailable,
		runtimeconfig.ReadinessCodeAdapterDegraded,
		runtimeconfig.ReadinessCodeAdapterRequiredCircuitOpen,
		runtimeconfig.ReadinessCodeAdapterOptionalCircuitOpen,
		runtimeconfig.ReadinessCodeAdapterHalfOpenDegraded,
		runtimeconfig.ReadinessCodeAdapterGovernanceRecovered,
	}
	for _, code := range codes {
		if !strings.HasPrefix(code, "adapter.health.") {
			t.Fatalf("taxonomy drift: code must stay in adapter.health.* namespace, got %q", code)
		}
		if err := ValidateReasonCode(code); err != nil {
			t.Fatalf("taxonomy drift: invalid reason code %q: %v", code, err)
		}
	}
}

func TestAdapterConformanceHealthGovernanceDiagnosticsReplayIdempotent(t *testing.T) {
	store := runtimediag.NewStore(16, 16, 8, 8, runtimediag.TimelineTrendConfig{Enabled: true, LastNRuns: 100, TimeWindow: 15 * time.Minute}, runtimediag.ContextStage2ExternalTrendConfig{Enabled: true, Window: 15 * time.Minute})
	rec := runtimediag.RunRecord{
		Time:                               time.Now().UTC(),
		RunID:                              "run-a46-governance-replay",
		Status:                             "success",
		AdapterHealthBackoffAppliedTotal:   4,
		AdapterHealthCircuitOpenTotal:      2,
		AdapterHealthCircuitHalfOpenTotal:  1,
		AdapterHealthCircuitRecoverTotal:   1,
		AdapterHealthCircuitState:          "half_open",
		AdapterHealthGovernancePrimaryCode: "adapter.health.circuit_half_open",
	}
	store.AddRun(rec)
	store.AddRun(rec)

	page, err := store.QueryRuns(runtimediag.UnifiedRunQueryRequest{RunID: rec.RunID})
	if err != nil {
		t.Fatalf("query governance replay failed: %v", err)
	}
	if len(page.Items) != 1 {
		t.Fatalf("replay idempotency mismatch: %#v", page.Items)
	}
	got := page.Items[0]
	if got.AdapterHealthBackoffAppliedTotal != 4 ||
		got.AdapterHealthCircuitOpenTotal != 2 ||
		got.AdapterHealthCircuitHalfOpenTotal != 1 ||
		got.AdapterHealthCircuitRecoverTotal != 1 ||
		got.AdapterHealthCircuitState != "half_open" ||
		got.AdapterHealthGovernancePrimaryCode != "adapter.health.circuit_half_open" {
		t.Fatalf("governance replay payload mismatch: %#v", got)
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

func mustActivationError(t *testing.T, _ adaptermanifest.ActivationResult, err error) *adaptermanifest.ContractError {
	t.Helper()
	if err == nil {
		t.Fatal("expected activation error")
	}
	return contractErr(t, err)
}

func containsReason(reasons []string, target string) bool {
	for _, reason := range reasons {
		if reason == target {
			return true
		}
	}
	return false
}

func assertReadinessFindingCode(t *testing.T, findings []runtimeconfig.ReadinessFinding, code string) {
	t.Helper()
	for i := range findings {
		if strings.TrimSpace(findings[i].Code) == strings.TrimSpace(code) {
			return
		}
	}
	t.Fatalf("expected readiness code=%q, findings=%#v", code, findings)
}

func readinessFingerprint(result runtimeconfig.ReadinessResult) string {
	payload := struct {
		Status        runtimeconfig.ReadinessStatus           `json:"status"`
		Findings      []runtimeconfig.ReadinessFinding        `json:"findings"`
		AdapterHealth []runtimeconfig.AdapterHealthEvaluation `json:"adapter_health"`
	}{
		Status:        result.Status,
		Findings:      result.Findings,
		AdapterHealth: result.AdapterHealth,
	}
	blob, _ := json.Marshal(payload)
	return string(blob)
}

func writeAdapterHealthGovernanceConfig(t *testing.T, path string, readinessStrict bool) {
	t.Helper()
	content := strings.Join([]string{
		"runtime:",
		"  readiness:",
		"    enabled: true",
		"    strict: false",
		"    remote_probe_enabled: false",
		"adapter:",
		"  health:",
		"    enabled: true",
		"    strict: false",
		"    probe_timeout: 500ms",
		"    cache_ttl: 1ms",
		"    backoff:",
		"      enabled: true",
		"      initial: 2ms",
		"      max: 10ms",
		"      multiplier: 2",
		"      jitter_ratio: 0",
		"    circuit:",
		"      enabled: true",
		"      failure_threshold: 2",
		"      open_duration: 30ms",
		"      half_open_max_probe: 1",
		"      half_open_success_threshold: 2",
		"",
	}, "\n")
	if readinessStrict {
		content = strings.Replace(content, "strict: false", "strict: true", 1)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
}

func writeAdapterHealthConfig(t *testing.T, path string, readinessStrict bool) {
	t.Helper()
	content := "runtime:\n  readiness:\n    enabled: true\n    strict: false\n    remote_probe_enabled: false\nadapter:\n  health:\n    enabled: true\n    strict: false\n    probe_timeout: 500ms\n    cache_ttl: 30s\n"
	if readinessStrict {
		content = strings.Replace(content, "strict: false", "strict: true", 1)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
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
