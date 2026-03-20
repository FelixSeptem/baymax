package manifest

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	adaptercap "github.com/FelixSeptem/baymax/adapter/capability"
)

func TestParseAndValidateManifestSuccess(t *testing.T) {
	raw := []byte(`{
  "type": "model",
  "name": "demo-model",
  "version": "0.1.0",
  "baymax_compat": ">=0.26.0-rc.1 <0.27.0",
  "capabilities": {
    "required": ["model.generate", "model.stream"],
    "optional": ["model.token_count"]
  },
  "conformance_profile": "model-run-stream-downgrade"
}`)
	manifest, err := Parse(raw)
	if err != nil {
		t.Fatalf("parse manifest: %v", err)
	}
	if manifest.Type != "model" || manifest.Name != "demo-model" {
		t.Fatalf("unexpected normalized manifest: %#v", manifest)
	}
}

func TestParseManifestDetectsMissingFieldDeterministically(t *testing.T) {
	raw := []byte(`{
  "type": "mcp",
  "name": "demo-mcp",
  "version": "0.1.0",
  "capabilities": {
    "required": ["mcp.invoke"],
    "optional": []
  },
  "conformance_profile": "mcp-normalization-fail-fast"
}`)
	_, err1 := Parse(raw)
	_, err2 := Parse(raw)
	if err1 == nil || err2 == nil {
		t.Fatal("expected missing field validation error")
	}
	e1 := contractErr(t, err1)
	e2 := contractErr(t, err2)
	if e1.Code != CodeMissingField || e1.Field != "baymax_compat" {
		t.Fatalf("unexpected error classification: %#v", e1)
	}
	if !reflect.DeepEqual(e1, e2) {
		t.Fatalf("non-deterministic classification: %#v vs %#v", e1, e2)
	}
}

func TestParseManifestDetectsInvalidCompatExpression(t *testing.T) {
	raw := []byte(`{
  "type": "tool",
  "name": "demo-tool",
  "version": "0.1.0",
  "baymax_compat": ">=0.26.x",
  "capabilities": {
    "required": ["tool.invoke.required_input"],
    "optional": []
  },
  "conformance_profile": "tool-invoke-fail-fast"
}`)
	_, err := Parse(raw)
	if err == nil {
		t.Fatal("expected invalid compat expression")
	}
	ce := contractErr(t, err)
	if ce.Code != CodeInvalidCompatExpression || ce.Field != "baymax_compat" {
		t.Fatalf("unexpected error classification: %#v", ce)
	}
}

func TestEvaluateSemverRangeSupportsPreReleaseRC(t *testing.T) {
	ok, err := evaluateSemverRange(">=0.26.0-rc.1 <0.27.0", "0.26.0-rc.2")
	if err != nil {
		t.Fatalf("evaluate range: %v", err)
	}
	if !ok {
		t.Fatal("expected rc version to satisfy range")
	}
	ok, err = evaluateSemverRange(">=0.26.0-rc.1 <0.27.0", "0.25.9")
	if err != nil {
		t.Fatalf("evaluate range: %v", err)
	}
	if ok {
		t.Fatal("expected out-of-range version to fail")
	}
	ok, err = evaluateSemverRange(">0.26.0-rc.2", "0.26.0")
	if err != nil {
		t.Fatalf("evaluate range: %v", err)
	}
	if !ok {
		t.Fatal("stable release should be greater than rc")
	}
}

func TestActivateManifestCompatibilityAndCapabilities(t *testing.T) {
	manifest := Manifest{
		Type:         "model",
		Name:         "demo-model",
		Version:      "0.1.0",
		BaymaxCompat: ">=0.26.0-rc.1 <0.27.0",
		Capabilities: Capabilities{
			Required: []string{"model.generate", "model.stream"},
			Optional: []string{"model.token_count", "model.safety_filter"},
		},
		Negotiation: Negotiation{
			DefaultStrategy:      adaptercap.StrategyFailFast,
			AllowRequestOverride: true,
		},
		ConformanceProfile: "model-run-stream-downgrade",
	}

	outcome, err := ActivateWithRequest(manifest, "0.26.0-rc.3", []string{"model.generate", "model.stream", "model.safety_filter"}, CapabilityRequest{
		Required:         []string{"model.generate", "model.stream"},
		Optional:         []string{"model.token_count"},
		StrategyOverride: adaptercap.StrategyBestEffort,
	})
	if err != nil {
		t.Fatalf("activate success path: %v", err)
	}
	if len(outcome.OptionalDowngrades) != 1 {
		t.Fatalf("expected one optional downgrade, got %#v", outcome.OptionalDowngrades)
	}
	if outcome.OptionalDowngrades[0].Capability != "model.token_count" {
		t.Fatalf("unexpected downgrade capability: %#v", outcome.OptionalDowngrades[0])
	}
	if !outcome.StrategyOverride || outcome.StrategyApplied != adaptercap.StrategyBestEffort {
		t.Fatalf("unexpected negotiation strategy info: %#v", outcome)
	}

	_, err = Activate(manifest, "0.27.0", []string{"model.generate", "model.stream"})
	if err == nil {
		t.Fatal("expected compatibility mismatch")
	}
	compatErr := contractErr(t, err)
	if compatErr.Code != CodeCompatibilityMismatch {
		t.Fatalf("unexpected compatibility mismatch error: %#v", compatErr)
	}

	_, err = ActivateWithRequest(manifest, "0.26.0-rc.3", []string{"model.generate", "model.stream"}, CapabilityRequest{
		Required: []string{"model.generate", "model.stream"},
		Optional: []string{"model.token_count"},
	})
	if err == nil {
		t.Fatal("expected fail_fast missing optional request to reject")
	}

	_, err = Activate(manifest, "0.26.0-rc.3", []string{"model.generate"})
	if err == nil {
		t.Fatal("expected required capability failure")
	}
	requiredErr := contractErr(t, err)
	if requiredErr.Code != CodeRequiredCapabilityMissing || requiredErr.Field != "capabilities.required" {
		t.Fatalf("unexpected required capability error: %#v", requiredErr)
	}
}

func TestValidateNegotiationConfigRejectsInvalidDefaultStrategy(t *testing.T) {
	err := Validate(Manifest{
		Type:         "tool",
		Name:         "demo-tool",
		Version:      "0.1.0",
		BaymaxCompat: ">=0.26.0-rc.1 <0.27.0",
		Capabilities: Capabilities{
			Required: []string{"tool.invoke.required_input"},
			Optional: []string{},
		},
		Negotiation: Negotiation{
			DefaultStrategy: "random",
		},
		ConformanceProfile: "tool-invoke-fail-fast",
	})
	if err == nil {
		t.Fatal("expected invalid negotiation config")
	}
	ce := contractErr(t, err)
	if ce.Code != CodeInvalidNegotiationConfig || ce.Field != "negotiation.default_strategy" {
		t.Fatalf("unexpected error: %#v", ce)
	}
}

func TestLoadFileMissingManifestFailFast(t *testing.T) {
	_, err := LoadFile(filepath.Join(t.TempDir(), "adapter-manifest.json"))
	if err == nil {
		t.Fatal("expected missing file error")
	}
	ce := contractErr(t, err)
	if ce.Code != CodeMissingFile {
		t.Fatalf("unexpected error classification: %#v", ce)
	}
}

func TestLoadFileParsesManifest(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "adapter-manifest.json")
	if err := os.WriteFile(path, []byte(`{
  "type": "tool",
  "name": "demo-tool",
  "version": "0.1.0",
  "baymax_compat": ">=0.26.0-rc.1 <0.27.0",
  "capabilities": {
    "required": ["tool.invoke.required_input"],
    "optional": []
  },
  "conformance_profile": "tool-invoke-fail-fast"
}`), 0o600); err != nil {
		t.Fatalf("write manifest file: %v", err)
	}
	manifest, err := LoadFile(path)
	if err != nil {
		t.Fatalf("load manifest: %v", err)
	}
	if manifest.Type != "tool" || manifest.Name != "demo-tool" {
		t.Fatalf("unexpected loaded manifest: %#v", manifest)
	}
}

func contractErr(t *testing.T, err error) *ContractError {
	t.Helper()
	ce := &ContractError{}
	if !errors.As(err, &ce) {
		t.Fatalf("expected ContractError, got %T (%v)", err, err)
	}
	return ce
}
