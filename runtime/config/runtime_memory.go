package config

import (
	"fmt"
	"path/filepath"
	"strings"
)

const (
	RuntimeMemoryModeExternalSPI       = "external_spi"
	RuntimeMemoryModeBuiltinFilesystem = "builtin_filesystem"
)

const (
	RuntimeMemoryFallbackPolicyFailFast             = "fail_fast"
	RuntimeMemoryFallbackPolicyDegradeToBuiltin     = "degrade_to_builtin"
	RuntimeMemoryFallbackPolicyDegradeWithoutMemory = "degrade_without_memory"
)

const (
	RuntimeMemoryContractVersionV1 = "memory.v1"
	RuntimeMemoryProviderGeneric   = "generic"
	RuntimeMemoryProfileGeneric    = "generic"
)

type RuntimeMemoryConfig struct {
	Mode     string                      `json:"mode"`
	External RuntimeMemoryExternalConfig `json:"external"`
	Builtin  RuntimeMemoryBuiltinConfig  `json:"builtin"`
	Fallback RuntimeMemoryFallbackConfig `json:"fallback"`
}

type RuntimeMemoryExternalConfig struct {
	Provider        string `json:"provider"`
	Profile         string `json:"profile"`
	ContractVersion string `json:"contract_version"`
}

type RuntimeMemoryBuiltinConfig struct {
	RootDir    string                               `json:"root_dir"`
	Compaction RuntimeMemoryBuiltinCompactionConfig `json:"compaction"`
}

type RuntimeMemoryBuiltinCompactionConfig struct {
	Enabled     bool  `json:"enabled"`
	MinOps      int   `json:"min_ops"`
	MaxWALBytes int64 `json:"max_wal_bytes"`
}

type RuntimeMemoryFallbackConfig struct {
	Policy string `json:"policy"`
}

func normalizeRuntimeMemoryConfig(in RuntimeMemoryConfig) RuntimeMemoryConfig {
	base := DefaultConfig().Runtime.Memory
	out := in
	out.Mode = strings.ToLower(strings.TrimSpace(out.Mode))
	if out.Mode == "" {
		out.Mode = base.Mode
	}
	out.External.Provider = strings.ToLower(strings.TrimSpace(out.External.Provider))
	out.External.Profile = strings.ToLower(strings.TrimSpace(out.External.Profile))
	out.External.ContractVersion = strings.ToLower(strings.TrimSpace(out.External.ContractVersion))
	if out.External.ContractVersion == "" {
		out.External.ContractVersion = base.External.ContractVersion
	}
	out.Builtin.RootDir = strings.TrimSpace(out.Builtin.RootDir)
	if out.Builtin.Compaction.MinOps <= 0 {
		out.Builtin.Compaction.MinOps = base.Builtin.Compaction.MinOps
	}
	if out.Builtin.Compaction.MaxWALBytes <= 0 {
		out.Builtin.Compaction.MaxWALBytes = base.Builtin.Compaction.MaxWALBytes
	}
	out.Fallback.Policy = strings.ToLower(strings.TrimSpace(out.Fallback.Policy))
	if out.Fallback.Policy == "" {
		out.Fallback.Policy = base.Fallback.Policy
	}
	return out
}

func ValidateRuntimeMemoryConfig(cfg RuntimeMemoryConfig) error {
	normalized := normalizeRuntimeMemoryConfig(cfg)
	switch normalized.Mode {
	case RuntimeMemoryModeExternalSPI, RuntimeMemoryModeBuiltinFilesystem:
	default:
		return fmt.Errorf(
			"runtime.memory.mode must be one of [%s,%s], got %q",
			RuntimeMemoryModeExternalSPI,
			RuntimeMemoryModeBuiltinFilesystem,
			cfg.Mode,
		)
	}
	switch normalized.Fallback.Policy {
	case RuntimeMemoryFallbackPolicyFailFast, RuntimeMemoryFallbackPolicyDegradeToBuiltin, RuntimeMemoryFallbackPolicyDegradeWithoutMemory:
	default:
		return fmt.Errorf(
			"runtime.memory.fallback.policy must be one of [%s,%s,%s], got %q",
			RuntimeMemoryFallbackPolicyFailFast,
			RuntimeMemoryFallbackPolicyDegradeToBuiltin,
			RuntimeMemoryFallbackPolicyDegradeWithoutMemory,
			cfg.Fallback.Policy,
		)
	}
	if normalized.External.ContractVersion != RuntimeMemoryContractVersionV1 {
		return fmt.Errorf(
			"runtime.memory.external.contract_version must be one of [%s], got %q",
			RuntimeMemoryContractVersionV1,
			cfg.External.ContractVersion,
		)
	}
	if normalized.Mode == RuntimeMemoryModeExternalSPI {
		if strings.TrimSpace(normalized.External.Provider) == "" {
			return fmt.Errorf("runtime.memory.external.provider is required when runtime.memory.mode=%s", RuntimeMemoryModeExternalSPI)
		}
		if strings.TrimSpace(normalized.External.Profile) == "" {
			return fmt.Errorf("runtime.memory.external.profile is required when runtime.memory.mode=%s", RuntimeMemoryModeExternalSPI)
		}
	}
	if normalized.Mode == RuntimeMemoryModeBuiltinFilesystem {
		if err := validateRuntimeMemoryRootDir(normalized.Builtin.RootDir); err != nil {
			return err
		}
		if strings.TrimSpace(cfg.External.Provider) != "" {
			return fmt.Errorf(
				"runtime.memory.external.provider must be empty when runtime.memory.mode=%s",
				RuntimeMemoryModeBuiltinFilesystem,
			)
		}
	}
	if normalized.Fallback.Policy == RuntimeMemoryFallbackPolicyDegradeToBuiltin {
		if err := validateRuntimeMemoryRootDir(normalized.Builtin.RootDir); err != nil {
			return err
		}
	}
	if normalized.Builtin.Compaction.MinOps <= 0 {
		return errorsRuntimeMemory("runtime.memory.builtin.compaction.min_ops must be > 0")
	}
	if normalized.Builtin.Compaction.MaxWALBytes <= 0 {
		return errorsRuntimeMemory("runtime.memory.builtin.compaction.max_wal_bytes must be > 0")
	}
	return nil
}

func validateRuntimeMemoryRootDir(root string) error {
	path := strings.TrimSpace(root)
	if path == "" {
		return errorsRuntimeMemory("runtime.memory.builtin.root_dir is required")
	}
	if strings.ContainsRune(path, '\x00') {
		return errorsRuntimeMemory("runtime.memory.builtin.root_dir contains invalid null character")
	}
	clean := filepath.Clean(path)
	if clean == "." || strings.TrimSpace(clean) == "" {
		return errorsRuntimeMemory("runtime.memory.builtin.root_dir is malformed")
	}
	return nil
}

func errorsRuntimeMemory(msg string) error {
	return fmt.Errorf("%s", strings.TrimSpace(msg))
}
