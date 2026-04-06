package assembler

import "context"

// SemanticEmbeddingScorer is a provider-agnostic SPI hook for semantic quality scoring.
// Provider adapter binding is intentionally deferred to keep the SPI boundary stable.
type SemanticEmbeddingScorer interface {
	Score(ctx context.Context, req SemanticEmbeddingScoreRequest) (float64, error)
}

type SemanticEmbeddingScoreRequest struct {
	Selector string
	Provider string
	Model    string
	Source   string
	Summary  string
}
