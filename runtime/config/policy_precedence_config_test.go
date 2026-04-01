package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRuntimePolicyConfigDefaults(t *testing.T) {
	cfg := DefaultConfig()
	got := cfg.Runtime.Policy
	if got.Precedence.Version != RuntimePolicyPrecedenceVersionPolicyStackV1 {
		t.Fatalf(
			"runtime.policy.precedence.version = %q, want %q",
			got.Precedence.Version,
			RuntimePolicyPrecedenceVersionPolicyStackV1,
		)
	}
	if got.TieBreaker.Mode != RuntimePolicyTieBreakerModeLexicalCodeThenSourceOrder {
		t.Fatalf(
			"runtime.policy.tie_breaker.mode = %q, want %q",
			got.TieBreaker.Mode,
			RuntimePolicyTieBreakerModeLexicalCodeThenSourceOrder,
		)
	}
	if !got.Explainability.Enabled {
		t.Fatal("runtime.policy.explainability.enabled = false, want true")
	}

	wantMatrix := DefaultRuntimePolicyPrecedenceMatrix()
	if len(got.Precedence.Matrix) != len(wantMatrix) {
		t.Fatalf("runtime.policy.precedence.matrix len = %d, want %d", len(got.Precedence.Matrix), len(wantMatrix))
	}
	for stage, wantRank := range wantMatrix {
		if got.Precedence.Matrix[stage] != wantRank {
			t.Fatalf("runtime.policy.precedence.matrix.%s = %d, want %d", stage, got.Precedence.Matrix[stage], wantRank)
		}
	}
	wantSourceOrder := RuntimePolicyCanonicalStages()
	if len(got.TieBreaker.SourceOrder) != len(wantSourceOrder) {
		t.Fatalf(
			"runtime.policy.tie_breaker.source_order len = %d, want %d",
			len(got.TieBreaker.SourceOrder),
			len(wantSourceOrder),
		)
	}
	for i := range wantSourceOrder {
		if got.TieBreaker.SourceOrder[i] != wantSourceOrder[i] {
			t.Fatalf(
				"runtime.policy.tie_breaker.source_order[%d] = %q, want %q",
				i,
				got.TieBreaker.SourceOrder[i],
				wantSourceOrder[i],
			)
		}
	}
}

func TestRuntimePolicyConfigEnvOverridePrecedence(t *testing.T) {
	t.Setenv("BAYMAX_RUNTIME_POLICY_PRECEDENCE_MATRIX_ACTION_GATE", "2")
	t.Setenv("BAYMAX_RUNTIME_POLICY_PRECEDENCE_MATRIX_SECURITY_S2", "1")
	t.Setenv("BAYMAX_RUNTIME_POLICY_TIE_BREAKER_SOURCE_ORDER", "security_s2,action_gate,sandbox_action,sandbox_egress,adapter_allowlist,readiness_admission")
	t.Setenv("BAYMAX_RUNTIME_POLICY_EXPLAINABILITY_ENABLED", "false")

	file := filepath.Join(t.TempDir(), "runtime.yaml")
	content := `
runtime:
  policy:
    precedence:
      version: policy_stack.v1
      matrix:
        action_gate: 1
        security_s2: 2
        sandbox_action: 3
        sandbox_egress: 4
        adapter_allowlist: 5
        readiness_admission: 6
    tie_breaker:
      mode: lexical_code_then_source_order
      source_order: [action_gate, security_s2, sandbox_action, sandbox_egress, adapter_allowlist, readiness_admission]
    explainability:
      enabled: true
`
	if err := os.WriteFile(file, []byte(strings.TrimSpace(content)), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	cfg, err := Load(LoadOptions{FilePath: file, EnvPrefix: "BAYMAX"})
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if cfg.Runtime.Policy.Precedence.Matrix[RuntimePolicyStageActionGate] != 2 {
		t.Fatalf("runtime.policy.precedence.matrix.action_gate = %d, want 2", cfg.Runtime.Policy.Precedence.Matrix[RuntimePolicyStageActionGate])
	}
	if cfg.Runtime.Policy.Precedence.Matrix[RuntimePolicyStageSecurityS2] != 1 {
		t.Fatalf("runtime.policy.precedence.matrix.security_s2 = %d, want 1", cfg.Runtime.Policy.Precedence.Matrix[RuntimePolicyStageSecurityS2])
	}
	if cfg.Runtime.Policy.TieBreaker.SourceOrder[0] != RuntimePolicyStageSecurityS2 ||
		cfg.Runtime.Policy.TieBreaker.SourceOrder[1] != RuntimePolicyStageActionGate {
		t.Fatalf("runtime.policy.tie_breaker.source_order env override mismatch: %#v", cfg.Runtime.Policy.TieBreaker.SourceOrder)
	}
	if cfg.Runtime.Policy.Explainability.Enabled {
		t.Fatal("runtime.policy.explainability.enabled = true, want false from env")
	}
}

func TestRuntimePolicyConfigInvalidBoolFailsFast(t *testing.T) {
	t.Setenv("BAYMAX_RUNTIME_POLICY_EXPLAINABILITY_ENABLED", "not-a-bool")
	_, err := Load(LoadOptions{EnvPrefix: "BAYMAX"})
	if err == nil {
		t.Fatal("expected runtime.policy.explainability.enabled invalid bool error")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "runtime.policy.explainability.enabled") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRuntimePolicyConfigMalformedMatrixEntryFailsFast(t *testing.T) {
	file := filepath.Join(t.TempDir(), "runtime.yaml")
	content := `
runtime:
  policy:
    precedence:
      version: policy_stack.v1
      matrix:
        action_gate: abc
        security_s2: 2
        sandbox_action: 3
        sandbox_egress: 4
        adapter_allowlist: 5
        readiness_admission: 6
    tie_breaker:
      mode: lexical_code_then_source_order
      source_order: [action_gate, security_s2, sandbox_action, sandbox_egress, adapter_allowlist, readiness_admission]
    explainability:
      enabled: true
`
	if err := os.WriteFile(file, []byte(strings.TrimSpace(content)), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	_, err := Load(LoadOptions{FilePath: file, EnvPrefix: "BAYMAX"})
	if err == nil {
		t.Fatal("expected malformed runtime.policy.precedence.matrix.action_gate to fail-fast")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "runtime.policy.precedence.matrix.action_gate") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRuntimePolicyConfigValidationRejectsInvalidValues(t *testing.T) {
	tests := []struct {
		name string
		cfg  RuntimePolicyConfig
	}{
		{
			name: "invalid-version",
			cfg: RuntimePolicyConfig{
				Precedence: RuntimePolicyPrecedenceConfig{
					Version: "a58.v1",
					Matrix:  DefaultRuntimePolicyPrecedenceMatrix(),
				},
				TieBreaker: RuntimePolicyTieBreakerConfig{
					Mode:        RuntimePolicyTieBreakerModeLexicalCodeThenSourceOrder,
					SourceOrder: RuntimePolicyCanonicalStages(),
				},
				Explainability: RuntimePolicyExplainabilityConfig{Enabled: true},
			},
		},
		{
			name: "invalid-stage",
			cfg: func() RuntimePolicyConfig {
				cfg := DefaultConfig().Runtime.Policy
				cfg.Precedence.Matrix["unknown_stage"] = 99
				return cfg
			}(),
		},
		{
			name: "duplicate-rank-conflict",
			cfg: func() RuntimePolicyConfig {
				cfg := DefaultConfig().Runtime.Policy
				cfg.Precedence.Matrix[RuntimePolicyStageSecurityS2] = cfg.Precedence.Matrix[RuntimePolicyStageActionGate]
				return cfg
			}(),
		},
		{
			name: "missing-canonical-stage",
			cfg: func() RuntimePolicyConfig {
				cfg := DefaultConfig().Runtime.Policy
				delete(cfg.Precedence.Matrix, RuntimePolicyStageReadinessAdmission)
				return cfg
			}(),
		},
		{
			name: "invalid-tie-break-mode",
			cfg: func() RuntimePolicyConfig {
				cfg := DefaultConfig().Runtime.Policy
				cfg.TieBreaker.Mode = "random"
				return cfg
			}(),
		},
		{
			name: "invalid-source-order-stage",
			cfg: func() RuntimePolicyConfig {
				cfg := DefaultConfig().Runtime.Policy
				cfg.TieBreaker.SourceOrder = []string{"action_gate", "unknown_stage"}
				return cfg
			}(),
		},
		{
			name: "duplicate-source-order-stage",
			cfg: func() RuntimePolicyConfig {
				cfg := DefaultConfig().Runtime.Policy
				cfg.TieBreaker.SourceOrder = append(RuntimePolicyCanonicalStages(), RuntimePolicyStageActionGate)
				return cfg
			}(),
		},
		{
			name: "incomplete-source-order",
			cfg: func() RuntimePolicyConfig {
				cfg := DefaultConfig().Runtime.Policy
				cfg.TieBreaker.SourceOrder = []string{RuntimePolicyStageActionGate}
				return cfg
			}(),
		},
	}

	for i := range tests {
		tc := tests[i]
		t.Run(tc.name, func(t *testing.T) {
			if err := ValidateRuntimePolicyConfig(tc.cfg); err == nil {
				t.Fatalf("case %q expected validation error", tc.name)
			}
		})
	}
}
