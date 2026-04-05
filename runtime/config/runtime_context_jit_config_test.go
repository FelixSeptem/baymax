package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRuntimeContextJITConfigDefaults(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.Runtime.Context.JIT.ReferenceFirst.Enabled {
		t.Fatal("runtime.context.jit.reference_first.enabled = true, want false")
	}
	if cfg.Runtime.Context.JIT.ReferenceFirst.MaxRefs != 8 {
		t.Fatalf("runtime.context.jit.reference_first.max_refs = %d, want 8", cfg.Runtime.Context.JIT.ReferenceFirst.MaxRefs)
	}
	if cfg.Runtime.Context.JIT.ReferenceFirst.MaxResolveTokens != 4096 {
		t.Fatalf("runtime.context.jit.reference_first.max_resolve_tokens = %d, want 4096", cfg.Runtime.Context.JIT.ReferenceFirst.MaxResolveTokens)
	}
	if cfg.Runtime.Context.JIT.IsolateHandoff.Enabled {
		t.Fatal("runtime.context.jit.isolate_handoff.enabled = true, want false")
	}
	if cfg.Runtime.Context.JIT.IsolateHandoff.DefaultTTLMS != 300000 {
		t.Fatalf("runtime.context.jit.isolate_handoff.default_ttl_ms = %d, want 300000", cfg.Runtime.Context.JIT.IsolateHandoff.DefaultTTLMS)
	}
	if cfg.Runtime.Context.JIT.IsolateHandoff.MinConfidence != 0.60 {
		t.Fatalf("runtime.context.jit.isolate_handoff.min_confidence = %f, want 0.60", cfg.Runtime.Context.JIT.IsolateHandoff.MinConfidence)
	}
	if cfg.Runtime.Context.JIT.EditGate.Enabled {
		t.Fatal("runtime.context.jit.edit_gate.enabled = true, want false")
	}
	if cfg.Runtime.Context.JIT.EditGate.ClearAtLeastTokens != 1024 {
		t.Fatalf("runtime.context.jit.edit_gate.clear_at_least_tokens = %d, want 1024", cfg.Runtime.Context.JIT.EditGate.ClearAtLeastTokens)
	}
	if cfg.Runtime.Context.JIT.EditGate.MinGainRatio != 0.20 {
		t.Fatalf("runtime.context.jit.edit_gate.min_gain_ratio = %f, want 0.20", cfg.Runtime.Context.JIT.EditGate.MinGainRatio)
	}
	if cfg.Runtime.Context.JIT.SwapBack.Enabled {
		t.Fatal("runtime.context.jit.swap_back.enabled = true, want false")
	}
	if cfg.Runtime.Context.JIT.SwapBack.MinRelevanceScore != 0.60 {
		t.Fatalf("runtime.context.jit.swap_back.min_relevance_score = %f, want 0.60", cfg.Runtime.Context.JIT.SwapBack.MinRelevanceScore)
	}
	if cfg.Runtime.Context.JIT.LifecycleTiering.Enabled {
		t.Fatal("runtime.context.jit.lifecycle_tiering.enabled = true, want false")
	}
	if cfg.Runtime.Context.JIT.LifecycleTiering.HotTTLMS != 300000 {
		t.Fatalf("runtime.context.jit.lifecycle_tiering.hot_ttl_ms = %d, want 300000", cfg.Runtime.Context.JIT.LifecycleTiering.HotTTLMS)
	}
	if cfg.Runtime.Context.JIT.LifecycleTiering.WarmTTLMS != 1800000 {
		t.Fatalf("runtime.context.jit.lifecycle_tiering.warm_ttl_ms = %d, want 1800000", cfg.Runtime.Context.JIT.LifecycleTiering.WarmTTLMS)
	}
	if cfg.Runtime.Context.JIT.LifecycleTiering.ColdTTLMS != 7200000 {
		t.Fatalf("runtime.context.jit.lifecycle_tiering.cold_ttl_ms = %d, want 7200000", cfg.Runtime.Context.JIT.LifecycleTiering.ColdTTLMS)
	}
}

func TestRuntimeContextJITConfigEnvOverridePrecedence(t *testing.T) {
	t.Setenv("BAYMAX_RUNTIME_CONTEXT_JIT_REFERENCE_FIRST_ENABLED", "true")
	t.Setenv("BAYMAX_RUNTIME_CONTEXT_JIT_REFERENCE_FIRST_MAX_REFS", "16")
	t.Setenv("BAYMAX_RUNTIME_CONTEXT_JIT_REFERENCE_FIRST_MAX_RESOLVE_TOKENS", "8192")
	t.Setenv("BAYMAX_RUNTIME_CONTEXT_JIT_ISOLATE_HANDOFF_ENABLED", "true")
	t.Setenv("BAYMAX_RUNTIME_CONTEXT_JIT_ISOLATE_HANDOFF_DEFAULT_TTL_MS", "450000")
	t.Setenv("BAYMAX_RUNTIME_CONTEXT_JIT_ISOLATE_HANDOFF_MIN_CONFIDENCE", "0.75")
	t.Setenv("BAYMAX_RUNTIME_CONTEXT_JIT_EDIT_GATE_ENABLED", "true")
	t.Setenv("BAYMAX_RUNTIME_CONTEXT_JIT_EDIT_GATE_CLEAR_AT_LEAST_TOKENS", "1536")
	t.Setenv("BAYMAX_RUNTIME_CONTEXT_JIT_EDIT_GATE_MIN_GAIN_RATIO", "0.35")
	t.Setenv("BAYMAX_RUNTIME_CONTEXT_JIT_SWAP_BACK_ENABLED", "true")
	t.Setenv("BAYMAX_RUNTIME_CONTEXT_JIT_SWAP_BACK_MIN_RELEVANCE_SCORE", "0.70")
	t.Setenv("BAYMAX_RUNTIME_CONTEXT_JIT_LIFECYCLE_TIERING_ENABLED", "true")
	t.Setenv("BAYMAX_RUNTIME_CONTEXT_JIT_LIFECYCLE_TIERING_HOT_TTL_MS", "200000")
	t.Setenv("BAYMAX_RUNTIME_CONTEXT_JIT_LIFECYCLE_TIERING_WARM_TTL_MS", "800000")
	t.Setenv("BAYMAX_RUNTIME_CONTEXT_JIT_LIFECYCLE_TIERING_COLD_TTL_MS", "1600000")

	file := filepath.Join(t.TempDir(), "runtime.yaml")
	content := `
runtime:
  context:
    jit:
      reference_first:
        enabled: false
        max_refs: 4
        max_resolve_tokens: 1024
      isolate_handoff:
        enabled: false
        default_ttl_ms: 120000
        min_confidence: 0.40
      edit_gate:
        enabled: false
        clear_at_least_tokens: 512
        min_gain_ratio: 0.10
      swap_back:
        enabled: false
        min_relevance_score: 0.50
      lifecycle_tiering:
        enabled: false
        hot_ttl_ms: 100000
        warm_ttl_ms: 200000
        cold_ttl_ms: 300000
`
	if err := os.WriteFile(file, []byte(strings.TrimSpace(content)), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	cfg, err := Load(LoadOptions{FilePath: file, EnvPrefix: "BAYMAX"})
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if !cfg.Runtime.Context.JIT.ReferenceFirst.Enabled {
		t.Fatal("runtime.context.jit.reference_first.enabled = false, want true from env")
	}
	if cfg.Runtime.Context.JIT.ReferenceFirst.MaxRefs != 16 {
		t.Fatalf("runtime.context.jit.reference_first.max_refs = %d, want 16 from env", cfg.Runtime.Context.JIT.ReferenceFirst.MaxRefs)
	}
	if cfg.Runtime.Context.JIT.ReferenceFirst.MaxResolveTokens != 8192 {
		t.Fatalf("runtime.context.jit.reference_first.max_resolve_tokens = %d, want 8192 from env", cfg.Runtime.Context.JIT.ReferenceFirst.MaxResolveTokens)
	}
	if !cfg.Runtime.Context.JIT.IsolateHandoff.Enabled {
		t.Fatal("runtime.context.jit.isolate_handoff.enabled = false, want true from env")
	}
	if cfg.Runtime.Context.JIT.IsolateHandoff.DefaultTTLMS != 450000 {
		t.Fatalf("runtime.context.jit.isolate_handoff.default_ttl_ms = %d, want 450000 from env", cfg.Runtime.Context.JIT.IsolateHandoff.DefaultTTLMS)
	}
	if cfg.Runtime.Context.JIT.IsolateHandoff.MinConfidence != 0.75 {
		t.Fatalf("runtime.context.jit.isolate_handoff.min_confidence = %f, want 0.75 from env", cfg.Runtime.Context.JIT.IsolateHandoff.MinConfidence)
	}
	if !cfg.Runtime.Context.JIT.EditGate.Enabled {
		t.Fatal("runtime.context.jit.edit_gate.enabled = false, want true from env")
	}
	if cfg.Runtime.Context.JIT.EditGate.ClearAtLeastTokens != 1536 {
		t.Fatalf("runtime.context.jit.edit_gate.clear_at_least_tokens = %d, want 1536 from env", cfg.Runtime.Context.JIT.EditGate.ClearAtLeastTokens)
	}
	if cfg.Runtime.Context.JIT.EditGate.MinGainRatio != 0.35 {
		t.Fatalf("runtime.context.jit.edit_gate.min_gain_ratio = %f, want 0.35 from env", cfg.Runtime.Context.JIT.EditGate.MinGainRatio)
	}
	if !cfg.Runtime.Context.JIT.SwapBack.Enabled {
		t.Fatal("runtime.context.jit.swap_back.enabled = false, want true from env")
	}
	if cfg.Runtime.Context.JIT.SwapBack.MinRelevanceScore != 0.70 {
		t.Fatalf("runtime.context.jit.swap_back.min_relevance_score = %f, want 0.70 from env", cfg.Runtime.Context.JIT.SwapBack.MinRelevanceScore)
	}
	if !cfg.Runtime.Context.JIT.LifecycleTiering.Enabled {
		t.Fatal("runtime.context.jit.lifecycle_tiering.enabled = false, want true from env")
	}
	if cfg.Runtime.Context.JIT.LifecycleTiering.HotTTLMS != 200000 {
		t.Fatalf("runtime.context.jit.lifecycle_tiering.hot_ttl_ms = %d, want 200000 from env", cfg.Runtime.Context.JIT.LifecycleTiering.HotTTLMS)
	}
	if cfg.Runtime.Context.JIT.LifecycleTiering.WarmTTLMS != 800000 {
		t.Fatalf("runtime.context.jit.lifecycle_tiering.warm_ttl_ms = %d, want 800000 from env", cfg.Runtime.Context.JIT.LifecycleTiering.WarmTTLMS)
	}
	if cfg.Runtime.Context.JIT.LifecycleTiering.ColdTTLMS != 1600000 {
		t.Fatalf("runtime.context.jit.lifecycle_tiering.cold_ttl_ms = %d, want 1600000 from env", cfg.Runtime.Context.JIT.LifecycleTiering.ColdTTLMS)
	}
}

func TestRuntimeContextJITConfigValidationRejectsInvalidValues(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Runtime.Context.JIT.ReferenceFirst.MaxRefs = 0
	if err := Validate(cfg); err == nil || !strings.Contains(err.Error(), "runtime.context.jit.reference_first.max_refs") {
		t.Fatalf("expected runtime.context.jit.reference_first.max_refs validation error, got %v", err)
	}

	cfg = DefaultConfig()
	cfg.Runtime.Context.JIT.ReferenceFirst.MaxResolveTokens = 0
	if err := Validate(cfg); err == nil || !strings.Contains(err.Error(), "runtime.context.jit.reference_first.max_resolve_tokens") {
		t.Fatalf("expected runtime.context.jit.reference_first.max_resolve_tokens validation error, got %v", err)
	}

	cfg = DefaultConfig()
	cfg.Runtime.Context.JIT.IsolateHandoff.DefaultTTLMS = 0
	if err := Validate(cfg); err == nil || !strings.Contains(err.Error(), "runtime.context.jit.isolate_handoff.default_ttl_ms") {
		t.Fatalf("expected runtime.context.jit.isolate_handoff.default_ttl_ms validation error, got %v", err)
	}

	cfg = DefaultConfig()
	cfg.Runtime.Context.JIT.IsolateHandoff.MinConfidence = -0.1
	if err := Validate(cfg); err == nil || !strings.Contains(err.Error(), "runtime.context.jit.isolate_handoff.min_confidence") {
		t.Fatalf("expected runtime.context.jit.isolate_handoff.min_confidence validation error, got %v", err)
	}

	cfg = DefaultConfig()
	cfg.Runtime.Context.JIT.EditGate.ClearAtLeastTokens = 0
	if err := Validate(cfg); err == nil || !strings.Contains(err.Error(), "runtime.context.jit.edit_gate.clear_at_least_tokens") {
		t.Fatalf("expected runtime.context.jit.edit_gate.clear_at_least_tokens validation error, got %v", err)
	}

	cfg = DefaultConfig()
	cfg.Runtime.Context.JIT.EditGate.MinGainRatio = 0
	if err := Validate(cfg); err == nil || !strings.Contains(err.Error(), "runtime.context.jit.edit_gate.min_gain_ratio") {
		t.Fatalf("expected runtime.context.jit.edit_gate.min_gain_ratio validation error, got %v", err)
	}

	cfg = DefaultConfig()
	cfg.Runtime.Context.JIT.SwapBack.MinRelevanceScore = 1.2
	if err := Validate(cfg); err == nil || !strings.Contains(err.Error(), "runtime.context.jit.swap_back.min_relevance_score") {
		t.Fatalf("expected runtime.context.jit.swap_back.min_relevance_score validation error, got %v", err)
	}

	cfg = DefaultConfig()
	cfg.Runtime.Context.JIT.LifecycleTiering.HotTTLMS = 700000
	cfg.Runtime.Context.JIT.LifecycleTiering.WarmTTLMS = 600000
	if err := Validate(cfg); err == nil || !strings.Contains(err.Error(), "runtime.context.jit.lifecycle_tiering.hot_ttl_ms") {
		t.Fatalf("expected lifecycle ordering validation error, got %v", err)
	}

	cfg = DefaultConfig()
	cfg.Runtime.Context.JIT.LifecycleTiering.WarmTTLMS = 9000000
	cfg.Runtime.Context.JIT.LifecycleTiering.ColdTTLMS = 8000000
	if err := Validate(cfg); err == nil || !strings.Contains(err.Error(), "runtime.context.jit.lifecycle_tiering.warm_ttl_ms") {
		t.Fatalf("expected lifecycle ordering validation error, got %v", err)
	}
}

func TestRuntimeContextJITConfigInvalidPrimitiveFailsFast(t *testing.T) {
	t.Setenv("BAYMAX_RUNTIME_CONTEXT_JIT_REFERENCE_FIRST_ENABLED", "not-a-bool")
	if _, err := Load(LoadOptions{EnvPrefix: "BAYMAX"}); err == nil || !strings.Contains(err.Error(), "runtime.context.jit.reference_first.enabled") {
		t.Fatalf("expected strict bool parse error for runtime.context.jit.reference_first.enabled, got %v", err)
	}

	t.Setenv("BAYMAX_RUNTIME_CONTEXT_JIT_REFERENCE_FIRST_ENABLED", "true")
	t.Setenv("BAYMAX_RUNTIME_CONTEXT_JIT_REFERENCE_FIRST_MAX_REFS", "not-int")
	if _, err := Load(LoadOptions{EnvPrefix: "BAYMAX"}); err == nil || !strings.Contains(err.Error(), "runtime.context.jit.reference_first.max_refs") {
		t.Fatalf("expected strict int parse error for runtime.context.jit.reference_first.max_refs, got %v", err)
	}

	t.Setenv("BAYMAX_RUNTIME_CONTEXT_JIT_REFERENCE_FIRST_MAX_REFS", "8")
	t.Setenv("BAYMAX_RUNTIME_CONTEXT_JIT_ISOLATE_HANDOFF_MIN_CONFIDENCE", "not-float")
	if _, err := Load(LoadOptions{EnvPrefix: "BAYMAX"}); err == nil || !strings.Contains(err.Error(), "runtime.context.jit.isolate_handoff.min_confidence") {
		t.Fatalf("expected strict float parse error for runtime.context.jit.isolate_handoff.min_confidence, got %v", err)
	}

	t.Setenv("BAYMAX_RUNTIME_CONTEXT_JIT_ISOLATE_HANDOFF_MIN_CONFIDENCE", "0.6")
	t.Setenv("BAYMAX_RUNTIME_CONTEXT_JIT_EDIT_GATE_MIN_GAIN_RATIO", "bad-ratio")
	if _, err := Load(LoadOptions{EnvPrefix: "BAYMAX"}); err == nil || !strings.Contains(err.Error(), "runtime.context.jit.edit_gate.min_gain_ratio") {
		t.Fatalf("expected strict float parse error for runtime.context.jit.edit_gate.min_gain_ratio, got %v", err)
	}
}
