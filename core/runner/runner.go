package runner

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/FelixSeptem/baymax/context/assembler"
	"github.com/FelixSeptem/baymax/core/types"
	obsTrace "github.com/FelixSeptem/baymax/observability/trace"
	runtimeconfig "github.com/FelixSeptem/baymax/runtime/config"
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
	model              types.ModelClient
	models             map[string]types.ModelClient
	modelOrder         []string
	dispatcher         *local.Dispatcher
	tracer             *obsTrace.Manager
	runtimeMgr         *runtimeconfig.Manager
	assembler          *assembler.Assembler
	actionGateMatcher  types.ActionGateMatcher
	actionGateResolver types.ActionGateResolver
	now                func() time.Time
	newRunID           func() string
	capCacheMu         sync.RWMutex
	capCache           map[string]cachedCapabilities
}

type cachedCapabilities struct {
	report    types.ProviderCapabilities
	expiresAt time.Time
}

type stepModelSelection struct {
	Provider     string
	Initial      string
	Attempted    []string
	Missing      map[string][]types.ModelCapability
	Required     []types.ModelCapability
	UsedFallback bool
	Reason       string
}

type classifiedModelError interface {
	ClassifiedError() *types.ClassifiedError
}

func New(model types.ModelClient, opts ...Option) *Engine {
	e := &Engine{
		model:    model,
		models:   map[string]types.ModelClient{},
		tracer:   obsTrace.NewManager("baymax/core/runner"),
		now:      time.Now,
		capCache: map[string]cachedCapabilities{},
		newRunID: func() string {
			return fmt.Sprintf("run-%d", time.Now().UnixNano())
		},
	}
	e.assembler = assembler.New(
		func() runtimeconfig.ContextAssemblerConfig {
			if e.runtimeMgr != nil {
				return e.runtimeMgr.EffectiveConfig().ContextAssembler
			}
			// Keep legacy runner behavior when runtime manager is not provided.
			cfg := runtimeconfig.DefaultConfig().ContextAssembler
			cfg.Enabled = false
			return cfg
		},
		assembler.WithRedactionConfigProvider(func() runtimeconfig.SecurityRedactionConfig {
			if e.runtimeMgr != nil {
				return e.runtimeMgr.EffectiveConfig().Security.Redaction
			}
			return runtimeconfig.DefaultConfig().Security.Redaction
		}),
	)
	e.registerModel(model)
	for _, opt := range opts {
		opt(e)
	}
	return e
}

func WithLocalRegistry(registry *local.Registry) Option {
	return func(e *Engine) {
		e.dispatcher = local.NewDispatcher(registry)
		if e.runtimeMgr != nil {
			e.dispatcher.SetRuntimeManager(e.runtimeMgr)
		}
	}
}

func WithTraceManager(tracer *obsTrace.Manager) Option {
	return func(e *Engine) {
		if tracer != nil {
			e.tracer = tracer
		}
	}
}

func WithRuntimeManager(mgr *runtimeconfig.Manager) Option {
	return func(e *Engine) {
		e.runtimeMgr = mgr
		if e.dispatcher != nil {
			e.dispatcher.SetRuntimeManager(mgr)
		}
	}
}

func WithProviderModels(primary string, providers map[string]types.ModelClient) Option {
	return func(e *Engine) {
		for name, client := range providers {
			e.registerNamedModel(name, client)
		}
		if strings.TrimSpace(primary) != "" {
			e.model = e.models[strings.ToLower(strings.TrimSpace(primary))]
		}
	}
}

func WithActionGateMatcher(matcher types.ActionGateMatcher) Option {
	return func(e *Engine) {
		e.actionGateMatcher = matcher
	}
}

func WithActionGateResolver(resolver types.ActionGateResolver) Option {
	return func(e *Engine) {
		e.actionGateResolver = resolver
	}
}

func (e *Engine) Run(ctx context.Context, req types.RunRequest, h types.EventHandler) (types.RunResult, error) {
	policy := resolvePolicy(req.Policy)
	policy = e.applyRuntimeDefaults(policy, req.Policy)
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
	lastSelection := stepModelSelection{}
	selectionPath := make([]string, 0, 4)
	fallbackUsed := false
	lastAssemble := types.ContextAssembleResult{}
	timelineSeq := int64(0)
	gateStats := actionGateStats{}
	var terminal *types.ClassifiedError
	var runErr error

	e.emit(ctx, h, types.Event{Version: "v1", Type: "run.started", RunID: runID, Time: start})
	e.emitTimeline(ctx, h, runID, 0, &timelineSeq, types.ActionPhaseRun, types.ActionStatusPending, "")
	e.emitTimeline(ctx, h, runID, 0, &timelineSeq, types.ActionPhaseRun, types.ActionStatusRunning, "")

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
			required := req.Capabilities.Normalized()
			modelReq := toModelRequest(runID, req, pendingOutcomes, required)
			selectedModel, selection, selErr := e.selectModelForStep(ctx, modelReq, false, len(required) > 0)
			if selErr != nil {
				terminal = selErr
				runErr = errors.New(selErr.Message)
				state = StateAbort
				continue
			}
			contextPhaseEnabled := e.contextAssemblerEnabled()
			if contextPhaseEnabled {
				e.emitTimeline(ctx, h, runID, iteration, &timelineSeq, types.ActionPhaseContextAssembler, types.ActionStatusPending, "")
				e.emitTimeline(ctx, h, runID, iteration, &timelineSeq, types.ActionPhaseContextAssembler, types.ActionStatusRunning, "")
			}
			var tokenCounter types.TokenCounter
			if tc, ok := selectedModel.(types.TokenCounter); ok {
				tokenCounter = tc
			}
			assembledReq, assembleResult, assembleErr := e.assembler.Assemble(ctx, types.ContextAssembleRequest{
				RunID:         runID,
				SessionID:     req.SessionID,
				PrefixVersion: e.resolvePrefixVersion(),
				ModelProvider: selection.Provider,
				Model:         modelReq.Model,
				Input:         modelReq.Input,
				Messages:      modelReq.Messages,
				ToolResult:    modelReq.ToolResult,
				Capabilities:  modelReq.Capabilities,
				TokenCounter:  tokenCounter,
			}, modelReq)
			lastAssemble = assembleResult
			if assembleErr != nil {
				if contextPhaseEnabled {
					status, reason := classifyTimelineError(assembleErr)
					e.emitTimeline(ctx, h, runID, iteration, &timelineSeq, types.ActionPhaseContextAssembler, status, reason)
				}
				terminal = classified(types.ErrContext, assembleErr.Error(), false)
				runErr = assembleErr
				state = StateAbort
				continue
			}
			if contextPhaseEnabled {
				e.emitTimeline(ctx, h, runID, iteration, &timelineSeq, types.ActionPhaseContextAssembler, types.ActionStatusSucceeded, "")
			}
			modelReq = assembledReq
			e.emitTimeline(ctx, h, runID, iteration, &timelineSeq, types.ActionPhaseModel, types.ActionStatusPending, "")
			e.emitTimeline(ctx, h, runID, iteration, &timelineSeq, types.ActionPhaseModel, types.ActionStatusRunning, "")
			lastSelection = selection
			if selection.Provider != "" {
				selectionPath = append(selectionPath, selection.Provider)
			}
			fallbackUsed = fallbackUsed || selection.UsedFallback
			e.emit(ctx, h, types.Event{
				Version:   "v1",
				Type:      "model.requested",
				RunID:     runID,
				Iteration: iteration,
				Time:      e.now(),
				Payload: map[string]any{
					"model_provider": selection.Provider,
					"fallback_used":  selection.UsedFallback,
				},
			})
			stepCtx, cancel := context.WithTimeout(ctx, policy.StepTimeout)
			modelCtx, modelSpan := e.tracer.StartStep(stepCtx, "model.generate", attribute.Int("iteration.index", iteration))
			resp, err := selectedModel.Generate(modelCtx, modelReq)
			modelSpan.End()
			cancel()
			pendingOutcomes = nil
			if err != nil {
				var classifiedErr classifiedModelError
				var timelineStatus types.ActionStatus
				var reason string
				switch {
				case errors.As(err, &classifiedErr) && classifiedErr.ClassifiedError() != nil:
					terminal = classifiedErr.ClassifiedError()
					timelineStatus, reason = classifyClassifiedTimelineError(terminal)
				case errors.Is(err, context.DeadlineExceeded) || errors.Is(stepCtx.Err(), context.DeadlineExceeded):
					terminal = classified(types.ErrPolicyTimeout, "model step timed out", true)
					timelineStatus, reason = types.ActionStatusCanceled, "policy_timeout"
				default:
					terminal = classified(types.ErrModel, err.Error(), false)
					timelineStatus, reason = types.ActionStatusFailed, "model_error"
				}
				e.emitTimeline(ctx, h, runID, iteration, &timelineSeq, types.ActionPhaseModel, timelineStatus, reason)
				runErr = err
				state = StateAbort
				continue
			}
			lastResponse = resp
			e.emit(ctx, h, types.Event{Version: "v1", Type: "model.completed", RunID: runID, Iteration: iteration, Time: e.now()})
			e.emitTimeline(ctx, h, runID, iteration, &timelineSeq, types.ActionPhaseModel, types.ActionStatusSucceeded, "")
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
				e.emitTimeline(ctx, h, runID, iteration, &timelineSeq, types.ActionPhaseTool, types.ActionStatusSkipped, "tool_runtime_disabled")
				warnings = append(warnings, "tool calls requested but tool runtime is not enabled")
				state = StateModelStep
				continue
			}
			terminal, runErr = e.enforceActionGateForToolCalls(
				ctx,
				h,
				req,
				runID,
				iteration,
				&timelineSeq,
				lastResponse.ToolCalls,
				&gateStats,
			)
			if terminal != nil {
				state = StateAbort
				continue
			}
			e.emitTimeline(ctx, h, runID, iteration, &timelineSeq, types.ActionPhaseTool, types.ActionStatusPending, "")
			e.emitTimeline(ctx, h, runID, iteration, &timelineSeq, types.ActionPhaseTool, types.ActionStatusRunning, "")
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
				e.emitTimeline(ctx, h, runID, iteration, &timelineSeq, types.ActionPhaseTool, types.ActionStatusCanceled, "policy_timeout")
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
				e.emitTimeline(ctx, h, runID, iteration, &timelineSeq, types.ActionPhaseTool, types.ActionStatusFailed, "dispatch_error")
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
			e.emitTimeline(ctx, h, runID, iteration, &timelineSeq, types.ActionPhaseTool, types.ActionStatusSucceeded, "")
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
			e.emitTimeline(ctx, h, runID, iteration, &timelineSeq, types.ActionPhaseRun, types.ActionStatusSucceeded, "")
			e.emit(ctx, h, types.Event{
				Version:   "v1",
				Type:      "run.finished",
				RunID:     runID,
				Iteration: iteration,
				Time:      e.now(),
				Payload: runFinishedPayload(result, "success", "", runFinishMeta{
					Provider:     lastSelection.Provider,
					Initial:      lastSelection.Initial,
					Path:         selectionPath,
					Required:     lastSelection.Required,
					UsedFallback: fallbackUsed,
					Assemble:     lastAssemble,
					GateChecks:   gateStats.Checks,
					GateDenied:   gateStats.DeniedCount,
					GateTimeout:  gateStats.TimeoutCount,
				}),
			})
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
			errClass := ""
			if terminal != nil {
				errClass = string(terminal.Class)
			}
			status, reason := classifyRunTerminal(terminal, runErr)
			e.emitTimeline(ctx, h, runID, iteration, &timelineSeq, types.ActionPhaseRun, status, reason)
			e.emit(ctx, h, types.Event{
				Version:   "v1",
				Type:      "run.finished",
				RunID:     runID,
				Iteration: iteration,
				Time:      e.now(),
				Payload: runFinishedPayload(result, "failed", errClass, runFinishMeta{
					Provider:     lastSelection.Provider,
					Initial:      lastSelection.Initial,
					Path:         selectionPath,
					Required:     lastSelection.Required,
					Reason:       lastSelection.Reason,
					UsedFallback: fallbackUsed,
					Assemble:     lastAssemble,
					GateChecks:   gateStats.Checks,
					GateDenied:   gateStats.DeniedCount,
					GateTimeout:  gateStats.TimeoutCount,
				}),
			})
			return result, runErr
		}
	}
}

func (e *Engine) Stream(ctx context.Context, req types.RunRequest, h types.EventHandler) (types.RunResult, error) {
	policy := resolvePolicy(req.Policy)
	policy = e.applyRuntimeDefaults(policy, req.Policy)
	runID := req.RunID
	if runID == "" {
		runID = e.newRunID()
	}
	start := e.now()
	ctx, runSpan := e.tracer.StartRun(ctx, runID)
	defer runSpan.End()
	iteration := 1
	timelineSeq := int64(0)
	final := ""
	usage := types.TokenUsage{}
	selectionPath := make([]string, 0, 2)
	gateStats := actionGateStats{}
	required := append(req.Capabilities.Normalized(), types.ModelCapabilityStreaming)
	modelReq := toModelRequest(runID, req, nil, required)
	selectedModel, selection, selErr := e.selectModelForStep(ctx, modelReq, true, e.fallbackEnabled())
	if selErr != nil {
		result := types.RunResult{
			RunID:      runID,
			Iterations: iteration,
			LatencyMs:  e.now().Sub(start).Milliseconds(),
			Error:      selErr,
		}
		runTimelineStatus, runTimelineReason := classifyRunTerminal(result.Error, errors.New(selErr.Message))
		e.emit(ctx, h, types.Event{Version: "v1", Type: "run.started", RunID: runID, Time: start})
		e.emitTimeline(ctx, h, runID, 0, &timelineSeq, types.ActionPhaseRun, types.ActionStatusPending, "")
		e.emitTimeline(ctx, h, runID, 0, &timelineSeq, types.ActionPhaseRun, types.ActionStatusRunning, "")
		e.emitTimeline(ctx, h, runID, iteration, &timelineSeq, types.ActionPhaseRun, runTimelineStatus, runTimelineReason)
		e.emit(ctx, h, types.Event{
			Version:   "v1",
			Type:      "run.finished",
			RunID:     runID,
			Iteration: iteration,
			Time:      e.now(),
			Payload: runFinishedPayload(result, "failed", string(selErr.Class), runFinishMeta{
				Provider:     selection.Provider,
				Initial:      selection.Initial,
				Path:         selectionPath,
				Required:     required,
				UsedFallback: selection.UsedFallback,
				Reason:       selErr.Message,
				Assemble:     types.ContextAssembleResult{},
			}),
		})
		return result, errors.New(selErr.Message)
	}
	if selection.Provider != "" {
		selectionPath = append(selectionPath, selection.Provider)
	}
	var tokenCounter types.TokenCounter
	if tc, ok := selectedModel.(types.TokenCounter); ok {
		tokenCounter = tc
	}
	assembledReq, assembleResult, assembleErr := e.assembler.Assemble(ctx, types.ContextAssembleRequest{
		RunID:         runID,
		SessionID:     req.SessionID,
		PrefixVersion: e.resolvePrefixVersion(),
		ModelProvider: selection.Provider,
		Model:         modelReq.Model,
		Input:         modelReq.Input,
		Messages:      modelReq.Messages,
		ToolResult:    modelReq.ToolResult,
		Capabilities:  modelReq.Capabilities,
		TokenCounter:  tokenCounter,
	}, modelReq)
	lastAssemble := assembleResult

	e.emit(ctx, h, types.Event{Version: "v1", Type: "run.started", RunID: runID, Time: start})
	e.emitTimeline(ctx, h, runID, 0, &timelineSeq, types.ActionPhaseRun, types.ActionStatusPending, "")
	e.emitTimeline(ctx, h, runID, 0, &timelineSeq, types.ActionPhaseRun, types.ActionStatusRunning, "")
	contextPhaseEnabled := e.contextAssemblerEnabled()
	if contextPhaseEnabled {
		e.emitTimeline(ctx, h, runID, iteration, &timelineSeq, types.ActionPhaseContextAssembler, types.ActionStatusPending, "")
		e.emitTimeline(ctx, h, runID, iteration, &timelineSeq, types.ActionPhaseContextAssembler, types.ActionStatusRunning, "")
	}
	if assembleErr != nil {
		if contextPhaseEnabled {
			status, reason := classifyTimelineError(assembleErr)
			e.emitTimeline(ctx, h, runID, iteration, &timelineSeq, types.ActionPhaseContextAssembler, status, reason)
		}
		result := types.RunResult{
			RunID:      runID,
			Iterations: iteration,
			LatencyMs:  e.now().Sub(start).Milliseconds(),
			Error:      classified(types.ErrContext, assembleErr.Error(), false),
		}
		runTimelineStatus, runTimelineReason := classifyRunTerminal(result.Error, assembleErr)
		e.emitTimeline(ctx, h, runID, iteration, &timelineSeq, types.ActionPhaseRun, runTimelineStatus, runTimelineReason)
		e.emit(ctx, h, types.Event{
			Version:   "v1",
			Type:      "run.finished",
			RunID:     runID,
			Iteration: iteration,
			Time:      e.now(),
			Payload: runFinishedPayload(result, "failed", string(types.ErrContext), runFinishMeta{
				Provider:     "",
				Initial:      "",
				Path:         selectionPath,
				Required:     required,
				UsedFallback: false,
				Assemble:     lastAssemble,
			}),
		})
		return result, assembleErr
	}
	if contextPhaseEnabled {
		e.emitTimeline(ctx, h, runID, iteration, &timelineSeq, types.ActionPhaseContextAssembler, types.ActionStatusSucceeded, "")
	}
	modelReq = assembledReq
	e.emitTimeline(ctx, h, runID, iteration, &timelineSeq, types.ActionPhaseModel, types.ActionStatusPending, "")
	e.emitTimeline(ctx, h, runID, iteration, &timelineSeq, types.ActionPhaseModel, types.ActionStatusRunning, "")
	e.emit(ctx, h, types.Event{
		Version:   "v1",
		Type:      "model.requested",
		RunID:     runID,
		Iteration: iteration,
		Time:      e.now(),
		Payload: map[string]any{
			"model_provider": selection.Provider,
			"fallback_used":  selection.UsedFallback,
		},
	})

	stepCtx, cancel := context.WithTimeout(ctx, policy.StepTimeout)
	modelCtx, modelSpan := e.tracer.StartStep(stepCtx, "model.stream", attribute.Int("iteration.index", iteration))
	err := selectedModel.Stream(modelCtx, modelReq, func(ev types.ModelEvent) error {
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
		if ev.ToolCall != nil {
			e.emitTimeline(stepCtx, h, runID, iteration, &timelineSeq, types.ActionPhaseTool, types.ActionStatusPending, "")
			e.emitTimeline(stepCtx, h, runID, iteration, &timelineSeq, types.ActionPhaseTool, types.ActionStatusRunning, "")
			gateTerm, gateRunErr := e.enforceActionGateForToolCalls(
				stepCtx,
				h,
				req,
				runID,
				iteration,
				&timelineSeq,
				[]types.ToolCall{*ev.ToolCall},
				&gateStats,
			)
			if gateTerm != nil {
				return &actionGateViolationError{
					classified: gateTerm,
					err:        gateRunErr,
				}
			}
			e.emitTimeline(stepCtx, h, runID, iteration, &timelineSeq, types.ActionPhaseTool, types.ActionStatusSkipped, "stream_tool_dispatch_not_supported")
		}
		return nil
	})
	modelSpan.End()
	cancel()
	if err != nil {
		var classifiedErr classifiedModelError
		var gateErr *actionGateViolationError
		terminal := classified(types.ErrModel, err.Error(), false)
		var timelineStatus types.ActionStatus
		var reason string
		switch {
		case errors.As(err, &gateErr) && gateErr.ClassifiedError() != nil:
			terminal = gateErr.ClassifiedError()
			timelineStatus, reason = classifyClassifiedTimelineError(terminal)
		case errors.As(err, &classifiedErr) && classifiedErr.ClassifiedError() != nil:
			terminal = classifiedErr.ClassifiedError()
			timelineStatus, reason = classifyClassifiedTimelineError(terminal)
		case errors.Is(err, context.DeadlineExceeded) || errors.Is(stepCtx.Err(), context.DeadlineExceeded):
			terminal = classified(types.ErrPolicyTimeout, "model stream timed out", true)
			timelineStatus, reason = types.ActionStatusCanceled, "policy_timeout"
		}
		e.emitTimeline(ctx, h, runID, iteration, &timelineSeq, types.ActionPhaseModel, timelineStatus, reason)
		result := types.RunResult{
			RunID:      runID,
			Iterations: iteration,
			LatencyMs:  e.now().Sub(start).Milliseconds(),
			Error:      terminal,
		}
		errClass := ""
		if terminal != nil {
			errClass = string(terminal.Class)
		}
		status, runReason := classifyRunTerminal(terminal, err)
		e.emitTimeline(ctx, h, runID, iteration, &timelineSeq, types.ActionPhaseRun, status, runReason)
		e.emit(ctx, h, types.Event{
			Version:   "v1",
			Type:      "run.finished",
			RunID:     runID,
			Iteration: iteration,
			Time:      e.now(),
			Payload: runFinishedPayload(result, "failed", errClass, runFinishMeta{
				Provider:     selection.Provider,
				Initial:      selection.Initial,
				Path:         selectionPath,
				Required:     required,
				UsedFallback: selection.UsedFallback,
				Assemble:     lastAssemble,
				GateChecks:   gateStats.Checks,
				GateDenied:   gateStats.DeniedCount,
				GateTimeout:  gateStats.TimeoutCount,
			}),
		})
		return result, err
	}

	e.emit(ctx, h, types.Event{Version: "v1", Type: "model.completed", RunID: runID, Iteration: iteration, Time: e.now()})
	e.emitTimeline(ctx, h, runID, iteration, &timelineSeq, types.ActionPhaseModel, types.ActionStatusSucceeded, "")
	result := types.RunResult{
		RunID:       runID,
		FinalAnswer: final,
		Iterations:  iteration,
		TokenUsage:  usage,
		LatencyMs:   e.now().Sub(start).Milliseconds(),
	}
	e.emitTimeline(ctx, h, runID, iteration, &timelineSeq, types.ActionPhaseRun, types.ActionStatusSucceeded, "")
	e.emit(ctx, h, types.Event{
		Version:   "v1",
		Type:      "run.finished",
		RunID:     runID,
		Iteration: iteration,
		Time:      e.now(),
		Payload: runFinishedPayload(result, "success", "", runFinishMeta{
			Provider:     selection.Provider,
			Initial:      selection.Initial,
			Path:         selectionPath,
			Required:     required,
			UsedFallback: selection.UsedFallback,
			Assemble:     lastAssemble,
			GateChecks:   gateStats.Checks,
			GateDenied:   gateStats.DeniedCount,
			GateTimeout:  gateStats.TimeoutCount,
		}),
	})
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

func toModelRequest(runID string, req types.RunRequest, outcomes []types.ToolCallOutcome, required []types.ModelCapability) types.ModelRequest {
	return types.ModelRequest{
		RunID:      runID,
		Input:      req.Input,
		Messages:   req.Messages,
		ToolResult: outcomes,
		Capabilities: types.CapabilityRequirements{
			Required: required,
		},
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

func (e *Engine) emitTimeline(
	ctx context.Context,
	h types.EventHandler,
	runID string,
	iteration int,
	seq *int64,
	phase types.ActionPhase,
	status types.ActionStatus,
	reason string,
) {
	if seq == nil {
		return
	}
	*seq++
	payload := map[string]any{
		"phase":    string(phase),
		"status":   string(status),
		"sequence": *seq,
	}
	if strings.TrimSpace(reason) != "" {
		payload["reason"] = strings.TrimSpace(reason)
	}
	e.emit(ctx, h, types.Event{
		Version:   types.EventSchemaVersionV1,
		Type:      types.EventTypeActionTimeline,
		RunID:     runID,
		Iteration: iteration,
		Time:      e.now(),
		Payload:   payload,
	})
}

func (e *Engine) applyRuntimeDefaults(policy types.LoopPolicy, input *types.LoopPolicy) types.LoopPolicy {
	if e.runtimeMgr == nil || input != nil {
		return policy
	}
	cfg := e.runtimeMgr.EffectiveConfig()
	if cfg.Concurrency.LocalMaxWorkers > 0 {
		policy.LocalDispatch.MaxWorkers = cfg.Concurrency.LocalMaxWorkers
	}
	if cfg.Concurrency.LocalQueueSize > 0 {
		policy.LocalDispatch.QueueSize = cfg.Concurrency.LocalQueueSize
	}
	if cfg.Concurrency.Backpressure != "" {
		policy.LocalDispatch.Backpressure = cfg.Concurrency.Backpressure
	}
	return policy
}

func (e *Engine) registerModel(model types.ModelClient) {
	if model == nil {
		return
	}
	name := "default"
	if d, ok := model.(types.ModelCapabilityDiscovery); ok {
		if provider := strings.ToLower(strings.TrimSpace(d.ProviderName())); provider != "" {
			name = provider
		}
	}
	e.registerNamedModel(name, model)
}

func (e *Engine) registerNamedModel(name string, model types.ModelClient) {
	if model == nil {
		return
	}
	key := strings.ToLower(strings.TrimSpace(name))
	if key == "" {
		key = "default"
	}
	if e.models == nil {
		e.models = map[string]types.ModelClient{}
	}
	if _, exists := e.models[key]; !exists {
		e.modelOrder = append(e.modelOrder, key)
	}
	e.models[key] = model
}

func (e *Engine) selectModelForStep(ctx context.Context, req types.ModelRequest, stream bool, strict bool) (types.ModelClient, stepModelSelection, *types.ClassifiedError) {
	selection := stepModelSelection{
		Missing:  map[string][]types.ModelCapability{},
		Required: req.Capabilities.Normalized(),
	}
	if stream {
		hasStreaming := false
		for _, cap := range selection.Required {
			if cap == types.ModelCapabilityStreaming {
				hasStreaming = true
				break
			}
		}
		if !hasStreaming {
			selection.Required = append(selection.Required, types.ModelCapabilityStreaming)
		}
	}
	primaryName, primaryClient := e.primaryModel()
	if primaryClient == nil {
		return nil, selection, classified(types.ErrModel, "no model client configured", false)
	}
	selection.Initial = primaryName

	order := e.resolveProviderOrder(primaryName)
	if len(order) == 0 {
		order = []string{primaryName}
	}

	timeout := 1500 * time.Millisecond
	cacheTTL := 5 * time.Minute
	if e.runtimeMgr != nil {
		cfg := e.runtimeMgr.EffectiveConfig()
		if cfg.ProviderFallback.DiscoveryTimeout > 0 {
			timeout = cfg.ProviderFallback.DiscoveryTimeout
		}
		if cfg.ProviderFallback.DiscoveryCacheTTL > 0 {
			cacheTTL = cfg.ProviderFallback.DiscoveryCacheTTL
		}
	}

	for _, name := range order {
		client, ok := e.models[name]
		if !ok || client == nil {
			continue
		}
		selection.Attempted = append(selection.Attempted, name)
		if len(selection.Required) == 0 || !strict {
			selection.Provider = name
			selection.UsedFallback = name != selection.Initial
			return client, selection, nil
		}
		discovery, ok := client.(types.ModelCapabilityDiscovery)
		if !ok {
			selection.Missing[name] = append([]types.ModelCapability(nil), selection.Required...)
			continue
		}
		report, err := e.discoverCapabilities(ctx, name, discovery, req, timeout, cacheTTL)
		if err != nil {
			selection.Missing[name] = append([]types.ModelCapability(nil), selection.Required...)
			continue
		}
		missing := report.Missing(selection.Required)
		if len(missing) == 0 {
			selection.Provider = name
			selection.UsedFallback = name != selection.Initial
			return client, selection, nil
		}
		selection.Missing[name] = missing
	}

	required := make([]string, 0, len(selection.Required))
	for _, cap := range selection.Required {
		required = append(required, string(cap))
	}
	selection.Reason = "capability_preflight_failed"
	err := classified(types.ErrModel, "no provider satisfies required capabilities", false)
	err.Details = map[string]any{
		"provider_reason":       "capability_unsupported",
		"required_capabilities": strings.Join(required, ","),
		"attempted_providers":   strings.Join(selection.Attempted, ","),
	}
	return nil, selection, err
}

func (e *Engine) primaryModel() (string, types.ModelClient) {
	if d, ok := e.model.(types.ModelCapabilityDiscovery); ok {
		name := strings.ToLower(strings.TrimSpace(d.ProviderName()))
		if name != "" {
			if client, exists := e.models[name]; exists && client != nil {
				return name, client
			}
		}
	}
	if len(e.modelOrder) > 0 {
		name := e.modelOrder[0]
		return name, e.models[name]
	}
	return "default", e.model
}

func (e *Engine) resolveProviderOrder(primary string) []string {
	ordered := make([]string, 0, len(e.modelOrder))
	seen := map[string]struct{}{}
	appendName := func(name string) {
		n := strings.ToLower(strings.TrimSpace(name))
		if n == "" {
			return
		}
		if _, ok := seen[n]; ok {
			return
		}
		seen[n] = struct{}{}
		ordered = append(ordered, n)
	}
	appendName(primary)

	enabled := false
	if e.runtimeMgr != nil {
		cfg := e.runtimeMgr.EffectiveConfig()
		enabled = cfg.ProviderFallback.Enabled
		if enabled && len(cfg.ProviderFallback.Providers) > 0 {
			for _, provider := range cfg.ProviderFallback.Providers {
				appendName(provider)
			}
		}
	}
	if !enabled {
		return ordered
	}
	for _, name := range e.modelOrder {
		appendName(name)
	}
	return ordered
}

func (e *Engine) fallbackEnabled() bool {
	if e.runtimeMgr == nil {
		return false
	}
	return e.runtimeMgr.EffectiveConfig().ProviderFallback.Enabled
}

func (e *Engine) discoverCapabilities(
	ctx context.Context,
	provider string,
	discovery types.ModelCapabilityDiscovery,
	req types.ModelRequest,
	timeout time.Duration,
	cacheTTL time.Duration,
) (types.ProviderCapabilities, error) {
	cacheKey := provider + "|" + req.Model
	e.capCacheMu.RLock()
	cached, ok := e.capCache[cacheKey]
	e.capCacheMu.RUnlock()
	if ok && e.now().Before(cached.expiresAt) {
		return cached.report, nil
	}

	probeCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	report, err := discovery.DiscoverCapabilities(probeCtx, req)
	if err != nil {
		return types.ProviderCapabilities{}, err
	}
	if report.Provider == "" {
		report.Provider = provider
	}
	if report.CheckedAt.IsZero() {
		report.CheckedAt = e.now()
	}
	e.capCacheMu.Lock()
	e.capCache[cacheKey] = cachedCapabilities{
		report:    report,
		expiresAt: e.now().Add(cacheTTL),
	}
	e.capCacheMu.Unlock()
	return report, nil
}

type runFinishMeta struct {
	Provider     string
	Initial      string
	Path         []string
	Required     []types.ModelCapability
	UsedFallback bool
	Reason       string
	Assemble     types.ContextAssembleResult
	GateChecks   int
	GateDenied   int
	GateTimeout  int
}

func runFinishedPayload(result types.RunResult, status string, errClass string, meta runFinishMeta) map[string]any {
	payload := map[string]any{
		"status":      status,
		"latency_ms":  result.LatencyMs,
		"tool_calls":  len(result.ToolCalls),
		"iterations":  result.Iterations,
		"warning_cnt": len(result.Warnings),
	}
	if errClass != "" {
		payload["error_class"] = errClass
	}
	if meta.Provider != "" {
		payload["model_provider"] = meta.Provider
	}
	if meta.Initial != "" {
		payload["fallback_initial"] = meta.Initial
	}
	payload["fallback_used"] = meta.UsedFallback
	if len(meta.Path) > 0 {
		payload["fallback_path"] = strings.Join(meta.Path, "->")
	}
	if len(meta.Required) > 0 {
		required := make([]string, 0, len(meta.Required))
		for _, cap := range meta.Required {
			required = append(required, string(cap))
		}
		payload["required_capabilities"] = strings.Join(required, ",")
	}
	if meta.Reason != "" {
		payload["fallback_reason"] = meta.Reason
	}
	if meta.Assemble.Prefix.PrefixHash != "" {
		payload["prefix_hash"] = meta.Assemble.Prefix.PrefixHash
	}
	payload["assemble_latency_ms"] = meta.Assemble.LatencyMs
	if meta.Assemble.Status != "" {
		payload["assemble_status"] = meta.Assemble.Status
	}
	if meta.Assemble.GuardFailure != "" {
		payload["guard_violation"] = meta.Assemble.GuardFailure
	}
	if meta.Assemble.Stage.Status != "" {
		payload["assemble_stage_status"] = string(meta.Assemble.Stage.Status)
	}
	if meta.Assemble.Stage.Stage2SkipReason != "" {
		payload["stage2_skip_reason"] = meta.Assemble.Stage.Stage2SkipReason
	}
	if meta.Assemble.Stage.Stage1LatencyMs > 0 {
		payload["stage1_latency_ms"] = meta.Assemble.Stage.Stage1LatencyMs
	}
	if meta.Assemble.Stage.Stage2LatencyMs > 0 {
		payload["stage2_latency_ms"] = meta.Assemble.Stage.Stage2LatencyMs
	}
	if meta.Assemble.Stage.Stage2Provider != "" {
		payload["stage2_provider"] = meta.Assemble.Stage.Stage2Provider
	}
	if meta.Assemble.Stage.Stage2Profile != "" {
		payload["stage2_profile"] = meta.Assemble.Stage.Stage2Profile
	}
	if meta.Assemble.Stage.Stage2HitCount > 0 {
		payload["stage2_hit_count"] = meta.Assemble.Stage.Stage2HitCount
	}
	if meta.Assemble.Stage.Stage2Source != "" {
		payload["stage2_source"] = meta.Assemble.Stage.Stage2Source
	}
	if meta.Assemble.Stage.Stage2Reason != "" {
		payload["stage2_reason"] = meta.Assemble.Stage.Stage2Reason
	}
	if meta.Assemble.Stage.Stage2ReasonCode != "" {
		payload["stage2_reason_code"] = meta.Assemble.Stage.Stage2ReasonCode
	}
	if meta.Assemble.Stage.Stage2ErrorLayer != "" {
		payload["stage2_error_layer"] = meta.Assemble.Stage.Stage2ErrorLayer
	}
	if meta.Assemble.Stage.PressureZone != "" {
		payload["ca3_pressure_zone"] = meta.Assemble.Stage.PressureZone
	}
	if meta.Assemble.Stage.PressureReason != "" {
		payload["ca3_pressure_reason"] = meta.Assemble.Stage.PressureReason
	}
	if meta.Assemble.Stage.PressureTriggerSource != "" {
		payload["ca3_pressure_trigger"] = meta.Assemble.Stage.PressureTriggerSource
	}
	if len(meta.Assemble.Stage.ZoneResidencyMs) > 0 {
		payload["ca3_zone_residency_ms"] = meta.Assemble.Stage.ZoneResidencyMs
	}
	if len(meta.Assemble.Stage.TriggerCounts) > 0 {
		payload["ca3_trigger_counts"] = meta.Assemble.Stage.TriggerCounts
	}
	if meta.Assemble.Stage.CompressionRatio > 0 {
		payload["ca3_compression_ratio"] = meta.Assemble.Stage.CompressionRatio
	}
	if meta.Assemble.Stage.SpillCount > 0 {
		payload["ca3_spill_count"] = meta.Assemble.Stage.SpillCount
	}
	if meta.Assemble.Stage.SwapBackCount > 0 {
		payload["ca3_swap_back_count"] = meta.Assemble.Stage.SwapBackCount
	}
	if meta.Assemble.Recap.Status != "" {
		payload["recap_status"] = string(meta.Assemble.Recap.Status)
	}
	payload["gate_checks"] = meta.GateChecks
	payload["gate_denied_count"] = meta.GateDenied
	payload["gate_timeout_count"] = meta.GateTimeout
	return payload
}

func (e *Engine) resolvePrefixVersion() string {
	if e.runtimeMgr == nil {
		return runtimeconfig.DefaultConfig().ContextAssembler.PrefixVersion
	}
	return e.runtimeMgr.EffectiveConfig().ContextAssembler.PrefixVersion
}

func (e *Engine) contextAssemblerEnabled() bool {
	if e.runtimeMgr == nil {
		return false
	}
	return e.runtimeMgr.EffectiveConfig().ContextAssembler.Enabled
}

func classifyRunTerminal(terminal *types.ClassifiedError, runErr error) (types.ActionStatus, string) {
	if errors.Is(runErr, context.Canceled) {
		return types.ActionStatusCanceled, "context_canceled"
	}
	if errors.Is(runErr, context.DeadlineExceeded) {
		return types.ActionStatusCanceled, "deadline_exceeded"
	}
	if terminal == nil {
		return types.ActionStatusFailed, "aborted"
	}
	switch terminal.Class {
	case types.ErrPolicyTimeout:
		return types.ActionStatusCanceled, "policy_timeout"
	case types.ErrIterationLimit:
		return types.ActionStatusFailed, "iteration_limit"
	case types.ErrContext:
		return types.ActionStatusFailed, "context_error"
	case types.ErrModel:
		return types.ActionStatusFailed, "model_error"
	case types.ErrTool:
		return types.ActionStatusFailed, "tool_error"
	default:
		return types.ActionStatusFailed, "failed"
	}
}

func classifyTimelineError(err error) (types.ActionStatus, string) {
	if errors.Is(err, context.Canceled) {
		return types.ActionStatusCanceled, "context_canceled"
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return types.ActionStatusCanceled, "deadline_exceeded"
	}
	return types.ActionStatusFailed, "failed"
}

func classifyClassifiedTimelineError(err *types.ClassifiedError) (types.ActionStatus, string) {
	if err == nil {
		return types.ActionStatusFailed, "failed"
	}
	switch err.Class {
	case types.ErrPolicyTimeout:
		return types.ActionStatusCanceled, "policy_timeout"
	case types.ErrContext:
		return types.ActionStatusFailed, "context_error"
	case types.ErrModel:
		return types.ActionStatusFailed, "model_error"
	case types.ErrTool:
		return types.ActionStatusFailed, "tool_error"
	case types.ErrIterationLimit:
		return types.ActionStatusFailed, "iteration_limit"
	default:
		return types.ActionStatusFailed, "failed"
	}
}

type actionGateStats struct {
	Checks       int
	DeniedCount  int
	TimeoutCount int
}

type actionGateViolationError struct {
	classified *types.ClassifiedError
	err        error
}

func (e *actionGateViolationError) Error() string {
	if e == nil {
		return ""
	}
	if e.err != nil {
		return e.err.Error()
	}
	if e.classified != nil {
		return e.classified.Message
	}
	return "action gate violation"
}

func (e *actionGateViolationError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.err
}

func (e *actionGateViolationError) ClassifiedError() *types.ClassifiedError {
	if e == nil {
		return nil
	}
	return e.classified
}

func (e *Engine) enforceActionGateForToolCalls(
	ctx context.Context,
	h types.EventHandler,
	req types.RunRequest,
	runID string,
	iteration int,
	seq *int64,
	calls []types.ToolCall,
	stats *actionGateStats,
) (*types.ClassifiedError, error) {
	for _, call := range calls {
		check := types.ActionGateCheck{
			RunID:     runID,
			SessionID: req.SessionID,
			Iteration: iteration,
			CallID:    call.CallID,
			ToolName:  call.Name,
			Input:     req.Input,
			Args:      call.Args,
		}
		decision, checked, err := e.evaluateActionGateDecision(ctx, check)
		if err != nil {
			ce := classified(types.ErrTool, fmt.Sprintf("action gate evaluate failed: %v", err), false)
			e.emitTimeline(ctx, h, runID, iteration, seq, types.ActionPhaseTool, types.ActionStatusFailed, "gate.denied")
			if stats != nil {
				stats.Checks++
				stats.DeniedCount++
			}
			return ce, err
		}
		if !checked {
			continue
		}
		if stats != nil {
			stats.Checks++
		}
		switch decision {
		case types.ActionGateDecisionAllow:
			continue
		case types.ActionGateDecisionDeny:
			if stats != nil {
				stats.DeniedCount++
			}
			e.emitTimeline(ctx, h, runID, iteration, seq, types.ActionPhaseTool, types.ActionStatusFailed, "gate.denied")
			msg := fmt.Sprintf("action gate denied tool call: %s", strings.TrimSpace(call.Name))
			return classified(types.ErrTool, msg, false), errors.New(msg)
		case types.ActionGateDecisionRequireConfirm:
			e.emitTimeline(ctx, h, runID, iteration, seq, types.ActionPhaseTool, types.ActionStatusPending, "gate.require_confirm")
			if e.actionGateResolver == nil {
				if stats != nil {
					stats.DeniedCount++
				}
				e.emitTimeline(ctx, h, runID, iteration, seq, types.ActionPhaseTool, types.ActionStatusFailed, "gate.denied")
				msg := fmt.Sprintf("action gate requires confirmation but resolver is not configured: %s", strings.TrimSpace(call.Name))
				return classified(types.ErrTool, msg, false), errors.New(msg)
			}
			timeout := e.actionGateTimeout()
			confirmCtx, cancel := context.WithTimeout(ctx, timeout)
			approved, confirmErr := e.actionGateResolver.Confirm(confirmCtx, types.ActionGateConfirmRequest{
				Check:   check,
				Timeout: timeout,
			})
			cancel()
			if confirmErr != nil || errors.Is(confirmCtx.Err(), context.DeadlineExceeded) {
				if errors.Is(confirmErr, context.DeadlineExceeded) || errors.Is(confirmCtx.Err(), context.DeadlineExceeded) {
					if stats != nil {
						stats.TimeoutCount++
						stats.DeniedCount++
					}
					e.emitTimeline(ctx, h, runID, iteration, seq, types.ActionPhaseTool, types.ActionStatusCanceled, "gate.timeout")
					msg := fmt.Sprintf("action gate confirmation timed out for tool: %s", strings.TrimSpace(call.Name))
					return classified(types.ErrPolicyTimeout, msg, true), context.DeadlineExceeded
				}
				if stats != nil {
					stats.DeniedCount++
				}
				e.emitTimeline(ctx, h, runID, iteration, seq, types.ActionPhaseTool, types.ActionStatusFailed, "gate.denied")
				msg := fmt.Sprintf("action gate confirmation failed for tool %s: %v", strings.TrimSpace(call.Name), confirmErr)
				return classified(types.ErrTool, msg, false), confirmErr
			}
			if !approved {
				if stats != nil {
					stats.DeniedCount++
				}
				e.emitTimeline(ctx, h, runID, iteration, seq, types.ActionPhaseTool, types.ActionStatusFailed, "gate.denied")
				msg := fmt.Sprintf("action gate confirmation denied tool call: %s", strings.TrimSpace(call.Name))
				return classified(types.ErrTool, msg, false), errors.New(msg)
			}
		default:
			if stats != nil {
				stats.DeniedCount++
			}
			e.emitTimeline(ctx, h, runID, iteration, seq, types.ActionPhaseTool, types.ActionStatusFailed, "gate.denied")
			msg := fmt.Sprintf("action gate returned unsupported decision %q for tool %s", decision, strings.TrimSpace(call.Name))
			return classified(types.ErrTool, msg, false), errors.New(msg)
		}
	}
	return nil, nil
}

func (e *Engine) evaluateActionGateDecision(ctx context.Context, check types.ActionGateCheck) (types.ActionGateDecision, bool, error) {
	if e.actionGateMatcher != nil {
		decision, err := e.actionGateMatcher.Evaluate(ctx, check)
		if err != nil {
			return "", false, err
		}
		return normalizeActionGateDecision(decision), true, nil
	}
	cfg := e.actionGateConfig()
	if !cfg.Enabled {
		return types.ActionGateDecisionAllow, false, nil
	}

	toolName := normalizeToolName(check.ToolName)
	content := strings.ToLower(strings.TrimSpace(check.Input) + " " + marshalToolArgs(check.Args))
	defaultDecision := normalizeActionGateDecision(types.ActionGateDecision(cfg.Policy))

	if policy, ok := cfg.DecisionByTool[toolName]; ok {
		return normalizeActionGateDecision(types.ActionGateDecision(policy)), true, nil
	}
	segments := strings.Split(toolName, ".")
	if len(segments) > 1 {
		shortName := strings.TrimSpace(segments[len(segments)-1])
		if policy, ok := cfg.DecisionByTool[shortName]; ok {
			return normalizeActionGateDecision(types.ActionGateDecision(policy)), true, nil
		}
	}

	for _, tool := range cfg.ToolNames {
		normalized := normalizeToolName(tool)
		if normalized == "" {
			continue
		}
		if normalized == toolName {
			return defaultDecision, true, nil
		}
		if len(segments) > 1 && normalized == strings.TrimSpace(segments[len(segments)-1]) {
			return defaultDecision, true, nil
		}
	}

	for keyword, policy := range cfg.DecisionByWord {
		kw := strings.ToLower(strings.TrimSpace(keyword))
		if kw == "" {
			continue
		}
		if strings.Contains(content, kw) {
			return normalizeActionGateDecision(types.ActionGateDecision(policy)), true, nil
		}
	}
	for _, keyword := range cfg.Keywords {
		kw := strings.ToLower(strings.TrimSpace(keyword))
		if kw == "" {
			continue
		}
		if strings.Contains(content, kw) {
			return defaultDecision, true, nil
		}
	}
	return types.ActionGateDecisionAllow, false, nil
}

func (e *Engine) actionGateConfig() runtimeconfig.ActionGateConfig {
	if e.runtimeMgr == nil {
		return runtimeconfig.DefaultConfig().ActionGate
	}
	return e.runtimeMgr.EffectiveConfig().ActionGate
}

func (e *Engine) actionGateTimeout() time.Duration {
	cfg := e.actionGateConfig()
	if cfg.Timeout <= 0 {
		return 15 * time.Second
	}
	return cfg.Timeout
}

func normalizeToolName(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}

func normalizeActionGateDecision(in types.ActionGateDecision) types.ActionGateDecision {
	switch strings.ToLower(strings.TrimSpace(string(in))) {
	case string(types.ActionGateDecisionAllow):
		return types.ActionGateDecisionAllow
	case string(types.ActionGateDecisionDeny):
		return types.ActionGateDecisionDeny
	case string(types.ActionGateDecisionRequireConfirm):
		return types.ActionGateDecisionRequireConfirm
	default:
		return types.ActionGateDecisionDeny
	}
}

func marshalToolArgs(args map[string]any) string {
	if len(args) == 0 {
		return ""
	}
	raw, err := json.Marshal(args)
	if err != nil {
		return fmt.Sprintf("%v", args)
	}
	return string(raw)
}

var _ types.Runner = (*Engine)(nil)
