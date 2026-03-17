package runner

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"reflect"
	"regexp"
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

// Option customizes engine wiring such as tool runtime, tracing, runtime config, and HITL hooks.
type Option func(*Engine)

// Engine orchestrates run/stream model loop, tool dispatch, context assembly, and diagnostics emission.
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
	clarification      types.ClarificationResolver
	modelInputFilters  []types.ModelInputSecurityFilter
	modelOutputFilters []types.ModelOutputSecurityFilter
	securityAlert      types.SecurityAlertCallback
	securityDeliveryMu sync.Mutex
	securityDelivery   *securityAlertDeliveryExecutor
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

// New creates a runner engine with default tracer and context assembler wiring.
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

// WithLocalRegistry enables local tool dispatch backed by the provided tool registry.
func WithLocalRegistry(registry *local.Registry) Option {
	return func(e *Engine) {
		e.dispatcher = local.NewDispatcher(registry)
		if e.runtimeMgr != nil {
			e.dispatcher.SetRuntimeManager(e.runtimeMgr)
		}
	}
}

// WithTraceManager overrides the default trace manager.
func WithTraceManager(tracer *obsTrace.Manager) Option {
	return func(e *Engine) {
		if tracer != nil {
			e.tracer = tracer
		}
	}
}

// WithRuntimeManager injects runtime configuration/diagnostics manager into engine and dispatcher.
func WithRuntimeManager(mgr *runtimeconfig.Manager) Option {
	return func(e *Engine) {
		e.runtimeMgr = mgr
		if e.dispatcher != nil {
			e.dispatcher.SetRuntimeManager(mgr)
		}
	}
}

// WithProviderModels registers provider-name to model-client mapping for step-level fallback selection.
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

// WithActionGateMatcher injects custom action gate matcher implementation.
func WithActionGateMatcher(matcher types.ActionGateMatcher) Option {
	return func(e *Engine) {
		e.actionGateMatcher = matcher
	}
}

// WithActionGateResolver injects custom action gate confirmation resolver implementation.
func WithActionGateResolver(resolver types.ActionGateResolver) Option {
	return func(e *Engine) {
		e.actionGateResolver = resolver
	}
}

// WithClarificationResolver injects custom clarification resolver for HITL await/resume flow.
func WithClarificationResolver(resolver types.ClarificationResolver) Option {
	return func(e *Engine) {
		e.clarification = resolver
	}
}

// WithModelInputFilters registers host-provided model input security filters.
func WithModelInputFilters(filters ...types.ModelInputSecurityFilter) Option {
	return func(e *Engine) {
		e.modelInputFilters = normalizeInputFilters(filters)
	}
}

// WithModelOutputFilters registers host-provided model output security filters.
func WithModelOutputFilters(filters ...types.ModelOutputSecurityFilter) Option {
	return func(e *Engine) {
		e.modelOutputFilters = normalizeOutputFilters(filters)
	}
}

// WithSecurityAlertCallback registers a host callback sink for deny-only security alerts.
func WithSecurityAlertCallback(callback types.SecurityAlertCallback) Option {
	return func(e *Engine) {
		e.securityAlert = callback
	}
}

// Run executes a non-streaming agent loop and returns a final run result.
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
	hitlStats := clarificationStats{}
	concurrencyStats := runtimeConcurrencyStats{}
	lastSecurity := securityDecision{}
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
				ModelClient:   selectedModel,
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
			filteredReq, filterDecision, filterTerminal, filterErr := e.applyInputFilters(ctx, runID, iteration, modelReq)
			if filterDecision != nil {
				lastSecurity = *filterDecision
			}
			if filterTerminal != nil {
				e.emitTimeline(ctx, h, runID, iteration, &timelineSeq, types.ActionPhaseModel, types.ActionStatusFailed, lastSecurity.ReasonCode)
				terminal = filterTerminal
				runErr = filterErr
				state = StateAbort
				continue
			}
			modelReq = filteredReq
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
				case errors.Is(err, context.Canceled) || errors.Is(ctx.Err(), context.Canceled):
					terminal = classified(types.ErrPolicyTimeout, "model step canceled", true)
					timelineStatus, reason = types.ActionStatusCanceled, "cancel.propagated"
					concurrencyStats.CancelPropagatedCount++
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
			if strings.TrimSpace(resp.FinalAnswer) != "" {
				filteredOutput, filterDecision, filterTerminal, filterErr := e.applyOutputFilters(ctx, runID, iteration, resp.FinalAnswer)
				if filterDecision != nil {
					lastSecurity = *filterDecision
				}
				if filterTerminal != nil {
					e.emitTimeline(ctx, h, runID, iteration, &timelineSeq, types.ActionPhaseModel, types.ActionStatusFailed, lastSecurity.ReasonCode)
					terminal = filterTerminal
					runErr = filterErr
					state = StateAbort
					continue
				}
				resp.FinalAnswer = filteredOutput
			}
			lastResponse = resp
			e.emit(ctx, h, types.Event{Version: "v1", Type: "model.completed", RunID: runID, Iteration: iteration, Time: e.now()})
			e.emitTimeline(ctx, h, runID, iteration, &timelineSeq, types.ActionPhaseModel, types.ActionStatusSucceeded, "")
			if resp.ClarificationRequest != nil {
				clarification, hitlTerminal, hitlErr := e.awaitClarification(
					ctx,
					h,
					req,
					runID,
					iteration,
					&timelineSeq,
					resp.ClarificationRequest,
					&hitlStats,
				)
				if hitlTerminal != nil {
					terminal = hitlTerminal
					runErr = hitlErr
					state = StateAbort
					continue
				}
				req = applyClarificationResponse(req, clarification)
				state = StateModelStep
				continue
			}
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
			toolSecurityDecision, toolSecurityTerminal, toolSecurityErr := e.enforceToolSecurityForCalls(
				ctx,
				h,
				runID,
				iteration,
				&timelineSeq,
				lastResponse.ToolCalls,
			)
			if toolSecurityDecision != nil {
				lastSecurity = *toolSecurityDecision
			}
			if toolSecurityTerminal != nil {
				terminal = toolSecurityTerminal
				runErr = toolSecurityErr
				state = StateAbort
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
			concurrencyStats.ObserveInflight(maxInflightEstimate(len(lastResponse.ToolCalls), dispatchCfg.Concurrency))
			if dispatchCfg.Backpressure == types.BackpressureBlock && len(lastResponse.ToolCalls) > dispatchCfg.QueueSize {
				e.emitTimeline(ctx, h, runID, iteration, &timelineSeq, types.ActionPhaseTool, types.ActionStatusPending, "backpressure.block")
			}
			if dispatchCfg.Backpressure == types.BackpressureDropLowPriority && len(lastResponse.ToolCalls) > dispatchCfg.QueueSize {
				e.emitTimeline(ctx, h, runID, iteration, &timelineSeq, types.ActionPhaseTool, types.ActionStatusPending, "backpressure.drop_low_priority")
				for _, phase := range dropRelevantPhases(lastResponse.ToolCalls) {
					if phase == types.ActionPhaseTool {
						continue
					}
					e.emitTimeline(ctx, h, runID, iteration, &timelineSeq, phase, types.ActionStatusPending, "backpressure.drop_low_priority")
				}
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
			dropTotal, dropByPhase := countBackpressureDrops(outcomes)
			concurrencyStats.BackpressureDropCount += dropTotal
			concurrencyStats.AddBackpressureDropByPhase(dropByPhase)
			if dispatchCfg.Backpressure == types.BackpressureDropLowPriority {
				for _, phase := range phasesFullyDroppedByLowPriority(lastResponse.ToolCalls, outcomes) {
					e.emitTimeline(ctx, h, runID, iteration, &timelineSeq, phase, types.ActionStatusFailed, "backpressure.drop_low_priority")
				}
			}
			if dispatchCfg.Backpressure == types.BackpressureDropLowPriority && anyPhaseFullyDroppedByLowPriority(lastResponse.ToolCalls, outcomes) {
				e.emitTimeline(ctx, h, runID, iteration, &timelineSeq, types.ActionPhaseTool, types.ActionStatusFailed, "backpressure.drop_low_priority")
				terminal = classified(types.ErrTool, "all tool calls dropped by low-priority backpressure", false)
				runErr = errors.New(terminal.Message)
				state = StateAbort
				continue
			}
			if errors.Is(err, context.Canceled) || errors.Is(ctx.Err(), context.Canceled) {
				concurrencyStats.CancelPropagatedCount++
				e.emitTimeline(ctx, h, runID, iteration, &timelineSeq, types.ActionPhaseTool, types.ActionStatusCanceled, "cancel.propagated")
				terminal = classified(types.ErrPolicyTimeout, "tool dispatch canceled", true)
				runErr = context.Canceled
				state = StateAbort
				continue
			}
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
					Provider:            lastSelection.Provider,
					Initial:             lastSelection.Initial,
					Path:                selectionPath,
					Required:            lastSelection.Required,
					UsedFallback:        fallbackUsed,
					Assemble:            lastAssemble,
					GateChecks:          gateStats.Checks,
					GateDenied:          gateStats.DeniedCount,
					GateTimeout:         gateStats.TimeoutCount,
					GateRuleHits:        gateStats.RuleHitCount,
					GateRuleLast:        gateStats.RuleLastID,
					HitlAwait:           hitlStats.AwaitCount,
					HitlResumed:         hitlStats.ResumeCount,
					HitlCanceled:        hitlStats.CancelByUserCount,
					CancelProp:          concurrencyStats.CancelPropagatedCount,
					BackDrop:            concurrencyStats.BackpressureDropCount,
					BackDropByPhase:     concurrencyStats.BackpressureDropByPhase,
					InflightPeak:        concurrencyStats.InflightPeak,
					SecurityPolicy:      lastSecurity.PolicyKind,
					NamespaceTool:       lastSecurity.NamespaceTool,
					FilterStage:         lastSecurity.FilterStage,
					SecurityDecision:    lastSecurity.Decision,
					ReasonCode:          lastSecurity.ReasonCode,
					Severity:            lastSecurity.Severity,
					AlertStatus:         lastSecurity.AlertDispatchStatus,
					AlertFailure:        lastSecurity.AlertFailureReason,
					AlertDeliveryMode:   lastSecurity.AlertDeliveryMode,
					AlertRetryCount:     lastSecurity.AlertRetryCount,
					AlertQueueDropped:   lastSecurity.AlertQueueDropped,
					AlertQueueDropCount: lastSecurity.AlertQueueDropCount,
					AlertCircuitState:   lastSecurity.AlertCircuitState,
					AlertCircuitReason:  lastSecurity.AlertCircuitReason,
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
					Provider:            lastSelection.Provider,
					Initial:             lastSelection.Initial,
					Path:                selectionPath,
					Required:            lastSelection.Required,
					Reason:              lastSelection.Reason,
					UsedFallback:        fallbackUsed,
					Assemble:            lastAssemble,
					GateChecks:          gateStats.Checks,
					GateDenied:          gateStats.DeniedCount,
					GateTimeout:         gateStats.TimeoutCount,
					GateRuleHits:        gateStats.RuleHitCount,
					GateRuleLast:        gateStats.RuleLastID,
					HitlAwait:           hitlStats.AwaitCount,
					HitlResumed:         hitlStats.ResumeCount,
					HitlCanceled:        hitlStats.CancelByUserCount,
					CancelProp:          concurrencyStats.CancelPropagatedCount,
					BackDrop:            concurrencyStats.BackpressureDropCount,
					BackDropByPhase:     concurrencyStats.BackpressureDropByPhase,
					InflightPeak:        concurrencyStats.InflightPeak,
					SecurityPolicy:      lastSecurity.PolicyKind,
					NamespaceTool:       lastSecurity.NamespaceTool,
					FilterStage:         lastSecurity.FilterStage,
					SecurityDecision:    lastSecurity.Decision,
					ReasonCode:          lastSecurity.ReasonCode,
					Severity:            lastSecurity.Severity,
					AlertStatus:         lastSecurity.AlertDispatchStatus,
					AlertFailure:        lastSecurity.AlertFailureReason,
					AlertDeliveryMode:   lastSecurity.AlertDeliveryMode,
					AlertRetryCount:     lastSecurity.AlertRetryCount,
					AlertQueueDropped:   lastSecurity.AlertQueueDropped,
					AlertQueueDropCount: lastSecurity.AlertQueueDropCount,
					AlertCircuitState:   lastSecurity.AlertCircuitState,
					AlertCircuitReason:  lastSecurity.AlertCircuitReason,
				}),
			})
			return result, runErr
		}
	}
}

// Stream executes a streaming agent loop while preserving timeline/error semantics with Run.
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
	hitlStats := clarificationStats{}
	concurrencyStats := runtimeConcurrencyStats{}
	lastSecurity := securityDecision{}
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
				Provider:            selection.Provider,
				Initial:             selection.Initial,
				Path:                selectionPath,
				Required:            required,
				UsedFallback:        selection.UsedFallback,
				Reason:              selErr.Message,
				Assemble:            types.ContextAssembleResult{},
				SecurityPolicy:      lastSecurity.PolicyKind,
				NamespaceTool:       lastSecurity.NamespaceTool,
				FilterStage:         lastSecurity.FilterStage,
				SecurityDecision:    lastSecurity.Decision,
				ReasonCode:          lastSecurity.ReasonCode,
				Severity:            lastSecurity.Severity,
				AlertStatus:         lastSecurity.AlertDispatchStatus,
				AlertFailure:        lastSecurity.AlertFailureReason,
				AlertDeliveryMode:   lastSecurity.AlertDeliveryMode,
				AlertRetryCount:     lastSecurity.AlertRetryCount,
				AlertQueueDropped:   lastSecurity.AlertQueueDropped,
				AlertQueueDropCount: lastSecurity.AlertQueueDropCount,
				AlertCircuitState:   lastSecurity.AlertCircuitState,
				AlertCircuitReason:  lastSecurity.AlertCircuitReason,
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
		ModelClient:   selectedModel,
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
				Provider:            "",
				Initial:             "",
				Path:                selectionPath,
				Required:            required,
				UsedFallback:        false,
				Assemble:            lastAssemble,
				SecurityPolicy:      lastSecurity.PolicyKind,
				NamespaceTool:       lastSecurity.NamespaceTool,
				FilterStage:         lastSecurity.FilterStage,
				SecurityDecision:    lastSecurity.Decision,
				ReasonCode:          lastSecurity.ReasonCode,
				Severity:            lastSecurity.Severity,
				AlertStatus:         lastSecurity.AlertDispatchStatus,
				AlertFailure:        lastSecurity.AlertFailureReason,
				AlertDeliveryMode:   lastSecurity.AlertDeliveryMode,
				AlertRetryCount:     lastSecurity.AlertRetryCount,
				AlertQueueDropped:   lastSecurity.AlertQueueDropped,
				AlertQueueDropCount: lastSecurity.AlertQueueDropCount,
				AlertCircuitState:   lastSecurity.AlertCircuitState,
				AlertCircuitReason:  lastSecurity.AlertCircuitReason,
			}),
		})
		return result, assembleErr
	}
	if contextPhaseEnabled {
		e.emitTimeline(ctx, h, runID, iteration, &timelineSeq, types.ActionPhaseContextAssembler, types.ActionStatusSucceeded, "")
	}
	modelReq = assembledReq
	filteredReq, filterDecision, filterTerminal, filterErr := e.applyInputFilters(ctx, runID, iteration, modelReq)
	if filterDecision != nil {
		lastSecurity = *filterDecision
	}
	if filterTerminal != nil {
		result := types.RunResult{
			RunID:      runID,
			Iterations: iteration,
			LatencyMs:  e.now().Sub(start).Milliseconds(),
			Error:      filterTerminal,
		}
		runTimelineStatus, runTimelineReason := classifyRunTerminal(result.Error, filterErr)
		e.emitTimeline(ctx, h, runID, iteration, &timelineSeq, types.ActionPhaseModel, types.ActionStatusFailed, lastSecurity.ReasonCode)
		e.emitTimeline(ctx, h, runID, iteration, &timelineSeq, types.ActionPhaseRun, runTimelineStatus, runTimelineReason)
		e.emit(ctx, h, types.Event{
			Version:   "v1",
			Type:      "run.finished",
			RunID:     runID,
			Iteration: iteration,
			Time:      e.now(),
			Payload: runFinishedPayload(result, "failed", string(types.ErrSecurity), runFinishMeta{
				Provider:            selection.Provider,
				Initial:             selection.Initial,
				Path:                selectionPath,
				Required:            required,
				UsedFallback:        selection.UsedFallback,
				Assemble:            lastAssemble,
				SecurityPolicy:      lastSecurity.PolicyKind,
				NamespaceTool:       lastSecurity.NamespaceTool,
				FilterStage:         lastSecurity.FilterStage,
				SecurityDecision:    lastSecurity.Decision,
				ReasonCode:          lastSecurity.ReasonCode,
				Severity:            lastSecurity.Severity,
				AlertStatus:         lastSecurity.AlertDispatchStatus,
				AlertFailure:        lastSecurity.AlertFailureReason,
				AlertDeliveryMode:   lastSecurity.AlertDeliveryMode,
				AlertRetryCount:     lastSecurity.AlertRetryCount,
				AlertQueueDropped:   lastSecurity.AlertQueueDropped,
				AlertQueueDropCount: lastSecurity.AlertQueueDropCount,
				AlertCircuitState:   lastSecurity.AlertCircuitState,
				AlertCircuitReason:  lastSecurity.AlertCircuitReason,
			}),
		})
		return result, filterErr
	}
	modelReq = filteredReq
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
		normalizedEvent := ev
		if normalizedEvent.Type == types.ModelEventTypeFinalAnswer || normalizedEvent.Type == types.ModelEventTypeOutputTextDelta {
			filteredOutput, filterDecision, filterTerminal, filterErr := e.applyOutputFilters(stepCtx, runID, iteration, normalizedEvent.TextDelta)
			if filterDecision != nil {
				lastSecurity = *filterDecision
			}
			if filterTerminal != nil {
				return &actionGateViolationError{
					classified: filterTerminal,
					err:        filterErr,
				}
			}
			normalizedEvent.TextDelta = filteredOutput
		}
		if normalizedEvent.ToolCall != nil {
			toolSecurityDecision, toolSecurityTerminal, toolSecurityErr := e.enforceToolSecurityForCalls(
				stepCtx,
				h,
				runID,
				iteration,
				&timelineSeq,
				[]types.ToolCall{*normalizedEvent.ToolCall},
			)
			if toolSecurityDecision != nil {
				lastSecurity = *toolSecurityDecision
			}
			if toolSecurityTerminal != nil {
				return &actionGateViolationError{
					classified: toolSecurityTerminal,
					err:        toolSecurityErr,
				}
			}
		}
		payload := map[string]any{
			"event_type": normalizedEvent.Type,
			"delta":      normalizedEvent.TextDelta,
		}
		if normalizedEvent.ToolCall != nil {
			payload["tool_call"] = normalizedEvent.ToolCall
		}
		if normalizedEvent.ClarificationRequest != nil {
			payload["clarification_request"] = map[string]any{
				"request_id":      strings.TrimSpace(normalizedEvent.ClarificationRequest.RequestID),
				"questions":       normalizedEvent.ClarificationRequest.Questions,
				"context_summary": strings.TrimSpace(normalizedEvent.ClarificationRequest.ContextSummary),
				"timeout_ms":      normalizedEvent.ClarificationRequest.Timeout.Milliseconds(),
			}
		}
		if len(normalizedEvent.Meta) > 0 {
			payload["meta"] = normalizedEvent.Meta
		}
		e.emit(stepCtx, h, types.Event{
			Version:   "v1",
			Type:      "model.delta",
			RunID:     runID,
			Iteration: iteration,
			Time:      e.now(),
			Payload:   payload,
		})
		if normalizedEvent.Type == types.ModelEventTypeFinalAnswer || normalizedEvent.Type == types.ModelEventTypeOutputTextDelta {
			final += normalizedEvent.TextDelta
		}
		if normalizedEvent.ClarificationRequest != nil || normalizedEvent.Type == types.ModelEventTypeClarificationRequest {
			request := normalizedEvent.ClarificationRequest
			if request == nil {
				request = &types.ClarificationRequest{}
			}
			if request.Timeout <= 0 {
				request.Timeout = e.clarificationTimeout()
			}
			clarification, hitlTerminal, hitlErr := e.awaitClarification(
				stepCtx,
				h,
				req,
				runID,
				iteration,
				&timelineSeq,
				request,
				&hitlStats,
			)
			if hitlTerminal != nil {
				return &actionGateViolationError{
					classified: hitlTerminal,
					err:        hitlErr,
				}
			}
			req = applyClarificationResponse(req, clarification)
		}
		if normalizedEvent.ToolCall != nil {
			e.emitTimeline(stepCtx, h, runID, iteration, &timelineSeq, types.ActionPhaseTool, types.ActionStatusPending, "")
			e.emitTimeline(stepCtx, h, runID, iteration, &timelineSeq, types.ActionPhaseTool, types.ActionStatusRunning, "")
			gateTerm, gateRunErr := e.enforceActionGateForToolCalls(
				stepCtx,
				h,
				req,
				runID,
				iteration,
				&timelineSeq,
				[]types.ToolCall{*normalizedEvent.ToolCall},
				&gateStats,
			)
			if gateTerm != nil {
				return &actionGateViolationError{
					classified: gateTerm,
					err:        gateRunErr,
				}
			}
			concurrencyStats.ObserveInflight(1)
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
		case errors.Is(err, context.Canceled) || errors.Is(ctx.Err(), context.Canceled):
			terminal = classified(types.ErrPolicyTimeout, "model stream canceled", true)
			timelineStatus, reason = types.ActionStatusCanceled, "cancel.propagated"
			concurrencyStats.CancelPropagatedCount++
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
				Provider:            selection.Provider,
				Initial:             selection.Initial,
				Path:                selectionPath,
				Required:            required,
				UsedFallback:        selection.UsedFallback,
				Assemble:            lastAssemble,
				GateChecks:          gateStats.Checks,
				GateDenied:          gateStats.DeniedCount,
				GateTimeout:         gateStats.TimeoutCount,
				GateRuleHits:        gateStats.RuleHitCount,
				GateRuleLast:        gateStats.RuleLastID,
				HitlAwait:           hitlStats.AwaitCount,
				HitlResumed:         hitlStats.ResumeCount,
				HitlCanceled:        hitlStats.CancelByUserCount,
				CancelProp:          concurrencyStats.CancelPropagatedCount,
				BackDrop:            concurrencyStats.BackpressureDropCount,
				BackDropByPhase:     concurrencyStats.BackpressureDropByPhase,
				InflightPeak:        concurrencyStats.InflightPeak,
				SecurityPolicy:      lastSecurity.PolicyKind,
				NamespaceTool:       lastSecurity.NamespaceTool,
				FilterStage:         lastSecurity.FilterStage,
				SecurityDecision:    lastSecurity.Decision,
				ReasonCode:          lastSecurity.ReasonCode,
				Severity:            lastSecurity.Severity,
				AlertStatus:         lastSecurity.AlertDispatchStatus,
				AlertFailure:        lastSecurity.AlertFailureReason,
				AlertDeliveryMode:   lastSecurity.AlertDeliveryMode,
				AlertRetryCount:     lastSecurity.AlertRetryCount,
				AlertQueueDropped:   lastSecurity.AlertQueueDropped,
				AlertQueueDropCount: lastSecurity.AlertQueueDropCount,
				AlertCircuitState:   lastSecurity.AlertCircuitState,
				AlertCircuitReason:  lastSecurity.AlertCircuitReason,
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
			Provider:            selection.Provider,
			Initial:             selection.Initial,
			Path:                selectionPath,
			Required:            required,
			UsedFallback:        selection.UsedFallback,
			Assemble:            lastAssemble,
			GateChecks:          gateStats.Checks,
			GateDenied:          gateStats.DeniedCount,
			GateTimeout:         gateStats.TimeoutCount,
			GateRuleHits:        gateStats.RuleHitCount,
			GateRuleLast:        gateStats.RuleLastID,
			HitlAwait:           hitlStats.AwaitCount,
			HitlResumed:         hitlStats.ResumeCount,
			HitlCanceled:        hitlStats.CancelByUserCount,
			CancelProp:          concurrencyStats.CancelPropagatedCount,
			BackDrop:            concurrencyStats.BackpressureDropCount,
			BackDropByPhase:     concurrencyStats.BackpressureDropByPhase,
			InflightPeak:        concurrencyStats.InflightPeak,
			SecurityPolicy:      lastSecurity.PolicyKind,
			NamespaceTool:       lastSecurity.NamespaceTool,
			FilterStage:         lastSecurity.FilterStage,
			SecurityDecision:    lastSecurity.Decision,
			ReasonCode:          lastSecurity.ReasonCode,
			Severity:            lastSecurity.Severity,
			AlertStatus:         lastSecurity.AlertDispatchStatus,
			AlertFailure:        lastSecurity.AlertFailureReason,
			AlertDeliveryMode:   lastSecurity.AlertDeliveryMode,
			AlertRetryCount:     lastSecurity.AlertRetryCount,
			AlertQueueDropped:   lastSecurity.AlertQueueDropped,
			AlertQueueDropCount: lastSecurity.AlertQueueDropCount,
			AlertCircuitState:   lastSecurity.AlertCircuitState,
			AlertCircuitReason:  lastSecurity.AlertCircuitReason,
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
	Provider            string
	Initial             string
	Path                []string
	Required            []types.ModelCapability
	UsedFallback        bool
	Reason              string
	Assemble            types.ContextAssembleResult
	GateChecks          int
	GateDenied          int
	GateTimeout         int
	GateRuleHits        int
	GateRuleLast        string
	HitlAwait           int
	HitlResumed         int
	HitlCanceled        int
	CancelProp          int
	BackDrop            int
	BackDropByPhase     map[string]int
	InflightPeak        int
	SecurityPolicy      string
	NamespaceTool       string
	FilterStage         string
	SecurityDecision    string
	ReasonCode          string
	Severity            string
	AlertStatus         string
	AlertFailure        string
	AlertDeliveryMode   string
	AlertRetryCount     int
	AlertQueueDropped   bool
	AlertQueueDropCount int
	AlertCircuitState   string
	AlertCircuitReason  string
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
	if meta.Assemble.Stage.Stage2TemplateProfile != "" {
		payload["stage2_template_profile"] = meta.Assemble.Stage.Stage2TemplateProfile
	}
	if meta.Assemble.Stage.Stage2TemplateResolutionSource != "" {
		payload["stage2_template_resolution_source"] = meta.Assemble.Stage.Stage2TemplateResolutionSource
	}
	payload["stage2_hint_applied"] = meta.Assemble.Stage.Stage2HintApplied
	if meta.Assemble.Stage.Stage2HintMismatchReason != "" {
		payload["stage2_hint_mismatch_reason"] = meta.Assemble.Stage.Stage2HintMismatchReason
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
	if meta.Assemble.Stage.CompactionMode != "" {
		payload["ca3_compaction_mode"] = meta.Assemble.Stage.CompactionMode
	}
	if meta.Assemble.Stage.CompactionFallback {
		payload["ca3_compaction_fallback"] = meta.Assemble.Stage.CompactionFallback
	}
	if meta.Assemble.Stage.CompactionFallbackReason != "" {
		payload["ca3_compaction_fallback_reason"] = meta.Assemble.Stage.CompactionFallbackReason
	}
	if meta.Assemble.Stage.CompactionQualityScore > 0 {
		payload["ca3_compaction_quality_score"] = meta.Assemble.Stage.CompactionQualityScore
	}
	if meta.Assemble.Stage.CompactionQualityReason != "" {
		payload["ca3_compaction_quality_reason"] = meta.Assemble.Stage.CompactionQualityReason
	}
	if meta.Assemble.Stage.CompactionEmbeddingProvider != "" {
		payload["ca3_compaction_embedding_provider"] = meta.Assemble.Stage.CompactionEmbeddingProvider
	}
	if meta.Assemble.Stage.CompactionEmbeddingSimilarity > 0 {
		payload["ca3_compaction_embedding_similarity"] = meta.Assemble.Stage.CompactionEmbeddingSimilarity
	}
	if meta.Assemble.Stage.CompactionEmbeddingContribution > 0 {
		payload["ca3_compaction_embedding_contribution"] = meta.Assemble.Stage.CompactionEmbeddingContribution
	}
	if meta.Assemble.Stage.CompactionEmbeddingStatus != "" {
		payload["ca3_compaction_embedding_status"] = meta.Assemble.Stage.CompactionEmbeddingStatus
	}
	if meta.Assemble.Stage.CompactionEmbeddingFallbackReason != "" {
		payload["ca3_compaction_embedding_fallback_reason"] = meta.Assemble.Stage.CompactionEmbeddingFallbackReason
	}
	payload["ca3_compaction_reranker_used"] = meta.Assemble.Stage.CompactionRerankerUsed
	if meta.Assemble.Stage.CompactionRerankerProvider != "" {
		payload["ca3_compaction_reranker_provider"] = meta.Assemble.Stage.CompactionRerankerProvider
	}
	if meta.Assemble.Stage.CompactionRerankerModel != "" {
		payload["ca3_compaction_reranker_model"] = meta.Assemble.Stage.CompactionRerankerModel
	}
	if meta.Assemble.Stage.CompactionRerankerThresholdSource != "" {
		payload["ca3_compaction_reranker_threshold_source"] = meta.Assemble.Stage.CompactionRerankerThresholdSource
	}
	payload["ca3_compaction_reranker_threshold_hit"] = meta.Assemble.Stage.CompactionRerankerThresholdHit
	if meta.Assemble.Stage.CompactionRerankerFallbackReason != "" {
		payload["ca3_compaction_reranker_fallback_reason"] = meta.Assemble.Stage.CompactionRerankerFallbackReason
	}
	if meta.Assemble.Stage.CompactionRerankerProfileVersion != "" {
		payload["ca3_compaction_reranker_profile_version"] = meta.Assemble.Stage.CompactionRerankerProfileVersion
	}
	payload["ca3_compaction_reranker_rollout_hit"] = meta.Assemble.Stage.CompactionRerankerRolloutHit
	if meta.Assemble.Stage.CompactionRerankerThresholdDrift > 0 {
		payload["ca3_compaction_reranker_threshold_drift"] = meta.Assemble.Stage.CompactionRerankerThresholdDrift
	}
	if meta.Assemble.Stage.RetainedEvidenceCount > 0 {
		payload["ca3_compaction_retained_evidence_count"] = meta.Assemble.Stage.RetainedEvidenceCount
	}
	if meta.Assemble.Recap.Status != "" {
		payload["recap_status"] = string(meta.Assemble.Recap.Status)
	}
	payload["gate_checks"] = meta.GateChecks
	payload["gate_denied_count"] = meta.GateDenied
	payload["gate_timeout_count"] = meta.GateTimeout
	payload["gate_rule_hit_count"] = meta.GateRuleHits
	payload["gate_rule_last_id"] = meta.GateRuleLast
	payload["await_count"] = meta.HitlAwait
	payload["resume_count"] = meta.HitlResumed
	payload["cancel_by_user_count"] = meta.HitlCanceled
	payload["cancel_propagated_count"] = meta.CancelProp
	payload["backpressure_drop_count"] = meta.BackDrop
	if len(meta.BackDropByPhase) > 0 {
		payload["backpressure_drop_count_by_phase"] = meta.BackDropByPhase
	}
	payload["inflight_peak"] = meta.InflightPeak
	if meta.SecurityPolicy != "" {
		payload["policy_kind"] = meta.SecurityPolicy
	}
	if meta.NamespaceTool != "" {
		payload["namespace_tool"] = meta.NamespaceTool
	}
	if meta.FilterStage != "" {
		payload["filter_stage"] = meta.FilterStage
	}
	if meta.SecurityDecision != "" {
		payload["decision"] = meta.SecurityDecision
	}
	if meta.ReasonCode != "" {
		payload["reason_code"] = meta.ReasonCode
	}
	if meta.Severity != "" {
		payload["severity"] = meta.Severity
	}
	if meta.AlertStatus != "" {
		payload["alert_dispatch_status"] = meta.AlertStatus
	}
	if meta.AlertFailure != "" {
		payload["alert_dispatch_failure_reason"] = meta.AlertFailure
	}
	if meta.AlertDeliveryMode != "" {
		payload["alert_delivery_mode"] = meta.AlertDeliveryMode
	}
	payload["alert_retry_count"] = meta.AlertRetryCount
	if meta.AlertQueueDropped {
		payload["alert_queue_dropped"] = true
	}
	if meta.AlertQueueDropCount > 0 {
		payload["alert_queue_drop_count"] = meta.AlertQueueDropCount
	}
	if meta.AlertCircuitState != "" {
		payload["alert_circuit_state"] = meta.AlertCircuitState
	}
	if meta.AlertCircuitReason != "" {
		payload["alert_circuit_open_reason"] = meta.AlertCircuitReason
	}
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
	case types.ErrSecurity:
		return types.ActionStatusFailed, "security_error"
	case types.ErrHITL:
		return types.ActionStatusFailed, "hitl_error"
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
	case types.ErrSecurity:
		return types.ActionStatusFailed, "security_error"
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
	RuleHitCount int
	RuleLastID   string
}

type clarificationStats struct {
	AwaitCount        int
	ResumeCount       int
	CancelByUserCount int
}

type runtimeConcurrencyStats struct {
	CancelPropagatedCount   int
	BackpressureDropCount   int
	BackpressureDropByPhase map[string]int
	InflightPeak            int
}

func (s *runtimeConcurrencyStats) ObserveInflight(candidate int) {
	if s == nil || candidate <= 0 {
		return
	}
	if candidate > s.InflightPeak {
		s.InflightPeak = candidate
	}
}

func (s *runtimeConcurrencyStats) AddBackpressureDropByPhase(in map[string]int) {
	if s == nil || len(in) == 0 {
		return
	}
	if s.BackpressureDropByPhase == nil {
		s.BackpressureDropByPhase = map[string]int{}
	}
	for phase, n := range in {
		if strings.TrimSpace(phase) == "" || n <= 0 {
			continue
		}
		s.BackpressureDropByPhase[phase] += n
	}
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
		evaluation, err := e.evaluateActionGateDecision(ctx, check)
		if err != nil {
			ce := classified(types.ErrTool, fmt.Sprintf("action gate evaluate failed: %v", err), false)
			e.emitTimeline(ctx, h, runID, iteration, seq, types.ActionPhaseTool, types.ActionStatusFailed, "gate.denied")
			if stats != nil {
				stats.Checks++
				stats.DeniedCount++
			}
			return ce, err
		}
		if !evaluation.Checked {
			continue
		}
		if stats != nil {
			stats.Checks++
			if evaluation.RuleHit {
				stats.RuleHitCount++
				stats.RuleLastID = evaluation.RuleID
			}
		}
		if evaluation.RuleHit {
			e.emitTimeline(ctx, h, runID, iteration, seq, types.ActionPhaseTool, types.ActionStatusPending, "gate.rule_match")
		}
		switch evaluation.Decision {
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
			msg := fmt.Sprintf("action gate returned unsupported decision %q for tool %s", evaluation.Decision, strings.TrimSpace(call.Name))
			return classified(types.ErrTool, msg, false), errors.New(msg)
		}
	}
	return nil, nil
}

type actionGateEvaluation struct {
	Decision types.ActionGateDecision
	Checked  bool
	RuleHit  bool
	RuleID   string
}

func (e *Engine) evaluateActionGateDecision(ctx context.Context, check types.ActionGateCheck) (actionGateEvaluation, error) {
	if e.actionGateMatcher != nil {
		decision, err := e.actionGateMatcher.Evaluate(ctx, check)
		if err != nil {
			return actionGateEvaluation{}, err
		}
		return actionGateEvaluation{
			Decision: normalizeActionGateDecision(decision),
			Checked:  true,
		}, nil
	}
	cfg := e.actionGateConfig()
	if !cfg.Enabled {
		return actionGateEvaluation{Decision: types.ActionGateDecisionAllow}, nil
	}

	toolName := normalizeToolName(check.ToolName)
	content := strings.ToLower(strings.TrimSpace(check.Input) + " " + marshalToolArgs(check.Args))
	defaultDecision := normalizeActionGateDecision(types.ActionGateDecision(cfg.Policy))
	for _, rule := range cfg.ParameterRules {
		if !ruleAppliesToTool(rule, toolName) {
			continue
		}
		matched, err := evaluateRuleCondition(rule.Condition, check.Args)
		if err != nil {
			return actionGateEvaluation{}, err
		}
		if !matched {
			continue
		}
		decision := defaultDecision
		if strings.TrimSpace(string(rule.Action)) != "" {
			decision = normalizeActionGateDecision(rule.Action)
		}
		return actionGateEvaluation{
			Decision: decision,
			Checked:  true,
			RuleHit:  true,
			RuleID:   strings.TrimSpace(rule.ID),
		}, nil
	}

	if policy, ok := cfg.DecisionByTool[toolName]; ok {
		return actionGateEvaluation{
			Decision: normalizeActionGateDecision(types.ActionGateDecision(policy)),
			Checked:  true,
		}, nil
	}
	segments := strings.Split(toolName, ".")
	if len(segments) > 1 {
		shortName := strings.TrimSpace(segments[len(segments)-1])
		if policy, ok := cfg.DecisionByTool[shortName]; ok {
			return actionGateEvaluation{
				Decision: normalizeActionGateDecision(types.ActionGateDecision(policy)),
				Checked:  true,
			}, nil
		}
	}

	for _, tool := range cfg.ToolNames {
		normalized := normalizeToolName(tool)
		if normalized == "" {
			continue
		}
		if normalized == toolName {
			return actionGateEvaluation{Decision: defaultDecision, Checked: true}, nil
		}
		if len(segments) > 1 && normalized == strings.TrimSpace(segments[len(segments)-1]) {
			return actionGateEvaluation{Decision: defaultDecision, Checked: true}, nil
		}
	}

	for keyword, policy := range cfg.DecisionByWord {
		kw := strings.ToLower(strings.TrimSpace(keyword))
		if kw == "" {
			continue
		}
		if strings.Contains(content, kw) {
			return actionGateEvaluation{
				Decision: normalizeActionGateDecision(types.ActionGateDecision(policy)),
				Checked:  true,
			}, nil
		}
	}
	for _, keyword := range cfg.Keywords {
		kw := strings.ToLower(strings.TrimSpace(keyword))
		if kw == "" {
			continue
		}
		if strings.Contains(content, kw) {
			return actionGateEvaluation{Decision: defaultDecision, Checked: true}, nil
		}
	}
	return actionGateEvaluation{Decision: types.ActionGateDecisionAllow}, nil
}

func ruleAppliesToTool(rule types.ActionGateParameterRule, toolName string) bool {
	if len(rule.ToolNames) == 0 {
		return true
	}
	segments := strings.Split(toolName, ".")
	shortName := toolName
	if len(segments) > 1 {
		shortName = strings.TrimSpace(segments[len(segments)-1])
	}
	for _, tool := range rule.ToolNames {
		normalized := normalizeToolName(tool)
		if normalized == "" {
			continue
		}
		if normalized == toolName || normalized == shortName {
			return true
		}
	}
	return false
}

func evaluateRuleCondition(condition types.ActionGateRuleCondition, args map[string]any) (bool, error) {
	if len(condition.All) > 0 {
		for _, child := range condition.All {
			ok, err := evaluateRuleCondition(child, args)
			if err != nil {
				return false, err
			}
			if !ok {
				return false, nil
			}
		}
		return true, nil
	}
	if len(condition.Any) > 0 {
		for _, child := range condition.Any {
			ok, err := evaluateRuleCondition(child, args)
			if err != nil {
				return false, err
			}
			if ok {
				return true, nil
			}
		}
		return false, nil
	}
	value, exists := lookupPathValue(args, condition.Path)
	return evaluateLeafCondition(condition, value, exists)
}

func lookupPathValue(args map[string]any, path string) (any, bool) {
	if len(args) == 0 {
		return nil, false
	}
	segments := strings.Split(strings.TrimSpace(path), ".")
	if len(segments) == 0 {
		return nil, false
	}
	var current any = args
	for _, segment := range segments {
		key := strings.TrimSpace(segment)
		if key == "" {
			return nil, false
		}
		next, ok := readMapValue(current, key)
		if !ok {
			return nil, false
		}
		current = next
	}
	return current, true
}

func readMapValue(container any, key string) (any, bool) {
	switch tv := container.(type) {
	case map[string]any:
		value, ok := tv[key]
		return value, ok
	case map[string]string:
		value, ok := tv[key]
		if !ok {
			return nil, false
		}
		return value, true
	default:
		rv := reflect.ValueOf(container)
		if rv.Kind() != reflect.Map {
			return nil, false
		}
		if rv.Type().Key().Kind() != reflect.String {
			return nil, false
		}
		entry := rv.MapIndex(reflect.ValueOf(key))
		if !entry.IsValid() {
			return nil, false
		}
		return entry.Interface(), true
	}
}

func evaluateLeafCondition(condition types.ActionGateRuleCondition, value any, exists bool) (bool, error) {
	operator := strings.ToLower(strings.TrimSpace(string(condition.Operator)))
	switch operator {
	case string(types.ActionGateRuleOperatorExists):
		return exists, nil
	case string(types.ActionGateRuleOperatorEQ):
		return compareEqual(value, condition.Expected), nil
	case string(types.ActionGateRuleOperatorNE):
		return !compareEqual(value, condition.Expected), nil
	case string(types.ActionGateRuleOperatorContains):
		return evaluateContains(value, condition.Expected), nil
	case string(types.ActionGateRuleOperatorRegex):
		if !exists {
			return false, nil
		}
		pattern, ok := condition.Expected.(string)
		if !ok {
			return false, fmt.Errorf("regex expected must be string, got %T", condition.Expected)
		}
		text, ok := value.(string)
		if !ok {
			return false, nil
		}
		return regexp.MatchString(pattern, text)
	case string(types.ActionGateRuleOperatorIn):
		return evaluateIn(value, condition.Expected), nil
	case string(types.ActionGateRuleOperatorNotIn):
		return !evaluateIn(value, condition.Expected), nil
	case string(types.ActionGateRuleOperatorGT):
		return evaluateNumericCompare(value, condition.Expected, func(left, right float64) bool { return left > right })
	case string(types.ActionGateRuleOperatorGTE):
		return evaluateNumericCompare(value, condition.Expected, func(left, right float64) bool { return left >= right })
	case string(types.ActionGateRuleOperatorLT):
		return evaluateNumericCompare(value, condition.Expected, func(left, right float64) bool { return left < right })
	case string(types.ActionGateRuleOperatorLTE):
		return evaluateNumericCompare(value, condition.Expected, func(left, right float64) bool { return left <= right })
	default:
		return false, fmt.Errorf("unsupported action gate operator: %s", condition.Operator)
	}
}

func evaluateContains(value any, expected any) bool {
	switch tv := value.(type) {
	case string:
		needle, ok := expected.(string)
		if !ok {
			return false
		}
		return strings.Contains(tv, needle)
	case []any:
		for _, item := range tv {
			if compareEqual(item, expected) {
				return true
			}
		}
	case []string:
		expectedString, ok := expected.(string)
		if !ok {
			return false
		}
		for _, item := range tv {
			if item == expectedString {
				return true
			}
		}
	}
	return false
}

func evaluateIn(value any, expected any) bool {
	switch tv := expected.(type) {
	case []any:
		for _, candidate := range tv {
			if compareEqual(value, candidate) {
				return true
			}
		}
	case []string:
		valueString, ok := value.(string)
		if !ok {
			return false
		}
		for _, candidate := range tv {
			if valueString == candidate {
				return true
			}
		}
	default:
		expectedValue := reflect.ValueOf(expected)
		if expectedValue.Kind() != reflect.Slice && expectedValue.Kind() != reflect.Array {
			return false
		}
		for i := 0; i < expectedValue.Len(); i++ {
			if compareEqual(value, expectedValue.Index(i).Interface()) {
				return true
			}
		}
	}
	return false
}

func evaluateNumericCompare(value any, expected any, compare func(float64, float64) bool) (bool, error) {
	left, ok := toFloat64(value)
	if !ok {
		return false, nil
	}
	right, ok := toFloat64(expected)
	if !ok {
		return false, fmt.Errorf("numeric operator expected number, got %T", expected)
	}
	return compare(left, right), nil
}

func compareEqual(left any, right any) bool {
	leftNum, leftNumOK := toFloat64(left)
	rightNum, rightNumOK := toFloat64(right)
	if leftNumOK && rightNumOK {
		return math.Abs(leftNum-rightNum) < 1e-9
	}
	return reflect.DeepEqual(left, right)
}

func toFloat64(value any) (float64, bool) {
	switch tv := value.(type) {
	case int:
		return float64(tv), true
	case int8:
		return float64(tv), true
	case int16:
		return float64(tv), true
	case int32:
		return float64(tv), true
	case int64:
		return float64(tv), true
	case uint:
		return float64(tv), true
	case uint8:
		return float64(tv), true
	case uint16:
		return float64(tv), true
	case uint32:
		return float64(tv), true
	case uint64:
		return float64(tv), true
	case float32:
		return float64(tv), true
	case float64:
		return tv, true
	default:
		return 0, false
	}
}

func maxInflightEstimate(fanout, workers int) int {
	if fanout <= 0 || workers <= 0 {
		return 0
	}
	if fanout < workers {
		return fanout
	}
	return workers
}

func countBackpressureDrops(outcomes []types.ToolCallOutcome) (int, map[string]int) {
	if len(outcomes) == 0 {
		return 0, nil
	}
	total := 0
	byPhase := map[string]int{}
	for _, outcome := range outcomes {
		if outcome.Result.Error == nil {
			continue
		}
		reason, _ := outcome.Result.Error.Details["reason"].(string)
		if strings.EqualFold(strings.TrimSpace(reason), "queue_full") {
			total++
			phase := dispatchPhaseFromOutcome(outcome)
			if phase == "" {
				continue
			}
			byPhase[phase]++
		}
	}
	if len(byPhase) == 0 {
		byPhase = nil
	}
	return total, byPhase
}

func anyPhaseFullyDroppedByLowPriority(calls []types.ToolCall, outcomes []types.ToolCallOutcome) bool {
	return len(phasesFullyDroppedByLowPriority(calls, outcomes)) > 0
}

func phasesFullyDroppedByLowPriority(calls []types.ToolCall, outcomes []types.ToolCallOutcome) []types.ActionPhase {
	if len(calls) == 0 {
		return nil
	}
	totals := map[types.ActionPhase]int{}
	callPhaseByID := map[string]types.ActionPhase{}
	for _, call := range calls {
		phase := dispatchPhaseForToolCall(call)
		totals[phase]++
		if strings.TrimSpace(call.CallID) != "" {
			callPhaseByID[strings.TrimSpace(call.CallID)] = phase
		}
	}
	droppedByPhase := map[types.ActionPhase]int{}
	for _, outcome := range outcomes {
		if outcome.Result.Error == nil || len(outcome.Result.Error.Details) == 0 {
			continue
		}
		reason, _ := outcome.Result.Error.Details["reason"].(string)
		dropReason, _ := outcome.Result.Error.Details["drop_reason"].(string)
		if strings.EqualFold(strings.TrimSpace(reason), "queue_full") &&
			strings.EqualFold(strings.TrimSpace(dropReason), "low_priority_dropped") {
			phase := dispatchPhaseForToolOutcome(outcome, callPhaseByID)
			droppedByPhase[phase]++
		}
	}
	failed := make([]types.ActionPhase, 0, len(totals))
	for phase, total := range totals {
		if total <= 0 {
			continue
		}
		if droppedByPhase[phase] == total {
			failed = append(failed, phase)
		}
	}
	return failed
}

func dispatchPhaseForToolCall(call types.ToolCall) types.ActionPhase {
	name := strings.ToLower(strings.TrimSpace(call.Name))
	switch {
	case strings.HasPrefix(name, "mcp."), strings.HasPrefix(name, "local.mcp_"), strings.HasPrefix(name, "local.mcp."):
		return types.ActionPhaseMCP
	case strings.HasPrefix(name, "skill."), strings.HasPrefix(name, "local.skill_"), strings.HasPrefix(name, "local.skill."):
		return types.ActionPhaseSkill
	default:
		return types.ActionPhaseTool
	}
}

func dropRelevantPhases(calls []types.ToolCall) []types.ActionPhase {
	if len(calls) == 0 {
		return nil
	}
	seen := map[types.ActionPhase]struct{}{}
	phases := make([]types.ActionPhase, 0, 3)
	for _, call := range calls {
		phase := dispatchPhaseForToolCall(call)
		if _, ok := seen[phase]; ok {
			continue
		}
		seen[phase] = struct{}{}
		phases = append(phases, phase)
	}
	return phases
}

func dispatchPhaseForToolOutcome(outcome types.ToolCallOutcome, byCallID map[string]types.ActionPhase) types.ActionPhase {
	if outcome.Result.Error != nil && outcome.Result.Error.Details != nil {
		if phase, ok := outcome.Result.Error.Details["dispatch_phase"].(string); ok {
			switch strings.ToLower(strings.TrimSpace(phase)) {
			case string(types.ActionPhaseMCP):
				return types.ActionPhaseMCP
			case string(types.ActionPhaseSkill):
				return types.ActionPhaseSkill
			}
		}
	}
	if phase, ok := byCallID[strings.TrimSpace(outcome.CallID)]; ok {
		return phase
	}
	return dispatchPhaseForToolCall(types.ToolCall{Name: outcome.Name})
}

func dispatchPhaseFromOutcome(outcome types.ToolCallOutcome) string {
	phase := dispatchPhaseForToolOutcome(outcome, nil)
	switch phase {
	case types.ActionPhaseMCP:
		return "mcp"
	case types.ActionPhaseSkill:
		return "skill"
	default:
		return "local"
	}
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

func (e *Engine) clarificationConfig() runtimeconfig.ClarificationConfig {
	if e.runtimeMgr == nil {
		return runtimeconfig.DefaultConfig().Clarification
	}
	return e.runtimeMgr.EffectiveConfig().Clarification
}

func (e *Engine) clarificationTimeout() time.Duration {
	cfg := e.clarificationConfig()
	if cfg.Timeout <= 0 {
		return 30 * time.Second
	}
	return cfg.Timeout
}

func (e *Engine) awaitClarification(
	ctx context.Context,
	h types.EventHandler,
	req types.RunRequest,
	runID string,
	iteration int,
	seq *int64,
	request *types.ClarificationRequest,
	stats *clarificationStats,
) (types.ClarificationResponse, *types.ClassifiedError, error) {
	if request == nil {
		return types.ClarificationResponse{}, nil, nil
	}
	cfg := e.clarificationConfig()
	if !cfg.Enabled {
		msg := "clarification requested but clarification.hitl is disabled"
		return types.ClarificationResponse{}, classified(types.ErrHITL, msg, false), errors.New(msg)
	}
	if e.clarification == nil {
		msg := "clarification requested but resolver is not configured"
		return types.ClarificationResponse{}, classified(types.ErrHITL, msg, false), errors.New(msg)
	}
	timeout := request.Timeout
	if timeout <= 0 {
		timeout = e.clarificationTimeout()
	}
	if stats != nil {
		stats.AwaitCount++
	}
	e.emitTimeline(ctx, h, runID, iteration, seq, types.ActionPhaseHITL, types.ActionStatusPending, "hitl.await_user")
	e.emit(ctx, h, types.Event{
		Version:   types.EventSchemaVersionV1,
		Type:      "hitl.clarification.requested",
		RunID:     runID,
		Iteration: iteration,
		Time:      e.now(),
		Payload: map[string]any{
			"clarification_request": map[string]any{
				"request_id":      strings.TrimSpace(request.RequestID),
				"questions":       request.Questions,
				"context_summary": strings.TrimSpace(request.ContextSummary),
				"timeout_ms":      timeout.Milliseconds(),
			},
		},
	})
	resolveCtx, cancel := context.WithTimeout(ctx, timeout)
	response, err := e.clarification.Resolve(resolveCtx, types.ClarificationResolveRequest{
		RunID:       runID,
		SessionID:   req.SessionID,
		Iteration:   iteration,
		Request:     *request,
		Timeout:     timeout,
		TriggeredBy: "model",
	})
	cancel()
	if err != nil || errors.Is(resolveCtx.Err(), context.DeadlineExceeded) {
		if stats != nil {
			stats.CancelByUserCount++
		}
		e.emitTimeline(ctx, h, runID, iteration, seq, types.ActionPhaseHITL, types.ActionStatusCanceled, "hitl.canceled_by_user")
		ce := classified(types.ErrPolicyTimeout, "clarification timed out and canceled_by_user", false)
		return types.ClarificationResponse{}, ce, context.DeadlineExceeded
	}
	if stats != nil {
		stats.ResumeCount++
	}
	e.emitTimeline(ctx, h, runID, iteration, seq, types.ActionPhaseHITL, types.ActionStatusSucceeded, "hitl.resumed")
	return response, nil, nil
}

func applyClarificationResponse(req types.RunRequest, response types.ClarificationResponse) types.RunRequest {
	if len(response.Answers) == 0 {
		return req
	}
	joined := strings.TrimSpace(strings.Join(response.Answers, "\n"))
	if joined == "" {
		return req
	}
	req.Messages = append(req.Messages, types.Message{
		Role:    "user",
		Content: "clarification:\n" + joined,
	})
	return req
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
