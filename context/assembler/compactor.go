package assembler

import (
	"context"
	"errors"
	"fmt"
	"math"
	"sort"
	"strings"

	"github.com/FelixSeptem/baymax/core/types"
	runtimeconfig "github.com/FelixSeptem/baymax/runtime/config"
)

type pressureCompactor interface {
	mode() string
	compact(ctx context.Context, req pressureCompactionRequest) (pressureCompactionResult, error)
}

type pressureCompactionRequest struct {
	AssembleReq types.ContextAssembleRequest
	ModelReq    types.ModelRequest
	Config      runtimeconfig.ContextAssemblerCA3Config
	StagePolicy string
}

type pressureCompactionResult struct {
	Messages                []types.Message
	CompressionRatio        float64
	Fallback                bool
	GateThreshold           float64
	QualityScore            float64
	QualityReason           string
	FallbackReason          string
	EmbeddingProvider       string
	EmbeddingSimilarity     float64
	EmbeddingContribution   float64
	EmbeddingStatus         string
	EmbeddingFallbackReason string
	RerankerUsed            bool
	RerankerProvider        string
	RerankerModel           string
	RerankerThresholdSource string
	RerankerThresholdHit    bool
	RerankerFallbackReason  string
	RerankerProfileVersion  string
	RerankerRolloutHit      bool
	RerankerThresholdDrift  float64
}

type truncateCompactor struct{}

func (c *truncateCompactor) mode() string {
	return "truncate"
}

func (c *truncateCompactor) compact(_ context.Context, req pressureCompactionRequest) (pressureCompactionResult, error) {
	before := 0
	after := 0
	messages := make([]types.Message, 0, len(req.ModelReq.Messages))
	maxRunes := req.Config.Squash.MaxContentRunes
	if maxRunes <= 0 {
		maxRunes = 320
	}
	for _, msg := range req.ModelReq.Messages {
		before += len([]rune(msg.Content))
		if strings.EqualFold(strings.TrimSpace(msg.Role), "system") || isProtectedMessage(msg.Content, req.Config.Protection) {
			after += len([]rune(msg.Content))
			messages = append(messages, msg)
			continue
		}
		content := msg.Content
		if len([]rune(content)) > maxRunes {
			content = string([]rune(content)[:maxRunes]) + " ...[squashed]"
		}
		after += len([]rune(content))
		msg.Content = content
		messages = append(messages, msg)
	}
	compression := 0.0
	if before > 0 {
		compression = float64(before-after) / float64(before)
		if compression < 0 {
			compression = 0
		}
	}
	return pressureCompactionResult{
		Messages:         messages,
		CompressionRatio: compression,
	}, nil
}

type semanticCompactor struct {
	client    types.ModelClient
	embedding SemanticEmbeddingScorer
	reranker  SemanticReranker
}

func (c *semanticCompactor) mode() string {
	return "semantic"
}

func (c *semanticCompactor) compact(ctx context.Context, req pressureCompactionRequest) (pressureCompactionResult, error) {
	if c.client == nil {
		return pressureCompactionResult{}, fmt.Errorf("semantic compactor model client not available")
	}
	before := 0
	after := 0
	out := make([]types.Message, 0, len(req.ModelReq.Messages))
	maxRunes := req.Config.Squash.MaxContentRunes
	if maxRunes <= 0 {
		maxRunes = 320
	}
	qualityScores := make([]float64, 0, len(req.ModelReq.Messages))
	qualityReasons := make([]string, 0, len(req.ModelReq.Messages))
	embeddingProvider := strings.ToLower(strings.TrimSpace(req.Config.Compaction.Embedding.Provider))
	embeddingModel := strings.TrimSpace(req.Config.Compaction.Embedding.Model)
	embeddingStatus := "disabled"
	embeddingSimilarityScores := make([]float64, 0, len(req.ModelReq.Messages))
	embeddingFallbackReason := ""
	rerankerUsed := false
	rerankerThreshold := req.Config.Compaction.Quality.Threshold
	effectiveGateThreshold := req.Config.Compaction.Quality.Threshold
	rerankerThresholdSource := "quality_threshold"
	rerankerThresholdHit := false
	rerankerFallbackReason := ""
	rerankerProfileVersion := strings.TrimSpace(req.Config.Compaction.Reranker.Governance.ProfileVersion)
	rerankerRolloutHit := false
	rerankerThresholdDrift := 0.0
	rerankerCfg := req.Config.Compaction.Reranker
	rerankerProvider := embeddingProvider
	rerankerModel := embeddingModel
	govMode := normalizeRerankerGovernanceMode(rerankerCfg.Governance.Mode)
	if req.Config.Compaction.Embedding.Enabled {
		embeddingStatus = "enabled"
		if c.embedding == nil {
			if !isBestEffortPolicy(req.StagePolicy) {
				return pressureCompactionResult{}, errors.New("embedding scorer is not configured")
			}
			embeddingStatus = "fallback_rule_only"
			embeddingFallbackReason = "embedding_hook_not_bound"
		}
	}
	if rerankerCfg.Enabled {
		if govMode == "" {
			if !isBestEffortPolicy(req.StagePolicy) {
				return pressureCompactionResult{}, errors.New("invalid reranker governance mode")
			}
			rerankerFallbackReason = "governance_mode_invalid"
		}
		var rolloutErr error
		rerankerRolloutHit, rolloutErr = isRerankerRolloutMatch(embeddingProvider, embeddingModel, rerankerCfg.Governance.RolloutProviderModels)
		if rolloutErr != nil {
			if !isBestEffortPolicy(req.StagePolicy) {
				return pressureCompactionResult{}, rolloutErr
			}
			if rerankerFallbackReason == "" {
				rerankerFallbackReason = "governance_rollout_invalid"
			}
			rerankerRolloutHit = false
		}
		key := normalizeThresholdProfileKey(embeddingProvider, embeddingModel)
		if value, ok := rerankerCfg.ThresholdProfiles[key]; ok {
			rerankerThreshold = value
			if rerankerRolloutHit {
				rerankerThresholdSource = "provider_model_profile"
				rerankerThresholdDrift = math.Abs(rerankerThreshold - req.Config.Compaction.Quality.Threshold)
			}
		} else if !isBestEffortPolicy(req.StagePolicy) {
			return pressureCompactionResult{}, fmt.Errorf("reranker threshold profile missing key %q", key)
		} else if rerankerFallbackReason == "" {
			rerankerFallbackReason = "governance_profile_missing"
		}
		if rerankerRolloutHit && govMode == runtimeconfig.CA3RerankerGovernanceModeEnforce {
			effectiveGateThreshold = rerankerThreshold
		}
	}
	for _, msg := range req.ModelReq.Messages {
		before += len([]rune(msg.Content))
		if strings.EqualFold(strings.TrimSpace(msg.Role), "system") || isProtectedMessage(msg.Content, req.Config.Protection) {
			after += len([]rune(msg.Content))
			out = append(out, msg)
			continue
		}
		if len([]rune(msg.Content)) <= maxRunes {
			after += len([]rune(msg.Content))
			out = append(out, msg)
			qualityScores = append(qualityScores, 1.0)
			qualityReasons = append(qualityReasons, "unchanged_under_limit")
			continue
		}
		prompt, err := buildSemanticCompactionPrompt(req, msg.Content, maxRunes)
		if err != nil {
			return pressureCompactionResult{}, fmt.Errorf("semantic prompt render failed: %w", err)
		}
		resp, err := c.client.Generate(ctx, types.ModelRequest{
			Model: req.ModelReq.Model,
			Input: prompt,
		})
		if err != nil {
			return pressureCompactionResult{}, fmt.Errorf("semantic compaction generate failed: %w", err)
		}
		content := strings.TrimSpace(resp.FinalAnswer)
		if content == "" {
			return pressureCompactionResult{}, fmt.Errorf("semantic compaction returned empty content")
		}
		if len([]rune(content)) > maxRunes {
			content = string([]rune(content)[:maxRunes]) + " ...[squashed]"
		}
		ruleScore, qualityReason := scoreSemanticCompaction(
			msg.Content,
			content,
			maxRunes,
			req.Config.Compaction.Quality,
			req.Config.Compaction.Evidence,
		)
		qualityScore := ruleScore
		lastSimilarity := 0.0
		if req.Config.Compaction.Embedding.Enabled && c.embedding != nil {
			scoreCtx := ctx
			cancel := func() {}
			if req.Config.Compaction.Embedding.Timeout > 0 {
				scoreCtx, cancel = context.WithTimeout(ctx, req.Config.Compaction.Embedding.Timeout)
			}
			similarity, scoreErr := c.embedding.Score(scoreCtx, SemanticEmbeddingScoreRequest{
				Selector: strings.TrimSpace(req.Config.Compaction.Embedding.Selector),
				Provider: embeddingProvider,
				Model:    strings.TrimSpace(req.Config.Compaction.Embedding.Model),
				Source:   msg.Content,
				Summary:  content,
			})
			cancel()
			if scoreErr != nil {
				if !isBestEffortPolicy(req.StagePolicy) {
					return pressureCompactionResult{}, fmt.Errorf("embedding scoring failed: %w", scoreErr)
				}
				embeddingStatus = "fallback_rule_only"
				if embeddingFallbackReason == "" {
					embeddingFallbackReason = "embedding_score_error"
				}
				qualityReason = appendReason(qualityReason, "embedding_score_error")
			} else {
				qualityScore = blendSemanticQuality(ruleScore, similarity, req.Config.Compaction.Embedding.RuleWeight, req.Config.Compaction.Embedding.EmbeddingWeight)
				embeddingSimilarityScores = append(embeddingSimilarityScores, similarity)
				lastSimilarity = similarity
				qualityReason = appendReason(qualityReason, "embedding_cosine")
			}
		}
		if rerankerCfg.Enabled && c.reranker != nil {
			rerankScore, rerankErr := applyRerankerWithRetry(ctx, c.reranker, rerankerCfg, SemanticRerankRequest{
				Provider:      embeddingProvider,
				Model:         embeddingModel,
				Source:        msg.Content,
				Summary:       content,
				RuleScore:     ruleScore,
				Embedding:     lastSimilarity,
				CurrentScore:  qualityScore,
				BaseThreshold: rerankerThreshold,
			})
			if rerankErr != nil {
				if !isBestEffortPolicy(req.StagePolicy) {
					return pressureCompactionResult{}, fmt.Errorf("reranker failed: %w", rerankErr)
				}
				rerankerFallbackReason = "reranker_error"
				qualityReason = appendReason(qualityReason, rerankerFallbackReason)
			} else {
				rerankerUsed = true
				qualityScore = rerankScore
				qualityReason = appendReason(qualityReason, "reranker_applied")
			}
		}
		if rerankerCfg.Enabled {
			observedThreshold := effectiveGateThreshold
			if rerankerRolloutHit {
				observedThreshold = rerankerThreshold
			}
			if qualityScore < observedThreshold {
				rerankerThresholdHit = true
			}
		}
		qualityScores = append(qualityScores, qualityScore)
		qualityReasons = append(qualityReasons, qualityReason)
		msg.Content = content
		after += len([]rune(content))
		out = append(out, msg)
	}
	compression := 0.0
	if before > 0 {
		compression = float64(before-after) / float64(before)
		if compression < 0 {
			compression = 0
		}
	}
	score := averageFloat64(qualityScores)
	reason := joinReasons(qualityReasons)
	embeddingSimilarity := averageFloat64(embeddingSimilarityScores)
	embeddingContribution := 0.0
	if req.Config.Compaction.Embedding.Enabled {
		embeddingContribution = embeddingSimilarity * req.Config.Compaction.Embedding.EmbeddingWeight
		if c.embedding != nil && embeddingStatus == "enabled" {
			embeddingStatus = "used"
		}
		if embeddingFallbackReason != "" {
			reason = appendReason(reason, embeddingFallbackReason)
		}
	}
	return pressureCompactionResult{
		Messages:                out,
		CompressionRatio:        compression,
		GateThreshold:           effectiveGateThreshold,
		QualityScore:            score,
		QualityReason:           reason,
		EmbeddingProvider:       embeddingProvider,
		EmbeddingSimilarity:     embeddingSimilarity,
		EmbeddingContribution:   embeddingContribution,
		EmbeddingStatus:         embeddingStatus,
		EmbeddingFallbackReason: embeddingFallbackReason,
		RerankerUsed:            rerankerUsed,
		RerankerProvider:        rerankerProvider,
		RerankerModel:           rerankerModel,
		RerankerThresholdSource: rerankerThresholdSource,
		RerankerThresholdHit:    rerankerThresholdHit,
		RerankerFallbackReason:  rerankerFallbackReason,
		RerankerProfileVersion:  rerankerProfileVersion,
		RerankerRolloutHit:      rerankerRolloutHit,
		RerankerThresholdDrift:  rerankerThresholdDrift,
	}, nil
}

func normalizeRerankerGovernanceMode(mode string) string {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case runtimeconfig.CA3RerankerGovernanceModeEnforce:
		return runtimeconfig.CA3RerankerGovernanceModeEnforce
	case runtimeconfig.CA3RerankerGovernanceModeDryRun:
		return runtimeconfig.CA3RerankerGovernanceModeDryRun
	default:
		return ""
	}
}

func isRerankerRolloutMatch(provider, model string, rollout []string) (bool, error) {
	if len(rollout) == 0 {
		return true, nil
	}
	selected := normalizeThresholdProfileKey(provider, model)
	if selected == "" {
		return false, errors.New("reranker rollout match requires provider:model key")
	}
	for _, raw := range rollout {
		key := strings.ToLower(strings.TrimSpace(raw))
		if key == "" {
			return false, errors.New("reranker rollout contains empty key")
		}
		if !strings.Contains(key, ":") {
			return false, fmt.Errorf("reranker rollout key %q must be provider:model", raw)
		}
		if key == selected {
			return true, nil
		}
	}
	return false, nil
}

func applyRerankerWithRetry(
	ctx context.Context,
	reranker SemanticReranker,
	cfg runtimeconfig.ContextAssemblerCA3CompactionRerankerConfig,
	req SemanticRerankRequest,
) (float64, error) {
	if reranker == nil {
		return req.CurrentScore, errors.New("reranker not configured")
	}
	attempts := cfg.MaxRetries + 1
	if attempts <= 0 {
		attempts = 1
	}
	var lastErr error
	for i := 0; i < attempts; i++ {
		rerankCtx := ctx
		cancel := func() {}
		if cfg.Timeout > 0 {
			rerankCtx, cancel = context.WithTimeout(ctx, cfg.Timeout)
		}
		out, err := reranker.Rerank(rerankCtx, req)
		cancel()
		if err == nil {
			score := out.Score
			if score < 0 {
				score = 0
			}
			if score > 1 {
				score = 1
			}
			return score, nil
		}
		lastErr = err
	}
	return req.CurrentScore, lastErr
}

func blendSemanticQuality(ruleScore, similarity, ruleWeight, embeddingWeight float64) float64 {
	total := ruleWeight + embeddingWeight
	if total <= 0 {
		return ruleScore
	}
	score := ((ruleScore * ruleWeight) + (similarity * embeddingWeight)) / total
	if score < 0 {
		return 0
	}
	if score > 1 {
		return 1
	}
	return score
}

func buildSemanticCompactionPrompt(req pressureCompactionRequest, content string, maxRunes int) (string, error) {
	template, err := newSemanticPromptTemplate(req.Config.Compaction.SemanticTemplate)
	if err != nil {
		return "", err
	}
	return template.Render(map[string]string{
		"input":          strings.TrimSpace(req.AssembleReq.Input),
		"source":         content,
		"max_runes":      fmt.Sprintf("%d", maxRunes),
		"model":          strings.TrimSpace(req.ModelReq.Model),
		"messages_count": fmt.Sprintf("%d", len(req.ModelReq.Messages)),
	})
}

func scoreSemanticCompaction(
	original string,
	compacted string,
	maxRunes int,
	quality runtimeconfig.ContextAssemblerCA3CompactionQualityConfig,
	evidence runtimeconfig.ContextAssemblerCA3CompactionEvidenceConfig,
) (float64, string) {
	weights := quality.Weights
	totalWeight := weights.Coverage + weights.Compression + weights.Validity
	if totalWeight <= 0 {
		return 0, "invalid_weights"
	}
	coverage := scoreCoverage(original, compacted, evidence)
	compression := scoreCompression(original, compacted, maxRunes)
	validity := scoreValidity(compacted, maxRunes)
	score := ((coverage * weights.Coverage) + (compression * weights.Compression) + (validity * weights.Validity)) / totalWeight
	if score < 0 {
		score = 0
	}
	if score > 1 {
		score = 1
	}
	reasons := make([]string, 0, 4)
	if coverage < 0.5 {
		reasons = append(reasons, "coverage_low")
	}
	if compression < 0.5 {
		reasons = append(reasons, "compression_sanity_low")
	}
	if validity < 1 {
		reasons = append(reasons, "output_invalid")
	}
	if len(reasons) == 0 {
		reasons = append(reasons, "quality_pass")
	}
	return score, strings.Join(reasons, ",")
}

func scoreCoverage(original, compacted string, evidence runtimeconfig.ContextAssemblerCA3CompactionEvidenceConfig) float64 {
	needles := make([]string, 0, len(evidence.Keywords))
	for _, kw := range evidence.Keywords {
		token := strings.ToLower(strings.TrimSpace(kw))
		if token == "" {
			continue
		}
		if strings.Contains(strings.ToLower(original), token) {
			needles = append(needles, token)
		}
	}
	if len(needles) == 0 {
		return 1
	}
	hit := 0
	lowerCompacted := strings.ToLower(compacted)
	for _, token := range needles {
		if strings.Contains(lowerCompacted, token) {
			hit++
		}
	}
	return float64(hit) / float64(len(needles))
}

func scoreCompression(original, compacted string, maxRunes int) float64 {
	origRunes := len([]rune(original))
	compRunes := len([]rune(compacted))
	if origRunes <= 0 || compRunes <= 0 {
		return 0
	}
	if maxRunes > 0 && compRunes > maxRunes {
		return 0
	}
	ratio := float64(compRunes) / float64(origRunes)
	switch {
	case ratio <= 0.15:
		return 0.35
	case ratio <= 0.85:
		return 1.0
	case ratio <= 1.0:
		return 0.55
	default:
		return 0
	}
}

func scoreValidity(compacted string, maxRunes int) float64 {
	content := strings.TrimSpace(compacted)
	if content == "" {
		return 0
	}
	if maxRunes > 0 && len([]rune(content)) > maxRunes {
		return 0
	}
	return 1
}

func averageFloat64(values []float64) float64 {
	if len(values) == 0 {
		return 1
	}
	total := 0.0
	for _, value := range values {
		total += value
	}
	out := total / float64(len(values))
	return math.Round(out*1000) / 1000
}

func joinReasons(reasons []string) string {
	if len(reasons) == 0 {
		return ""
	}
	items := strings.Split(strings.Join(reasons, ","), ",")
	seen := map[string]struct{}{}
	out := make([]string, 0, len(items))
	for _, item := range items {
		token := strings.ToLower(strings.TrimSpace(item))
		if token == "" {
			continue
		}
		if _, ok := seen[token]; ok {
			continue
		}
		seen[token] = struct{}{}
		out = append(out, token)
	}
	sort.Strings(out)
	return strings.Join(out, ",")
}

func appendReason(base, extra string) string {
	if strings.TrimSpace(extra) == "" {
		return strings.TrimSpace(base)
	}
	if strings.TrimSpace(base) == "" {
		return strings.TrimSpace(extra)
	}
	return joinReasons([]string{base, extra})
}
