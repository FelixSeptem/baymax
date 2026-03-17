package assembler

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/FelixSeptem/baymax/core/types"
	runtimeconfig "github.com/FelixSeptem/baymax/runtime/config"
	tiktoken "github.com/pkoukk/tiktoken-go"
)

type ca3Zone string

const (
	ca3ZoneSafe      ca3Zone = "safe"
	ca3ZoneComfort   ca3Zone = "comfort"
	ca3ZoneWarning   ca3Zone = "warning"
	ca3ZoneDanger    ca3Zone = "danger"
	ca3ZoneEmergency ca3Zone = "emergency"
)

type ca3Decision struct {
	Zone                  ca3Zone
	BlockLowPriorityLoads bool
}

type ca3RunState struct {
	CurrentZone        ca3Zone
	ZoneEnteredAt      time.Time
	ZoneResidencyMs    map[string]int64
	TriggerCounts      map[string]int
	AccessFrequency    map[string]int
	SpilledByRun       map[string]spillRecord
	SpillWrites        map[string]struct{}
	LastTokenEstimate  int
	LastTokenSignature string
	LastSDKCountAt     time.Time
}

type spillRecord struct {
	RunID     string    `json:"run_id"`
	SessionID string    `json:"session_id,omitempty"`
	Stage     string    `json:"stage"`
	OriginRef string    `json:"origin_ref"`
	Content   string    `json:"content"`
	SpilledAt time.Time `json:"spilled_at"`
}

type SpillBackend interface {
	Append(ctx context.Context, rec spillRecord) error
	LoadByRun(ctx context.Context, runID string, limit int) ([]spillRecord, error)
}

// DBSpillBackend is a placeholder SPI for future DB implementations.
type DBSpillBackend interface {
	SpillBackend
}

// ObjectSpillBackend is a placeholder SPI for future object-storage implementations.
type ObjectSpillBackend interface {
	SpillBackend
}

type fileSpillBackend struct {
	path string
}

func newFileSpillBackend(path string) *fileSpillBackend {
	return &fileSpillBackend{path: strings.TrimSpace(path)}
}

func (f *fileSpillBackend) Append(_ context.Context, rec spillRecord) error {
	if strings.TrimSpace(f.path) == "" {
		return fmt.Errorf("ca3 spill path is required")
	}
	if err := os.MkdirAll(filepath.Dir(f.path), 0o755); err != nil {
		return fmt.Errorf("create spill dir: %w", err)
	}
	row, err := json.Marshal(rec)
	if err != nil {
		return fmt.Errorf("marshal spill record: %w", err)
	}
	fd, err := os.OpenFile(f.path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return fmt.Errorf("open spill file: %w", err)
	}
	defer func() { _ = fd.Close() }()
	if _, err := fd.Write(append(row, '\n')); err != nil {
		return fmt.Errorf("write spill file: %w", err)
	}
	return nil
}

func (f *fileSpillBackend) LoadByRun(_ context.Context, runID string, limit int) ([]spillRecord, error) {
	if strings.TrimSpace(f.path) == "" {
		return nil, nil
	}
	raw, err := os.ReadFile(f.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read spill file: %w", err)
	}
	lines := strings.Split(strings.TrimSpace(string(raw)), "\n")
	if len(lines) == 1 && strings.TrimSpace(lines[0]) == "" {
		return nil, nil
	}
	out := make([]spillRecord, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var rec spillRecord
		if err := json.Unmarshal([]byte(line), &rec); err != nil {
			continue
		}
		if strings.TrimSpace(rec.RunID) != strings.TrimSpace(runID) {
			continue
		}
		out = append(out, rec)
	}
	if limit <= 0 || len(out) <= limit {
		return out, nil
	}
	return out[:limit], nil
}

func (a *Assembler) applyCA3(
	ctx context.Context,
	req types.ContextAssembleRequest,
	modelReq types.ModelRequest,
	outcome types.ContextAssembleResult,
	cfg runtimeconfig.ContextAssemblerConfig,
	stage string,
) (types.ModelRequest, types.ContextAssembleResult, ca3Decision, error) {
	decision := ca3Decision{Zone: ca3ZoneSafe}
	state := a.ca3StateFor(req.RunID)
	now := a.now()
	compactionMode := normalizeCompactionMode(cfg.CA3.Compaction.Mode)
	outcome.Stage.CompactionMode = compactionMode

	swapBackCount, err := a.swapBackIfNeeded(ctx, req, &modelReq, cfg.CA3, state)
	if err != nil {
		return modelReq, outcome, decision, err
	}

	usageTokens := a.countContextTokens(ctx, req, modelReq, cfg.CA3, state)
	thresholdsPercent, thresholdsTokens := resolveCA3Thresholds(cfg.CA3, stage)
	usagePercent := 0
	if cfg.CA3.MaxContextTokens > 0 {
		usagePercent = (usageTokens * 100) / cfg.CA3.MaxContextTokens
	}
	zone, reason, triggerReason := evaluateCA3Zone(usagePercent, usageTokens, thresholdsPercent, thresholdsTokens)
	decision.Zone = zone
	decision.BlockLowPriorityLoads = zone == ca3ZoneEmergency && cfg.CA3.Emergency.RejectLowPriority

	updateZoneState(state, zone, triggerReason, now)
	beforeTokens := usageTokens
	actionsCompression := 0.0
	spillCount := 0
	removed := make([]spillRecord, 0)
	fallbackUsed := false
	fallbackReason := ""
	qualityScore := 0.0
	qualityReason := ""
	embeddingProvider := ""
	embeddingSimilarity := 0.0
	embeddingContribution := 0.0
	embeddingStatus := ""
	embeddingFallbackReason := ""
	rerankerUsed := false
	rerankerProvider := ""
	rerankerModel := ""
	rerankerThresholdSource := ""
	rerankerThresholdHit := false
	rerankerFallbackReason := ""
	rerankerProfileVersion := ""
	rerankerRolloutHit := false
	rerankerThresholdDrift := 0.0
	retainedEvidenceCount := 0

	switch zone {
	case ca3ZoneWarning:
		if cfg.CA3.Squash.Enabled {
			var compaction ca3CompactionResult
			compaction, err = a.applyCompaction(ctx, req, modelReq, cfg, state, stage)
			if err != nil {
				return modelReq, outcome, decision, err
			}
			modelReq.Messages = compaction.Messages
			actionsCompression = compaction.CompressionRatio
			fallbackUsed = compaction.Fallback
			fallbackReason = compaction.FallbackReason
			qualityScore = compaction.QualityScore
			qualityReason = compaction.QualityReason
			embeddingProvider = compaction.EmbeddingProvider
			embeddingSimilarity = compaction.EmbeddingSimilarity
			embeddingContribution = compaction.EmbeddingContribution
			embeddingStatus = compaction.EmbeddingStatus
			embeddingFallbackReason = compaction.EmbeddingFallbackReason
			rerankerUsed = compaction.RerankerUsed
			rerankerProvider = compaction.RerankerProvider
			rerankerModel = compaction.RerankerModel
			rerankerThresholdSource = compaction.RerankerThresholdSource
			rerankerThresholdHit = compaction.RerankerThresholdHit
			rerankerFallbackReason = compaction.RerankerFallbackReason
			rerankerProfileVersion = compaction.RerankerProfileVersion
			rerankerRolloutHit = compaction.RerankerRolloutHit
			rerankerThresholdDrift = compaction.RerankerThresholdDrift
		}
	case ca3ZoneDanger:
		if cfg.CA3.Squash.Enabled {
			var compaction ca3CompactionResult
			compaction, err = a.applyCompaction(ctx, req, modelReq, cfg, state, stage)
			if err != nil {
				return modelReq, outcome, decision, err
			}
			modelReq.Messages = compaction.Messages
			actionsCompression = compaction.CompressionRatio
			fallbackUsed = compaction.Fallback
			fallbackReason = compaction.FallbackReason
			qualityScore = compaction.QualityScore
			qualityReason = compaction.QualityReason
			embeddingProvider = compaction.EmbeddingProvider
			embeddingSimilarity = compaction.EmbeddingSimilarity
			embeddingContribution = compaction.EmbeddingContribution
			embeddingStatus = compaction.EmbeddingStatus
			embeddingFallbackReason = compaction.EmbeddingFallbackReason
			rerankerUsed = compaction.RerankerUsed
			rerankerProvider = compaction.RerankerProvider
			rerankerModel = compaction.RerankerModel
			rerankerThresholdSource = compaction.RerankerThresholdSource
			rerankerThresholdHit = compaction.RerankerThresholdHit
			rerankerFallbackReason = compaction.RerankerFallbackReason
			rerankerProfileVersion = compaction.RerankerProfileVersion
			rerankerRolloutHit = compaction.RerankerRolloutHit
			rerankerThresholdDrift = compaction.RerankerThresholdDrift
		}
		if cfg.CA3.Prune.Enabled {
			var pruned []spillRecord
			modelReq.Messages, pruned, retainedEvidenceCount = pruneMessages(modelReq.Messages, cfg.CA3, state)
			_ = pruned
		}
	case ca3ZoneEmergency:
		if cfg.CA3.Squash.Enabled {
			var compaction ca3CompactionResult
			compaction, err = a.applyCompaction(ctx, req, modelReq, cfg, state, stage)
			if err != nil {
				return modelReq, outcome, decision, err
			}
			modelReq.Messages = compaction.Messages
			actionsCompression = compaction.CompressionRatio
			fallbackUsed = compaction.Fallback
			fallbackReason = compaction.FallbackReason
			qualityScore = compaction.QualityScore
			qualityReason = compaction.QualityReason
			embeddingProvider = compaction.EmbeddingProvider
			embeddingSimilarity = compaction.EmbeddingSimilarity
			embeddingContribution = compaction.EmbeddingContribution
			embeddingStatus = compaction.EmbeddingStatus
			embeddingFallbackReason = compaction.EmbeddingFallbackReason
			rerankerUsed = compaction.RerankerUsed
			rerankerProvider = compaction.RerankerProvider
			rerankerModel = compaction.RerankerModel
			rerankerThresholdSource = compaction.RerankerThresholdSource
			rerankerThresholdHit = compaction.RerankerThresholdHit
			rerankerFallbackReason = compaction.RerankerFallbackReason
			rerankerProfileVersion = compaction.RerankerProfileVersion
			rerankerRolloutHit = compaction.RerankerRolloutHit
			rerankerThresholdDrift = compaction.RerankerThresholdDrift
		}
		if cfg.CA3.Prune.Enabled {
			modelReq.Messages, removed, retainedEvidenceCount = pruneMessages(modelReq.Messages, cfg.CA3, state)
		}
		if cfg.CA3.Spill.Enabled {
			spillCount, err = a.spillRecords(ctx, req, stage, removed, cfg.CA3, state)
			if err != nil {
				return modelReq, outcome, decision, err
			}
		}
	}

	afterTokens := estimateContextTokens(modelReq)
	compressionRatio := actionsCompression
	if beforeTokens > 0 && compressionRatio == 0 {
		compressionRatio = float64(beforeTokens-afterTokens) / float64(beforeTokens)
		if compressionRatio < 0 {
			compressionRatio = 0
		}
	}

	outcome.Stage.PressureZone = string(zone)
	outcome.Stage.PressureReason = reason
	outcome.Stage.PressureTriggerSource = triggerReason
	outcome.Stage.ZoneResidencyMs = cloneInt64Map(state.ZoneResidencyMs)
	outcome.Stage.TriggerCounts = cloneIntMap(state.TriggerCounts)
	outcome.Stage.CompressionRatio = compressionRatio
	outcome.Stage.SpillCount += spillCount
	outcome.Stage.SwapBackCount += swapBackCount
	outcome.Stage.CompactionFallback = outcome.Stage.CompactionFallback || fallbackUsed
	if strings.TrimSpace(fallbackReason) != "" {
		outcome.Stage.CompactionFallbackReason = fallbackReason
	}
	if qualityScore > 0 {
		outcome.Stage.CompactionQualityScore = qualityScore
	}
	if strings.TrimSpace(qualityReason) != "" {
		outcome.Stage.CompactionQualityReason = qualityReason
	}
	if strings.TrimSpace(embeddingProvider) != "" {
		outcome.Stage.CompactionEmbeddingProvider = embeddingProvider
	}
	if embeddingSimilarity > 0 {
		outcome.Stage.CompactionEmbeddingSimilarity = embeddingSimilarity
	}
	if embeddingContribution > 0 {
		outcome.Stage.CompactionEmbeddingContribution = embeddingContribution
	}
	if strings.TrimSpace(embeddingStatus) != "" {
		outcome.Stage.CompactionEmbeddingStatus = embeddingStatus
	}
	if strings.TrimSpace(embeddingFallbackReason) != "" {
		outcome.Stage.CompactionEmbeddingFallbackReason = embeddingFallbackReason
	}
	outcome.Stage.CompactionRerankerUsed = rerankerUsed
	if strings.TrimSpace(rerankerProvider) != "" {
		outcome.Stage.CompactionRerankerProvider = rerankerProvider
	}
	if strings.TrimSpace(rerankerModel) != "" {
		outcome.Stage.CompactionRerankerModel = rerankerModel
	}
	if strings.TrimSpace(rerankerThresholdSource) != "" {
		outcome.Stage.CompactionRerankerThresholdSource = rerankerThresholdSource
	}
	outcome.Stage.CompactionRerankerThresholdHit = rerankerThresholdHit
	if strings.TrimSpace(rerankerFallbackReason) != "" {
		outcome.Stage.CompactionRerankerFallbackReason = rerankerFallbackReason
	}
	if strings.TrimSpace(rerankerProfileVersion) != "" {
		outcome.Stage.CompactionRerankerProfileVersion = rerankerProfileVersion
	}
	outcome.Stage.CompactionRerankerRolloutHit = rerankerRolloutHit
	if rerankerThresholdDrift > 0 {
		outcome.Stage.CompactionRerankerThresholdDrift = rerankerThresholdDrift
	}
	if retainedEvidenceCount > 0 {
		outcome.Stage.RetainedEvidenceCount += retainedEvidenceCount
	}
	return modelReq, outcome, decision, nil
}

func (a *Assembler) applyCompaction(
	ctx context.Context,
	assembleReq types.ContextAssembleRequest,
	modelReq types.ModelRequest,
	cfg runtimeconfig.ContextAssemblerConfig,
	state *ca3RunState,
	stage string,
) (ca3CompactionResult, error) {
	request := ca3CompactionRequest{
		AssembleReq: assembleReq,
		ModelReq:    modelReq,
		Config:      cfg.CA3,
		StagePolicy: ca3StagePolicy(cfg, stage),
	}
	mode := normalizeCompactionMode(cfg.CA3.Compaction.Mode)
	primary, err := a.compactorFor(mode, assembleReq, cfg.CA3.Compaction.Embedding, cfg.CA3.Compaction.Reranker)
	if err != nil {
		return ca3CompactionResult{}, err
	}
	semanticCtx := ctx
	var cancel context.CancelFunc
	if mode == "semantic" {
		semanticCtx, cancel = context.WithTimeout(ctx, cfg.CA3.Compaction.SemanticTimeout)
		defer cancel()
	}
	result, err := primary.compact(semanticCtx, request)
	if err == nil {
		threshold := cfg.CA3.Compaction.Quality.Threshold
		if result.GateThreshold > 0 {
			threshold = result.GateThreshold
		}
		if mode == "semantic" && result.QualityScore < threshold {
			err = newSemanticQualityGateError(result.QualityScore, threshold, result.QualityReason)
		}
	}
	if err == nil {
		recordCompactionAccess(result.Messages, state)
		return result, nil
	}
	if mode != "semantic" || !isBestEffortPolicy(ca3StagePolicy(cfg, stage)) {
		return ca3CompactionResult{}, err
	}
	fallback := (&truncateCompactor{})
	fallbackRes, fallbackErr := fallback.compact(ctx, request)
	if fallbackErr != nil {
		return ca3CompactionResult{}, fallbackErr
	}
	fallbackRes.Fallback = true
	fallbackRes.QualityScore = result.QualityScore
	fallbackRes.QualityReason = result.QualityReason
	fallbackRes.EmbeddingProvider = result.EmbeddingProvider
	fallbackRes.EmbeddingSimilarity = result.EmbeddingSimilarity
	fallbackRes.EmbeddingContribution = result.EmbeddingContribution
	fallbackRes.EmbeddingStatus = result.EmbeddingStatus
	fallbackRes.EmbeddingFallbackReason = result.EmbeddingFallbackReason
	fallbackRes.RerankerUsed = result.RerankerUsed
	fallbackRes.RerankerProvider = result.RerankerProvider
	fallbackRes.RerankerModel = result.RerankerModel
	fallbackRes.RerankerThresholdSource = result.RerankerThresholdSource
	fallbackRes.RerankerThresholdHit = result.RerankerThresholdHit
	fallbackRes.RerankerFallbackReason = result.RerankerFallbackReason
	fallbackRes.RerankerProfileVersion = result.RerankerProfileVersion
	fallbackRes.RerankerRolloutHit = result.RerankerRolloutHit
	fallbackRes.RerankerThresholdDrift = result.RerankerThresholdDrift
	fallbackRes.GateThreshold = result.GateThreshold
	fallbackRes.FallbackReason = semanticFallbackReason(err)
	recordCompactionAccess(fallbackRes.Messages, state)
	return fallbackRes, nil
}

func (a *Assembler) compactorFor(
	mode string,
	req types.ContextAssembleRequest,
	embeddingCfg runtimeconfig.ContextAssemblerCA3CompactionEmbeddingConfig,
	rerankerCfg runtimeconfig.ContextAssemblerCA3CompactionRerankerConfig,
) (ca3Compactor, error) {
	if mode == "semantic" {
		scorer, err := a.ensureEmbeddingScorer(embeddingCfg)
		if err != nil {
			return nil, err
		}
		reranker := a.resolveReranker(embeddingCfg, rerankerCfg)
		return &semanticCompactor{client: req.ModelClient, embedding: scorer, reranker: reranker}, nil
	}
	return &truncateCompactor{}, nil
}

func (a *Assembler) ensureEmbeddingScorer(cfg runtimeconfig.ContextAssemblerCA3CompactionEmbeddingConfig) (SemanticEmbeddingScorer, error) {
	if !cfg.Enabled {
		return nil, nil
	}
	key := strings.Join([]string{
		strings.ToLower(strings.TrimSpace(cfg.Provider)),
		strings.TrimSpace(cfg.Model),
		strings.TrimSpace(cfg.Selector),
		strings.TrimSpace(cfg.Auth.APIKey),
		strings.TrimSpace(cfg.Auth.BaseURL),
		strings.TrimSpace(cfg.ProviderAuth.OpenAI.APIKey),
		strings.TrimSpace(cfg.ProviderAuth.OpenAI.BaseURL),
		strings.TrimSpace(cfg.ProviderAuth.Gemini.APIKey),
		strings.TrimSpace(cfg.ProviderAuth.Gemini.BaseURL),
		strings.TrimSpace(cfg.ProviderAuth.Anthropic.APIKey),
		strings.TrimSpace(cfg.ProviderAuth.Anthropic.BaseURL),
	}, "|")
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.embeddingScorer != nil && strings.TrimSpace(a.embeddingKey) == "" {
		return a.embeddingScorer, nil
	}
	if a.embeddingScorer != nil && a.embeddingKey == key {
		return a.embeddingScorer, nil
	}
	scorer, err := buildEmbeddingScorer(cfg)
	if err != nil {
		return nil, err
	}
	a.embeddingScorer = scorer
	a.embeddingKey = key
	return a.embeddingScorer, nil
}

func (a *Assembler) resolveReranker(
	embeddingCfg runtimeconfig.ContextAssemblerCA3CompactionEmbeddingConfig,
	rerankerCfg runtimeconfig.ContextAssemblerCA3CompactionRerankerConfig,
) SemanticReranker {
	if !rerankerCfg.Enabled {
		return nil
	}
	provider := strings.ToLower(strings.TrimSpace(embeddingCfg.Provider))
	a.mu.Lock()
	defer a.mu.Unlock()
	if rr, ok := a.rerankers[provider]; ok && rr != nil {
		return rr
	}
	return a.defaultReranker
}

type semanticQualityGateError struct {
	score     float64
	threshold float64
	reason    string
}

func newSemanticQualityGateError(score float64, threshold float64, reason string) error {
	return semanticQualityGateError{
		score:     score,
		threshold: threshold,
		reason:    strings.TrimSpace(reason),
	}
}

func (e semanticQualityGateError) Error() string {
	return fmt.Sprintf(
		"semantic compaction quality score %.3f below threshold %.3f (%s)",
		e.score,
		e.threshold,
		e.reason,
	)
}

func semanticFallbackReason(err error) string {
	if err == nil {
		return ""
	}
	var qualityErr semanticQualityGateError
	if errors.As(err, &qualityErr) {
		return "quality_below_threshold"
	}
	return "semantic_compaction_error"
}

func ca3StagePolicy(cfg runtimeconfig.ContextAssemblerConfig, stage string) string {
	if strings.EqualFold(strings.TrimSpace(stage), "stage2") {
		return cfg.CA2.StagePolicy.Stage2
	}
	return cfg.CA2.StagePolicy.Stage1
}

func normalizeCompactionMode(mode string) string {
	out := strings.ToLower(strings.TrimSpace(mode))
	if out == "" {
		return "truncate"
	}
	return out
}

func recordCompactionAccess(messages []types.Message, state *ca3RunState) {
	if state == nil {
		return
	}
	for _, msg := range messages {
		digest := contentDigest(msg.Content)
		state.AccessFrequency[digest]++
	}
}

func (a *Assembler) spillRecords(
	ctx context.Context,
	req types.ContextAssembleRequest,
	stage string,
	records []spillRecord,
	cfg runtimeconfig.ContextAssemblerCA3Config,
	state *ca3RunState,
) (int, error) {
	if len(records) == 0 {
		return 0, nil
	}
	backend, err := a.ensureSpillBackend(cfg)
	if err != nil {
		return 0, err
	}
	written := 0
	for _, rec := range records {
		rec.RunID = req.RunID
		rec.SessionID = req.SessionID
		rec.Stage = stage
		rec.SpilledAt = a.now()
		if _, exists := state.SpillWrites[rec.OriginRef]; exists {
			continue
		}
		if err := backend.Append(ctx, rec); err != nil {
			return written, err
		}
		state.SpillWrites[rec.OriginRef] = struct{}{}
		state.SpilledByRun[rec.OriginRef] = rec
		written++
	}
	return written, nil
}

func (a *Assembler) swapBackIfNeeded(
	ctx context.Context,
	req types.ContextAssembleRequest,
	modelReq *types.ModelRequest,
	cfg runtimeconfig.ContextAssemblerCA3Config,
	state *ca3RunState,
) (int, error) {
	if !cfg.Spill.Enabled || strings.ToLower(strings.TrimSpace(cfg.Spill.Backend)) != "file" {
		return 0, nil
	}
	if cfg.Spill.SwapBackLimit <= 0 {
		return 0, nil
	}
	backend, err := a.ensureSpillBackend(cfg)
	if err != nil {
		return 0, err
	}
	recs, err := backend.LoadByRun(ctx, req.RunID, cfg.Spill.SwapBackLimit)
	if err != nil {
		return 0, err
	}
	appended := 0
	for _, rec := range recs {
		if _, ok := state.SpilledByRun[rec.OriginRef]; ok {
			continue
		}
		modelReq.Messages = append(modelReq.Messages, types.Message{
			Role:    "system",
			Content: "swap_back_context:" + rec.Content,
		})
		state.SpilledByRun[rec.OriginRef] = rec
		appended++
	}
	return appended, nil
}

func (a *Assembler) ensureSpillBackend(cfg runtimeconfig.ContextAssemblerCA3Config) (SpillBackend, error) {
	backend := strings.ToLower(strings.TrimSpace(cfg.Spill.Backend))
	if backend == "" {
		backend = "file"
	}
	key := backend + "|" + strings.TrimSpace(cfg.Spill.Path)
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.spillBackend != nil && a.spillBackendKey == key {
		return a.spillBackend, nil
	}
	switch backend {
	case "file":
		a.spillBackend = newFileSpillBackend(cfg.Spill.Path)
		a.spillBackendKey = key
		return a.spillBackend, nil
	case "db", "object":
		return nil, fmt.Errorf("ca3 spill backend %q is not implemented", backend)
	default:
		return nil, fmt.Errorf("unsupported ca3 spill backend %q", backend)
	}
}

func (a *Assembler) ca3StateFor(runID string) *ca3RunState {
	key := strings.TrimSpace(runID)
	if key == "" {
		key = "anon"
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	st := a.ca3State[key]
	if st != nil {
		return st
	}
	st = &ca3RunState{
		CurrentZone:     ca3ZoneSafe,
		ZoneEnteredAt:   a.now(),
		ZoneResidencyMs: map[string]int64{},
		TriggerCounts:   map[string]int{},
		AccessFrequency: map[string]int{},
		SpilledByRun:    map[string]spillRecord{},
		SpillWrites:     map[string]struct{}{},
	}
	a.ca3State[key] = st
	return st
}

func evaluateCA3Zone(percent int, tokens int, percentThresholds runtimeconfig.ContextAssemblerCA3Thresholds, tokenThresholds runtimeconfig.ContextAssemblerCA3Thresholds) (ca3Zone, string, string) {
	percentZone := zoneFromThreshold(percent, percentThresholds)
	tokenZone := zoneFromThreshold(tokens, tokenThresholds)
	if zonePriority(tokenZone) > zonePriority(percentZone) {
		return tokenZone, "absolute_token_trigger", string(tokenZone)
	}
	return percentZone, "usage_percent_trigger", string(percentZone)
}

func zoneFromThreshold(current int, thresholds runtimeconfig.ContextAssemblerCA3Thresholds) ca3Zone {
	switch {
	case current >= thresholds.Emergency:
		return ca3ZoneEmergency
	case current >= thresholds.Danger:
		return ca3ZoneDanger
	case current >= thresholds.Warning:
		return ca3ZoneWarning
	case current >= thresholds.Comfort:
		return ca3ZoneComfort
	default:
		return ca3ZoneSafe
	}
}

func zonePriority(zone ca3Zone) int {
	switch zone {
	case ca3ZoneSafe:
		return 0
	case ca3ZoneComfort:
		return 1
	case ca3ZoneWarning:
		return 2
	case ca3ZoneDanger:
		return 3
	case ca3ZoneEmergency:
		return 4
	default:
		return 0
	}
}

func resolveCA3Thresholds(cfg runtimeconfig.ContextAssemblerCA3Config, stage string) (runtimeconfig.ContextAssemblerCA3Thresholds, runtimeconfig.ContextAssemblerCA3Thresholds) {
	percent := cfg.PercentThresholds
	absolute := cfg.AbsoluteThresholds
	var override runtimeconfig.ContextAssemblerCA3StageThresholdOverrides
	if strings.EqualFold(strings.TrimSpace(stage), "stage2") {
		override = cfg.Stage2
	} else {
		override = cfg.Stage1
	}
	// CA4 rule: stage override fully replaces global thresholds once configured and validated.
	if hasAnyThresholdValue(override.PercentThresholds) {
		percent = override.PercentThresholds
	}
	if hasAnyThresholdValue(override.AbsoluteThresholds) {
		absolute = override.AbsoluteThresholds
	}
	return percent, absolute
}

func hasAnyThresholdValue(t runtimeconfig.ContextAssemblerCA3Thresholds) bool {
	return t.Safe > 0 || t.Comfort > 0 || t.Warning > 0 || t.Danger > 0 || t.Emergency > 0
}

func updateZoneState(state *ca3RunState, next ca3Zone, triggerReason string, now time.Time) {
	if state == nil {
		return
	}
	if state.ZoneEnteredAt.IsZero() {
		state.ZoneEnteredAt = now
	}
	if state.CurrentZone != "" {
		delta := now.Sub(state.ZoneEnteredAt).Milliseconds()
		if delta < 0 {
			delta = 0
		}
		state.ZoneResidencyMs[string(state.CurrentZone)] += delta
	}
	state.CurrentZone = next
	state.ZoneEnteredAt = now
	if strings.TrimSpace(triggerReason) != "" {
		state.TriggerCounts[strings.TrimSpace(triggerReason)]++
	}
}

func (a *Assembler) countContextTokens(
	ctx context.Context,
	assembleReq types.ContextAssembleRequest,
	req types.ModelRequest,
	cfg runtimeconfig.ContextAssemblerCA3Config,
	state *ca3RunState,
) int {
	estimate := estimateContextTokens(req)
	if state == nil {
		return estimate
	}
	signature := ca3TokenSignature(req)
	if strings.EqualFold(strings.TrimSpace(cfg.Tokenizer.Mode), "estimate_only") {
		state.LastTokenEstimate = estimate
		state.LastTokenSignature = signature
		return estimate
	}
	delta := estimate - state.LastTokenEstimate
	if delta < 0 {
		delta = -delta
	}
	if state.LastTokenSignature != "" &&
		delta <= cfg.Tokenizer.SmallDeltaTokens &&
		a.now().Sub(state.LastSDKCountAt) < cfg.Tokenizer.SDKRefreshInterval {
		state.LastTokenEstimate = estimate
		state.LastTokenSignature = signature
		return estimate
	}

	if assembleReq.TokenCounter != nil {
		tokenReq := req
		if strings.TrimSpace(tokenReq.Model) == "" {
			tokenReq.Model = strings.TrimSpace(assembleReq.Model)
		}
		if count, err := assembleReq.TokenCounter.CountTokens(ctx, tokenReq); err == nil && count > 0 {
			state.LastTokenEstimate = count
			state.LastTokenSignature = signature
			state.LastSDKCountAt = a.now()
			return count
		}
	}
	state.LastTokenEstimate = estimate
	state.LastTokenSignature = signature
	return estimate
}

func ca3TokenSignature(req types.ModelRequest) string {
	var builder strings.Builder
	builder.WriteString(req.Input)
	for _, msg := range req.Messages {
		builder.WriteString("|")
		builder.WriteString(msg.Role)
		builder.WriteString(":")
		builder.WriteString(msg.Content)
	}
	return contentDigest(builder.String())
}

func estimateContextTokens(req types.ModelRequest) int {
	if tokens := estimateContextTokensByTiktoken(req); tokens > 0 {
		return tokens
	}
	// Keep rune-based fallback for environments where local tokenizer cannot initialize.
	runes := len([]rune(req.Input))
	for _, msg := range req.Messages {
		runes += len([]rune(msg.Content))
	}
	for _, tr := range req.ToolResult {
		runes += len([]rune(tr.Name))
		runes += len([]rune(tr.Result.Content))
	}
	if runes <= 0 {
		return 0
	}
	if runes < 4 {
		return 1
	}
	return runes / 4
}

var (
	tiktokenDefaultOnce sync.Once
	tiktokenDefaultEnc  *tiktoken.Tiktoken
	tiktokenDefaultErr  error
	encodingForModelFn  = tiktoken.EncodingForModel
	getEncodingFn       = tiktoken.GetEncoding
)

func estimateContextTokensByTiktoken(req types.ModelRequest) int {
	enc, err := tokenizerForEstimate(strings.TrimSpace(req.Model))
	if err != nil || enc == nil {
		return 0
	}
	total := 0
	total += len(enc.Encode(req.Input, nil, nil))
	for _, msg := range req.Messages {
		total += len(enc.Encode(msg.Content, nil, nil))
	}
	for _, tr := range req.ToolResult {
		total += len(enc.Encode(tr.Name, nil, nil))
		total += len(enc.Encode(tr.Result.Content, nil, nil))
	}
	if total < 0 {
		return 0
	}
	return total
}

func tokenizerForEstimate(model string) (*tiktoken.Tiktoken, error) {
	if model != "" {
		if enc, err := encodingForModelFn(model); err == nil && enc != nil {
			return enc, nil
		}
	}
	tiktokenDefaultOnce.Do(func() {
		tiktokenDefaultEnc, tiktokenDefaultErr = getEncodingFn("cl100k_base")
	})
	if tiktokenDefaultEnc != nil {
		return tiktokenDefaultEnc, nil
	}
	return nil, tiktokenDefaultErr
}

func pruneMessages(messages []types.Message, cfg runtimeconfig.ContextAssemblerCA3Config, state *ca3RunState) ([]types.Message, []spillRecord, int) {
	if len(messages) == 0 {
		return messages, nil, 0
	}
	targetPercent := cfg.Prune.TargetPercent
	if targetPercent <= 0 {
		targetPercent = cfg.GoldilocksMaxPercent
	}
	targetTokens := (cfg.MaxContextTokens * targetPercent) / 100
	working := append([]types.Message(nil), messages...)
	removed := make([]spillRecord, 0)
	retained := retainedEvidenceCount(working, cfg.Compaction.Evidence)
	for estimateContextTokens(types.ModelRequest{Messages: working}) > targetTokens {
		idx := selectPruneCandidate(working, cfg, state, cfg.Compaction.Evidence)
		if idx < 0 {
			break
		}
		msg := working[idx]
		removed = append(removed, spillRecord{
			OriginRef: contentDigest(msg.Role + ":" + msg.Content),
			Content:   msg.Content,
		})
		working = append(working[:idx], working[idx+1:]...)
	}
	return working, removed, retained
}

func selectPruneCandidate(
	messages []types.Message,
	cfg runtimeconfig.ContextAssemblerCA3Config,
	state *ca3RunState,
	evidence runtimeconfig.ContextAssemblerCA3CompactionEvidenceConfig,
) int {
	type candidate struct {
		idx   int
		score int
	}
	candidates := make([]candidate, 0, len(messages))
	for i, msg := range messages {
		if strings.EqualFold(strings.TrimSpace(msg.Role), "system") {
			continue
		}
		if isProtectedMessage(msg.Content, cfg.Protection) {
			continue
		}
		if shouldRetainEvidence(i, len(messages), msg.Content, evidence) {
			continue
		}
		score := i
		lower := strings.ToLower(msg.Content)
		for _, kw := range cfg.Prune.KeywordPriority {
			if strings.Contains(lower, strings.ToLower(strings.TrimSpace(kw))) {
				score += 200
			}
		}
		score += state.AccessFrequency[contentDigest(msg.Content)]
		candidates = append(candidates, candidate{idx: i, score: score})
	}
	if len(candidates) == 0 {
		return -1
	}
	sort.Slice(candidates, func(i, j int) bool { return candidates[i].score < candidates[j].score })
	return candidates[0].idx
}

func retainedEvidenceCount(messages []types.Message, evidence runtimeconfig.ContextAssemblerCA3CompactionEvidenceConfig) int {
	count := 0
	for i, msg := range messages {
		if strings.EqualFold(strings.TrimSpace(msg.Role), "system") {
			continue
		}
		if shouldRetainEvidence(i, len(messages), msg.Content, evidence) {
			count++
		}
	}
	return count
}

func shouldRetainEvidence(idx, total int, content string, evidence runtimeconfig.ContextAssemblerCA3CompactionEvidenceConfig) bool {
	lower := strings.ToLower(content)
	for _, kw := range evidence.Keywords {
		if strings.TrimSpace(kw) == "" {
			continue
		}
		if strings.Contains(lower, strings.ToLower(strings.TrimSpace(kw))) {
			return true
		}
	}
	if evidence.RecentWindow > 0 && idx >= maxInt(0, total-evidence.RecentWindow) {
		return true
	}
	return false
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func isProtectedMessage(content string, cfg runtimeconfig.ContextAssemblerCA3ProtectionConfig) bool {
	lower := strings.ToLower(content)
	for _, kw := range cfg.CriticalKeywords {
		if strings.Contains(lower, strings.ToLower(strings.TrimSpace(kw))) {
			return true
		}
	}
	for _, kw := range cfg.ImmutableKeywords {
		if strings.Contains(lower, strings.ToLower(strings.TrimSpace(kw))) {
			return true
		}
	}
	return false
}

func isHighPriorityRequest(input string, markers []string) bool {
	lower := strings.ToLower(strings.TrimSpace(input))
	for _, marker := range markers {
		if strings.Contains(lower, strings.ToLower(strings.TrimSpace(marker))) {
			return true
		}
	}
	return false
}

func contentDigest(content string) string {
	sum := sha1.Sum([]byte(content))
	return hex.EncodeToString(sum[:])
}

func cloneIntMap(src map[string]int) map[string]int {
	if len(src) == 0 {
		return nil
	}
	out := make(map[string]int, len(src))
	for k, v := range src {
		out[k] = v
	}
	return out
}

func cloneInt64Map(src map[string]int64) map[string]int64 {
	if len(src) == 0 {
		return nil
	}
	out := make(map[string]int64, len(src))
	for k, v := range src {
		out[k] = v
	}
	return out
}
