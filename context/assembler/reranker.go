package assembler

import (
	"context"
	"errors"
	"math"
	"strings"
)

// SemanticReranker is an extension hook for provider-specific score reranking.
type SemanticReranker interface {
	Rerank(ctx context.Context, req SemanticRerankRequest) (SemanticRerankResult, error)
}

type SemanticRerankRequest struct {
	Provider      string
	Model         string
	Source        string
	Summary       string
	RuleScore     float64
	Embedding     float64
	CurrentScore  float64
	BaseThreshold float64
}

type SemanticRerankResult struct {
	Score float64
}

type defaultSemanticReranker struct{}

func (r *defaultSemanticReranker) Rerank(_ context.Context, req SemanticRerankRequest) (SemanticRerankResult, error) {
	if req.CurrentScore < 0 || req.CurrentScore > 1 {
		return SemanticRerankResult{}, errors.New("current score out of range")
	}
	// Deterministic light-touch rerank that keeps score stable while adding slight signal from rule/embedding agreement.
	agreement := 1 - math.Abs(req.RuleScore-req.Embedding)
	if agreement < 0 {
		agreement = 0
	}
	boost := 0.05 * agreement
	score := req.CurrentScore + boost
	if score > 1 {
		score = 1
	}
	if score < 0 {
		score = 0
	}
	return SemanticRerankResult{Score: score}, nil
}

func normalizeThresholdProfileKey(provider, model string) string {
	p := strings.ToLower(strings.TrimSpace(provider))
	m := strings.ToLower(strings.TrimSpace(model))
	if p == "" || m == "" {
		return ""
	}
	return p + ":" + m
}
