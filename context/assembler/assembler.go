package assembler

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
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

var ErrAgenticRoutingNotReady = errors.New("context stage routing agentic mode not ready")

type Assembler struct {
	cfgProvider          func() runtimeconfig.ContextAssemblerConfig
	redactionCfgProvider func() runtimeconfig.SecurityRedactionConfig
	now                  func() time.Time

	mu              sync.Mutex
	storageKey      string
	storage         journal.Storage
	stage2Key       string
	stage2Provider  provider.Provider
	prefixCache     map[string]string
	ca3State        map[string]*ca3RunState
	spillBackend    SpillBackend
	spillBackendKey string
	embeddingScorer SemanticEmbeddingScorer
	embeddingKey    string
	rerankers       map[string]SemanticReranker
	defaultReranker SemanticReranker
}

type Option func(*Assembler)

func WithRedactionConfigProvider(provider func() runtimeconfig.SecurityRedactionConfig) Option {
	return func(a *Assembler) {
		if provider != nil {
			a.redactionCfgProvider = provider
		}
	}
}

func WithSemanticEmbeddingScorer(key string, scorer SemanticEmbeddingScorer) Option {
	return func(a *Assembler) {
		a.embeddingKey = strings.TrimSpace(key)
		a.embeddingScorer = scorer
	}
}

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

func New(cfgProvider func() runtimeconfig.ContextAssemblerConfig, opts ...Option) *Assembler {
	baseSecurity := runtimeconfig.DefaultConfig().Security.Redaction
	a := &Assembler{
		cfgProvider: cfgProvider,
		redactionCfgProvider: func() runtimeconfig.SecurityRedactionConfig {
			return baseSecurity
		},
		now:             time.Now,
		prefixCache:     map[string]string{},
		ca3State:        map[string]*ca3RunState{},
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

func (a *Assembler) Assemble(ctx context.Context, req types.ContextAssembleRequest, modelReq types.ModelRequest) (types.ModelRequest, types.ContextAssembleResult, error) {
	start := a.now()
	cfg := a.cfgProvider()
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
		if !isBestEffortPolicy(cfg.CA2.StagePolicy.Stage1) {
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

	var ca3Decision ca3Decision
	if cfg.CA3.Enabled {
		updatedReq, updatedOutcome, decision, err := a.applyCA3(ctx, req, modelReq, outcome, cfg, "stage1")
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
		ca3Decision = decision
	}

	if cfg.CA2.Enabled {
		modelReq, outcome, err = a.applyCA2(ctx, modelReq, req, cfg, outcome, ca3Decision)
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
	if cfg.CA3.Enabled {
		modelReq, outcome, _, err = a.applyCA3(ctx, req, modelReq, outcome, cfg, "stage2")
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
		Recap: types.RecapMetadata{
			Status: types.RecapStatusFailed,
		},
	}
}

func (a *Assembler) applyCA2(
	ctx context.Context,
	modelReq types.ModelRequest,
	req types.ContextAssembleRequest,
	cfg runtimeconfig.ContextAssemblerConfig,
	outcome types.ContextAssembleResult,
	ca3 ca3Decision,
) (types.ModelRequest, types.ContextAssembleResult, error) {
	if strings.EqualFold(strings.TrimSpace(cfg.CA2.RoutingMode), "agentic") {
		// TODO(ca2): plug in agentic decision provider once the dedicated milestone lands.
		return modelReq, outcome, ErrAgenticRoutingNotReady
	}

	shouldStage2, skipReason := shouldRunStage2(modelReq, cfg.CA2.Routing)
	if !shouldStage2 {
		outcome.Stage.Status = types.AssembleStageStatusStage1Only
		outcome.Stage.Stage2SkipReason = skipReason
		modelReq, recap := a.appendTailRecap(modelReq, cfg.CA2, outcome)
		outcome.Recap = recap
		return modelReq, outcome, nil
	}
	if ca3.BlockLowPriorityLoads && !isHighPriorityRequest(modelReq.Input, cfg.CA3.Emergency.HighPriorityTokens) {
		outcome.Stage.Status = types.AssembleStageStatusDegraded
		outcome.Stage.Stage2SkipReason = "ca3.emergency.low_priority_rejected"
		modelReq, recap := a.appendTailRecap(modelReq, cfg.CA2, outcome)
		outcome.Recap = recap
		return modelReq, outcome, nil
	}

	stage2Start := a.now()
	p, err := a.ensureStage2Provider(cfg)
	if err != nil {
		if isBestEffortPolicy(cfg.CA2.StagePolicy.Stage2) {
			outcome.Stage.Status = types.AssembleStageStatusDegraded
			outcome.Stage.Stage2SkipReason = "stage2.provider.not_ready"
			modelReq, recap := a.appendTailRecap(modelReq, cfg.CA2, outcome)
			outcome.Recap = recap
			return modelReq, outcome, nil
		}
		return modelReq, outcome, err
	}
	outcome.Stage.Stage2Provider = p.Name()
	outcome.Stage.Stage2Profile = stage2ProfileFromConfig(cfg.CA2.Stage2.External.Profile, p.Name())
	outcome.Stage.Stage2TemplateProfile = stage2TemplateProfileFromConfig(cfg.CA2.Stage2.External.Profile, p.Name())
	outcome.Stage.Stage2TemplateResolutionSource = stage2TemplateResolutionSourceFromConfig(
		cfg.CA2.Stage2.External.TemplateResolutionSource,
		cfg.CA2.Stage2.External.Profile,
		p.Name(),
	)
	stage2Ctx, cancel := context.WithTimeout(ctx, cfg.CA2.Timeout.Stage2)
	defer cancel()
	resp, err := p.Fetch(stage2Ctx, provider.Request{
		RunID:     req.RunID,
		SessionID: req.SessionID,
		Input:     modelReq.Input,
		MaxItems:  cfg.CA2.TailRecap.MaxItems,
		Hints:     stage2HintsFromConfig(cfg.CA2.Stage2.External.Hints),
	})
	outcome.Stage.Stage2LatencyMs = a.now().Sub(stage2Start).Milliseconds()
	if err != nil {
		reason, reasonCode, errorLayer := stage2ReasonFromError(err)
		outcome.Stage.Stage2Reason = reason
		outcome.Stage.Stage2ReasonCode = reasonCode
		outcome.Stage.Stage2ErrorLayer = errorLayer
		if isBestEffortPolicy(cfg.CA2.StagePolicy.Stage2) {
			outcome.Stage.Status = types.AssembleStageStatusDegraded
			outcome.Stage.Stage2SkipReason = "stage2.fetch.failed"
			modelReq, recap := a.appendTailRecap(modelReq, cfg.CA2, outcome)
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
		modelReq, recap := a.appendTailRecap(modelReq, cfg.CA2, outcome)
		outcome.Recap = recap
		return modelReq, outcome, nil
	}
	resp.Chunks = a.sanitizeStage2Chunks(resp.Chunks)
	modelReq.Messages = append(modelReq.Messages, types.Message{
		Role:    "system",
		Content: "stage2_context:\n" + strings.Join(resp.Chunks, "\n"),
	})
	outcome.Stage.Status = types.AssembleStageStatusStage2Used
	outcome.Stage.Stage2SkipReason = ""
	outcome.Stage.Stage2HitCount = len(resp.Chunks)
	outcome.Stage.Stage2Source = sourceFromMeta(resp.Meta, p.Name())
	outcome.Stage.Stage2Reason = stage2ReasonFromMeta(resp.Meta, "ok")
	outcome.Stage.Stage2ReasonCode = stage2ReasonCodeFromMeta(resp.Meta, "ok")
	outcome.Stage.Stage2ErrorLayer = stage2ErrorLayerFromMeta(resp.Meta, "")
	outcome.Stage.Stage2Profile = stage2ProfileFromMeta(resp.Meta, outcome.Stage.Stage2Profile)
	outcome.Stage.Stage2TemplateProfile = stage2TemplateProfileFromMeta(resp.Meta, outcome.Stage.Stage2TemplateProfile)
	outcome.Stage.Stage2TemplateResolutionSource = stage2TemplateResolutionSourceFromMeta(
		resp.Meta,
		outcome.Stage.Stage2TemplateResolutionSource,
	)
	outcome.Stage.Stage2HintApplied = stage2HintAppliedFromMeta(resp.Meta)
	outcome.Stage.Stage2HintMismatchReason = stage2HintMismatchReasonFromMeta(resp.Meta)
	modelReq, recap := a.appendTailRecap(modelReq, cfg.CA2, outcome)
	outcome.Recap = recap
	return modelReq, outcome, nil
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

func (a *Assembler) appendTailRecap(modelReq types.ModelRequest, cfg runtimeconfig.ContextAssemblerCA2Config, outcome types.ContextAssembleResult) (types.ModelRequest, types.RecapMetadata) {
	if !cfg.TailRecap.Enabled {
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
	recap := types.TailRecap{
		Status:    string(outcome.Stage.Status),
		Decisions: cropItems([]string{"routing_mode=" + cfg.RoutingMode, "stage2_provider=" + cfg.Stage2.Provider}, maxItems),
		Todo:      cropItems([]string{"review_stage2_quality", "evaluate_agentic_routing_todo"}, maxItems),
		Risks:     cropItems([]string{riskForStage(outcome.Stage.Status)}, maxItems),
	}
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

func riskForStage(s types.AssembleStageStatus) string {
	if s == types.AssembleStageStatusDegraded {
		return "stage2_degraded"
	}
	return "none"
}

func isBestEffortPolicy(policy string) bool {
	return strings.EqualFold(strings.TrimSpace(policy), "best_effort")
}
