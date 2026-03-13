package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
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

const (
	SecurityScanModeStrict      = "strict"
	SecurityScanModeWarn        = "warn"
	SecurityRedactionKeyword    = "keyword"
	SecurityGovulncheckToolName = "govulncheck"
)

const (
	ContextStage2ProviderFile          = "file"
	ContextStage2ProviderHTTP          = "http"
	ContextStage2ProviderRAG           = "rag"
	ContextStage2ProviderDB            = "db"
	ContextStage2ProviderElasticsearch = "elasticsearch"
)

type Config struct {
	MCP              MCPConfig              `json:"mcp"`
	Concurrency      ConcurrencyConfig      `json:"concurrency"`
	Diagnostics      DiagnosticsConfig      `json:"diagnostics"`
	Reload           ReloadConfig           `json:"reload"`
	ProviderFallback ProviderFallbackConfig `json:"provider_fallback"`
	ContextAssembler ContextAssemblerConfig `json:"context_assembler"`
	Security         SecurityConfig         `json:"security"`
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

type ProviderFallbackConfig struct {
	Enabled           bool          `json:"enabled"`
	Providers         []string      `json:"providers"`
	DiscoveryTimeout  time.Duration `json:"discovery_timeout"`
	DiscoveryCacheTTL time.Duration `json:"discovery_cache_ttl"`
}

type ContextAssemblerConfig struct {
	Enabled       bool                          `json:"enabled"`
	JournalPath   string                        `json:"journal_path"`
	PrefixVersion string                        `json:"prefix_version"`
	Storage       ContextAssemblerStorageConfig `json:"storage"`
	Guard         ContextAssemblerGuardConfig   `json:"guard"`
	CA2           ContextAssemblerCA2Config     `json:"ca2"`
}

type ContextAssemblerStorageConfig struct {
	Backend string `json:"backend"`
}

type ContextAssemblerGuardConfig struct {
	FailFast bool `json:"fail_fast"`
}

type ContextAssemblerCA2Config struct {
	Enabled     bool                               `json:"enabled"`
	RoutingMode string                             `json:"routing_mode"`
	StagePolicy ContextAssemblerCA2StagePolicy     `json:"stage_policy"`
	Timeout     ContextAssemblerCA2TimeoutConfig   `json:"timeout"`
	Stage2      ContextAssemblerCA2Stage2Config    `json:"stage2"`
	Routing     ContextAssemblerCA2RoutingConfig   `json:"routing"`
	TailRecap   ContextAssemblerCA2TailRecapConfig `json:"tail_recap"`
}

type ContextAssemblerCA2StagePolicy struct {
	Stage1 string `json:"stage1"`
	Stage2 string `json:"stage2"`
}

type ContextAssemblerCA2TimeoutConfig struct {
	Stage1 time.Duration `json:"stage1"`
	Stage2 time.Duration `json:"stage2"`
}

type ContextAssemblerCA2Stage2Config struct {
	Provider string                            `json:"provider"`
	FilePath string                            `json:"file_path"`
	External ContextAssemblerCA2ExternalConfig `json:"external"`
}

type ContextAssemblerCA2ExternalConfig struct {
	Endpoint string                                   `json:"endpoint"`
	Method   string                                   `json:"method"`
	Auth     ContextAssemblerCA2ExternalAuthConfig    `json:"auth"`
	Headers  map[string]string                        `json:"headers"`
	Mapping  ContextAssemblerCA2ExternalMappingConfig `json:"mapping"`
}

type ContextAssemblerCA2ExternalAuthConfig struct {
	BearerToken string `json:"bearer_token"`
	HeaderName  string `json:"header_name"`
}

type ContextAssemblerCA2ExternalMappingConfig struct {
	Request  ContextAssemblerCA2RequestMappingConfig  `json:"request"`
	Response ContextAssemblerCA2ResponseMappingConfig `json:"response"`
}

type ContextAssemblerCA2RequestMappingConfig struct {
	Mode           string `json:"mode"`
	MethodName     string `json:"method_name"`
	JSONRPCVersion string `json:"jsonrpc_version"`
	QueryField     string `json:"query_field"`
	SessionIDField string `json:"session_id_field"`
	RunIDField     string `json:"run_id_field"`
	MaxItemsField  string `json:"max_items_field"`
}

type ContextAssemblerCA2ResponseMappingConfig struct {
	ChunksField       string `json:"chunks_field"`
	SourceField       string `json:"source_field"`
	ReasonField       string `json:"reason_field"`
	ErrorField        string `json:"error_field"`
	ErrorMessageField string `json:"error_message_field"`
}

type ContextAssemblerCA2RoutingConfig struct {
	MinInputChars      int      `json:"min_input_chars"`
	TriggerKeywords    []string `json:"trigger_keywords"`
	RequireSystemGuard bool     `json:"require_system_guard"`
}

type ContextAssemblerCA2TailRecapConfig struct {
	Enabled       bool `json:"enabled"`
	MaxItems      int  `json:"max_items"`
	MaxFieldChars int  `json:"max_field_chars"`
}

type SecurityConfig struct {
	Scan      SecurityScanConfig      `json:"scan"`
	Redaction SecurityRedactionConfig `json:"redaction"`
}

type SecurityScanConfig struct {
	Mode              string `json:"mode"`
	GovulncheckEnable bool   `json:"govulncheck_enabled"`
}

type SecurityRedactionConfig struct {
	Enabled  bool     `json:"enabled"`
	Strategy string   `json:"strategy"`
	Keywords []string `json:"keywords"`
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
		ProviderFallback: ProviderFallbackConfig{
			Enabled:           false,
			Providers:         nil,
			DiscoveryTimeout:  1500 * time.Millisecond,
			DiscoveryCacheTTL: 5 * time.Minute,
		},
		ContextAssembler: ContextAssemblerConfig{
			Enabled:       true,
			JournalPath:   filepath.Join(os.TempDir(), "baymax", "context-journal.jsonl"),
			PrefixVersion: "ca1",
			Storage: ContextAssemblerStorageConfig{
				Backend: "file",
			},
			Guard: ContextAssemblerGuardConfig{
				FailFast: true,
			},
			CA2: ContextAssemblerCA2Config{
				Enabled:     false,
				RoutingMode: "rules",
				StagePolicy: ContextAssemblerCA2StagePolicy{
					Stage1: "fail_fast",
					Stage2: "best_effort",
				},
				Timeout: ContextAssemblerCA2TimeoutConfig{
					Stage1: 80 * time.Millisecond,
					Stage2: 120 * time.Millisecond,
				},
				Stage2: ContextAssemblerCA2Stage2Config{
					Provider: "file",
					FilePath: filepath.Join(os.TempDir(), "baymax", "context-stage2.jsonl"),
					External: ContextAssemblerCA2ExternalConfig{
						Endpoint: "",
						Method:   "POST",
						Auth: ContextAssemblerCA2ExternalAuthConfig{
							BearerToken: "",
							HeaderName:  "Authorization",
						},
						Headers: map[string]string{},
						Mapping: ContextAssemblerCA2ExternalMappingConfig{
							Request: ContextAssemblerCA2RequestMappingConfig{
								Mode:           "plain",
								MethodName:     "",
								JSONRPCVersion: "2.0",
								QueryField:     "query",
								SessionIDField: "session_id",
								RunIDField:     "run_id",
								MaxItemsField:  "max_items",
							},
							Response: ContextAssemblerCA2ResponseMappingConfig{
								ChunksField:       "chunks",
								SourceField:       "source",
								ReasonField:       "reason",
								ErrorField:        "error",
								ErrorMessageField: "error.message",
							},
						},
					},
				},
				Routing: ContextAssemblerCA2RoutingConfig{
					MinInputChars:      120,
					TriggerKeywords:    []string{"search", "retrieve", "reference", "lookup", "资料", "检索"},
					RequireSystemGuard: true,
				},
				TailRecap: ContextAssemblerCA2TailRecapConfig{
					Enabled:       true,
					MaxItems:      4,
					MaxFieldChars: 256,
				},
			},
		},
		Security: SecurityConfig{
			Scan: SecurityScanConfig{
				Mode:              SecurityScanModeStrict,
				GovulncheckEnable: true,
			},
			Redaction: SecurityRedactionConfig{
				Enabled:  true,
				Strategy: SecurityRedactionKeyword,
				Keywords: []string{"token", "password", "secret", "api_key", "apikey"},
			},
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
	if cfg.ProviderFallback.DiscoveryTimeout <= 0 {
		return errors.New("provider_fallback.discovery_timeout must be > 0")
	}
	if cfg.ProviderFallback.DiscoveryCacheTTL <= 0 {
		return errors.New("provider_fallback.discovery_cache_ttl must be > 0")
	}
	if cfg.ProviderFallback.Enabled {
		if len(cfg.ProviderFallback.Providers) == 0 {
			return errors.New("provider_fallback.providers must not be empty when enabled")
		}
		seen := map[string]struct{}{}
		for i, provider := range cfg.ProviderFallback.Providers {
			name := strings.ToLower(strings.TrimSpace(provider))
			if name == "" {
				return fmt.Errorf("provider_fallback.providers[%d] must not be empty", i)
			}
			if _, ok := seen[name]; ok {
				return fmt.Errorf("provider_fallback.providers[%d]=%q is duplicated", i, name)
			}
			seen[name] = struct{}{}
			cfg.ProviderFallback.Providers[i] = name
		}
	}
	if cfg.ContextAssembler.Enabled {
		if strings.TrimSpace(cfg.ContextAssembler.JournalPath) == "" {
			return errors.New("context_assembler.journal_path is required when enabled")
		}
		if strings.TrimSpace(cfg.ContextAssembler.PrefixVersion) == "" {
			return errors.New("context_assembler.prefix_version is required when enabled")
		}
		backend := strings.ToLower(strings.TrimSpace(cfg.ContextAssembler.Storage.Backend))
		if backend == "" {
			backend = "file"
		}
		switch backend {
		case "file", "db":
		default:
			return fmt.Errorf("context_assembler.storage.backend must be one of [file,db], got %q", cfg.ContextAssembler.Storage.Backend)
		}
		cfg.ContextAssembler.Storage.Backend = backend
		if cfg.ContextAssembler.CA2.Enabled {
			mode := strings.ToLower(strings.TrimSpace(cfg.ContextAssembler.CA2.RoutingMode))
			switch mode {
			case "rules", "agentic":
			default:
				return fmt.Errorf("context_assembler.ca2.routing_mode must be one of [rules,agentic], got %q", cfg.ContextAssembler.CA2.RoutingMode)
			}
			cfg.ContextAssembler.CA2.RoutingMode = mode

			stage1Policy := strings.ToLower(strings.TrimSpace(cfg.ContextAssembler.CA2.StagePolicy.Stage1))
			if err := validateStagePolicy(stage1Policy, "context_assembler.ca2.stage_policy.stage1"); err != nil {
				return err
			}
			cfg.ContextAssembler.CA2.StagePolicy.Stage1 = stage1Policy
			stage2Policy := strings.ToLower(strings.TrimSpace(cfg.ContextAssembler.CA2.StagePolicy.Stage2))
			if err := validateStagePolicy(stage2Policy, "context_assembler.ca2.stage_policy.stage2"); err != nil {
				return err
			}
			cfg.ContextAssembler.CA2.StagePolicy.Stage2 = stage2Policy
			if cfg.ContextAssembler.CA2.Timeout.Stage1 <= 0 {
				return errors.New("context_assembler.ca2.timeout.stage1 must be > 0")
			}
			if cfg.ContextAssembler.CA2.Timeout.Stage2 <= 0 {
				return errors.New("context_assembler.ca2.timeout.stage2 must be > 0")
			}
			provider := strings.ToLower(strings.TrimSpace(cfg.ContextAssembler.CA2.Stage2.Provider))
			switch provider {
			case ContextStage2ProviderFile, ContextStage2ProviderHTTP, ContextStage2ProviderRAG, ContextStage2ProviderDB, ContextStage2ProviderElasticsearch:
			default:
				return fmt.Errorf("context_assembler.ca2.stage2.provider must be one of [file,http,rag,db,elasticsearch], got %q", cfg.ContextAssembler.CA2.Stage2.Provider)
			}
			cfg.ContextAssembler.CA2.Stage2.Provider = provider
			if provider == ContextStage2ProviderFile && strings.TrimSpace(cfg.ContextAssembler.CA2.Stage2.FilePath) == "" {
				return errors.New("context_assembler.ca2.stage2.file_path is required when provider=file")
			}
			if provider != ContextStage2ProviderFile {
				if err := validateStage2ExternalConfig(cfg.ContextAssembler.CA2.Stage2.External); err != nil {
					return err
				}
			}
			if cfg.ContextAssembler.CA2.Routing.MinInputChars < 0 {
				return errors.New("context_assembler.ca2.routing.min_input_chars must be >= 0")
			}
			if cfg.ContextAssembler.CA2.TailRecap.MaxItems <= 0 {
				return errors.New("context_assembler.ca2.tail_recap.max_items must be > 0")
			}
			if cfg.ContextAssembler.CA2.TailRecap.MaxFieldChars <= 0 {
				return errors.New("context_assembler.ca2.tail_recap.max_field_chars must be > 0")
			}
		}
	}
	scanMode := strings.ToLower(strings.TrimSpace(cfg.Security.Scan.Mode))
	switch scanMode {
	case SecurityScanModeStrict, SecurityScanModeWarn:
	default:
		return fmt.Errorf("security.scan.mode must be one of [strict,warn], got %q", cfg.Security.Scan.Mode)
	}
	redactionStrategy := strings.ToLower(strings.TrimSpace(cfg.Security.Redaction.Strategy))
	switch redactionStrategy {
	case SecurityRedactionKeyword:
	default:
		return fmt.Errorf("security.redaction.strategy must be one of [keyword], got %q", cfg.Security.Redaction.Strategy)
	}
	if cfg.Security.Redaction.Enabled && len(normalizeKeywords(cfg.Security.Redaction.Keywords)) == 0 {
		return errors.New("security.redaction.keywords must not be empty when security.redaction.enabled=true")
	}
	return nil
}

func validateStagePolicy(v, field string) error {
	switch v {
	case "fail_fast", "best_effort":
		return nil
	default:
		return fmt.Errorf("%s must be one of [fail_fast,best_effort]", field)
	}
}

func validateBackpressure(v types.BackpressureMode, field string) error {
	switch v {
	case types.BackpressureBlock, types.BackpressureReject:
		return nil
	default:
		return fmt.Errorf("%s must be one of [block,reject]", field)
	}
}

func validateStage2ExternalConfig(cfg ContextAssemblerCA2ExternalConfig) error {
	if strings.TrimSpace(cfg.Endpoint) == "" {
		return errors.New("context_assembler.ca2.stage2.external.endpoint is required for non-file providers")
	}
	method := strings.ToUpper(strings.TrimSpace(cfg.Method))
	switch method {
	case "", "POST", "PUT":
	default:
		return fmt.Errorf("context_assembler.ca2.stage2.external.method must be one of [POST,PUT], got %q", cfg.Method)
	}
	mode := strings.ToLower(strings.TrimSpace(cfg.Mapping.Request.Mode))
	switch mode {
	case "", "plain", "jsonrpc2":
	default:
		return fmt.Errorf("context_assembler.ca2.stage2.external.mapping.request.mode must be one of [plain,jsonrpc2], got %q", cfg.Mapping.Request.Mode)
	}
	if strings.TrimSpace(cfg.Mapping.Request.QueryField) == "" {
		return errors.New("context_assembler.ca2.stage2.external.mapping.request.query_field is required")
	}
	if strings.TrimSpace(cfg.Mapping.Response.ChunksField) == "" {
		return errors.New("context_assembler.ca2.stage2.external.mapping.response.chunks_field is required")
	}
	if mode == "jsonrpc2" && strings.TrimSpace(cfg.Mapping.Request.MethodName) == "" {
		return errors.New("context_assembler.ca2.stage2.external.mapping.request.method_name is required when mode=jsonrpc2")
	}
	return nil
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
	v.SetDefault("provider_fallback.enabled", base.ProviderFallback.Enabled)
	v.SetDefault("provider_fallback.providers", base.ProviderFallback.Providers)
	v.SetDefault("provider_fallback.discovery_timeout", base.ProviderFallback.DiscoveryTimeout)
	v.SetDefault("provider_fallback.discovery_cache_ttl", base.ProviderFallback.DiscoveryCacheTTL)
	v.SetDefault("context_assembler.enabled", base.ContextAssembler.Enabled)
	v.SetDefault("context_assembler.journal_path", base.ContextAssembler.JournalPath)
	v.SetDefault("context_assembler.prefix_version", base.ContextAssembler.PrefixVersion)
	v.SetDefault("context_assembler.storage.backend", base.ContextAssembler.Storage.Backend)
	v.SetDefault("context_assembler.guard.fail_fast", base.ContextAssembler.Guard.FailFast)
	v.SetDefault("context_assembler.ca2.enabled", base.ContextAssembler.CA2.Enabled)
	v.SetDefault("context_assembler.ca2.routing_mode", base.ContextAssembler.CA2.RoutingMode)
	v.SetDefault("context_assembler.ca2.stage_policy.stage1", base.ContextAssembler.CA2.StagePolicy.Stage1)
	v.SetDefault("context_assembler.ca2.stage_policy.stage2", base.ContextAssembler.CA2.StagePolicy.Stage2)
	v.SetDefault("context_assembler.ca2.timeout.stage1", base.ContextAssembler.CA2.Timeout.Stage1)
	v.SetDefault("context_assembler.ca2.timeout.stage2", base.ContextAssembler.CA2.Timeout.Stage2)
	v.SetDefault("context_assembler.ca2.stage2.provider", base.ContextAssembler.CA2.Stage2.Provider)
	v.SetDefault("context_assembler.ca2.stage2.file_path", base.ContextAssembler.CA2.Stage2.FilePath)
	v.SetDefault("context_assembler.ca2.stage2.external.endpoint", base.ContextAssembler.CA2.Stage2.External.Endpoint)
	v.SetDefault("context_assembler.ca2.stage2.external.method", base.ContextAssembler.CA2.Stage2.External.Method)
	v.SetDefault("context_assembler.ca2.stage2.external.auth.bearer_token", base.ContextAssembler.CA2.Stage2.External.Auth.BearerToken)
	v.SetDefault("context_assembler.ca2.stage2.external.auth.header_name", base.ContextAssembler.CA2.Stage2.External.Auth.HeaderName)
	v.SetDefault("context_assembler.ca2.stage2.external.headers", base.ContextAssembler.CA2.Stage2.External.Headers)
	v.SetDefault("context_assembler.ca2.stage2.external.mapping.request.mode", base.ContextAssembler.CA2.Stage2.External.Mapping.Request.Mode)
	v.SetDefault("context_assembler.ca2.stage2.external.mapping.request.method_name", base.ContextAssembler.CA2.Stage2.External.Mapping.Request.MethodName)
	v.SetDefault("context_assembler.ca2.stage2.external.mapping.request.jsonrpc_version", base.ContextAssembler.CA2.Stage2.External.Mapping.Request.JSONRPCVersion)
	v.SetDefault("context_assembler.ca2.stage2.external.mapping.request.query_field", base.ContextAssembler.CA2.Stage2.External.Mapping.Request.QueryField)
	v.SetDefault("context_assembler.ca2.stage2.external.mapping.request.session_id_field", base.ContextAssembler.CA2.Stage2.External.Mapping.Request.SessionIDField)
	v.SetDefault("context_assembler.ca2.stage2.external.mapping.request.run_id_field", base.ContextAssembler.CA2.Stage2.External.Mapping.Request.RunIDField)
	v.SetDefault("context_assembler.ca2.stage2.external.mapping.request.max_items_field", base.ContextAssembler.CA2.Stage2.External.Mapping.Request.MaxItemsField)
	v.SetDefault("context_assembler.ca2.stage2.external.mapping.response.chunks_field", base.ContextAssembler.CA2.Stage2.External.Mapping.Response.ChunksField)
	v.SetDefault("context_assembler.ca2.stage2.external.mapping.response.source_field", base.ContextAssembler.CA2.Stage2.External.Mapping.Response.SourceField)
	v.SetDefault("context_assembler.ca2.stage2.external.mapping.response.reason_field", base.ContextAssembler.CA2.Stage2.External.Mapping.Response.ReasonField)
	v.SetDefault("context_assembler.ca2.stage2.external.mapping.response.error_field", base.ContextAssembler.CA2.Stage2.External.Mapping.Response.ErrorField)
	v.SetDefault("context_assembler.ca2.stage2.external.mapping.response.error_message_field", base.ContextAssembler.CA2.Stage2.External.Mapping.Response.ErrorMessageField)
	v.SetDefault("context_assembler.ca2.routing.min_input_chars", base.ContextAssembler.CA2.Routing.MinInputChars)
	v.SetDefault("context_assembler.ca2.routing.trigger_keywords", base.ContextAssembler.CA2.Routing.TriggerKeywords)
	v.SetDefault("context_assembler.ca2.routing.require_system_guard", base.ContextAssembler.CA2.Routing.RequireSystemGuard)
	v.SetDefault("context_assembler.ca2.tail_recap.enabled", base.ContextAssembler.CA2.TailRecap.Enabled)
	v.SetDefault("context_assembler.ca2.tail_recap.max_items", base.ContextAssembler.CA2.TailRecap.MaxItems)
	v.SetDefault("context_assembler.ca2.tail_recap.max_field_chars", base.ContextAssembler.CA2.TailRecap.MaxFieldChars)
	v.SetDefault("security.scan.mode", base.Security.Scan.Mode)
	v.SetDefault("security.scan.govulncheck_enabled", base.Security.Scan.GovulncheckEnable)
	v.SetDefault("security.redaction.enabled", base.Security.Redaction.Enabled)
	v.SetDefault("security.redaction.strategy", base.Security.Redaction.Strategy)
	v.SetDefault("security.redaction.keywords", base.Security.Redaction.Keywords)
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
	cfg.ProviderFallback.Enabled = v.GetBool("provider_fallback.enabled")
	cfg.ProviderFallback.Providers = normalizeProviders(v.GetStringSlice("provider_fallback.providers"))
	cfg.ProviderFallback.DiscoveryTimeout = v.GetDuration("provider_fallback.discovery_timeout")
	cfg.ProviderFallback.DiscoveryCacheTTL = v.GetDuration("provider_fallback.discovery_cache_ttl")
	cfg.ContextAssembler.Enabled = v.GetBool("context_assembler.enabled")
	cfg.ContextAssembler.JournalPath = strings.TrimSpace(v.GetString("context_assembler.journal_path"))
	cfg.ContextAssembler.PrefixVersion = strings.TrimSpace(v.GetString("context_assembler.prefix_version"))
	cfg.ContextAssembler.Storage.Backend = strings.ToLower(strings.TrimSpace(v.GetString("context_assembler.storage.backend")))
	cfg.ContextAssembler.Guard.FailFast = v.GetBool("context_assembler.guard.fail_fast")
	cfg.ContextAssembler.CA2.Enabled = v.GetBool("context_assembler.ca2.enabled")
	cfg.ContextAssembler.CA2.RoutingMode = strings.ToLower(strings.TrimSpace(v.GetString("context_assembler.ca2.routing_mode")))
	cfg.ContextAssembler.CA2.StagePolicy.Stage1 = strings.ToLower(strings.TrimSpace(v.GetString("context_assembler.ca2.stage_policy.stage1")))
	cfg.ContextAssembler.CA2.StagePolicy.Stage2 = strings.ToLower(strings.TrimSpace(v.GetString("context_assembler.ca2.stage_policy.stage2")))
	cfg.ContextAssembler.CA2.Timeout.Stage1 = v.GetDuration("context_assembler.ca2.timeout.stage1")
	cfg.ContextAssembler.CA2.Timeout.Stage2 = v.GetDuration("context_assembler.ca2.timeout.stage2")
	cfg.ContextAssembler.CA2.Stage2.Provider = strings.ToLower(strings.TrimSpace(v.GetString("context_assembler.ca2.stage2.provider")))
	cfg.ContextAssembler.CA2.Stage2.FilePath = strings.TrimSpace(v.GetString("context_assembler.ca2.stage2.file_path"))
	cfg.ContextAssembler.CA2.Stage2.External.Endpoint = strings.TrimSpace(v.GetString("context_assembler.ca2.stage2.external.endpoint"))
	cfg.ContextAssembler.CA2.Stage2.External.Method = strings.ToUpper(strings.TrimSpace(v.GetString("context_assembler.ca2.stage2.external.method")))
	cfg.ContextAssembler.CA2.Stage2.External.Auth.BearerToken = strings.TrimSpace(v.GetString("context_assembler.ca2.stage2.external.auth.bearer_token"))
	cfg.ContextAssembler.CA2.Stage2.External.Auth.HeaderName = strings.TrimSpace(v.GetString("context_assembler.ca2.stage2.external.auth.header_name"))
	cfg.ContextAssembler.CA2.Stage2.External.Headers = normalizeStringMap(v.GetStringMapString("context_assembler.ca2.stage2.external.headers"))
	cfg.ContextAssembler.CA2.Stage2.External.Mapping.Request.Mode = strings.ToLower(strings.TrimSpace(v.GetString("context_assembler.ca2.stage2.external.mapping.request.mode")))
	cfg.ContextAssembler.CA2.Stage2.External.Mapping.Request.MethodName = strings.TrimSpace(v.GetString("context_assembler.ca2.stage2.external.mapping.request.method_name"))
	cfg.ContextAssembler.CA2.Stage2.External.Mapping.Request.JSONRPCVersion = strings.TrimSpace(v.GetString("context_assembler.ca2.stage2.external.mapping.request.jsonrpc_version"))
	cfg.ContextAssembler.CA2.Stage2.External.Mapping.Request.QueryField = strings.TrimSpace(v.GetString("context_assembler.ca2.stage2.external.mapping.request.query_field"))
	cfg.ContextAssembler.CA2.Stage2.External.Mapping.Request.SessionIDField = strings.TrimSpace(v.GetString("context_assembler.ca2.stage2.external.mapping.request.session_id_field"))
	cfg.ContextAssembler.CA2.Stage2.External.Mapping.Request.RunIDField = strings.TrimSpace(v.GetString("context_assembler.ca2.stage2.external.mapping.request.run_id_field"))
	cfg.ContextAssembler.CA2.Stage2.External.Mapping.Request.MaxItemsField = strings.TrimSpace(v.GetString("context_assembler.ca2.stage2.external.mapping.request.max_items_field"))
	cfg.ContextAssembler.CA2.Stage2.External.Mapping.Response.ChunksField = strings.TrimSpace(v.GetString("context_assembler.ca2.stage2.external.mapping.response.chunks_field"))
	cfg.ContextAssembler.CA2.Stage2.External.Mapping.Response.SourceField = strings.TrimSpace(v.GetString("context_assembler.ca2.stage2.external.mapping.response.source_field"))
	cfg.ContextAssembler.CA2.Stage2.External.Mapping.Response.ReasonField = strings.TrimSpace(v.GetString("context_assembler.ca2.stage2.external.mapping.response.reason_field"))
	cfg.ContextAssembler.CA2.Stage2.External.Mapping.Response.ErrorField = strings.TrimSpace(v.GetString("context_assembler.ca2.stage2.external.mapping.response.error_field"))
	cfg.ContextAssembler.CA2.Stage2.External.Mapping.Response.ErrorMessageField = strings.TrimSpace(v.GetString("context_assembler.ca2.stage2.external.mapping.response.error_message_field"))
	cfg.ContextAssembler.CA2.Routing.MinInputChars = v.GetInt("context_assembler.ca2.routing.min_input_chars")
	cfg.ContextAssembler.CA2.Routing.TriggerKeywords = normalizeKeywords(v.GetStringSlice("context_assembler.ca2.routing.trigger_keywords"))
	cfg.ContextAssembler.CA2.Routing.RequireSystemGuard = v.GetBool("context_assembler.ca2.routing.require_system_guard")
	cfg.ContextAssembler.CA2.TailRecap.Enabled = v.GetBool("context_assembler.ca2.tail_recap.enabled")
	cfg.ContextAssembler.CA2.TailRecap.MaxItems = v.GetInt("context_assembler.ca2.tail_recap.max_items")
	cfg.ContextAssembler.CA2.TailRecap.MaxFieldChars = v.GetInt("context_assembler.ca2.tail_recap.max_field_chars")
	cfg.Security.Scan.Mode = strings.ToLower(strings.TrimSpace(v.GetString("security.scan.mode")))
	cfg.Security.Scan.GovulncheckEnable = v.GetBool("security.scan.govulncheck_enabled")
	cfg.Security.Redaction.Enabled = v.GetBool("security.redaction.enabled")
	cfg.Security.Redaction.Strategy = strings.ToLower(strings.TrimSpace(v.GetString("security.redaction.strategy")))
	cfg.Security.Redaction.Keywords = normalizeKeywords(v.GetStringSlice("security.redaction.keywords"))

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

func normalizeProviders(in []string) []string {
	if len(in) == 0 {
		return nil
	}
	out := make([]string, 0, len(in))
	seen := map[string]struct{}{}
	for _, provider := range in {
		chunks := strings.Split(provider, ",")
		for _, chunk := range chunks {
			name := strings.ToLower(strings.TrimSpace(chunk))
			if name == "" {
				continue
			}
			if _, ok := seen[name]; ok {
				continue
			}
			seen[name] = struct{}{}
			out = append(out, name)
		}
	}
	return out
}

func normalizeKeywords(in []string) []string {
	if len(in) == 0 {
		return nil
	}
	out := make([]string, 0, len(in))
	seen := map[string]struct{}{}
	for _, v := range in {
		chunks := strings.Split(v, ",")
		for _, chunk := range chunks {
			item := strings.ToLower(strings.TrimSpace(chunk))
			if item == "" {
				continue
			}
			if _, ok := seen[item]; ok {
				continue
			}
			seen[item] = struct{}{}
			out = append(out, item)
		}
	}
	return out
}

func normalizeStringMap(in map[string]string) map[string]string {
	if len(in) == 0 {
		return map[string]string{}
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		key := strings.TrimSpace(k)
		if key == "" {
			continue
		}
		out[key] = strings.TrimSpace(v)
	}
	return out
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
