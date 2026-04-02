package runner

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"reflect"
	"regexp"
	"sort"
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
	lifecycleHooks     []types.AgentLifecycleHook
	toolMiddlewares    []types.ToolMiddleware
	skillLoader        types.SkillLoader
	securityAlert      types.SecurityAlertCallback
	securityDeliveryMu sync.Mutex
	securityDelivery   *securityAlertDeliveryExecutor
	sandboxExecutor    types.SandboxExecutor
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
		assembler.WithMemoryConfigProvider(func() runtimeconfig.RuntimeMemoryConfig {
			if e.runtimeMgr != nil {
				return e.runtimeMgr.EffectiveConfig().Runtime.Memory
			}
			return runtimeconfig.DefaultConfig().Runtime.Memory
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
		if len(e.toolMiddlewares) > 0 {
			e.dispatcher.SetMiddlewares(e.toolMiddlewares...)
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
		if e.runtimeMgr != nil && e.sandboxExecutor != nil {
			e.runtimeMgr.SetSandboxExecutor(e.sandboxExecutor)
		}
		if e.dispatcher != nil {
			e.dispatcher.SetRuntimeManager(mgr)
			if len(e.toolMiddlewares) > 0 {
				e.dispatcher.SetMiddlewares(e.toolMiddlewares...)
			}
		}
	}
}

// WithSandboxExecutor injects host-provided sandbox executor and bridges it into runtime manager when available.
func WithSandboxExecutor(executor types.SandboxExecutor) Option {
	return func(e *Engine) {
		e.sandboxExecutor = executor
		if e.runtimeMgr != nil {
			e.runtimeMgr.SetSandboxExecutor(executor)
		}
	}
}

// WithContextAssemblerAgenticRouter injects host callback for CA2 agentic routing decisions.
func WithContextAssemblerAgenticRouter(router assembler.AgenticRouter) Option {
	return func(e *Engine) {
		if e.assembler != nil {
			e.assembler.SetAgenticRouter(router)
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

// WithLifecycleHooks registers lifecycle hooks for before/after reasoning, acting, and reply phases.
func WithLifecycleHooks(hooks ...types.AgentLifecycleHook) Option {
	return func(e *Engine) {
		e.lifecycleHooks = normalizeLifecycleHooks(hooks)
	}
}

// WithToolMiddlewares registers onion-style tool middlewares for local tool dispatch.
func WithToolMiddlewares(middlewares ...types.ToolMiddleware) Option {
	return func(e *Engine) {
		e.toolMiddlewares = normalizeToolMiddlewares(middlewares)
		if e.dispatcher != nil {
			e.dispatcher.SetMiddlewares(e.toolMiddlewares...)
		}
	}
}

// WithSkillLoader registers a skill loader used by run/stream preprocess stage.
func WithSkillLoader(loader types.SkillLoader) Option {
	return func(e *Engine) {
		e.skillLoader = loader
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
	reactCfg := e.runtimeReactConfigSnapshot()
	hooksCfg := e.runtimeHooksConfigSnapshot()
	toolMiddlewareCfg := e.runtimeToolMiddlewareConfigSnapshot()
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
	memoryRuntime := e.runtimeMemorySnapshot()
	memoryAggregate := memoryRunDiagnosticsAccumulator{}
	sandboxRuntime := e.sandboxRuntimeSnapshot()
	sandboxAggregate := sandboxRunDiagnosticsAccumulator{}
	reactToolCallTotal := 0
	reactToolCallBudgetHitTotal := 0
	reactIterationBudgetHitTotal := 0
	reactTerminationReason := ""
	var skillPreprocess skillPreprocessState
	var terminal *types.ClassifiedError
	var runErr error

	e.emit(ctx, h, types.Event{Version: "v1", Type: "run.started", RunID: runID, Time: start})
	e.emitTimeline(ctx, h, runID, 0, &timelineSeq, types.ActionPhaseRun, types.ActionStatusPending, "")
	e.emitTimeline(ctx, h, runID, 0, &timelineSeq, types.ActionPhaseRun, types.ActionStatusRunning, "")
	req, skillPreprocess, terminal, runErr = e.runSkillPreprocess(ctx, req, runID, false, h, &warnings)
	if terminal != nil {
		state = StateAbort
	}

	for {
		switch state {
		case StateInit:
			if iteration >= policy.MaxIterations {
				terminal = classified(types.ErrIterationLimit, "max iterations reached", false)
				runErr = errors.New(terminal.Message)
				reactIterationBudgetHitTotal++
				state = StateAbort
				continue
			}
			state = StateModelStep
		case StateModelStep:
			iteration++
			hookTerminal, hookErr := e.runLifecycleHooks(
				ctx,
				req,
				runID,
				iteration,
				false,
				types.AgentLifecyclePhaseBeforeReasoning,
				"",
				nil,
				h,
				&warnings,
			)
			if hookTerminal != nil {
				terminal = hookTerminal
				runErr = hookErr
				state = StateAbort
				continue
			}
			prepared, prepTerminal, prepErr := e.prepareReactModelStep(
				ctx,
				req,
				runID,
				iteration,
				&timelineSeq,
				pendingOutcomes,
				false,
				h,
				&memoryAggregate,
				&lastSecurity,
			)
			if prepTerminal != nil {
				terminal = prepTerminal
				runErr = prepErr
				state = StateAbort
				continue
			}
			selectedModel := prepared.SelectedModel
			modelReq := prepared.ModelRequest
			selection := prepared.Selection
			lastAssemble = prepared.AssembleResult
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
			hookTerminal, hookErr = e.runLifecycleHooks(
				ctx,
				req,
				runID,
				iteration,
				false,
				types.AgentLifecyclePhaseAfterReasoning,
				resp.FinalAnswer,
				resp.ToolCalls,
				h,
				&warnings,
			)
			if hookTerminal != nil {
				terminal = hookTerminal
				runErr = hookErr
				state = StateAbort
				continue
			}
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
				reactIterationBudgetHitTotal++
				state = StateAbort
				continue
			}
			filteredCalls, whitelistTerminal, whitelistErr := e.enforceSkillWhitelistForToolCalls(
				ctx,
				h,
				runID,
				iteration,
				lastResponse.ToolCalls,
				skillPreprocess,
				&warnings,
			)
			if whitelistTerminal != nil {
				terminal = whitelistTerminal
				runErr = whitelistErr
				state = StateAbort
				continue
			}
			lastResponse.ToolCalls = filteredCalls
			if len(lastResponse.ToolCalls) == 0 {
				state = StateFinalize
				continue
			}
			if policy.ToolCallLimit > 0 && reactToolCallTotal+len(lastResponse.ToolCalls) > policy.ToolCallLimit {
				terminal = classified(types.ErrIterationLimit, "tool call limit exceeded", false)
				runErr = errors.New(terminal.Message)
				reactToolCallBudgetHitTotal++
				state = StateAbort
				continue
			}
			hookTerminal, hookErr := e.runLifecycleHooks(
				ctx,
				req,
				runID,
				iteration,
				false,
				types.AgentLifecyclePhaseBeforeActing,
				lastResponse.FinalAnswer,
				lastResponse.ToolCalls,
				h,
				&warnings,
			)
			if hookTerminal != nil {
				terminal = hookTerminal
				runErr = hookErr
				state = StateAbort
				continue
			}
			if e.dispatcher == nil {
				e.emitTimeline(ctx, h, runID, iteration, &timelineSeq, types.ActionPhaseTool, types.ActionStatusSkipped, "tool_runtime_disabled")
				warnings = append(warnings, "tool calls requested but tool runtime is not enabled")
				state = StateModelStep
				continue
			}
			dispatchResult, dispatchTerminal, dispatchErr := e.dispatchReactToolCalls(
				ctx,
				req,
				h,
				runID,
				iteration,
				&timelineSeq,
				lastResponse.ToolCalls,
				policy,
				&gateStats,
				&concurrencyStats,
				&lastSecurity,
				&warnings,
				&mergedCalls,
				&sandboxAggregate,
				reactToolDispatchOptions{FailOnToolRuntimeDisabled: false},
			)
			if dispatchTerminal != nil {
				terminal = dispatchTerminal
				runErr = dispatchErr
				state = StateAbort
				continue
			}
			if !dispatchResult.Dispatched {
				state = StateModelStep
				continue
			}
			hookTerminal, hookErr = e.runLifecycleHooks(
				ctx,
				req,
				runID,
				iteration,
				false,
				types.AgentLifecyclePhaseAfterActing,
				lastResponse.FinalAnswer,
				lastResponse.ToolCalls,
				h,
				&warnings,
			)
			if hookTerminal != nil {
				terminal = hookTerminal
				runErr = hookErr
				state = StateAbort
				continue
			}
			reactToolCallTotal += len(lastResponse.ToolCalls)
			pendingOutcomes = dispatchResult.Outcomes
			state = StateModelStep
		case StateFinalize:
			hookTerminal, hookErr := e.runLifecycleHooks(
				ctx,
				req,
				runID,
				iteration,
				false,
				types.AgentLifecyclePhaseBeforeReply,
				lastResponse.FinalAnswer,
				nil,
				h,
				&warnings,
			)
			if hookTerminal != nil {
				terminal = hookTerminal
				runErr = hookErr
				state = StateAbort
				continue
			}
			hookTerminal, hookErr = e.runLifecycleHooks(
				ctx,
				req,
				runID,
				iteration,
				false,
				types.AgentLifecyclePhaseAfterReply,
				lastResponse.FinalAnswer,
				nil,
				h,
				&warnings,
			)
			if hookTerminal != nil {
				terminal = hookTerminal
				runErr = hookErr
				state = StateAbort
				continue
			}
			reactTerminationReason = runtimeconfig.RuntimeReactTerminationCompleted
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
					Provider:                          lastSelection.Provider,
					Initial:                           lastSelection.Initial,
					Path:                              selectionPath,
					Required:                          lastSelection.Required,
					UsedFallback:                      fallbackUsed,
					Assemble:                          lastAssemble,
					GateChecks:                        gateStats.Checks,
					GateDenied:                        gateStats.DeniedCount,
					GateTimeout:                       gateStats.TimeoutCount,
					GateRuleHits:                      gateStats.RuleHitCount,
					GateRuleLast:                      gateStats.RuleLastID,
					HitlAwait:                         hitlStats.AwaitCount,
					HitlResumed:                       hitlStats.ResumeCount,
					HitlCanceled:                      hitlStats.CancelByUserCount,
					CancelProp:                        concurrencyStats.CancelPropagatedCount,
					BackDrop:                          concurrencyStats.BackpressureDropCount,
					BackDropByPhase:                   concurrencyStats.BackpressureDropByPhase,
					InflightPeak:                      concurrencyStats.InflightPeak,
					SecurityPolicy:                    lastSecurity.PolicyKind,
					NamespaceTool:                     lastSecurity.NamespaceTool,
					FilterStage:                       lastSecurity.FilterStage,
					SecurityDecision:                  lastSecurity.Decision,
					ReasonCode:                        lastSecurity.ReasonCode,
					Severity:                          lastSecurity.Severity,
					AlertStatus:                       lastSecurity.AlertDispatchStatus,
					AlertFailure:                      lastSecurity.AlertFailureReason,
					AlertDeliveryMode:                 lastSecurity.AlertDeliveryMode,
					AlertRetryCount:                   lastSecurity.AlertRetryCount,
					AlertQueueDropped:                 lastSecurity.AlertQueueDropped,
					AlertQueueDropCount:               lastSecurity.AlertQueueDropCount,
					AlertCircuitState:                 lastSecurity.AlertCircuitState,
					AlertCircuitReason:                lastSecurity.AlertCircuitReason,
					ReactEnabled:                      reactCfg.Enabled,
					ReactIterationTotal:               iteration,
					ReactToolCallTotal:                reactToolCallTotal,
					ReactToolBudgetHit:                reactToolCallBudgetHitTotal,
					ReactIterBudgetHit:                reactIterationBudgetHitTotal,
					ReactTermination:                  reactTerminationReason,
					ReactStreamDispatch:               reactCfg.StreamToolDispatchEnabled,
					A65Recorded:                       true,
					HooksEnabled:                      hooksCfg.Enabled,
					HooksFailMode:                     strings.ToLower(strings.TrimSpace(hooksCfg.FailMode)),
					HooksPhases:                       append([]string(nil), hooksCfg.Phases...),
					ToolMiddlewareEnabled:             toolMiddlewareCfg.Enabled,
					ToolMiddlewareFailMode:            strings.ToLower(strings.TrimSpace(toolMiddlewareCfg.FailMode)),
					SkillDiscoveryMode:                skillPreprocess.DiscoveryMode,
					SkillDiscoveryRoots:               append([]string(nil), skillPreprocess.DiscoveryRoots...),
					SkillPreprocessEnabled:            skillPreprocess.PreprocessEnabled,
					SkillPreprocessPhase:              skillPreprocess.PreprocessPhase,
					SkillPreprocessFailMode:           skillPreprocess.PreprocessFailMode,
					SkillPreprocessStatus:             skillPreprocess.PreprocessStatus,
					SkillPreprocessReasonCode:         skillPreprocess.PreprocessReasonCode,
					SkillPreprocessSpecCount:          skillPreprocess.SpecCount,
					SkillBundlePromptMode:             skillPreprocess.PromptMode,
					SkillBundleWhitelistMode:          skillPreprocess.WhitelistMode,
					SkillBundleConflictPolicy:         skillPreprocess.ConflictPolicy,
					SkillBundlePromptTotal:            skillPreprocess.PromptFragmentCount,
					SkillBundleWhitelistTotal:         skillPreprocess.WhitelistCount,
					SkillBundleWhitelistRejectedTotal: skillPreprocess.WhitelistRejectedTotal,
					Memory:                            memoryAggregate.snapshot(memoryRuntime),
					Sandbox:                           sandboxAggregate.snapshot(sandboxRuntime),
				}),
			})
			return result, nil
		case StateAbort:
			hookTerminal, hookErr := e.runLifecycleHooks(
				ctx,
				req,
				runID,
				iteration,
				false,
				types.AgentLifecyclePhaseBeforeReply,
				lastResponse.FinalAnswer,
				nil,
				h,
				&warnings,
			)
			if terminal == nil && hookTerminal != nil {
				terminal = hookTerminal
				runErr = hookErr
			}
			hookTerminal, hookErr = e.runLifecycleHooks(
				ctx,
				req,
				runID,
				iteration,
				false,
				types.AgentLifecyclePhaseAfterReply,
				lastResponse.FinalAnswer,
				nil,
				h,
				&warnings,
			)
			if terminal == nil && hookTerminal != nil {
				terminal = hookTerminal
				runErr = hookErr
			}
			reactTerminationReason = resolveReactTerminationReason(
				terminal,
				runErr,
				reactToolCallBudgetHitTotal,
				reactIterationBudgetHitTotal,
			)
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
					Provider:                          lastSelection.Provider,
					Initial:                           lastSelection.Initial,
					Path:                              selectionPath,
					Required:                          lastSelection.Required,
					Reason:                            lastSelection.Reason,
					UsedFallback:                      fallbackUsed,
					Assemble:                          lastAssemble,
					GateChecks:                        gateStats.Checks,
					GateDenied:                        gateStats.DeniedCount,
					GateTimeout:                       gateStats.TimeoutCount,
					GateRuleHits:                      gateStats.RuleHitCount,
					GateRuleLast:                      gateStats.RuleLastID,
					HitlAwait:                         hitlStats.AwaitCount,
					HitlResumed:                       hitlStats.ResumeCount,
					HitlCanceled:                      hitlStats.CancelByUserCount,
					CancelProp:                        concurrencyStats.CancelPropagatedCount,
					BackDrop:                          concurrencyStats.BackpressureDropCount,
					BackDropByPhase:                   concurrencyStats.BackpressureDropByPhase,
					InflightPeak:                      concurrencyStats.InflightPeak,
					SecurityPolicy:                    lastSecurity.PolicyKind,
					NamespaceTool:                     lastSecurity.NamespaceTool,
					FilterStage:                       lastSecurity.FilterStage,
					SecurityDecision:                  lastSecurity.Decision,
					ReasonCode:                        lastSecurity.ReasonCode,
					Severity:                          lastSecurity.Severity,
					AlertStatus:                       lastSecurity.AlertDispatchStatus,
					AlertFailure:                      lastSecurity.AlertFailureReason,
					AlertDeliveryMode:                 lastSecurity.AlertDeliveryMode,
					AlertRetryCount:                   lastSecurity.AlertRetryCount,
					AlertQueueDropped:                 lastSecurity.AlertQueueDropped,
					AlertQueueDropCount:               lastSecurity.AlertQueueDropCount,
					AlertCircuitState:                 lastSecurity.AlertCircuitState,
					AlertCircuitReason:                lastSecurity.AlertCircuitReason,
					ReactEnabled:                      reactCfg.Enabled,
					ReactIterationTotal:               iteration,
					ReactToolCallTotal:                reactToolCallTotal,
					ReactToolBudgetHit:                reactToolCallBudgetHitTotal,
					ReactIterBudgetHit:                reactIterationBudgetHitTotal,
					ReactTermination:                  reactTerminationReason,
					ReactStreamDispatch:               reactCfg.StreamToolDispatchEnabled,
					A65Recorded:                       true,
					HooksEnabled:                      hooksCfg.Enabled,
					HooksFailMode:                     strings.ToLower(strings.TrimSpace(hooksCfg.FailMode)),
					HooksPhases:                       append([]string(nil), hooksCfg.Phases...),
					ToolMiddlewareEnabled:             toolMiddlewareCfg.Enabled,
					ToolMiddlewareFailMode:            strings.ToLower(strings.TrimSpace(toolMiddlewareCfg.FailMode)),
					SkillDiscoveryMode:                skillPreprocess.DiscoveryMode,
					SkillDiscoveryRoots:               append([]string(nil), skillPreprocess.DiscoveryRoots...),
					SkillPreprocessEnabled:            skillPreprocess.PreprocessEnabled,
					SkillPreprocessPhase:              skillPreprocess.PreprocessPhase,
					SkillPreprocessFailMode:           skillPreprocess.PreprocessFailMode,
					SkillPreprocessStatus:             skillPreprocess.PreprocessStatus,
					SkillPreprocessReasonCode:         skillPreprocess.PreprocessReasonCode,
					SkillPreprocessSpecCount:          skillPreprocess.SpecCount,
					SkillBundlePromptMode:             skillPreprocess.PromptMode,
					SkillBundleWhitelistMode:          skillPreprocess.WhitelistMode,
					SkillBundleConflictPolicy:         skillPreprocess.ConflictPolicy,
					SkillBundlePromptTotal:            skillPreprocess.PromptFragmentCount,
					SkillBundleWhitelistTotal:         skillPreprocess.WhitelistCount,
					SkillBundleWhitelistRejectedTotal: skillPreprocess.WhitelistRejectedTotal,
					Memory:                            memoryAggregate.snapshot(memoryRuntime),
					Sandbox:                           sandboxAggregate.snapshot(sandboxRuntime),
				}),
			})
			return result, runErr
		}
	}
}

// Stream executes a streaming agent loop while preserving timeline/error semantics with Run.
func (e *Engine) Stream(ctx context.Context, req types.RunRequest, h types.EventHandler) (types.RunResult, error) {
	return e.streamReact(ctx, req, h)
}

//nolint:unused // kept as rollback path for non-react stream loop during staged migrations
func (e *Engine) streamLegacy(ctx context.Context, req types.RunRequest, h types.EventHandler) (types.RunResult, error) {
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
	memoryRuntime := e.runtimeMemorySnapshot()
	memoryAggregate := memoryRunDiagnosticsAccumulator{}
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
				Memory:              memoryAggregate.snapshot(memoryRuntime),
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
	memoryAggregate.observeAssemble(assembleResult)

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
				Memory:              memoryAggregate.snapshot(memoryRuntime),
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
				Memory:              memoryAggregate.snapshot(memoryRuntime),
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
			e.emitTimeline(stepCtx, h, runID, iteration, &timelineSeq, types.ActionPhaseTool, types.ActionStatusSkipped, "tool_dispatch_deferred_to_react_loop")
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
				Memory:              memoryAggregate.snapshot(memoryRuntime),
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
			Memory:              memoryAggregate.snapshot(memoryRuntime),
		}),
	})
	return result, nil
}

func (e *Engine) streamReact(ctx context.Context, req types.RunRequest, h types.EventHandler) (types.RunResult, error) {
	policy := resolvePolicy(req.Policy)
	policy = e.applyRuntimeDefaults(policy, req.Policy)
	reactCfg := e.runtimeReactConfigSnapshot()
	hooksCfg := e.runtimeHooksConfigSnapshot()
	toolMiddlewareCfg := e.runtimeToolMiddlewareConfigSnapshot()
	runID := req.RunID
	if runID == "" {
		runID = e.newRunID()
	}
	start := e.now()
	ctx, runSpan := e.tracer.StartRun(ctx, runID)
	defer runSpan.End()

	iteration := 0
	timelineSeq := int64(0)
	final := ""
	usage := types.TokenUsage{}
	warnings := make([]string, 0)
	mergedCalls := make([]types.ToolCallSummary, 0)
	pendingOutcomes := make([]types.ToolCallOutcome, 0)
	lastSelection := stepModelSelection{}
	lastAssemble := types.ContextAssembleResult{}
	selectionPath := make([]string, 0, 4)
	fallbackUsed := false
	gateStats := actionGateStats{}
	hitlStats := clarificationStats{}
	concurrencyStats := runtimeConcurrencyStats{}
	lastSecurity := securityDecision{}
	memoryRuntime := e.runtimeMemorySnapshot()
	memoryAggregate := memoryRunDiagnosticsAccumulator{}
	sandboxRuntime := e.sandboxRuntimeSnapshot()
	sandboxAggregate := sandboxRunDiagnosticsAccumulator{}
	reactToolCallTotal := 0
	reactToolCallBudgetHitTotal := 0
	reactIterationBudgetHitTotal := 0
	reactTerminationReason := ""
	var skillPreprocess skillPreprocessState

	var terminal *types.ClassifiedError
	var runErr error

	e.emit(ctx, h, types.Event{Version: "v1", Type: "run.started", RunID: runID, Time: start})
	e.emitTimeline(ctx, h, runID, 0, &timelineSeq, types.ActionPhaseRun, types.ActionStatusPending, "")
	e.emitTimeline(ctx, h, runID, 0, &timelineSeq, types.ActionPhaseRun, types.ActionStatusRunning, "")
	req, skillPreprocess, terminal, runErr = e.runSkillPreprocess(ctx, req, runID, true, h, &warnings)

	for terminal == nil {
		if iteration >= policy.MaxIterations {
			terminal = classified(types.ErrIterationLimit, "max iterations reached", false)
			runErr = errors.New(terminal.Message)
			reactIterationBudgetHitTotal++
			break
		}

		iteration++
		hookTerminal, hookErr := e.runLifecycleHooks(
			ctx,
			req,
			runID,
			iteration,
			true,
			types.AgentLifecyclePhaseBeforeReasoning,
			"",
			nil,
			h,
			&warnings,
		)
		if hookTerminal != nil {
			terminal = hookTerminal
			runErr = hookErr
			break
		}
		prepared, prepTerminal, prepErr := e.prepareReactModelStep(
			ctx,
			req,
			runID,
			iteration,
			&timelineSeq,
			pendingOutcomes,
			true,
			h,
			&memoryAggregate,
			&lastSecurity,
		)
		selection := prepared.Selection
		lastSelection = selection
		if selection.Provider != "" {
			selectionPath = append(selectionPath, selection.Provider)
		}
		fallbackUsed = fallbackUsed || selection.UsedFallback
		if prepTerminal != nil {
			terminal = prepTerminal
			runErr = prepErr
			break
		}
		selectedModel := prepared.SelectedModel
		modelReq := prepared.ModelRequest
		lastAssemble = prepared.AssembleResult

		stepResult, stepErr := e.streamModelStep(
			ctx,
			policy,
			selectedModel,
			&req,
			modelReq,
			h,
			runID,
			iteration,
			&timelineSeq,
			selection.Provider,
			selection.UsedFallback,
			&hitlStats,
			&lastSecurity,
		)
		if stepErr != nil {
			var classifiedErr classifiedModelError
			var gateErr *actionGateViolationError
			terminal = classified(types.ErrModel, stepErr.Error(), false)
			runErr = stepErr
			var timelineStatus types.ActionStatus
			var reason string
			switch {
			case errors.As(stepErr, &gateErr) && gateErr.ClassifiedError() != nil:
				terminal = gateErr.ClassifiedError()
				if gateErr.err != nil {
					runErr = gateErr.err
				}
				timelineStatus, reason = classifyClassifiedTimelineError(terminal)
			case errors.As(stepErr, &classifiedErr) && classifiedErr.ClassifiedError() != nil:
				terminal = classifiedErr.ClassifiedError()
				timelineStatus, reason = classifyClassifiedTimelineError(terminal)
			case errors.Is(stepErr, context.Canceled) || errors.Is(ctx.Err(), context.Canceled):
				terminal = classified(types.ErrPolicyTimeout, "model stream canceled", true)
				timelineStatus, reason = types.ActionStatusCanceled, "cancel.propagated"
				concurrencyStats.CancelPropagatedCount++
			case errors.Is(stepErr, context.DeadlineExceeded):
				terminal = classified(types.ErrPolicyTimeout, "model stream timed out", true)
				timelineStatus, reason = types.ActionStatusCanceled, "policy_timeout"
			default:
				timelineStatus, reason = types.ActionStatusFailed, "model_error"
			}
			e.emitTimeline(ctx, h, runID, iteration, &timelineSeq, types.ActionPhaseModel, timelineStatus, reason)
			break
		}
		hookTerminal, hookErr = e.runLifecycleHooks(
			ctx,
			req,
			runID,
			iteration,
			true,
			types.AgentLifecyclePhaseAfterReasoning,
			stepResult.Text,
			stepResult.ToolCalls,
			h,
			&warnings,
		)
		if hookTerminal != nil {
			terminal = hookTerminal
			runErr = hookErr
			break
		}
		filteredCalls, whitelistTerminal, whitelistErr := e.enforceSkillWhitelistForToolCalls(
			ctx,
			h,
			runID,
			iteration,
			stepResult.ToolCalls,
			skillPreprocess,
			&warnings,
		)
		if whitelistTerminal != nil {
			terminal = whitelistTerminal
			runErr = whitelistErr
			break
		}
		stepResult.ToolCalls = filteredCalls

		if len(stepResult.ToolCalls) == 0 {
			final = stepResult.Text
			hookTerminal, hookErr = e.runLifecycleHooks(
				ctx,
				req,
				runID,
				iteration,
				true,
				types.AgentLifecyclePhaseBeforeReply,
				final,
				nil,
				h,
				&warnings,
			)
			if hookTerminal != nil {
				terminal = hookTerminal
				runErr = hookErr
				break
			}
			hookTerminal, hookErr = e.runLifecycleHooks(
				ctx,
				req,
				runID,
				iteration,
				true,
				types.AgentLifecyclePhaseAfterReply,
				final,
				nil,
				h,
				&warnings,
			)
			if hookTerminal != nil {
				terminal = hookTerminal
				runErr = hookErr
				break
			}
			reactTerminationReason = runtimeconfig.RuntimeReactTerminationCompleted
			result := types.RunResult{
				RunID:       runID,
				FinalAnswer: final,
				Iterations:  iteration,
				ToolCalls:   mergedCalls,
				TokenUsage:  usage,
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
					Provider:                          lastSelection.Provider,
					Initial:                           lastSelection.Initial,
					Path:                              selectionPath,
					Required:                          lastSelection.Required,
					UsedFallback:                      fallbackUsed,
					Assemble:                          lastAssemble,
					GateChecks:                        gateStats.Checks,
					GateDenied:                        gateStats.DeniedCount,
					GateTimeout:                       gateStats.TimeoutCount,
					GateRuleHits:                      gateStats.RuleHitCount,
					GateRuleLast:                      gateStats.RuleLastID,
					HitlAwait:                         hitlStats.AwaitCount,
					HitlResumed:                       hitlStats.ResumeCount,
					HitlCanceled:                      hitlStats.CancelByUserCount,
					CancelProp:                        concurrencyStats.CancelPropagatedCount,
					BackDrop:                          concurrencyStats.BackpressureDropCount,
					BackDropByPhase:                   concurrencyStats.BackpressureDropByPhase,
					InflightPeak:                      concurrencyStats.InflightPeak,
					SecurityPolicy:                    lastSecurity.PolicyKind,
					NamespaceTool:                     lastSecurity.NamespaceTool,
					FilterStage:                       lastSecurity.FilterStage,
					SecurityDecision:                  lastSecurity.Decision,
					ReasonCode:                        lastSecurity.ReasonCode,
					Severity:                          lastSecurity.Severity,
					AlertStatus:                       lastSecurity.AlertDispatchStatus,
					AlertFailure:                      lastSecurity.AlertFailureReason,
					AlertDeliveryMode:                 lastSecurity.AlertDeliveryMode,
					AlertRetryCount:                   lastSecurity.AlertRetryCount,
					AlertQueueDropped:                 lastSecurity.AlertQueueDropped,
					AlertQueueDropCount:               lastSecurity.AlertQueueDropCount,
					AlertCircuitState:                 lastSecurity.AlertCircuitState,
					AlertCircuitReason:                lastSecurity.AlertCircuitReason,
					ReactEnabled:                      reactCfg.Enabled,
					ReactIterationTotal:               iteration,
					ReactToolCallTotal:                reactToolCallTotal,
					ReactToolBudgetHit:                reactToolCallBudgetHitTotal,
					ReactIterBudgetHit:                reactIterationBudgetHitTotal,
					ReactTermination:                  reactTerminationReason,
					ReactStreamDispatch:               reactCfg.StreamToolDispatchEnabled,
					A65Recorded:                       true,
					HooksEnabled:                      hooksCfg.Enabled,
					HooksFailMode:                     strings.ToLower(strings.TrimSpace(hooksCfg.FailMode)),
					HooksPhases:                       append([]string(nil), hooksCfg.Phases...),
					ToolMiddlewareEnabled:             toolMiddlewareCfg.Enabled,
					ToolMiddlewareFailMode:            strings.ToLower(strings.TrimSpace(toolMiddlewareCfg.FailMode)),
					SkillDiscoveryMode:                skillPreprocess.DiscoveryMode,
					SkillDiscoveryRoots:               append([]string(nil), skillPreprocess.DiscoveryRoots...),
					SkillPreprocessEnabled:            skillPreprocess.PreprocessEnabled,
					SkillPreprocessPhase:              skillPreprocess.PreprocessPhase,
					SkillPreprocessFailMode:           skillPreprocess.PreprocessFailMode,
					SkillPreprocessStatus:             skillPreprocess.PreprocessStatus,
					SkillPreprocessReasonCode:         skillPreprocess.PreprocessReasonCode,
					SkillPreprocessSpecCount:          skillPreprocess.SpecCount,
					SkillBundlePromptMode:             skillPreprocess.PromptMode,
					SkillBundleWhitelistMode:          skillPreprocess.WhitelistMode,
					SkillBundleConflictPolicy:         skillPreprocess.ConflictPolicy,
					SkillBundlePromptTotal:            skillPreprocess.PromptFragmentCount,
					SkillBundleWhitelistTotal:         skillPreprocess.WhitelistCount,
					SkillBundleWhitelistRejectedTotal: skillPreprocess.WhitelistRejectedTotal,
					Memory:                            memoryAggregate.snapshot(memoryRuntime),
					Sandbox:                           sandboxAggregate.snapshot(sandboxRuntime),
				}),
			})
			return result, nil
		}

		if policy.ToolCallLimit > 0 && reactToolCallTotal+len(stepResult.ToolCalls) > policy.ToolCallLimit {
			terminal = classified(types.ErrIterationLimit, "tool call limit exceeded", false)
			runErr = errors.New(terminal.Message)
			reactToolCallBudgetHitTotal++
			break
		}
		if !reactCfg.StreamToolDispatchEnabled {
			terminal = classified(types.ErrTool, "stream tool dispatch disabled", false)
			runErr = errors.New(terminal.Message)
			break
		}
		hookTerminal, hookErr = e.runLifecycleHooks(
			ctx,
			req,
			runID,
			iteration,
			true,
			types.AgentLifecyclePhaseBeforeActing,
			stepResult.Text,
			stepResult.ToolCalls,
			h,
			&warnings,
		)
		if hookTerminal != nil {
			terminal = hookTerminal
			runErr = hookErr
			break
		}

		dispatchResult, dispatchTerminal, dispatchErr := e.dispatchReactToolCalls(
			ctx,
			req,
			h,
			runID,
			iteration,
			&timelineSeq,
			stepResult.ToolCalls,
			policy,
			&gateStats,
			&concurrencyStats,
			&lastSecurity,
			&warnings,
			&mergedCalls,
			&sandboxAggregate,
			reactToolDispatchOptions{FailOnToolRuntimeDisabled: true},
		)
		if dispatchTerminal != nil {
			terminal = dispatchTerminal
			runErr = dispatchErr
			break
		}
		hookTerminal, hookErr = e.runLifecycleHooks(
			ctx,
			req,
			runID,
			iteration,
			true,
			types.AgentLifecyclePhaseAfterActing,
			stepResult.Text,
			stepResult.ToolCalls,
			h,
			&warnings,
		)
		if hookTerminal != nil {
			terminal = hookTerminal
			runErr = hookErr
			break
		}
		pendingOutcomes = dispatchResult.Outcomes
		reactToolCallTotal += len(stepResult.ToolCalls)
	}

	hookTerminal, hookErr := e.runLifecycleHooks(
		ctx,
		req,
		runID,
		iteration,
		true,
		types.AgentLifecyclePhaseBeforeReply,
		final,
		nil,
		h,
		&warnings,
	)
	if terminal == nil && hookTerminal != nil {
		terminal = hookTerminal
		runErr = hookErr
	}
	hookTerminal, hookErr = e.runLifecycleHooks(
		ctx,
		req,
		runID,
		iteration,
		true,
		types.AgentLifecyclePhaseAfterReply,
		final,
		nil,
		h,
		&warnings,
	)
	if terminal == nil && hookTerminal != nil {
		terminal = hookTerminal
		runErr = hookErr
	}

	if reactTerminationReason == "" {
		reactTerminationReason = resolveReactTerminationReason(
			terminal,
			runErr,
			reactToolCallBudgetHitTotal,
			reactIterationBudgetHitTotal,
		)
	}
	result := types.RunResult{
		RunID:       runID,
		FinalAnswer: final,
		Iterations:  iteration,
		ToolCalls:   mergedCalls,
		TokenUsage:  usage,
		LatencyMs:   e.now().Sub(start).Milliseconds(),
		Warnings:    warnings,
		Error:       terminal,
	}
	errClass := ""
	if terminal != nil {
		errClass = string(terminal.Class)
	}
	status, runReason := classifyRunTerminal(terminal, runErr)
	e.emitTimeline(ctx, h, runID, iteration, &timelineSeq, types.ActionPhaseRun, status, runReason)
	e.emit(ctx, h, types.Event{
		Version:   "v1",
		Type:      "run.finished",
		RunID:     runID,
		Iteration: iteration,
		Time:      e.now(),
		Payload: runFinishedPayload(result, "failed", errClass, runFinishMeta{
			Provider:                          lastSelection.Provider,
			Initial:                           lastSelection.Initial,
			Path:                              selectionPath,
			Required:                          lastSelection.Required,
			Reason:                            lastSelection.Reason,
			UsedFallback:                      fallbackUsed,
			Assemble:                          lastAssemble,
			GateChecks:                        gateStats.Checks,
			GateDenied:                        gateStats.DeniedCount,
			GateTimeout:                       gateStats.TimeoutCount,
			GateRuleHits:                      gateStats.RuleHitCount,
			GateRuleLast:                      gateStats.RuleLastID,
			HitlAwait:                         hitlStats.AwaitCount,
			HitlResumed:                       hitlStats.ResumeCount,
			HitlCanceled:                      hitlStats.CancelByUserCount,
			CancelProp:                        concurrencyStats.CancelPropagatedCount,
			BackDrop:                          concurrencyStats.BackpressureDropCount,
			BackDropByPhase:                   concurrencyStats.BackpressureDropByPhase,
			InflightPeak:                      concurrencyStats.InflightPeak,
			SecurityPolicy:                    lastSecurity.PolicyKind,
			NamespaceTool:                     lastSecurity.NamespaceTool,
			FilterStage:                       lastSecurity.FilterStage,
			SecurityDecision:                  lastSecurity.Decision,
			ReasonCode:                        lastSecurity.ReasonCode,
			Severity:                          lastSecurity.Severity,
			AlertStatus:                       lastSecurity.AlertDispatchStatus,
			AlertFailure:                      lastSecurity.AlertFailureReason,
			AlertDeliveryMode:                 lastSecurity.AlertDeliveryMode,
			AlertRetryCount:                   lastSecurity.AlertRetryCount,
			AlertQueueDropped:                 lastSecurity.AlertQueueDropped,
			AlertQueueDropCount:               lastSecurity.AlertQueueDropCount,
			AlertCircuitState:                 lastSecurity.AlertCircuitState,
			AlertCircuitReason:                lastSecurity.AlertCircuitReason,
			ReactEnabled:                      reactCfg.Enabled,
			ReactIterationTotal:               iteration,
			ReactToolCallTotal:                reactToolCallTotal,
			ReactToolBudgetHit:                reactToolCallBudgetHitTotal,
			ReactIterBudgetHit:                reactIterationBudgetHitTotal,
			ReactTermination:                  reactTerminationReason,
			ReactStreamDispatch:               reactCfg.StreamToolDispatchEnabled,
			A65Recorded:                       true,
			HooksEnabled:                      hooksCfg.Enabled,
			HooksFailMode:                     strings.ToLower(strings.TrimSpace(hooksCfg.FailMode)),
			HooksPhases:                       append([]string(nil), hooksCfg.Phases...),
			ToolMiddlewareEnabled:             toolMiddlewareCfg.Enabled,
			ToolMiddlewareFailMode:            strings.ToLower(strings.TrimSpace(toolMiddlewareCfg.FailMode)),
			SkillDiscoveryMode:                skillPreprocess.DiscoveryMode,
			SkillDiscoveryRoots:               append([]string(nil), skillPreprocess.DiscoveryRoots...),
			SkillPreprocessEnabled:            skillPreprocess.PreprocessEnabled,
			SkillPreprocessPhase:              skillPreprocess.PreprocessPhase,
			SkillPreprocessFailMode:           skillPreprocess.PreprocessFailMode,
			SkillPreprocessStatus:             skillPreprocess.PreprocessStatus,
			SkillPreprocessReasonCode:         skillPreprocess.PreprocessReasonCode,
			SkillPreprocessSpecCount:          skillPreprocess.SpecCount,
			SkillBundlePromptMode:             skillPreprocess.PromptMode,
			SkillBundleWhitelistMode:          skillPreprocess.WhitelistMode,
			SkillBundleConflictPolicy:         skillPreprocess.ConflictPolicy,
			SkillBundlePromptTotal:            skillPreprocess.PromptFragmentCount,
			SkillBundleWhitelistTotal:         skillPreprocess.WhitelistCount,
			SkillBundleWhitelistRejectedTotal: skillPreprocess.WhitelistRejectedTotal,
			Memory:                            memoryAggregate.snapshot(memoryRuntime),
			Sandbox:                           sandboxAggregate.snapshot(sandboxRuntime),
		}),
	})
	return result, runErr
}

type streamModelStepResult struct {
	Text      string
	ToolCalls []types.ToolCall
}

type reactPreparedModelStep struct {
	SelectedModel  types.ModelClient
	ModelRequest   types.ModelRequest
	Selection      stepModelSelection
	AssembleResult types.ContextAssembleResult
}

type reactToolDispatchOptions struct {
	FailOnToolRuntimeDisabled bool
}

type reactToolDispatchResult struct {
	Outcomes   []types.ToolCallOutcome
	Dispatched bool
}

func (e *Engine) prepareReactModelStep(
	ctx context.Context,
	req types.RunRequest,
	runID string,
	iteration int,
	seq *int64,
	pendingOutcomes []types.ToolCallOutcome,
	stream bool,
	h types.EventHandler,
	memoryAggregate *memoryRunDiagnosticsAccumulator,
	lastSecurity *securityDecision,
) (reactPreparedModelStep, *types.ClassifiedError, error) {
	required := req.Capabilities.Normalized()
	if stream {
		required = append(required, types.ModelCapabilityStreaming)
	}
	modelReq := toModelRequest(runID, req, pendingOutcomes, required)
	selectedModel, selection, selErr := e.selectModelForStep(ctx, modelReq, stream, len(required) > 0)
	if selErr != nil {
		return reactPreparedModelStep{}, selErr, errors.New(selErr.Message)
	}

	contextPhaseEnabled := e.contextAssemblerEnabled()
	if contextPhaseEnabled {
		e.emitTimeline(ctx, h, runID, iteration, seq, types.ActionPhaseContextAssembler, types.ActionStatusPending, "")
		e.emitTimeline(ctx, h, runID, iteration, seq, types.ActionPhaseContextAssembler, types.ActionStatusRunning, "")
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
	if memoryAggregate != nil {
		memoryAggregate.observeAssemble(assembleResult)
	}
	if assembleErr != nil {
		if contextPhaseEnabled {
			status, reason := classifyTimelineError(assembleErr)
			e.emitTimeline(ctx, h, runID, iteration, seq, types.ActionPhaseContextAssembler, status, reason)
		}
		terminal := classified(types.ErrContext, assembleErr.Error(), false)
		return reactPreparedModelStep{}, terminal, assembleErr
	}
	if contextPhaseEnabled {
		e.emitTimeline(ctx, h, runID, iteration, seq, types.ActionPhaseContextAssembler, types.ActionStatusSucceeded, "")
	}

	filteredReq, filterDecision, filterTerminal, filterErr := e.applyInputFilters(ctx, runID, iteration, assembledReq)
	if filterDecision != nil && lastSecurity != nil {
		*lastSecurity = *filterDecision
	}
	if filterTerminal != nil {
		reason := ""
		if lastSecurity != nil {
			reason = lastSecurity.ReasonCode
		}
		e.emitTimeline(ctx, h, runID, iteration, seq, types.ActionPhaseModel, types.ActionStatusFailed, reason)
		return reactPreparedModelStep{}, filterTerminal, filterErr
	}

	return reactPreparedModelStep{
		SelectedModel:  selectedModel,
		ModelRequest:   filteredReq,
		Selection:      selection,
		AssembleResult: assembleResult,
	}, nil, nil
}

func (e *Engine) streamModelStep(
	ctx context.Context,
	policy types.LoopPolicy,
	selectedModel types.ModelClient,
	req *types.RunRequest,
	modelReq types.ModelRequest,
	h types.EventHandler,
	runID string,
	iteration int,
	seq *int64,
	modelProvider string,
	fallbackUsed bool,
	hitlStats *clarificationStats,
	lastSecurity *securityDecision,
) (streamModelStepResult, error) {
	e.emitTimeline(ctx, h, runID, iteration, seq, types.ActionPhaseModel, types.ActionStatusPending, "")
	e.emitTimeline(ctx, h, runID, iteration, seq, types.ActionPhaseModel, types.ActionStatusRunning, "")
	e.emit(ctx, h, types.Event{
		Version:   "v1",
		Type:      "model.requested",
		RunID:     runID,
		Iteration: iteration,
		Time:      e.now(),
		Payload: map[string]any{
			"model_provider": modelProvider,
			"fallback_used":  fallbackUsed,
		},
	})

	stepCtx, cancel := context.WithTimeout(ctx, policy.StepTimeout)
	modelCtx, modelSpan := e.tracer.StartStep(stepCtx, "model.stream", attribute.Int("iteration.index", iteration))
	defer func() {
		modelSpan.End()
		cancel()
	}()

	stepText := strings.Builder{}
	stepToolCalls := make([]types.ToolCall, 0)
	streamErr := selectedModel.Stream(modelCtx, modelReq, func(ev types.ModelEvent) error {
		normalizedEvent := ev
		if normalizedEvent.Type == types.ModelEventTypeFinalAnswer || normalizedEvent.Type == types.ModelEventTypeOutputTextDelta {
			filteredOutput, filterDecision, filterTerminal, filterErr := e.applyOutputFilters(stepCtx, runID, iteration, normalizedEvent.TextDelta)
			if filterDecision != nil && lastSecurity != nil {
				*lastSecurity = *filterDecision
			}
			if filterTerminal != nil {
				return &actionGateViolationError{
					classified: filterTerminal,
					err:        filterErr,
				}
			}
			normalizedEvent.TextDelta = filteredOutput
			stepText.WriteString(normalizedEvent.TextDelta)
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

		if normalizedEvent.ClarificationRequest != nil || normalizedEvent.Type == types.ModelEventTypeClarificationRequest {
			request := normalizedEvent.ClarificationRequest
			if request == nil {
				request = &types.ClarificationRequest{}
			}
			if request.Timeout <= 0 {
				request.Timeout = e.clarificationTimeout()
			}
			baseReq := types.RunRequest{}
			if req != nil {
				baseReq = *req
			}
			clarification, hitlTerminal, hitlErr := e.awaitClarification(
				stepCtx,
				h,
				baseReq,
				runID,
				iteration,
				seq,
				request,
				hitlStats,
			)
			if hitlTerminal != nil {
				return &actionGateViolationError{
					classified: hitlTerminal,
					err:        hitlErr,
				}
			}
			if req != nil {
				*req = applyClarificationResponse(baseReq, clarification)
			}
		}

		if normalizedEvent.ToolCall != nil {
			stepToolCalls = mergeStreamToolCall(stepToolCalls, *normalizedEvent.ToolCall)
		}
		return nil
	})
	if streamErr != nil {
		return streamModelStepResult{}, streamErr
	}

	e.emit(ctx, h, types.Event{Version: "v1", Type: "model.completed", RunID: runID, Iteration: iteration, Time: e.now()})
	e.emitTimeline(ctx, h, runID, iteration, seq, types.ActionPhaseModel, types.ActionStatusSucceeded, "")
	return streamModelStepResult{
		Text:      stepText.String(),
		ToolCalls: stepToolCalls,
	}, nil
}

func (e *Engine) dispatchReactToolCalls(
	ctx context.Context,
	req types.RunRequest,
	h types.EventHandler,
	runID string,
	iteration int,
	seq *int64,
	toolCalls []types.ToolCall,
	policy types.LoopPolicy,
	gateStats *actionGateStats,
	concurrencyStats *runtimeConcurrencyStats,
	lastSecurity *securityDecision,
	warnings *[]string,
	mergedCalls *[]types.ToolCallSummary,
	sandboxAggregate *sandboxRunDiagnosticsAccumulator,
	options reactToolDispatchOptions,
) (reactToolDispatchResult, *types.ClassifiedError, error) {
	toolSecurityDecision, toolSecurityTerminal, toolSecurityErr := e.enforceToolSecurityForCalls(
		ctx,
		h,
		runID,
		iteration,
		seq,
		toolCalls,
	)
	if toolSecurityDecision != nil && lastSecurity != nil {
		*lastSecurity = *toolSecurityDecision
	}
	if toolSecurityTerminal != nil {
		return reactToolDispatchResult{}, toolSecurityTerminal, toolSecurityErr
	}

	gateTerm, gateRunErr := e.enforceActionGateForToolCalls(
		ctx,
		h,
		req,
		runID,
		iteration,
		seq,
		toolCalls,
		gateStats,
	)
	if gateTerm != nil {
		return reactToolDispatchResult{}, gateTerm, gateRunErr
	}
	if e.dispatcher == nil {
		e.emitTimeline(ctx, h, runID, iteration, seq, types.ActionPhaseTool, types.ActionStatusSkipped, "tool_runtime_disabled")
		if warnings != nil {
			*warnings = append(*warnings, "tool calls requested but tool runtime is not enabled")
		}
		if !options.FailOnToolRuntimeDisabled {
			return reactToolDispatchResult{Dispatched: false}, nil, nil
		}
		terminal := classified(types.ErrTool, "tool runtime is not enabled", false)
		return reactToolDispatchResult{}, terminal, errors.New(terminal.Message)
	}

	e.emitTimeline(ctx, h, runID, iteration, seq, types.ActionPhaseTool, types.ActionStatusPending, "")
	e.emitTimeline(ctx, h, runID, iteration, seq, types.ActionPhaseTool, types.ActionStatusRunning, "")
	dispatchCfg := local.DispatchConfig{
		MaxCalls:     policy.MaxToolCallsPerIteration,
		Concurrency:  policy.LocalDispatch.MaxWorkers,
		FailFast:     !policy.ContinueOnToolError,
		QueueSize:    policy.LocalDispatch.QueueSize,
		Backpressure: policy.LocalDispatch.Backpressure,
		Retry:        policy.ToolRetry,
	}
	if concurrencyStats != nil {
		concurrencyStats.ObserveInflight(maxInflightEstimate(len(toolCalls), dispatchCfg.Concurrency))
	}
	if dispatchCfg.Backpressure == types.BackpressureBlock && len(toolCalls) > dispatchCfg.QueueSize {
		e.emitTimeline(ctx, h, runID, iteration, seq, types.ActionPhaseTool, types.ActionStatusPending, "backpressure.block")
	}
	if dispatchCfg.Backpressure == types.BackpressureDropLowPriority && len(toolCalls) > dispatchCfg.QueueSize {
		e.emitTimeline(ctx, h, runID, iteration, seq, types.ActionPhaseTool, types.ActionStatusPending, "backpressure.drop_low_priority")
		for _, phase := range dropRelevantPhases(toolCalls) {
			if phase == types.ActionPhaseTool {
				continue
			}
			e.emitTimeline(ctx, h, runID, iteration, seq, phase, types.ActionStatusPending, "backpressure.drop_low_priority")
		}
	}
	e.emit(ctx, h, types.Event{
		Version:   "v1",
		Type:      "tool.dispatch.started",
		RunID:     runID,
		Iteration: iteration,
		Time:      e.now(),
		Payload: map[string]any{
			"fanout":       len(toolCalls),
			"max_calls":    dispatchCfg.MaxCalls,
			"workers":      dispatchCfg.Concurrency,
			"queue_size":   dispatchCfg.QueueSize,
			"backpressure": dispatchCfg.Backpressure,
			"retry":        dispatchCfg.Retry,
		},
	})

	stepCtx, cancel := context.WithTimeout(ctx, policy.StepTimeout)
	toolCtx, toolSpan := e.tracer.StartStep(stepCtx, "tool.dispatch", attribute.Int("iteration.index", iteration))
	outcomes, dispatchErr := e.dispatcher.Dispatch(toolCtx, toolCalls, dispatchCfg)
	toolSpan.End()
	cancel()

	dropTotal, dropByPhase := countBackpressureDrops(outcomes)
	if concurrencyStats != nil {
		concurrencyStats.BackpressureDropCount += dropTotal
		concurrencyStats.AddBackpressureDropByPhase(dropByPhase)
	}
	if dispatchCfg.Backpressure == types.BackpressureDropLowPriority {
		for _, phase := range phasesFullyDroppedByLowPriority(toolCalls, outcomes) {
			e.emitTimeline(ctx, h, runID, iteration, seq, phase, types.ActionStatusFailed, "backpressure.drop_low_priority")
		}
	}
	if dispatchCfg.Backpressure == types.BackpressureDropLowPriority && anyPhaseFullyDroppedByLowPriority(toolCalls, outcomes) {
		e.emitTimeline(ctx, h, runID, iteration, seq, types.ActionPhaseTool, types.ActionStatusFailed, "backpressure.drop_low_priority")
		terminal := classified(types.ErrTool, "all tool calls dropped by low-priority backpressure", false)
		return reactToolDispatchResult{}, terminal, errors.New(terminal.Message)
	}
	if errors.Is(dispatchErr, context.Canceled) || errors.Is(ctx.Err(), context.Canceled) {
		if concurrencyStats != nil {
			concurrencyStats.CancelPropagatedCount++
		}
		e.emitTimeline(ctx, h, runID, iteration, seq, types.ActionPhaseTool, types.ActionStatusCanceled, "cancel.propagated")
		terminal := classified(types.ErrPolicyTimeout, "tool dispatch canceled", true)
		return reactToolDispatchResult{}, terminal, context.Canceled
	}
	if dispatchErr != nil && errors.Is(stepCtx.Err(), context.DeadlineExceeded) {
		e.emitTimeline(ctx, h, runID, iteration, seq, types.ActionPhaseTool, types.ActionStatusCanceled, "policy_timeout")
		terminal := classified(types.ErrPolicyTimeout, "tool dispatch timed out", true)
		return reactToolDispatchResult{}, terminal, stepCtx.Err()
	}
	if errors.Is(dispatchErr, context.DeadlineExceeded) {
		e.emitTimeline(ctx, h, runID, iteration, seq, types.ActionPhaseTool, types.ActionStatusCanceled, "policy_timeout")
		terminal := classified(types.ErrPolicyTimeout, "tool dispatch timed out", true)
		return reactToolDispatchResult{}, terminal, dispatchErr
	}

	for _, o := range outcomes {
		if mergedCalls != nil {
			*mergedCalls = append(*mergedCalls, types.ToolCallSummary{CallID: o.CallID, Name: o.Name, Error: o.Result.Error})
		}
		if warnings != nil && o.Result.Error != nil {
			*warnings = append(*warnings, o.Result.Error.Message)
		}
	}
	if sandboxAggregate != nil {
		sandboxAggregate.observeOutcomes(outcomes)
	}
	if dispatchErr != nil {
		primaryErr := primaryToolOutcomeError(outcomes)
		reason := "dispatch_error"
		terminal := classified(types.ErrTool, dispatchErr.Error(), false)
		if primaryErr != nil {
			if timelineReason := toolDispatchTimelineReason(primaryErr); timelineReason != "" {
				reason = timelineReason
			}
			terminal = &types.ClassifiedError{
				Class:     primaryErr.Class,
				Message:   strings.TrimSpace(primaryErr.Message),
				Retryable: false,
				Details:   cloneDetailsMap(primaryErr.Details),
			}
			if terminal.Message == "" {
				terminal.Message = dispatchErr.Error()
			}
			if decision, ok := securityDecisionFromToolError(primaryErr); ok {
				resolved := e.finalizeSecurityDecision(ctx, runID, iteration, decision)
				if lastSecurity != nil {
					*lastSecurity = resolved
				}
				terminal = e.securityDeniedError(terminal.Message, resolved, cloneDetailsMap(terminal.Details))
			}
		}
		e.emitTimeline(ctx, h, runID, iteration, seq, types.ActionPhaseTool, types.ActionStatusFailed, reason)
		return reactToolDispatchResult{}, terminal, dispatchErr
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
			"fanout":       len(toolCalls),
			"completed":    len(outcomes),
			"failed":       failed,
			"backpressure": dispatchCfg.Backpressure,
		},
	})
	e.emitTimeline(
		ctx,
		h,
		runID,
		iteration,
		seq,
		types.ActionPhaseTool,
		types.ActionStatusSucceeded,
		toolDispatchObservedSandboxReason(outcomes),
	)
	return reactToolDispatchResult{
		Outcomes:   outcomes,
		Dispatched: true,
	}, nil, nil
}

func resolvePolicy(p *types.LoopPolicy) types.LoopPolicy {
	if p == nil {
		return types.DefaultLoopPolicy()
	}
	policy := *p
	def := types.DefaultLoopPolicy()
	if policy.ToolCallLimit <= 0 {
		policy.ToolCallLimit = def.ToolCallLimit
	}
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
	if cfg.Runtime.React.MaxIterations > 0 {
		policy.MaxIterations = cfg.Runtime.React.MaxIterations
	}
	if cfg.Runtime.React.ToolCallLimit > 0 {
		policy.ToolCallLimit = cfg.Runtime.React.ToolCallLimit
	}
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

//nolint:unused // used by legacy stream fallback path retained for staged rollback safety
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
	Provider                          string
	Initial                           string
	Path                              []string
	Required                          []types.ModelCapability
	UsedFallback                      bool
	Reason                            string
	Assemble                          types.ContextAssembleResult
	GateChecks                        int
	GateDenied                        int
	GateTimeout                       int
	GateRuleHits                      int
	GateRuleLast                      string
	HitlAwait                         int
	HitlResumed                       int
	HitlCanceled                      int
	CancelProp                        int
	BackDrop                          int
	BackDropByPhase                   map[string]int
	InflightPeak                      int
	SecurityPolicy                    string
	NamespaceTool                     string
	FilterStage                       string
	SecurityDecision                  string
	ReasonCode                        string
	Severity                          string
	AlertStatus                       string
	AlertFailure                      string
	AlertDeliveryMode                 string
	AlertRetryCount                   int
	AlertQueueDropped                 bool
	AlertQueueDropCount               int
	AlertCircuitState                 string
	AlertCircuitReason                string
	ReactEnabled                      bool
	ReactIterationTotal               int
	ReactToolCallTotal                int
	ReactToolBudgetHit                int
	ReactIterBudgetHit                int
	ReactTermination                  string
	ReactStreamDispatch               bool
	A65Recorded                       bool
	HooksEnabled                      bool
	HooksFailMode                     string
	HooksPhases                       []string
	ToolMiddlewareEnabled             bool
	ToolMiddlewareFailMode            string
	SkillDiscoveryMode                string
	SkillDiscoveryRoots               []string
	SkillPreprocessEnabled            bool
	SkillPreprocessPhase              string
	SkillPreprocessFailMode           string
	SkillPreprocessStatus             string
	SkillPreprocessReasonCode         string
	SkillPreprocessSpecCount          int
	SkillBundlePromptMode             string
	SkillBundleWhitelistMode          string
	SkillBundleConflictPolicy         string
	SkillBundlePromptTotal            int
	SkillBundleWhitelistTotal         int
	SkillBundleWhitelistRejectedTotal int
	Memory                            memoryRunDiagnostics
	Sandbox                           sandboxRunDiagnostics
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
	if meta.Assemble.Stage.Stage2RouterMode != "" {
		payload["stage2_router_mode"] = meta.Assemble.Stage.Stage2RouterMode
	}
	if meta.Assemble.Stage.Stage2RouterDecision != "" {
		payload["stage2_router_decision"] = meta.Assemble.Stage.Stage2RouterDecision
	}
	if meta.Assemble.Stage.Stage2RouterReason != "" {
		payload["stage2_router_reason"] = meta.Assemble.Stage.Stage2RouterReason
	}
	if meta.Assemble.Stage.Stage2RouterLatencyMs > 0 {
		payload["stage2_router_latency_ms"] = meta.Assemble.Stage.Stage2RouterLatencyMs
	}
	if meta.Assemble.Stage.Stage2RouterError != "" {
		payload["stage2_router_error"] = meta.Assemble.Stage.Stage2RouterError
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
	payload["react_enabled"] = meta.ReactEnabled
	payload["react_iteration_total"] = meta.ReactIterationTotal
	payload["react_tool_call_total"] = meta.ReactToolCallTotal
	payload["react_tool_call_budget_hit_total"] = meta.ReactToolBudgetHit
	payload["react_iteration_budget_hit_total"] = meta.ReactIterBudgetHit
	payload["react_stream_dispatch_enabled"] = meta.ReactStreamDispatch
	if meta.ReactTermination != "" {
		payload["react_termination_reason"] = meta.ReactTermination
	}
	if meta.A65Recorded {
		payload["hooks_enabled"] = meta.HooksEnabled
		if strings.TrimSpace(meta.HooksFailMode) != "" {
			payload["hooks_fail_mode"] = strings.ToLower(strings.TrimSpace(meta.HooksFailMode))
		}
		if len(meta.HooksPhases) > 0 {
			payload["hooks_phases"] = cloneNormalizedStringSlice(meta.HooksPhases)
		}
		payload["tool_middleware_enabled"] = meta.ToolMiddlewareEnabled
		if strings.TrimSpace(meta.ToolMiddlewareFailMode) != "" {
			payload["tool_middleware_fail_mode"] = strings.ToLower(strings.TrimSpace(meta.ToolMiddlewareFailMode))
		}
		if strings.TrimSpace(meta.SkillDiscoveryMode) != "" {
			payload["skill_discovery_mode"] = strings.ToLower(strings.TrimSpace(meta.SkillDiscoveryMode))
		}
		if len(meta.SkillDiscoveryRoots) > 0 {
			payload["skill_discovery_roots"] = append([]string(nil), meta.SkillDiscoveryRoots...)
		}
		payload["skill_preprocess_enabled"] = meta.SkillPreprocessEnabled
		if strings.TrimSpace(meta.SkillPreprocessPhase) != "" {
			payload["skill_preprocess_phase"] = strings.ToLower(strings.TrimSpace(meta.SkillPreprocessPhase))
		}
		if strings.TrimSpace(meta.SkillPreprocessFailMode) != "" {
			payload["skill_preprocess_fail_mode"] = strings.ToLower(strings.TrimSpace(meta.SkillPreprocessFailMode))
		}
		if strings.TrimSpace(meta.SkillPreprocessStatus) != "" {
			payload["skill_preprocess_status"] = strings.ToLower(strings.TrimSpace(meta.SkillPreprocessStatus))
		}
		if strings.TrimSpace(meta.SkillPreprocessReasonCode) != "" {
			payload["skill_preprocess_reason_code"] = strings.ToLower(strings.TrimSpace(meta.SkillPreprocessReasonCode))
		}
		payload["skill_preprocess_spec_count"] = meta.SkillPreprocessSpecCount
		if strings.TrimSpace(meta.SkillBundlePromptMode) != "" {
			payload["skill_bundle_prompt_mode"] = strings.ToLower(strings.TrimSpace(meta.SkillBundlePromptMode))
		}
		if strings.TrimSpace(meta.SkillBundleWhitelistMode) != "" {
			payload["skill_bundle_whitelist_mode"] = strings.ToLower(strings.TrimSpace(meta.SkillBundleWhitelistMode))
		}
		if strings.TrimSpace(meta.SkillBundleConflictPolicy) != "" {
			payload["skill_bundle_conflict_policy"] = strings.ToLower(strings.TrimSpace(meta.SkillBundleConflictPolicy))
		}
		payload["skill_bundle_prompt_total"] = meta.SkillBundlePromptTotal
		payload["skill_bundle_whitelist_total"] = meta.SkillBundleWhitelistTotal
		payload["skill_bundle_whitelist_rejected_total"] = meta.SkillBundleWhitelistRejectedTotal
	}
	if meta.Memory.Observed {
		if meta.Memory.Mode != "" {
			payload["memory_mode"] = meta.Memory.Mode
		}
		if meta.Memory.Provider != "" {
			payload["memory_provider"] = meta.Memory.Provider
		}
		if meta.Memory.Profile != "" {
			payload["memory_profile"] = meta.Memory.Profile
		}
		if meta.Memory.ContractVersion != "" {
			payload["memory_contract_version"] = meta.Memory.ContractVersion
		}
		payload["memory_query_total"] = meta.Memory.QueryTotal
		payload["memory_upsert_total"] = meta.Memory.UpsertTotal
		payload["memory_delete_total"] = meta.Memory.DeleteTotal
		payload["memory_error_total"] = meta.Memory.ErrorTotal
		payload["memory_fallback_total"] = meta.Memory.FallbackTotal
		if meta.Memory.FallbackReasonCode != "" {
			payload["memory_fallback_reason_code"] = meta.Memory.FallbackReasonCode
		}
		payload["memory_latency_ms_p95"] = meta.Memory.LatencyMsP95
		if meta.Memory.ScopeSelected != "" {
			payload["memory_scope_selected"] = meta.Memory.ScopeSelected
		}
		payload["memory_budget_used"] = meta.Memory.BudgetUsed
		payload["memory_hits"] = meta.Memory.Hits
		if len(meta.Memory.RerankStats) > 0 {
			payload["memory_rerank_stats"] = cloneIntMap(meta.Memory.RerankStats)
		}
		if meta.Memory.LifecycleAction != "" {
			payload["memory_lifecycle_action"] = meta.Memory.LifecycleAction
		}
	}
	if meta.Sandbox.Observed {
		if meta.Sandbox.Mode != "" {
			payload["sandbox_mode"] = meta.Sandbox.Mode
		}
		if meta.Sandbox.Backend != "" {
			payload["sandbox_backend"] = meta.Sandbox.Backend
		}
		if meta.Sandbox.Profile != "" {
			payload["sandbox_profile"] = meta.Sandbox.Profile
		}
		if meta.Sandbox.SessionMode != "" {
			payload["sandbox_session_mode"] = meta.Sandbox.SessionMode
		}
		if len(meta.Sandbox.RequiredCapabilities) > 0 {
			payload["sandbox_required_capabilities"] = append([]string(nil), meta.Sandbox.RequiredCapabilities...)
		}
		if meta.Sandbox.Decision != "" {
			payload["sandbox_decision"] = meta.Sandbox.Decision
		}
		if meta.Sandbox.ReasonCode != "" {
			payload["sandbox_reason_code"] = meta.Sandbox.ReasonCode
		}
		if meta.Sandbox.EgressAction != "" {
			payload["sandbox_egress_action"] = meta.Sandbox.EgressAction
		}
		if meta.Sandbox.EgressPolicySource != "" {
			payload["sandbox_egress_policy_source"] = meta.Sandbox.EgressPolicySource
		}
		payload["sandbox_egress_violation_total"] = meta.Sandbox.EgressViolationTotal
		payload["sandbox_fallback_used"] = meta.Sandbox.FallbackUsed
		if meta.Sandbox.FallbackReason != "" {
			payload["sandbox_fallback_reason"] = meta.Sandbox.FallbackReason
		}
		payload["sandbox_timeout_total"] = meta.Sandbox.TimeoutTotal
		payload["sandbox_launch_failed_total"] = meta.Sandbox.LaunchFailedTotal
		payload["sandbox_capability_mismatch_total"] = meta.Sandbox.CapabilityMismatchTotal
		payload["sandbox_queue_wait_ms_p95"] = meta.Sandbox.QueueWaitMsP95
		payload["sandbox_exec_latency_ms_p95"] = meta.Sandbox.ExecLatencyMsP95
		if meta.Sandbox.HasExitCode {
			payload["sandbox_exit_code_last"] = meta.Sandbox.ExitCodeLast
		}
		payload["sandbox_oom_total"] = meta.Sandbox.OOMTotal
		payload["sandbox_resource_cpu_ms_total"] = meta.Sandbox.ResourceCPUMsTotal
		payload["sandbox_resource_memory_peak_bytes_p95"] = meta.Sandbox.ResourceMemoryPeakBytesP95
	}
	if result.Error != nil {
		overlayRunFinishedPayloadFromErrorDetails(payload, result.Error.Details)
	}
	return payload
}

func overlayRunFinishedPayloadFromErrorDetails(payload map[string]any, details map[string]any) {
	if len(payload) == 0 || len(details) == 0 {
		return
	}
	keys := []string{
		"policy_kind",
		"namespace_tool",
		"filter_stage",
		"decision",
		"reason_code",
		"severity",
		"policy_precedence_version",
		"winner_stage",
		"deny_source",
		"tie_break_reason",
	}
	for i := range keys {
		key := keys[i]
		if _, exists := payload[key]; exists {
			continue
		}
		if value, ok := details[key].(string); ok && strings.TrimSpace(value) != "" {
			payload[key] = strings.TrimSpace(value)
		}
	}
	if _, exists := payload["policy_decision_path"]; exists {
		return
	}
	raw, ok := details["policy_decision_path"]
	if !ok {
		return
	}
	switch value := raw.(type) {
	case []runtimeconfig.RuntimePolicyCandidate:
		if len(value) == 0 {
			return
		}
		payload["policy_decision_path"] = append([]runtimeconfig.RuntimePolicyCandidate(nil), value...)
	case []any:
		if len(value) == 0 {
			return
		}
		payload["policy_decision_path"] = append([]any(nil), value...)
	}
}

type memoryRuntimeSnapshot struct {
	Mode            string
	Provider        string
	Profile         string
	ContractVersion string
}

type memoryRunDiagnostics struct {
	Observed           bool
	Mode               string
	Provider           string
	Profile            string
	ContractVersion    string
	QueryTotal         int
	UpsertTotal        int
	DeleteTotal        int
	ErrorTotal         int
	FallbackTotal      int
	FallbackReasonCode string
	LatencyMsP95       int64
	ScopeSelected      string
	BudgetUsed         int
	Hits               int
	RerankStats        map[string]int
	LifecycleAction    string
}

type memoryRunDiagnosticsAccumulator struct {
	observed           bool
	queryTotal         int
	upsertTotal        int
	deleteTotal        int
	errorTotal         int
	fallbackTotal      int
	fallbackReasonCode string
	latencySamples     []int64
	scopeSelected      string
	budgetUsed         int
	hits               int
	rerankStats        map[string]int
	lifecycleAction    string
}

func (a *memoryRunDiagnosticsAccumulator) observeAssemble(assemble types.ContextAssembleResult) {
	if a == nil {
		return
	}
	stage := assemble.Stage
	provider := strings.ToLower(strings.TrimSpace(stage.Stage2Provider))
	reasonCode := strings.ToLower(strings.TrimSpace(stage.Stage2ReasonCode))
	if provider != runtimeconfig.ContextStage2ProviderMemory && !strings.HasPrefix(reasonCode, "memory.") && reasonCode != "memory_error" {
		return
	}
	a.observed = true
	a.queryTotal++
	if stage.Stage2LatencyMs >= 0 {
		a.latencySamples = append(a.latencySamples, stage.Stage2LatencyMs)
	}
	if strings.EqualFold(strings.TrimSpace(stage.Stage2Source), runtimeconfig.ContextStage2ProviderFile) && stage.Stage2HitCount > 0 {
		// Migration compatibility path: memory provider fell back to legacy file source then backfilled.
		a.upsertTotal++
	}
	if memoryReasonCodeIsError(reasonCode, provider == runtimeconfig.ContextStage2ProviderMemory) {
		a.errorTotal++
	}
	if reasonCode == "memory.fallback.used" {
		a.fallbackTotal++
		a.fallbackReasonCode = reasonCode
	} else if strings.HasPrefix(reasonCode, "memory.fallback.") && reasonCode != "" {
		a.fallbackReasonCode = reasonCode
	}
	if scope := strings.TrimSpace(stage.MemoryScopeSelected); scope != "" {
		a.scopeSelected = scope
	}
	if stage.MemoryBudgetUsed > 0 {
		a.budgetUsed += stage.MemoryBudgetUsed
	}
	if stage.MemoryHits > 0 {
		a.hits += stage.MemoryHits
	}
	if len(stage.MemoryRerankStats) > 0 {
		if a.rerankStats == nil {
			a.rerankStats = map[string]int{}
		}
		for key, value := range stage.MemoryRerankStats {
			a.rerankStats[key] += value
		}
	}
	if action := strings.TrimSpace(stage.MemoryLifecycleAction); action != "" {
		a.lifecycleAction = action
	}
}

func (a *memoryRunDiagnosticsAccumulator) snapshot(runtime memoryRuntimeSnapshot) memoryRunDiagnostics {
	if a == nil || !a.observed {
		return memoryRunDiagnostics{}
	}
	reasonCode := strings.TrimSpace(strings.ToLower(a.fallbackReasonCode))
	if reasonCode == "" && a.fallbackTotal > 0 {
		reasonCode = "memory.fallback.used"
	}
	return memoryRunDiagnostics{
		Observed:           true,
		Mode:               runtime.Mode,
		Provider:           runtime.Provider,
		Profile:            runtime.Profile,
		ContractVersion:    runtime.ContractVersion,
		QueryTotal:         a.queryTotal,
		UpsertTotal:        a.upsertTotal,
		DeleteTotal:        a.deleteTotal,
		ErrorTotal:         a.errorTotal,
		FallbackTotal:      a.fallbackTotal,
		FallbackReasonCode: reasonCode,
		LatencyMsP95:       percentileP95Int64(a.latencySamples),
		ScopeSelected:      strings.TrimSpace(a.scopeSelected),
		BudgetUsed:         a.budgetUsed,
		Hits:               a.hits,
		RerankStats:        cloneIntMap(a.rerankStats),
		LifecycleAction:    strings.TrimSpace(a.lifecycleAction),
	}
}

func memoryReasonCodeIsError(reasonCode string, providerMemory bool) bool {
	code := strings.ToLower(strings.TrimSpace(reasonCode))
	switch code {
	case "", "ok", "memory.ok", "memory.not_found", "memory.fallback.used":
		return false
	case "memory_error":
		return true
	}
	if strings.HasPrefix(code, "memory.") {
		return true
	}
	if !providerMemory {
		return false
	}
	switch code {
	case "fetch_error", "timeout", "auth_failed", "mapping_invalid", "request_failed":
		return true
	default:
		return false
	}
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

type sandboxRuntimeSnapshot struct {
	Enabled              bool
	Mode                 string
	Backend              string
	Profile              string
	SessionMode          string
	RequiredCapabilities []string
}

type sandboxRunDiagnostics struct {
	Observed                   bool
	Mode                       string
	Backend                    string
	Profile                    string
	SessionMode                string
	RequiredCapabilities       []string
	Decision                   string
	ReasonCode                 string
	EgressAction               string
	EgressPolicySource         string
	EgressViolationTotal       int
	FallbackUsed               bool
	FallbackReason             string
	TimeoutTotal               int
	LaunchFailedTotal          int
	CapabilityMismatchTotal    int
	QueueWaitMsP95             int64
	ExecLatencyMsP95           int64
	ExitCodeLast               int
	HasExitCode                bool
	OOMTotal                   int
	ResourceCPUMsTotal         int64
	ResourceMemoryPeakBytesP95 int64
}

type sandboxRunDiagnosticsAccumulator struct {
	observed                bool
	mode                    string
	backend                 string
	profile                 string
	sessionMode             string
	requiredCapabilities    []string
	decision                string
	reasonCode              string
	egressAction            string
	egressPolicySource      string
	egressViolationTotal    int
	fallbackUsed            bool
	fallbackReason          string
	timeoutTotal            int
	launchFailedTotal       int
	capabilityMismatchTotal int
	queueWaitSamples        []int64
	execLatencySamples      []int64
	exitCodeLast            int
	hasExitCode             bool
	oomTotal                int
	resourceCPUMsTotal      int64
	memoryPeakSamples       []int64
}

func (e *Engine) sandboxRuntimeSnapshot() sandboxRuntimeSnapshot {
	if e == nil || e.runtimeMgr == nil {
		return sandboxRuntimeSnapshot{}
	}
	cfg := e.runtimeMgr.EffectiveConfig().Security.Sandbox
	return sandboxRuntimeSnapshot{
		Enabled:              cfg.Enabled,
		Mode:                 strings.ToLower(strings.TrimSpace(cfg.Mode)),
		Backend:              strings.ToLower(strings.TrimSpace(cfg.Executor.Backend)),
		Profile:              strings.ToLower(strings.TrimSpace(cfg.Policy.Profile)),
		SessionMode:          strings.ToLower(strings.TrimSpace(cfg.Executor.SessionMode)),
		RequiredCapabilities: cloneNormalizedStringSlice(cfg.Executor.RequiredCapabilities),
	}
}

func (e *Engine) runtimeMemorySnapshot() memoryRuntimeSnapshot {
	cfg := runtimeconfig.DefaultConfig().Runtime.Memory
	if e != nil && e.runtimeMgr != nil {
		cfg = e.runtimeMgr.EffectiveConfig().Runtime.Memory
	}
	mode := strings.ToLower(strings.TrimSpace(cfg.Mode))
	if mode == "" {
		mode = runtimeconfig.RuntimeMemoryModeBuiltinFilesystem
	}
	provider := runtimeconfig.RuntimeMemoryProviderGeneric
	if mode == runtimeconfig.RuntimeMemoryModeBuiltinFilesystem {
		provider = runtimeconfig.RuntimeMemoryModeBuiltinFilesystem
	} else if candidate := strings.ToLower(strings.TrimSpace(cfg.External.Provider)); candidate != "" {
		provider = candidate
	}
	profile := strings.ToLower(strings.TrimSpace(cfg.External.Profile))
	if profile == "" {
		profile = runtimeconfig.RuntimeMemoryProfileGeneric
	}
	contractVersion := strings.ToLower(strings.TrimSpace(cfg.External.ContractVersion))
	if contractVersion == "" {
		contractVersion = runtimeconfig.RuntimeMemoryContractVersionV1
	}
	return memoryRuntimeSnapshot{
		Mode:            mode,
		Provider:        provider,
		Profile:         profile,
		ContractVersion: contractVersion,
	}
}

func (e *Engine) runtimeReactConfigSnapshot() runtimeconfig.RuntimeReactConfig {
	cfg := runtimeconfig.DefaultConfig().Runtime.React
	if e != nil && e.runtimeMgr != nil {
		cfg = e.runtimeMgr.EffectiveConfig().Runtime.React
	}
	return cfg
}

func (e *Engine) runtimeHooksConfigSnapshot() runtimeconfig.RuntimeHooksConfig {
	cfg := runtimeconfig.DefaultConfig().Runtime.Hooks
	if e != nil && e.runtimeMgr != nil {
		cfg = e.runtimeMgr.EffectiveConfig().Runtime.Hooks
	}
	return cfg
}

func (e *Engine) runtimeToolMiddlewareConfigSnapshot() runtimeconfig.RuntimeToolMiddlewareConfig {
	cfg := runtimeconfig.DefaultConfig().Runtime.ToolMiddleware
	if e != nil && e.runtimeMgr != nil {
		cfg = e.runtimeMgr.EffectiveConfig().Runtime.ToolMiddleware
	}
	return cfg
}

func (e *Engine) runtimeSkillPreprocessConfigSnapshot() runtimeconfig.RuntimeSkillPreprocessConfig {
	cfg := runtimeconfig.DefaultConfig().Runtime.Skill.Preprocess
	if e != nil && e.runtimeMgr != nil {
		cfg = e.runtimeMgr.EffectiveConfig().Runtime.Skill.Preprocess
	}
	return cfg
}

func (e *Engine) runtimeSkillDiscoveryConfigSnapshot() runtimeconfig.RuntimeSkillDiscoveryConfig {
	cfg := runtimeconfig.DefaultConfig().Runtime.Skill.Discovery
	if e != nil && e.runtimeMgr != nil {
		cfg = e.runtimeMgr.EffectiveConfig().Runtime.Skill.Discovery
	}
	return cfg
}

func (e *Engine) runtimeSkillBundleMappingConfigSnapshot() runtimeconfig.RuntimeSkillBundleMappingConfig {
	cfg := runtimeconfig.DefaultConfig().Runtime.Skill.BundleMapping
	if e != nil && e.runtimeMgr != nil {
		cfg = e.runtimeMgr.EffectiveConfig().Runtime.Skill.BundleMapping
	}
	return cfg
}

type skillPreprocessState struct {
	DiscoveryMode          string
	DiscoveryRoots         []string
	PreprocessEnabled      bool
	PreprocessPhase        string
	PreprocessFailMode     string
	PreprocessStatus       string
	PreprocessReasonCode   string
	PromptMode             string
	WhitelistMode          string
	ConflictPolicy         string
	PromptFragmentCount    int
	WhitelistCount         int
	WhitelistRejectedTotal int
	SpecCount              int
	AllowedToolSet         map[string]struct{}
}

func (s skillPreprocessState) whitelistEnforced() bool {
	return strings.EqualFold(strings.TrimSpace(s.WhitelistMode), runtimeconfig.RuntimeSkillBundleMappingWhitelistModeMerge) &&
		len(s.AllowedToolSet) > 0
}

type skillPreprocessError struct {
	reasonCode string
	cause      error
}

func (e *skillPreprocessError) Error() string {
	if e == nil {
		return ""
	}
	if e.cause != nil {
		return e.cause.Error()
	}
	return "skill preprocess error"
}

func (e *skillPreprocessError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.cause
}

func newSkillPreprocessError(reasonCode string, cause error) error {
	return &skillPreprocessError{
		reasonCode: strings.ToLower(strings.TrimSpace(reasonCode)),
		cause:      cause,
	}
}

func (e *Engine) runSkillPreprocess(
	ctx context.Context,
	req types.RunRequest,
	runID string,
	stream bool,
	h types.EventHandler,
	warnings *[]string,
) (types.RunRequest, skillPreprocessState, *types.ClassifiedError, error) {
	discoveryCfg := e.runtimeSkillDiscoveryConfigSnapshot()
	preprocessCfg := e.runtimeSkillPreprocessConfigSnapshot()
	mappingCfg := e.runtimeSkillBundleMappingConfigSnapshot()
	state := skillPreprocessState{
		DiscoveryMode:        strings.ToLower(strings.TrimSpace(discoveryCfg.Mode)),
		DiscoveryRoots:       append([]string(nil), discoveryCfg.Roots...),
		PreprocessEnabled:    preprocessCfg.Enabled,
		PreprocessPhase:      strings.ToLower(strings.TrimSpace(preprocessCfg.Phase)),
		PreprocessFailMode:   strings.ToLower(strings.TrimSpace(preprocessCfg.FailMode)),
		PreprocessStatus:     "skipped",
		PromptMode:           strings.ToLower(strings.TrimSpace(mappingCfg.PromptMode)),
		WhitelistMode:        strings.ToLower(strings.TrimSpace(mappingCfg.WhitelistMode)),
		ConflictPolicy:       strings.ToLower(strings.TrimSpace(mappingCfg.ConflictPolicy)),
		AllowedToolSet:       map[string]struct{}{},
		PromptFragmentCount:  0,
		WhitelistCount:       0,
		SpecCount:            0,
		PreprocessReasonCode: "",
	}
	if e == nil || e.skillLoader == nil {
		return req, state, nil, nil
	}
	if e.runtimeMgr != nil && !preprocessCfg.Enabled {
		return req, state, nil, nil
	}
	if !strings.EqualFold(strings.TrimSpace(preprocessCfg.Phase), runtimeconfig.RuntimeSkillPreprocessPhaseBeforeRunStream) {
		return req, state, nil, nil
	}
	skillCtx := map[string]string{
		"run_id":      runID,
		"session_id":  req.SessionID,
		"entry_point": "run",
	}
	if stream {
		skillCtx["entry_point"] = "stream"
	}
	specs, err := e.skillLoader.Discover(ctx, ".")
	if err != nil {
		return e.resolveSkillPreprocessFailure(req, state, preprocessCfg, err, warnings)
	}
	state.SpecCount = len(specs)
	bundle, err := e.skillLoader.Compile(ctx, specs, types.SkillInput{
		UserInput: req.Input,
		Context:   skillCtx,
	})
	if err != nil {
		return e.resolveSkillPreprocessFailure(req, state, preprocessCfg, err, warnings)
	}
	req, state, err = e.applySkillBundleMappings(req, state, bundle)
	if err != nil {
		return e.resolveSkillPreprocessFailure(req, state, preprocessCfg, err, warnings)
	}
	state.PreprocessStatus = "success"
	e.emit(ctx, h, types.Event{
		Version: types.EventSchemaVersionV1,
		Type:    "skill.preprocess.completed",
		RunID:   runID,
		Time:    e.now(),
		Payload: map[string]any{
			"spec_count":                state.SpecCount,
			"stream":                    stream,
			"phase":                     runtimeconfig.RuntimeSkillPreprocessPhaseBeforeRunStream,
			"preprocess_status":         state.PreprocessStatus,
			"preprocess_fail_mode":      state.PreprocessFailMode,
			"discovery_mode":            state.DiscoveryMode,
			"bundle_prompt_mode":        state.PromptMode,
			"bundle_whitelist_mode":     state.WhitelistMode,
			"bundle_conflict_policy":    state.ConflictPolicy,
			"bundle_prompt_total":       state.PromptFragmentCount,
			"bundle_whitelist_total":    state.WhitelistCount,
			"bundle_whitelist_rejected": state.WhitelistRejectedTotal,
		},
	})
	return req, state, nil, nil
}

func (e *Engine) applySkillBundleMappings(
	req types.RunRequest,
	state skillPreprocessState,
	bundle types.SkillBundle,
) (types.RunRequest, skillPreprocessState, error) {
	mappedReq := req
	normalizedConflict := strings.ToLower(strings.TrimSpace(state.ConflictPolicy))
	if normalizedConflict == "" {
		normalizedConflict = runtimeconfig.RuntimeSkillBundleMappingConflictPolicyFirstWin
	}

	if strings.EqualFold(strings.TrimSpace(state.PromptMode), runtimeconfig.RuntimeSkillBundleMappingPromptModeAppend) {
		seenPrompt := map[string]struct{}{}
		for _, msg := range mappedReq.Messages {
			if !strings.EqualFold(strings.TrimSpace(msg.Role), "system") {
				continue
			}
			key := strings.ToLower(strings.TrimSpace(msg.Content))
			if key == "" {
				continue
			}
			seenPrompt[key] = struct{}{}
		}
		for _, fragment := range bundle.SystemPromptFragments {
			content := strings.TrimSpace(fragment)
			if content == "" {
				continue
			}
			key := strings.ToLower(content)
			if _, exists := seenPrompt[key]; exists {
				if normalizedConflict == runtimeconfig.RuntimeSkillBundleMappingConflictPolicyFailFast {
					return req, state, newSkillPreprocessError(
						"skill_bundle_prompt_conflict",
						fmt.Errorf("prompt fragment conflict detected for %q", content),
					)
				}
				continue
			}
			seenPrompt[key] = struct{}{}
			mappedReq.Messages = append(mappedReq.Messages, types.Message{
				Role:    "system",
				Content: content,
			})
			state.PromptFragmentCount++
		}
	}

	if strings.EqualFold(strings.TrimSpace(state.WhitelistMode), runtimeconfig.RuntimeSkillBundleMappingWhitelistModeMerge) {
		seenTool := map[string]struct{}{}
		for _, raw := range bundle.EnabledTools {
			name := normalizeToolName(raw)
			if name == "" {
				continue
			}
			if _, exists := seenTool[name]; exists {
				if normalizedConflict == runtimeconfig.RuntimeSkillBundleMappingConflictPolicyFailFast {
					return req, state, newSkillPreprocessError(
						"skill_bundle_whitelist_conflict",
						fmt.Errorf("tool whitelist conflict detected for %q", name),
					)
				}
				continue
			}
			seenTool[name] = struct{}{}
			allowed, reasonCode := e.skillWhitelistWithinSecurityUpperBound(name)
			if !allowed {
				state.WhitelistRejectedTotal++
				if normalizedConflict == runtimeconfig.RuntimeSkillBundleMappingConflictPolicyFailFast {
					return req, state, newSkillPreprocessError(
						reasonCode,
						fmt.Errorf("tool whitelist exceeds upper-bound for %q", name),
					)
				}
				continue
			}
			state.AllowedToolSet[name] = struct{}{}
			state.WhitelistCount++
		}
	}
	return mappedReq, state, nil
}

func (e *Engine) skillWhitelistWithinSecurityUpperBound(name string) (bool, string) {
	namespaceTool, ok := namespaceToolKey(name)
	if !ok {
		return false, "skill_bundle_whitelist_invalid_tool"
	}
	sandboxCfg := e.securitySandboxConfig()
	if sandboxCfg.Enabled && strings.EqualFold(strings.TrimSpace(sandboxCfg.Mode), runtimeconfig.SecuritySandboxModeEnforce) {
		if runtimeconfig.ResolveSandboxAction(sandboxCfg, namespaceTool) == runtimeconfig.SecuritySandboxActionDeny {
			return false, "skill_bundle_whitelist_exceeds_sandbox"
		}
	}
	if e == nil || e.runtimeMgr == nil {
		return true, ""
	}
	allowlistCfg := e.runtimeMgr.EffectiveConfig().Adapter.Allowlist
	if !allowlistCfg.Enabled || !strings.EqualFold(strings.TrimSpace(allowlistCfg.EnforcementMode), runtimeconfig.AdapterAllowlistEnforcementModeEnforce) {
		return true, ""
	}
	namespace := strings.TrimSpace(strings.SplitN(namespaceTool, "+", 2)[0])
	if namespace == "" || namespace == "local" {
		return true, ""
	}
	denyUnknown := strings.EqualFold(strings.TrimSpace(allowlistCfg.OnUnknownSignature), runtimeconfig.AdapterAllowlistUnknownSignatureDeny)
	for i := range allowlistCfg.Entries {
		entry := allowlistCfg.Entries[i]
		if !strings.EqualFold(strings.TrimSpace(entry.AdapterID), namespace) {
			continue
		}
		status := strings.ToLower(strings.TrimSpace(entry.SignatureStatus))
		switch status {
		case runtimeconfig.AdapterAllowlistSignatureStatusValid:
			return true, ""
		case runtimeconfig.AdapterAllowlistSignatureStatusUnknown:
			if !denyUnknown {
				return true, ""
			}
		}
	}
	return false, "skill_bundle_whitelist_exceeds_adapter_allowlist"
}

func (e *Engine) enforceSkillWhitelistForToolCalls(
	ctx context.Context,
	h types.EventHandler,
	runID string,
	iteration int,
	calls []types.ToolCall,
	state skillPreprocessState,
	warnings *[]string,
) ([]types.ToolCall, *types.ClassifiedError, error) {
	if len(calls) == 0 || !state.whitelistEnforced() {
		return calls, nil, nil
	}
	allowedSet := state.AllowedToolSet
	filtered := make([]types.ToolCall, 0, len(calls))
	blocked := make([]string, 0, len(calls))
	for i := range calls {
		call := calls[i]
		name := normalizeToolName(call.Name)
		if _, ok := allowedSet[name]; ok {
			filtered = append(filtered, call)
			continue
		}
		blocked = append(blocked, name)
	}
	if len(blocked) == 0 {
		return calls, nil, nil
	}
	e.emit(ctx, h, types.Event{
		Version:   types.EventSchemaVersionV1,
		Type:      "skill.bundle_mapping.whitelist.filtered",
		RunID:     runID,
		Iteration: iteration,
		Time:      e.now(),
		Payload: map[string]any{
			"blocked_tools":    append([]string(nil), blocked...),
			"whitelist_total":  state.WhitelistCount,
			"conflict_policy":  state.ConflictPolicy,
			"preprocess_phase": state.PreprocessPhase,
		},
	})
	if strings.EqualFold(strings.TrimSpace(state.ConflictPolicy), runtimeconfig.RuntimeSkillBundleMappingConflictPolicyFailFast) {
		terminal := classified(types.ErrSecurity, fmt.Sprintf("tool call rejected by skill whitelist: %s", blocked[0]), false)
		terminal.Details = map[string]any{
			"reason_code":      "skill_bundle_whitelist_violation",
			"blocked_tools":    append([]string(nil), blocked...),
			"whitelist_total":  state.WhitelistCount,
			"preprocess_phase": state.PreprocessPhase,
		}
		return nil, terminal, errors.New(terminal.Message)
	}
	if warnings != nil {
		*warnings = append(*warnings, fmt.Sprintf("skill whitelist filtered tools: %s", strings.Join(blocked, ",")))
	}
	return filtered, nil, nil
}

func (e *Engine) resolveSkillPreprocessFailure(
	req types.RunRequest,
	state skillPreprocessState,
	cfg runtimeconfig.RuntimeSkillPreprocessConfig,
	err error,
	warnings *[]string,
) (types.RunRequest, skillPreprocessState, *types.ClassifiedError, error) {
	reasonCode := "skill_preprocess_failed"
	var preprocessErr *skillPreprocessError
	if errors.As(err, &preprocessErr) {
		if code := strings.ToLower(strings.TrimSpace(preprocessErr.reasonCode)); code != "" {
			reasonCode = code
		}
	}
	if strings.EqualFold(strings.TrimSpace(cfg.FailMode), runtimeconfig.RuntimeSkillPreprocessFailModeDegrade) {
		state.PreprocessStatus = "degraded"
		state.PreprocessReasonCode = reasonCode
		if warnings != nil {
			*warnings = append(*warnings, fmt.Sprintf("skill preprocess degraded: %v", err))
		}
		return req, state, nil, nil
	}
	state.PreprocessStatus = "failed"
	state.PreprocessReasonCode = reasonCode
	terminal := classified(types.ErrSkill, fmt.Sprintf("skill preprocess failed: %v", err), false)
	terminal.Details = map[string]any{
		"phase":       runtimeconfig.RuntimeSkillPreprocessPhaseBeforeRunStream,
		"reason_code": reasonCode,
	}
	return req, state, terminal, err
}

func (e *Engine) runLifecycleHooks(
	ctx context.Context,
	req types.RunRequest,
	runID string,
	iteration int,
	stream bool,
	phase types.AgentLifecyclePhase,
	finalAnswer string,
	toolCalls []types.ToolCall,
	h types.EventHandler,
	warnings *[]string,
) (*types.ClassifiedError, error) {
	if e == nil || len(e.lifecycleHooks) == 0 {
		return nil, nil
	}
	cfg := e.runtimeHooksConfigSnapshot()
	if e.runtimeMgr != nil && !cfg.Enabled {
		return nil, nil
	}
	if !lifecyclePhaseEnabled(cfg.Phases, phase) {
		return nil, nil
	}
	for i := range e.lifecycleHooks {
		hook := e.lifecycleHooks[i]
		if hook == nil {
			continue
		}
		hookCtx := ctx
		cancel := func() {}
		if cfg.Timeout > 0 {
			hookCtx, cancel = context.WithTimeout(ctx, cfg.Timeout)
		}
		err := hook.OnPhase(hookCtx, types.AgentLifecycleHookContext{
			RunID:       runID,
			SessionID:   req.SessionID,
			Iteration:   iteration,
			Stream:      stream,
			Phase:       phase,
			FinalAnswer: finalAnswer,
			ToolCalls:   append([]types.ToolCall(nil), toolCalls...),
		})
		cancel()
		if err == nil {
			continue
		}
		reasonCode := "hook_error"
		errClass := types.ErrContext
		retryable := false
		switch {
		case errors.Is(err, context.DeadlineExceeded):
			reasonCode = "hook_timeout"
			errClass = types.ErrPolicyTimeout
			retryable = true
		case errors.Is(err, context.Canceled):
			reasonCode = "hook_canceled"
			errClass = types.ErrPolicyTimeout
			retryable = true
		}
		e.emit(ctx, h, types.Event{
			Version:   types.EventSchemaVersionV1,
			Type:      "hook.failed",
			RunID:     runID,
			Iteration: iteration,
			Time:      e.now(),
			Payload: map[string]any{
				"phase":       string(phase),
				"reason_code": reasonCode,
				"hook_index":  i,
				"stream":      stream,
			},
		})
		if strings.EqualFold(strings.TrimSpace(cfg.FailMode), runtimeconfig.RuntimeHooksFailModeDegrade) {
			if warnings != nil {
				*warnings = append(*warnings, fmt.Sprintf("lifecycle hook degraded at %s: %v", phase, err))
			}
			continue
		}
		terminal := classified(errClass, fmt.Sprintf("lifecycle hook failed at %s: %v", phase, err), retryable)
		terminal.Details = map[string]any{
			"phase":       string(phase),
			"reason_code": reasonCode,
			"hook_index":  i,
		}
		return terminal, err
	}
	return nil, nil
}

func lifecyclePhaseEnabled(phases []string, phase types.AgentLifecyclePhase) bool {
	target := strings.ToLower(strings.TrimSpace(string(phase)))
	if target == "" {
		return false
	}
	normalized := runtimeconfig.DefaultConfig().Runtime.Hooks.Phases
	if len(phases) > 0 {
		normalized = phases
	}
	for i := range normalized {
		if strings.EqualFold(strings.TrimSpace(normalized[i]), target) {
			return true
		}
	}
	return false
}

func resolveReactTerminationReason(
	terminal *types.ClassifiedError,
	runErr error,
	toolBudgetHitTotal int,
	iterationBudgetHitTotal int,
) string {
	if toolBudgetHitTotal > 0 {
		return runtimeconfig.RuntimeReactTerminationToolCallLimitExceeded
	}
	if iterationBudgetHitTotal > 0 {
		return runtimeconfig.RuntimeReactTerminationMaxIterationsExceeded
	}
	if errors.Is(runErr, context.Canceled) {
		return runtimeconfig.RuntimeReactTerminationContextCanceled
	}
	if terminal == nil {
		return runtimeconfig.RuntimeReactTerminationProviderError
	}
	switch terminal.Class {
	case types.ErrIterationLimit:
		return runtimeconfig.RuntimeReactTerminationMaxIterationsExceeded
	case types.ErrTool, types.ErrSecurity:
		return runtimeconfig.RuntimeReactTerminationToolDispatchFailed
	case types.ErrPolicyTimeout:
		return runtimeconfig.RuntimeReactTerminationContextCanceled
	case types.ErrContext:
		if errors.Is(runErr, context.Canceled) {
			return runtimeconfig.RuntimeReactTerminationContextCanceled
		}
		return runtimeconfig.RuntimeReactTerminationProviderError
	case types.ErrModel, types.ErrHITL:
		return runtimeconfig.RuntimeReactTerminationProviderError
	default:
		return runtimeconfig.RuntimeReactTerminationProviderError
	}
}

func mergeStreamToolCall(calls []types.ToolCall, call types.ToolCall) []types.ToolCall {
	out := append([]types.ToolCall(nil), calls...)
	call.CallID = strings.TrimSpace(call.CallID)
	call.Name = strings.TrimSpace(call.Name)
	if call.CallID != "" {
		for i := range out {
			if strings.TrimSpace(out[i].CallID) != call.CallID {
				continue
			}
			out[i] = call
			return out
		}
	}
	out = append(out, call)
	return out
}

func (a *sandboxRunDiagnosticsAccumulator) observeOutcomes(outcomes []types.ToolCallOutcome) {
	if a == nil || len(outcomes) == 0 {
		return
	}
	for i := range outcomes {
		outcome := outcomes[i]
		if outcome.Result.Error != nil {
			a.observeDetailFields(outcome.Result.Error.Details)
		}
		a.observeStructuredFields(outcome.Result.Structured)
	}
}

func (a *sandboxRunDiagnosticsAccumulator) observeDetailFields(details map[string]any) {
	if a == nil || len(details) == 0 {
		return
	}
	if mode := normalizeSandboxMode(sandboxMapString(details, "sandbox_mode")); mode != "" {
		a.mode = mode
		a.observed = true
	}
	if backend := strings.ToLower(strings.TrimSpace(sandboxMapString(details, "sandbox_backend"))); backend != "" {
		a.backend = backend
		a.observed = true
	}
	if profile := strings.ToLower(strings.TrimSpace(sandboxMapString(details, "sandbox_profile"))); profile != "" {
		a.profile = profile
		a.observed = true
	}
	if session := normalizeSandboxSessionMode(sandboxMapString(details, "sandbox_session_mode")); session != "" {
		a.sessionMode = session
		a.observed = true
	}
	required := sandboxMapStringSlice(details, "sandbox_required_capabilities")
	if len(required) == 0 {
		required = sandboxMapStringSlice(details, "required_capabilities")
	}
	if len(required) > 0 {
		a.requiredCapabilities = required
		a.observed = true
	}
	if decision := normalizeSandboxDecision(sandboxMapString(details, "sandbox_action")); decision != "" {
		a.decision = decision
		a.observed = true
	}
	if fallbackAction := strings.ToLower(strings.TrimSpace(sandboxMapString(details, "sandbox_fallback"))); fallbackAction == runtimeconfig.SecuritySandboxFallbackAllowAndRecord {
		a.fallbackUsed = true
		if a.decision == "" {
			a.decision = runtimeconfig.SecuritySandboxActionHost
		}
		a.observed = true
	}
	a.observeEgressFields(details)
	a.observeReason(sandboxMapString(details, "reason_code"))
}

func (a *sandboxRunDiagnosticsAccumulator) observeStructuredFields(structured map[string]any) {
	if a == nil || len(structured) == 0 {
		return
	}
	if mode := normalizeSandboxMode(sandboxMapString(structured, "sandbox_mode")); mode != "" {
		a.mode = mode
		a.observed = true
	}
	if backend := strings.ToLower(strings.TrimSpace(sandboxMapString(structured, "sandbox_backend"))); backend != "" {
		a.backend = backend
		a.observed = true
	}
	if profile := strings.ToLower(strings.TrimSpace(sandboxMapString(structured, "sandbox_profile"))); profile != "" {
		a.profile = profile
		a.observed = true
	}
	if session := normalizeSandboxSessionMode(sandboxMapString(structured, "sandbox_session_mode")); session != "" {
		a.sessionMode = session
		a.observed = true
	}
	if required := sandboxMapStringSlice(structured, "sandbox_required_capabilities"); len(required) > 0 {
		a.requiredCapabilities = required
		a.observed = true
	}
	if decision := normalizeSandboxDecision(sandboxMapString(structured, "sandbox_decision")); decision != "" {
		a.decision = decision
		a.observed = true
	}
	if sandboxMapBool(structured, "sandbox_fallback") || sandboxMapBool(structured, "sandbox_fallback_used") {
		a.fallbackUsed = true
		if a.decision == "" {
			a.decision = runtimeconfig.SecuritySandboxActionHost
		}
		a.observed = true
	}
	if reason := strings.ToLower(strings.TrimSpace(sandboxMapString(structured, "sandbox_fallback_reason"))); reason != "" {
		a.fallbackReason = reason
		a.observeReason(reason)
	}
	a.observeEgressFields(structured)
	a.observeReason(sandboxMapString(structured, "sandbox_reason_code"))

	if queueWait, ok := sandboxMapInt64(structured, "sandbox_queue_wait_ms"); ok && queueWait >= 0 {
		a.queueWaitSamples = append(a.queueWaitSamples, queueWait)
		a.observed = true
	}
	if execLatency, ok := sandboxMapInt64(structured, "sandbox_exec_latency_ms"); ok && execLatency >= 0 {
		a.execLatencySamples = append(a.execLatencySamples, execLatency)
		a.observed = true
	}
	if exitCode, ok := sandboxMapInt(structured, "sandbox_exit_code"); ok {
		a.exitCodeLast = exitCode
		a.hasExitCode = true
		a.observed = true
	}
	if sandboxMapBool(structured, "sandbox_oom") {
		a.oomTotal++
		a.observed = true
	}
	if cpuMs, ok := sandboxMapInt64(structured, "sandbox_resource_cpu_ms"); ok && cpuMs >= 0 {
		a.resourceCPUMsTotal += cpuMs
		a.observed = true
	}
	if memoryPeak, ok := sandboxMapInt64(structured, "sandbox_resource_memory_peak_bytes"); ok && memoryPeak >= 0 {
		a.memoryPeakSamples = append(a.memoryPeakSamples, memoryPeak)
		a.observed = true
	}
}

func (a *sandboxRunDiagnosticsAccumulator) observeEgressFields(payload map[string]any) {
	if a == nil || len(payload) == 0 {
		return
	}
	if action := normalizeSandboxEgressAction(sandboxMapString(payload, "sandbox_egress_action")); action != "" {
		a.egressAction = action
		a.observed = true
	}
	if source := normalizeSandboxEgressPolicySource(sandboxMapString(payload, "sandbox_egress_policy_source")); source != "" {
		a.egressPolicySource = source
		a.observed = true
	}
	if total, ok := sandboxMapInt(payload, "sandbox_egress_violation_total"); ok && total > 0 {
		a.egressViolationTotal += total
		a.observed = true
	}
}

func (a *sandboxRunDiagnosticsAccumulator) observeReason(reason string) {
	if a == nil {
		return
	}
	normalized := strings.ToLower(strings.TrimSpace(reason))
	if !strings.HasPrefix(normalized, "sandbox.") {
		return
	}
	a.reasonCode = normalized
	a.observed = true
	switch normalized {
	case types.SandboxViolationTimeout:
		a.timeoutTotal++
	case "sandbox.launch_failed", types.SandboxViolationLaunchFailed:
		a.launchFailedTotal++
	case types.SandboxViolationCapabilityMismatch:
		a.capabilityMismatchTotal++
	case types.SandboxViolationOOM:
		a.oomTotal++
	case "sandbox.fallback_allow_and_record":
		a.fallbackUsed = true
		if a.fallbackReason == "" {
			a.fallbackReason = normalized
		}
		if a.decision == "" {
			a.decision = runtimeconfig.SecuritySandboxActionHost
		}
	case "sandbox.egress_deny":
		a.egressViolationTotal++
		if a.egressAction == "" {
			a.egressAction = runtimeconfig.SecuritySandboxEgressActionDeny
		}
	case "sandbox.egress_allow_and_record":
		a.egressViolationTotal++
		if a.egressAction == "" {
			a.egressAction = runtimeconfig.SecuritySandboxEgressActionAllowAndRecord
		}
	}
	if a.decision == "" {
		a.decision = runtimeconfig.SecuritySandboxActionDeny
	}
}

func (a sandboxRunDiagnosticsAccumulator) snapshot(runtime sandboxRuntimeSnapshot) sandboxRunDiagnostics {
	out := sandboxRunDiagnostics{
		Observed:                   a.observed,
		Mode:                       a.mode,
		Backend:                    a.backend,
		Profile:                    a.profile,
		SessionMode:                a.sessionMode,
		RequiredCapabilities:       cloneNormalizedStringSlice(a.requiredCapabilities),
		Decision:                   a.decision,
		ReasonCode:                 a.reasonCode,
		EgressAction:               a.egressAction,
		EgressPolicySource:         a.egressPolicySource,
		EgressViolationTotal:       a.egressViolationTotal,
		FallbackUsed:               a.fallbackUsed,
		FallbackReason:             a.fallbackReason,
		TimeoutTotal:               a.timeoutTotal,
		LaunchFailedTotal:          a.launchFailedTotal,
		CapabilityMismatchTotal:    a.capabilityMismatchTotal,
		QueueWaitMsP95:             percentileP95Int64(a.queueWaitSamples),
		ExecLatencyMsP95:           percentileP95Int64(a.execLatencySamples),
		ExitCodeLast:               a.exitCodeLast,
		HasExitCode:                a.hasExitCode,
		OOMTotal:                   a.oomTotal,
		ResourceCPUMsTotal:         a.resourceCPUMsTotal,
		ResourceMemoryPeakBytesP95: percentileP95Int64(a.memoryPeakSamples),
	}
	if !out.Observed {
		return out
	}
	if out.Mode == "" {
		out.Mode = runtime.Mode
	}
	if out.Backend == "" {
		out.Backend = runtime.Backend
	}
	if out.Profile == "" {
		out.Profile = runtime.Profile
	}
	if out.SessionMode == "" {
		out.SessionMode = runtime.SessionMode
	}
	if len(out.RequiredCapabilities) == 0 {
		out.RequiredCapabilities = cloneNormalizedStringSlice(runtime.RequiredCapabilities)
	}
	if out.Decision == "" && out.FallbackUsed {
		out.Decision = runtimeconfig.SecuritySandboxActionHost
	}
	if out.Decision == "" && out.ReasonCode != "" {
		out.Decision = runtimeconfig.SecuritySandboxActionDeny
	}
	if out.FallbackReason == "" && out.FallbackUsed {
		out.FallbackReason = "sandbox.fallback_allow_and_record"
	}
	if out.ReasonCode == "" && out.FallbackReason != "" {
		out.ReasonCode = out.FallbackReason
	}
	return out
}

func normalizeSandboxEgressAction(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case runtimeconfig.SecuritySandboxEgressActionDeny:
		return runtimeconfig.SecuritySandboxEgressActionDeny
	case runtimeconfig.SecuritySandboxEgressActionAllow:
		return runtimeconfig.SecuritySandboxEgressActionAllow
	case runtimeconfig.SecuritySandboxEgressActionAllowAndRecord:
		return runtimeconfig.SecuritySandboxEgressActionAllowAndRecord
	default:
		return ""
	}
}

func normalizeSandboxEgressPolicySource(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "default_action":
		return "default_action"
	case "by_tool":
		return "by_tool"
	case "allowlist":
		return "allowlist"
	case "on_violation":
		return "on_violation"
	default:
		return ""
	}
}

func normalizeSandboxMode(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case runtimeconfig.SecuritySandboxModeObserve:
		return runtimeconfig.SecuritySandboxModeObserve
	case runtimeconfig.SecuritySandboxModeEnforce:
		return runtimeconfig.SecuritySandboxModeEnforce
	default:
		return ""
	}
}

func normalizeSandboxSessionMode(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case runtimeconfig.SecuritySandboxSessionModePerCall:
		return runtimeconfig.SecuritySandboxSessionModePerCall
	case runtimeconfig.SecuritySandboxSessionModePerSession:
		return runtimeconfig.SecuritySandboxSessionModePerSession
	default:
		return ""
	}
}

func normalizeSandboxDecision(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case runtimeconfig.SecuritySandboxActionHost:
		return runtimeconfig.SecuritySandboxActionHost
	case runtimeconfig.SecuritySandboxActionSandbox:
		return runtimeconfig.SecuritySandboxActionSandbox
	case runtimeconfig.SecuritySandboxActionDeny:
		return runtimeconfig.SecuritySandboxActionDeny
	default:
		return ""
	}
}

func cloneNormalizedStringSlice(in []string) []string {
	if len(in) == 0 {
		return nil
	}
	out := make([]string, 0, len(in))
	for i := range in {
		item := strings.ToLower(strings.TrimSpace(in[i]))
		if item == "" {
			continue
		}
		out = append(out, item)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func percentileP95Int64(samples []int64) int64 {
	if len(samples) == 0 {
		return 0
	}
	if len(samples) == 1 {
		return samples[0]
	}
	cp := make([]int64, len(samples))
	copy(cp, samples)
	sort.Slice(cp, func(i, j int) bool { return cp[i] < cp[j] })
	index := int(math.Ceil(0.95*float64(len(cp)))) - 1
	if index < 0 {
		index = 0
	}
	if index >= len(cp) {
		index = len(cp) - 1
	}
	return cp[index]
}

func sandboxMapString(values map[string]any, key string) string {
	if len(values) == 0 {
		return ""
	}
	raw, ok := values[key]
	if !ok {
		return ""
	}
	typed, _ := raw.(string)
	return strings.TrimSpace(typed)
}

func sandboxMapStringSlice(values map[string]any, key string) []string {
	if len(values) == 0 {
		return nil
	}
	raw, ok := values[key]
	if !ok {
		return nil
	}
	switch typed := raw.(type) {
	case []string:
		return cloneNormalizedStringSlice(typed)
	case []any:
		items := make([]string, 0, len(typed))
		for i := range typed {
			item, ok := typed[i].(string)
			if !ok {
				continue
			}
			items = append(items, item)
		}
		return cloneNormalizedStringSlice(items)
	case string:
		parts := strings.Split(typed, ",")
		return cloneNormalizedStringSlice(parts)
	default:
		return nil
	}
}

func sandboxMapBool(values map[string]any, key string) bool {
	if len(values) == 0 {
		return false
	}
	raw, ok := values[key]
	if !ok {
		return false
	}
	typed, _ := raw.(bool)
	return typed
}

func sandboxMapInt64(values map[string]any, key string) (int64, bool) {
	if len(values) == 0 {
		return 0, false
	}
	raw, ok := values[key]
	if !ok {
		return 0, false
	}
	switch typed := raw.(type) {
	case int64:
		return typed, true
	case int:
		return int64(typed), true
	case float64:
		return int64(typed), true
	default:
		return 0, false
	}
}

func sandboxMapInt(values map[string]any, key string) (int, bool) {
	v, ok := sandboxMapInt64(values, key)
	return int(v), ok
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

func primaryToolOutcomeError(outcomes []types.ToolCallOutcome) *types.ClassifiedError {
	var first *types.ClassifiedError
	for i := range outcomes {
		candidate := outcomes[i].Result.Error
		if candidate == nil {
			continue
		}
		if first == nil {
			first = candidate
		}
		if candidate.Class == types.ErrSecurity {
			return candidate
		}
	}
	return first
}

func toolDispatchTimelineReason(err *types.ClassifiedError) string {
	if err == nil {
		return "dispatch_error"
	}
	if err.Details != nil {
		if reason, ok := err.Details["reason_code"].(string); ok {
			reason = strings.ToLower(strings.TrimSpace(reason))
			if strings.HasPrefix(reason, "sandbox.") {
				return reason
			}
		}
	}
	if err.Class == types.ErrSecurity {
		return "security_error"
	}
	return "dispatch_error"
}

func toolDispatchObservedSandboxReason(outcomes []types.ToolCallOutcome) string {
	for i := range outcomes {
		outcome := outcomes[i]
		if outcome.Result.Error != nil && outcome.Result.Error.Details != nil {
			if reason, ok := outcome.Result.Error.Details["reason_code"].(string); ok {
				normalized := strings.ToLower(strings.TrimSpace(reason))
				if strings.HasPrefix(normalized, "sandbox.") {
					return normalized
				}
			}
		}
		if outcome.Result.Structured != nil {
			if reason, ok := outcome.Result.Structured["sandbox_fallback_reason"].(string); ok {
				normalized := strings.ToLower(strings.TrimSpace(reason))
				if strings.HasPrefix(normalized, "sandbox.") {
					return normalized
				}
			}
		}
	}
	return ""
}

func securityDecisionFromToolError(err *types.ClassifiedError) (securityDecision, bool) {
	if err == nil || err.Class != types.ErrSecurity {
		return securityDecision{}, false
	}
	reason := ""
	namespaceTool := ""
	policyKind := ""
	decisionValue := "deny"
	if err.Details != nil {
		if value, ok := err.Details["reason_code"].(string); ok {
			reason = strings.ToLower(strings.TrimSpace(value))
		}
		if value, ok := err.Details["namespace_tool"].(string); ok {
			namespaceTool = strings.ToLower(strings.TrimSpace(value))
		}
		if value, ok := err.Details["policy_kind"].(string); ok {
			policyKind = strings.ToLower(strings.TrimSpace(value))
		}
		if value, ok := err.Details["decision"].(string); ok && strings.TrimSpace(value) != "" {
			decisionValue = strings.ToLower(strings.TrimSpace(value))
		}
	}
	if !strings.HasPrefix(reason, "sandbox.") {
		return securityDecision{}, false
	}
	if policyKind == "" {
		policyKind = "sandbox"
	}
	return securityDecision{
		PolicyKind:    policyKind,
		NamespaceTool: namespaceTool,
		Decision:      decisionValue,
		ReasonCode:    reason,
	}, true
}

func cloneDetailsMap(in map[string]any) map[string]any {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]any, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}

func cloneIntMap(in map[string]int) map[string]int {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]int, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
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
	classifiedActionGate := func(
		call types.ToolCall,
		class types.ErrorClass,
		msg string,
		retryable bool,
		reasonCode string,
	) *types.ClassifiedError {
		ce := classified(class, msg, retryable)
		code := strings.ToLower(strings.TrimSpace(reasonCode))
		if code == "" {
			code = "action.gate.denied"
		}
		details := map[string]any{
			"policy_kind": "action_gate",
			"decision":    "deny",
			"reason_code": code,
		}
		if namespaceTool, ok := namespaceToolKey(call.Name); ok {
			details["namespace_tool"] = namespaceTool
		}
		if callID := strings.TrimSpace(call.CallID); callID != "" {
			details["call_id"] = callID
		}
		if tool := strings.TrimSpace(call.Name); tool != "" {
			details["tool"] = tool
		}
		if trace, ok := e.evaluateRuntimePolicyTrace([]runtimeconfig.RuntimePolicyCandidate{
			{
				Stage:    runtimeconfig.RuntimePolicyStageActionGate,
				Code:     code,
				Source:   runtimeconfig.RuntimePolicyStageActionGate,
				Decision: runtimeconfig.RuntimePolicyDecisionDeny,
			},
		}); ok {
			if strings.TrimSpace(trace.Version) != "" {
				details["policy_precedence_version"] = strings.TrimSpace(trace.Version)
			}
			if strings.TrimSpace(trace.WinnerStage) != "" {
				details["winner_stage"] = strings.TrimSpace(trace.WinnerStage)
			}
			if strings.TrimSpace(trace.DenySource) != "" {
				details["deny_source"] = strings.TrimSpace(trace.DenySource)
			}
			if strings.TrimSpace(trace.TieBreakReason) != "" {
				details["tie_break_reason"] = strings.TrimSpace(trace.TieBreakReason)
			}
			if len(trace.PolicyDecisionPath) > 0 {
				details["policy_decision_path"] = append([]runtimeconfig.RuntimePolicyCandidate(nil), trace.PolicyDecisionPath...)
			}
		}
		ce.Details = details
		return ce
	}

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
			ce := classifiedActionGate(
				call,
				types.ErrTool,
				fmt.Sprintf("action gate evaluate failed: %v", err),
				false,
				"action.gate.evaluate_failed",
			)
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
			return classifiedActionGate(call, types.ErrTool, msg, false, "action.gate.denied"), errors.New(msg)
		case types.ActionGateDecisionRequireConfirm:
			e.emitTimeline(ctx, h, runID, iteration, seq, types.ActionPhaseTool, types.ActionStatusPending, "gate.require_confirm")
			if e.actionGateResolver == nil {
				if stats != nil {
					stats.DeniedCount++
				}
				e.emitTimeline(ctx, h, runID, iteration, seq, types.ActionPhaseTool, types.ActionStatusFailed, "gate.denied")
				msg := fmt.Sprintf("action gate requires confirmation but resolver is not configured: %s", strings.TrimSpace(call.Name))
				return classifiedActionGate(call, types.ErrTool, msg, false, "action.gate.resolver_missing"), errors.New(msg)
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
					return classifiedActionGate(call, types.ErrPolicyTimeout, msg, true, "action.gate.timeout"), context.DeadlineExceeded
				}
				if stats != nil {
					stats.DeniedCount++
				}
				e.emitTimeline(ctx, h, runID, iteration, seq, types.ActionPhaseTool, types.ActionStatusFailed, "gate.denied")
				msg := fmt.Sprintf("action gate confirmation failed for tool %s: %v", strings.TrimSpace(call.Name), confirmErr)
				return classifiedActionGate(call, types.ErrTool, msg, false, "action.gate.confirmation_failed"), confirmErr
			}
			if !approved {
				if stats != nil {
					stats.DeniedCount++
				}
				e.emitTimeline(ctx, h, runID, iteration, seq, types.ActionPhaseTool, types.ActionStatusFailed, "gate.denied")
				msg := fmt.Sprintf("action gate confirmation denied tool call: %s", strings.TrimSpace(call.Name))
				return classifiedActionGate(call, types.ErrTool, msg, false, "action.gate.confirmation_denied"), errors.New(msg)
			}
		default:
			if stats != nil {
				stats.DeniedCount++
			}
			e.emitTimeline(ctx, h, runID, iteration, seq, types.ActionPhaseTool, types.ActionStatusFailed, "gate.denied")
			msg := fmt.Sprintf("action gate returned unsupported decision %q for tool %s", evaluation.Decision, strings.TrimSpace(call.Name))
			return classifiedActionGate(call, types.ErrTool, msg, false, "action.gate.decision_unsupported"), errors.New(msg)
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

func normalizeLifecycleHooks(hooks []types.AgentLifecycleHook) []types.AgentLifecycleHook {
	if len(hooks) == 0 {
		return nil
	}
	out := make([]types.AgentLifecycleHook, 0, len(hooks))
	for _, hook := range hooks {
		if hook == nil {
			continue
		}
		out = append(out, hook)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func normalizeToolMiddlewares(middlewares []types.ToolMiddleware) []types.ToolMiddleware {
	if len(middlewares) == 0 {
		return nil
	}
	out := make([]types.ToolMiddleware, 0, len(middlewares))
	for _, middleware := range middlewares {
		if middleware == nil {
			continue
		}
		out = append(out, middleware)
	}
	if len(out) == 0 {
		return nil
	}
	return out
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
