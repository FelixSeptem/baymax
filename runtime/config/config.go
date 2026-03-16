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

const (
	ContextStage2ExternalProfileHTTPGeneric       = "http_generic"
	ContextStage2ExternalProfileRAGFlowLike       = "ragflow_like"
	ContextStage2ExternalProfileGraphRAGLike      = "graphrag_like"
	ContextStage2ExternalProfileElasticsearchLike = "elasticsearch_like"
)

const (
	ActionGatePolicyAllow          = "allow"
	ActionGatePolicyRequireConfirm = "require_confirm"
	ActionGatePolicyDeny           = "deny"
)

const (
	ClarificationTimeoutPolicyCancelByUser = "cancel_by_user"
)

type Config struct {
	MCP              MCPConfig              `json:"mcp"`
	Concurrency      ConcurrencyConfig      `json:"concurrency"`
	Diagnostics      DiagnosticsConfig      `json:"diagnostics"`
	Reload           ReloadConfig           `json:"reload"`
	ProviderFallback ProviderFallbackConfig `json:"provider_fallback"`
	ActionGate       ActionGateConfig       `json:"action_gate"`
	Clarification    ClarificationConfig    `json:"clarification"`
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

type ActionGateConfig struct {
	Enabled        bool              `json:"enabled"`
	Policy         string            `json:"policy"`
	Timeout        time.Duration     `json:"timeout"`
	ToolNames      []string          `json:"tool_names"`
	Keywords       []string          `json:"keywords"`
	DecisionByTool map[string]string `json:"decision_by_tool"`
	DecisionByWord map[string]string `json:"decision_by_keyword"`
}

type ClarificationConfig struct {
	Enabled       bool          `json:"enabled"`
	Timeout       time.Duration `json:"timeout"`
	TimeoutPolicy string        `json:"timeout_policy"`
}

type ContextAssemblerConfig struct {
	Enabled       bool                          `json:"enabled"`
	JournalPath   string                        `json:"journal_path"`
	PrefixVersion string                        `json:"prefix_version"`
	Storage       ContextAssemblerStorageConfig `json:"storage"`
	Guard         ContextAssemblerGuardConfig   `json:"guard"`
	CA2           ContextAssemblerCA2Config     `json:"ca2"`
	CA3           ContextAssemblerCA3Config     `json:"ca3"`
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
	Profile  string                                   `json:"profile"`
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

type ContextAssemblerCA3Config struct {
	Enabled              bool                                       `json:"enabled"`
	MaxContextTokens     int                                        `json:"max_context_tokens"`
	GoldilocksMinPercent int                                        `json:"goldilocks_min_percent"`
	GoldilocksMaxPercent int                                        `json:"goldilocks_max_percent"`
	PercentThresholds    ContextAssemblerCA3Thresholds              `json:"percent_thresholds"`
	AbsoluteThresholds   ContextAssemblerCA3Thresholds              `json:"absolute_thresholds"`
	Stage1               ContextAssemblerCA3StageThresholdOverrides `json:"stage1"`
	Stage2               ContextAssemblerCA3StageThresholdOverrides `json:"stage2"`
	Protection           ContextAssemblerCA3ProtectionConfig        `json:"protection"`
	Squash               ContextAssemblerCA3SquashConfig            `json:"squash"`
	Prune                ContextAssemblerCA3PruneConfig             `json:"prune"`
	Emergency            ContextAssemblerCA3EmergencyConfig         `json:"emergency"`
	Spill                ContextAssemblerCA3SpillConfig             `json:"spill"`
	Tokenizer            ContextAssemblerCA3TokenizerConfig         `json:"tokenizer"`
}

type ContextAssemblerCA3Thresholds struct {
	Safe      int `json:"safe"`
	Comfort   int `json:"comfort"`
	Warning   int `json:"warning"`
	Danger    int `json:"danger"`
	Emergency int `json:"emergency"`
}

type ContextAssemblerCA3StageThresholdOverrides struct {
	PercentThresholds  ContextAssemblerCA3Thresholds `json:"percent_thresholds"`
	AbsoluteThresholds ContextAssemblerCA3Thresholds `json:"absolute_thresholds"`
}

type ContextAssemblerCA3ProtectionConfig struct {
	CriticalKeywords  []string `json:"critical_keywords"`
	ImmutableKeywords []string `json:"immutable_keywords"`
}

type ContextAssemblerCA3SquashConfig struct {
	Enabled         bool `json:"enabled"`
	MaxContentRunes int  `json:"max_content_runes"`
}

type ContextAssemblerCA3PruneConfig struct {
	Enabled         bool     `json:"enabled"`
	TargetPercent   int      `json:"target_percent"`
	KeywordPriority []string `json:"keyword_priority"`
}

type ContextAssemblerCA3EmergencyConfig struct {
	RejectLowPriority  bool     `json:"reject_low_priority"`
	HighPriorityTokens []string `json:"high_priority_tokens"`
}

type ContextAssemblerCA3SpillConfig struct {
	Enabled       bool   `json:"enabled"`
	Backend       string `json:"backend"`
	Path          string `json:"path"`
	SwapBackLimit int    `json:"swap_back_limit"`
}

type ContextAssemblerCA3TokenizerConfig struct {
	Mode               string        `json:"mode"`
	Provider           string        `json:"provider"`
	Model              string        `json:"model"`
	SmallDeltaTokens   int           `json:"small_delta_tokens"`
	SDKRefreshInterval time.Duration `json:"sdk_refresh_interval"`
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
		ActionGate: ActionGateConfig{
			Enabled:        true,
			Policy:         ActionGatePolicyRequireConfirm,
			Timeout:        15 * time.Second,
			ToolNames:      nil,
			Keywords:       nil,
			DecisionByTool: map[string]string{},
			DecisionByWord: map[string]string{},
		},
		Clarification: ClarificationConfig{
			Enabled:       true,
			Timeout:       30 * time.Second,
			TimeoutPolicy: ClarificationTimeoutPolicyCancelByUser,
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
						Profile:  ContextStage2ExternalProfileHTTPGeneric,
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
			CA3: ContextAssemblerCA3Config{
				Enabled:              true,
				MaxContextTokens:     128000,
				GoldilocksMinPercent: 35,
				GoldilocksMaxPercent: 60,
				PercentThresholds: ContextAssemblerCA3Thresholds{
					Safe:      20,
					Comfort:   40,
					Warning:   60,
					Danger:    75,
					Emergency: 90,
				},
				AbsoluteThresholds: ContextAssemblerCA3Thresholds{
					Safe:      24000,
					Comfort:   48000,
					Warning:   72000,
					Danger:    96000,
					Emergency: 115200,
				},
				Stage1: ContextAssemblerCA3StageThresholdOverrides{},
				Stage2: ContextAssemblerCA3StageThresholdOverrides{},
				Protection: ContextAssemblerCA3ProtectionConfig{
					CriticalKeywords:  []string{"critical"},
					ImmutableKeywords: []string{"immutable"},
				},
				Squash: ContextAssemblerCA3SquashConfig{
					Enabled:         true,
					MaxContentRunes: 320,
				},
				Prune: ContextAssemblerCA3PruneConfig{
					Enabled:         true,
					TargetPercent:   55,
					KeywordPriority: []string{"error", "decision", "constraint", "risk", "todo"},
				},
				Emergency: ContextAssemblerCA3EmergencyConfig{
					RejectLowPriority:  true,
					HighPriorityTokens: []string{"urgent", "critical", "incident"},
				},
				Spill: ContextAssemblerCA3SpillConfig{
					Enabled:       true,
					Backend:       "file",
					Path:          filepath.Join(os.TempDir(), "baymax", "context-spill.jsonl"),
					SwapBackLimit: 4,
				},
				Tokenizer: ContextAssemblerCA3TokenizerConfig{
					Mode:               "sdk_preferred",
					Provider:           "openai",
					Model:              "gpt-5.4",
					SmallDeltaTokens:   1024 * 8,
					SDKRefreshInterval: 5 * time.Second,
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
	if cfg.ActionGate.Enabled {
		policy := strings.ToLower(strings.TrimSpace(cfg.ActionGate.Policy))
		if err := validateActionGatePolicy(policy, "action_gate.policy"); err != nil {
			return err
		}
		if cfg.ActionGate.Timeout <= 0 {
			return errors.New("action_gate.timeout must be > 0")
		}
		for i, name := range cfg.ActionGate.ToolNames {
			if strings.TrimSpace(name) == "" {
				return fmt.Errorf("action_gate.tool_names[%d] must not be empty", i)
			}
		}
		for i, keyword := range cfg.ActionGate.Keywords {
			if strings.TrimSpace(keyword) == "" {
				return fmt.Errorf("action_gate.keywords[%d] must not be empty", i)
			}
		}
		for tool, decision := range cfg.ActionGate.DecisionByTool {
			if strings.TrimSpace(tool) == "" {
				return errors.New("action_gate.decision_by_tool contains empty key")
			}
			if err := validateActionGatePolicy(decision, fmt.Sprintf("action_gate.decision_by_tool.%s", tool)); err != nil {
				return err
			}
		}
		for keyword, decision := range cfg.ActionGate.DecisionByWord {
			if strings.TrimSpace(keyword) == "" {
				return errors.New("action_gate.decision_by_keyword contains empty key")
			}
			if err := validateActionGatePolicy(decision, fmt.Sprintf("action_gate.decision_by_keyword.%s", keyword)); err != nil {
				return err
			}
		}
	}
	if cfg.Clarification.Enabled {
		if cfg.Clarification.Timeout <= 0 {
			return errors.New("clarification.timeout must be > 0")
		}
		policy := strings.ToLower(strings.TrimSpace(cfg.Clarification.TimeoutPolicy))
		if policy != ClarificationTimeoutPolicyCancelByUser {
			return fmt.Errorf("clarification.timeout_policy must be %q, got %q", ClarificationTimeoutPolicyCancelByUser, cfg.Clarification.TimeoutPolicy)
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
				precheck := PrecheckStage2External(provider, cfg.ContextAssembler.CA2.Stage2.External)
				if err := precheck.FirstError(); err != nil {
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
		if cfg.ContextAssembler.CA3.Enabled {
			if err := validateCA3Config(cfg.ContextAssembler.CA3); err != nil {
				return err
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

func validateActionGatePolicy(v, field string) error {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case ActionGatePolicyAllow, ActionGatePolicyRequireConfirm, ActionGatePolicyDeny:
		return nil
	default:
		return fmt.Errorf("%s must be one of [allow,require_confirm,deny]", field)
	}
}

func validateCA3Config(cfg ContextAssemblerCA3Config) error {
	if cfg.MaxContextTokens <= 0 {
		return errors.New("context_assembler.ca3.max_context_tokens must be > 0")
	}
	if cfg.GoldilocksMinPercent < 0 || cfg.GoldilocksMinPercent > 100 {
		return errors.New("context_assembler.ca3.goldilocks_min_percent must be in [0,100]")
	}
	if cfg.GoldilocksMaxPercent < 0 || cfg.GoldilocksMaxPercent > 100 {
		return errors.New("context_assembler.ca3.goldilocks_max_percent must be in [0,100]")
	}
	if cfg.GoldilocksMinPercent >= cfg.GoldilocksMaxPercent {
		return errors.New("context_assembler.ca3.goldilocks_min_percent must be < goldilocks_max_percent")
	}
	if err := validateCA3Thresholds("context_assembler.ca3.percent_thresholds", cfg.PercentThresholds, 0, 100); err != nil {
		return err
	}
	if err := validateCA3Thresholds("context_assembler.ca3.absolute_thresholds", cfg.AbsoluteThresholds, 0, cfg.MaxContextTokens); err != nil {
		return err
	}
	if err := validateCA3StageOverride("context_assembler.ca3.stage1.percent_thresholds", cfg.Stage1.PercentThresholds, 0, 100); err != nil {
		return err
	}
	if err := validateCA3StageOverride("context_assembler.ca3.stage1.absolute_thresholds", cfg.Stage1.AbsoluteThresholds, 0, cfg.MaxContextTokens); err != nil {
		return err
	}
	if err := validateCA3StageOverride("context_assembler.ca3.stage2.percent_thresholds", cfg.Stage2.PercentThresholds, 0, 100); err != nil {
		return err
	}
	if err := validateCA3StageOverride("context_assembler.ca3.stage2.absolute_thresholds", cfg.Stage2.AbsoluteThresholds, 0, cfg.MaxContextTokens); err != nil {
		return err
	}
	if cfg.Squash.Enabled && cfg.Squash.MaxContentRunes <= 0 {
		return errors.New("context_assembler.ca3.squash.max_content_runes must be > 0")
	}
	if cfg.Prune.Enabled {
		if cfg.Prune.TargetPercent < 0 || cfg.Prune.TargetPercent > 100 {
			return errors.New("context_assembler.ca3.prune.target_percent must be in [0,100]")
		}
		if cfg.Prune.TargetPercent > cfg.GoldilocksMaxPercent {
			return errors.New("context_assembler.ca3.prune.target_percent must be <= goldilocks_max_percent")
		}
	}
	if cfg.Spill.Enabled {
		backend := strings.ToLower(strings.TrimSpace(cfg.Spill.Backend))
		if backend == "" {
			backend = "file"
		}
		switch backend {
		case "file":
		case "db", "object":
			// Placeholder only. Keep accepted for forward-compatible configs.
		default:
			return fmt.Errorf("context_assembler.ca3.spill.backend must be one of [file,db,object], got %q", cfg.Spill.Backend)
		}
		if backend == "file" && strings.TrimSpace(cfg.Spill.Path) == "" {
			return errors.New("context_assembler.ca3.spill.path is required when spill.backend=file")
		}
		if cfg.Spill.SwapBackLimit <= 0 {
			return errors.New("context_assembler.ca3.spill.swap_back_limit must be > 0")
		}
	}
	mode := strings.ToLower(strings.TrimSpace(cfg.Tokenizer.Mode))
	if mode == "" {
		mode = "sdk_preferred"
	}
	switch mode {
	case "estimate_only", "sdk_preferred":
	default:
		return fmt.Errorf("context_assembler.ca3.tokenizer.mode must be one of [estimate_only,sdk_preferred], got %q", cfg.Tokenizer.Mode)
	}
	provider := strings.ToLower(strings.TrimSpace(cfg.Tokenizer.Provider))
	if provider == "" {
		provider = "anthropic"
	}
	switch provider {
	case "anthropic", "gemini", "openai":
	default:
		return fmt.Errorf("context_assembler.ca3.tokenizer.provider must be one of [anthropic,gemini,openai], got %q", cfg.Tokenizer.Provider)
	}
	if cfg.Tokenizer.SmallDeltaTokens < 0 {
		return errors.New("context_assembler.ca3.tokenizer.small_delta_tokens must be >= 0")
	}
	if cfg.Tokenizer.SDKRefreshInterval <= 0 {
		return errors.New("context_assembler.ca3.tokenizer.sdk_refresh_interval must be > 0")
	}
	return nil
}

func validateCA3Thresholds(field string, thresholds ContextAssemblerCA3Thresholds, min, max int) error {
	values := []struct {
		name  string
		value int
	}{
		{name: "safe", value: thresholds.Safe},
		{name: "comfort", value: thresholds.Comfort},
		{name: "warning", value: thresholds.Warning},
		{name: "danger", value: thresholds.Danger},
		{name: "emergency", value: thresholds.Emergency},
	}
	prev := min - 1
	for _, item := range values {
		if item.value < min || item.value > max {
			return fmt.Errorf("%s.%s must be in [%d,%d]", field, item.name, min, max)
		}
		if item.value <= prev {
			return fmt.Errorf("%s must be strictly increasing", field)
		}
		prev = item.value
	}
	return nil
}

func validateCA3StageOverride(field string, thresholds ContextAssemblerCA3Thresholds, min, max int) error {
	if thresholds.Safe == 0 && thresholds.Comfort == 0 && thresholds.Warning == 0 && thresholds.Danger == 0 && thresholds.Emergency == 0 {
		return nil
	}
	return validateCA3Thresholds(field, thresholds, min, max)
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
	v.SetDefault("action_gate.enabled", base.ActionGate.Enabled)
	v.SetDefault("action_gate.policy", base.ActionGate.Policy)
	v.SetDefault("action_gate.timeout", base.ActionGate.Timeout)
	v.SetDefault("action_gate.tool_names", base.ActionGate.ToolNames)
	v.SetDefault("action_gate.keywords", base.ActionGate.Keywords)
	v.SetDefault("action_gate.decision_by_tool", base.ActionGate.DecisionByTool)
	v.SetDefault("action_gate.decision_by_keyword", base.ActionGate.DecisionByWord)
	v.SetDefault("clarification.enabled", base.Clarification.Enabled)
	v.SetDefault("clarification.timeout", base.Clarification.Timeout)
	v.SetDefault("clarification.timeout_policy", base.Clarification.TimeoutPolicy)
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
	v.SetDefault("context_assembler.ca2.stage2.external.profile", base.ContextAssembler.CA2.Stage2.External.Profile)
	v.SetDefault("context_assembler.ca2.routing.min_input_chars", base.ContextAssembler.CA2.Routing.MinInputChars)
	v.SetDefault("context_assembler.ca2.routing.trigger_keywords", base.ContextAssembler.CA2.Routing.TriggerKeywords)
	v.SetDefault("context_assembler.ca2.routing.require_system_guard", base.ContextAssembler.CA2.Routing.RequireSystemGuard)
	v.SetDefault("context_assembler.ca2.tail_recap.enabled", base.ContextAssembler.CA2.TailRecap.Enabled)
	v.SetDefault("context_assembler.ca2.tail_recap.max_items", base.ContextAssembler.CA2.TailRecap.MaxItems)
	v.SetDefault("context_assembler.ca2.tail_recap.max_field_chars", base.ContextAssembler.CA2.TailRecap.MaxFieldChars)
	v.SetDefault("context_assembler.ca3.enabled", base.ContextAssembler.CA3.Enabled)
	v.SetDefault("context_assembler.ca3.max_context_tokens", base.ContextAssembler.CA3.MaxContextTokens)
	v.SetDefault("context_assembler.ca3.goldilocks_min_percent", base.ContextAssembler.CA3.GoldilocksMinPercent)
	v.SetDefault("context_assembler.ca3.goldilocks_max_percent", base.ContextAssembler.CA3.GoldilocksMaxPercent)
	v.SetDefault("context_assembler.ca3.percent_thresholds.safe", base.ContextAssembler.CA3.PercentThresholds.Safe)
	v.SetDefault("context_assembler.ca3.percent_thresholds.comfort", base.ContextAssembler.CA3.PercentThresholds.Comfort)
	v.SetDefault("context_assembler.ca3.percent_thresholds.warning", base.ContextAssembler.CA3.PercentThresholds.Warning)
	v.SetDefault("context_assembler.ca3.percent_thresholds.danger", base.ContextAssembler.CA3.PercentThresholds.Danger)
	v.SetDefault("context_assembler.ca3.percent_thresholds.emergency", base.ContextAssembler.CA3.PercentThresholds.Emergency)
	v.SetDefault("context_assembler.ca3.absolute_thresholds.safe", base.ContextAssembler.CA3.AbsoluteThresholds.Safe)
	v.SetDefault("context_assembler.ca3.absolute_thresholds.comfort", base.ContextAssembler.CA3.AbsoluteThresholds.Comfort)
	v.SetDefault("context_assembler.ca3.absolute_thresholds.warning", base.ContextAssembler.CA3.AbsoluteThresholds.Warning)
	v.SetDefault("context_assembler.ca3.absolute_thresholds.danger", base.ContextAssembler.CA3.AbsoluteThresholds.Danger)
	v.SetDefault("context_assembler.ca3.absolute_thresholds.emergency", base.ContextAssembler.CA3.AbsoluteThresholds.Emergency)
	v.SetDefault("context_assembler.ca3.protection.critical_keywords", base.ContextAssembler.CA3.Protection.CriticalKeywords)
	v.SetDefault("context_assembler.ca3.protection.immutable_keywords", base.ContextAssembler.CA3.Protection.ImmutableKeywords)
	v.SetDefault("context_assembler.ca3.squash.enabled", base.ContextAssembler.CA3.Squash.Enabled)
	v.SetDefault("context_assembler.ca3.squash.max_content_runes", base.ContextAssembler.CA3.Squash.MaxContentRunes)
	v.SetDefault("context_assembler.ca3.prune.enabled", base.ContextAssembler.CA3.Prune.Enabled)
	v.SetDefault("context_assembler.ca3.prune.target_percent", base.ContextAssembler.CA3.Prune.TargetPercent)
	v.SetDefault("context_assembler.ca3.prune.keyword_priority", base.ContextAssembler.CA3.Prune.KeywordPriority)
	v.SetDefault("context_assembler.ca3.emergency.reject_low_priority", base.ContextAssembler.CA3.Emergency.RejectLowPriority)
	v.SetDefault("context_assembler.ca3.emergency.high_priority_tokens", base.ContextAssembler.CA3.Emergency.HighPriorityTokens)
	v.SetDefault("context_assembler.ca3.spill.enabled", base.ContextAssembler.CA3.Spill.Enabled)
	v.SetDefault("context_assembler.ca3.spill.backend", base.ContextAssembler.CA3.Spill.Backend)
	v.SetDefault("context_assembler.ca3.spill.path", base.ContextAssembler.CA3.Spill.Path)
	v.SetDefault("context_assembler.ca3.spill.swap_back_limit", base.ContextAssembler.CA3.Spill.SwapBackLimit)
	v.SetDefault("context_assembler.ca3.tokenizer.mode", base.ContextAssembler.CA3.Tokenizer.Mode)
	v.SetDefault("context_assembler.ca3.tokenizer.provider", base.ContextAssembler.CA3.Tokenizer.Provider)
	v.SetDefault("context_assembler.ca3.tokenizer.model", base.ContextAssembler.CA3.Tokenizer.Model)
	v.SetDefault("context_assembler.ca3.tokenizer.small_delta_tokens", base.ContextAssembler.CA3.Tokenizer.SmallDeltaTokens)
	v.SetDefault("context_assembler.ca3.tokenizer.sdk_refresh_interval", base.ContextAssembler.CA3.Tokenizer.SDKRefreshInterval)
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
	cfg.ActionGate.Enabled = v.GetBool("action_gate.enabled")
	cfg.ActionGate.Policy = strings.ToLower(strings.TrimSpace(v.GetString("action_gate.policy")))
	cfg.ActionGate.Timeout = v.GetDuration("action_gate.timeout")
	cfg.ActionGate.ToolNames = normalizeKeywords(v.GetStringSlice("action_gate.tool_names"))
	cfg.ActionGate.Keywords = normalizeKeywords(v.GetStringSlice("action_gate.keywords"))
	cfg.ActionGate.DecisionByTool = normalizeStringToPolicyMap(v.GetStringMapString("action_gate.decision_by_tool"))
	cfg.ActionGate.DecisionByWord = normalizeStringToPolicyMap(v.GetStringMapString("action_gate.decision_by_keyword"))
	cfg.Clarification.Enabled = v.GetBool("clarification.enabled")
	cfg.Clarification.Timeout = v.GetDuration("clarification.timeout")
	cfg.Clarification.TimeoutPolicy = strings.ToLower(strings.TrimSpace(v.GetString("clarification.timeout_policy")))
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
	cfg.ContextAssembler.CA2.Stage2.External = buildStage2ExternalConfig(v, cfg.ContextAssembler.CA2.Stage2.External)
	cfg.ContextAssembler.CA2.Routing.MinInputChars = v.GetInt("context_assembler.ca2.routing.min_input_chars")
	cfg.ContextAssembler.CA2.Routing.TriggerKeywords = normalizeKeywords(v.GetStringSlice("context_assembler.ca2.routing.trigger_keywords"))
	cfg.ContextAssembler.CA2.Routing.RequireSystemGuard = v.GetBool("context_assembler.ca2.routing.require_system_guard")
	cfg.ContextAssembler.CA2.TailRecap.Enabled = v.GetBool("context_assembler.ca2.tail_recap.enabled")
	cfg.ContextAssembler.CA2.TailRecap.MaxItems = v.GetInt("context_assembler.ca2.tail_recap.max_items")
	cfg.ContextAssembler.CA2.TailRecap.MaxFieldChars = v.GetInt("context_assembler.ca2.tail_recap.max_field_chars")
	cfg.ContextAssembler.CA3.Enabled = v.GetBool("context_assembler.ca3.enabled")
	cfg.ContextAssembler.CA3.MaxContextTokens = v.GetInt("context_assembler.ca3.max_context_tokens")
	cfg.ContextAssembler.CA3.GoldilocksMinPercent = v.GetInt("context_assembler.ca3.goldilocks_min_percent")
	cfg.ContextAssembler.CA3.GoldilocksMaxPercent = v.GetInt("context_assembler.ca3.goldilocks_max_percent")
	cfg.ContextAssembler.CA3.PercentThresholds = buildCA3Thresholds(v, "context_assembler.ca3.percent_thresholds")
	cfg.ContextAssembler.CA3.AbsoluteThresholds = buildCA3Thresholds(v, "context_assembler.ca3.absolute_thresholds")
	cfg.ContextAssembler.CA3.Stage1.PercentThresholds = buildCA3Thresholds(v, "context_assembler.ca3.stage1.percent_thresholds")
	cfg.ContextAssembler.CA3.Stage1.AbsoluteThresholds = buildCA3Thresholds(v, "context_assembler.ca3.stage1.absolute_thresholds")
	cfg.ContextAssembler.CA3.Stage2.PercentThresholds = buildCA3Thresholds(v, "context_assembler.ca3.stage2.percent_thresholds")
	cfg.ContextAssembler.CA3.Stage2.AbsoluteThresholds = buildCA3Thresholds(v, "context_assembler.ca3.stage2.absolute_thresholds")
	cfg.ContextAssembler.CA3.Protection.CriticalKeywords = normalizeKeywords(v.GetStringSlice("context_assembler.ca3.protection.critical_keywords"))
	cfg.ContextAssembler.CA3.Protection.ImmutableKeywords = normalizeKeywords(v.GetStringSlice("context_assembler.ca3.protection.immutable_keywords"))
	cfg.ContextAssembler.CA3.Squash.Enabled = v.GetBool("context_assembler.ca3.squash.enabled")
	cfg.ContextAssembler.CA3.Squash.MaxContentRunes = v.GetInt("context_assembler.ca3.squash.max_content_runes")
	cfg.ContextAssembler.CA3.Prune.Enabled = v.GetBool("context_assembler.ca3.prune.enabled")
	cfg.ContextAssembler.CA3.Prune.TargetPercent = v.GetInt("context_assembler.ca3.prune.target_percent")
	cfg.ContextAssembler.CA3.Prune.KeywordPriority = normalizeKeywords(v.GetStringSlice("context_assembler.ca3.prune.keyword_priority"))
	cfg.ContextAssembler.CA3.Emergency.RejectLowPriority = v.GetBool("context_assembler.ca3.emergency.reject_low_priority")
	cfg.ContextAssembler.CA3.Emergency.HighPriorityTokens = normalizeKeywords(v.GetStringSlice("context_assembler.ca3.emergency.high_priority_tokens"))
	cfg.ContextAssembler.CA3.Spill.Enabled = v.GetBool("context_assembler.ca3.spill.enabled")
	cfg.ContextAssembler.CA3.Spill.Backend = strings.ToLower(strings.TrimSpace(v.GetString("context_assembler.ca3.spill.backend")))
	cfg.ContextAssembler.CA3.Spill.Path = strings.TrimSpace(v.GetString("context_assembler.ca3.spill.path"))
	cfg.ContextAssembler.CA3.Spill.SwapBackLimit = v.GetInt("context_assembler.ca3.spill.swap_back_limit")
	cfg.ContextAssembler.CA3.Tokenizer.Mode = strings.ToLower(strings.TrimSpace(v.GetString("context_assembler.ca3.tokenizer.mode")))
	cfg.ContextAssembler.CA3.Tokenizer.Provider = strings.ToLower(strings.TrimSpace(v.GetString("context_assembler.ca3.tokenizer.provider")))
	cfg.ContextAssembler.CA3.Tokenizer.Model = strings.TrimSpace(v.GetString("context_assembler.ca3.tokenizer.model"))
	cfg.ContextAssembler.CA3.Tokenizer.SmallDeltaTokens = v.GetInt("context_assembler.ca3.tokenizer.small_delta_tokens")
	cfg.ContextAssembler.CA3.Tokenizer.SDKRefreshInterval = v.GetDuration("context_assembler.ca3.tokenizer.sdk_refresh_interval")
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

func buildStage2ExternalConfig(v *viper.Viper, base ContextAssemblerCA2ExternalConfig) ContextAssemblerCA2ExternalConfig {
	profile := strings.ToLower(strings.TrimSpace(v.GetString("context_assembler.ca2.stage2.external.profile")))
	if profile == "" {
		profile = strings.ToLower(strings.TrimSpace(base.Profile))
	}
	out := applyStage2ExternalProfile(ContextAssemblerCA2ExternalConfig{Profile: profile})
	if endpoint := strings.TrimSpace(v.GetString("context_assembler.ca2.stage2.external.endpoint")); endpoint != "" {
		out.Endpoint = endpoint
	}
	if method := strings.ToUpper(strings.TrimSpace(v.GetString("context_assembler.ca2.stage2.external.method"))); method != "" {
		out.Method = method
	}
	if token := strings.TrimSpace(v.GetString("context_assembler.ca2.stage2.external.auth.bearer_token")); token != "" {
		out.Auth.BearerToken = token
	}
	if header := strings.TrimSpace(v.GetString("context_assembler.ca2.stage2.external.auth.header_name")); header != "" {
		out.Auth.HeaderName = header
	}
	if headers := normalizeStringMap(v.GetStringMapString("context_assembler.ca2.stage2.external.headers")); len(headers) > 0 {
		out.Headers = headers
	}
	if mode := strings.ToLower(strings.TrimSpace(v.GetString("context_assembler.ca2.stage2.external.mapping.request.mode"))); mode != "" {
		out.Mapping.Request.Mode = mode
	}
	if methodName := strings.TrimSpace(v.GetString("context_assembler.ca2.stage2.external.mapping.request.method_name")); methodName != "" {
		out.Mapping.Request.MethodName = methodName
	}
	if version := strings.TrimSpace(v.GetString("context_assembler.ca2.stage2.external.mapping.request.jsonrpc_version")); version != "" {
		out.Mapping.Request.JSONRPCVersion = version
	}
	if queryField := strings.TrimSpace(v.GetString("context_assembler.ca2.stage2.external.mapping.request.query_field")); queryField != "" {
		out.Mapping.Request.QueryField = queryField
	}
	if sessionField := strings.TrimSpace(v.GetString("context_assembler.ca2.stage2.external.mapping.request.session_id_field")); sessionField != "" {
		out.Mapping.Request.SessionIDField = sessionField
	}
	if runField := strings.TrimSpace(v.GetString("context_assembler.ca2.stage2.external.mapping.request.run_id_field")); runField != "" {
		out.Mapping.Request.RunIDField = runField
	}
	if maxField := strings.TrimSpace(v.GetString("context_assembler.ca2.stage2.external.mapping.request.max_items_field")); maxField != "" {
		out.Mapping.Request.MaxItemsField = maxField
	}
	if chunksField := strings.TrimSpace(v.GetString("context_assembler.ca2.stage2.external.mapping.response.chunks_field")); chunksField != "" {
		out.Mapping.Response.ChunksField = chunksField
	}
	if sourceField := strings.TrimSpace(v.GetString("context_assembler.ca2.stage2.external.mapping.response.source_field")); sourceField != "" {
		out.Mapping.Response.SourceField = sourceField
	}
	if reasonField := strings.TrimSpace(v.GetString("context_assembler.ca2.stage2.external.mapping.response.reason_field")); reasonField != "" {
		out.Mapping.Response.ReasonField = reasonField
	}
	if errorField := strings.TrimSpace(v.GetString("context_assembler.ca2.stage2.external.mapping.response.error_field")); errorField != "" {
		out.Mapping.Response.ErrorField = errorField
	}
	if messageField := strings.TrimSpace(v.GetString("context_assembler.ca2.stage2.external.mapping.response.error_message_field")); messageField != "" {
		out.Mapping.Response.ErrorMessageField = messageField
	}
	return out
}

func buildCA3Thresholds(v *viper.Viper, prefix string) ContextAssemblerCA3Thresholds {
	return ContextAssemblerCA3Thresholds{
		Safe:      v.GetInt(prefix + ".safe"),
		Comfort:   v.GetInt(prefix + ".comfort"),
		Warning:   v.GetInt(prefix + ".warning"),
		Danger:    v.GetInt(prefix + ".danger"),
		Emergency: v.GetInt(prefix + ".emergency"),
	}
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

func normalizeStringToPolicyMap(in map[string]string) map[string]string {
	if len(in) == 0 {
		return map[string]string{}
	}
	out := make(map[string]string, len(in))
	for rawKey, rawValue := range in {
		key := strings.ToLower(strings.TrimSpace(rawKey))
		if key == "" {
			continue
		}
		value := strings.ToLower(strings.TrimSpace(rawValue))
		if value == "" {
			continue
		}
		out[key] = value
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
