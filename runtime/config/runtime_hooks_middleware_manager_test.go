package config

import (
	"path/filepath"
	"testing"
	"time"
)

func TestManagerRuntimeHooksAndSkillInvalidReloadRollsBack(t *testing.T) {
	file := filepath.Join(t.TempDir(), "runtime.yaml")
	writeConfig(t, file, `
runtime:
  hooks:
    enabled: true
    phases: [before_reasoning, after_reasoning, before_acting, after_acting, before_reply, after_reply]
    fail_mode: fail_fast
    timeout: 2s
  tool_middleware:
    enabled: true
    timeout: 2s
    fail_mode: fail_fast
  skill:
    discovery:
      mode: folder
      roots: [./skills]
    preprocess:
      enabled: true
      phase: before_run_stream
      fail_mode: degrade
    bundle_mapping:
      prompt_mode: append
      whitelist_mode: merge
      conflict_policy: first_win
reload:
  enabled: true
  debounce: 20ms
`)

	mgr, err := NewManager(ManagerOptions{FilePath: file, EnvPrefix: "BAYMAX_A65_TEST", EnableHotReload: true})
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}
	defer func() { _ = mgr.Close() }()

	before := mgr.EffectiveConfig()
	if before.Runtime.Skill.BundleMapping.ConflictPolicy != RuntimeSkillBundleMappingConflictPolicyFirstWin {
		t.Fatalf(
			"before runtime.skill.bundle_mapping.conflict_policy = %q, want %q",
			before.Runtime.Skill.BundleMapping.ConflictPolicy,
			RuntimeSkillBundleMappingConflictPolicyFirstWin,
		)
	}

	writeConfig(t, file, `
runtime:
  hooks:
    enabled: true
    phases: [before_reasoning, after_reasoning, before_acting, after_acting, before_reply, after_reply]
    fail_mode: fail_fast
    timeout: 2s
  tool_middleware:
    enabled: true
    timeout: 2s
    fail_mode: fail_fast
  skill:
    discovery:
      mode: folder
      roots: [./skills]
    preprocess:
      enabled: true
      phase: before_run_stream
      fail_mode: degrade
    bundle_mapping:
      prompt_mode: append
      whitelist_mode: merge
      conflict_policy: last_win
reload:
  enabled: true
  debounce: 20ms
`)
	time.Sleep(250 * time.Millisecond)

	after := mgr.EffectiveConfig()
	if after.Runtime.Skill.BundleMapping.ConflictPolicy != before.Runtime.Skill.BundleMapping.ConflictPolicy {
		t.Fatalf(
			"invalid runtime.skill.bundle_mapping.conflict_policy should rollback, got %q want %q",
			after.Runtime.Skill.BundleMapping.ConflictPolicy,
			before.Runtime.Skill.BundleMapping.ConflictPolicy,
		)
	}
	reloads := mgr.RecentReloads(1)
	if len(reloads) == 0 || reloads[0].Success {
		t.Fatalf("expected failed reload record, got %#v", reloads)
	}
	if got := reloads[0].Error; got == "" {
		t.Fatalf("expected reload error for invalid runtime.skill.bundle_mapping.conflict_policy")
	}
}
