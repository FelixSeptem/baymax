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
	RunID     string      `json:"run_id,omitempty"`
	SessionID string      `json:"session_id,omitempty"`
	Input     string      `json:"input"`
	Messages  []Message   `json:"messages,omitempty"`
	Policy    *LoopPolicy `json:"policy,omitempty"`
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
	RunID      string            `json:"run_id,omitempty"`
	Model      string            `json:"model,omitempty"`
	Input      string            `json:"input,omitempty"`
	Messages   []Message         `json:"messages,omitempty"`
	ToolResult []ToolCallOutcome `json:"tool_results,omitempty"`
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
	ErrPolicyTimeout  ErrorClass = "ErrPolicyTimeout"
	ErrIterationLimit ErrorClass = "ErrIterationLimit"
)

type ClassifiedError struct {
	Class     ErrorClass     `json:"class"`
	Message   string         `json:"message"`
	Retryable bool           `json:"retryable"`
	Details   map[string]any `json:"details,omitempty"`
}
