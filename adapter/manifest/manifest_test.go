package manifest

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	adaptercap "github.com/FelixSeptem/baymax/adapter/capability"
	adapterprofile "github.com/FelixSeptem/baymax/adapter/profile"
)

func TestParseAndValidateManifestSuccess(t *testing.T) {
	raw := []byte(`{
  "type": "model",
  "name": "demo-model",
  "version": "0.1.0",
  "contract_profile_version": "v1alpha1",
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
  "contract_profile_version": "v1alpha1",
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
  "contract_profile_version": "v1alpha1",
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
		Type:                   "model",
		Name:                   "demo-model",
		Version:                "0.1.0",
		ContractProfileVersion: adapterprofile.ProfileV1Alpha1,
		BaymaxCompat:           ">=0.26.0-rc.1 <0.27.0",
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
	if outcome.ContractProfileVersion != adapterprofile.ProfileV1Alpha1 {
		t.Fatalf("unexpected contract profile version: %#v", outcome.ContractProfileVersion)
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
		Type:                   "tool",
		Name:                   "demo-tool",
		Version:                "0.1.0",
		ContractProfileVersion: adapterprofile.ProfileV1Alpha1,
		BaymaxCompat:           ">=0.26.0-rc.1 <0.27.0",
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

func TestParseManifestRejectsUnknownContractProfileVersion(t *testing.T) {
	raw := []byte(`{
  "type": "tool",
  "name": "demo-tool",
  "version": "0.1.0",
  "contract_profile_version": "v9alpha9",
  "baymax_compat": ">=0.26.0-rc.1 <0.27.0",
  "capabilities": {
    "required": ["tool.invoke.required_input"],
    "optional": []
  },
  "conformance_profile": "tool-invoke-fail-fast"
}`)
	_, err := Parse(raw)
	if err == nil {
		t.Fatal("expected unknown contract profile version")
	}
	ce := contractErr(t, err)
	if ce.Code != CodeUnknownContractProfile || ce.Field != "contract_profile_version" {
		t.Fatalf("unexpected error classification: %#v", ce)
	}
}

func TestActivateManifestRejectsProfileOutOfWindow(t *testing.T) {
	manifest := Manifest{
		Type:                   "model",
		Name:                   "demo-model",
		Version:                "0.1.0",
		ContractProfileVersion: adapterprofile.ProfileV1Alpha1,
		BaymaxCompat:           ">=0.26.0-rc.1 <0.27.0",
		Capabilities: Capabilities{
			Required: []string{"model.generate", "model.stream"},
			Optional: []string{},
		},
		ConformanceProfile: "model-run-stream-downgrade",
	}
	window, err := adapterprofile.NewWindow(adapterprofile.ProfileV1Alpha0, true)
	if err != nil {
		t.Fatalf("new profile window: %v", err)
	}
	_, err = ActivateWithRequestAndProfileWindow(manifest, "0.26.0-rc.3", []string{"model.generate", "model.stream"}, CapabilityRequest{
		Required: []string{"model.generate", "model.stream"},
	}, window)
	if err == nil {
		t.Fatal("expected out-of-window contract profile failure")
	}
	ce := contractErr(t, err)
	if ce.Code != CodeContractProfileOutOfWindow || ce.Field != "contract_profile_version" {
		t.Fatalf("unexpected error classification: %#v", ce)
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
  "contract_profile_version": "v1alpha1",
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

func TestParseSandboxManifestCompleteMetadata(t *testing.T) {
	raw := []byte(`{
  "type": "tool",
  "name": "sandbox-tool",
  "version": "0.1.0",
  "contract_profile_version": "v1alpha1",
  "baymax_compat": ">=0.26.0-rc.1 <0.27.0",
  "capabilities": {
    "required": ["tool.invoke.required_input"],
    "optional": []
  },
  "conformance_profile": "tool-invoke-fail-fast",
  "sandbox_backend": "linux_nsjail",
  "sandbox_profile_id": "linux_nsjail",
  "host_os": "linux",
  "host_arch": "amd64",
  "session_modes_supported": ["per_call", "per_session"]
}`)
	got, err := Parse(raw)
	if err != nil {
		t.Fatalf("parse sandbox manifest: %v", err)
	}
	if got.SandboxBackend != "linux_nsjail" ||
		got.SandboxProfileID != "linux_nsjail" ||
		got.HostOS != "linux" ||
		got.HostArch != "amd64" {
		t.Fatalf("unexpected sandbox metadata normalization: %#v", got)
	}
}

func TestParseSandboxManifestMissingBackendFailFast(t *testing.T) {
	raw := []byte(`{
  "type": "tool",
  "name": "sandbox-tool",
  "version": "0.1.0",
  "contract_profile_version": "v1alpha1",
  "baymax_compat": ">=0.26.0-rc.1 <0.27.0",
  "capabilities": {
    "required": ["tool.invoke.required_input"],
    "optional": []
  },
  "conformance_profile": "tool-invoke-fail-fast",
  "sandbox_profile_id": "linux_nsjail",
  "host_os": "linux",
  "host_arch": "amd64",
  "session_modes_supported": ["per_call"]
}`)
	_, err := Parse(raw)
	if err == nil {
		t.Fatal("expected missing sandbox backend field")
	}
	ce := contractErr(t, err)
	if ce.Code != CodeMissingField || ce.Field != "sandbox_backend" {
		t.Fatalf("unexpected error: %#v", ce)
	}
}

func TestParseSandboxManifestUnknownProfileFailFast(t *testing.T) {
	raw := []byte(`{
  "type": "tool",
  "name": "sandbox-tool",
  "version": "0.1.0",
  "contract_profile_version": "v1alpha1",
  "baymax_compat": ">=0.26.0-rc.1 <0.27.0",
  "capabilities": {
    "required": ["tool.invoke.required_input"],
    "optional": []
  },
  "conformance_profile": "tool-invoke-fail-fast",
  "sandbox_backend": "linux_nsjail",
  "sandbox_profile_id": "missing-profile",
  "host_os": "linux",
  "host_arch": "amd64",
  "session_modes_supported": ["per_call"]
}`)
	_, err := Parse(raw)
	if err == nil {
		t.Fatal("expected unknown sandbox profile id failure")
	}
	ce := contractErr(t, err)
	if ce.Code != CodeSandboxProfileUnknown || ce.Field != "sandbox_profile_id" {
		t.Fatalf("unexpected error: %#v", ce)
	}
}

func TestActivateSandboxManifestHostMismatchFailFast(t *testing.T) {
	manifest := Manifest{
		Type:                   "tool",
		Name:                   "sandbox-tool",
		Version:                "0.1.0",
		ContractProfileVersion: adapterprofile.ProfileV1Alpha1,
		BaymaxCompat:           ">=0.26.0-rc.1 <0.27.0",
		Capabilities: Capabilities{
			Required: []string{"tool.invoke.required_input"},
			Optional: []string{},
		},
		ConformanceProfile:    "tool-invoke-fail-fast",
		SandboxBackend:        SandboxBackendLinuxNSJail,
		SandboxProfileID:      SandboxBackendLinuxNSJail,
		HostOS:                "linux",
		HostArch:              "amd64",
		SessionModesSupported: []string{SandboxSessionModePerCall, SandboxSessionModePerSession},
	}
	_, err := ActivateWithRequestAndProfileWindowWithContext(
		manifest,
		"0.26.0-rc.2",
		[]string{"tool.invoke.required_input"},
		CapabilityRequest{Required: []string{"tool.invoke.required_input"}},
		adapterprofile.DefaultWindow(),
		ActivationContext{HostOS: "windows", HostArch: "amd64", RequestedSession: SandboxSessionModePerCall},
	)
	if err == nil {
		t.Fatal("expected host mismatch fail-fast")
	}
	ce := contractErr(t, err)
	if ce.Code != CodeSandboxHostMismatch || ce.Field != "host_os" {
		t.Fatalf("unexpected error: %#v", ce)
	}
}

func TestActivateSandboxManifestSessionModeUnsupportedFailFast(t *testing.T) {
	manifest := Manifest{
		Type:                   "tool",
		Name:                   "sandbox-tool",
		Version:                "0.1.0",
		ContractProfileVersion: adapterprofile.ProfileV1Alpha1,
		BaymaxCompat:           ">=0.26.0-rc.1 <0.27.0",
		Capabilities: Capabilities{
			Required: []string{"tool.invoke.required_input"},
			Optional: []string{},
		},
		ConformanceProfile:    "tool-invoke-fail-fast",
		SandboxBackend:        SandboxBackendLinuxNSJail,
		SandboxProfileID:      SandboxBackendLinuxNSJail,
		HostOS:                "linux",
		HostArch:              "amd64",
		SessionModesSupported: []string{SandboxSessionModePerCall},
	}
	_, err := ActivateWithRequestAndProfileWindowWithContext(
		manifest,
		"0.26.0-rc.2",
		[]string{"tool.invoke.required_input"},
		CapabilityRequest{Required: []string{"tool.invoke.required_input"}},
		adapterprofile.DefaultWindow(),
		ActivationContext{HostOS: "linux", HostArch: "amd64", RequestedSession: SandboxSessionModePerSession},
	)
	if err == nil {
		t.Fatal("expected unsupported session mode fail-fast")
	}
	ce := contractErr(t, err)
	if ce.Code != CodeSandboxSessionUnsupported || ce.Field != "session_modes_supported" {
		t.Fatalf("unexpected error: %#v", ce)
	}
}

func TestActivateSandboxManifestMissingProfileFailFast(t *testing.T) {
	manifest := Manifest{
		Type:                   "tool",
		Name:                   "sandbox-tool",
		Version:                "0.1.0",
		ContractProfileVersion: adapterprofile.ProfileV1Alpha1,
		BaymaxCompat:           ">=0.26.0-rc.1 <0.27.0",
		Capabilities: Capabilities{
			Required: []string{"tool.invoke.required_input"},
			Optional: []string{},
		},
		ConformanceProfile:    "tool-invoke-fail-fast",
		SandboxBackend:        SandboxBackendLinuxNSJail,
		SandboxProfileID:      "missing-profile",
		HostOS:                "linux",
		HostArch:              "amd64",
		SessionModesSupported: []string{SandboxSessionModePerCall},
	}
	_, err := Activate(manifest, "0.26.0-rc.2", []string{"tool.invoke.required_input"})
	if err == nil {
		t.Fatal("expected missing profile fail-fast")
	}
	ce := contractErr(t, err)
	if ce.Code != CodeSandboxProfileUnknown || ce.Field != "sandbox_profile_id" {
		t.Fatalf("unexpected error: %#v", ce)
	}
}

func TestParseMemoryManifestCompleteMetadata(t *testing.T) {
	raw := []byte(`{
  "type": "tool",
  "name": "memory-tool",
  "version": "0.1.0",
  "contract_profile_version": "v1alpha1",
  "baymax_compat": ">=0.26.0-rc.1 <0.27.0",
  "capabilities": {
    "required": ["tool.invoke.required_input"],
    "optional": []
  },
  "conformance_profile": "tool-invoke-fail-fast",
  "memory": {
    "provider": "mem0",
    "profile": "mem0",
    "contract_version": "memory.v1",
    "operations": {
      "required": ["query", "upsert", "delete"],
      "optional": ["metadata_filter"]
    },
    "fallback": {
      "supported": true
    }
  }
}`)
	manifest, err := Parse(raw)
	if err != nil {
		t.Fatalf("parse memory manifest: %v", err)
	}
	if manifest.Memory == nil {
		t.Fatal("memory metadata should be present")
	}
	if manifest.Memory.Provider != "mem0" || manifest.Memory.Profile != "mem0" || manifest.Memory.ContractVersion != "memory.v1" {
		t.Fatalf("unexpected memory metadata normalization: %#v", manifest.Memory)
	}
}

func TestParseMemoryManifestMissingContractVersionFailFast(t *testing.T) {
	raw := []byte(`{
  "type": "tool",
  "name": "memory-tool",
  "version": "0.1.0",
  "contract_profile_version": "v1alpha1",
  "baymax_compat": ">=0.26.0-rc.1 <0.27.0",
  "capabilities": {
    "required": ["tool.invoke.required_input"],
    "optional": []
  },
  "conformance_profile": "tool-invoke-fail-fast",
  "memory": {
    "provider": "mem0",
    "profile": "mem0",
    "operations": {
      "required": ["query", "upsert", "delete"],
      "optional": []
    },
    "fallback": {
      "supported": true
    }
  }
}`)
	_, err := Parse(raw)
	if err == nil {
		t.Fatal("expected missing memory.contract_version failure")
	}
	ce := contractErr(t, err)
	if ce.Code != CodeMissingField || ce.Field != "memory.contract_version" {
		t.Fatalf("unexpected error: %#v", ce)
	}
}

func TestActivateMemoryManifestProfileMismatchFailFast(t *testing.T) {
	manifest := Manifest{
		Type:                   "tool",
		Name:                   "memory-tool",
		Version:                "0.1.0",
		ContractProfileVersion: adapterprofile.ProfileV1Alpha1,
		BaymaxCompat:           ">=0.26.0-rc.1 <0.27.0",
		Capabilities: Capabilities{
			Required: []string{"tool.invoke.required_input"},
			Optional: []string{},
		},
		ConformanceProfile: "tool-invoke-fail-fast",
		Memory: &Memory{
			Provider:        "mem0",
			Profile:         "mem0",
			ContractVersion: "memory.v1",
			Operations: MemoryOperations{
				Required: []string{"query", "upsert", "delete"},
				Optional: []string{"metadata_filter"},
			},
			Fallback: MemoryFallback{Supported: boolPtr(true)},
		},
	}
	_, err := ActivateWithRequestAndProfileWindowWithContext(
		manifest,
		"0.26.0-rc.2",
		[]string{"tool.invoke.required_input"},
		CapabilityRequest{Required: []string{"tool.invoke.required_input"}},
		adapterprofile.DefaultWindow(),
		ActivationContext{
			MemoryMode:       "external_spi",
			MemoryProvider:   "mem0",
			MemoryProfile:    "zep",
			MemoryContract:   "memory.v1",
			MemoryOperations: []string{"query", "upsert", "delete"},
		},
	)
	if err == nil {
		t.Fatal("expected memory profile mismatch fail-fast")
	}
	ce := contractErr(t, err)
	if ce.Code != CodeMemoryProfileMismatch || ce.Field != "memory.profile" {
		t.Fatalf("unexpected error: %#v", ce)
	}
}

func TestActivateMemoryManifestRequiredOperationMissingFailFast(t *testing.T) {
	manifest := Manifest{
		Type:                   "tool",
		Name:                   "memory-tool",
		Version:                "0.1.0",
		ContractProfileVersion: adapterprofile.ProfileV1Alpha1,
		BaymaxCompat:           ">=0.26.0-rc.1 <0.27.0",
		Capabilities: Capabilities{
			Required: []string{"tool.invoke.required_input"},
			Optional: []string{},
		},
		ConformanceProfile: "tool-invoke-fail-fast",
		Memory: &Memory{
			Provider:        "mem0",
			Profile:         "mem0",
			ContractVersion: "memory.v1",
			Operations: MemoryOperations{
				Required: []string{"query", "upsert", "delete"},
				Optional: []string{"metadata_filter"},
			},
			Fallback: MemoryFallback{Supported: boolPtr(true)},
		},
	}
	_, err := ActivateWithRequestAndProfileWindowWithContext(
		manifest,
		"0.26.0-rc.2",
		[]string{"tool.invoke.required_input"},
		CapabilityRequest{Required: []string{"tool.invoke.required_input"}},
		adapterprofile.DefaultWindow(),
		ActivationContext{
			MemoryMode:       "external_spi",
			MemoryProvider:   "mem0",
			MemoryProfile:    "mem0",
			MemoryContract:   "memory.v1",
			MemoryOperations: []string{"query", "upsert"},
		},
	)
	if err == nil {
		t.Fatal("expected memory required operation fail-fast")
	}
	ce := contractErr(t, err)
	if ce.Code != CodeMemoryRequiredOpMissing || ce.Field != "memory.operations.required" {
		t.Fatalf("unexpected error: %#v", ce)
	}
}

func TestActivateMemoryManifestOptionalOperationDowngrade(t *testing.T) {
	manifest := Manifest{
		Type:                   "tool",
		Name:                   "memory-tool",
		Version:                "0.1.0",
		ContractProfileVersion: adapterprofile.ProfileV1Alpha1,
		BaymaxCompat:           ">=0.26.0-rc.1 <0.27.0",
		Capabilities: Capabilities{
			Required: []string{"tool.invoke.required_input"},
			Optional: []string{},
		},
		ConformanceProfile: "tool-invoke-fail-fast",
		Memory: &Memory{
			Provider:        "mem0",
			Profile:         "mem0",
			ContractVersion: "memory.v1",
			Operations: MemoryOperations{
				Required: []string{"query", "upsert", "delete"},
				Optional: []string{"metadata_filter"},
			},
			Fallback: MemoryFallback{Supported: boolPtr(true)},
		},
	}
	out, err := ActivateWithRequestAndProfileWindowWithContext(
		manifest,
		"0.26.0-rc.2",
		[]string{"tool.invoke.required_input"},
		CapabilityRequest{Required: []string{"tool.invoke.required_input"}},
		adapterprofile.DefaultWindow(),
		ActivationContext{
			MemoryMode:       "external_spi",
			MemoryProvider:   "mem0",
			MemoryProfile:    "mem0",
			MemoryContract:   "memory.v1",
			MemoryOperations: []string{"query", "upsert", "delete"},
		},
	)
	if err != nil {
		t.Fatalf("memory optional operation downgrade should not block activation: %v", err)
	}
	if len(out.OptionalDowngrades) != 1 {
		t.Fatalf("expected one optional downgrade, got %#v", out.OptionalDowngrades)
	}
	if out.OptionalDowngrades[0].Capability != "memory.metadata_filter" {
		t.Fatalf("unexpected optional downgrade: %#v", out.OptionalDowngrades[0])
	}
	if out.OptionalDowngrades[0].ReasonCode != "adapter.manifest.memory.operation.optional_missing.metadata_filter" {
		t.Fatalf("unexpected downgrade reason: %#v", out.OptionalDowngrades[0])
	}
}

func TestActivateMemoryManifestMissingActivationContextFailFast(t *testing.T) {
	manifest := Manifest{
		Type:                   "tool",
		Name:                   "memory-tool",
		Version:                "0.1.0",
		ContractProfileVersion: adapterprofile.ProfileV1Alpha1,
		BaymaxCompat:           ">=0.26.0-rc.1 <0.27.0",
		Capabilities: Capabilities{
			Required: []string{"tool.invoke.required_input"},
			Optional: []string{},
		},
		ConformanceProfile: "tool-invoke-fail-fast",
		Memory: &Memory{
			Provider:        "mem0",
			Profile:         "mem0",
			ContractVersion: "memory.v1",
			Operations: MemoryOperations{
				Required: []string{"query", "upsert", "delete"},
				Optional: []string{},
			},
			Fallback: MemoryFallback{Supported: boolPtr(true)},
		},
	}
	_, err := Activate(manifest, "0.26.0-rc.2", []string{"tool.invoke.required_input"})
	if err == nil {
		t.Fatal("expected missing memory activation context fail-fast")
	}
	ce := contractErr(t, err)
	if ce.Code != CodeMemoryContextMissing || ce.Field != "memory_mode" {
		t.Fatalf("unexpected error: %#v", ce)
	}
}

func boolPtr(v bool) *bool {
	return &v
}

func contractErr(t *testing.T, err error) *ContractError {
	t.Helper()
	ce := &ContractError{}
	if !errors.As(err, &ce) {
		t.Fatalf("expected ContractError, got %T (%v)", err, err)
	}
	return ce
}
