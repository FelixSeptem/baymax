package config

import (
	"path/filepath"
	"testing"
	"time"
)

func TestManagerRuntimeContextJITInvalidReloadRollsBack(t *testing.T) {
	file := filepath.Join(t.TempDir(), "runtime.yaml")
	writeConfig(t, file, `
runtime:
  context:
    jit:
      reference_first:
        enabled: true
        max_refs: 8
        max_resolve_tokens: 4096
      isolate_handoff:
        enabled: true
        default_ttl_ms: 300000
        min_confidence: 0.60
      edit_gate:
        enabled: true
        clear_at_least_tokens: 1024
        min_gain_ratio: 0.20
      swap_back:
        enabled: true
        min_relevance_score: 0.60
      lifecycle_tiering:
        enabled: true
        hot_ttl_ms: 300000
        warm_ttl_ms: 1800000
        cold_ttl_ms: 7200000
reload:
  enabled: true
  debounce: 20ms
`)

	mgr, err := NewManager(ManagerOptions{
		FilePath:        file,
		EnvPrefix:       "BAYMAX_A67_CTX_JIT_MANAGER_TEST",
		EnableHotReload: true,
	})
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}
	defer func() { _ = mgr.Close() }()

	before := mgr.EffectiveConfig().Runtime.Context.JIT
	if !before.ReferenceFirst.Enabled || !before.EditGate.Enabled {
		t.Fatalf("expected runtime.context.jit enabled before reload, got %#v", before)
	}

	writeConfig(t, file, `
runtime:
  context:
    jit:
      reference_first:
        enabled: true
        max_refs: 8
        max_resolve_tokens: 4096
      isolate_handoff:
        enabled: true
        default_ttl_ms: 300000
        min_confidence: 0.60
      edit_gate:
        enabled: true
        clear_at_least_tokens: 1024
        min_gain_ratio: 0.20
      swap_back:
        enabled: true
        min_relevance_score: 0.60
      lifecycle_tiering:
        enabled: true
        hot_ttl_ms: 300000
        warm_ttl_ms: 200000
        cold_ttl_ms: 7200000
reload:
  enabled: true
  debounce: 20ms
`)
	time.Sleep(250 * time.Millisecond)

	after := mgr.EffectiveConfig().Runtime.Context.JIT
	if after.LifecycleTiering.WarmTTLMS != before.LifecycleTiering.WarmTTLMS {
		t.Fatalf(
			"invalid runtime.context.jit.lifecycle_tiering.warm_ttl_ms should rollback, got %d want %d",
			after.LifecycleTiering.WarmTTLMS,
			before.LifecycleTiering.WarmTTLMS,
		)
	}
	reloads := mgr.RecentReloads(1)
	if len(reloads) == 0 || reloads[0].Success {
		t.Fatalf("expected failed reload record, got %#v", reloads)
	}
}

func TestManagerRuntimeContextJITInvalidReloadRollsBackWithEnvPrecedence(t *testing.T) {
	t.Setenv("BAYMAX_A67_CTX_JIT_ENV_TEST_RUNTIME_CONTEXT_JIT_REFERENCE_FIRST_MAX_REFS", "19")
	file := filepath.Join(t.TempDir(), "runtime.yaml")
	writeConfig(t, file, `
runtime:
  context:
    jit:
      reference_first:
        enabled: true
        max_refs: 8
        max_resolve_tokens: 4096
      isolate_handoff:
        enabled: true
        default_ttl_ms: 300000
        min_confidence: 0.60
      edit_gate:
        enabled: true
        clear_at_least_tokens: 1024
        min_gain_ratio: 0.20
      swap_back:
        enabled: true
        min_relevance_score: 0.60
      lifecycle_tiering:
        enabled: true
        hot_ttl_ms: 300000
        warm_ttl_ms: 1800000
        cold_ttl_ms: 7200000
reload:
  enabled: true
  debounce: 20ms
`)

	mgr, err := NewManager(ManagerOptions{
		FilePath:        file,
		EnvPrefix:       "BAYMAX_A67_CTX_JIT_ENV_TEST",
		EnableHotReload: true,
	})
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}
	defer func() { _ = mgr.Close() }()

	before := mgr.EffectiveConfig().Runtime.Context.JIT
	if before.ReferenceFirst.MaxRefs != 19 {
		t.Fatalf("env precedence max_refs=%d, want 19", before.ReferenceFirst.MaxRefs)
	}

	writeConfig(t, file, `
runtime:
  context:
    jit:
      reference_first:
        enabled: true
        max_refs: 8
        max_resolve_tokens: 4096
      isolate_handoff:
        enabled: true
        default_ttl_ms: 300000
        min_confidence: 0.60
      edit_gate:
        enabled: true
        clear_at_least_tokens: 1024
        min_gain_ratio: 0
      swap_back:
        enabled: true
        min_relevance_score: 0.60
      lifecycle_tiering:
        enabled: true
        hot_ttl_ms: 300000
        warm_ttl_ms: 1800000
        cold_ttl_ms: 7200000
reload:
  enabled: true
  debounce: 20ms
`)
	time.Sleep(250 * time.Millisecond)

	after := mgr.EffectiveConfig().Runtime.Context.JIT
	if after.ReferenceFirst.MaxRefs != 19 {
		t.Fatalf("env-derived max_refs should remain 19 after failed reload, got %d", after.ReferenceFirst.MaxRefs)
	}
	if after.EditGate.MinGainRatio != before.EditGate.MinGainRatio {
		t.Fatalf(
			"invalid runtime.context.jit.edit_gate.min_gain_ratio should rollback, got %f want %f",
			after.EditGate.MinGainRatio,
			before.EditGate.MinGainRatio,
		)
	}
	reloads := mgr.RecentReloads(1)
	if len(reloads) == 0 || reloads[0].Success {
		t.Fatalf("expected failed reload record, got %#v", reloads)
	}
}
