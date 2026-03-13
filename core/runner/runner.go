package runner

import (
	"context"
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
	model      types.ModelClient
	models     map[string]types.ModelClient
	modelOrder []string
	dispatcher *local.Dispatcher
	tracer     *obsTrace.Manager
	runtimeMgr *runtimeconfig.Manager
	assembler  *assembler.Assembler
	now        func() time.Time
	newRunID   func() string
	capCacheMu sync.RWMutex
	capCache   map[string]cachedCapabilities
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
			required := req.Capabilities.Normalized()
			modelReq := toModelRequest(runID, req, pendingOutcomes, required)
			assembledReq, assembleResult, assembleErr := e.assembler.Assemble(ctx, types.ContextAssembleRequest{
				RunID:         runID,
				SessionID:     req.SessionID,
				PrefixVersion: e.resolvePrefixVersion(),
				Input:         modelReq.Input,
				Messages:      modelReq.Messages,
				ToolResult:    modelReq.ToolResult,
				Capabilities:  modelReq.Capabilities,
			}, modelReq)
			lastAssemble = assembleResult
			if assembleErr != nil {
				terminal = classified(types.ErrContext, assembleErr.Error(), false)
				runErr = assembleErr
				state = StateAbort
				continue
			}
			modelReq = assembledReq
			selectedModel, selection, selErr := e.selectModelForStep(ctx, modelReq, false, len(required) > 0)
			if selErr != nil {
				terminal = selErr
				runErr = errors.New(selErr.Message)
				state = StateAbort
				continue
			}
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
				switch {
				case errors.As(err, &classifiedErr) && classifiedErr.ClassifiedError() != nil:
					terminal = classifiedErr.ClassifiedError()
				case errors.Is(err, context.DeadlineExceeded) || errors.Is(stepCtx.Err(), context.DeadlineExceeded):
					terminal = classified(types.ErrPolicyTimeout, "model step timed out", true)
				default:
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
	final := ""
	usage := types.TokenUsage{}
	selectionPath := make([]string, 0, 2)
	required := append(req.Capabilities.Normalized(), types.ModelCapabilityStreaming)
	modelReq := toModelRequest(runID, req, nil, required)
	assembledReq, assembleResult, assembleErr := e.assembler.Assemble(ctx, types.ContextAssembleRequest{
		RunID:         runID,
		SessionID:     req.SessionID,
		PrefixVersion: e.resolvePrefixVersion(),
		Input:         modelReq.Input,
		Messages:      modelReq.Messages,
		ToolResult:    modelReq.ToolResult,
		Capabilities:  modelReq.Capabilities,
	}, modelReq)
	lastAssemble := assembleResult

	e.emit(ctx, h, types.Event{Version: "v1", Type: "run.started", RunID: runID, Time: start})
	if assembleErr != nil {
		result := types.RunResult{
			RunID:      runID,
			Iterations: iteration,
			LatencyMs:  e.now().Sub(start).Milliseconds(),
			Error:      classified(types.ErrContext, assembleErr.Error(), false),
		}
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
	modelReq = assembledReq
	selectedModel, selection, selErr := e.selectModelForStep(ctx, modelReq, true, e.fallbackEnabled())
	if selErr != nil {
		result := types.RunResult{
			RunID:      runID,
			Iterations: iteration,
			LatencyMs:  e.now().Sub(start).Milliseconds(),
			Error:      selErr,
		}
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
				Assemble:     lastAssemble,
			}),
		})
		return result, errors.New(selErr.Message)
	}
	if selection.Provider != "" {
		selectionPath = append(selectionPath, selection.Provider)
	}
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
		return nil
	})
	modelSpan.End()
	cancel()
	if err != nil {
		var classifiedErr classifiedModelError
		terminal := classified(types.ErrModel, err.Error(), false)
		switch {
		case errors.As(err, &classifiedErr) && classifiedErr.ClassifiedError() != nil:
			terminal = classifiedErr.ClassifiedError()
		case errors.Is(err, context.DeadlineExceeded) || errors.Is(stepCtx.Err(), context.DeadlineExceeded):
			terminal = classified(types.ErrPolicyTimeout, "model stream timed out", true)
		}
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
			}),
		})
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
	if meta.Assemble.Stage.Stage2HitCount > 0 {
		payload["stage2_hit_count"] = meta.Assemble.Stage.Stage2HitCount
	}
	if meta.Assemble.Stage.Stage2Source != "" {
		payload["stage2_source"] = meta.Assemble.Stage.Stage2Source
	}
	if meta.Assemble.Stage.Stage2Reason != "" {
		payload["stage2_reason"] = meta.Assemble.Stage.Stage2Reason
	}
	if meta.Assemble.Recap.Status != "" {
		payload["recap_status"] = string(meta.Assemble.Recap.Status)
	}
	return payload
}

func (e *Engine) resolvePrefixVersion() string {
	if e.runtimeMgr == nil {
		return runtimeconfig.DefaultConfig().ContextAssembler.PrefixVersion
	}
	return e.runtimeMgr.EffectiveConfig().ContextAssembler.PrefixVersion
}

var _ types.Runner = (*Engine)(nil)
