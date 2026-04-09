package assembler

import (
	"bytes"
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

type pressureZone string

const (
	pressureZoneSafe      pressureZone = "safe"
	pressureZoneComfort   pressureZone = "comfort"
	pressureZoneWarning   pressureZone = "warning"
	pressureZoneDanger    pressureZone = "danger"
	pressureZoneEmergency pressureZone = "emergency"
)

type pressureDecision struct {
	Zone                  pressureZone
	BlockLowPriorityLoads bool
}

type pressureRunState struct {
	CurrentZone        pressureZone
	ZoneEnteredAt      time.Time
	ZoneResidencyMs    map[string]int64
	TriggerCounts      map[string]int
	AccessFrequency    map[string]int
	SpilledByRun       map[string]spillRecord
	SpillTierByRef     map[string]string
	SpillWrites        map[string]struct{}
	LastTokenEstimate  int
	LastTokenSignature string
	LastSDKCountAt     time.Time
}

type spillRecord struct {
	RunID        string    `json:"run_id"`
	SessionID    string    `json:"session_id,omitempty"`
	Stage        string    `json:"stage"`
	OriginRef    string    `json:"origin_ref"`
	Content      string    `json:"content"`
	EvidenceTags []string  `json:"evidence_tags,omitempty"`
	SpilledAt    time.Time `json:"spilled_at"`
}

type swapBackCandidate struct {
	rec       spillRecord
	relevance float64
}

type SpillBackend interface {
	Append(ctx context.Context, rec spillRecord) error
	LoadByRun(ctx context.Context, runID string, limit int) ([]spillRecord, error)
}

type spillBatchAppender interface {
	AppendBatch(ctx context.Context, recs []spillRecord) error
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
	path               string
	opts               fileSpillBackendOptions
	mu                 sync.Mutex
	handle             *os.File
	lastGovernance     string
	lastRecoveryMarker string
}

type fileSpillBackendOptions struct {
	ReuseHandle bool
	ColdStore   runtimeconfig.RuntimeContextJITColdStoreConfig
}

type spillGovernanceReporter interface {
	LastGovernanceAction() string
	LastRecoveryMarker() string
}

func newFileSpillBackendWithOptions(path string, opts fileSpillBackendOptions) *fileSpillBackend {
	return &fileSpillBackend{
		path: strings.TrimSpace(path),
		opts: opts,
	}
}

func (f *fileSpillBackend) Append(ctx context.Context, rec spillRecord) error {
	return f.AppendBatch(ctx, []spillRecord{rec})
}

func (f *fileSpillBackend) AppendBatch(ctx context.Context, recs []spillRecord) error {
	if len(recs) == 0 {
		return nil
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	f.lastGovernance = ""
	f.lastRecoveryMarker = ""
	if err := ctx.Err(); err != nil {
		return err
	}
	if strings.TrimSpace(f.path) == "" {
		return fmt.Errorf("context pressure spill path is required")
	}
	if err := f.ensurePathLocked(); err != nil {
		return err
	}

	working, malformed, err := f.readRecordsLocked()
	if err != nil {
		return err
	}
	if malformed > 0 {
		f.lastRecoveryMarker = "cold_store_recovered_malformed_lines"
		if f.opts.ColdStore.Cleanup.Enabled {
			f.lastGovernance = mergeGovernanceAction(f.lastGovernance, "cleanup")
		}
	}

	working = append(working, recs...)
	countAfterAppend := len(working)
	working = applyColdStoreRetention(working, f.opts.ColdStore.Retention)
	if len(working) < countAfterAppend {
		f.lastGovernance = mergeGovernanceAction(f.lastGovernance, "retention")
	}
	beforeQuota := len(working)
	working = applyColdStoreQuota(working, f.opts.ColdStore.Quota.MaxBytes)
	if len(working) < beforeQuota {
		f.lastGovernance = mergeGovernanceAction(f.lastGovernance, "quota")
	}
	if f.opts.ColdStore.Compact.Enabled {
		denom := countAfterAppend
		if denom > 0 {
			fragmentation := float64(denom-len(working)) / float64(denom)
			if fragmentation >= f.opts.ColdStore.Compact.MinFragmentationRatio {
				f.lastGovernance = mergeGovernanceAction(f.lastGovernance, "compact")
			}
		}
	}

	return f.rewriteRecordsLocked(working)
}

func (f *fileSpillBackend) LastGovernanceAction() string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return strings.TrimSpace(f.lastGovernance)
}

func (f *fileSpillBackend) LastRecoveryMarker() string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return strings.TrimSpace(f.lastRecoveryMarker)
}

func (f *fileSpillBackend) rewriteRecordsLocked(records []spillRecord) error {
	if f.handle != nil {
		_ = f.handle.Close()
		f.handle = nil
	}
	var output bytes.Buffer
	for i := range records {
		row, err := json.Marshal(records[i])
		if err != nil {
			return fmt.Errorf("marshal spill record: %w", err)
		}
		output.Write(row)
		output.WriteByte('\n')
	}
	fd, release, err := f.acquireWriteRewriteHandleLocked()
	if err != nil {
		return err
	}
	if _, err := fd.Write(output.Bytes()); err != nil {
		release()
		return fmt.Errorf("write spill file: %w", err)
	}
	release()
	return nil
}

func (f *fileSpillBackend) Close() error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.handle == nil {
		return nil
	}
	if err := f.handle.Close(); err != nil {
		return fmt.Errorf("close spill file: %w", err)
	}
	f.handle = nil
	return nil
}

func (f *fileSpillBackend) ensurePathLocked() error {
	if err := os.MkdirAll(filepath.Dir(f.path), 0o755); err != nil {
		return fmt.Errorf("create spill dir: %w", err)
	}
	return nil
}

func (f *fileSpillBackend) acquireWriteRewriteHandleLocked() (*os.File, func(), error) {
	if f.opts.ReuseHandle {
		fd, err := os.OpenFile(f.path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0o600)
		if err != nil {
			return nil, nil, fmt.Errorf("open spill file: %w", err)
		}
		return fd, func() { _ = fd.Close() }, nil
	}
	fd, err := os.OpenFile(f.path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return nil, nil, fmt.Errorf("open spill file: %w", err)
	}
	return fd, func() { _ = fd.Close() }, nil
}

func (f *fileSpillBackend) readRecordsLocked() ([]spillRecord, int, error) {
	raw, err := os.ReadFile(f.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, 0, nil
		}
		return nil, 0, fmt.Errorf("read spill file: %w", err)
	}
	trimmed := strings.TrimSpace(string(raw))
	if trimmed == "" {
		return nil, 0, nil
	}
	lines := strings.Split(trimmed, "\n")
	out := make([]spillRecord, 0, len(lines))
	malformed := 0
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var rec spillRecord
		if err := json.Unmarshal([]byte(line), &rec); err != nil {
			malformed++
			continue
		}
		out = append(out, rec)
	}
	return out, malformed, nil
}

func mergeGovernanceAction(current string, candidate string) string {
	currentNormalized := normalizeGovernanceAction(current)
	candidateNormalized := normalizeGovernanceAction(candidate)
	if candidateNormalized == "" {
		return currentNormalized
	}
	if currentNormalized == "" {
		return candidateNormalized
	}
	priority := map[string]int{
		"retention_applied": 1,
		"cleanup_applied":   2,
		"compact_applied":   3,
		"quota_cleanup":     4,
	}
	if priority[candidateNormalized] >= priority[currentNormalized] {
		return candidateNormalized
	}
	return currentNormalized
}

func normalizeGovernanceAction(action string) string {
	switch strings.ToLower(strings.TrimSpace(action)) {
	case "retention", "retention_applied":
		return "retention_applied"
	case "cleanup", "cleanup_applied":
		return "cleanup_applied"
	case "compact", "compact_applied":
		return "compact_applied"
	case "quota", "quota_cleanup":
		return "quota_cleanup"
	default:
		return ""
	}
}

func applyColdStoreRetention(
	records []spillRecord,
	cfg runtimeconfig.RuntimeContextJITColdStoreRetentionConfig,
) []spillRecord {
	if len(records) == 0 {
		return records
	}
	working := append([]spillRecord(nil), records...)
	if cfg.MaxAgeMS > 0 {
		cutoff := time.Now().UTC().Add(-time.Duration(cfg.MaxAgeMS) * time.Millisecond)
		filtered := make([]spillRecord, 0, len(working))
		for i := range working {
			rec := working[i]
			if !rec.SpilledAt.IsZero() && rec.SpilledAt.Before(cutoff) {
				continue
			}
			filtered = append(filtered, rec)
		}
		working = filtered
	}
	if cfg.MaxRecords > 0 && len(working) > cfg.MaxRecords {
		working = append([]spillRecord(nil), working[len(working)-cfg.MaxRecords:]...)
	}
	return working
}

func applyColdStoreQuota(records []spillRecord, maxBytes int) []spillRecord {
	if len(records) == 0 {
		return records
	}
	if maxBytes <= 0 {
		return nil
	}
	sizes := make([]int, len(records))
	total := 0
	for i := range records {
		sizes[i] = estimatedSpillRecordBytes(records[i])
		total += sizes[i]
	}
	start := 0
	for total > maxBytes && start < len(records) {
		total -= sizes[start]
		start++
	}
	if start >= len(records) {
		return nil
	}
	return append([]spillRecord(nil), records[start:]...)
}

func estimatedSpillRecordBytes(rec spillRecord) int {
	raw, err := json.Marshal(rec)
	if err != nil {
		return len([]byte(rec.Content)) + 128
	}
	return len(raw) + 1
}

func (f *fileSpillBackend) LoadByRun(_ context.Context, runID string, limit int) ([]spillRecord, error) {
	if strings.TrimSpace(f.path) == "" {
		return nil, nil
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	f.lastGovernance = ""
	f.lastRecoveryMarker = ""
	records, malformed, err := f.readRecordsLocked()
	if err != nil {
		return nil, err
	}
	if malformed > 0 {
		f.lastRecoveryMarker = "cold_store_recovered_malformed_lines"
		if f.opts.ColdStore.Cleanup.Enabled {
			f.lastGovernance = mergeGovernanceAction(f.lastGovernance, "cleanup")
			if rewriteErr := f.rewriteRecordsLocked(records); rewriteErr != nil {
				return nil, rewriteErr
			}
		}
	}
	out := make([]spillRecord, 0, len(records))
	for i := range records {
		rec := records[i]
		if strings.TrimSpace(rec.RunID) != strings.TrimSpace(runID) {
			continue
		}
		out = append(out, rec)
	}
	if limit <= 0 || len(out) <= limit {
		return out, nil
	}
	return out[len(out)-limit:], nil
}

func (a *Assembler) applyPressureCompactionAndSwapback(
	ctx context.Context,
	req types.ContextAssembleRequest,
	modelReq types.ModelRequest,
	outcome types.ContextAssembleResult,
	cfg runtimeconfig.ContextAssemblerConfig,
	stage string,
) (types.ModelRequest, types.ContextAssembleResult, pressureDecision, error) {
	decision := pressureDecision{Zone: pressureZoneSafe}
	state := a.pressureStateFor(req.RunID)
	now := a.now()
	pressureConfig := cfg.CA3
	compactionMode := normalizeCompactionMode(pressureConfig.Compaction.Mode)
	outcome.Stage.CompactionMode = compactionMode
	runtimeContextCfg := runtimeconfig.DefaultConfig().Runtime.Context
	if a.runtimeContextConfig != nil {
		runtimeContextCfg = a.runtimeContextConfig()
	}
	outcome.Stage.ContextSwapbackRankingStrategy = strings.ToLower(strings.TrimSpace(runtimeContextCfg.JIT.SwapBack.RankingStrategy))
	outcome.Stage.ContextSwapbackCandidateWindow = runtimeContextCfg.JIT.SwapBack.CandidateWindow

	swapBackCount, swapBackRelevance, swapBackGovernanceAction, swapBackRecoveryMarker, err := a.swapBackIfNeeded(
		ctx,
		req,
		&modelReq,
		pressureConfig,
		runtimeContextCfg,
		state,
	)
	if err != nil {
		return modelReq, outcome, decision, err
	}
	coldStoreGovernanceAction := swapBackGovernanceAction
	recoveryConsistencyMarker := swapBackRecoveryMarker

	usageTokens := a.countContextTokens(ctx, req, modelReq, pressureConfig, state)
	thresholdsPercent, thresholdsTokens := resolvePressureThresholds(pressureConfig, stage)
	usagePercent := 0
	if pressureConfig.MaxContextTokens > 0 {
		usagePercent = (usageTokens * 100) / pressureConfig.MaxContextTokens
	}
	zone, reason, triggerReason := evaluatePressureZone(usagePercent, usageTokens, thresholdsPercent, thresholdsTokens)
	decision.Zone = zone
	decision.BlockLowPriorityLoads = zone == pressureZoneEmergency && pressureConfig.Emergency.RejectLowPriority

	updateZoneState(state, zone, triggerReason, now)
	beforeTokens := usageTokens
	actionsCompression := 0.0
	spillCount := 0
	removed := make([]spillRecord, 0)
	fallbackUsed, fallbackReason := false, ""
	qualityScore, qualityReason := 0.0, ""
	embeddingProvider, embeddingStatus, embeddingFallbackReason := "", "", ""
	embeddingSimilarity := 0.0
	embeddingContribution := 0.0
	rerankerUsed, rerankerThresholdHit, rerankerRolloutHit := false, false, false
	rerankerProvider := ""
	rerankerModel := ""
	rerankerThresholdSource := ""
	rerankerFallbackReason := ""
	rerankerProfileVersion := ""
	rerankerThresholdDrift := 0.0
	retainedEvidenceCount := 0

	switch zone {
	case pressureZoneWarning:
		if pressureConfig.Squash.Enabled {
			var compaction pressureCompactionResult
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
	case pressureZoneDanger:
		if pressureConfig.Squash.Enabled {
			var compaction pressureCompactionResult
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
		if pressureConfig.Prune.Enabled {
			var pruned []spillRecord
			modelReq.Messages, pruned, retainedEvidenceCount = pruneMessages(
				modelReq.Messages,
				pressureConfig,
				state,
				runtimeContextCfg.JIT.Compaction.RuleEligibility,
			)
			_ = pruned
		}
	case pressureZoneEmergency:
		if pressureConfig.Squash.Enabled {
			var compaction pressureCompactionResult
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
		if pressureConfig.Prune.Enabled {
			modelReq.Messages, removed, retainedEvidenceCount = pruneMessages(
				modelReq.Messages,
				pressureConfig,
				state,
				runtimeContextCfg.JIT.Compaction.RuleEligibility,
			)
		}
		if pressureConfig.Spill.Enabled {
			var spillGovernanceAction string
			var spillRecoveryMarker string
			spillCount, spillGovernanceAction, spillRecoveryMarker, err = a.spillRecords(
				ctx,
				req,
				stage,
				removed,
				pressureConfig,
				runtimeContextCfg.JIT.ColdStore,
				state,
			)
			if err != nil {
				return modelReq, outcome, decision, err
			}
			coldStoreGovernanceAction = mergeGovernanceAction(coldStoreGovernanceAction, spillGovernanceAction)
			recoveryConsistencyMarker = mergeRecoveryMarker(recoveryConsistencyMarker, spillRecoveryMarker)
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
	outcome.Stage.CompactionOutcomeClass = classifyCompactionOutcomeClass(
		outcome.Stage.CompactionMode,
		outcome.Stage.CompactionFallback,
		outcome.Stage.CompactionFallbackReason,
		outcome.Stage.CompactionQualityScore,
	)
	if retainedEvidenceCount > 0 {
		outcome.Stage.RetainedEvidenceCount += retainedEvidenceCount
	}
	if swapBackRelevance > 0 {
		outcome.Stage.ContextSwapbackRelevanceScore = swapBackRelevance
	}
	if strings.TrimSpace(coldStoreGovernanceAction) != "" {
		outcome.Stage.ContextColdStoreGovernanceAction = coldStoreGovernanceAction
	}
	if strings.TrimSpace(recoveryConsistencyMarker) != "" {
		outcome.Stage.ContextRecoveryConsistencyMarker = recoveryConsistencyMarker
	}
	if runtimeContextCfg.JIT.LifecycleTiering.Enabled {
		tierStats, lifecycleAction := applyLifecycleTiering(state, now, runtimeContextCfg.JIT.LifecycleTiering)
		if len(tierStats) > 0 {
			if outcome.Stage.ContextLifecycleTierStats == nil {
				outcome.Stage.ContextLifecycleTierStats = map[string]int{}
			}
			for key, value := range tierStats {
				outcome.Stage.ContextLifecycleTierStats[key] += value
			}
		}
		if strings.TrimSpace(lifecycleAction) != "" {
			outcome.Stage.MemoryLifecycleAction = lifecycleAction
			outcome.Stage.ContextTierTransitionReason = lifecycleAction
		}
	}
	return modelReq, outcome, decision, nil
}

func (a *Assembler) applyCompaction(
	ctx context.Context,
	assembleReq types.ContextAssembleRequest,
	modelReq types.ModelRequest,
	cfg runtimeconfig.ContextAssemblerConfig,
	state *pressureRunState,
	stage string,
) (pressureCompactionResult, error) {
	pressureConfig := cfg.CA3
	runtimeContextCfg := runtimeconfig.DefaultConfig().Runtime.Context
	if a.runtimeContextConfig != nil {
		runtimeContextCfg = a.runtimeContextConfig()
	}
	stagePolicy := resolveCompactionStagePolicy(pressureStagePolicy(cfg, stage), runtimeContextCfg.JIT.Compaction.FallbackPolicy)
	pressureConfig.Compaction.Quality.Threshold = runtimeContextCfg.JIT.Compaction.QualityThreshold
	request := pressureCompactionRequest{
		AssembleReq: assembleReq,
		ModelReq:    modelReq,
		Config:      pressureConfig,
		StagePolicy: stagePolicy,
	}
	mode := normalizeCompactionMode(pressureConfig.Compaction.Mode)
	primary, err := a.compactorFor(mode, assembleReq, pressureConfig.Compaction.Embedding, pressureConfig.Compaction.Reranker)
	if err != nil {
		return pressureCompactionResult{}, err
	}
	semanticCtx := ctx
	var cancel context.CancelFunc
	if mode == "semantic" {
		semanticCtx, cancel = context.WithTimeout(ctx, pressureConfig.Compaction.SemanticTimeout)
		defer cancel()
	}
	result, err := primary.compact(semanticCtx, request)
	if err == nil {
		threshold := pressureConfig.Compaction.Quality.Threshold
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
	if mode != "semantic" || !isBestEffortPolicy(stagePolicy) {
		return pressureCompactionResult{}, err
	}
	fallback := (&truncateCompactor{})
	fallbackRes, fallbackErr := fallback.compact(ctx, request)
	if fallbackErr != nil {
		return pressureCompactionResult{}, fallbackErr
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
) (pressureCompactor, error) {
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

func pressureStagePolicy(cfg runtimeconfig.ContextAssemblerConfig, stage string) string {
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

func resolveCompactionStagePolicy(stagePolicy string, runtimeFallbackPolicy string) string {
	stagePolicy = strings.ToLower(strings.TrimSpace(stagePolicy))
	fallbackPolicy := strings.ToLower(strings.TrimSpace(runtimeFallbackPolicy))
	if fallbackPolicy == runtimeconfig.RuntimeContextJITCompactionFallbackPolicyFailFast {
		return runtimeconfig.RuntimeContextJITCompactionFallbackPolicyFailFast
	}
	if stagePolicy != "" {
		return stagePolicy
	}
	if fallbackPolicy != "" {
		return fallbackPolicy
	}
	return runtimeconfig.RuntimeContextJITCompactionFallbackPolicyBestEffort
}

func classifyCompactionOutcomeClass(mode string, fallback bool, fallbackReason string, qualityScore float64) string {
	normalizedMode := strings.ToLower(strings.TrimSpace(mode))
	if normalizedMode == "" && !fallback && qualityScore <= 0 {
		return ""
	}
	if !fallback {
		return "applied"
	}
	reason := strings.ToLower(strings.TrimSpace(fallbackReason))
	if reason == "quality_below_threshold" {
		return "degraded"
	}
	if reason == "semantic_compaction_error" {
		return "fallback"
	}
	if strings.Contains(reason, "error") {
		return "fallback"
	}
	return "fallback"
}

func recordCompactionAccess(messages []types.Message, state *pressureRunState) {
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
	coldStoreCfg runtimeconfig.RuntimeContextJITColdStoreConfig,
	state *pressureRunState,
) (int, string, string, error) {
	if len(records) == 0 {
		return 0, "", "", nil
	}
	backend, err := a.ensureSpillBackend(cfg, coldStoreCfg)
	if err != nil {
		return 0, "", "", err
	}
	pending := make([]spillRecord, 0, len(records))
	for _, rec := range records {
		rec.RunID = req.RunID
		rec.SessionID = req.SessionID
		rec.Stage = stage
		rec.SpilledAt = a.now()
		rec.EvidenceTags = extractEvidenceTags(rec.Content, cfg.Compaction.Evidence.Keywords)
		if _, exists := state.SpillWrites[rec.OriginRef]; exists {
			continue
		}
		pending = append(pending, rec)
	}
	if len(pending) == 0 {
		return 0, "", "", nil
	}
	batchSize := cfg.Spill.File.BatchFlushSize
	if batchSize <= 0 {
		batchSize = 1
	}
	written := 0
	batchWriter, hasBatchWriter := backend.(spillBatchAppender)
	for i := 0; i < len(pending); i += batchSize {
		end := i + batchSize
		if end > len(pending) {
			end = len(pending)
		}
		chunk := pending[i:end]
		if hasBatchWriter {
			if err := batchWriter.AppendBatch(ctx, chunk); err != nil {
				return written, "", "", err
			}
		} else {
			for _, rec := range chunk {
				if err := backend.Append(ctx, rec); err != nil {
					return written, "", "", err
				}
			}
		}
		for _, rec := range chunk {
			state.SpillWrites[rec.OriginRef] = struct{}{}
			state.SpilledByRun[rec.OriginRef] = rec
			if state.SpillTierByRef == nil {
				state.SpillTierByRef = map[string]string{}
			}
			state.SpillTierByRef[rec.OriginRef] = "hot"
			written++
		}
	}
	governanceAction, recoveryMarker := coldStoreBackendReport(backend)
	if recoveryMarker == "" {
		recoveryMarker = "idempotent"
	}
	return written, governanceAction, recoveryMarker, nil
}

func (a *Assembler) swapBackIfNeeded(
	ctx context.Context,
	req types.ContextAssembleRequest,
	modelReq *types.ModelRequest,
	cfg runtimeconfig.ContextAssemblerCA3Config,
	runtimeContextCfg runtimeconfig.RuntimeContextConfig,
	state *pressureRunState,
) (int, float64, string, string, error) {
	if !cfg.Spill.Enabled || strings.ToLower(strings.TrimSpace(cfg.Spill.Backend)) != "file" {
		return 0, 0, "", "", nil
	}
	if cfg.Spill.SwapBackLimit <= 0 {
		return 0, 0, "", "", nil
	}
	backend, err := a.ensureSpillBackend(cfg, runtimeContextCfg.JIT.ColdStore)
	if err != nil {
		return 0, 0, "", "", err
	}
	candidateWindow := runtimeContextCfg.JIT.SwapBack.CandidateWindow
	if candidateWindow <= 0 {
		candidateWindow = cfg.Spill.SwapBackLimit
	}
	loadLimit := candidateWindow
	if loadLimit < cfg.Spill.SwapBackLimit {
		loadLimit = cfg.Spill.SwapBackLimit
	}
	recs, err := backend.LoadByRun(ctx, req.RunID, loadLimit)
	if err != nil {
		return 0, 0, "", "", err
	}
	appended := 0
	maxRelevance := 0.0
	minScore := 0.0
	if runtimeContextCfg.JIT.SwapBack.Enabled {
		minScore = runtimeContextCfg.JIT.SwapBack.MinRelevanceScore
	}
	now := a.now()
	candidates := make([]swapBackCandidate, 0, len(recs))
	dedupCount := 0
	for _, rec := range recs {
		if runtimeContextCfg.JIT.LifecycleTiering.Enabled &&
			runtimeContextCfg.JIT.LifecycleTiering.ColdTTLMS > 0 &&
			!rec.SpilledAt.IsZero() &&
			now.Sub(rec.SpilledAt).Milliseconds() > int64(runtimeContextCfg.JIT.LifecycleTiering.ColdTTLMS) {
			continue
		}
		if _, ok := state.SpilledByRun[rec.OriginRef]; ok {
			dedupCount++
			continue
		}
		relevance := scoreSwapBackRelevance(req.Input, rec)
		if relevance > maxRelevance {
			maxRelevance = relevance
		}
		candidates = append(candidates, swapBackCandidate{rec: rec, relevance: relevance})
	}
	sortSwapBackCandidates(candidates, runtimeContextCfg.JIT.SwapBack.RankingStrategy)
	if candidateWindow > 0 && len(candidates) > candidateWindow {
		candidates = candidates[:candidateWindow]
	}
	for _, candidate := range candidates {
		if candidate.relevance < minScore {
			continue
		}
		modelReq.Messages = append(modelReq.Messages, types.Message{
			Role:    "system",
			Content: "swap_back_context:" + candidate.rec.Content,
		})
		state.SpilledByRun[candidate.rec.OriginRef] = candidate.rec
		if state.SpillTierByRef == nil {
			state.SpillTierByRef = map[string]string{}
		}
		state.SpillTierByRef[candidate.rec.OriginRef] = "cold"
		appended++
		if appended >= cfg.Spill.SwapBackLimit {
			break
		}
	}
	governanceAction, recoveryMarker := coldStoreBackendReport(backend)
	if strings.TrimSpace(recoveryMarker) == "" {
		if dedupCount > 0 {
			recoveryMarker = "deduplicated"
		} else {
			recoveryMarker = "idempotent"
		}
	}
	return appended, maxRelevance, governanceAction, recoveryMarker, nil
}

func coldStoreBackendReport(backend SpillBackend) (string, string) {
	reporter, ok := backend.(spillGovernanceReporter)
	if !ok {
		return "", ""
	}
	return strings.TrimSpace(reporter.LastGovernanceAction()), strings.TrimSpace(reporter.LastRecoveryMarker())
}

func mergeRecoveryMarker(current string, candidate string) string {
	current = strings.ToLower(strings.TrimSpace(current))
	candidate = strings.ToLower(strings.TrimSpace(candidate))
	if candidate == "" {
		return current
	}
	if current == "" {
		return candidate
	}
	priority := map[string]int{
		"idempotent":                           1,
		"deduplicated":                         2,
		"cold_store_recovered_malformed_lines": 3,
	}
	if priority[candidate] >= priority[current] {
		return candidate
	}
	return current
}

func sortSwapBackCandidates(candidates []swapBackCandidate, strategy string) {
	resolved := normalizeSwapBackRankingStrategy(strategy)
	sort.SliceStable(candidates, func(i, j int) bool {
		left := candidates[i]
		right := candidates[j]
		if resolved == runtimeconfig.RuntimeContextJITSwapBackRankingStrategyRelevanceThenRecency {
			if left.relevance != right.relevance {
				return left.relevance > right.relevance
			}
		}
		if !left.rec.SpilledAt.Equal(right.rec.SpilledAt) {
			return left.rec.SpilledAt.After(right.rec.SpilledAt)
		}
		return strings.Compare(strings.TrimSpace(left.rec.OriginRef), strings.TrimSpace(right.rec.OriginRef)) < 0
	})
}

func normalizeSwapBackRankingStrategy(strategy string) string {
	normalized := strings.ToLower(strings.TrimSpace(strategy))
	switch normalized {
	case runtimeconfig.RuntimeContextJITSwapBackRankingStrategyRecencyOnly:
		return runtimeconfig.RuntimeContextJITSwapBackRankingStrategyRecencyOnly
	default:
		return runtimeconfig.RuntimeContextJITSwapBackRankingStrategyRelevanceThenRecency
	}
}

func applyLifecycleTiering(
	state *pressureRunState,
	now time.Time,
	cfg runtimeconfig.RuntimeContextJITLifecycleTieringConfig,
) (map[string]int, string) {
	if state == nil || len(state.SpilledByRun) == 0 {
		return nil, ""
	}
	if state.SpillTierByRef == nil {
		state.SpillTierByRef = map[string]string{}
	}
	stats := map[string]int{
		"hot":    0,
		"warm":   0,
		"cold":   0,
		"pruned": 0,
	}
	lifecycleAction := ""
	for originRef, rec := range state.SpilledByRun {
		if rec.SpilledAt.IsZero() {
			rec.SpilledAt = now
		}
		ageMS := now.Sub(rec.SpilledAt).Milliseconds()
		var nextTier string
		switch {
		case ageMS <= int64(cfg.HotTTLMS):
			nextTier = "hot"
		case ageMS <= int64(cfg.WarmTTLMS):
			nextTier = "warm"
			if len([]rune(rec.Content)) > 256 {
				rec.Content = truncateRunes(rec.Content, 256)
				state.SpilledByRun[originRef] = rec
				lifecycleAction = mergeLifecycleAction(lifecycleAction, "compress")
			}
		case ageMS <= int64(cfg.ColdTTLMS):
			nextTier = "cold"
			lifecycleAction = mergeLifecycleAction(lifecycleAction, "spill")
		default:
			stats["pruned"]++
			delete(state.SpilledByRun, originRef)
			delete(state.SpillTierByRef, originRef)
			lifecycleAction = mergeLifecycleAction(lifecycleAction, "prune")
			continue
		}
		stats[nextTier]++
		prevTier := strings.TrimSpace(state.SpillTierByRef[originRef])
		if prevTier != "" && prevTier != nextTier {
			stats["migrate_"+prevTier+"_to_"+nextTier]++
		}
		state.SpillTierByRef[originRef] = nextTier
	}
	if stats["hot"] == 0 && stats["warm"] == 0 && stats["cold"] == 0 && stats["pruned"] == 0 {
		return nil, lifecycleAction
	}
	return stats, lifecycleAction
}

func scoreSwapBackRelevance(query string, rec spillRecord) float64 {
	queryTerms := tokenizeRelevanceTerms(query)
	if len(queryTerms) == 0 {
		return 0
	}
	tagTerms := map[string]struct{}{}
	for _, tag := range rec.EvidenceTags {
		normalized := strings.ToLower(strings.TrimSpace(tag))
		if normalized == "" {
			continue
		}
		tagTerms[normalized] = struct{}{}
	}
	contentTerms := tokenizeRelevanceTerms(rec.Content)
	tagHit := overlapRatio(queryTerms, tagTerms)
	contentHit := overlapRatio(queryTerms, contentTerms)
	score := (0.7 * tagHit) + (0.3 * contentHit)
	if score < 0 {
		return 0
	}
	if score > 1 {
		return 1
	}
	return score
}

func extractEvidenceTags(content string, configured []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(configured))
	lower := strings.ToLower(content)
	for _, kw := range configured {
		tag := strings.ToLower(strings.TrimSpace(kw))
		if tag == "" {
			continue
		}
		if _, ok := seen[tag]; ok {
			continue
		}
		if strings.Contains(lower, tag) {
			seen[tag] = struct{}{}
			out = append(out, tag)
		}
	}
	if len(out) > 0 {
		return out
	}
	for token := range tokenizeRelevanceTerms(content) {
		if len(out) >= 6 {
			break
		}
		if len(token) < 4 {
			continue
		}
		if _, ok := seen[token]; ok {
			continue
		}
		seen[token] = struct{}{}
		out = append(out, token)
	}
	sort.Strings(out)
	return out
}

func tokenizeRelevanceTerms(raw string) map[string]struct{} {
	out := map[string]struct{}{}
	normalized := strings.ToLower(strings.TrimSpace(raw))
	if normalized == "" {
		return out
	}
	replacer := strings.NewReplacer(
		"\n", " ",
		"\r", " ",
		"\t", " ",
		",", " ",
		".", " ",
		";", " ",
		":", " ",
		"!", " ",
		"?", " ",
		"(", " ",
		")", " ",
		"[", " ",
		"]", " ",
		"{", " ",
		"}", " ",
		"\"", " ",
		"'", " ",
		"`", " ",
		"/", " ",
		"\\", " ",
		"|", " ",
		"+", " ",
		"-", " ",
		"_", " ",
		"#", " ",
		"@", " ",
	)
	normalized = replacer.Replace(normalized)
	parts := strings.Fields(normalized)
	for _, part := range parts {
		token := strings.TrimSpace(part)
		if token == "" {
			continue
		}
		if len(token) <= 1 {
			continue
		}
		out[token] = struct{}{}
	}
	return out
}

func overlapRatio(queryTerms map[string]struct{}, targetTerms map[string]struct{}) float64 {
	if len(queryTerms) == 0 || len(targetTerms) == 0 {
		return 0
	}
	hits := 0
	for token := range queryTerms {
		if _, ok := targetTerms[token]; ok {
			hits++
		}
	}
	return float64(hits) / float64(len(queryTerms))
}

func mergeLifecycleAction(current string, candidate string) string {
	priority := map[string]int{
		"":         0,
		"spill":    1,
		"compress": 2,
		"prune":    3,
	}
	if priority[strings.TrimSpace(candidate)] > priority[strings.TrimSpace(current)] {
		return strings.TrimSpace(candidate)
	}
	return strings.TrimSpace(current)
}

func (a *Assembler) ensureSpillBackend(
	cfg runtimeconfig.ContextAssemblerCA3Config,
	coldStoreCfg runtimeconfig.RuntimeContextJITColdStoreConfig,
) (SpillBackend, error) {
	backend := strings.ToLower(strings.TrimSpace(cfg.Spill.Backend))
	if backend == "" {
		backend = "file"
	}
	key := strings.Join([]string{
		backend,
		strings.TrimSpace(cfg.Spill.Path),
		fmt.Sprintf("reuse=%t", cfg.Spill.File.ReuseHandle),
		fmt.Sprintf("batch=%d", cfg.Spill.File.BatchFlushSize),
		fmt.Sprintf("retention_age_ms=%d", coldStoreCfg.Retention.MaxAgeMS),
		fmt.Sprintf("retention_max_records=%d", coldStoreCfg.Retention.MaxRecords),
		fmt.Sprintf("quota_max_bytes=%d", coldStoreCfg.Quota.MaxBytes),
		fmt.Sprintf("cleanup_enabled=%t", coldStoreCfg.Cleanup.Enabled),
		fmt.Sprintf("cleanup_batch=%d", coldStoreCfg.Cleanup.BatchSize),
		fmt.Sprintf("compact_enabled=%t", coldStoreCfg.Compact.Enabled),
		fmt.Sprintf("compact_min_frag=%.6f", coldStoreCfg.Compact.MinFragmentationRatio),
	}, "|")
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.spillBackend != nil && a.spillBackendKey == key {
		return a.spillBackend, nil
	}
	if closer, ok := a.spillBackend.(interface{ Close() error }); ok {
		_ = closer.Close()
	}
	switch backend {
	case "file":
		a.spillBackend = newFileSpillBackendWithOptions(cfg.Spill.Path, fileSpillBackendOptions{
			ReuseHandle: cfg.Spill.File.ReuseHandle,
			ColdStore:   coldStoreCfg,
		})
		a.spillBackendKey = key
		return a.spillBackend, nil
	case "db", "object":
		return nil, fmt.Errorf("context pressure spill backend %q is not implemented", backend)
	default:
		return nil, fmt.Errorf("unsupported context pressure spill backend %q", backend)
	}
}

func (a *Assembler) pressureStateFor(runID string) *pressureRunState {
	cfg := a.cacheConfigSnapshot()
	now := a.now()
	key := strings.TrimSpace(runID)
	if key == "" {
		key = "anon"
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	a.compactCachesLocked(now, cfg)
	st := a.pressureState[key]
	if st != nil {
		a.pressureUsed[key] = now
		return st
	}
	st = &pressureRunState{
		CurrentZone:     pressureZoneSafe,
		ZoneEnteredAt:   now,
		ZoneResidencyMs: map[string]int64{},
		TriggerCounts:   map[string]int{},
		AccessFrequency: map[string]int{},
		SpilledByRun:    map[string]spillRecord{},
		SpillTierByRef:  map[string]string{},
		SpillWrites:     map[string]struct{}{},
	}
	a.pressureState[key] = st
	a.pressureUsed[key] = now
	a.compactCachesLocked(now, cfg)
	return st
}

func evaluatePressureZone(percent int, tokens int, percentThresholds runtimeconfig.ContextAssemblerCA3Thresholds, tokenThresholds runtimeconfig.ContextAssemblerCA3Thresholds) (pressureZone, string, string) {
	percentZone := zoneFromThreshold(percent, percentThresholds)
	tokenZone := zoneFromThreshold(tokens, tokenThresholds)
	if zonePriority(tokenZone) > zonePriority(percentZone) {
		return tokenZone, "absolute_token_trigger", string(tokenZone)
	}
	return percentZone, "usage_percent_trigger", string(percentZone)
}

func zoneFromThreshold(current int, thresholds runtimeconfig.ContextAssemblerCA3Thresholds) pressureZone {
	switch {
	case current >= thresholds.Emergency:
		return pressureZoneEmergency
	case current >= thresholds.Danger:
		return pressureZoneDanger
	case current >= thresholds.Warning:
		return pressureZoneWarning
	case current >= thresholds.Comfort:
		return pressureZoneComfort
	default:
		return pressureZoneSafe
	}
}

func zonePriority(zone pressureZone) int {
	switch zone {
	case pressureZoneSafe:
		return 0
	case pressureZoneComfort:
		return 1
	case pressureZoneWarning:
		return 2
	case pressureZoneDanger:
		return 3
	case pressureZoneEmergency:
		return 4
	default:
		return 0
	}
}

func resolvePressureThresholds(cfg runtimeconfig.ContextAssemblerCA3Config, stage string) (runtimeconfig.ContextAssemblerCA3Thresholds, runtimeconfig.ContextAssemblerCA3Thresholds) {
	percent := cfg.PercentThresholds
	absolute := cfg.AbsoluteThresholds
	var override runtimeconfig.ContextAssemblerCA3StageThresholdOverrides
	if strings.EqualFold(strings.TrimSpace(stage), "stage2") {
		override = cfg.Stage2
	} else {
		override = cfg.Stage1
	}
	// Threshold-governance rule: stage override fully replaces global thresholds once configured and validated.
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

func updateZoneState(state *pressureRunState, next pressureZone, triggerReason string, now time.Time) {
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
	state *pressureRunState,
) int {
	estimate := estimateContextTokens(req)
	if state == nil {
		return estimate
	}
	signature := pressureTokenSignature(req)
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

func pressureTokenSignature(req types.ModelRequest) string {
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

func shouldSkipStage2CA3Pass(
	cfg runtimeconfig.ContextAssemblerCA3Config,
	stage2Enabled bool,
	stage2InputSignature string,
	req types.ModelRequest,
) bool {
	if !stage2Enabled || !cfg.Stage2Pass.SkipNoDelta {
		return false
	}
	before := strings.TrimSpace(stage2InputSignature)
	if before == "" {
		return false
	}
	after := strings.TrimSpace(pressureTokenSignature(req))
	if after == "" {
		return false
	}
	return before == after
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

func pruneMessages(
	messages []types.Message,
	cfg runtimeconfig.ContextAssemblerCA3Config,
	state *pressureRunState,
	ruleEligibility runtimeconfig.RuntimeContextJITCompactionRuleEligibility,
) ([]types.Message, []spillRecord, int) {
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
		idx := selectPruneCandidate(
			working,
			cfg,
			state,
			cfg.Compaction.Evidence,
			ruleEligibility,
			retained,
		)
		if idx < 0 {
			break
		}
		msg := working[idx]
		if shouldRetainEvidence(idx, len(working), msg.Content, cfg.Compaction.Evidence) && retained > 0 {
			retained--
		}
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
	state *pressureRunState,
	evidence runtimeconfig.ContextAssemblerCA3CompactionEvidenceConfig,
	ruleEligibility runtimeconfig.RuntimeContextJITCompactionRuleEligibility,
	currentRetainedEvidence int,
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
		evidenceRetained := shouldRetainEvidence(i, len(messages), msg.Content, evidence)
		if evidenceRetained && currentRetainedEvidence <= ruleEligibility.MinRetainedEvidence {
			continue
		}
		if !ruleEligibility.AllowOldestToolResult && isOldestToolResultCandidate(messages, i) {
			continue
		}
		score := i
		lower := strings.ToLower(msg.Content)
		for _, kw := range cfg.Prune.KeywordPriority {
			if strings.Contains(lower, strings.ToLower(strings.TrimSpace(kw))) {
				score += 200
			}
		}
		if evidenceRetained {
			score += 300
		}
		score += state.AccessFrequency[contentDigest(msg.Content)]
		candidates = append(candidates, candidate{idx: i, score: score})
	}
	if len(candidates) == 0 {
		return -1
	}
	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].score != candidates[j].score {
			return candidates[i].score < candidates[j].score
		}
		return candidates[i].idx < candidates[j].idx
	})
	return candidates[0].idx
}

func isOldestToolResultCandidate(messages []types.Message, idx int) bool {
	oldestToolResultIdx := -1
	for i := range messages {
		if isToolResultMessage(messages[i]) {
			oldestToolResultIdx = i
			break
		}
	}
	return oldestToolResultIdx >= 0 && idx == oldestToolResultIdx
}

func isToolResultMessage(msg types.Message) bool {
	role := strings.ToLower(strings.TrimSpace(msg.Role))
	if role == "tool" {
		return true
	}
	content := strings.ToLower(strings.TrimSpace(msg.Content))
	return strings.HasPrefix(content, "tool_result:") || strings.HasPrefix(content, "tool_call_result:")
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
