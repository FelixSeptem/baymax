package runner

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/FelixSeptem/baymax/core/types"
	obsTrace "github.com/FelixSeptem/baymax/observability/trace"
	"github.com/FelixSeptem/baymax/tool/local"
	"go.opentelemetry.io/otel/attribute"
)

type State string

const (
	StateInit       State = "Init"
	StateModelStep  State = "ModelStep"
	StateDecideNext State = "DecideNext"
	StateFinalize   State = "Finalize"
	StateAbort      State = "Abort"
)

type Option func(*Engine)

type Engine struct {
	model      types.ModelClient
	dispatcher *local.Dispatcher
	tracer     *obsTrace.Manager
	now        func() time.Time
	newRunID   func() string
}

func New(model types.ModelClient, opts ...Option) *Engine {
	e := &Engine{
		model:  model,
		tracer: obsTrace.NewManager("baymax/core/runner"),
		now:    time.Now,
		newRunID: func() string {
			return fmt.Sprintf("run-%d", time.Now().UnixNano())
		},
	}
	for _, opt := range opts {
		opt(e)
	}
	return e
}

func WithLocalRegistry(registry *local.Registry) Option {
	return func(e *Engine) {
		e.dispatcher = local.NewDispatcher(registry)
	}
}

func WithTraceManager(tracer *obsTrace.Manager) Option {
	return func(e *Engine) {
		if tracer != nil {
			e.tracer = tracer
		}
	}
}

func (e *Engine) Run(ctx context.Context, req types.RunRequest, h types.EventHandler) (types.RunResult, error) {
	policy := resolvePolicy(req.Policy)
	runID := req.RunID
	if runID == "" {
		runID = e.newRunID()
	}

	start := e.now()
	ctx, runSpan := e.tracer.StartRun(ctx, runID)
	defer runSpan.End()
	state := StateInit
	iteration := 0
	lastResponse := types.ModelResponse{}
	warnings := make([]string, 0)
	mergedCalls := make([]types.ToolCallSummary, 0)
	pendingOutcomes := make([]types.ToolCallOutcome, 0)
	var terminal *types.ClassifiedError
	var runErr error

	e.emit(ctx, h, types.Event{Version: "v1", Type: "run.started", RunID: runID, Time: start})

	for {
		switch state {
		case StateInit:
			if iteration >= policy.MaxIterations {
				terminal = classified(types.ErrIterationLimit, "max iterations reached", false)
				runErr = errors.New(terminal.Message)
				state = StateAbort
				continue
			}
			state = StateModelStep
		case StateModelStep:
			iteration++
			e.emit(ctx, h, types.Event{Version: "v1", Type: "model.requested", RunID: runID, Iteration: iteration, Time: e.now()})
			stepCtx, cancel := context.WithTimeout(ctx, policy.StepTimeout)
			modelCtx, modelSpan := e.tracer.StartStep(stepCtx, "model.generate", attribute.Int("iteration.index", iteration))
			resp, err := e.model.Generate(modelCtx, toModelRequest(runID, req, pendingOutcomes))
			modelSpan.End()
			cancel()
			pendingOutcomes = nil
			if err != nil {
				if errors.Is(err, context.DeadlineExceeded) || errors.Is(stepCtx.Err(), context.DeadlineExceeded) {
					terminal = classified(types.ErrPolicyTimeout, "model step timed out", true)
				} else {
					terminal = classified(types.ErrModel, err.Error(), false)
				}
				runErr = err
				state = StateAbort
				continue
			}
			lastResponse = resp
			e.emit(ctx, h, types.Event{Version: "v1", Type: "model.completed", RunID: runID, Iteration: iteration, Time: e.now()})
			if resp.FinalAnswer != "" && len(resp.ToolCalls) == 0 {
				state = StateFinalize
				continue
			}
			state = StateDecideNext
		case StateDecideNext:
			if iteration >= policy.MaxIterations {
				terminal = classified(types.ErrIterationLimit, "max iterations reached", false)
				runErr = errors.New(terminal.Message)
				state = StateAbort
				continue
			}
			if len(lastResponse.ToolCalls) == 0 {
				state = StateFinalize
				continue
			}
			if e.dispatcher == nil {
				warnings = append(warnings, "tool calls requested but tool runtime is not enabled")
				state = StateModelStep
				continue
			}
			dispatchCfg := local.DispatchConfig{
				MaxCalls:     policy.MaxToolCallsPerIteration,
				Concurrency:  policy.LocalDispatch.MaxWorkers,
				FailFast:     !policy.ContinueOnToolError,
				QueueSize:    policy.LocalDispatch.QueueSize,
				Backpressure: policy.LocalDispatch.Backpressure,
				Retry:        policy.ToolRetry,
			}
			e.emit(ctx, h, types.Event{
				Version:   "v1",
				Type:      "tool.dispatch.started",
				RunID:     runID,
				Iteration: iteration,
				Time:      e.now(),
				Payload: map[string]any{
					"fanout":       len(lastResponse.ToolCalls),
					"max_calls":    dispatchCfg.MaxCalls,
					"workers":      dispatchCfg.Concurrency,
					"queue_size":   dispatchCfg.QueueSize,
					"backpressure": dispatchCfg.Backpressure,
					"retry":        dispatchCfg.Retry,
				},
			})
			stepCtx, cancel := context.WithTimeout(ctx, policy.StepTimeout)
			toolCtx, toolSpan := e.tracer.StartStep(stepCtx, "tool.dispatch", attribute.Int("iteration.index", iteration))
			outcomes, err := e.dispatcher.Dispatch(toolCtx, lastResponse.ToolCalls, dispatchCfg)
			toolSpan.End()
			cancel()
			if err != nil && errors.Is(stepCtx.Err(), context.DeadlineExceeded) {
				terminal = classified(types.ErrPolicyTimeout, "tool dispatch timed out", true)
				runErr = stepCtx.Err()
				state = StateAbort
				continue
			}
			for _, o := range outcomes {
				mergedCalls = append(mergedCalls, types.ToolCallSummary{CallID: o.CallID, Name: o.Name, Error: o.Result.Error})
				if o.Result.Error != nil {
					warnings = append(warnings, o.Result.Error.Message)
				}
			}
			if err != nil {
				terminal = classified(types.ErrTool, err.Error(), false)
				runErr = err
				state = StateAbort
				continue
			}
			failed := 0
			for _, o := range outcomes {
				if o.Result.Error != nil {
					failed++
				}
			}
			e.emit(ctx, h, types.Event{
				Version:   "v1",
				Type:      "tool.dispatch.completed",
				RunID:     runID,
				Iteration: iteration,
				Time:      e.now(),
				Payload: map[string]any{
					"fanout":       len(lastResponse.ToolCalls),
					"completed":    len(outcomes),
					"failed":       failed,
					"backpressure": dispatchCfg.Backpressure,
				},
			})
			pendingOutcomes = outcomes
			state = StateModelStep
		case StateFinalize:
			result := types.RunResult{
				RunID:       runID,
				FinalAnswer: lastResponse.FinalAnswer,
				Iterations:  iteration,
				ToolCalls:   mergedCalls,
				TokenUsage:  lastResponse.Usage,
				LatencyMs:   e.now().Sub(start).Milliseconds(),
				Warnings:    warnings,
			}
			e.emit(ctx, h, types.Event{Version: "v1", Type: "run.finished", RunID: runID, Iteration: iteration, Time: e.now()})
			return result, nil
		case StateAbort:
			result := types.RunResult{
				RunID:      runID,
				Iterations: iteration,
				ToolCalls:  mergedCalls,
				LatencyMs:  e.now().Sub(start).Milliseconds(),
				Warnings:   warnings,
				Error:      terminal,
			}
			e.emit(ctx, h, types.Event{Version: "v1", Type: "run.finished", RunID: runID, Iteration: iteration, Time: e.now()})
			return result, runErr
		}
	}
}

func (e *Engine) Stream(ctx context.Context, req types.RunRequest, h types.EventHandler) (types.RunResult, error) {
	policy := resolvePolicy(req.Policy)
	runID := req.RunID
	if runID == "" {
		runID = e.newRunID()
	}
	start := e.now()
	ctx, runSpan := e.tracer.StartRun(ctx, runID)
	defer runSpan.End()
	iteration := 1
	final := ""
	usage := types.TokenUsage{}

	e.emit(ctx, h, types.Event{Version: "v1", Type: "run.started", RunID: runID, Time: start})
	e.emit(ctx, h, types.Event{Version: "v1", Type: "model.requested", RunID: runID, Iteration: iteration, Time: e.now()})

	stepCtx, cancel := context.WithTimeout(ctx, policy.StepTimeout)
	modelCtx, modelSpan := e.tracer.StartStep(stepCtx, "model.stream", attribute.Int("iteration.index", iteration))
	err := e.model.Stream(modelCtx, toModelRequest(runID, req, nil), func(ev types.ModelEvent) error {
		payload := map[string]any{
			"event_type": ev.Type,
			"delta":      ev.TextDelta,
		}
		if ev.ToolCall != nil {
			payload["tool_call"] = ev.ToolCall
		}
		if len(ev.Meta) > 0 {
			payload["meta"] = ev.Meta
		}
		e.emit(stepCtx, h, types.Event{
			Version:   "v1",
			Type:      "model.delta",
			RunID:     runID,
			Iteration: iteration,
			Time:      e.now(),
			Payload:   payload,
		})
		if ev.Type == "final_answer" || ev.Type == "response.output_text.delta" {
			final += ev.TextDelta
		}
		return nil
	})
	modelSpan.End()
	cancel()
	if err != nil {
		terminal := classified(types.ErrModel, err.Error(), false)
		if errors.Is(err, context.DeadlineExceeded) || errors.Is(stepCtx.Err(), context.DeadlineExceeded) {
			terminal = classified(types.ErrPolicyTimeout, "model stream timed out", true)
		}
		result := types.RunResult{
			RunID:      runID,
			Iterations: iteration,
			LatencyMs:  e.now().Sub(start).Milliseconds(),
			Error:      terminal,
		}
		e.emit(ctx, h, types.Event{Version: "v1", Type: "run.finished", RunID: runID, Iteration: iteration, Time: e.now()})
		return result, err
	}

	e.emit(ctx, h, types.Event{Version: "v1", Type: "model.completed", RunID: runID, Iteration: iteration, Time: e.now()})
	result := types.RunResult{
		RunID:       runID,
		FinalAnswer: final,
		Iterations:  iteration,
		TokenUsage:  usage,
		LatencyMs:   e.now().Sub(start).Milliseconds(),
	}
	e.emit(ctx, h, types.Event{Version: "v1", Type: "run.finished", RunID: runID, Iteration: iteration, Time: e.now()})
	return result, nil
}

func resolvePolicy(p *types.LoopPolicy) types.LoopPolicy {
	if p == nil {
		return types.DefaultLoopPolicy()
	}
	policy := *p
	def := types.DefaultLoopPolicy()
	if policy.LocalDispatch.MaxWorkers <= 0 {
		policy.LocalDispatch.MaxWorkers = def.LocalDispatch.MaxWorkers
	}
	if policy.LocalDispatch.QueueSize <= 0 {
		policy.LocalDispatch.QueueSize = def.LocalDispatch.QueueSize
	}
	if policy.LocalDispatch.Backpressure == "" {
		policy.LocalDispatch.Backpressure = def.LocalDispatch.Backpressure
	}
	return policy
}

func toModelRequest(runID string, req types.RunRequest, outcomes []types.ToolCallOutcome) types.ModelRequest {
	return types.ModelRequest{
		RunID:      runID,
		Input:      req.Input,
		Messages:   req.Messages,
		ToolResult: outcomes,
	}
}

func classified(class types.ErrorClass, msg string, retryable bool) *types.ClassifiedError {
	return &types.ClassifiedError{Class: class, Message: msg, Retryable: retryable}
}

func (e *Engine) emit(ctx context.Context, h types.EventHandler, ev types.Event) {
	if h == nil {
		return
	}
	if ev.Version == "" {
		ev.Version = types.EventSchemaVersionV1
	}
	ev.TraceID = obsTrace.TraceIDFromContext(ctx)
	ev.SpanID = obsTrace.SpanIDFromContext(ctx)
	h.OnEvent(ctx, ev)
}

var _ types.Runner = (*Engine)(nil)
