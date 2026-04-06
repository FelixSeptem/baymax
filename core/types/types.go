package types

import (
	"context"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type Runner interface {
	Run(ctx context.Context, req RunRequest, h EventHandler) (RunResult, error)
	Stream(ctx context.Context, req RunRequest, h EventHandler) (RunResult, error)
}

type ModelClient interface {
	Generate(ctx context.Context, req ModelRequest) (ModelResponse, error)
	Stream(ctx context.Context, req ModelRequest, onEvent func(ModelEvent) error) error
}

type ActionGateDecision string

const (
	ActionGateDecisionAllow          ActionGateDecision = "allow"
	ActionGateDecisionRequireConfirm ActionGateDecision = "require_confirm"
	ActionGateDecisionDeny           ActionGateDecision = "deny"
)

type ActionGateCheck struct {
	RunID     string         `json:"run_id,omitempty"`
	SessionID string         `json:"session_id,omitempty"`
	Iteration int            `json:"iteration,omitempty"`
	CallID    string         `json:"call_id,omitempty"`
	ToolName  string         `json:"tool_name,omitempty"`
	Input     string         `json:"input,omitempty"`
	Args      map[string]any `json:"args,omitempty"`
}

type ActionGateMatcher interface {
	Evaluate(ctx context.Context, check ActionGateCheck) (ActionGateDecision, error)
}

type ActionGateConfirmRequest struct {
	Check   ActionGateCheck `json:"check"`
	Timeout time.Duration   `json:"timeout"`
}

type ActionGateResolver interface {
	Confirm(ctx context.Context, req ActionGateConfirmRequest) (bool, error)
}

type SecurityFilterDecision string

const (
	SecurityFilterDecisionAllow SecurityFilterDecision = "allow"
	SecurityFilterDecisionMatch SecurityFilterDecision = "match"
	SecurityFilterDecisionDeny  SecurityFilterDecision = "deny"
)

type SecurityFilterResult struct {
	Decision   SecurityFilterDecision `json:"decision,omitempty"`
	ReasonCode string                 `json:"reason_code,omitempty"`
}

type SecurityEvent struct {
	EventID       string    `json:"event_id,omitempty"`
	RunID         string    `json:"run_id,omitempty"`
	Iteration     int       `json:"iteration,omitempty"`
	PolicyKind    string    `json:"policy_kind,omitempty"`
	NamespaceTool string    `json:"namespace_tool,omitempty"`
	FilterStage   string    `json:"filter_stage,omitempty"`
	Decision      string    `json:"decision,omitempty"`
	ReasonCode    string    `json:"reason_code,omitempty"`
	Severity      string    `json:"severity,omitempty"`
	Timestamp     time.Time `json:"timestamp,omitempty"`
}

type SecurityAlertCallback func(ctx context.Context, event SecurityEvent) error

type SandboxSessionMode string

const (
	SandboxSessionModePerCall    SandboxSessionMode = "per_call"
	SandboxSessionModePerSession SandboxSessionMode = "per_session"
)

const (
	SandboxViolationTimeout            = "sandbox.timeout"
	SandboxViolationOOM                = "sandbox.oom"
	SandboxViolationNetwork            = "sandbox.network_violation"
	SandboxViolationMount              = "sandbox.mount_violation"
	SandboxViolationCapabilityMismatch = "sandbox.capability_mismatch"
	SandboxViolationLaunchFailed       = "sandbox.runtime_launch_failed"
)

type SandboxMount struct {
	Source   string `json:"source,omitempty"`
	Target   string `json:"target,omitempty"`
	ReadOnly bool   `json:"read_only,omitempty"`
}

type SandboxResourceLimits struct {
	CPUMilli    int64 `json:"cpu_milli,omitempty"`
	MemoryBytes int64 `json:"memory_bytes,omitempty"`
	PIDLimit    int   `json:"pid_limit,omitempty"`
}

type SandboxNetworkPolicy struct {
	Mode            string   `json:"mode,omitempty"`
	EgressAllowlist []string `json:"egress_allowlist,omitempty"`
}

type SandboxExecSpec struct {
	NamespaceTool  string                `json:"namespace_tool,omitempty"`
	Command        string                `json:"command,omitempty"`
	Args           []string              `json:"args,omitempty"`
	Env            map[string]string     `json:"env,omitempty"`
	Workdir        string                `json:"workdir,omitempty"`
	Mounts         []SandboxMount        `json:"mounts,omitempty"`
	Network        SandboxNetworkPolicy  `json:"network"`
	ResourceLimits SandboxResourceLimits `json:"resource_limits"`
	SessionMode    SandboxSessionMode    `json:"session_mode,omitempty"`
	LaunchTimeout  time.Duration         `json:"launch_timeout,omitempty"`
	ExecTimeout    time.Duration         `json:"exec_timeout,omitempty"`
}

type SandboxResourceUsage struct {
	CPUTimeMs       int64 `json:"cpu_time_ms,omitempty"`
	MemoryPeakBytes int64 `json:"memory_peak_bytes,omitempty"`
}

type SandboxExecResult struct {
	ExitCode       int                  `json:"exit_code"`
	Stdout         string               `json:"stdout,omitempty"`
	Stderr         string               `json:"stderr,omitempty"`
	TimedOut       bool                 `json:"timed_out,omitempty"`
	OOMKilled      bool                 `json:"oom_killed,omitempty"`
	LaunchFailed   bool                 `json:"launch_failed,omitempty"`
	ViolationCodes []string             `json:"violation_codes,omitempty"`
	ResourceUsage  SandboxResourceUsage `json:"resource_usage"`
}

type SandboxCapabilityProbe struct {
	Backend        string   `json:"backend,omitempty"`
	Capabilities   []string `json:"capabilities,omitempty"`
	SupportedModes []string `json:"supported_modes,omitempty"`
}

func (p SandboxCapabilityProbe) Supports(capability string) bool {
	needle := strings.ToLower(strings.TrimSpace(capability))
	for i := range p.Capabilities {
		if strings.ToLower(strings.TrimSpace(p.Capabilities[i])) == needle {
			return true
		}
	}
	return false
}

func (p SandboxCapabilityProbe) SupportsSessionMode(mode SandboxSessionMode) bool {
	needle := strings.ToLower(strings.TrimSpace(string(mode)))
	for i := range p.SupportedModes {
		if strings.ToLower(strings.TrimSpace(p.SupportedModes[i])) == needle {
			return true
		}
	}
	return false
}

type SandboxExecutor interface {
	Probe(ctx context.Context) (SandboxCapabilityProbe, error)
	Execute(ctx context.Context, spec SandboxExecSpec) (SandboxExecResult, error)
}

func NormalizeSandboxExecSpec(in SandboxExecSpec) (SandboxExecSpec, error) {
	out := in
	out.NamespaceTool = strings.ToLower(strings.TrimSpace(out.NamespaceTool))
	out.Command = strings.TrimSpace(out.Command)
	if out.Command == "" {
		return SandboxExecSpec{}, fmt.Errorf("sandbox exec command is required")
	}
	out.Workdir = strings.TrimSpace(out.Workdir)
	if out.Workdir != "" {
		abs, err := filepath.Abs(out.Workdir)
		if err != nil {
			return SandboxExecSpec{}, fmt.Errorf("sandbox exec workdir must be absolute or resolvable: %w", err)
		}
		out.Workdir = abs
	}
	if out.SessionMode == "" {
		out.SessionMode = SandboxSessionModePerCall
	}
	sanitizedEnv := map[string]string{}
	for key, value := range out.Env {
		name := strings.TrimSpace(key)
		if name == "" {
			continue
		}
		val := strings.TrimSpace(value)
		if val == "" {
			continue
		}
		sanitizedEnv[name] = val
	}
	if len(sanitizedEnv) == 0 {
		out.Env = nil
	} else {
		out.Env = sanitizedEnv
	}
	if len(out.Mounts) > 0 {
		mounts := make([]SandboxMount, 0, len(out.Mounts))
		for i := range out.Mounts {
			mount := out.Mounts[i]
			mount.Source = strings.TrimSpace(mount.Source)
			mount.Target = strings.TrimSpace(mount.Target)
			if mount.Source == "" || mount.Target == "" {
				return SandboxExecSpec{}, fmt.Errorf("sandbox mount[%d] source/target must not be empty", i)
			}
			mounts = append(mounts, mount)
		}
		sort.SliceStable(mounts, func(i, j int) bool {
			if mounts[i].Target != mounts[j].Target {
				return mounts[i].Target < mounts[j].Target
			}
			return mounts[i].Source < mounts[j].Source
		})
		out.Mounts = mounts
	}
	return out, nil
}

type ModelInputSecurityFilter interface {
	FilterModelInput(ctx context.Context, req ModelRequest) (ModelRequest, SecurityFilterResult, error)
}

type ModelOutputSecurityFilter interface {
	FilterModelOutput(ctx context.Context, output string) (string, SecurityFilterResult, error)
}

type ActionGateRuleOperator string

const (
	ActionGateRuleOperatorEQ       ActionGateRuleOperator = "eq"
	ActionGateRuleOperatorNE       ActionGateRuleOperator = "ne"
	ActionGateRuleOperatorContains ActionGateRuleOperator = "contains"
	ActionGateRuleOperatorRegex    ActionGateRuleOperator = "regex"
	ActionGateRuleOperatorIn       ActionGateRuleOperator = "in"
	ActionGateRuleOperatorNotIn    ActionGateRuleOperator = "not_in"
	ActionGateRuleOperatorGT       ActionGateRuleOperator = "gt"
	ActionGateRuleOperatorGTE      ActionGateRuleOperator = "gte"
	ActionGateRuleOperatorLT       ActionGateRuleOperator = "lt"
	ActionGateRuleOperatorLTE      ActionGateRuleOperator = "lte"
	ActionGateRuleOperatorExists   ActionGateRuleOperator = "exists"
)

type ActionGateRuleCondition struct {
	All      []ActionGateRuleCondition `json:"all,omitempty"`
	Any      []ActionGateRuleCondition `json:"any,omitempty"`
	Path     string                    `json:"path,omitempty"`
	Operator ActionGateRuleOperator    `json:"operator,omitempty"`
	Expected any                       `json:"expected,omitempty"`
}

type ActionGateParameterRule struct {
	ID        string                  `json:"id,omitempty"`
	ToolNames []string                `json:"tool_names,omitempty"`
	Condition ActionGateRuleCondition `json:"condition"`
	Action    ActionGateDecision      `json:"action,omitempty"`
}

type ClarificationTimeoutPolicy string

const (
	ClarificationTimeoutPolicyCancelByUser ClarificationTimeoutPolicy = "cancel_by_user"
)

type ClarificationRequest struct {
	RequestID      string        `json:"request_id"`
	Questions      []string      `json:"questions"`
	ContextSummary string        `json:"context_summary,omitempty"`
	Timeout        time.Duration `json:"timeout,omitempty"`
}

type ClarificationResponse struct {
	RequestID string         `json:"request_id,omitempty"`
	Answers   []string       `json:"answers,omitempty"`
	Meta      map[string]any `json:"meta,omitempty"`
}

type ClarificationResolveRequest struct {
	RunID       string               `json:"run_id,omitempty"`
	SessionID   string               `json:"session_id,omitempty"`
	Iteration   int                  `json:"iteration,omitempty"`
	Request     ClarificationRequest `json:"request"`
	Timeout     time.Duration        `json:"timeout"`
	TriggeredBy string               `json:"triggered_by,omitempty"`
}

type ClarificationResolver interface {
	Resolve(ctx context.Context, req ClarificationResolveRequest) (ClarificationResponse, error)
}

type TokenCounter interface {
	CountTokens(ctx context.Context, req ModelRequest) (int, error)
}

type ModelCapability string

const (
	ModelCapabilityStreaming ModelCapability = "streaming"
	ModelCapabilityToolCall  ModelCapability = "tool_call"
)

type CapabilitySupport string

const (
	CapabilitySupportSupported   CapabilitySupport = "supported"
	CapabilitySupportUnsupported CapabilitySupport = "unsupported"
	CapabilitySupportUnknown     CapabilitySupport = "unknown"
)

type CapabilityRequirements struct {
	Required []ModelCapability `json:"required,omitempty"`
}

func (r CapabilityRequirements) IsEmpty() bool {
	return len(r.Required) == 0
}

func (r CapabilityRequirements) Normalized() []ModelCapability {
	if len(r.Required) == 0 {
		return nil
	}
	out := make([]ModelCapability, 0, len(r.Required))
	seen := make(map[ModelCapability]struct{}, len(r.Required))
	for _, cap := range r.Required {
		if cap == "" {
			continue
		}
		if _, ok := seen[cap]; ok {
			continue
		}
		seen[cap] = struct{}{}
		out = append(out, cap)
	}
	return out
}

type ProviderCapabilities struct {
	Provider  string                                `json:"provider"`
	Model     string                                `json:"model,omitempty"`
	Support   map[ModelCapability]CapabilitySupport `json:"support,omitempty"`
	Source    string                                `json:"source,omitempty"`
	CheckedAt time.Time                             `json:"checked_at,omitempty"`
}

func (c ProviderCapabilities) Missing(required []ModelCapability) []ModelCapability {
	if len(required) == 0 {
		return nil
	}
	out := make([]ModelCapability, 0, len(required))
	for _, cap := range required {
		status := c.Support[cap]
		if status != CapabilitySupportSupported {
			out = append(out, cap)
		}
	}
	return out
}

type ModelCapabilityDiscovery interface {
	ProviderName() string
	DiscoverCapabilities(ctx context.Context, req ModelRequest) (ProviderCapabilities, error)
}

type Tool interface {
	Name() string
	Description() string
	JSONSchema() map[string]any
	Invoke(ctx context.Context, args map[string]any) (ToolResult, error)
}

type MCPClient interface {
	ListTools(ctx context.Context) ([]MCPToolMeta, error)
	CallTool(ctx context.Context, name string, args map[string]any) (ToolResult, error)
	Close() error
}

type SkillLoader interface {
	Discover(ctx context.Context, root string) ([]SkillSpec, error)
	Compile(ctx context.Context, specs []SkillSpec, in SkillInput) (SkillBundle, error)
}

type AgentLifecyclePhase string

const (
	AgentLifecyclePhaseBeforeReasoning  AgentLifecyclePhase = "before_reasoning"
	AgentLifecyclePhaseAfterReasoning   AgentLifecyclePhase = "after_reasoning"
	AgentLifecyclePhaseBeforeActing     AgentLifecyclePhase = "before_acting"
	AgentLifecyclePhaseAfterActing      AgentLifecyclePhase = "after_acting"
	AgentLifecyclePhaseBeforePlanChange AgentLifecyclePhase = "before_plan_change"
	AgentLifecyclePhaseAfterPlanChange  AgentLifecyclePhase = "after_plan_change"
	AgentLifecyclePhaseBeforeReply      AgentLifecyclePhase = "before_reply"
	AgentLifecyclePhaseAfterReply       AgentLifecyclePhase = "after_reply"
)

type AgentLifecycleHookContext struct {
	RunID       string              `json:"run_id"`
	SessionID   string              `json:"session_id,omitempty"`
	Iteration   int                 `json:"iteration"`
	Stream      bool                `json:"stream"`
	Phase       AgentLifecyclePhase `json:"phase"`
	FinalAnswer string              `json:"final_answer,omitempty"`
	ToolCalls   []ToolCall          `json:"tool_calls,omitempty"`
	PlanID      string              `json:"plan_id,omitempty"`
	PlanVersion int                 `json:"plan_version,omitempty"`
	PlanAction  string              `json:"plan_action,omitempty"`
	PlanReason  string              `json:"plan_reason,omitempty"`
}

type AgentLifecycleHook interface {
	OnPhase(ctx context.Context, in AgentLifecycleHookContext) error
}

type AgentLifecycleHookFunc func(ctx context.Context, in AgentLifecycleHookContext) error

func (f AgentLifecycleHookFunc) OnPhase(ctx context.Context, in AgentLifecycleHookContext) error {
	if f == nil {
		return nil
	}
	return f(ctx, in)
}

type ToolInvokeFunc func(ctx context.Context, call ToolCall) (ToolResult, error)

type ToolMiddleware interface {
	Invoke(ctx context.Context, call ToolCall, next ToolInvokeFunc) (ToolResult, error)
}

type ToolMiddlewareFunc func(ctx context.Context, call ToolCall, next ToolInvokeFunc) (ToolResult, error)

func (f ToolMiddlewareFunc) Invoke(ctx context.Context, call ToolCall, next ToolInvokeFunc) (ToolResult, error) {
	if f == nil {
		if next == nil {
			return ToolResult{}, nil
		}
		return next(ctx, call)
	}
	return f(ctx, call, next)
}

type EventHandler interface {
	OnEvent(ctx context.Context, ev Event)
}

type LoopPolicy struct {
	MaxIterations            int           `json:"max_iterations"`
	MaxToolCallsPerIteration int           `json:"max_tool_calls_per_iteration"`
	ToolCallLimit            int           `json:"tool_call_limit"`
	StepTimeout              time.Duration `json:"step_timeout"`
	ModelRetry               int           `json:"model_retry"`
	ToolRetry                int           `json:"tool_retry"`
	ContinueOnToolError      bool          `json:"continue_on_tool_error"`
	LocalDispatch            LocalDispatchPolicy
}

func DefaultLoopPolicy() LoopPolicy {
	return LoopPolicy{
		MaxIterations:            12,
		MaxToolCallsPerIteration: 8,
		ToolCallLimit:            64,
		StepTimeout:              60 * time.Second,
		ModelRetry:               2,
		ToolRetry:                1,
		ContinueOnToolError:      false,
		LocalDispatch:            DefaultLocalDispatchPolicy(),
	}
}

type BackpressureMode string

const (
	BackpressureBlock           BackpressureMode = "block"
	BackpressureReject          BackpressureMode = "reject"
	BackpressureDropLowPriority BackpressureMode = "drop_low_priority"
)

type LocalDispatchPolicy struct {
	MaxWorkers   int              `json:"max_workers"`
	QueueSize    int              `json:"queue_size"`
	Backpressure BackpressureMode `json:"backpressure"`
}

func DefaultLocalDispatchPolicy() LocalDispatchPolicy {
	return LocalDispatchPolicy{
		MaxWorkers:   8,
		QueueSize:    32,
		Backpressure: BackpressureBlock,
	}
}

type MCPRuntimePolicy struct {
	CallTimeout   time.Duration    `json:"call_timeout"`
	Retry         int              `json:"retry"`
	Backoff       time.Duration    `json:"backoff"`
	QueueSize     int              `json:"queue_size"`
	Backpressure  BackpressureMode `json:"backpressure"`
	ReadPoolSize  int              `json:"read_pool_size"`
	WritePoolSize int              `json:"write_pool_size"`
}

func DefaultMCPRuntimePolicy() MCPRuntimePolicy {
	return MCPRuntimePolicy{
		CallTimeout:   10 * time.Second,
		Retry:         1,
		Backoff:       50 * time.Millisecond,
		QueueSize:     32,
		Backpressure:  BackpressureBlock,
		ReadPoolSize:  4,
		WritePoolSize: 1,
	}
}

type RunRequest struct {
	RunID                  string                 `json:"run_id,omitempty"`
	SessionID              string                 `json:"session_id,omitempty"`
	Realtime               *RealtimeRunRequest    `json:"realtime,omitempty"`
	Input                  string                 `json:"input"`
	Messages               []Message              `json:"messages,omitempty"`
	Policy                 *LoopPolicy            `json:"policy,omitempty"`
	Capabilities           CapabilityRequirements `json:"capabilities,omitempty"`
	ArbitrationRuleVersion string                 `json:"arbitration_rule_version,omitempty"`
}

type RunResult struct {
	RunID       string            `json:"run_id"`
	FinalAnswer string            `json:"final_answer,omitempty"`
	Iterations  int               `json:"iterations"`
	ToolCalls   []ToolCallSummary `json:"tool_calls,omitempty"`
	TokenUsage  TokenUsage        `json:"token_usage"`
	LatencyMs   int64             `json:"latency_ms"`
	Warnings    []string          `json:"warnings,omitempty"`
	Error       *ClassifiedError  `json:"error,omitempty"`
}

type ModelRequest struct {
	RunID        string                 `json:"run_id,omitempty"`
	Model        string                 `json:"model,omitempty"`
	Input        string                 `json:"input,omitempty"`
	Messages     []Message              `json:"messages,omitempty"`
	ToolResult   []ToolCallOutcome      `json:"tool_results,omitempty"`
	Capabilities CapabilityRequirements `json:"capabilities,omitempty"`
}

type ContextAssembleRequest struct {
	RunID         string                 `json:"run_id,omitempty"`
	SessionID     string                 `json:"session_id,omitempty"`
	PrefixVersion string                 `json:"prefix_version,omitempty"`
	ModelProvider string                 `json:"model_provider,omitempty"`
	Model         string                 `json:"model,omitempty"`
	Input         string                 `json:"input,omitempty"`
	Messages      []Message              `json:"messages,omitempty"`
	ToolResult    []ToolCallOutcome      `json:"tool_results,omitempty"`
	Capabilities  CapabilityRequirements `json:"capabilities,omitempty"`
	TokenCounter  TokenCounter           `json:"-"`
	ModelClient   ModelClient            `json:"-"`
}

type ContextAssembleResult struct {
	Prefix       PrefixMetadata `json:"prefix"`
	LatencyMs    int64          `json:"latency_ms"`
	Status       string         `json:"status"`
	GuardFailure string         `json:"guard_failure,omitempty"`
	Stage        AssembleStage  `json:"stage,omitempty"`
	Recap        RecapMetadata  `json:"recap,omitempty"`
}

type ContextReference struct {
	ID      string   `json:"id,omitempty"`
	Type    string   `json:"type,omitempty"`
	Locator string   `json:"locator,omitempty"`
	Source  string   `json:"source,omitempty"`
	Summary string   `json:"summary,omitempty"`
	Tags    []string `json:"tags,omitempty"`
}

type ContextReferenceDiscoveryPayload struct {
	References       []ContextReference `json:"references,omitempty"`
	Deduplicated     int                `json:"deduplicated,omitempty"`
	MaxRefsApplied   int                `json:"max_refs_applied,omitempty"`
	DiscoverFrom     string             `json:"discover_from,omitempty"`
	ReferenceVersion string             `json:"reference_version,omitempty"`
}

type ContextReferenceResolutionItem struct {
	Reference ContextReference `json:"reference"`
	Content   string           `json:"content,omitempty"`
	Tokens    int              `json:"tokens,omitempty"`
}

type ContextReferenceResolutionPayload struct {
	Resolved         []ContextReferenceResolutionItem `json:"resolved,omitempty"`
	Missing          []ContextReference               `json:"missing,omitempty"`
	BudgetUsedTokens int                              `json:"budget_used_tokens,omitempty"`
	MaxTokens        int                              `json:"max_tokens,omitempty"`
	Truncated        bool                             `json:"truncated,omitempty"`
}

type IsolateHandoffArtifact struct {
	ID      string `json:"id,omitempty"`
	Type    string `json:"type,omitempty"`
	Locator string `json:"locator,omitempty"`
	Body    string `json:"body,omitempty"`
}

type IsolateHandoffPayload struct {
	Summary      string                   `json:"summary,omitempty"`
	Artifacts    []IsolateHandoffArtifact `json:"artifacts,omitempty"`
	EvidenceRefs []ContextReference       `json:"evidence_refs,omitempty"`
	Confidence   float64                  `json:"confidence,omitempty"`
	TTL          int64                    `json:"ttl,omitempty"`
}

type IsolateHandoffIngestionPayload struct {
	Handoffs        []IsolateHandoffPayload `json:"handoffs,omitempty"`
	AcceptedTotal   int                     `json:"accepted_total,omitempty"`
	RejectedTotal   int                     `json:"rejected_total,omitempty"`
	RejectedReasons []string                `json:"rejected_reasons,omitempty"`
	DeferBody       bool                    `json:"defer_body,omitempty"`
	Version         string                  `json:"version,omitempty"`
}

type AssembleStageStatus string

const (
	AssembleStageStatusStage1Only AssembleStageStatus = "stage1_only"
	AssembleStageStatusStage2Used AssembleStageStatus = "stage2_used"
	AssembleStageStatusDegraded   AssembleStageStatus = "degraded"
	AssembleStageStatusBypass     AssembleStageStatus = "bypass"
	AssembleStageStatusFailed     AssembleStageStatus = "failed"
)

type RecapStatus string

const (
	RecapStatusDisabled  RecapStatus = "disabled"
	RecapStatusAppended  RecapStatus = "appended"
	RecapStatusTruncated RecapStatus = "truncated"
	RecapStatusFailed    RecapStatus = "failed"
)

type AssembleStage struct {
	Status                         AssembleStageStatus `json:"status,omitempty"`
	Stage2SkipReason               string              `json:"stage2_skip_reason,omitempty"`
	Stage2RouterMode               string              `json:"stage2_router_mode,omitempty"`
	Stage2RouterDecision           string              `json:"stage2_router_decision,omitempty"`
	Stage2RouterReason             string              `json:"stage2_router_reason,omitempty"`
	Stage2RouterLatencyMs          int64               `json:"stage2_router_latency_ms,omitempty"`
	Stage2RouterError              string              `json:"stage2_router_error,omitempty"`
	Stage1LatencyMs                int64               `json:"stage1_latency_ms,omitempty"`
	Stage2LatencyMs                int64               `json:"stage2_latency_ms,omitempty"`
	Stage2Provider                 string              `json:"stage2_provider,omitempty"`
	Stage2Profile                  string              `json:"stage2_profile,omitempty"`
	Stage2TemplateProfile          string              `json:"stage2_template_profile,omitempty"`
	Stage2TemplateResolutionSource string              `json:"stage2_template_resolution_source,omitempty"`
	Stage2HintApplied              bool                `json:"stage2_hint_applied,omitempty"`
	Stage2HintMismatchReason       string              `json:"stage2_hint_mismatch_reason,omitempty"`
	Stage2HitCount                 int                 `json:"stage2_hit_count,omitempty"`
	Stage2Source                   string              `json:"stage2_source,omitempty"`
	Stage2Reason                   string              `json:"stage2_reason,omitempty"`
	Stage2ReasonCode               string              `json:"stage2_reason_code,omitempty"`
	Stage2ErrorLayer               string              `json:"stage2_error_layer,omitempty"`
	MemoryScopeSelected            string              `json:"memory_scope_selected,omitempty"`
	MemoryBudgetUsed               int                 `json:"memory_budget_used,omitempty"`
	MemoryHits                     int                 `json:"memory_hits,omitempty"`
	MemoryRerankStats              map[string]int      `json:"memory_rerank_stats,omitempty"`
	MemoryLifecycleAction          string              `json:"memory_lifecycle_action,omitempty"`
	PressureZone                   string              `json:"pressure_zone,omitempty"`
	PressureReason                 string              `json:"pressure_reason,omitempty"`
	// PressureTriggerSource stores the concrete trigger branch selected by pressure governance logic.
	PressureTriggerSource             string           `json:"pressure_trigger_source,omitempty"`
	ZoneResidencyMs                   map[string]int64 `json:"zone_residency_ms,omitempty"`
	TriggerCounts                     map[string]int   `json:"trigger_counts,omitempty"`
	CompressionRatio                  float64          `json:"compression_ratio,omitempty"`
	SpillCount                        int              `json:"spill_count,omitempty"`
	SwapBackCount                     int              `json:"swap_back_count,omitempty"`
	CompactionMode                    string           `json:"compaction_mode,omitempty"`
	CompactionFallback                bool             `json:"compaction_fallback,omitempty"`
	CompactionFallbackReason          string           `json:"compaction_fallback_reason,omitempty"`
	CompactionQualityScore            float64          `json:"compaction_quality_score,omitempty"`
	CompactionQualityReason           string           `json:"compaction_quality_reason,omitempty"`
	CompactionEmbeddingProvider       string           `json:"compaction_embedding_provider,omitempty"`
	CompactionEmbeddingSimilarity     float64          `json:"compaction_embedding_similarity,omitempty"`
	CompactionEmbeddingContribution   float64          `json:"compaction_embedding_contribution,omitempty"`
	CompactionEmbeddingStatus         string           `json:"compaction_embedding_status,omitempty"`
	CompactionEmbeddingFallbackReason string           `json:"compaction_embedding_fallback_reason,omitempty"`
	CompactionRerankerUsed            bool             `json:"compaction_reranker_used,omitempty"`
	CompactionRerankerProvider        string           `json:"compaction_reranker_provider,omitempty"`
	CompactionRerankerModel           string           `json:"compaction_reranker_model,omitempty"`
	CompactionRerankerThresholdSource string           `json:"compaction_reranker_threshold_source,omitempty"`
	CompactionRerankerThresholdHit    bool             `json:"compaction_reranker_threshold_hit,omitempty"`
	CompactionRerankerFallbackReason  string           `json:"compaction_reranker_fallback_reason,omitempty"`
	CompactionRerankerProfileVersion  string           `json:"compaction_reranker_profile_version,omitempty"`
	CompactionRerankerRolloutHit      bool             `json:"compaction_reranker_rollout_hit,omitempty"`
	CompactionRerankerThresholdDrift  float64          `json:"compaction_reranker_threshold_drift,omitempty"`
	RetainedEvidenceCount             int              `json:"retained_evidence_count,omitempty"`
	ContextRefDiscoverCount           int              `json:"context_ref_discover_count,omitempty"`
	ContextRefResolveCount            int              `json:"context_ref_resolve_count,omitempty"`
	ContextEditEstimatedSavedTokens   int              `json:"context_edit_estimated_saved_tokens,omitempty"`
	ContextEditGateDecision           string           `json:"context_edit_gate_decision,omitempty"`
	ContextSwapbackRelevanceScore     float64          `json:"context_swapback_relevance_score,omitempty"`
	ContextLifecycleTierStats         map[string]int   `json:"context_lifecycle_tier_stats,omitempty"`
	ContextRecapSource                string           `json:"context_recap_source,omitempty"`
}

type TailRecap struct {
	Status    string   `json:"status,omitempty"`
	Decisions []string `json:"decisions,omitempty"`
	Todo      []string `json:"todo,omitempty"`
	Risks     []string `json:"risks,omitempty"`
}

type RecapMetadata struct {
	Status RecapStatus `json:"status,omitempty"`
	Tail   TailRecap   `json:"tail,omitempty"`
}

type PrefixMetadata struct {
	SessionID     string `json:"session_id,omitempty"`
	PrefixVersion string `json:"prefix_version,omitempty"`
	PrefixHash    string `json:"prefix_hash,omitempty"`
}

type ModelResponse struct {
	FinalAnswer          string                `json:"final_answer,omitempty"`
	ToolCalls            []ToolCall            `json:"tool_calls,omitempty"`
	ClarificationRequest *ClarificationRequest `json:"clarification_request,omitempty"`
	Usage                TokenUsage            `json:"usage"`
}

type ToolResult struct {
	Content    string           `json:"content,omitempty"`
	Structured map[string]any   `json:"structured,omitempty"`
	Error      *ClassifiedError `json:"error,omitempty"`
}

type ToolCall struct {
	CallID string         `json:"call_id"`
	Name   string         `json:"name"`
	Args   map[string]any `json:"args,omitempty"`
}

type ToolCallOutcome struct {
	CallID string     `json:"call_id"`
	Name   string     `json:"name"`
	Result ToolResult `json:"result"`
}

type ToolCallSummary struct {
	CallID string           `json:"call_id"`
	Name   string           `json:"name"`
	Error  *ClassifiedError `json:"error,omitempty"`
}

type TokenUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
	TotalTokens  int `json:"total_tokens"`
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ModelEvent struct {
	Type                 string                `json:"type"`
	TextDelta            string                `json:"text_delta,omitempty"`
	ToolCall             *ToolCall             `json:"tool_call,omitempty"`
	ClarificationRequest *ClarificationRequest `json:"clarification_request,omitempty"`
	Meta                 map[string]any        `json:"meta,omitempty"`
}

const (
	ModelEventTypeFinalAnswer            = "final_answer"
	ModelEventTypeToolCall               = "tool_call"
	ModelEventTypeResponseError          = "response.error"
	ModelEventTypeOutputTextDelta        = "response.output_text.delta"
	ModelEventTypeOutputTextDone         = "response.output_text.done"
	ModelEventTypeFunctionArgsDelta      = "response.function_call_arguments.delta"
	ModelEventTypeFunctionArgsDone       = "response.function_call_arguments.done"
	ModelEventTypeOutputItemAdded        = "response.output_item.added"
	ModelEventTypeOutputItemDone         = "response.output_item.done"
	ModelEventTypeResponseCompleted      = "response.completed"
	ModelEventTypeResponseInProgress     = "response.in_progress"
	ModelEventTypeResponseQueued         = "response.queued"
	ModelEventTypeResponseReasoningDelta = "response.reasoning_summary.delta"
	ModelEventTypeClarificationRequest   = "clarification_request"
)

type MCPToolMeta struct {
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	InputSchema map[string]any `json:"input_schema,omitempty"`
}

type SkillSpec struct {
	Name        string            `json:"name"`
	Path        string            `json:"path"`
	Description string            `json:"description,omitempty"`
	Triggers    []string          `json:"triggers,omitempty"`
	Priority    int               `json:"priority,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

type SkillInput struct {
	UserInput string            `json:"user_input,omitempty"`
	Context   map[string]string `json:"context,omitempty"`
}

type SkillBundle struct {
	SystemPromptFragments []string `json:"system_prompt_fragments,omitempty"`
	EnabledTools          []string `json:"enabled_tools,omitempty"`
	WorkflowHints         []string `json:"workflow_hints,omitempty"`
}

const EventSchemaVersionV1 = "v1"

const EventTypeActionTimeline = "action.timeline"

type ActionPhase string

const (
	ActionPhaseRun              ActionPhase = "run"
	ActionPhaseContextAssembler ActionPhase = "context_assembler"
	ActionPhaseModel            ActionPhase = "model"
	ActionPhaseTool             ActionPhase = "tool"
	ActionPhaseMCP              ActionPhase = "mcp"
	ActionPhaseSkill            ActionPhase = "skill"
	ActionPhaseHITL             ActionPhase = "hitl"
)

type ActionStatus string

const (
	ActionStatusPending   ActionStatus = "pending"
	ActionStatusRunning   ActionStatus = "running"
	ActionStatusSucceeded ActionStatus = "succeeded"
	ActionStatusFailed    ActionStatus = "failed"
	ActionStatusSkipped   ActionStatus = "skipped"
	ActionStatusCanceled  ActionStatus = "canceled"
)

type ActionTimelineEvent struct {
	RunID     string       `json:"run_id"`
	Iteration int          `json:"iteration,omitempty"`
	Phase     ActionPhase  `json:"phase"`
	Status    ActionStatus `json:"status"`
	Reason    string       `json:"reason,omitempty"`
	Sequence  int64        `json:"sequence"`
	Time      time.Time    `json:"time"`
}

type Event struct {
	Version   string         `json:"version"`
	Type      string         `json:"type"`
	RunID     string         `json:"run_id"`
	Iteration int            `json:"iteration,omitempty"`
	CallID    string         `json:"call_id,omitempty"`
	TraceID   string         `json:"trace_id,omitempty"`
	SpanID    string         `json:"span_id,omitempty"`
	Time      time.Time      `json:"time"`
	Payload   map[string]any `json:"payload,omitempty"`
}

type ErrorClass string

const (
	ErrModel          ErrorClass = "ErrModel"
	ErrTool           ErrorClass = "ErrTool"
	ErrMCP            ErrorClass = "ErrMCP"
	ErrSkill          ErrorClass = "ErrSkill"
	ErrContext        ErrorClass = "ErrContext"
	ErrSecurity       ErrorClass = "ErrSecurity"
	ErrPolicyTimeout  ErrorClass = "ErrPolicyTimeout"
	ErrIterationLimit ErrorClass = "ErrIterationLimit"
	ErrHITL           ErrorClass = "ErrHITL"
)

type ClassifiedError struct {
	Class     ErrorClass     `json:"class"`
	Message   string         `json:"message"`
	Retryable bool           `json:"retryable"`
	Details   map[string]any `json:"details,omitempty"`
}
