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
	ContextCA2AgenticFailurePolicyBestEffortRules = "best_effort_rules"
)

const (
	ContextStage2ExternalProfileHTTPGeneric       = "http_generic"
	ContextStage2ExternalProfileRAGFlowLike       = "ragflow_like"
	ContextStage2ExternalProfileGraphRAGLike      = "graphrag_like"
	ContextStage2ExternalProfileElasticsearchLike = "elasticsearch_like"
	ContextStage2ExternalProfileExplicitOnly      = "explicit_only"
)

const (
	ActionGatePolicyAllow          = "allow"
	ActionGatePolicyRequireConfirm = "require_confirm"
	ActionGatePolicyDeny           = "deny"
)

const (
	ClarificationTimeoutPolicyCancelByUser = "cancel_by_user"
)

const (
	DropPriorityLow    = "low"
	DropPriorityNormal = "normal"
	DropPriorityHigh   = "high"
)

const (
	SkillTriggerScoringStrategyLexicalWeightedKeywords = "lexical_weighted_keywords"
	SkillTriggerScoringStrategyLexicalPlusEmbedding    = "lexical_plus_embedding"
	SkillTriggerScoringTieBreakHighestPriority         = "highest_priority"
	SkillTriggerScoringTieBreakFirstRegistered         = "first_registered"
	SkillTriggerScoringSimilarityCosine                = "cosine"
	SkillTriggerScoringTokenizerMixedCJKEN             = "mixed_cjk_en"
	SkillTriggerScoringBudgetModeFixed                 = "fixed"
	SkillTriggerScoringBudgetModeAdaptive              = "adaptive"
)

const (
	CA3RerankerGovernanceModeEnforce = "enforce"
	CA3RerankerGovernanceModeDryRun  = "dry_run"
)

const (
	SecurityGovernanceModeEnforce        = "enforce"
	SecurityToolPolicyAllow              = "allow"
	SecurityToolPolicyDeny               = "deny"
	SecurityToolRateLimitScopeProcess    = "process"
	SecurityModelIOFilterStageInput      = "input"
	SecurityModelIOFilterStageOutput     = "output"
	SecurityModelIOFilterBlockActionDeny = "deny"
	SecurityEventAlertPolicyDenyOnly     = "deny_only"
	SecurityEventAlertSinkCallback       = "callback"
	SecurityEventDeliveryModeSync        = "sync"
	SecurityEventDeliveryModeAsync       = "async"
	SecurityEventDeliveryOverflowDropOld = "drop_old"
	SecurityEventCircuitStateClosed      = "closed"
	SecurityEventCircuitStateOpen        = "open"
	SecurityEventCircuitStateHalfOpen    = "half_open"
	SecurityEventSeverityLow             = "low"
	SecurityEventSeverityMedium          = "medium"
	SecurityEventSeverityHigh            = "high"
)

type Config struct {
	MCP              MCPConfig              `json:"mcp"`
	Concurrency      ConcurrencyConfig      `json:"concurrency"`
	Diagnostics      DiagnosticsConfig      `json:"diagnostics"`
	Reload           ReloadConfig           `json:"reload"`
	ProviderFallback ProviderFallbackConfig `json:"provider_fallback"`
	Skill            SkillConfig            `json:"skill"`
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
	LocalMaxWorkers          int                    `json:"local_max_workers"`
	LocalQueueSize           int                    `json:"local_queue_size"`
	Backpressure             types.BackpressureMode `json:"backpressure"`
	CancelPropagationTimeout time.Duration          `json:"cancel_propagation_timeout"`
	DropLowPriority          DropLowPriorityConfig  `json:"drop_low_priority"`
}

type DropLowPriorityConfig struct {
	PriorityByTool      map[string]string `json:"priority_by_tool"`
	PriorityByKeyword   map[string]string `json:"priority_by_keyword"`
	DroppablePriorities []string          `json:"droppable_priorities"`
}

type DiagnosticsConfig struct {
	MaxCallRecords   int                               `json:"max_call_records"`
	MaxRunRecords    int                               `json:"max_run_records"`
	MaxReloadErrors  int                               `json:"max_reload_errors"`
	MaxSkillRecords  int                               `json:"max_skill_records"`
	TimelineTrend    DiagnosticsTimelineTrendConfig    `json:"timeline_trend"`
	CA2ExternalTrend DiagnosticsCA2ExternalTrendConfig `json:"ca2_external_trend"`
}

type DiagnosticsTimelineTrendConfig struct {
	Enabled    bool          `json:"enabled"`
	LastNRuns  int           `json:"last_n_runs"`
	TimeWindow time.Duration `json:"time_window"`
}

type DiagnosticsCA2ExternalTrendConfig struct {
	Enabled    bool                                  `json:"enabled"`
	Window     time.Duration                         `json:"window"`
	Thresholds DiagnosticsCA2ExternalTrendThresholds `json:"thresholds"`
}

type DiagnosticsCA2ExternalTrendThresholds struct {
	P95LatencyMs int64   `json:"p95_latency_ms"`
	ErrorRate    float64 `json:"error_rate"`
	HitRate      float64 `json:"hit_rate"`
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

type SkillConfig struct {
	TriggerScoring SkillTriggerScoringConfig `json:"trigger_scoring"`
}

type SkillTriggerScoringConfig struct {
	Strategy              string                             `json:"strategy"`
	ConfidenceThreshold   float64                            `json:"confidence_threshold"`
	TieBreak              string                             `json:"tie_break"`
	SuppressLowConfidence bool                               `json:"suppress_low_confidence"`
	KeywordWeights        map[string]float64                 `json:"keyword_weights"`
	Lexical               SkillTriggerScoringLexicalConfig   `json:"lexical"`
	MaxSemanticCandidates int                                `json:"max_semantic_candidates"`
	Budget                SkillTriggerScoringBudgetConfig    `json:"budget"`
	Embedding             SkillTriggerScoringEmbeddingConfig `json:"embedding"`
}

type SkillTriggerScoringLexicalConfig struct {
	TokenizerMode string `json:"tokenizer_mode"`
}

type SkillTriggerScoringBudgetConfig struct {
	Mode     string                                  `json:"mode"`
	Adaptive SkillTriggerScoringAdaptiveBudgetConfig `json:"adaptive"`
}

type SkillTriggerScoringAdaptiveBudgetConfig struct {
	MinK           int     `json:"min_k"`
	MaxK           int     `json:"max_k"`
	MinScoreMargin float64 `json:"min_score_margin"`
}

type SkillTriggerScoringEmbeddingConfig struct {
	Enabled          bool          `json:"enabled"`
	Provider         string        `json:"provider"`
	Model            string        `json:"model"`
	Timeout          time.Duration `json:"timeout"`
	SimilarityMetric string        `json:"similarity_metric"`
	LexicalWeight    float64       `json:"lexical_weight"`
	EmbeddingWeight  float64       `json:"embedding_weight"`
}

type ActionGateConfig struct {
	Enabled        bool                            `json:"enabled"`
	Policy         string                          `json:"policy"`
	Timeout        time.Duration                   `json:"timeout"`
	ToolNames      []string                        `json:"tool_names"`
	Keywords       []string                        `json:"keywords"`
	DecisionByTool map[string]string               `json:"decision_by_tool"`
	DecisionByWord map[string]string               `json:"decision_by_keyword"`
	ParameterRules []types.ActionGateParameterRule `json:"parameter_rules"`
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
	Agentic     ContextAssemblerCA2AgenticConfig   `json:"agentic"`
	StagePolicy ContextAssemblerCA2StagePolicy     `json:"stage_policy"`
	Timeout     ContextAssemblerCA2TimeoutConfig   `json:"timeout"`
	Stage2      ContextAssemblerCA2Stage2Config    `json:"stage2"`
	Routing     ContextAssemblerCA2RoutingConfig   `json:"routing"`
	TailRecap   ContextAssemblerCA2TailRecapConfig `json:"tail_recap"`
}

type ContextAssemblerCA2AgenticConfig struct {
	DecisionTimeout time.Duration `json:"decision_timeout"`
	FailurePolicy   string        `json:"failure_policy"`
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
	Profile                  string                                   `json:"profile"`
	TemplateResolutionSource string                                   `json:"template_resolution_source,omitempty"`
	Endpoint                 string                                   `json:"endpoint"`
	Method                   string                                   `json:"method"`
	Auth                     ContextAssemblerCA2ExternalAuthConfig    `json:"auth"`
	Headers                  map[string]string                        `json:"headers"`
	Mapping                  ContextAssemblerCA2ExternalMappingConfig `json:"mapping"`
	Hints                    ContextAssemblerCA2ExternalHintConfig    `json:"hints"`
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

type ContextAssemblerCA2ExternalHintConfig struct {
	Enabled      bool     `json:"enabled"`
	Capabilities []string `json:"capabilities,omitempty"`
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
	Compaction           ContextAssemblerCA3CompactionConfig        `json:"compaction"`
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

type ContextAssemblerCA3CompactionConfig struct {
	Mode             string                                       `json:"mode"`
	SemanticTimeout  time.Duration                                `json:"semantic_timeout"`
	Quality          ContextAssemblerCA3CompactionQualityConfig   `json:"quality"`
	SemanticTemplate ContextAssemblerCA3SemanticTemplateConfig    `json:"semantic_template"`
	Embedding        ContextAssemblerCA3CompactionEmbeddingConfig `json:"embedding"`
	Reranker         ContextAssemblerCA3CompactionRerankerConfig  `json:"reranker"`
	Evidence         ContextAssemblerCA3CompactionEvidenceConfig  `json:"evidence"`
}

type ContextAssemblerCA3CompactionQualityConfig struct {
	Threshold float64                                     `json:"threshold"`
	Weights   ContextAssemblerCA3CompactionQualityWeights `json:"weights"`
}

type ContextAssemblerCA3CompactionQualityWeights struct {
	Coverage    float64 `json:"coverage"`
	Compression float64 `json:"compression"`
	Validity    float64 `json:"validity"`
}

type ContextAssemblerCA3SemanticTemplateConfig struct {
	Prompt              string   `json:"prompt"`
	AllowedPlaceholders []string `json:"allowed_placeholders"`
}

type ContextAssemblerCA3CompactionEmbeddingConfig struct {
	Enabled          bool                                            `json:"enabled"`
	Selector         string                                          `json:"selector"`
	Provider         string                                          `json:"provider"`
	Model            string                                          `json:"model"`
	Timeout          time.Duration                                   `json:"timeout"`
	SimilarityMetric string                                          `json:"similarity_metric"`
	RuleWeight       float64                                         `json:"rule_weight"`
	EmbeddingWeight  float64                                         `json:"embedding_weight"`
	Auth             ContextAssemblerCA3EmbeddingAuthConfig          `json:"auth"`
	ProviderAuth     ContextAssemblerCA3EmbeddingProviderAuthsConfig `json:"provider_auth"`
}

type ContextAssemblerCA3EmbeddingAuthConfig struct {
	APIKey  string `json:"api_key"`
	BaseURL string `json:"base_url"`
}

type ContextAssemblerCA3EmbeddingProviderAuthsConfig struct {
	OpenAI    ContextAssemblerCA3EmbeddingAuthConfig `json:"openai"`
	Gemini    ContextAssemblerCA3EmbeddingAuthConfig `json:"gemini"`
	Anthropic ContextAssemblerCA3EmbeddingAuthConfig `json:"anthropic"`
}

type ContextAssemblerCA3CompactionEvidenceConfig struct {
	Keywords     []string `json:"keywords"`
	RecentWindow int      `json:"recent_window"`
}

type ContextAssemblerCA3CompactionRerankerConfig struct {
	Enabled           bool                                                  `json:"enabled"`
	Timeout           time.Duration                                         `json:"timeout"`
	MaxRetries        int                                                   `json:"max_retries"`
	ThresholdProfiles map[string]float64                                    `json:"threshold_profiles"`
	Governance        ContextAssemblerCA3CompactionRerankerGovernanceConfig `json:"governance"`
}

type ContextAssemblerCA3CompactionRerankerGovernanceConfig struct {
	Mode                  string   `json:"mode"`
	ProfileVersion        string   `json:"profile_version"`
	RolloutProviderModels []string `json:"rollout_provider_models"`
}

type SecurityConfig struct {
	Scan             SecurityScanConfig             `json:"scan"`
	Redaction        SecurityRedactionConfig        `json:"redaction"`
	ToolGovernance   SecurityToolGovernanceConfig   `json:"tool_governance"`
	ModelIOFiltering SecurityModelIOFilteringConfig `json:"model_io_filtering"`
	SecurityEvent    SecurityEventConfig            `json:"security_event"`
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

type SecurityToolGovernanceConfig struct {
	Enabled    bool                     `json:"enabled"`
	Mode       string                   `json:"mode"`
	Permission SecurityPermissionConfig `json:"permission"`
	RateLimit  SecurityRateLimitConfig  `json:"rate_limit"`
}

type SecurityPermissionConfig struct {
	Default    string            `json:"default"`
	DenyAction string            `json:"deny_action"`
	ByTool     map[string]string `json:"by_tool"`
}

type SecurityRateLimitConfig struct {
	Enabled      bool           `json:"enabled"`
	Scope        string         `json:"scope"`
	Window       time.Duration  `json:"window"`
	Limit        int            `json:"limit"`
	ByToolLimit  map[string]int `json:"by_tool_limit"`
	ExceedAction string         `json:"exceed_action"`
}

type SecurityModelIOFilteringConfig struct {
	Enabled                 bool                             `json:"enabled"`
	RequireRegisteredFilter bool                             `json:"require_registered_filter"`
	Input                   SecurityModelIOFilterStageConfig `json:"input"`
	Output                  SecurityModelIOFilterStageConfig `json:"output"`
}

type SecurityModelIOFilterStageConfig struct {
	Enabled     bool   `json:"enabled"`
	BlockAction string `json:"block_action"`
}

type SecurityEventConfig struct {
	Enabled  bool                        `json:"enabled"`
	Alert    SecurityEventAlertConfig    `json:"alert"`
	Delivery SecurityEventDeliveryConfig `json:"delivery"`
	Severity SecuritySeverityConfig      `json:"severity"`
}

type SecurityEventAlertConfig struct {
	TriggerPolicy string                           `json:"trigger_policy"`
	Sink          string                           `json:"sink"`
	Callback      SecurityEventAlertCallbackConfig `json:"callback"`
}

type SecurityEventAlertCallbackConfig struct {
	RequireRegistered bool `json:"require_registered"`
}

type SecurityEventDeliveryConfig struct {
	Mode           string                             `json:"mode"`
	Queue          SecurityEventDeliveryQueueConfig   `json:"queue"`
	Timeout        time.Duration                      `json:"timeout"`
	Retry          SecurityEventDeliveryRetryConfig   `json:"retry"`
	CircuitBreaker SecurityEventDeliveryCircuitConfig `json:"circuit_breaker"`
}

type SecurityEventDeliveryQueueConfig struct {
	Size           int    `json:"size"`
	OverflowPolicy string `json:"overflow_policy"`
}

type SecurityEventDeliveryRetryConfig struct {
	MaxAttempts    int           `json:"max_attempts"`
	BackoffInitial time.Duration `json:"backoff_initial"`
	BackoffMax     time.Duration `json:"backoff_max"`
}

type SecurityEventDeliveryCircuitConfig struct {
	FailureThreshold int           `json:"failure_threshold"`
	OpenWindow       time.Duration `json:"open_window"`
	HalfOpenProbes   int           `json:"half_open_probes"`
}

type SecuritySeverityConfig struct {
	Default      string            `json:"default"`
	ByPolicyKind map[string]string `json:"by_policy_kind"`
	ByReasonCode map[string]string `json:"by_reason_code"`
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
			LocalMaxWorkers:          8,
			LocalQueueSize:           32,
			Backpressure:             types.BackpressureBlock,
			CancelPropagationTimeout: 1500 * time.Millisecond,
			DropLowPriority: DropLowPriorityConfig{
				PriorityByTool:      map[string]string{},
				PriorityByKeyword:   map[string]string{},
				DroppablePriorities: []string{DropPriorityLow},
			},
		},
		Diagnostics: DiagnosticsConfig{
			MaxCallRecords:  200,
			MaxRunRecords:   200,
			MaxReloadErrors: 100,
			MaxSkillRecords: 200,
			TimelineTrend: DiagnosticsTimelineTrendConfig{
				Enabled:    true,
				LastNRuns:  100,
				TimeWindow: 15 * time.Minute,
			},
			CA2ExternalTrend: DiagnosticsCA2ExternalTrendConfig{
				Enabled: true,
				Window:  15 * time.Minute,
				Thresholds: DiagnosticsCA2ExternalTrendThresholds{
					P95LatencyMs: 1500,
					ErrorRate:    0.10,
					HitRate:      0.20,
				},
			},
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
		Skill: SkillConfig{
			TriggerScoring: SkillTriggerScoringConfig{
				Strategy:              SkillTriggerScoringStrategyLexicalWeightedKeywords,
				ConfidenceThreshold:   0.25,
				TieBreak:              SkillTriggerScoringTieBreakHighestPriority,
				SuppressLowConfidence: true,
				KeywordWeights: map[string]float64{
					"database": 1.5,
					"db":       1.5,
					"sql":      1.6,
					"search":   1.2,
					"retrieve": 1.2,
					"lookup":   1.1,
					"migrate":  1.3,
				},
				Lexical: SkillTriggerScoringLexicalConfig{
					TokenizerMode: SkillTriggerScoringTokenizerMixedCJKEN,
				},
				MaxSemanticCandidates: 5,
				Budget: SkillTriggerScoringBudgetConfig{
					Mode: SkillTriggerScoringBudgetModeAdaptive,
					Adaptive: SkillTriggerScoringAdaptiveBudgetConfig{
						MinK:           1,
						MaxK:           5,
						MinScoreMargin: 0.08,
					},
				},
				Embedding: SkillTriggerScoringEmbeddingConfig{
					Enabled:          false,
					Provider:         "openai",
					Model:            "text-embedding-3-small",
					Timeout:          300 * time.Millisecond,
					SimilarityMetric: SkillTriggerScoringSimilarityCosine,
					LexicalWeight:    0.7,
					EmbeddingWeight:  0.3,
				},
			},
		},
		ActionGate: ActionGateConfig{
			Enabled:        true,
			Policy:         ActionGatePolicyRequireConfirm,
			Timeout:        15 * time.Second,
			ToolNames:      nil,
			Keywords:       nil,
			DecisionByTool: map[string]string{},
			DecisionByWord: map[string]string{},
			ParameterRules: nil,
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
				Agentic: ContextAssemblerCA2AgenticConfig{
					DecisionTimeout: 80 * time.Millisecond,
					FailurePolicy:   ContextCA2AgenticFailurePolicyBestEffortRules,
				},
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
						Profile:                  ContextStage2ExternalProfileHTTPGeneric,
						TemplateResolutionSource: Stage2TemplateResolutionProfileDefaultsOnly,
						Endpoint:                 "",
						Method:                   "POST",
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
						Hints: ContextAssemblerCA2ExternalHintConfig{
							Enabled:      false,
							Capabilities: []string{},
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
				Compaction: ContextAssemblerCA3CompactionConfig{
					Mode:            "truncate",
					SemanticTimeout: 800 * time.Millisecond,
					Quality: ContextAssemblerCA3CompactionQualityConfig{
						Threshold: 0.60,
						Weights: ContextAssemblerCA3CompactionQualityWeights{
							Coverage:    0.50,
							Compression: 0.30,
							Validity:    0.20,
						},
					},
					SemanticTemplate: ContextAssemblerCA3SemanticTemplateConfig{
						Prompt:              "Compress the text for context-window efficiency while preserving intent, constraints, decisions, todo, and risk details. Return plain text only in Chinese if source is Chinese, otherwise keep source language. Keep output under {{max_runes}} characters.\n\nUser input:\n{{input}}\n\nSource:\n{{source}}",
						AllowedPlaceholders: []string{"input", "source", "max_runes", "model", "messages_count"},
					},
					Embedding: ContextAssemblerCA3CompactionEmbeddingConfig{
						Enabled:          false,
						Selector:         "",
						Provider:         "openai",
						Model:            "text-embedding-3-small",
						Timeout:          800 * time.Millisecond,
						SimilarityMetric: "cosine",
						RuleWeight:       0.7,
						EmbeddingWeight:  0.3,
						Auth:             ContextAssemblerCA3EmbeddingAuthConfig{},
						ProviderAuth:     ContextAssemblerCA3EmbeddingProviderAuthsConfig{},
					},
					Reranker: ContextAssemblerCA3CompactionRerankerConfig{
						Enabled:           false,
						Timeout:           500 * time.Millisecond,
						MaxRetries:        1,
						ThresholdProfiles: map[string]float64{},
						Governance: ContextAssemblerCA3CompactionRerankerGovernanceConfig{
							Mode:                  CA3RerankerGovernanceModeEnforce,
							ProfileVersion:        "",
							RolloutProviderModels: []string{},
						},
					},
					Evidence: ContextAssemblerCA3CompactionEvidenceConfig{
						Keywords:     []string{"decision", "constraint", "todo", "risk"},
						RecentWindow: 0,
					},
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
			ToolGovernance: SecurityToolGovernanceConfig{
				Enabled: true,
				Mode:    SecurityGovernanceModeEnforce,
				Permission: SecurityPermissionConfig{
					Default:    SecurityToolPolicyAllow,
					DenyAction: SecurityToolPolicyDeny,
					ByTool:     map[string]string{},
				},
				RateLimit: SecurityRateLimitConfig{
					Enabled:      true,
					Scope:        SecurityToolRateLimitScopeProcess,
					Window:       time.Minute,
					Limit:        120,
					ByToolLimit:  map[string]int{},
					ExceedAction: SecurityToolPolicyDeny,
				},
			},
			ModelIOFiltering: SecurityModelIOFilteringConfig{
				Enabled:                 true,
				RequireRegisteredFilter: false,
				Input: SecurityModelIOFilterStageConfig{
					Enabled:     true,
					BlockAction: SecurityModelIOFilterBlockActionDeny,
				},
				Output: SecurityModelIOFilterStageConfig{
					Enabled:     true,
					BlockAction: SecurityModelIOFilterBlockActionDeny,
				},
			},
			SecurityEvent: SecurityEventConfig{
				Enabled: true,
				Alert: SecurityEventAlertConfig{
					TriggerPolicy: SecurityEventAlertPolicyDenyOnly,
					Sink:          SecurityEventAlertSinkCallback,
					Callback: SecurityEventAlertCallbackConfig{
						RequireRegistered: false,
					},
				},
				Delivery: SecurityEventDeliveryConfig{
					Mode: SecurityEventDeliveryModeAsync,
					Queue: SecurityEventDeliveryQueueConfig{
						Size:           128,
						OverflowPolicy: SecurityEventDeliveryOverflowDropOld,
					},
					Timeout: 1200 * time.Millisecond,
					Retry: SecurityEventDeliveryRetryConfig{
						MaxAttempts:    3,
						BackoffInitial: 120 * time.Millisecond,
						BackoffMax:     800 * time.Millisecond,
					},
					CircuitBreaker: SecurityEventDeliveryCircuitConfig{
						FailureThreshold: 5,
						OpenWindow:       5 * time.Second,
						HalfOpenProbes:   1,
					},
				},
				Severity: SecuritySeverityConfig{
					Default: SecurityEventSeverityHigh,
					ByPolicyKind: map[string]string{
						"permission": SecurityEventSeverityHigh,
						"rate_limit": SecurityEventSeverityHigh,
						"io_filter":  SecurityEventSeverityHigh,
					},
					ByReasonCode: map[string]string{
						"security.permission_denied":   SecurityEventSeverityHigh,
						"security.rate_limit_exceeded": SecurityEventSeverityHigh,
						"security.io_filter_denied":    SecurityEventSeverityHigh,
						"security.io_filter_error":     SecurityEventSeverityHigh,
						"security.io_filter_missing":   SecurityEventSeverityHigh,
						"security.io_filter_match":     SecurityEventSeverityMedium,
					},
				},
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
		if err := validateMCPBackpressure(p.Backpressure, fmt.Sprintf("mcp.profiles.%s.backpressure", name)); err != nil {
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
	if cfg.Concurrency.CancelPropagationTimeout <= 0 {
		return errors.New("concurrency.cancel_propagation_timeout must be > 0")
	}
	if len(cfg.Concurrency.DropLowPriority.DroppablePriorities) == 0 {
		return errors.New("concurrency.drop_low_priority.droppable_priorities must not be empty")
	}
	for i, priority := range cfg.Concurrency.DropLowPriority.DroppablePriorities {
		if err := validateDropPriority(priority, fmt.Sprintf("concurrency.drop_low_priority.droppable_priorities[%d]", i)); err != nil {
			return err
		}
	}
	for tool, priority := range cfg.Concurrency.DropLowPriority.PriorityByTool {
		if strings.TrimSpace(tool) == "" {
			return errors.New("concurrency.drop_low_priority.priority_by_tool contains empty key")
		}
		if err := validateDropPriority(priority, fmt.Sprintf("concurrency.drop_low_priority.priority_by_tool.%s", tool)); err != nil {
			return err
		}
	}
	for keyword, priority := range cfg.Concurrency.DropLowPriority.PriorityByKeyword {
		if strings.TrimSpace(keyword) == "" {
			return errors.New("concurrency.drop_low_priority.priority_by_keyword contains empty key")
		}
		if err := validateDropPriority(priority, fmt.Sprintf("concurrency.drop_low_priority.priority_by_keyword.%s", keyword)); err != nil {
			return err
		}
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
	if cfg.Diagnostics.TimelineTrend.LastNRuns <= 0 {
		return errors.New("diagnostics.timeline_trend.last_n_runs must be > 0")
	}
	if cfg.Diagnostics.TimelineTrend.TimeWindow <= 0 {
		return errors.New("diagnostics.timeline_trend.time_window must be > 0")
	}
	if cfg.Diagnostics.CA2ExternalTrend.Window <= 0 {
		return errors.New("diagnostics.ca2_external_trend.window must be > 0")
	}
	if cfg.Diagnostics.CA2ExternalTrend.Thresholds.P95LatencyMs <= 0 {
		return errors.New("diagnostics.ca2_external_trend.thresholds.p95_latency_ms must be > 0")
	}
	if cfg.Diagnostics.CA2ExternalTrend.Thresholds.ErrorRate < 0 || cfg.Diagnostics.CA2ExternalTrend.Thresholds.ErrorRate > 1 {
		return errors.New("diagnostics.ca2_external_trend.thresholds.error_rate must be in [0,1]")
	}
	if cfg.Diagnostics.CA2ExternalTrend.Thresholds.HitRate < 0 || cfg.Diagnostics.CA2ExternalTrend.Thresholds.HitRate > 1 {
		return errors.New("diagnostics.ca2_external_trend.thresholds.hit_rate must be in [0,1]")
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
	scoring := cfg.Skill.TriggerScoring
	switch strategy := strings.ToLower(strings.TrimSpace(scoring.Strategy)); strategy {
	case SkillTriggerScoringStrategyLexicalWeightedKeywords, SkillTriggerScoringStrategyLexicalPlusEmbedding:
	default:
		return fmt.Errorf(
			"skill.trigger_scoring.strategy must be one of [%s,%s], got %q",
			SkillTriggerScoringStrategyLexicalWeightedKeywords,
			SkillTriggerScoringStrategyLexicalPlusEmbedding,
			scoring.Strategy,
		)
	}
	if scoring.ConfidenceThreshold < 0 || scoring.ConfidenceThreshold > 1 {
		return errors.New("skill.trigger_scoring.confidence_threshold must be in [0,1]")
	}
	switch tieBreak := strings.ToLower(strings.TrimSpace(scoring.TieBreak)); tieBreak {
	case SkillTriggerScoringTieBreakHighestPriority, SkillTriggerScoringTieBreakFirstRegistered:
	default:
		return fmt.Errorf("skill.trigger_scoring.tie_break must be one of [%s,%s], got %q", SkillTriggerScoringTieBreakHighestPriority, SkillTriggerScoringTieBreakFirstRegistered, scoring.TieBreak)
	}
	if len(scoring.KeywordWeights) == 0 {
		return errors.New("skill.trigger_scoring.keyword_weights must not be empty")
	}
	switch mode := strings.ToLower(strings.TrimSpace(scoring.Lexical.TokenizerMode)); mode {
	case SkillTriggerScoringTokenizerMixedCJKEN:
	default:
		return fmt.Errorf("skill.trigger_scoring.lexical.tokenizer_mode must be one of [%s], got %q", SkillTriggerScoringTokenizerMixedCJKEN, scoring.Lexical.TokenizerMode)
	}
	if scoring.MaxSemanticCandidates <= 0 {
		return errors.New("skill.trigger_scoring.max_semantic_candidates must be > 0")
	}
	switch mode := strings.ToLower(strings.TrimSpace(scoring.Budget.Mode)); mode {
	case SkillTriggerScoringBudgetModeFixed, SkillTriggerScoringBudgetModeAdaptive:
	default:
		return fmt.Errorf(
			"skill.trigger_scoring.budget.mode must be one of [%s,%s], got %q",
			SkillTriggerScoringBudgetModeFixed,
			SkillTriggerScoringBudgetModeAdaptive,
			scoring.Budget.Mode,
		)
	}
	if scoring.Budget.Adaptive.MinK <= 0 {
		return errors.New("skill.trigger_scoring.budget.adaptive.min_k must be > 0")
	}
	if scoring.Budget.Adaptive.MaxK < scoring.Budget.Adaptive.MinK {
		return errors.New("skill.trigger_scoring.budget.adaptive.max_k must be >= min_k")
	}
	if scoring.Budget.Adaptive.MaxK > scoring.MaxSemanticCandidates {
		return errors.New("skill.trigger_scoring.budget.adaptive.max_k must be <= max_semantic_candidates")
	}
	if scoring.Budget.Adaptive.MinScoreMargin < 0 || scoring.Budget.Adaptive.MinScoreMargin > 1 {
		return errors.New("skill.trigger_scoring.budget.adaptive.min_score_margin must be in [0,1]")
	}
	for keyword, weight := range scoring.KeywordWeights {
		k := strings.TrimSpace(strings.ToLower(keyword))
		if k == "" {
			return errors.New("skill.trigger_scoring.keyword_weights contains empty key")
		}
		if weight <= 0 {
			return fmt.Errorf("skill.trigger_scoring.keyword_weights.%s must be > 0", k)
		}
	}
	if err := validateSkillTriggerEmbeddingConfig(scoring, "skill.trigger_scoring.embedding"); err != nil {
		return err
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
		seenRuleID := map[string]struct{}{}
		for i, rule := range cfg.ActionGate.ParameterRules {
			if strings.TrimSpace(rule.ID) == "" {
				return fmt.Errorf("action_gate.parameter_rules[%d].id must not be empty", i)
			}
			ruleID := strings.ToLower(strings.TrimSpace(rule.ID))
			if _, ok := seenRuleID[ruleID]; ok {
				return fmt.Errorf("action_gate.parameter_rules[%d].id=%q is duplicated", i, rule.ID)
			}
			seenRuleID[ruleID] = struct{}{}
			for j, tool := range rule.ToolNames {
				if strings.TrimSpace(tool) == "" {
					return fmt.Errorf("action_gate.parameter_rules[%d].tool_names[%d] must not be empty", i, j)
				}
			}
			if strings.TrimSpace(string(rule.Action)) != "" {
				if err := validateActionGatePolicy(strings.TrimSpace(string(rule.Action)), fmt.Sprintf("action_gate.parameter_rules[%d].action", i)); err != nil {
					return err
				}
			}
			if err := validateActionGateRuleCondition(rule.Condition, fmt.Sprintf("action_gate.parameter_rules[%d].condition", i)); err != nil {
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
			if cfg.ContextAssembler.CA2.Agentic.DecisionTimeout <= 0 {
				return errors.New("context_assembler.ca2.agentic.decision_timeout must be > 0")
			}
			agenticPolicy := strings.ToLower(strings.TrimSpace(cfg.ContextAssembler.CA2.Agentic.FailurePolicy))
			if agenticPolicy == "" {
				agenticPolicy = ContextCA2AgenticFailurePolicyBestEffortRules
			}
			if err := validateCA2AgenticFailurePolicy(agenticPolicy, "context_assembler.ca2.agentic.failure_policy"); err != nil {
				return err
			}
			cfg.ContextAssembler.CA2.Agentic.FailurePolicy = agenticPolicy

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
	if err := validateSecurityToolGovernance(cfg.Security.ToolGovernance); err != nil {
		return err
	}
	if err := validateSecurityModelIOFiltering(cfg.Security.ModelIOFiltering); err != nil {
		return err
	}
	return validateSecurityEventConfig(cfg.Security.SecurityEvent)
}

func validateStagePolicy(v, field string) error {
	switch v {
	case "fail_fast", "best_effort":
		return nil
	default:
		return fmt.Errorf("%s must be one of [fail_fast,best_effort]", field)
	}
}

func validateCA2AgenticFailurePolicy(v, field string) error {
	switch v {
	case ContextCA2AgenticFailurePolicyBestEffortRules:
		return nil
	default:
		return fmt.Errorf("%s must be one of [%s]", field, ContextCA2AgenticFailurePolicyBestEffortRules)
	}
}

func validateSkillTriggerEmbeddingConfig(scoring SkillTriggerScoringConfig, field string) error {
	embedding := scoring.Embedding
	if embedding.Timeout <= 0 {
		return fmt.Errorf("%s.timeout must be > 0", field)
	}
	metric := strings.ToLower(strings.TrimSpace(embedding.SimilarityMetric))
	if metric == "" {
		metric = SkillTriggerScoringSimilarityCosine
	}
	if metric != SkillTriggerScoringSimilarityCosine {
		return fmt.Errorf("%s.similarity_metric must be %s, got %q", field, SkillTriggerScoringSimilarityCosine, embedding.SimilarityMetric)
	}
	if embedding.LexicalWeight < 0 || embedding.LexicalWeight > 1 {
		return fmt.Errorf("%s.lexical_weight must be in [0,1]", field)
	}
	if embedding.EmbeddingWeight < 0 || embedding.EmbeddingWeight > 1 {
		return fmt.Errorf("%s.embedding_weight must be in [0,1]", field)
	}
	if (embedding.LexicalWeight + embedding.EmbeddingWeight) <= 0 {
		return fmt.Errorf("%s.lexical_weight + %s.embedding_weight must be > 0", field, field)
	}
	strategy := strings.ToLower(strings.TrimSpace(scoring.Strategy))
	if strategy == SkillTriggerScoringStrategyLexicalPlusEmbedding && !embedding.Enabled {
		return fmt.Errorf("%s.enabled must be true when skill.trigger_scoring.strategy=%s", field, SkillTriggerScoringStrategyLexicalPlusEmbedding)
	}
	if !embedding.Enabled {
		return nil
	}
	provider := strings.ToLower(strings.TrimSpace(embedding.Provider))
	switch provider {
	case "openai", "gemini", "anthropic":
	default:
		return fmt.Errorf("%s.provider must be one of [openai,gemini,anthropic], got %q", field, embedding.Provider)
	}
	if strings.TrimSpace(embedding.Model) == "" {
		return fmt.Errorf("%s.model must not be empty when %s.enabled=true", field, field)
	}
	return nil
}

func validateBackpressure(v types.BackpressureMode, field string) error {
	switch v {
	case types.BackpressureBlock, types.BackpressureReject, types.BackpressureDropLowPriority:
		return nil
	default:
		return fmt.Errorf("%s must be one of [block,reject,drop_low_priority]", field)
	}
}

func validateMCPBackpressure(v types.BackpressureMode, field string) error {
	switch v {
	case types.BackpressureBlock, types.BackpressureReject:
		return nil
	default:
		return fmt.Errorf("%s must be one of [block,reject]", field)
	}
}

func validateDropPriority(v, field string) error {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case DropPriorityLow, DropPriorityNormal, DropPriorityHigh:
		return nil
	default:
		return fmt.Errorf("%s must be one of [low,normal,high], got %q", field, v)
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

func validateSecurityToolGovernance(cfg SecurityToolGovernanceConfig) error {
	mode := strings.ToLower(strings.TrimSpace(cfg.Mode))
	if mode == "" {
		mode = SecurityGovernanceModeEnforce
	}
	switch mode {
	case SecurityGovernanceModeEnforce:
	default:
		return fmt.Errorf("security.tool_governance.mode must be one of [%s], got %q", SecurityGovernanceModeEnforce, cfg.Mode)
	}
	if err := validateSecurityPolicyValue(cfg.Permission.Default, "security.tool_governance.permission.default", []string{SecurityToolPolicyAllow, SecurityToolPolicyDeny}); err != nil {
		return err
	}
	if err := validateSecurityPolicyValue(cfg.Permission.DenyAction, "security.tool_governance.permission.deny_action", []string{SecurityToolPolicyDeny}); err != nil {
		return err
	}
	for key, policy := range cfg.Permission.ByTool {
		if err := validateNamespaceToolKey(key, fmt.Sprintf("security.tool_governance.permission.by_tool.%s", key)); err != nil {
			return err
		}
		if err := validateSecurityPolicyValue(policy, fmt.Sprintf("security.tool_governance.permission.by_tool.%s", key), []string{SecurityToolPolicyAllow, SecurityToolPolicyDeny}); err != nil {
			return err
		}
	}
	if cfg.RateLimit.Enabled {
		scope := strings.ToLower(strings.TrimSpace(cfg.RateLimit.Scope))
		if scope == "" {
			scope = SecurityToolRateLimitScopeProcess
		}
		switch scope {
		case SecurityToolRateLimitScopeProcess:
		default:
			return fmt.Errorf("security.tool_governance.rate_limit.scope must be one of [%s], got %q", SecurityToolRateLimitScopeProcess, cfg.RateLimit.Scope)
		}
		if cfg.RateLimit.Window <= 0 {
			return errors.New("security.tool_governance.rate_limit.window must be > 0")
		}
		if cfg.RateLimit.Limit <= 0 {
			return errors.New("security.tool_governance.rate_limit.limit must be > 0")
		}
		for key, limit := range cfg.RateLimit.ByToolLimit {
			if err := validateNamespaceToolKey(key, fmt.Sprintf("security.tool_governance.rate_limit.by_tool_limit.%s", key)); err != nil {
				return err
			}
			if limit <= 0 {
				return fmt.Errorf("security.tool_governance.rate_limit.by_tool_limit.%s must be > 0", key)
			}
		}
		if err := validateSecurityPolicyValue(cfg.RateLimit.ExceedAction, "security.tool_governance.rate_limit.exceed_action", []string{SecurityToolPolicyDeny}); err != nil {
			return err
		}
	}
	return nil
}

func validateSecurityModelIOFiltering(cfg SecurityModelIOFilteringConfig) error {
	if !cfg.Enabled {
		return nil
	}
	if err := validateSecurityModelIOFilterStage(cfg.Input, SecurityModelIOFilterStageInput); err != nil {
		return err
	}
	return validateSecurityModelIOFilterStage(cfg.Output, SecurityModelIOFilterStageOutput)
}

func validateSecurityModelIOFilterStage(cfg SecurityModelIOFilterStageConfig, stage string) error {
	field := fmt.Sprintf("security.model_io_filtering.%s.block_action", stage)
	return validateSecurityPolicyValue(cfg.BlockAction, field, []string{SecurityModelIOFilterBlockActionDeny})
}

func validateSecurityEventConfig(cfg SecurityEventConfig) error {
	if err := validateSecurityEventAlert(cfg.Alert); err != nil {
		return err
	}
	if err := validateSecurityEventDelivery(cfg.Delivery); err != nil {
		return err
	}
	if err := validateSecuritySeverity(cfg.Severity.Default, "security.security_event.severity.default"); err != nil {
		return err
	}
	for key, level := range cfg.Severity.ByPolicyKind {
		kind := strings.ToLower(strings.TrimSpace(key))
		if kind == "" {
			return errors.New("security.security_event.severity.by_policy_kind contains empty key")
		}
		switch kind {
		case "permission", "rate_limit", "io_filter":
		default:
			return fmt.Errorf("security.security_event.severity.by_policy_kind.%s must be one of [permission,rate_limit,io_filter]", kind)
		}
		if err := validateSecuritySeverity(level, fmt.Sprintf("security.security_event.severity.by_policy_kind.%s", kind)); err != nil {
			return err
		}
	}
	for key, level := range cfg.Severity.ByReasonCode {
		reason := strings.ToLower(strings.TrimSpace(key))
		if reason == "" {
			return errors.New("security.security_event.severity.by_reason_code contains empty key")
		}
		if err := validateSecuritySeverity(level, fmt.Sprintf("security.security_event.severity.by_reason_code.%s", reason)); err != nil {
			return err
		}
	}
	return nil
}

func validateSecurityEventAlert(cfg SecurityEventAlertConfig) error {
	switch strings.ToLower(strings.TrimSpace(cfg.TriggerPolicy)) {
	case SecurityEventAlertPolicyDenyOnly:
	default:
		return fmt.Errorf("security.security_event.alert.trigger_policy must be one of [%s], got %q", SecurityEventAlertPolicyDenyOnly, cfg.TriggerPolicy)
	}
	switch strings.ToLower(strings.TrimSpace(cfg.Sink)) {
	case SecurityEventAlertSinkCallback:
	default:
		return fmt.Errorf("security.security_event.alert.sink must be one of [%s], got %q", SecurityEventAlertSinkCallback, cfg.Sink)
	}
	return nil
}

func validateSecurityEventDelivery(cfg SecurityEventDeliveryConfig) error {
	switch strings.ToLower(strings.TrimSpace(cfg.Mode)) {
	case SecurityEventDeliveryModeSync, SecurityEventDeliveryModeAsync:
	default:
		return fmt.Errorf(
			"security.security_event.delivery.mode must be one of [%s,%s], got %q",
			SecurityEventDeliveryModeSync,
			SecurityEventDeliveryModeAsync,
			cfg.Mode,
		)
	}
	if cfg.Queue.Size <= 0 {
		return errors.New("security.security_event.delivery.queue.size must be > 0")
	}
	switch strings.ToLower(strings.TrimSpace(cfg.Queue.OverflowPolicy)) {
	case SecurityEventDeliveryOverflowDropOld:
	default:
		return fmt.Errorf(
			"security.security_event.delivery.queue.overflow_policy must be one of [%s], got %q",
			SecurityEventDeliveryOverflowDropOld,
			cfg.Queue.OverflowPolicy,
		)
	}
	if cfg.Timeout <= 0 {
		return errors.New("security.security_event.delivery.timeout must be > 0")
	}
	if cfg.Retry.MaxAttempts <= 0 || cfg.Retry.MaxAttempts > 3 {
		return errors.New("security.security_event.delivery.retry.max_attempts must be in [1,3]")
	}
	if cfg.Retry.BackoffInitial <= 0 {
		return errors.New("security.security_event.delivery.retry.backoff_initial must be > 0")
	}
	if cfg.Retry.BackoffMax <= 0 {
		return errors.New("security.security_event.delivery.retry.backoff_max must be > 0")
	}
	if cfg.Retry.BackoffMax < cfg.Retry.BackoffInitial {
		return errors.New("security.security_event.delivery.retry.backoff_max must be >= backoff_initial")
	}
	if cfg.CircuitBreaker.FailureThreshold <= 0 {
		return errors.New("security.security_event.delivery.circuit_breaker.failure_threshold must be > 0")
	}
	if cfg.CircuitBreaker.OpenWindow <= 0 {
		return errors.New("security.security_event.delivery.circuit_breaker.open_window must be > 0")
	}
	if cfg.CircuitBreaker.HalfOpenProbes <= 0 {
		return errors.New("security.security_event.delivery.circuit_breaker.half_open_probes must be > 0")
	}
	return nil
}

func validateSecuritySeverity(value, field string) error {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case SecurityEventSeverityLow, SecurityEventSeverityMedium, SecurityEventSeverityHigh:
		return nil
	default:
		return fmt.Errorf("%s must be one of [%s,%s,%s], got %q", field, SecurityEventSeverityLow, SecurityEventSeverityMedium, SecurityEventSeverityHigh, value)
	}
}

func validateSecurityPolicyValue(value, field string, allowed []string) error {
	normalized := strings.ToLower(strings.TrimSpace(value))
	if normalized == "" && len(allowed) > 0 {
		normalized = allowed[0]
	}
	for _, item := range allowed {
		if normalized == item {
			return nil
		}
	}
	return fmt.Errorf("%s must be one of [%s], got %q", field, strings.Join(allowed, ","), value)
}

func validateNamespaceToolKey(raw, field string) error {
	key := strings.ToLower(strings.TrimSpace(raw))
	parts := strings.Split(key, "+")
	if len(parts) != 2 {
		return fmt.Errorf("%s must be in namespace+tool format, got %q", field, raw)
	}
	if strings.TrimSpace(parts[0]) == "" || strings.TrimSpace(parts[1]) == "" {
		return fmt.Errorf("%s must be in namespace+tool format, got %q", field, raw)
	}
	return nil
}

func validateActionGateRuleCondition(c types.ActionGateRuleCondition, field string) error {
	if len(c.All) > 0 && len(c.Any) > 0 {
		return fmt.Errorf("%s must not define both all and any", field)
	}
	if len(c.All) > 0 {
		for i, child := range c.All {
			if err := validateActionGateRuleCondition(child, fmt.Sprintf("%s.all[%d]", field, i)); err != nil {
				return err
			}
		}
		return nil
	}
	if len(c.Any) > 0 {
		for i, child := range c.Any {
			if err := validateActionGateRuleCondition(child, fmt.Sprintf("%s.any[%d]", field, i)); err != nil {
				return err
			}
		}
		return nil
	}
	if strings.TrimSpace(c.Path) == "" {
		return fmt.Errorf("%s.path must not be empty", field)
	}
	switch strings.ToLower(strings.TrimSpace(string(c.Operator))) {
	case string(types.ActionGateRuleOperatorEQ),
		string(types.ActionGateRuleOperatorNE),
		string(types.ActionGateRuleOperatorContains),
		string(types.ActionGateRuleOperatorRegex),
		string(types.ActionGateRuleOperatorIn),
		string(types.ActionGateRuleOperatorNotIn),
		string(types.ActionGateRuleOperatorGT),
		string(types.ActionGateRuleOperatorGTE),
		string(types.ActionGateRuleOperatorLT),
		string(types.ActionGateRuleOperatorLTE),
		string(types.ActionGateRuleOperatorExists):
	default:
		return fmt.Errorf("%s.operator=%q is not supported", field, c.Operator)
	}
	return nil
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
	compactionMode := strings.ToLower(strings.TrimSpace(cfg.Compaction.Mode))
	if compactionMode == "" {
		compactionMode = "truncate"
	}
	switch compactionMode {
	case "truncate", "semantic":
	default:
		return fmt.Errorf("context_assembler.ca3.compaction.mode must be one of [truncate,semantic], got %q", cfg.Compaction.Mode)
	}
	if cfg.Compaction.SemanticTimeout <= 0 {
		return errors.New("context_assembler.ca3.compaction.semantic_timeout must be > 0")
	}
	if cfg.Compaction.Quality.Threshold < 0 || cfg.Compaction.Quality.Threshold > 1 {
		return errors.New("context_assembler.ca3.compaction.quality.threshold must be in [0,1]")
	}
	weights := cfg.Compaction.Quality.Weights
	if weights.Coverage < 0 || weights.Compression < 0 || weights.Validity < 0 {
		return errors.New("context_assembler.ca3.compaction.quality.weights.* must be >= 0")
	}
	if (weights.Coverage + weights.Compression + weights.Validity) <= 0 {
		return errors.New("context_assembler.ca3.compaction.quality.weights total must be > 0")
	}
	if err := validateSemanticTemplate(cfg.Compaction.SemanticTemplate); err != nil {
		return err
	}
	embedding := cfg.Compaction.Embedding
	if strings.TrimSpace(embedding.SimilarityMetric) == "" {
		embedding.SimilarityMetric = "cosine"
	}
	if !strings.EqualFold(strings.TrimSpace(embedding.SimilarityMetric), "cosine") {
		return fmt.Errorf("context_assembler.ca3.compaction.embedding.similarity_metric must be cosine, got %q", cfg.Compaction.Embedding.SimilarityMetric)
	}
	if embedding.RuleWeight < 0 || embedding.RuleWeight > 1 {
		return errors.New("context_assembler.ca3.compaction.embedding.rule_weight must be in [0,1]")
	}
	if embedding.EmbeddingWeight < 0 || embedding.EmbeddingWeight > 1 {
		return errors.New("context_assembler.ca3.compaction.embedding.embedding_weight must be in [0,1]")
	}
	if (embedding.RuleWeight + embedding.EmbeddingWeight) <= 0 {
		return errors.New("context_assembler.ca3.compaction.embedding.rule_weight + embedding.embedding_weight must be > 0")
	}
	if embedding.Enabled {
		if strings.TrimSpace(embedding.Selector) == "" {
			return errors.New("context_assembler.ca3.compaction.embedding.selector is required when embedding.enabled=true")
		}
		provider := strings.ToLower(strings.TrimSpace(embedding.Provider))
		switch provider {
		case "openai", "gemini", "anthropic":
		default:
			return fmt.Errorf("context_assembler.ca3.compaction.embedding.provider must be one of [openai,gemini,anthropic], got %q", cfg.Compaction.Embedding.Provider)
		}
		if strings.TrimSpace(embedding.Model) == "" {
			return errors.New("context_assembler.ca3.compaction.embedding.model is required when embedding.enabled=true")
		}
		if embedding.Timeout <= 0 {
			return errors.New("context_assembler.ca3.compaction.embedding.timeout must be > 0 when embedding.enabled=true")
		}
	}
	reranker := cfg.Compaction.Reranker
	if reranker.MaxRetries < 0 {
		return errors.New("context_assembler.ca3.compaction.reranker.max_retries must be >= 0")
	}
	govMode := strings.ToLower(strings.TrimSpace(reranker.Governance.Mode))
	if govMode == "" {
		govMode = CA3RerankerGovernanceModeEnforce
	}
	switch govMode {
	case CA3RerankerGovernanceModeEnforce, CA3RerankerGovernanceModeDryRun:
	default:
		return fmt.Errorf(
			"context_assembler.ca3.compaction.reranker.governance.mode must be one of [%s,%s], got %q",
			CA3RerankerGovernanceModeEnforce,
			CA3RerankerGovernanceModeDryRun,
			reranker.Governance.Mode,
		)
	}
	for idx, key := range reranker.Governance.RolloutProviderModels {
		normalized := strings.ToLower(strings.TrimSpace(key))
		if normalized == "" {
			return fmt.Errorf("context_assembler.ca3.compaction.reranker.governance.rollout_provider_models[%d] must not be empty", idx)
		}
		if !isValidProviderModelKey(normalized) {
			return fmt.Errorf("context_assembler.ca3.compaction.reranker.governance.rollout_provider_models[%d] must be in provider:model format, got %q", idx, key)
		}
	}
	if reranker.Enabled {
		if !embedding.Enabled {
			return errors.New("context_assembler.ca3.compaction.reranker requires embedding.enabled=true")
		}
		if reranker.Timeout <= 0 {
			return errors.New("context_assembler.ca3.compaction.reranker.timeout must be > 0 when reranker.enabled=true")
		}
		if len(reranker.ThresholdProfiles) == 0 {
			return errors.New("context_assembler.ca3.compaction.reranker.threshold_profiles must not be empty when reranker.enabled=true")
		}
		selectedKey := buildCA3ThresholdProfileKey(embedding.Provider, embedding.Model)
		if selectedKey == "" {
			return errors.New("context_assembler.ca3.compaction.reranker requires embedding provider/model to resolve threshold profile")
		}
		threshold, ok := reranker.ThresholdProfiles[selectedKey]
		if !ok {
			return fmt.Errorf("context_assembler.ca3.compaction.reranker.threshold_profiles missing key %q", selectedKey)
		}
		if threshold < 0 || threshold > 1 {
			return fmt.Errorf("context_assembler.ca3.compaction.reranker.threshold_profiles[%q] must be in [0,1]", selectedKey)
		}
	}
	for key, value := range reranker.ThresholdProfiles {
		if strings.TrimSpace(key) == "" {
			return errors.New("context_assembler.ca3.compaction.reranker.threshold_profiles contains empty key")
		}
		if value < 0 || value > 1 {
			return fmt.Errorf("context_assembler.ca3.compaction.reranker.threshold_profiles[%q] must be in [0,1]", key)
		}
	}
	if cfg.Compaction.Evidence.RecentWindow < 0 {
		return errors.New("context_assembler.ca3.compaction.evidence.recent_window must be >= 0")
	}
	return nil
}

func buildCA3ThresholdProfileKey(provider, model string) string {
	p := strings.ToLower(strings.TrimSpace(provider))
	m := strings.ToLower(strings.TrimSpace(model))
	if p == "" || m == "" {
		return ""
	}
	return p + ":" + m
}

func isValidProviderModelKey(key string) bool {
	parts := strings.SplitN(strings.TrimSpace(key), ":", 2)
	if len(parts) != 2 {
		return false
	}
	return strings.TrimSpace(parts[0]) != "" && strings.TrimSpace(parts[1]) != ""
}

func validateSemanticTemplate(cfg ContextAssemblerCA3SemanticTemplateConfig) error {
	prompt := strings.TrimSpace(cfg.Prompt)
	if prompt == "" {
		return errors.New("context_assembler.ca3.compaction.semantic_template.prompt must not be empty")
	}
	allowed := normalizeKeywords(cfg.AllowedPlaceholders)
	if len(allowed) == 0 {
		return errors.New("context_assembler.ca3.compaction.semantic_template.allowed_placeholders must not be empty")
	}
	allowedSet := make(map[string]struct{}, len(allowed))
	for _, key := range allowed {
		allowedSet[key] = struct{}{}
	}
	if strings.Count(prompt, "{{") != strings.Count(prompt, "}}") {
		return errors.New("context_assembler.ca3.compaction.semantic_template.prompt has unbalanced placeholders")
	}
	parts := strings.Split(prompt, "{{")
	for i := 1; i < len(parts); i++ {
		right := strings.SplitN(parts[i], "}}", 2)
		if len(right) < 2 {
			return errors.New("context_assembler.ca3.compaction.semantic_template.prompt has invalid placeholder")
		}
		name := strings.ToLower(strings.TrimSpace(right[0]))
		if name == "" {
			return errors.New("context_assembler.ca3.compaction.semantic_template.prompt has empty placeholder")
		}
		if _, ok := allowedSet[name]; !ok {
			return fmt.Errorf("context_assembler.ca3.compaction.semantic_template.prompt placeholder %q is not allowed", name)
		}
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
	v.SetDefault("concurrency.cancel_propagation_timeout", base.Concurrency.CancelPropagationTimeout)
	v.SetDefault("concurrency.drop_low_priority.priority_by_tool", base.Concurrency.DropLowPriority.PriorityByTool)
	v.SetDefault("concurrency.drop_low_priority.priority_by_keyword", base.Concurrency.DropLowPriority.PriorityByKeyword)
	v.SetDefault("concurrency.drop_low_priority.droppable_priorities", base.Concurrency.DropLowPriority.DroppablePriorities)
	v.SetDefault("diagnostics.max_call_records", base.Diagnostics.MaxCallRecords)
	v.SetDefault("diagnostics.max_run_records", base.Diagnostics.MaxRunRecords)
	v.SetDefault("diagnostics.max_reload_errors", base.Diagnostics.MaxReloadErrors)
	v.SetDefault("diagnostics.max_skill_records", base.Diagnostics.MaxSkillRecords)
	v.SetDefault("diagnostics.timeline_trend.enabled", base.Diagnostics.TimelineTrend.Enabled)
	v.SetDefault("diagnostics.timeline_trend.last_n_runs", base.Diagnostics.TimelineTrend.LastNRuns)
	v.SetDefault("diagnostics.timeline_trend.time_window", base.Diagnostics.TimelineTrend.TimeWindow)
	v.SetDefault("diagnostics.ca2_external_trend.enabled", base.Diagnostics.CA2ExternalTrend.Enabled)
	v.SetDefault("diagnostics.ca2_external_trend.window", base.Diagnostics.CA2ExternalTrend.Window)
	v.SetDefault("diagnostics.ca2_external_trend.thresholds.p95_latency_ms", base.Diagnostics.CA2ExternalTrend.Thresholds.P95LatencyMs)
	v.SetDefault("diagnostics.ca2_external_trend.thresholds.error_rate", base.Diagnostics.CA2ExternalTrend.Thresholds.ErrorRate)
	v.SetDefault("diagnostics.ca2_external_trend.thresholds.hit_rate", base.Diagnostics.CA2ExternalTrend.Thresholds.HitRate)
	v.SetDefault("reload.enabled", base.Reload.Enabled)
	v.SetDefault("reload.debounce", base.Reload.Debounce)
	v.SetDefault("provider_fallback.enabled", base.ProviderFallback.Enabled)
	v.SetDefault("provider_fallback.providers", base.ProviderFallback.Providers)
	v.SetDefault("provider_fallback.discovery_timeout", base.ProviderFallback.DiscoveryTimeout)
	v.SetDefault("provider_fallback.discovery_cache_ttl", base.ProviderFallback.DiscoveryCacheTTL)
	v.SetDefault("skill.trigger_scoring.strategy", base.Skill.TriggerScoring.Strategy)
	v.SetDefault("skill.trigger_scoring.confidence_threshold", base.Skill.TriggerScoring.ConfidenceThreshold)
	v.SetDefault("skill.trigger_scoring.tie_break", base.Skill.TriggerScoring.TieBreak)
	v.SetDefault("skill.trigger_scoring.suppress_low_confidence", base.Skill.TriggerScoring.SuppressLowConfidence)
	v.SetDefault("skill.trigger_scoring.keyword_weights", base.Skill.TriggerScoring.KeywordWeights)
	v.SetDefault("skill.trigger_scoring.lexical.tokenizer_mode", base.Skill.TriggerScoring.Lexical.TokenizerMode)
	v.SetDefault("skill.trigger_scoring.max_semantic_candidates", base.Skill.TriggerScoring.MaxSemanticCandidates)
	v.SetDefault("skill.trigger_scoring.budget.mode", base.Skill.TriggerScoring.Budget.Mode)
	v.SetDefault("skill.trigger_scoring.budget.adaptive.min_k", base.Skill.TriggerScoring.Budget.Adaptive.MinK)
	v.SetDefault("skill.trigger_scoring.budget.adaptive.max_k", base.Skill.TriggerScoring.Budget.Adaptive.MaxK)
	v.SetDefault("skill.trigger_scoring.budget.adaptive.min_score_margin", base.Skill.TriggerScoring.Budget.Adaptive.MinScoreMargin)
	v.SetDefault("skill.trigger_scoring.embedding.enabled", base.Skill.TriggerScoring.Embedding.Enabled)
	v.SetDefault("skill.trigger_scoring.embedding.provider", base.Skill.TriggerScoring.Embedding.Provider)
	v.SetDefault("skill.trigger_scoring.embedding.model", base.Skill.TriggerScoring.Embedding.Model)
	v.SetDefault("skill.trigger_scoring.embedding.timeout", base.Skill.TriggerScoring.Embedding.Timeout)
	v.SetDefault("skill.trigger_scoring.embedding.similarity_metric", base.Skill.TriggerScoring.Embedding.SimilarityMetric)
	v.SetDefault("skill.trigger_scoring.embedding.lexical_weight", base.Skill.TriggerScoring.Embedding.LexicalWeight)
	v.SetDefault("skill.trigger_scoring.embedding.embedding_weight", base.Skill.TriggerScoring.Embedding.EmbeddingWeight)
	v.SetDefault("action_gate.enabled", base.ActionGate.Enabled)
	v.SetDefault("action_gate.policy", base.ActionGate.Policy)
	v.SetDefault("action_gate.timeout", base.ActionGate.Timeout)
	v.SetDefault("action_gate.tool_names", base.ActionGate.ToolNames)
	v.SetDefault("action_gate.keywords", base.ActionGate.Keywords)
	v.SetDefault("action_gate.decision_by_tool", base.ActionGate.DecisionByTool)
	v.SetDefault("action_gate.decision_by_keyword", base.ActionGate.DecisionByWord)
	v.SetDefault("action_gate.parameter_rules", base.ActionGate.ParameterRules)
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
	v.SetDefault("context_assembler.ca2.agentic.decision_timeout", base.ContextAssembler.CA2.Agentic.DecisionTimeout)
	v.SetDefault("context_assembler.ca2.agentic.failure_policy", base.ContextAssembler.CA2.Agentic.FailurePolicy)
	v.SetDefault("context_assembler.ca2.stage_policy.stage1", base.ContextAssembler.CA2.StagePolicy.Stage1)
	v.SetDefault("context_assembler.ca2.stage_policy.stage2", base.ContextAssembler.CA2.StagePolicy.Stage2)
	v.SetDefault("context_assembler.ca2.timeout.stage1", base.ContextAssembler.CA2.Timeout.Stage1)
	v.SetDefault("context_assembler.ca2.timeout.stage2", base.ContextAssembler.CA2.Timeout.Stage2)
	v.SetDefault("context_assembler.ca2.stage2.provider", base.ContextAssembler.CA2.Stage2.Provider)
	v.SetDefault("context_assembler.ca2.stage2.file_path", base.ContextAssembler.CA2.Stage2.FilePath)
	v.SetDefault("context_assembler.ca2.stage2.external.profile", base.ContextAssembler.CA2.Stage2.External.Profile)
	v.SetDefault("context_assembler.ca2.stage2.external.hints.enabled", base.ContextAssembler.CA2.Stage2.External.Hints.Enabled)
	v.SetDefault("context_assembler.ca2.stage2.external.hints.capabilities", base.ContextAssembler.CA2.Stage2.External.Hints.Capabilities)
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
	v.SetDefault("context_assembler.ca3.compaction.mode", base.ContextAssembler.CA3.Compaction.Mode)
	v.SetDefault("context_assembler.ca3.compaction.semantic_timeout", base.ContextAssembler.CA3.Compaction.SemanticTimeout)
	v.SetDefault("context_assembler.ca3.compaction.quality.threshold", base.ContextAssembler.CA3.Compaction.Quality.Threshold)
	v.SetDefault("context_assembler.ca3.compaction.quality.weights.coverage", base.ContextAssembler.CA3.Compaction.Quality.Weights.Coverage)
	v.SetDefault("context_assembler.ca3.compaction.quality.weights.compression", base.ContextAssembler.CA3.Compaction.Quality.Weights.Compression)
	v.SetDefault("context_assembler.ca3.compaction.quality.weights.validity", base.ContextAssembler.CA3.Compaction.Quality.Weights.Validity)
	v.SetDefault("context_assembler.ca3.compaction.semantic_template.prompt", base.ContextAssembler.CA3.Compaction.SemanticTemplate.Prompt)
	v.SetDefault("context_assembler.ca3.compaction.semantic_template.allowed_placeholders", base.ContextAssembler.CA3.Compaction.SemanticTemplate.AllowedPlaceholders)
	v.SetDefault("context_assembler.ca3.compaction.embedding.enabled", base.ContextAssembler.CA3.Compaction.Embedding.Enabled)
	v.SetDefault("context_assembler.ca3.compaction.embedding.selector", base.ContextAssembler.CA3.Compaction.Embedding.Selector)
	v.SetDefault("context_assembler.ca3.compaction.embedding.provider", base.ContextAssembler.CA3.Compaction.Embedding.Provider)
	v.SetDefault("context_assembler.ca3.compaction.embedding.model", base.ContextAssembler.CA3.Compaction.Embedding.Model)
	v.SetDefault("context_assembler.ca3.compaction.embedding.timeout", base.ContextAssembler.CA3.Compaction.Embedding.Timeout)
	v.SetDefault("context_assembler.ca3.compaction.embedding.similarity_metric", base.ContextAssembler.CA3.Compaction.Embedding.SimilarityMetric)
	v.SetDefault("context_assembler.ca3.compaction.embedding.rule_weight", base.ContextAssembler.CA3.Compaction.Embedding.RuleWeight)
	v.SetDefault("context_assembler.ca3.compaction.embedding.embedding_weight", base.ContextAssembler.CA3.Compaction.Embedding.EmbeddingWeight)
	v.SetDefault("context_assembler.ca3.compaction.embedding.auth.api_key", base.ContextAssembler.CA3.Compaction.Embedding.Auth.APIKey)
	v.SetDefault("context_assembler.ca3.compaction.embedding.auth.base_url", base.ContextAssembler.CA3.Compaction.Embedding.Auth.BaseURL)
	v.SetDefault("context_assembler.ca3.compaction.embedding.provider_auth.openai.api_key", base.ContextAssembler.CA3.Compaction.Embedding.ProviderAuth.OpenAI.APIKey)
	v.SetDefault("context_assembler.ca3.compaction.embedding.provider_auth.openai.base_url", base.ContextAssembler.CA3.Compaction.Embedding.ProviderAuth.OpenAI.BaseURL)
	v.SetDefault("context_assembler.ca3.compaction.embedding.provider_auth.gemini.api_key", base.ContextAssembler.CA3.Compaction.Embedding.ProviderAuth.Gemini.APIKey)
	v.SetDefault("context_assembler.ca3.compaction.embedding.provider_auth.gemini.base_url", base.ContextAssembler.CA3.Compaction.Embedding.ProviderAuth.Gemini.BaseURL)
	v.SetDefault("context_assembler.ca3.compaction.embedding.provider_auth.anthropic.api_key", base.ContextAssembler.CA3.Compaction.Embedding.ProviderAuth.Anthropic.APIKey)
	v.SetDefault("context_assembler.ca3.compaction.embedding.provider_auth.anthropic.base_url", base.ContextAssembler.CA3.Compaction.Embedding.ProviderAuth.Anthropic.BaseURL)
	v.SetDefault("context_assembler.ca3.compaction.reranker.enabled", base.ContextAssembler.CA3.Compaction.Reranker.Enabled)
	v.SetDefault("context_assembler.ca3.compaction.reranker.timeout", base.ContextAssembler.CA3.Compaction.Reranker.Timeout)
	v.SetDefault("context_assembler.ca3.compaction.reranker.max_retries", base.ContextAssembler.CA3.Compaction.Reranker.MaxRetries)
	v.SetDefault("context_assembler.ca3.compaction.reranker.threshold_profiles", base.ContextAssembler.CA3.Compaction.Reranker.ThresholdProfiles)
	v.SetDefault("context_assembler.ca3.compaction.reranker.governance.mode", base.ContextAssembler.CA3.Compaction.Reranker.Governance.Mode)
	v.SetDefault("context_assembler.ca3.compaction.reranker.governance.profile_version", base.ContextAssembler.CA3.Compaction.Reranker.Governance.ProfileVersion)
	v.SetDefault("context_assembler.ca3.compaction.reranker.governance.rollout_provider_models", base.ContextAssembler.CA3.Compaction.Reranker.Governance.RolloutProviderModels)
	v.SetDefault("context_assembler.ca3.compaction.evidence.keywords", base.ContextAssembler.CA3.Compaction.Evidence.Keywords)
	v.SetDefault("context_assembler.ca3.compaction.evidence.recent_window", base.ContextAssembler.CA3.Compaction.Evidence.RecentWindow)
	v.SetDefault("security.scan.mode", base.Security.Scan.Mode)
	v.SetDefault("security.scan.govulncheck_enabled", base.Security.Scan.GovulncheckEnable)
	v.SetDefault("security.redaction.enabled", base.Security.Redaction.Enabled)
	v.SetDefault("security.redaction.strategy", base.Security.Redaction.Strategy)
	v.SetDefault("security.redaction.keywords", base.Security.Redaction.Keywords)
	v.SetDefault("security.tool_governance.enabled", base.Security.ToolGovernance.Enabled)
	v.SetDefault("security.tool_governance.mode", base.Security.ToolGovernance.Mode)
	v.SetDefault("security.tool_governance.permission.default", base.Security.ToolGovernance.Permission.Default)
	v.SetDefault("security.tool_governance.permission.deny_action", base.Security.ToolGovernance.Permission.DenyAction)
	v.SetDefault("security.tool_governance.permission.by_tool", base.Security.ToolGovernance.Permission.ByTool)
	v.SetDefault("security.tool_governance.rate_limit.enabled", base.Security.ToolGovernance.RateLimit.Enabled)
	v.SetDefault("security.tool_governance.rate_limit.scope", base.Security.ToolGovernance.RateLimit.Scope)
	v.SetDefault("security.tool_governance.rate_limit.window", base.Security.ToolGovernance.RateLimit.Window)
	v.SetDefault("security.tool_governance.rate_limit.limit", base.Security.ToolGovernance.RateLimit.Limit)
	v.SetDefault("security.tool_governance.rate_limit.by_tool_limit", base.Security.ToolGovernance.RateLimit.ByToolLimit)
	v.SetDefault("security.tool_governance.rate_limit.exceed_action", base.Security.ToolGovernance.RateLimit.ExceedAction)
	v.SetDefault("security.model_io_filtering.enabled", base.Security.ModelIOFiltering.Enabled)
	v.SetDefault("security.model_io_filtering.require_registered_filter", base.Security.ModelIOFiltering.RequireRegisteredFilter)
	v.SetDefault("security.model_io_filtering.input.enabled", base.Security.ModelIOFiltering.Input.Enabled)
	v.SetDefault("security.model_io_filtering.input.block_action", base.Security.ModelIOFiltering.Input.BlockAction)
	v.SetDefault("security.model_io_filtering.output.enabled", base.Security.ModelIOFiltering.Output.Enabled)
	v.SetDefault("security.model_io_filtering.output.block_action", base.Security.ModelIOFiltering.Output.BlockAction)
	v.SetDefault("security.security_event.enabled", base.Security.SecurityEvent.Enabled)
	v.SetDefault("security.security_event.alert.trigger_policy", base.Security.SecurityEvent.Alert.TriggerPolicy)
	v.SetDefault("security.security_event.alert.sink", base.Security.SecurityEvent.Alert.Sink)
	v.SetDefault("security.security_event.alert.callback.require_registered", base.Security.SecurityEvent.Alert.Callback.RequireRegistered)
	v.SetDefault("security.security_event.delivery.mode", base.Security.SecurityEvent.Delivery.Mode)
	v.SetDefault("security.security_event.delivery.queue.size", base.Security.SecurityEvent.Delivery.Queue.Size)
	v.SetDefault("security.security_event.delivery.queue.overflow_policy", base.Security.SecurityEvent.Delivery.Queue.OverflowPolicy)
	v.SetDefault("security.security_event.delivery.timeout", base.Security.SecurityEvent.Delivery.Timeout)
	v.SetDefault("security.security_event.delivery.retry.max_attempts", base.Security.SecurityEvent.Delivery.Retry.MaxAttempts)
	v.SetDefault("security.security_event.delivery.retry.backoff_initial", base.Security.SecurityEvent.Delivery.Retry.BackoffInitial)
	v.SetDefault("security.security_event.delivery.retry.backoff_max", base.Security.SecurityEvent.Delivery.Retry.BackoffMax)
	v.SetDefault("security.security_event.delivery.circuit_breaker.failure_threshold", base.Security.SecurityEvent.Delivery.CircuitBreaker.FailureThreshold)
	v.SetDefault("security.security_event.delivery.circuit_breaker.open_window", base.Security.SecurityEvent.Delivery.CircuitBreaker.OpenWindow)
	v.SetDefault("security.security_event.delivery.circuit_breaker.half_open_probes", base.Security.SecurityEvent.Delivery.CircuitBreaker.HalfOpenProbes)
	v.SetDefault("security.security_event.severity.default", base.Security.SecurityEvent.Severity.Default)
	v.SetDefault("security.security_event.severity.by_policy_kind", base.Security.SecurityEvent.Severity.ByPolicyKind)
	v.SetDefault("security.security_event.severity.by_reason_code", base.Security.SecurityEvent.Severity.ByReasonCode)
}

func buildConfig(v *viper.Viper) Config {
	cfg := DefaultConfig()
	cfg.MCP.ActiveProfile = strings.TrimSpace(v.GetString("mcp.active_profile"))
	cfg.Concurrency.LocalMaxWorkers = v.GetInt("concurrency.local_max_workers")
	cfg.Concurrency.LocalQueueSize = v.GetInt("concurrency.local_queue_size")
	cfg.Concurrency.Backpressure = types.BackpressureMode(v.GetString("concurrency.backpressure"))
	cfg.Concurrency.CancelPropagationTimeout = v.GetDuration("concurrency.cancel_propagation_timeout")
	cfg.Concurrency.DropLowPriority.PriorityByTool = normalizePriorityMap(v.GetStringMapString("concurrency.drop_low_priority.priority_by_tool"))
	cfg.Concurrency.DropLowPriority.PriorityByKeyword = normalizePriorityMap(v.GetStringMapString("concurrency.drop_low_priority.priority_by_keyword"))
	cfg.Concurrency.DropLowPriority.DroppablePriorities = normalizeKeywords(v.GetStringSlice("concurrency.drop_low_priority.droppable_priorities"))
	cfg.Diagnostics.MaxCallRecords = v.GetInt("diagnostics.max_call_records")
	cfg.Diagnostics.MaxRunRecords = v.GetInt("diagnostics.max_run_records")
	cfg.Diagnostics.MaxReloadErrors = v.GetInt("diagnostics.max_reload_errors")
	cfg.Diagnostics.MaxSkillRecords = v.GetInt("diagnostics.max_skill_records")
	cfg.Diagnostics.TimelineTrend.Enabled = v.GetBool("diagnostics.timeline_trend.enabled")
	cfg.Diagnostics.TimelineTrend.LastNRuns = v.GetInt("diagnostics.timeline_trend.last_n_runs")
	cfg.Diagnostics.TimelineTrend.TimeWindow = v.GetDuration("diagnostics.timeline_trend.time_window")
	cfg.Diagnostics.CA2ExternalTrend.Enabled = v.GetBool("diagnostics.ca2_external_trend.enabled")
	cfg.Diagnostics.CA2ExternalTrend.Window = v.GetDuration("diagnostics.ca2_external_trend.window")
	cfg.Diagnostics.CA2ExternalTrend.Thresholds.P95LatencyMs = v.GetInt64("diagnostics.ca2_external_trend.thresholds.p95_latency_ms")
	cfg.Diagnostics.CA2ExternalTrend.Thresholds.ErrorRate = v.GetFloat64("diagnostics.ca2_external_trend.thresholds.error_rate")
	cfg.Diagnostics.CA2ExternalTrend.Thresholds.HitRate = v.GetFloat64("diagnostics.ca2_external_trend.thresholds.hit_rate")
	cfg.Reload.Enabled = v.GetBool("reload.enabled")
	cfg.Reload.Debounce = v.GetDuration("reload.debounce")
	cfg.ProviderFallback.Enabled = v.GetBool("provider_fallback.enabled")
	cfg.ProviderFallback.Providers = normalizeProviders(v.GetStringSlice("provider_fallback.providers"))
	cfg.ProviderFallback.DiscoveryTimeout = v.GetDuration("provider_fallback.discovery_timeout")
	cfg.ProviderFallback.DiscoveryCacheTTL = v.GetDuration("provider_fallback.discovery_cache_ttl")
	cfg.Skill.TriggerScoring.Strategy = strings.ToLower(strings.TrimSpace(v.GetString("skill.trigger_scoring.strategy")))
	cfg.Skill.TriggerScoring.ConfidenceThreshold = v.GetFloat64("skill.trigger_scoring.confidence_threshold")
	cfg.Skill.TriggerScoring.TieBreak = strings.ToLower(strings.TrimSpace(v.GetString("skill.trigger_scoring.tie_break")))
	cfg.Skill.TriggerScoring.SuppressLowConfidence = v.GetBool("skill.trigger_scoring.suppress_low_confidence")
	if weights := normalizeFloatMap(v.GetStringMap("skill.trigger_scoring.keyword_weights")); len(weights) > 0 {
		cfg.Skill.TriggerScoring.KeywordWeights = weights
	}
	cfg.Skill.TriggerScoring.Lexical.TokenizerMode = strings.ToLower(strings.TrimSpace(v.GetString("skill.trigger_scoring.lexical.tokenizer_mode")))
	cfg.Skill.TriggerScoring.MaxSemanticCandidates = v.GetInt("skill.trigger_scoring.max_semantic_candidates")
	cfg.Skill.TriggerScoring.Budget.Mode = strings.ToLower(strings.TrimSpace(v.GetString("skill.trigger_scoring.budget.mode")))
	cfg.Skill.TriggerScoring.Budget.Adaptive.MinK = v.GetInt("skill.trigger_scoring.budget.adaptive.min_k")
	cfg.Skill.TriggerScoring.Budget.Adaptive.MaxK = v.GetInt("skill.trigger_scoring.budget.adaptive.max_k")
	cfg.Skill.TriggerScoring.Budget.Adaptive.MinScoreMargin = v.GetFloat64("skill.trigger_scoring.budget.adaptive.min_score_margin")
	cfg.Skill.TriggerScoring.Embedding.Enabled = v.GetBool("skill.trigger_scoring.embedding.enabled")
	cfg.Skill.TriggerScoring.Embedding.Provider = strings.ToLower(strings.TrimSpace(v.GetString("skill.trigger_scoring.embedding.provider")))
	cfg.Skill.TriggerScoring.Embedding.Model = strings.TrimSpace(v.GetString("skill.trigger_scoring.embedding.model"))
	cfg.Skill.TriggerScoring.Embedding.Timeout = v.GetDuration("skill.trigger_scoring.embedding.timeout")
	cfg.Skill.TriggerScoring.Embedding.SimilarityMetric = strings.ToLower(strings.TrimSpace(v.GetString("skill.trigger_scoring.embedding.similarity_metric")))
	cfg.Skill.TriggerScoring.Embedding.LexicalWeight = v.GetFloat64("skill.trigger_scoring.embedding.lexical_weight")
	cfg.Skill.TriggerScoring.Embedding.EmbeddingWeight = v.GetFloat64("skill.trigger_scoring.embedding.embedding_weight")
	cfg.ActionGate.Enabled = v.GetBool("action_gate.enabled")
	cfg.ActionGate.Policy = strings.ToLower(strings.TrimSpace(v.GetString("action_gate.policy")))
	cfg.ActionGate.Timeout = v.GetDuration("action_gate.timeout")
	cfg.ActionGate.ToolNames = normalizeKeywords(v.GetStringSlice("action_gate.tool_names"))
	cfg.ActionGate.Keywords = normalizeKeywords(v.GetStringSlice("action_gate.keywords"))
	cfg.ActionGate.DecisionByTool = normalizeStringToPolicyMap(v.GetStringMapString("action_gate.decision_by_tool"))
	cfg.ActionGate.DecisionByWord = normalizeStringToPolicyMap(v.GetStringMapString("action_gate.decision_by_keyword"))
	cfg.ActionGate.ParameterRules = normalizeActionGateParameterRules(v.Get("action_gate.parameter_rules"))
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
	cfg.ContextAssembler.CA2.Agentic.DecisionTimeout = v.GetDuration("context_assembler.ca2.agentic.decision_timeout")
	cfg.ContextAssembler.CA2.Agentic.FailurePolicy = strings.ToLower(strings.TrimSpace(v.GetString("context_assembler.ca2.agentic.failure_policy")))
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
	cfg.ContextAssembler.CA3.Compaction.Mode = strings.ToLower(strings.TrimSpace(v.GetString("context_assembler.ca3.compaction.mode")))
	cfg.ContextAssembler.CA3.Compaction.SemanticTimeout = v.GetDuration("context_assembler.ca3.compaction.semantic_timeout")
	cfg.ContextAssembler.CA3.Compaction.Quality.Threshold = v.GetFloat64("context_assembler.ca3.compaction.quality.threshold")
	cfg.ContextAssembler.CA3.Compaction.Quality.Weights.Coverage = v.GetFloat64("context_assembler.ca3.compaction.quality.weights.coverage")
	cfg.ContextAssembler.CA3.Compaction.Quality.Weights.Compression = v.GetFloat64("context_assembler.ca3.compaction.quality.weights.compression")
	cfg.ContextAssembler.CA3.Compaction.Quality.Weights.Validity = v.GetFloat64("context_assembler.ca3.compaction.quality.weights.validity")
	cfg.ContextAssembler.CA3.Compaction.SemanticTemplate.Prompt = strings.TrimSpace(v.GetString("context_assembler.ca3.compaction.semantic_template.prompt"))
	cfg.ContextAssembler.CA3.Compaction.SemanticTemplate.AllowedPlaceholders = normalizeKeywords(v.GetStringSlice("context_assembler.ca3.compaction.semantic_template.allowed_placeholders"))
	cfg.ContextAssembler.CA3.Compaction.Embedding.Enabled = v.GetBool("context_assembler.ca3.compaction.embedding.enabled")
	cfg.ContextAssembler.CA3.Compaction.Embedding.Selector = strings.TrimSpace(v.GetString("context_assembler.ca3.compaction.embedding.selector"))
	cfg.ContextAssembler.CA3.Compaction.Embedding.Provider = strings.ToLower(strings.TrimSpace(v.GetString("context_assembler.ca3.compaction.embedding.provider")))
	cfg.ContextAssembler.CA3.Compaction.Embedding.Model = strings.TrimSpace(v.GetString("context_assembler.ca3.compaction.embedding.model"))
	cfg.ContextAssembler.CA3.Compaction.Embedding.Timeout = v.GetDuration("context_assembler.ca3.compaction.embedding.timeout")
	cfg.ContextAssembler.CA3.Compaction.Embedding.SimilarityMetric = strings.ToLower(strings.TrimSpace(v.GetString("context_assembler.ca3.compaction.embedding.similarity_metric")))
	cfg.ContextAssembler.CA3.Compaction.Embedding.RuleWeight = v.GetFloat64("context_assembler.ca3.compaction.embedding.rule_weight")
	cfg.ContextAssembler.CA3.Compaction.Embedding.EmbeddingWeight = v.GetFloat64("context_assembler.ca3.compaction.embedding.embedding_weight")
	cfg.ContextAssembler.CA3.Compaction.Embedding.Auth.APIKey = strings.TrimSpace(v.GetString("context_assembler.ca3.compaction.embedding.auth.api_key"))
	cfg.ContextAssembler.CA3.Compaction.Embedding.Auth.BaseURL = strings.TrimSpace(v.GetString("context_assembler.ca3.compaction.embedding.auth.base_url"))
	cfg.ContextAssembler.CA3.Compaction.Embedding.ProviderAuth.OpenAI.APIKey = strings.TrimSpace(v.GetString("context_assembler.ca3.compaction.embedding.provider_auth.openai.api_key"))
	cfg.ContextAssembler.CA3.Compaction.Embedding.ProviderAuth.OpenAI.BaseURL = strings.TrimSpace(v.GetString("context_assembler.ca3.compaction.embedding.provider_auth.openai.base_url"))
	cfg.ContextAssembler.CA3.Compaction.Embedding.ProviderAuth.Gemini.APIKey = strings.TrimSpace(v.GetString("context_assembler.ca3.compaction.embedding.provider_auth.gemini.api_key"))
	cfg.ContextAssembler.CA3.Compaction.Embedding.ProviderAuth.Gemini.BaseURL = strings.TrimSpace(v.GetString("context_assembler.ca3.compaction.embedding.provider_auth.gemini.base_url"))
	cfg.ContextAssembler.CA3.Compaction.Embedding.ProviderAuth.Anthropic.APIKey = strings.TrimSpace(v.GetString("context_assembler.ca3.compaction.embedding.provider_auth.anthropic.api_key"))
	cfg.ContextAssembler.CA3.Compaction.Embedding.ProviderAuth.Anthropic.BaseURL = strings.TrimSpace(v.GetString("context_assembler.ca3.compaction.embedding.provider_auth.anthropic.base_url"))
	cfg.ContextAssembler.CA3.Compaction.Reranker.Enabled = v.GetBool("context_assembler.ca3.compaction.reranker.enabled")
	cfg.ContextAssembler.CA3.Compaction.Reranker.Timeout = v.GetDuration("context_assembler.ca3.compaction.reranker.timeout")
	cfg.ContextAssembler.CA3.Compaction.Reranker.MaxRetries = v.GetInt("context_assembler.ca3.compaction.reranker.max_retries")
	cfg.ContextAssembler.CA3.Compaction.Reranker.ThresholdProfiles = normalizeFloatMap(v.GetStringMap("context_assembler.ca3.compaction.reranker.threshold_profiles"))
	cfg.ContextAssembler.CA3.Compaction.Reranker.Governance.Mode = strings.ToLower(strings.TrimSpace(v.GetString("context_assembler.ca3.compaction.reranker.governance.mode")))
	cfg.ContextAssembler.CA3.Compaction.Reranker.Governance.ProfileVersion = strings.TrimSpace(v.GetString("context_assembler.ca3.compaction.reranker.governance.profile_version"))
	cfg.ContextAssembler.CA3.Compaction.Reranker.Governance.RolloutProviderModels = normalizeKeywords(v.GetStringSlice("context_assembler.ca3.compaction.reranker.governance.rollout_provider_models"))
	cfg.ContextAssembler.CA3.Compaction.Evidence.Keywords = normalizeKeywords(v.GetStringSlice("context_assembler.ca3.compaction.evidence.keywords"))
	cfg.ContextAssembler.CA3.Compaction.Evidence.RecentWindow = v.GetInt("context_assembler.ca3.compaction.evidence.recent_window")
	cfg.Security.Scan.Mode = strings.ToLower(strings.TrimSpace(v.GetString("security.scan.mode")))
	cfg.Security.Scan.GovulncheckEnable = v.GetBool("security.scan.govulncheck_enabled")
	cfg.Security.Redaction.Enabled = v.GetBool("security.redaction.enabled")
	cfg.Security.Redaction.Strategy = strings.ToLower(strings.TrimSpace(v.GetString("security.redaction.strategy")))
	cfg.Security.Redaction.Keywords = normalizeKeywords(v.GetStringSlice("security.redaction.keywords"))
	cfg.Security.ToolGovernance.Enabled = v.GetBool("security.tool_governance.enabled")
	cfg.Security.ToolGovernance.Mode = strings.ToLower(strings.TrimSpace(v.GetString("security.tool_governance.mode")))
	cfg.Security.ToolGovernance.Permission.Default = strings.ToLower(strings.TrimSpace(v.GetString("security.tool_governance.permission.default")))
	cfg.Security.ToolGovernance.Permission.DenyAction = strings.ToLower(strings.TrimSpace(v.GetString("security.tool_governance.permission.deny_action")))
	cfg.Security.ToolGovernance.Permission.ByTool = normalizeNamespaceToolPolicyMap(v.GetStringMapString("security.tool_governance.permission.by_tool"))
	cfg.Security.ToolGovernance.RateLimit.Enabled = v.GetBool("security.tool_governance.rate_limit.enabled")
	cfg.Security.ToolGovernance.RateLimit.Scope = strings.ToLower(strings.TrimSpace(v.GetString("security.tool_governance.rate_limit.scope")))
	cfg.Security.ToolGovernance.RateLimit.Window = v.GetDuration("security.tool_governance.rate_limit.window")
	cfg.Security.ToolGovernance.RateLimit.Limit = v.GetInt("security.tool_governance.rate_limit.limit")
	cfg.Security.ToolGovernance.RateLimit.ByToolLimit = normalizeNamespaceToolIntMap(v.GetStringMap("security.tool_governance.rate_limit.by_tool_limit"))
	cfg.Security.ToolGovernance.RateLimit.ExceedAction = strings.ToLower(strings.TrimSpace(v.GetString("security.tool_governance.rate_limit.exceed_action")))
	cfg.Security.ModelIOFiltering.Enabled = v.GetBool("security.model_io_filtering.enabled")
	cfg.Security.ModelIOFiltering.RequireRegisteredFilter = v.GetBool("security.model_io_filtering.require_registered_filter")
	cfg.Security.ModelIOFiltering.Input.Enabled = v.GetBool("security.model_io_filtering.input.enabled")
	cfg.Security.ModelIOFiltering.Input.BlockAction = strings.ToLower(strings.TrimSpace(v.GetString("security.model_io_filtering.input.block_action")))
	cfg.Security.ModelIOFiltering.Output.Enabled = v.GetBool("security.model_io_filtering.output.enabled")
	cfg.Security.ModelIOFiltering.Output.BlockAction = strings.ToLower(strings.TrimSpace(v.GetString("security.model_io_filtering.output.block_action")))
	cfg.Security.SecurityEvent.Enabled = v.GetBool("security.security_event.enabled")
	cfg.Security.SecurityEvent.Alert.TriggerPolicy = strings.ToLower(strings.TrimSpace(v.GetString("security.security_event.alert.trigger_policy")))
	cfg.Security.SecurityEvent.Alert.Sink = strings.ToLower(strings.TrimSpace(v.GetString("security.security_event.alert.sink")))
	cfg.Security.SecurityEvent.Alert.Callback.RequireRegistered = v.GetBool("security.security_event.alert.callback.require_registered")
	cfg.Security.SecurityEvent.Delivery.Mode = strings.ToLower(strings.TrimSpace(v.GetString("security.security_event.delivery.mode")))
	cfg.Security.SecurityEvent.Delivery.Queue.Size = v.GetInt("security.security_event.delivery.queue.size")
	cfg.Security.SecurityEvent.Delivery.Queue.OverflowPolicy = strings.ToLower(strings.TrimSpace(v.GetString("security.security_event.delivery.queue.overflow_policy")))
	cfg.Security.SecurityEvent.Delivery.Timeout = v.GetDuration("security.security_event.delivery.timeout")
	cfg.Security.SecurityEvent.Delivery.Retry.MaxAttempts = v.GetInt("security.security_event.delivery.retry.max_attempts")
	cfg.Security.SecurityEvent.Delivery.Retry.BackoffInitial = v.GetDuration("security.security_event.delivery.retry.backoff_initial")
	cfg.Security.SecurityEvent.Delivery.Retry.BackoffMax = v.GetDuration("security.security_event.delivery.retry.backoff_max")
	cfg.Security.SecurityEvent.Delivery.CircuitBreaker.FailureThreshold = v.GetInt("security.security_event.delivery.circuit_breaker.failure_threshold")
	cfg.Security.SecurityEvent.Delivery.CircuitBreaker.OpenWindow = v.GetDuration("security.security_event.delivery.circuit_breaker.open_window")
	cfg.Security.SecurityEvent.Delivery.CircuitBreaker.HalfOpenProbes = v.GetInt("security.security_event.delivery.circuit_breaker.half_open_probes")
	cfg.Security.SecurityEvent.Severity.Default = strings.ToLower(strings.TrimSpace(v.GetString("security.security_event.severity.default")))
	cfg.Security.SecurityEvent.Severity.ByPolicyKind = normalizeStringToPolicyMap(v.GetStringMapString("security.security_event.severity.by_policy_kind"))
	cfg.Security.SecurityEvent.Severity.ByReasonCode = normalizeStringToPolicyMap(v.GetStringMapString("security.security_event.severity.by_reason_code"))

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
	explicitOverride := false
	if endpoint := strings.TrimSpace(v.GetString("context_assembler.ca2.stage2.external.endpoint")); endpoint != "" {
		out.Endpoint = endpoint
		explicitOverride = true
	}
	if method := strings.ToUpper(strings.TrimSpace(v.GetString("context_assembler.ca2.stage2.external.method"))); method != "" {
		out.Method = method
		explicitOverride = true
	}
	if token := strings.TrimSpace(v.GetString("context_assembler.ca2.stage2.external.auth.bearer_token")); token != "" {
		out.Auth.BearerToken = token
		explicitOverride = true
	}
	if header := strings.TrimSpace(v.GetString("context_assembler.ca2.stage2.external.auth.header_name")); header != "" {
		out.Auth.HeaderName = header
		explicitOverride = true
	}
	if headers := normalizeStringMap(v.GetStringMapString("context_assembler.ca2.stage2.external.headers")); len(headers) > 0 {
		out.Headers = headers
		explicitOverride = true
	}
	if mode := strings.ToLower(strings.TrimSpace(v.GetString("context_assembler.ca2.stage2.external.mapping.request.mode"))); mode != "" {
		out.Mapping.Request.Mode = mode
		explicitOverride = true
	}
	if methodName := strings.TrimSpace(v.GetString("context_assembler.ca2.stage2.external.mapping.request.method_name")); methodName != "" {
		out.Mapping.Request.MethodName = methodName
		explicitOverride = true
	}
	if version := strings.TrimSpace(v.GetString("context_assembler.ca2.stage2.external.mapping.request.jsonrpc_version")); version != "" {
		out.Mapping.Request.JSONRPCVersion = version
		explicitOverride = true
	}
	if queryField := strings.TrimSpace(v.GetString("context_assembler.ca2.stage2.external.mapping.request.query_field")); queryField != "" {
		out.Mapping.Request.QueryField = queryField
		explicitOverride = true
	}
	if sessionField := strings.TrimSpace(v.GetString("context_assembler.ca2.stage2.external.mapping.request.session_id_field")); sessionField != "" {
		out.Mapping.Request.SessionIDField = sessionField
		explicitOverride = true
	}
	if runField := strings.TrimSpace(v.GetString("context_assembler.ca2.stage2.external.mapping.request.run_id_field")); runField != "" {
		out.Mapping.Request.RunIDField = runField
		explicitOverride = true
	}
	if maxField := strings.TrimSpace(v.GetString("context_assembler.ca2.stage2.external.mapping.request.max_items_field")); maxField != "" {
		out.Mapping.Request.MaxItemsField = maxField
		explicitOverride = true
	}
	if chunksField := strings.TrimSpace(v.GetString("context_assembler.ca2.stage2.external.mapping.response.chunks_field")); chunksField != "" {
		out.Mapping.Response.ChunksField = chunksField
		explicitOverride = true
	}
	if sourceField := strings.TrimSpace(v.GetString("context_assembler.ca2.stage2.external.mapping.response.source_field")); sourceField != "" {
		out.Mapping.Response.SourceField = sourceField
		explicitOverride = true
	}
	if reasonField := strings.TrimSpace(v.GetString("context_assembler.ca2.stage2.external.mapping.response.reason_field")); reasonField != "" {
		out.Mapping.Response.ReasonField = reasonField
		explicitOverride = true
	}
	if errorField := strings.TrimSpace(v.GetString("context_assembler.ca2.stage2.external.mapping.response.error_field")); errorField != "" {
		out.Mapping.Response.ErrorField = errorField
		explicitOverride = true
	}
	if messageField := strings.TrimSpace(v.GetString("context_assembler.ca2.stage2.external.mapping.response.error_message_field")); messageField != "" {
		out.Mapping.Response.ErrorMessageField = messageField
		explicitOverride = true
	}
	out.Hints.Enabled = v.GetBool("context_assembler.ca2.stage2.external.hints.enabled")
	out.Hints.Capabilities = normalizeHintCapabilities(v.GetStringSlice("context_assembler.ca2.stage2.external.hints.capabilities"))
	out.TemplateResolutionSource = resolveStage2TemplateResolutionSource(out.Profile, explicitOverride)
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

func normalizeNamespaceToolPolicyMap(in map[string]string) map[string]string {
	if len(in) == 0 {
		return map[string]string{}
	}
	out := make(map[string]string, len(in))
	for rawKey, rawValue := range in {
		key := strings.ToLower(strings.TrimSpace(rawKey))
		value := strings.ToLower(strings.TrimSpace(rawValue))
		if key == "" || value == "" {
			continue
		}
		out[key] = value
	}
	return out
}

func normalizeNamespaceToolIntMap(in map[string]any) map[string]int {
	if len(in) == 0 {
		return map[string]int{}
	}
	out := make(map[string]int, len(in))
	for rawKey, rawValue := range in {
		key := strings.ToLower(strings.TrimSpace(rawKey))
		if key == "" {
			continue
		}
		switch tv := rawValue.(type) {
		case int:
			out[key] = tv
		case int64:
			out[key] = int(tv)
		case int32:
			out[key] = int(tv)
		case float64:
			out[key] = int(tv)
		case float32:
			out[key] = int(tv)
		case string:
			var parsed int
			if _, err := fmt.Sscanf(strings.TrimSpace(tv), "%d", &parsed); err == nil {
				out[key] = parsed
			}
		}
	}
	return out
}

func normalizePriorityMap(in map[string]string) map[string]string {
	if len(in) == 0 {
		return map[string]string{}
	}
	out := make(map[string]string, len(in))
	for rawKey, rawValue := range in {
		key := strings.ToLower(strings.TrimSpace(rawKey))
		value := strings.ToLower(strings.TrimSpace(rawValue))
		if key == "" || value == "" {
			continue
		}
		out[key] = value
	}
	return out
}

func normalizeFloatMap(in map[string]any) map[string]float64 {
	if len(in) == 0 {
		return map[string]float64{}
	}
	out := make(map[string]float64, len(in))
	for rawKey, rawValue := range in {
		key := strings.ToLower(strings.TrimSpace(rawKey))
		if key == "" {
			continue
		}
		switch tv := rawValue.(type) {
		case float64:
			out[key] = tv
		case float32:
			out[key] = float64(tv)
		case int:
			out[key] = float64(tv)
		case int64:
			out[key] = float64(tv)
		case int32:
			out[key] = float64(tv)
		case uint:
			out[key] = float64(tv)
		case uint64:
			out[key] = float64(tv)
		case uint32:
			out[key] = float64(tv)
		case string:
			var parsed float64
			if _, err := fmt.Sscanf(strings.TrimSpace(tv), "%f", &parsed); err == nil {
				out[key] = parsed
			}
		}
	}
	return out
}

func normalizeActionGateParameterRules(raw any) []types.ActionGateParameterRule {
	if raw == nil {
		return nil
	}
	decode := func(src any) []types.ActionGateParameterRule {
		b, err := json.Marshal(src)
		if err != nil {
			return nil
		}
		out := make([]types.ActionGateParameterRule, 0)
		if err := json.Unmarshal(b, &out); err != nil {
			return nil
		}
		for i := range out {
			out[i].ID = strings.TrimSpace(out[i].ID)
			out[i].ToolNames = normalizeKeywords(out[i].ToolNames)
			out[i].Action = types.ActionGateDecision(strings.ToLower(strings.TrimSpace(string(out[i].Action))))
			normalizeActionGateCondition(&out[i].Condition)
		}
		return out
	}
	switch tv := raw.(type) {
	case string:
		trimmed := strings.TrimSpace(tv)
		if trimmed == "" {
			return nil
		}
		out := decode(json.RawMessage(trimmed))
		if len(out) > 0 {
			return out
		}
		return nil
	default:
		return decode(raw)
	}
}

func normalizeActionGateCondition(c *types.ActionGateRuleCondition) {
	if c == nil {
		return
	}
	c.Path = strings.TrimSpace(c.Path)
	c.Operator = types.ActionGateRuleOperator(strings.ToLower(strings.TrimSpace(string(c.Operator))))
	for i := range c.All {
		normalizeActionGateCondition(&c.All[i])
	}
	for i := range c.Any {
		normalizeActionGateCondition(&c.Any[i])
	}
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
