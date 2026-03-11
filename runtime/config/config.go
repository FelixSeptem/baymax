package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/FelixSeptem/baymax/core/types"
	"github.com/spf13/viper"
)

const (
	DefaultEnvPrefix = "BAYMAX"
)

const (
	ProfileDev            = "dev"
	ProfileDefault        = "default"
	ProfileHighThroughput = "high-throughput"
	ProfileHighReliab     = "high-reliability"
)

type Config struct {
	MCP         MCPConfig         `json:"mcp"`
	Concurrency ConcurrencyConfig `json:"concurrency"`
	Diagnostics DiagnosticsConfig `json:"diagnostics"`
	Reload      ReloadConfig      `json:"reload"`
}

type MCPConfig struct {
	ActiveProfile string                            `json:"active_profile"`
	Profiles      map[string]types.MCPRuntimePolicy `json:"profiles"`
}

type ConcurrencyConfig struct {
	LocalMaxWorkers int                    `json:"local_max_workers"`
	LocalQueueSize  int                    `json:"local_queue_size"`
	Backpressure    types.BackpressureMode `json:"backpressure"`
}

type DiagnosticsConfig struct {
	MaxCallRecords  int `json:"max_call_records"`
	MaxRunRecords   int `json:"max_run_records"`
	MaxReloadErrors int `json:"max_reload_errors"`
	MaxSkillRecords int `json:"max_skill_records"`
}

type ReloadConfig struct {
	Enabled  bool          `json:"enabled"`
	Debounce time.Duration `json:"debounce"`
}

type LoadOptions struct {
	FilePath  string
	EnvPrefix string
}

func DefaultConfig() Config {
	return Config{
		MCP: MCPConfig{
			ActiveProfile: ProfileDefault,
			Profiles: map[string]types.MCPRuntimePolicy{
				ProfileDev:            defaultPolicyFor(ProfileDev),
				ProfileDefault:        defaultPolicyFor(ProfileDefault),
				ProfileHighThroughput: defaultPolicyFor(ProfileHighThroughput),
				ProfileHighReliab:     defaultPolicyFor(ProfileHighReliab),
			},
		},
		Concurrency: ConcurrencyConfig{
			LocalMaxWorkers: 8,
			LocalQueueSize:  32,
			Backpressure:    types.BackpressureBlock,
		},
		Diagnostics: DiagnosticsConfig{
			MaxCallRecords:  200,
			MaxRunRecords:   200,
			MaxReloadErrors: 100,
			MaxSkillRecords: 200,
		},
		Reload: ReloadConfig{
			Enabled:  false,
			Debounce: 200 * time.Millisecond,
		},
	}
}

func Load(opts LoadOptions) (Config, error) {
	cfg, _, err := loadWithSnapshot(opts)
	return cfg, err
}

func loadWithSnapshot(opts LoadOptions) (Config, map[string]any, error) {
	v := viper.New()
	envPrefix := strings.TrimSpace(opts.EnvPrefix)
	if envPrefix == "" {
		envPrefix = DefaultEnvPrefix
	}
	v.SetEnvPrefix(envPrefix)
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()
	applyDefaults(v)

	if strings.TrimSpace(opts.FilePath) != "" {
		v.SetConfigFile(opts.FilePath)
		v.SetConfigType("yaml")
		if err := v.ReadInConfig(); err != nil {
			return Config{}, nil, fmt.Errorf("read runtime config: %w", err)
		}
	}

	cfg := buildConfig(v)
	if err := Validate(cfg); err != nil {
		return Config{}, nil, err
	}
	raw, err := toMap(cfg)
	if err != nil {
		return Config{}, nil, err
	}
	return cfg, raw, nil
}

func Validate(cfg Config) error {
	if strings.TrimSpace(cfg.MCP.ActiveProfile) == "" {
		return errors.New("mcp.active_profile is required")
	}
	if len(cfg.MCP.Profiles) == 0 {
		return errors.New("mcp.profiles must not be empty")
	}
	if _, ok := cfg.MCP.Profiles[cfg.MCP.ActiveProfile]; !ok {
		return fmt.Errorf("mcp.active_profile=%q not found in mcp.profiles", cfg.MCP.ActiveProfile)
	}
	for name, p := range cfg.MCP.Profiles {
		if strings.TrimSpace(name) == "" {
			return errors.New("mcp.profiles contains empty profile name")
		}
		if p.CallTimeout <= 0 {
			return fmt.Errorf("mcp.profiles.%s.call_timeout must be > 0", name)
		}
		if p.Retry < 0 {
			return fmt.Errorf("mcp.profiles.%s.retry must be >= 0", name)
		}
		if p.Backoff <= 0 {
			return fmt.Errorf("mcp.profiles.%s.backoff must be > 0", name)
		}
		if p.QueueSize <= 0 {
			return fmt.Errorf("mcp.profiles.%s.queue_size must be > 0", name)
		}
		if p.ReadPoolSize <= 0 {
			return fmt.Errorf("mcp.profiles.%s.read_pool_size must be > 0", name)
		}
		if p.WritePoolSize <= 0 {
			return fmt.Errorf("mcp.profiles.%s.write_pool_size must be > 0", name)
		}
		if err := validateBackpressure(p.Backpressure, fmt.Sprintf("mcp.profiles.%s.backpressure", name)); err != nil {
			return err
		}
	}
	if cfg.Concurrency.LocalMaxWorkers <= 0 {
		return errors.New("concurrency.local_max_workers must be > 0")
	}
	if cfg.Concurrency.LocalQueueSize <= 0 {
		return errors.New("concurrency.local_queue_size must be > 0")
	}
	if err := validateBackpressure(cfg.Concurrency.Backpressure, "concurrency.backpressure"); err != nil {
		return err
	}
	if cfg.Diagnostics.MaxCallRecords <= 0 {
		return errors.New("diagnostics.max_call_records must be > 0")
	}
	if cfg.Diagnostics.MaxRunRecords <= 0 {
		return errors.New("diagnostics.max_run_records must be > 0")
	}
	if cfg.Diagnostics.MaxReloadErrors <= 0 {
		return errors.New("diagnostics.max_reload_errors must be > 0")
	}
	if cfg.Diagnostics.MaxSkillRecords <= 0 {
		return errors.New("diagnostics.max_skill_records must be > 0")
	}
	if cfg.Reload.Debounce <= 0 {
		return errors.New("reload.debounce must be > 0")
	}
	return nil
}

func validateBackpressure(v types.BackpressureMode, field string) error {
	switch v {
	case types.BackpressureBlock, types.BackpressureReject:
		return nil
	default:
		return fmt.Errorf("%s must be one of [block,reject]", field)
	}
}

func applyDefaults(v *viper.Viper) {
	base := DefaultConfig()
	v.SetDefault("mcp.active_profile", base.MCP.ActiveProfile)
	for name, p := range base.MCP.Profiles {
		prefix := "mcp.profiles." + name + "."
		v.SetDefault(prefix+"call_timeout", p.CallTimeout)
		v.SetDefault(prefix+"retry", p.Retry)
		v.SetDefault(prefix+"backoff", p.Backoff)
		v.SetDefault(prefix+"queue_size", p.QueueSize)
		v.SetDefault(prefix+"backpressure", string(p.Backpressure))
		v.SetDefault(prefix+"read_pool_size", p.ReadPoolSize)
		v.SetDefault(prefix+"write_pool_size", p.WritePoolSize)
	}
	v.SetDefault("concurrency.local_max_workers", base.Concurrency.LocalMaxWorkers)
	v.SetDefault("concurrency.local_queue_size", base.Concurrency.LocalQueueSize)
	v.SetDefault("concurrency.backpressure", string(base.Concurrency.Backpressure))
	v.SetDefault("diagnostics.max_call_records", base.Diagnostics.MaxCallRecords)
	v.SetDefault("diagnostics.max_run_records", base.Diagnostics.MaxRunRecords)
	v.SetDefault("diagnostics.max_reload_errors", base.Diagnostics.MaxReloadErrors)
	v.SetDefault("diagnostics.max_skill_records", base.Diagnostics.MaxSkillRecords)
	v.SetDefault("reload.enabled", base.Reload.Enabled)
	v.SetDefault("reload.debounce", base.Reload.Debounce)
}

func buildConfig(v *viper.Viper) Config {
	cfg := DefaultConfig()
	cfg.MCP.ActiveProfile = strings.TrimSpace(v.GetString("mcp.active_profile"))
	cfg.Concurrency.LocalMaxWorkers = v.GetInt("concurrency.local_max_workers")
	cfg.Concurrency.LocalQueueSize = v.GetInt("concurrency.local_queue_size")
	cfg.Concurrency.Backpressure = types.BackpressureMode(v.GetString("concurrency.backpressure"))
	cfg.Diagnostics.MaxCallRecords = v.GetInt("diagnostics.max_call_records")
	cfg.Diagnostics.MaxRunRecords = v.GetInt("diagnostics.max_run_records")
	cfg.Diagnostics.MaxReloadErrors = v.GetInt("diagnostics.max_reload_errors")
	cfg.Diagnostics.MaxSkillRecords = v.GetInt("diagnostics.max_skill_records")
	cfg.Reload.Enabled = v.GetBool("reload.enabled")
	cfg.Reload.Debounce = v.GetDuration("reload.debounce")

	names := map[string]struct{}{}
	for name := range cfg.MCP.Profiles {
		names[name] = struct{}{}
	}
	for name := range v.GetStringMap("mcp.profiles") {
		names[name] = struct{}{}
	}
	for name := range names {
		p := cfg.MCP.Profiles[name]
		prefix := "mcp.profiles." + name + "."
		if v.IsSet(prefix + "call_timeout") {
			p.CallTimeout = v.GetDuration(prefix + "call_timeout")
		}
		if v.IsSet(prefix + "retry") {
			p.Retry = v.GetInt(prefix + "retry")
		}
		if v.IsSet(prefix + "backoff") {
			p.Backoff = v.GetDuration(prefix + "backoff")
		}
		if v.IsSet(prefix + "queue_size") {
			p.QueueSize = v.GetInt(prefix + "queue_size")
		}
		if v.IsSet(prefix + "backpressure") {
			p.Backpressure = types.BackpressureMode(v.GetString(prefix + "backpressure"))
		}
		if v.IsSet(prefix + "read_pool_size") {
			p.ReadPoolSize = v.GetInt(prefix + "read_pool_size")
		}
		if v.IsSet(prefix + "write_pool_size") {
			p.WritePoolSize = v.GetInt(prefix + "write_pool_size")
		}
		cfg.MCP.Profiles[name] = p
	}
	return cfg
}

func toMap(cfg Config) (map[string]any, error) {
	b, err := json.Marshal(cfg)
	if err != nil {
		return nil, fmt.Errorf("marshal runtime config: %w", err)
	}
	out := map[string]any{}
	if err := json.Unmarshal(b, &out); err != nil {
		return nil, fmt.Errorf("unmarshal runtime config map: %w", err)
	}
	return out, nil
}

func ResolveMCPPolicyWithConfig(cfg Config, profile string, override *types.MCPRuntimePolicy) (types.MCPRuntimePolicy, error) {
	name := strings.TrimSpace(profile)
	if name == "" {
		name = cfg.MCP.ActiveProfile
	}
	base, ok := cfg.MCP.Profiles[name]
	if !ok {
		return types.MCPRuntimePolicy{}, fmt.Errorf("profile %q not configured", name)
	}
	return applyPolicyOverride(base, override), nil
}

func applyPolicyOverride(base types.MCPRuntimePolicy, override *types.MCPRuntimePolicy) types.MCPRuntimePolicy {
	if override == nil {
		return base
	}
	if override.CallTimeout > 0 {
		base.CallTimeout = override.CallTimeout
	}
	if override.Retry >= 0 {
		base.Retry = override.Retry
	}
	if override.Backoff > 0 {
		base.Backoff = override.Backoff
	}
	if override.QueueSize > 0 {
		base.QueueSize = override.QueueSize
	}
	if override.Backpressure != "" {
		base.Backpressure = override.Backpressure
	}
	if override.ReadPoolSize > 0 {
		base.ReadPoolSize = override.ReadPoolSize
	}
	if override.WritePoolSize > 0 {
		base.WritePoolSize = override.WritePoolSize
	}
	return base
}

func defaultPolicyFor(profile string) types.MCPRuntimePolicy {
	switch profile {
	case ProfileDev:
		return types.MCPRuntimePolicy{
			CallTimeout:   5 * time.Second,
			Retry:         0,
			Backoff:       20 * time.Millisecond,
			QueueSize:     16,
			Backpressure:  types.BackpressureBlock,
			ReadPoolSize:  2,
			WritePoolSize: 1,
		}
	case ProfileHighThroughput:
		return types.MCPRuntimePolicy{
			CallTimeout:   8 * time.Second,
			Retry:         1,
			Backoff:       20 * time.Millisecond,
			QueueSize:     128,
			Backpressure:  types.BackpressureReject,
			ReadPoolSize:  16,
			WritePoolSize: 2,
		}
	case ProfileHighReliab:
		return types.MCPRuntimePolicy{
			CallTimeout:   15 * time.Second,
			Retry:         3,
			Backoff:       80 * time.Millisecond,
			QueueSize:     64,
			Backpressure:  types.BackpressureBlock,
			ReadPoolSize:  8,
			WritePoolSize: 1,
		}
	default:
		return types.DefaultMCPRuntimePolicy()
	}
}
