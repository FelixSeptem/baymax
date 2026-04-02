package config

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"
)

const (
	RuntimeHookPhaseBeforeReasoning = "before_reasoning"
	RuntimeHookPhaseAfterReasoning  = "after_reasoning"
	RuntimeHookPhaseBeforeActing    = "before_acting"
	RuntimeHookPhaseAfterActing     = "after_acting"
	RuntimeHookPhaseBeforeReply     = "before_reply"
	RuntimeHookPhaseAfterReply      = "after_reply"
)

const (
	RuntimeHooksFailModeFailFast = "fail_fast"
	RuntimeHooksFailModeDegrade  = "degrade"
)

const (
	RuntimeToolMiddlewareFailModeFailFast = "fail_fast"
	RuntimeToolMiddlewareFailModeDegrade  = "degrade"
)

const (
	RuntimeSkillDiscoveryModeAgentsMD = "agents_md"
	RuntimeSkillDiscoveryModeFolder   = "folder"
	RuntimeSkillDiscoveryModeHybrid   = "hybrid"
)

const (
	RuntimeSkillPreprocessPhaseBeforeRunStream = "before_run_stream"
	RuntimeSkillPreprocessFailModeFailFast     = "fail_fast"
	RuntimeSkillPreprocessFailModeDegrade      = "degrade"
)

const (
	RuntimeSkillBundleMappingPromptModeDisabled = "disabled"
	RuntimeSkillBundleMappingPromptModeAppend   = "append"
)

const (
	RuntimeSkillBundleMappingWhitelistModeDisabled = "disabled"
	RuntimeSkillBundleMappingWhitelistModeMerge    = "merge"
)

const (
	RuntimeSkillBundleMappingConflictPolicyFailFast = "fail_fast"
	RuntimeSkillBundleMappingConflictPolicyFirstWin = "first_win"
)

type RuntimeHooksConfig struct {
	Enabled  bool          `json:"enabled"`
	Phases   []string      `json:"phases"`
	FailMode string        `json:"fail_mode"`
	Timeout  time.Duration `json:"timeout"`
}

type RuntimeToolMiddlewareConfig struct {
	Enabled  bool          `json:"enabled"`
	Timeout  time.Duration `json:"timeout"`
	FailMode string        `json:"fail_mode"`
}

type RuntimeSkillConfig struct {
	Discovery     RuntimeSkillDiscoveryConfig     `json:"discovery"`
	Preprocess    RuntimeSkillPreprocessConfig    `json:"preprocess"`
	BundleMapping RuntimeSkillBundleMappingConfig `json:"bundle_mapping"`
}

type RuntimeSkillDiscoveryConfig struct {
	Mode  string   `json:"mode"`
	Roots []string `json:"roots"`
}

type RuntimeSkillPreprocessConfig struct {
	Enabled  bool   `json:"enabled"`
	Phase    string `json:"phase"`
	FailMode string `json:"fail_mode"`
}

type RuntimeSkillBundleMappingConfig struct {
	PromptMode     string `json:"prompt_mode"`
	WhitelistMode  string `json:"whitelist_mode"`
	ConflictPolicy string `json:"conflict_policy"`
}

func normalizeRuntimeHooksConfig(in RuntimeHooksConfig) RuntimeHooksConfig {
	base := DefaultConfig().Runtime.Hooks
	out := in
	out.Phases = normalizeRuntimeHookPhases(out.Phases)
	if len(out.Phases) == 0 {
		out.Phases = append([]string(nil), base.Phases...)
	}
	out.FailMode = strings.ToLower(strings.TrimSpace(out.FailMode))
	if out.FailMode == "" {
		out.FailMode = strings.ToLower(strings.TrimSpace(base.FailMode))
	}
	if out.Timeout <= 0 {
		out.Timeout = base.Timeout
	}
	return out
}

func normalizeRuntimeToolMiddlewareConfig(in RuntimeToolMiddlewareConfig) RuntimeToolMiddlewareConfig {
	base := DefaultConfig().Runtime.ToolMiddleware
	out := in
	out.FailMode = strings.ToLower(strings.TrimSpace(out.FailMode))
	if out.FailMode == "" {
		out.FailMode = strings.ToLower(strings.TrimSpace(base.FailMode))
	}
	if out.Timeout <= 0 {
		out.Timeout = base.Timeout
	}
	return out
}

func normalizeRuntimeSkillConfig(in RuntimeSkillConfig) RuntimeSkillConfig {
	base := DefaultConfig().Runtime.Skill
	out := in
	out.Discovery.Mode = strings.ToLower(strings.TrimSpace(out.Discovery.Mode))
	if out.Discovery.Mode == "" {
		out.Discovery.Mode = strings.ToLower(strings.TrimSpace(base.Discovery.Mode))
	}
	out.Discovery.Roots = normalizeRuntimeSkillDiscoveryRoots(out.Discovery.Roots)
	out.Preprocess.Phase = strings.ToLower(strings.TrimSpace(out.Preprocess.Phase))
	if out.Preprocess.Phase == "" {
		out.Preprocess.Phase = strings.ToLower(strings.TrimSpace(base.Preprocess.Phase))
	}
	out.Preprocess.FailMode = strings.ToLower(strings.TrimSpace(out.Preprocess.FailMode))
	if out.Preprocess.FailMode == "" {
		out.Preprocess.FailMode = strings.ToLower(strings.TrimSpace(base.Preprocess.FailMode))
	}
	out.BundleMapping.PromptMode = strings.ToLower(strings.TrimSpace(out.BundleMapping.PromptMode))
	if out.BundleMapping.PromptMode == "" {
		out.BundleMapping.PromptMode = strings.ToLower(strings.TrimSpace(base.BundleMapping.PromptMode))
	}
	out.BundleMapping.WhitelistMode = strings.ToLower(strings.TrimSpace(out.BundleMapping.WhitelistMode))
	if out.BundleMapping.WhitelistMode == "" {
		out.BundleMapping.WhitelistMode = strings.ToLower(strings.TrimSpace(base.BundleMapping.WhitelistMode))
	}
	out.BundleMapping.ConflictPolicy = strings.ToLower(strings.TrimSpace(out.BundleMapping.ConflictPolicy))
	if out.BundleMapping.ConflictPolicy == "" {
		out.BundleMapping.ConflictPolicy = strings.ToLower(strings.TrimSpace(base.BundleMapping.ConflictPolicy))
	}
	return out
}

func ValidateRuntimeHooksConfig(cfg RuntimeHooksConfig) error {
	normalized := normalizeRuntimeHooksConfig(cfg)
	switch normalized.FailMode {
	case RuntimeHooksFailModeFailFast, RuntimeHooksFailModeDegrade:
	default:
		return fmt.Errorf(
			"runtime.hooks.fail_mode must be one of [%s,%s], got %q",
			RuntimeHooksFailModeFailFast,
			RuntimeHooksFailModeDegrade,
			cfg.FailMode,
		)
	}
	if normalized.Timeout <= 0 {
		return fmt.Errorf("runtime.hooks.timeout must be > 0")
	}
	if len(normalized.Phases) == 0 {
		return fmt.Errorf("runtime.hooks.phases must not be empty")
	}
	for i := range normalized.Phases {
		if !isRuntimeHookPhase(normalized.Phases[i]) {
			return fmt.Errorf(
				"runtime.hooks.phases[%d] must be one of [%s,%s,%s,%s,%s,%s], got %q",
				i,
				RuntimeHookPhaseBeforeReasoning,
				RuntimeHookPhaseAfterReasoning,
				RuntimeHookPhaseBeforeActing,
				RuntimeHookPhaseAfterActing,
				RuntimeHookPhaseBeforeReply,
				RuntimeHookPhaseAfterReply,
				cfg.Phases[i],
			)
		}
	}
	return nil
}

func ValidateRuntimeToolMiddlewareConfig(cfg RuntimeToolMiddlewareConfig) error {
	normalized := normalizeRuntimeToolMiddlewareConfig(cfg)
	if normalized.Timeout <= 0 {
		return fmt.Errorf("runtime.tool_middleware.timeout must be > 0")
	}
	switch normalized.FailMode {
	case RuntimeToolMiddlewareFailModeFailFast, RuntimeToolMiddlewareFailModeDegrade:
	default:
		return fmt.Errorf(
			"runtime.tool_middleware.fail_mode must be one of [%s,%s], got %q",
			RuntimeToolMiddlewareFailModeFailFast,
			RuntimeToolMiddlewareFailModeDegrade,
			cfg.FailMode,
		)
	}
	return nil
}

func ValidateRuntimeSkillConfig(cfg RuntimeSkillConfig) error {
	if err := ValidateRuntimeSkillDiscoveryConfig(cfg.Discovery); err != nil {
		return err
	}
	if err := ValidateRuntimeSkillPreprocessConfig(cfg.Preprocess); err != nil {
		return err
	}
	return ValidateRuntimeSkillBundleMappingConfig(cfg.BundleMapping)
}

func ValidateRuntimeSkillDiscoveryConfig(cfg RuntimeSkillDiscoveryConfig) error {
	normalized := cfg
	normalized.Mode = strings.ToLower(strings.TrimSpace(normalized.Mode))
	normalized.Roots = normalizeRuntimeSkillDiscoveryRoots(normalized.Roots)
	switch normalized.Mode {
	case RuntimeSkillDiscoveryModeAgentsMD, RuntimeSkillDiscoveryModeFolder, RuntimeSkillDiscoveryModeHybrid:
	default:
		return fmt.Errorf(
			"runtime.skill.discovery.mode must be one of [%s,%s,%s], got %q",
			RuntimeSkillDiscoveryModeAgentsMD,
			RuntimeSkillDiscoveryModeFolder,
			RuntimeSkillDiscoveryModeHybrid,
			cfg.Mode,
		)
	}
	if normalized.Mode == RuntimeSkillDiscoveryModeFolder || normalized.Mode == RuntimeSkillDiscoveryModeHybrid {
		if len(normalized.Roots) == 0 {
			return fmt.Errorf(
				"runtime.skill.discovery.roots must not be empty when runtime.skill.discovery.mode is %q or %q",
				RuntimeSkillDiscoveryModeFolder,
				RuntimeSkillDiscoveryModeHybrid,
			)
		}
	}
	for i := range normalized.Roots {
		if err := validateRuntimeSkillDiscoveryRoot(normalized.Roots[i], i); err != nil {
			return err
		}
	}
	return nil
}

func ValidateRuntimeSkillPreprocessConfig(cfg RuntimeSkillPreprocessConfig) error {
	normalized := cfg
	normalized.Phase = strings.ToLower(strings.TrimSpace(normalized.Phase))
	normalized.FailMode = strings.ToLower(strings.TrimSpace(normalized.FailMode))
	switch normalized.Phase {
	case RuntimeSkillPreprocessPhaseBeforeRunStream:
	default:
		return fmt.Errorf(
			"runtime.skill.preprocess.phase must be one of [%s], got %q",
			RuntimeSkillPreprocessPhaseBeforeRunStream,
			cfg.Phase,
		)
	}
	switch normalized.FailMode {
	case RuntimeSkillPreprocessFailModeFailFast, RuntimeSkillPreprocessFailModeDegrade:
	default:
		return fmt.Errorf(
			"runtime.skill.preprocess.fail_mode must be one of [%s,%s], got %q",
			RuntimeSkillPreprocessFailModeFailFast,
			RuntimeSkillPreprocessFailModeDegrade,
			cfg.FailMode,
		)
	}
	return nil
}

func ValidateRuntimeSkillBundleMappingConfig(cfg RuntimeSkillBundleMappingConfig) error {
	normalized := cfg
	normalized.PromptMode = strings.ToLower(strings.TrimSpace(normalized.PromptMode))
	normalized.WhitelistMode = strings.ToLower(strings.TrimSpace(normalized.WhitelistMode))
	normalized.ConflictPolicy = strings.ToLower(strings.TrimSpace(normalized.ConflictPolicy))
	switch normalized.PromptMode {
	case RuntimeSkillBundleMappingPromptModeDisabled, RuntimeSkillBundleMappingPromptModeAppend:
	default:
		return fmt.Errorf(
			"runtime.skill.bundle_mapping.prompt_mode must be one of [%s,%s], got %q",
			RuntimeSkillBundleMappingPromptModeDisabled,
			RuntimeSkillBundleMappingPromptModeAppend,
			cfg.PromptMode,
		)
	}
	switch normalized.WhitelistMode {
	case RuntimeSkillBundleMappingWhitelistModeDisabled, RuntimeSkillBundleMappingWhitelistModeMerge:
	default:
		return fmt.Errorf(
			"runtime.skill.bundle_mapping.whitelist_mode must be one of [%s,%s], got %q",
			RuntimeSkillBundleMappingWhitelistModeDisabled,
			RuntimeSkillBundleMappingWhitelistModeMerge,
			cfg.WhitelistMode,
		)
	}
	switch normalized.ConflictPolicy {
	case RuntimeSkillBundleMappingConflictPolicyFailFast, RuntimeSkillBundleMappingConflictPolicyFirstWin:
	default:
		return fmt.Errorf(
			"runtime.skill.bundle_mapping.conflict_policy must be one of [%s,%s], got %q",
			RuntimeSkillBundleMappingConflictPolicyFailFast,
			RuntimeSkillBundleMappingConflictPolicyFirstWin,
			cfg.ConflictPolicy,
		)
	}
	return nil
}

func normalizeRuntimeHookPhases(in []string) []string {
	if len(in) == 0 {
		return nil
	}
	out := make([]string, 0, len(in))
	seen := map[string]struct{}{}
	for _, raw := range in {
		parts := strings.Split(raw, ",")
		for _, part := range parts {
			phase := strings.ToLower(strings.TrimSpace(part))
			if phase == "" {
				continue
			}
			if _, ok := seen[phase]; ok {
				continue
			}
			seen[phase] = struct{}{}
			out = append(out, phase)
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func normalizeRuntimeSkillDiscoveryRoots(in []string) []string {
	if len(in) == 0 {
		return nil
	}
	out := make([]string, 0, len(in))
	seen := map[string]struct{}{}
	for _, raw := range in {
		parts := strings.Split(raw, ",")
		for _, part := range parts {
			trimmed := strings.TrimSpace(part)
			if trimmed == "" {
				continue
			}
			clean := filepath.Clean(trimmed)
			key := strings.ToLower(strings.TrimSpace(clean))
			if key == "" {
				continue
			}
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			out = append(out, clean)
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func isRuntimeHookPhase(raw string) bool {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case RuntimeHookPhaseBeforeReasoning,
		RuntimeHookPhaseAfterReasoning,
		RuntimeHookPhaseBeforeActing,
		RuntimeHookPhaseAfterActing,
		RuntimeHookPhaseBeforeReply,
		RuntimeHookPhaseAfterReply:
		return true
	default:
		return false
	}
}

func validateRuntimeSkillDiscoveryRoot(root string, i int) error {
	trimmed := strings.TrimSpace(root)
	if trimmed == "" {
		return fmt.Errorf("runtime.skill.discovery.roots[%d] must not be empty", i)
	}
	if strings.ContainsRune(trimmed, '\x00') {
		return fmt.Errorf("runtime.skill.discovery.roots[%d] contains invalid null character", i)
	}
	clean := filepath.Clean(trimmed)
	if clean == "." || strings.TrimSpace(clean) == "" {
		return fmt.Errorf("runtime.skill.discovery.roots[%d] is malformed", i)
	}
	return nil
}
