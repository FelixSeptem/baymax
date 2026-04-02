package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestRuntimeHooksToolMiddlewareSkillConfigDefaults(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.Runtime.Hooks.Enabled {
		t.Fatal("runtime.hooks.enabled = true, want false")
	}
	if cfg.Runtime.Hooks.FailMode != RuntimeHooksFailModeFailFast {
		t.Fatalf(
			"runtime.hooks.fail_mode = %q, want %q",
			cfg.Runtime.Hooks.FailMode,
			RuntimeHooksFailModeFailFast,
		)
	}
	if cfg.Runtime.Hooks.Timeout <= 0 {
		t.Fatalf("runtime.hooks.timeout = %v, want > 0", cfg.Runtime.Hooks.Timeout)
	}
	if len(cfg.Runtime.Hooks.Phases) != 6 {
		t.Fatalf("runtime.hooks.phases size = %d, want 6", len(cfg.Runtime.Hooks.Phases))
	}

	if cfg.Runtime.ToolMiddleware.Enabled {
		t.Fatal("runtime.tool_middleware.enabled = true, want false")
	}
	if cfg.Runtime.ToolMiddleware.Timeout <= 0 {
		t.Fatalf("runtime.tool_middleware.timeout = %v, want > 0", cfg.Runtime.ToolMiddleware.Timeout)
	}
	if cfg.Runtime.ToolMiddleware.FailMode != RuntimeToolMiddlewareFailModeFailFast {
		t.Fatalf(
			"runtime.tool_middleware.fail_mode = %q, want %q",
			cfg.Runtime.ToolMiddleware.FailMode,
			RuntimeToolMiddlewareFailModeFailFast,
		)
	}

	if cfg.Runtime.Skill.Discovery.Mode != RuntimeSkillDiscoveryModeAgentsMD {
		t.Fatalf(
			"runtime.skill.discovery.mode = %q, want %q",
			cfg.Runtime.Skill.Discovery.Mode,
			RuntimeSkillDiscoveryModeAgentsMD,
		)
	}
	if cfg.Runtime.Skill.Preprocess.Enabled {
		t.Fatal("runtime.skill.preprocess.enabled = true, want false")
	}
	if cfg.Runtime.Skill.Preprocess.Phase != RuntimeSkillPreprocessPhaseBeforeRunStream {
		t.Fatalf(
			"runtime.skill.preprocess.phase = %q, want %q",
			cfg.Runtime.Skill.Preprocess.Phase,
			RuntimeSkillPreprocessPhaseBeforeRunStream,
		)
	}
	if cfg.Runtime.Skill.Preprocess.FailMode != RuntimeSkillPreprocessFailModeFailFast {
		t.Fatalf(
			"runtime.skill.preprocess.fail_mode = %q, want %q",
			cfg.Runtime.Skill.Preprocess.FailMode,
			RuntimeSkillPreprocessFailModeFailFast,
		)
	}
	if cfg.Runtime.Skill.BundleMapping.PromptMode != RuntimeSkillBundleMappingPromptModeDisabled {
		t.Fatalf(
			"runtime.skill.bundle_mapping.prompt_mode = %q, want %q",
			cfg.Runtime.Skill.BundleMapping.PromptMode,
			RuntimeSkillBundleMappingPromptModeDisabled,
		)
	}
	if cfg.Runtime.Skill.BundleMapping.WhitelistMode != RuntimeSkillBundleMappingWhitelistModeDisabled {
		t.Fatalf(
			"runtime.skill.bundle_mapping.whitelist_mode = %q, want %q",
			cfg.Runtime.Skill.BundleMapping.WhitelistMode,
			RuntimeSkillBundleMappingWhitelistModeDisabled,
		)
	}
	if cfg.Runtime.Skill.BundleMapping.ConflictPolicy != RuntimeSkillBundleMappingConflictPolicyFailFast {
		t.Fatalf(
			"runtime.skill.bundle_mapping.conflict_policy = %q, want %q",
			cfg.Runtime.Skill.BundleMapping.ConflictPolicy,
			RuntimeSkillBundleMappingConflictPolicyFailFast,
		)
	}
}

func TestRuntimeHooksToolMiddlewareSkillConfigEnvOverridePrecedence(t *testing.T) {
	envRootA := filepath.ToSlash(filepath.Join(t.TempDir(), "skills-a"))
	envRootB := filepath.ToSlash(filepath.Join(t.TempDir(), "skills-b"))

	t.Setenv("BAYMAX_RUNTIME_HOOKS_ENABLED", "true")
	t.Setenv("BAYMAX_RUNTIME_HOOKS_PHASES", "before_reasoning,after_reply")
	t.Setenv("BAYMAX_RUNTIME_HOOKS_FAIL_MODE", RuntimeHooksFailModeDegrade)
	t.Setenv("BAYMAX_RUNTIME_HOOKS_TIMEOUT", "3s")
	t.Setenv("BAYMAX_RUNTIME_TOOL_MIDDLEWARE_ENABLED", "true")
	t.Setenv("BAYMAX_RUNTIME_TOOL_MIDDLEWARE_TIMEOUT", "4s")
	t.Setenv("BAYMAX_RUNTIME_TOOL_MIDDLEWARE_FAIL_MODE", RuntimeToolMiddlewareFailModeDegrade)
	t.Setenv("BAYMAX_RUNTIME_SKILL_DISCOVERY_MODE", RuntimeSkillDiscoveryModeHybrid)
	t.Setenv("BAYMAX_RUNTIME_SKILL_DISCOVERY_ROOTS", envRootA+","+envRootB)
	t.Setenv("BAYMAX_RUNTIME_SKILL_PREPROCESS_ENABLED", "true")
	t.Setenv("BAYMAX_RUNTIME_SKILL_PREPROCESS_PHASE", RuntimeSkillPreprocessPhaseBeforeRunStream)
	t.Setenv("BAYMAX_RUNTIME_SKILL_PREPROCESS_FAIL_MODE", RuntimeSkillPreprocessFailModeDegrade)
	t.Setenv("BAYMAX_RUNTIME_SKILL_BUNDLE_MAPPING_PROMPT_MODE", RuntimeSkillBundleMappingPromptModeAppend)
	t.Setenv("BAYMAX_RUNTIME_SKILL_BUNDLE_MAPPING_WHITELIST_MODE", RuntimeSkillBundleMappingWhitelistModeMerge)
	t.Setenv("BAYMAX_RUNTIME_SKILL_BUNDLE_MAPPING_CONFLICT_POLICY", RuntimeSkillBundleMappingConflictPolicyFirstWin)

	file := filepath.Join(t.TempDir(), "runtime.yaml")
	content := `
runtime:
  hooks:
    enabled: false
    phases: [before_reasoning]
    fail_mode: fail_fast
    timeout: 1s
  tool_middleware:
    enabled: false
    timeout: 1s
    fail_mode: fail_fast
  skill:
    discovery:
      mode: agents_md
      roots: [./file-only]
    preprocess:
      enabled: false
      phase: before_run_stream
      fail_mode: fail_fast
    bundle_mapping:
      prompt_mode: disabled
      whitelist_mode: disabled
      conflict_policy: fail_fast
`
	if err := os.WriteFile(file, []byte(strings.TrimSpace(content)), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	cfg, err := Load(LoadOptions{FilePath: file, EnvPrefix: "BAYMAX"})
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if !cfg.Runtime.Hooks.Enabled {
		t.Fatal("runtime.hooks.enabled = false, want true from env")
	}
	if cfg.Runtime.Hooks.FailMode != RuntimeHooksFailModeDegrade {
		t.Fatalf("runtime.hooks.fail_mode = %q, want %q from env", cfg.Runtime.Hooks.FailMode, RuntimeHooksFailModeDegrade)
	}
	if cfg.Runtime.Hooks.Timeout != 3*time.Second {
		t.Fatalf("runtime.hooks.timeout = %v, want 3s from env", cfg.Runtime.Hooks.Timeout)
	}
	if got := strings.Join(cfg.Runtime.Hooks.Phases, ","); got != "before_reasoning,after_reply" {
		t.Fatalf("runtime.hooks.phases = %q, want before_reasoning,after_reply from env", got)
	}

	if !cfg.Runtime.ToolMiddleware.Enabled {
		t.Fatal("runtime.tool_middleware.enabled = false, want true from env")
	}
	if cfg.Runtime.ToolMiddleware.Timeout != 4*time.Second {
		t.Fatalf("runtime.tool_middleware.timeout = %v, want 4s from env", cfg.Runtime.ToolMiddleware.Timeout)
	}
	if cfg.Runtime.ToolMiddleware.FailMode != RuntimeToolMiddlewareFailModeDegrade {
		t.Fatalf(
			"runtime.tool_middleware.fail_mode = %q, want %q from env",
			cfg.Runtime.ToolMiddleware.FailMode,
			RuntimeToolMiddlewareFailModeDegrade,
		)
	}

	if cfg.Runtime.Skill.Discovery.Mode != RuntimeSkillDiscoveryModeHybrid {
		t.Fatalf("runtime.skill.discovery.mode = %q, want %q from env", cfg.Runtime.Skill.Discovery.Mode, RuntimeSkillDiscoveryModeHybrid)
	}
	if len(cfg.Runtime.Skill.Discovery.Roots) != 2 {
		t.Fatalf("runtime.skill.discovery.roots size = %d, want 2", len(cfg.Runtime.Skill.Discovery.Roots))
	}
	if cfg.Runtime.Skill.Discovery.Roots[0] != filepath.Clean(envRootA) || cfg.Runtime.Skill.Discovery.Roots[1] != filepath.Clean(envRootB) {
		t.Fatalf("runtime.skill.discovery.roots = %#v, want env values", cfg.Runtime.Skill.Discovery.Roots)
	}
	if !cfg.Runtime.Skill.Preprocess.Enabled {
		t.Fatal("runtime.skill.preprocess.enabled = false, want true from env")
	}
	if cfg.Runtime.Skill.Preprocess.FailMode != RuntimeSkillPreprocessFailModeDegrade {
		t.Fatalf(
			"runtime.skill.preprocess.fail_mode = %q, want %q from env",
			cfg.Runtime.Skill.Preprocess.FailMode,
			RuntimeSkillPreprocessFailModeDegrade,
		)
	}
	if cfg.Runtime.Skill.BundleMapping.PromptMode != RuntimeSkillBundleMappingPromptModeAppend {
		t.Fatalf(
			"runtime.skill.bundle_mapping.prompt_mode = %q, want %q from env",
			cfg.Runtime.Skill.BundleMapping.PromptMode,
			RuntimeSkillBundleMappingPromptModeAppend,
		)
	}
	if cfg.Runtime.Skill.BundleMapping.WhitelistMode != RuntimeSkillBundleMappingWhitelistModeMerge {
		t.Fatalf(
			"runtime.skill.bundle_mapping.whitelist_mode = %q, want %q from env",
			cfg.Runtime.Skill.BundleMapping.WhitelistMode,
			RuntimeSkillBundleMappingWhitelistModeMerge,
		)
	}
	if cfg.Runtime.Skill.BundleMapping.ConflictPolicy != RuntimeSkillBundleMappingConflictPolicyFirstWin {
		t.Fatalf(
			"runtime.skill.bundle_mapping.conflict_policy = %q, want %q from env",
			cfg.Runtime.Skill.BundleMapping.ConflictPolicy,
			RuntimeSkillBundleMappingConflictPolicyFirstWin,
		)
	}
}

func TestRuntimeHooksToolMiddlewareSkillConfigValidationRejectsInvalidValues(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Runtime.Hooks.Phases = []string{"before_reasoning", "before_lunch"}
	if err := Validate(cfg); err == nil || !strings.Contains(err.Error(), "runtime.hooks.phases") {
		t.Fatalf("expected runtime.hooks.phases validation error, got %v", err)
	}

	cfg = DefaultConfig()
	cfg.Runtime.Skill.Discovery.Mode = "manifest"
	if err := Validate(cfg); err == nil || !strings.Contains(err.Error(), "runtime.skill.discovery.mode") {
		t.Fatalf("expected runtime.skill.discovery.mode validation error, got %v", err)
	}

	cfg = DefaultConfig()
	cfg.Runtime.Skill.Discovery.Mode = RuntimeSkillDiscoveryModeFolder
	cfg.Runtime.Skill.Discovery.Roots = []string{"."}
	if err := Validate(cfg); err == nil || !strings.Contains(err.Error(), "runtime.skill.discovery.roots") {
		t.Fatalf("expected runtime.skill.discovery.roots validation error, got %v", err)
	}

	cfg = DefaultConfig()
	cfg.Runtime.Skill.Preprocess.Phase = "before_run"
	if err := Validate(cfg); err == nil || !strings.Contains(err.Error(), "runtime.skill.preprocess.phase") {
		t.Fatalf("expected runtime.skill.preprocess.phase validation error, got %v", err)
	}

	cfg = DefaultConfig()
	cfg.Runtime.Skill.Preprocess.FailMode = "best_effort"
	if err := Validate(cfg); err == nil || !strings.Contains(err.Error(), "runtime.skill.preprocess.fail_mode") {
		t.Fatalf("expected runtime.skill.preprocess.fail_mode validation error, got %v", err)
	}

	cfg = DefaultConfig()
	cfg.Runtime.Skill.BundleMapping.ConflictPolicy = "last_win"
	if err := Validate(cfg); err == nil || !strings.Contains(err.Error(), "runtime.skill.bundle_mapping.conflict_policy") {
		t.Fatalf("expected runtime.skill.bundle_mapping.conflict_policy validation error, got %v", err)
	}
}

func TestRuntimeHooksToolMiddlewareSkillConfigInvalidBoolFailsFast(t *testing.T) {
	t.Setenv("BAYMAX_RUNTIME_HOOKS_ENABLED", "not-a-bool")
	if _, err := Load(LoadOptions{EnvPrefix: "BAYMAX"}); err == nil || !strings.Contains(err.Error(), "runtime.hooks.enabled") {
		t.Fatalf("expected strict bool parse error for runtime.hooks.enabled, got %v", err)
	}

	t.Setenv("BAYMAX_RUNTIME_HOOKS_ENABLED", "false")
	t.Setenv("BAYMAX_RUNTIME_TOOL_MIDDLEWARE_ENABLED", "not-a-bool")
	if _, err := Load(LoadOptions{EnvPrefix: "BAYMAX"}); err == nil || !strings.Contains(err.Error(), "runtime.tool_middleware.enabled") {
		t.Fatalf("expected strict bool parse error for runtime.tool_middleware.enabled, got %v", err)
	}

	t.Setenv("BAYMAX_RUNTIME_TOOL_MIDDLEWARE_ENABLED", "false")
	t.Setenv("BAYMAX_RUNTIME_SKILL_PREPROCESS_ENABLED", "not-a-bool")
	if _, err := Load(LoadOptions{EnvPrefix: "BAYMAX"}); err == nil || !strings.Contains(err.Error(), "runtime.skill.preprocess.enabled") {
		t.Fatalf("expected strict bool parse error for runtime.skill.preprocess.enabled, got %v", err)
	}
}
