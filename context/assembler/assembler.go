package assembler

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/FelixSeptem/baymax/context/guard"
	"github.com/FelixSeptem/baymax/context/journal"
	"github.com/FelixSeptem/baymax/context/provider"
	"github.com/FelixSeptem/baymax/core/types"
	runtimeconfig "github.com/FelixSeptem/baymax/runtime/config"
	"github.com/FelixSeptem/baymax/runtime/security/redaction"
)

const (
	StatusSuccess = "success"
	StatusFailed  = "failed"
	StatusBypass  = "bypass"
)

const (
	stage2RouterModeRules   = "rules"
	stage2RouterModeAgentic = "agentic"

	stage2RouterDecisionRun        = "run_stage2"
	stage2RouterDecisionSkip       = "skip_stage2"
	stage2RouterErrCallbackMissing = "agentic.callback_missing"
	stage2RouterErrCallbackTimeout = "agentic.callback_timeout"
	stage2RouterErrCallbackError   = "agentic.callback_error"
	stage2RouterErrInvalidDecision = "agentic.invalid_decision"

	contextRecapSourceTaskAwareV1 = "task_aware.stage_actions.v1"
)

// AgenticRoutingRequest is the normalized callback input for context stage2 agentic routing.
type AgenticRoutingRequest struct {
	RunID         string
	SessionID     string
	ModelProvider string
	Model         string
	Input         string
	Messages      []types.Message
	Capabilities  []types.ModelCapability
}

// AgenticRoutingDecision is the callback output for context stage2 routing.
type AgenticRoutingDecision struct {
	RunStage2 bool
	Reason    string
}

// AgenticRouter decides whether Stage2 should run for a context stage2 assemble cycle.
type AgenticRouter interface {
	DecideStage2(ctx context.Context, req AgenticRoutingRequest) (AgenticRoutingDecision, error)
}

// AgenticRouterFunc adapts a function to AgenticRouter.
type AgenticRouterFunc func(ctx context.Context, req AgenticRoutingRequest) (AgenticRoutingDecision, error)

// DecideStage2 invokes wrapped function.
func (f AgenticRouterFunc) DecideStage2(ctx context.Context, req AgenticRoutingRequest) (AgenticRoutingDecision, error) {
	return f(ctx, req)
}

// Assembler composes context before model execution using prefix-baseline, stage2-routing, and pressure-compaction policies.
type Assembler struct {
	cfgProvider          func() runtimeconfig.ContextAssemblerConfig
	redactionCfgProvider func() runtimeconfig.SecurityRedactionConfig
	memoryCfgProvider    func() runtimeconfig.RuntimeMemoryConfig
	runtimeContextConfig func() runtimeconfig.RuntimeContextConfig
	now                  func() time.Time

	mu              sync.Mutex
	storageKey      string
	storage         journal.Storage
	stage2Key       string
	stage2Provider  provider.Provider
	prefixCache     map[string]string
	pressureState   map[string]*pressureRunState
	spillBackend    SpillBackend
	spillBackendKey string
	embeddingScorer SemanticEmbeddingScorer
	embeddingKey    string
	rerankers       map[string]SemanticReranker
	defaultReranker SemanticReranker
	agenticRouter   AgenticRouter
}

// Option customizes assembler behavior for embedding/reranker/redaction integrations.
type Option func(*Assembler)

// WithRedactionConfigProvider injects runtime redaction config provider for recap/stage2 sanitization.
func WithRedactionConfigProvider(provider func() runtimeconfig.SecurityRedactionConfig) Option {
	return func(a *Assembler) {
		if provider != nil {
			a.redactionCfgProvider = provider
		}
	}
}

// WithMemoryConfigProvider injects runtime memory config provider for stage2 memory facade mode.
func WithMemoryConfigProvider(provider func() runtimeconfig.RuntimeMemoryConfig) Option {
	return func(a *Assembler) {
		if provider != nil {
			a.memoryCfgProvider = provider
		}
	}
}

// WithRuntimeContextConfigProvider injects runtime context-domain config provider for JIT controls.
func WithRuntimeContextConfigProvider(provider func() runtimeconfig.RuntimeContextConfig) Option {
	return func(a *Assembler) {
		if provider != nil {
			a.runtimeContextConfig = provider
		}
	}
}

// WithSemanticEmbeddingScorer registers semantic embedding scorer extension.
func WithSemanticEmbeddingScorer(key string, scorer SemanticEmbeddingScorer) Option {
	return func(a *Assembler) {
		a.embeddingKey = strings.TrimSpace(key)
		a.embeddingScorer = scorer
	}
}

// WithSemanticReranker registers provider-specific semantic reranker extension.
func WithSemanticReranker(provider string, reranker SemanticReranker) Option {
	return func(a *Assembler) {
		if reranker == nil {
			return
		}
		key := strings.ToLower(strings.TrimSpace(provider))
		if key == "" {
			return
		}
		if a.rerankers == nil {
			a.rerankers = map[string]SemanticReranker{}
		}
		a.rerankers[key] = reranker
	}
}

// WithAgenticRouter registers a host callback for context stage2 agentic routing decisions.
func WithAgenticRouter(router AgenticRouter) Option {
	return func(a *Assembler) {
		a.agenticRouter = router
	}
}

// New creates a context assembler with runtime config provider and optional extensions.
func New(cfgProvider func() runtimeconfig.ContextAssemblerConfig, opts ...Option) *Assembler {
	baseSecurity := runtimeconfig.DefaultConfig().Security.Redaction
	a := &Assembler{
		cfgProvider: cfgProvider,
		redactionCfgProvider: func() runtimeconfig.SecurityRedactionConfig {
			return baseSecurity
		},
		memoryCfgProvider: func() runtimeconfig.RuntimeMemoryConfig {
			return runtimeconfig.DefaultConfig().Runtime.Memory
		},
		runtimeContextConfig: func() runtimeconfig.RuntimeContextConfig {
			return runtimeconfig.DefaultConfig().Runtime.Context
		},
		now:             time.Now,
		prefixCache:     map[string]string{},
		pressureState:   map[string]*pressureRunState{},
		rerankers:       map[string]SemanticReranker{},
		defaultReranker: &defaultSemanticReranker{},
	}
	for _, opt := range opts {
		if opt != nil {
			opt(a)
		}
	}
	return a
}

// SetAgenticRouter updates agentic router callback at runtime.
func (a *Assembler) SetAgenticRouter(router AgenticRouter) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.agenticRouter = router
}

// Assemble builds model-ready context with prefix-baseline, stage2-routing, and pressure-compaction policies.
func (a *Assembler) Assemble(ctx context.Context, req types.ContextAssembleRequest, modelReq types.ModelRequest) (types.ModelRequest, types.ContextAssembleResult, error) {
	start := a.now()
	cfg := a.cfgProvider()
	stage2Config, pressureConfig := cfg.CA2, cfg.CA3
	if !cfg.Enabled {
		return modelReq, types.ContextAssembleResult{
			Prefix: types.PrefixMetadata{
				SessionID:     req.SessionID,
				PrefixVersion: req.PrefixVersion,
			},
			LatencyMs: 0,
			Status:    StatusBypass,
			Stage: types.AssembleStage{
				Status: types.AssembleStageStatusBypass,
			},
			Recap: types.RecapMetadata{Status: types.RecapStatusDisabled},
		}, nil
	}

	storage, err := a.ensureStorage(cfg)
	if err != nil {
		return modelReq, failedResult(req, start, "storage.backend.not_ready"), err
	}
	req.PrefixVersion = strings.TrimSpace(req.PrefixVersion)
	if req.PrefixVersion == "" {
		req.PrefixVersion = strings.TrimSpace(cfg.PrefixVersion)
	}
	g := guard.New(cfg.Guard.FailFast)
	stage1Start := a.now()
	prefixHash, err := buildPrefixHash(req)
	if err != nil {
		return modelReq, failedResult(req, start, "prefix.build.failed"), err
	}

	sessionKey := stableSessionKey(req.SessionID, req.RunID, req.PrefixVersion)
	expected := a.cachedHash(sessionKey)
	guardResult, guardErr := g.Apply(req, prefixHash, expected)

	intent := journal.Entry{
		Time:          start,
		RunID:         req.RunID,
		SessionID:     req.SessionID,
		Phase:         "intent",
		PrefixVersion: req.PrefixVersion,
		PrefixHash:    prefixHash,
	}
	if err := storage.Append(ctx, intent); err != nil {
		return modelReq, failedResult(req, start, "journal.intent.write_failed"), err
	}

	outcome := types.ContextAssembleResult{
		Prefix: types.PrefixMetadata{
			SessionID:     req.SessionID,
			PrefixVersion: req.PrefixVersion,
			PrefixHash:    prefixHash,
		},
		Status: StatusSuccess,
		Stage: types.AssembleStage{
			Status:          types.AssembleStageStatusStage1Only,
			Stage1LatencyMs: a.now().Sub(stage1Start).Milliseconds(),
		},
		Recap: types.RecapMetadata{Status: types.RecapStatusDisabled},
	}

	if guardErr != nil {
		outcome.GuardFailure = guardResult.GuardViolation
		if !isBestEffortPolicy(stage2Config.StagePolicy.Stage1) {
			commit := journal.Entry{
				Time:          a.now(),
				RunID:         req.RunID,
				SessionID:     req.SessionID,
				Phase:         "commit",
				PrefixVersion: req.PrefixVersion,
				PrefixHash:    prefixHash,
				Status:        StatusFailed,
				Violation:     guardResult.GuardViolation,
			}
			_ = storage.Append(ctx, commit)
			return modelReq, failedResult(req, start, guardResult.GuardViolation), guardErr
		}
		outcome.Status = StatusSuccess
		outcome.Stage.Status = types.AssembleStageStatusDegraded
		outcome.Stage.Stage2SkipReason = "stage1.best_effort.guard_violation"
	}

	a.rememberHash(sessionKey, prefixHash)
	modelReq.Input = guardResult.Input
	modelReq.Messages = guardResult.Messages

	var pressureGateDecision pressureDecision
	if pressureConfig.Enabled {
		updatedReq, updatedOutcome, decision, err := a.applyPressureCompactionAndSwapback(ctx, req, modelReq, outcome, cfg, "stage1")
		if err != nil {
			commit := journal.Entry{
				Time:          a.now(),
				RunID:         req.RunID,
				SessionID:     req.SessionID,
				Phase:         "commit",
				PrefixVersion: req.PrefixVersion,
				PrefixHash:    prefixHash,
				Status:        StatusFailed,
				Violation:     err.Error(),
			}
			_ = storage.Append(ctx, commit)
			return modelReq, failedResult(req, start, err.Error()), err
		}
		modelReq = updatedReq
		outcome = updatedOutcome
		pressureGateDecision = decision
	}

	if stage2Config.Enabled {
		modelReq, outcome, err = a.applyStage2RoutingAndDisclosure(ctx, modelReq, req, cfg, outcome, pressureGateDecision)
		if err != nil {
			commit := journal.Entry{
				Time:          a.now(),
				RunID:         req.RunID,
				SessionID:     req.SessionID,
				Phase:         "commit",
				PrefixVersion: req.PrefixVersion,
				PrefixHash:    prefixHash,
				Status:        StatusFailed,
				Violation:     err.Error(),
			}
			_ = storage.Append(ctx, commit)
			return modelReq, failedResult(req, start, err.Error()), err
		}
	}
	if pressureConfig.Enabled {
		modelReq, outcome, _, err = a.applyPressureCompactionAndSwapback(ctx, req, modelReq, outcome, cfg, "stage2")
		if err != nil {
			commit := journal.Entry{
				Time:          a.now(),
				RunID:         req.RunID,
				SessionID:     req.SessionID,
				Phase:         "commit",
				PrefixVersion: req.PrefixVersion,
				PrefixHash:    prefixHash,
				Status:        StatusFailed,
				Violation:     err.Error(),
			}
			_ = storage.Append(ctx, commit)
			return modelReq, failedResult(req, start, err.Error()), err
		}
	}

	commit := journal.Entry{
		Time:          a.now(),
		RunID:         req.RunID,
		SessionID:     req.SessionID,
		Phase:         "commit",
		PrefixVersion: req.PrefixVersion,
		PrefixHash:    prefixHash,
		Status:        outcome.Status,
		Violation:     outcome.GuardFailure,
	}
	if err := storage.Append(ctx, commit); err != nil {
		return modelReq, failedResult(req, start, "journal.commit.write_failed"), err
	}
	outcome.LatencyMs = a.now().Sub(start).Milliseconds()
	return modelReq, outcome, nil
}

func (a *Assembler) ensureStorage(cfg runtimeconfig.ContextAssemblerConfig) (journal.Storage, error) {
	key := strings.ToLower(strings.TrimSpace(cfg.Storage.Backend)) + "|" + strings.TrimSpace(cfg.JournalPath)
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.storage != nil && a.storageKey == key {
		return a.storage, nil
	}
	s, err := journal.NewStorage(cfg.Storage.Backend, cfg.JournalPath)
	if err != nil {
		return nil, err
	}
	a.storage = s
	a.storageKey = key
	return a.storage, nil
}

func (a *Assembler) ensureStage2Provider(cfg runtimeconfig.ContextAssemblerConfig) (provider.Provider, error) {
	stage2KeyRaw, _ := json.Marshal(cfg.CA2.Stage2)
	key := string(stage2KeyRaw)
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.stage2Provider != nil && a.stage2Key == key {
		return a.stage2Provider, nil
	}
	p, err := provider.NewWithConfig(provider.Config{
		Name:     cfg.CA2.Stage2.Provider,
		FilePath: cfg.CA2.Stage2.FilePath,
		External: cfg.CA2.Stage2.External,
		Memory:   a.memoryCfgProvider(),
	})
	if err != nil {
		return nil, err
	}
	a.stage2Provider = p
	a.stage2Key = key
	return a.stage2Provider, nil
}

func (a *Assembler) cachedHash(key string) string {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.prefixCache[key]
}

func (a *Assembler) rememberHash(key, hash string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.prefixCache[key] = hash
}

func (a *Assembler) snapshotAgenticRouter() AgenticRouter {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.agenticRouter
}

func stableSessionKey(sessionID, runID, prefixVersion string) string {
	base := strings.TrimSpace(sessionID)
	if base == "" {
		base = strings.TrimSpace(runID)
	}
	return base + "|" + strings.TrimSpace(prefixVersion)
}

func buildPrefixHash(req types.ContextAssembleRequest) (string, error) {
	systemMessages := make([]string, 0, len(req.Messages))
	for _, m := range req.Messages {
		if strings.EqualFold(strings.TrimSpace(m.Role), "system") {
			systemMessages = append(systemMessages, strings.TrimSpace(m.Content))
		}
	}
	payload := map[string]any{
		"prefix_version":  req.PrefixVersion,
		"system_messages": systemMessages,
		"capabilities":    req.Capabilities.Normalized(),
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("marshal prefix blocks: %w", err)
	}
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:]), nil
}

func failedResult(req types.ContextAssembleRequest, start time.Time, violation string) types.ContextAssembleResult {
	return types.ContextAssembleResult{
		Prefix: types.PrefixMetadata{
			SessionID:     req.SessionID,
			PrefixVersion: req.PrefixVersion,
		},
		LatencyMs:    time.Since(start).Milliseconds(),
		Status:       StatusFailed,
		GuardFailure: violation,
		Stage: types.AssembleStage{
			Status: types.AssembleStageStatusFailed,
		},
		Recap: types.RecapMetadata{Status: types.RecapStatusFailed},
	}
}

func (a *Assembler) applyStage2RoutingAndDisclosure(
	ctx context.Context,
	modelReq types.ModelRequest,
	req types.ContextAssembleRequest,
	cfg runtimeconfig.ContextAssemblerConfig,
	outcome types.ContextAssembleResult,
	pressure pressureDecision,
) (types.ModelRequest, types.ContextAssembleResult, error) {
	stage2Config := cfg.CA2
	mode := normalizedStage2RouterMode(stage2Config.RoutingMode)
	outcome.Stage.Stage2RouterMode = mode
	shouldStage2, skipReason, routerReason, routerError, routerLatency := a.resolveStage2Decision(ctx, req, modelReq, cfg)
	outcome.Stage.Stage2RouterLatencyMs = routerLatency
	outcome.Stage.Stage2RouterError = routerError
	if shouldStage2 {
		outcome.Stage.Stage2RouterDecision = stage2RouterDecisionRun
	} else {
		outcome.Stage.Stage2RouterDecision = stage2RouterDecisionSkip
	}
	outcome.Stage.Stage2RouterReason = routerReason
	if !shouldStage2 {
		outcome.Stage.Status = types.AssembleStageStatusStage1Only
		outcome.Stage.Stage2SkipReason = skipReason
		modelReq, recap := a.appendTailRecap(modelReq, stage2Config, &outcome)
		outcome.Recap = recap
		return modelReq, outcome, nil
	}
	if pressure.BlockLowPriorityLoads && !isHighPriorityRequest(modelReq.Input, cfg.CA3.Emergency.HighPriorityTokens) {
		outcome.Stage.Status = types.AssembleStageStatusDegraded
		outcome.Stage.Stage2SkipReason = "ca3.emergency.low_priority_rejected"
		modelReq, recap := a.appendTailRecap(modelReq, stage2Config, &outcome)
		outcome.Recap = recap
		return modelReq, outcome, nil
	}
	stage2Start := a.now()
	p, err := a.ensureStage2Provider(cfg)
	if err != nil {
		if isBestEffortPolicy(stage2Config.StagePolicy.Stage2) {
			outcome.Stage.Status = types.AssembleStageStatusDegraded
			outcome.Stage.Stage2SkipReason = "stage2.provider.not_ready"
			modelReq, recap := a.appendTailRecap(modelReq, stage2Config, &outcome)
			outcome.Recap = recap
			return modelReq, outcome, nil
		}
		return modelReq, outcome, err
	}
	outcome.Stage.Stage2Provider = p.Name()
	outcome.Stage.Stage2Profile = stage2ProfileFromConfig(stage2Config.Stage2.External.Profile, p.Name())
	outcome.Stage.Stage2TemplateProfile = stage2TemplateProfileFromConfig(stage2Config.Stage2.External.Profile, p.Name())
	outcome.Stage.Stage2TemplateResolutionSource = stage2TemplateResolutionSourceFromConfig(
		stage2Config.Stage2.External.TemplateResolutionSource,
		stage2Config.Stage2.External.Profile,
		p.Name(),
	)
	stage2Ctx, cancel := context.WithTimeout(ctx, stage2Config.Timeout.Stage2)
	defer cancel()
	resp, err := p.Fetch(stage2Ctx, provider.Request{
		RunID:     req.RunID,
		SessionID: req.SessionID,
		Input:     modelReq.Input,
		MaxItems:  stage2Config.TailRecap.MaxItems,
		Hints:     stage2HintsFromConfig(stage2Config.Stage2.External.Hints),
	})
	outcome.Stage.Stage2LatencyMs = a.now().Sub(stage2Start).Milliseconds()
	if err != nil {
		reason, reasonCode, errorLayer := stage2ReasonFromError(err)
		outcome.Stage.Stage2Reason = reason
		outcome.Stage.Stage2ReasonCode = reasonCode
		outcome.Stage.Stage2ErrorLayer = errorLayer
		if isBestEffortPolicy(stage2Config.StagePolicy.Stage2) {
			outcome.Stage.Status = types.AssembleStageStatusDegraded
			outcome.Stage.Stage2SkipReason = "stage2.fetch.failed"
			modelReq, recap := a.appendTailRecap(modelReq, stage2Config, &outcome)
			outcome.Recap = recap
			return modelReq, outcome, nil
		}
		return modelReq, outcome, err
	}
	if len(resp.Chunks) == 0 {
		outcome.Stage.Status = types.AssembleStageStatusStage1Only
		outcome.Stage.Stage2SkipReason = "stage2.empty"
		outcome.Stage.Stage2Reason = stage2ReasonFromMeta(resp.Meta, "empty")
		outcome.Stage.Stage2ReasonCode = stage2ReasonCodeFromMeta(resp.Meta, "empty")
		outcome.Stage.Stage2ErrorLayer = stage2ErrorLayerFromMeta(resp.Meta, "")
		outcome.Stage.Stage2Source = sourceFromMeta(resp.Meta, p.Name())
		outcome.Stage.Stage2Profile = stage2ProfileFromMeta(resp.Meta, outcome.Stage.Stage2Profile)
		outcome.Stage.Stage2TemplateProfile = stage2TemplateProfileFromMeta(resp.Meta, outcome.Stage.Stage2TemplateProfile)
		outcome.Stage.Stage2TemplateResolutionSource = stage2TemplateResolutionSourceFromMeta(
			resp.Meta,
			outcome.Stage.Stage2TemplateResolutionSource,
		)
		outcome.Stage.Stage2HintApplied = stage2HintAppliedFromMeta(resp.Meta)
		outcome.Stage.Stage2HintMismatchReason = stage2HintMismatchReasonFromMeta(resp.Meta)
		outcome.Stage.MemoryScopeSelected = memoryScopeSelectedFromMeta(resp.Meta)
		outcome.Stage.MemoryBudgetUsed = memoryBudgetUsedFromMeta(resp.Meta)
		outcome.Stage.MemoryHits = memoryHitsFromMeta(resp.Meta, 0)
		outcome.Stage.MemoryRerankStats = memoryRerankStatsFromMeta(resp.Meta)
		outcome.Stage.MemoryLifecycleAction = memoryLifecycleActionFromMeta(resp.Meta)
		modelReq, recap := a.appendTailRecap(modelReq, stage2Config, &outcome)
		outcome.Recap = recap
		return modelReq, outcome, nil
	}
	resp.Chunks = a.sanitizeStage2Chunks(resp.Chunks)
	resolvedChunks := append([]string(nil), resp.Chunks...)
	stage2ReasonOverride, stage2ReasonCodeOverride := "", ""
	runtimeContextCfg := runtimeconfig.DefaultConfig().Runtime.Context
	if a.runtimeContextConfig != nil {
		runtimeContextCfg = a.runtimeContextConfig()
	}
	stage2Source := sourceFromMeta(resp.Meta, p.Name())
	if runtimeContextCfg.JIT.IsolateHandoff.Enabled {
		ingestedChunks, handoffPayload, err := ingestIsolateHandoffChunks(
			resolvedChunks,
			stage2Source,
			a.now(),
			runtimeContextCfg.JIT.IsolateHandoff,
			stage2Config.StagePolicy.Stage2,
		)
		if err != nil {
			if isBestEffortPolicy(stage2Config.StagePolicy.Stage2) {
				outcome.Stage.Status = types.AssembleStageStatusDegraded
				outcome.Stage.Stage2SkipReason = "stage2.isolate_handoff.failed"
				modelReq, recap := a.appendTailRecap(modelReq, stage2Config, &outcome)
				outcome.Recap = recap
				return modelReq, outcome, nil
			}
			return modelReq, outcome, err
		}
		resolvedChunks = ingestedChunks
		if handoffPayload.AcceptedTotal > 0 || handoffPayload.RejectedTotal > 0 {
			raw, err := json.Marshal(handoffPayload)
			if err != nil {
				if isBestEffortPolicy(stage2Config.StagePolicy.Stage2) {
					outcome.Stage.Status = types.AssembleStageStatusDegraded
					outcome.Stage.Stage2SkipReason = "stage2.isolate_handoff.serialize_failed"
					modelReq, recap := a.appendTailRecap(modelReq, stage2Config, &outcome)
					outcome.Recap = recap
					return modelReq, outcome, nil
				}
				return modelReq, outcome, err
			}
			modelReq.Messages = append(modelReq.Messages, types.Message{
				Role:    "system",
				Content: "stage2_isolate_handoff:" + string(raw),
			})
		}
		if handoffPayload.RejectedTotal > 0 {
			stage2ReasonOverride = "isolate_handoff_rejected"
			stage2ReasonCodeOverride = "isolate_handoff_rejected"
		}
		if len(resolvedChunks) == 0 {
			outcome.Stage.Status = types.AssembleStageStatusStage1Only
			outcome.Stage.Stage2SkipReason = "stage2.isolate_handoff.empty"
			modelReq, recap := a.appendTailRecap(modelReq, stage2Config, &outcome)
			outcome.Recap = recap
			return modelReq, outcome, nil
		}
	}
	if runtimeContextCfg.JIT.ReferenceFirst.Enabled {
		discovery, catalog := discoverStage2References(
			resolvedChunks,
			stage2Source,
			runtimeContextCfg.JIT.ReferenceFirst.MaxRefs,
		)
		outcome.Stage.ContextRefDiscoverCount = len(discovery.References)
		discoveryRaw, err := json.Marshal(discovery)
		if err != nil {
			if isBestEffortPolicy(stage2Config.StagePolicy.Stage2) {
				outcome.Stage.Status = types.AssembleStageStatusDegraded
				outcome.Stage.Stage2SkipReason = "stage2.discover_refs.failed"
				modelReq, recap := a.appendTailRecap(modelReq, stage2Config, &outcome)
				outcome.Recap = recap
				return modelReq, outcome, nil
			}
			return modelReq, outcome, err
		}
		modelReq.Messages = append(modelReq.Messages, types.Message{
			Role:    "system",
			Content: "stage2_refs:" + string(discoveryRaw),
		})
		resolution, err := resolveSelectedStage2References(
			discovery.References,
			catalog,
			runtimeContextCfg.JIT.ReferenceFirst.MaxResolveTokens,
			resolveReferenceMissingPolicy(stage2Config.StagePolicy.Stage2),
		)
		if err != nil {
			if isBestEffortPolicy(stage2Config.StagePolicy.Stage2) {
				outcome.Stage.Status = types.AssembleStageStatusDegraded
				outcome.Stage.Stage2SkipReason = "stage2.resolve_refs.failed"
				modelReq, recap := a.appendTailRecap(modelReq, stage2Config, &outcome)
				outcome.Recap = recap
				return modelReq, outcome, nil
			}
			return modelReq, outcome, err
		}
		outcome.Stage.ContextRefResolveCount = len(resolution.Resolved)
		if len(resolution.Missing) > 0 {
			stage2ReasonOverride = "partial_missing_refs"
			stage2ReasonCodeOverride = "partial_missing_refs"
		}
		resolutionRaw, err := json.Marshal(resolution)
		if err != nil {
			if isBestEffortPolicy(stage2Config.StagePolicy.Stage2) {
				outcome.Stage.Status = types.AssembleStageStatusDegraded
				outcome.Stage.Stage2SkipReason = "stage2.resolve_refs.serialize_failed"
				modelReq, recap := a.appendTailRecap(modelReq, stage2Config, &outcome)
				outcome.Recap = recap
				return modelReq, outcome, nil
			}
			return modelReq, outcome, err
		}
		modelReq.Messages = append(modelReq.Messages, types.Message{
			Role:    "system",
			Content: "stage2_resolved_refs:" + string(resolutionRaw),
		})
		resolvedChunks = resolvedChunks[:0]
		for _, item := range resolution.Resolved {
			resolvedChunks = append(resolvedChunks, item.Content)
		}
		if len(resolvedChunks) == 0 {
			outcome.Stage.Status = types.AssembleStageStatusStage1Only
			outcome.Stage.Stage2SkipReason = "stage2.resolve_refs.empty"
			modelReq, recap := a.appendTailRecap(modelReq, stage2Config, &outcome)
			outcome.Recap = recap
			return modelReq, outcome, nil
		}
	}
	if runtimeContextCfg.JIT.EditGate.Enabled {
		editGate := applyContextEditGate(resolvedChunks, runtimeContextCfg.JIT.EditGate)
		outcome.Stage.ContextEditEstimatedSavedTokens = editGate.EstimatedSavedTokens
		outcome.Stage.ContextEditGateDecision = editGate.Decision
		resolvedChunks = editGate.Chunks
	}
	if len(resolvedChunks) == 0 {
		outcome.Stage.Status = types.AssembleStageStatusStage1Only
		outcome.Stage.Stage2SkipReason = "stage2.context.empty"
		modelReq, recap := a.appendTailRecap(modelReq, stage2Config, &outcome)
		outcome.Recap = recap
		return modelReq, outcome, nil
	}
	modelReq.Messages = append(modelReq.Messages, types.Message{
		Role:    "system",
		Content: "stage2_context:\n" + strings.Join(resolvedChunks, "\n"),
	})
	outcome.Stage.Status = types.AssembleStageStatusStage2Used
	outcome.Stage.Stage2SkipReason = ""
	outcome.Stage.Stage2HitCount = len(resolvedChunks)
	outcome.Stage.Stage2Source = stage2Source
	outcome.Stage.Stage2Reason = stage2ReasonFromMeta(resp.Meta, "ok")
	if stage2ReasonOverride != "" {
		outcome.Stage.Stage2Reason = stage2ReasonOverride
	}
	outcome.Stage.Stage2ReasonCode = stage2ReasonCodeFromMeta(resp.Meta, "ok")
	if stage2ReasonCodeOverride != "" {
		outcome.Stage.Stage2ReasonCode = stage2ReasonCodeOverride
	}
	outcome.Stage.Stage2ErrorLayer = stage2ErrorLayerFromMeta(resp.Meta, "")
	outcome.Stage.Stage2Profile = stage2ProfileFromMeta(resp.Meta, outcome.Stage.Stage2Profile)
	outcome.Stage.Stage2TemplateProfile = stage2TemplateProfileFromMeta(resp.Meta, outcome.Stage.Stage2TemplateProfile)
	outcome.Stage.Stage2TemplateResolutionSource = stage2TemplateResolutionSourceFromMeta(
		resp.Meta,
		outcome.Stage.Stage2TemplateResolutionSource,
	)
	outcome.Stage.Stage2HintApplied = stage2HintAppliedFromMeta(resp.Meta)
	outcome.Stage.Stage2HintMismatchReason = stage2HintMismatchReasonFromMeta(resp.Meta)
	outcome.Stage.MemoryScopeSelected = memoryScopeSelectedFromMeta(resp.Meta)
	outcome.Stage.MemoryBudgetUsed = memoryBudgetUsedFromMeta(resp.Meta)
	outcome.Stage.MemoryHits = memoryHitsFromMeta(resp.Meta, len(resolvedChunks))
	outcome.Stage.MemoryRerankStats = memoryRerankStatsFromMeta(resp.Meta)
	outcome.Stage.MemoryLifecycleAction = memoryLifecycleActionFromMeta(resp.Meta)
	modelReq, recap := a.appendTailRecap(modelReq, stage2Config, &outcome)
	outcome.Recap = recap
	return modelReq, outcome, nil
}

func (a *Assembler) resolveStage2Decision(
	ctx context.Context,
	req types.ContextAssembleRequest,
	modelReq types.ModelRequest,
	cfg runtimeconfig.ContextAssemblerConfig,
) (bool, string, string, string, int64) {
	stage2Config, mode := cfg.CA2, normalizedStage2RouterMode(cfg.CA2.RoutingMode)
	if mode != stage2RouterModeAgentic {
		shouldStage2, skipReason := shouldRunStage2(modelReq, stage2Config.Routing)
		if shouldStage2 {
			return true, "", "rules.threshold.met", "", 0
		}
		return false, skipReason, skipReason, "", 0
	}

	start := a.now()
	router := a.snapshotAgenticRouter()
	if router == nil {
		return fallbackStage2Decision(modelReq, stage2Config.Routing, stage2RouterErrCallbackMissing, 0)
	}

	timeout := stage2Config.Agentic.DecisionTimeout
	if timeout <= 0 {
		timeout = runtimeconfig.DefaultConfig().ContextAssembler.CA2.Agentic.DecisionTimeout
	}
	routerCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	decision, err := router.DecideStage2(routerCtx, AgenticRoutingRequest{
		RunID:         req.RunID,
		SessionID:     req.SessionID,
		ModelProvider: req.ModelProvider,
		Model:         req.Model,
		Input:         modelReq.Input,
		Messages:      append([]types.Message(nil), modelReq.Messages...),
		Capabilities:  req.Capabilities.Normalized(),
	})
	latency := a.now().Sub(start).Milliseconds()
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(routerCtx.Err(), context.DeadlineExceeded) {
		return fallbackStage2Decision(modelReq, stage2Config.Routing, stage2RouterErrCallbackTimeout, latency)
	}
	if err != nil {
		return fallbackStage2Decision(modelReq, stage2Config.Routing, stage2RouterErrCallbackError, latency)
	}
	reason := strings.TrimSpace(decision.Reason)
	if reason == "" {
		return fallbackStage2Decision(modelReq, stage2Config.Routing, stage2RouterErrInvalidDecision, latency)
	}
	if decision.RunStage2 {
		return true, "", reason, "", latency
	}
	return false, "routing.agentic.skip", reason, "", latency
}

func fallbackStage2Decision(
	modelReq types.ModelRequest,
	routing runtimeconfig.ContextAssemblerCA2RoutingConfig,
	routerError string,
	latencyMs int64,
) (bool, string, string, string, int64) {
	shouldStage2, skipReason := shouldRunStage2(modelReq, routing)
	baseReason := "agentic.fallback." + routerError
	if shouldStage2 {
		return true, "", baseReason, routerError, latencyMs
	}
	if skipReason != "" {
		return false, skipReason, baseReason + "|" + skipReason, routerError, latencyMs
	}
	return false, "routing.threshold.not_met", baseReason, routerError, latencyMs
}

func normalizedStage2RouterMode(raw string) string {
	if strings.EqualFold(strings.TrimSpace(raw), stage2RouterModeAgentic) {
		return stage2RouterModeAgentic
	}
	return stage2RouterModeRules
}

func sourceFromMeta(meta map[string]any, fallback string) string {
	if len(meta) == 0 {
		return fallback
	}
	if source, ok := meta["source"].(string); ok && strings.TrimSpace(source) != "" {
		return strings.TrimSpace(source)
	}
	return fallback
}

func stage2ReasonFromMeta(meta map[string]any, fallback string) string {
	if len(meta) == 0 {
		return fallback
	}
	if reason, ok := meta["reason"].(string); ok && strings.TrimSpace(reason) != "" {
		return strings.TrimSpace(reason)
	}
	return fallback
}

func stage2ReasonCodeFromMeta(meta map[string]any, fallback string) string {
	if len(meta) == 0 {
		return fallback
	}
	if reasonCode, ok := meta["reason_code"].(string); ok && strings.TrimSpace(reasonCode) != "" {
		return strings.TrimSpace(reasonCode)
	}
	return fallback
}

func stage2ErrorLayerFromMeta(meta map[string]any, fallback string) string {
	if len(meta) == 0 {
		return fallback
	}
	if layer, ok := meta["error_layer"].(string); ok && strings.TrimSpace(layer) != "" {
		return strings.TrimSpace(layer)
	}
	return fallback
}

func stage2ProfileFromMeta(meta map[string]any, fallback string) string {
	if len(meta) == 0 {
		return fallback
	}
	if profile, ok := meta["profile"].(string); ok && strings.TrimSpace(profile) != "" {
		return strings.TrimSpace(profile)
	}
	return fallback
}

func stage2TemplateProfileFromMeta(meta map[string]any, fallback string) string {
	if len(meta) == 0 {
		return fallback
	}
	if profile, ok := meta["template_profile"].(string); ok && strings.TrimSpace(profile) != "" {
		return strings.TrimSpace(profile)
	}
	return fallback
}

func stage2TemplateResolutionSourceFromMeta(meta map[string]any, fallback string) string {
	if len(meta) == 0 {
		return fallback
	}
	if source, ok := meta["template_resolution_source"].(string); ok && strings.TrimSpace(source) != "" {
		return strings.TrimSpace(source)
	}
	return fallback
}

func stage2HintAppliedFromMeta(meta map[string]any) bool {
	if len(meta) == 0 {
		return false
	}
	applied, _ := meta["hint_applied"].(bool)
	return applied
}

func stage2HintMismatchReasonFromMeta(meta map[string]any) string {
	if len(meta) == 0 {
		return ""
	}
	if reason, ok := meta["hint_mismatch_reason"].(string); ok && strings.TrimSpace(reason) != "" {
		return strings.TrimSpace(reason)
	}
	return ""
}

func memoryScopeSelectedFromMeta(meta map[string]any) string {
	if len(meta) == 0 {
		return ""
	}
	if scope, ok := meta["memory_scope_selected"].(string); ok && strings.TrimSpace(scope) != "" {
		return strings.TrimSpace(scope)
	}
	return ""
}

func memoryBudgetUsedFromMeta(meta map[string]any) int {
	if len(meta) == 0 {
		return 0
	}
	switch raw := meta["memory_budget_used"].(type) {
	case int:
		if raw > 0 {
			return raw
		}
	case int64:
		if raw > 0 {
			return int(raw)
		}
	case float64:
		if raw > 0 {
			return int(raw)
		}
	}
	return 0
}

func memoryHitsFromMeta(meta map[string]any, fallback int) int {
	if len(meta) == 0 {
		if fallback > 0 {
			return fallback
		}
		return 0
	}
	switch raw := meta["memory_hits"].(type) {
	case int:
		if raw >= 0 {
			return raw
		}
	case int64:
		if raw >= 0 {
			return int(raw)
		}
	case float64:
		if raw >= 0 {
			return int(raw)
		}
	}
	if fallback > 0 {
		return fallback
	}
	return 0
}

func memoryRerankStatsFromMeta(meta map[string]any) map[string]int {
	if len(meta) == 0 {
		return nil
	}
	raw, ok := meta["memory_rerank_stats"]
	if !ok {
		return nil
	}
	switch typed := raw.(type) {
	case map[string]int:
		if len(typed) == 0 {
			return nil
		}
		out := make(map[string]int, len(typed))
		for key, value := range typed {
			out[strings.TrimSpace(key)] = value
		}
		return out
	case map[string]any:
		if len(typed) == 0 {
			return nil
		}
		out := make(map[string]int, len(typed))
		for key, value := range typed {
			normalizedKey := strings.TrimSpace(key)
			if normalizedKey == "" {
				continue
			}
			switch tv := value.(type) {
			case int:
				out[normalizedKey] = tv
			case int64:
				out[normalizedKey] = int(tv)
			case float64:
				out[normalizedKey] = int(tv)
			}
		}
		if len(out) == 0 {
			return nil
		}
		return out
	default:
		return nil
	}
}

func memoryLifecycleActionFromMeta(meta map[string]any) string {
	if len(meta) == 0 {
		return ""
	}
	if action, ok := meta["memory_lifecycle_action"].(string); ok && strings.TrimSpace(action) != "" {
		return strings.TrimSpace(action)
	}
	return ""
}

func stage2ProfileFromConfig(profile, providerName string) string {
	out := strings.TrimSpace(profile)
	if out != "" {
		return out
	}
	if strings.EqualFold(strings.TrimSpace(providerName), runtimeconfig.ContextStage2ProviderFile) {
		return "file"
	}
	return runtimeconfig.ContextStage2ExternalProfileHTTPGeneric
}

func stage2TemplateProfileFromConfig(profile, providerName string) string {
	out := strings.TrimSpace(profile)
	if out != "" {
		return out
	}
	if strings.EqualFold(strings.TrimSpace(providerName), runtimeconfig.ContextStage2ProviderFile) {
		return "file"
	}
	return runtimeconfig.ContextStage2ExternalProfileHTTPGeneric
}

func stage2TemplateResolutionSourceFromConfig(source, profile, providerName string) string {
	resolved := strings.TrimSpace(source)
	if resolved != "" {
		return resolved
	}
	if strings.EqualFold(strings.TrimSpace(providerName), runtimeconfig.ContextStage2ProviderFile) {
		return runtimeconfig.Stage2TemplateResolutionExplicitOnly
	}
	p := strings.ToLower(strings.TrimSpace(profile))
	if p == "" || p == runtimeconfig.ContextStage2ExternalProfileExplicitOnly {
		return runtimeconfig.Stage2TemplateResolutionExplicitOnly
	}
	return runtimeconfig.Stage2TemplateResolutionProfileDefaultsOnly
}

func stage2HintsFromConfig(cfg runtimeconfig.ContextAssemblerCA2ExternalHintConfig) provider.CapabilityHints {
	if !cfg.Enabled || len(cfg.Capabilities) == 0 {
		return provider.CapabilityHints{}
	}
	out := make([]string, 0, len(cfg.Capabilities))
	seen := map[string]struct{}{}
	for _, capability := range cfg.Capabilities {
		item := strings.ToLower(strings.TrimSpace(capability))
		if item == "" {
			continue
		}
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		out = append(out, item)
	}
	return provider.CapabilityHints{Capabilities: out}
}

func stage2ReasonFromError(err error) (reason string, reasonCode string, errorLayer string) {
	var fetchErr *provider.FetchError
	if errors.As(err, &fetchErr) {
		code := strings.TrimSpace(fetchErr.Code)
		layer := strings.TrimSpace(string(fetchErr.Layer))
		if code == "" {
			code = "fetch_error"
		}
		if layer == "" {
			layer = "protocol"
		}
		return reasonFromCode(code), code, layer
	}

	msg := strings.ToLower(strings.TrimSpace(err.Error()))
	switch {
	case strings.Contains(msg, "timeout"), strings.Contains(msg, "deadline"):
		return "timeout", "timeout", "transport"
	case strings.Contains(msg, "auth"), strings.Contains(msg, "unauthorized"), strings.Contains(msg, "forbidden"):
		return "auth", "auth_failed", "semantic"
	case strings.Contains(msg, "mapping"), strings.Contains(msg, "decode"), strings.Contains(msg, "marshal"):
		return "mapping", "mapping_invalid", "protocol"
	default:
		return "fetch_error", "fetch_error", "protocol"
	}
}

func reasonFromCode(code string) string {
	switch strings.ToLower(strings.TrimSpace(code)) {
	case "timeout":
		return "timeout"
	case "auth_failed":
		return "auth"
	case "request_encode_failed", "request_build_failed", "response_decode_failed":
		return "mapping"
	default:
		return "fetch_error"
	}
}

func shouldRunStage2(req types.ModelRequest, cfg runtimeconfig.ContextAssemblerCA2RoutingConfig) (bool, string) {
	trimmed := strings.TrimSpace(req.Input)
	if cfg.MinInputChars > 0 && len([]rune(trimmed)) >= cfg.MinInputChars {
		return true, ""
	}
	if len(cfg.TriggerKeywords) > 0 {
		lowerInput := strings.ToLower(trimmed)
		for _, kw := range cfg.TriggerKeywords {
			if strings.Contains(lowerInput, strings.ToLower(strings.TrimSpace(kw))) {
				return true, ""
			}
		}
	}
	if cfg.RequireSystemGuard {
		for _, msg := range req.Messages {
			if strings.EqualFold(strings.TrimSpace(msg.Role), "system") {
				return false, "routing.system_guard.present"
			}
		}
		return true, ""
	}
	return false, "routing.threshold.not_met"
}

func (a *Assembler) appendTailRecap(
	modelReq types.ModelRequest,
	cfg runtimeconfig.ContextAssemblerCA2Config,
	outcome *types.ContextAssembleResult,
) (types.ModelRequest, types.RecapMetadata) {
	if outcome == nil {
		return modelReq, types.RecapMetadata{Status: types.RecapStatusDisabled}
	}
	if !cfg.TailRecap.Enabled {
		outcome.Stage.ContextRecapSource = ""
		return modelReq, types.RecapMetadata{Status: types.RecapStatusDisabled}
	}
	maxItems := cfg.TailRecap.MaxItems
	if maxItems <= 0 {
		maxItems = 4
	}
	maxChars := cfg.TailRecap.MaxFieldChars
	if maxChars <= 0 {
		maxChars = 256
	}
	recap, recapSource := buildTaskAwareTailRecap(cfg, outcome.Stage)
	outcome.Stage.ContextRecapSource = recapSource
	recap.Decisions = cropItems(recap.Decisions, maxItems)
	recap.Todo = cropItems(recap.Todo, maxItems)
	recap.Risks = cropItems(recap.Risks, maxItems)
	recap.Status = truncateField(recap.Status, maxChars)
	recap.Decisions = truncateSlice(recap.Decisions, maxChars)
	recap.Todo = truncateSlice(recap.Todo, maxChars)
	recap.Risks = truncateSlice(recap.Risks, maxChars)
	recap = a.sanitizeRecap(recap)

	raw, _ := json.Marshal(recap)
	modelReq.Messages = append(modelReq.Messages, types.Message{
		Role:    "system",
		Content: "tail_recap:" + string(raw),
	})
	meta := types.RecapMetadata{
		Status: types.RecapStatusAppended,
		Tail:   recap,
	}
	if isAnyTruncated(recap, maxChars) {
		meta.Status = types.RecapStatusTruncated
	}
	return modelReq, meta
}

func buildTaskAwareTailRecap(cfg runtimeconfig.ContextAssemblerCA2Config, stage types.AssembleStage) (types.TailRecap, string) {
	status := string(stage.Status)
	routerMode := strings.TrimSpace(stage.Stage2RouterMode)
	if routerMode == "" {
		routerMode = normalizedStage2RouterMode(cfg.RoutingMode)
	}
	decisions := make([]string, 0, 12)
	if status != "" {
		decisions = append(decisions, "stage_status="+status)
	}
	if routerMode != "" {
		decisions = append(decisions, "stage2_router_mode="+routerMode)
	}
	if stage.Stage2RouterDecision != "" {
		decisions = append(decisions, "stage2_router_decision="+stage.Stage2RouterDecision)
	}
	if stage.Stage2Provider != "" {
		decisions = append(decisions, "stage2_provider="+stage.Stage2Provider)
	}
	if stage.Stage2ReasonCode != "" {
		decisions = append(decisions, "stage2_reason_code="+stage.Stage2ReasonCode)
	}
	if stage.Stage2SkipReason != "" {
		decisions = append(decisions, "stage2_skip_reason="+stage.Stage2SkipReason)
	}
	if stage.ContextRefDiscoverCount > 0 || stage.ContextRefResolveCount > 0 {
		decisions = append(decisions, fmt.Sprintf("reference_first.discover=%d", stage.ContextRefDiscoverCount))
		decisions = append(decisions, fmt.Sprintf("reference_first.resolve=%d", stage.ContextRefResolveCount))
	}
	if stage.ContextEditGateDecision != "" {
		decisions = append(decisions, "edit_gate_decision="+stage.ContextEditGateDecision)
	}
	if stage.ContextSwapbackRelevanceScore > 0 {
		decisions = append(decisions, fmt.Sprintf("swapback_relevance_score=%.4f", stage.ContextSwapbackRelevanceScore))
	}
	if tierStats := stableTierStatsSummary(stage.ContextLifecycleTierStats); tierStats != "" {
		decisions = append(decisions, "lifecycle_tiering="+tierStats)
	}

	todo := make([]string, 0, 4)
	if stage.Stage2SkipReason != "" {
		todo = append(todo, "review_stage2_skip_reason="+stage.Stage2SkipReason)
	}
	if missingRefs := stage.ContextRefDiscoverCount - stage.ContextRefResolveCount; missingRefs > 0 {
		todo = append(todo, fmt.Sprintf("resolve_missing_refs=%d", missingRefs))
	}
	if strings.HasPrefix(strings.ToLower(strings.TrimSpace(stage.ContextEditGateDecision)), "deny.") {
		todo = append(todo, "tune_edit_gate_thresholds")
	}
	if stage.Status == types.AssembleStageStatusDegraded {
		todo = append(todo, "inspect_stage2_degraded_path")
	}

	risks := make([]string, 0, 4)
	if stage.Status == types.AssembleStageStatusDegraded {
		risks = append(risks, "stage2_degraded")
	}
	if stage.Stage2ErrorLayer != "" {
		risks = append(risks, "stage2_error_layer="+stage.Stage2ErrorLayer)
	}
	if strings.EqualFold(strings.TrimSpace(stage.Stage2ReasonCode), "partial_missing_refs") {
		risks = append(risks, "reference_resolution_partial")
	}
	if strings.EqualFold(strings.TrimSpace(stage.ContextEditGateDecision), contextEditGateDecisionDenyConfig) {
		risks = append(risks, "edit_gate_config_conflict")
	}
	if len(risks) == 0 {
		risks = append(risks, "none")
	}

	return types.TailRecap{
		Status:    status,
		Decisions: decisions,
		Todo:      todo,
		Risks:     risks,
	}, contextRecapSourceTaskAwareV1
}

func stableTierStatsSummary(stats map[string]int) string {
	if len(stats) == 0 {
		return ""
	}
	ordered := []string{
		"hot",
		"warm",
		"cold",
		"pruned",
		"migrate_hot_to_warm",
		"migrate_warm_to_cold",
		"migrate_cold_to_pruned",
	}
	parts := make([]string, 0, len(stats))
	seen := map[string]struct{}{}
	for _, key := range ordered {
		value, ok := stats[key]
		if !ok {
			continue
		}
		parts = append(parts, fmt.Sprintf("%s=%d", key, value))
		seen[key] = struct{}{}
	}
	extraKeys := make([]string, 0, len(stats))
	for key := range stats {
		if _, ok := seen[key]; ok {
			continue
		}
		extraKeys = append(extraKeys, key)
	}
	sort.Strings(extraKeys)
	for _, key := range extraKeys {
		parts = append(parts, fmt.Sprintf("%s=%d", key, stats[key]))
	}
	return strings.Join(parts, ",")
}

func (a *Assembler) sanitizeStage2Chunks(chunks []string) []string {
	if len(chunks) == 0 {
		return chunks
	}
	out := make([]string, 0, len(chunks))
	rd := a.newRedactor()
	for _, chunk := range chunks {
		out = append(out, rd.SanitizeJSONText(chunk))
	}
	return out
}

func (a *Assembler) sanitizeRecap(recap types.TailRecap) types.TailRecap {
	raw, err := json.Marshal(recap)
	if err != nil {
		return recap
	}
	payload := map[string]any{}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return recap
	}
	sanitized := a.newRedactor().SanitizeMap(payload)
	nextRaw, err := json.Marshal(sanitized)
	if err != nil {
		return recap
	}
	var out types.TailRecap
	if err := json.Unmarshal(nextRaw, &out); err != nil {
		return recap
	}
	return out
}

func (a *Assembler) newRedactor() *redaction.Redactor {
	cfg := a.redactionCfgProvider()
	return redaction.New(cfg.Enabled, cfg.Keywords)
}

func cropItems(in []string, max int) []string {
	if len(in) <= max {
		return in
	}
	return in[:max]
}

func truncateField(v string, max int) string {
	if max <= 0 || len([]rune(v)) <= max {
		return v
	}
	return string([]rune(v)[:max])
}

func truncateSlice(in []string, max int) []string {
	out := make([]string, 0, len(in))
	for _, v := range in {
		out = append(out, truncateField(v, max))
	}
	return out
}

func isAnyTruncated(recap types.TailRecap, max int) bool {
	if max <= 0 {
		return false
	}
	for _, v := range append(append(recap.Decisions, recap.Todo...), recap.Risks...) {
		if len([]rune(v)) >= max {
			return true
		}
	}
	return len([]rune(recap.Status)) >= max
}

func isBestEffortPolicy(policy string) bool {
	return strings.EqualFold(strings.TrimSpace(policy), "best_effort")
}
