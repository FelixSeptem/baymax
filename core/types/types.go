package types

import (
	"context"
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

type EventHandler interface {
	OnEvent(ctx context.Context, ev Event)
}

type LoopPolicy struct {
	MaxIterations            int           `json:"max_iterations"`
	MaxToolCallsPerIteration int           `json:"max_tool_calls_per_iteration"`
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
		StepTimeout:              60 * time.Second,
		ModelRetry:               2,
		ToolRetry:                1,
		ContinueOnToolError:      false,
		LocalDispatch:            DefaultLocalDispatchPolicy(),
	}
}

type BackpressureMode string

const (
	BackpressureBlock  BackpressureMode = "block"
	BackpressureReject BackpressureMode = "reject"
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
	RunID        string                 `json:"run_id,omitempty"`
	SessionID    string                 `json:"session_id,omitempty"`
	Input        string                 `json:"input"`
	Messages     []Message              `json:"messages,omitempty"`
	Policy       *LoopPolicy            `json:"policy,omitempty"`
	Capabilities CapabilityRequirements `json:"capabilities,omitempty"`
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
	Input         string                 `json:"input,omitempty"`
	Messages      []Message              `json:"messages,omitempty"`
	ToolResult    []ToolCallOutcome      `json:"tool_results,omitempty"`
	Capabilities  CapabilityRequirements `json:"capabilities,omitempty"`
}

type ContextAssembleResult struct {
	Prefix       PrefixMetadata `json:"prefix"`
	LatencyMs    int64          `json:"latency_ms"`
	Status       string         `json:"status"`
	GuardFailure string         `json:"guard_failure,omitempty"`
	Stage        AssembleStage  `json:"stage,omitempty"`
	Recap        RecapMetadata  `json:"recap,omitempty"`
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
	Status           AssembleStageStatus `json:"status,omitempty"`
	Stage2SkipReason string              `json:"stage2_skip_reason,omitempty"`
	Stage1LatencyMs  int64               `json:"stage1_latency_ms,omitempty"`
	Stage2LatencyMs  int64               `json:"stage2_latency_ms,omitempty"`
	Stage2Provider   string              `json:"stage2_provider,omitempty"`
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
	FinalAnswer string     `json:"final_answer,omitempty"`
	ToolCalls   []ToolCall `json:"tool_calls,omitempty"`
	Usage       TokenUsage `json:"usage"`
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
	Type      string         `json:"type"`
	TextDelta string         `json:"text_delta,omitempty"`
	ToolCall  *ToolCall      `json:"tool_call,omitempty"`
	Meta      map[string]any `json:"meta,omitempty"`
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
	ErrPolicyTimeout  ErrorClass = "ErrPolicyTimeout"
	ErrIterationLimit ErrorClass = "ErrIterationLimit"
)

type ClassifiedError struct {
	Class     ErrorClass     `json:"class"`
	Message   string         `json:"message"`
	Retryable bool           `json:"retryable"`
	Details   map[string]any `json:"details,omitempty"`
}
