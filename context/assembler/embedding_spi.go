package assembler

import "context"

// SemanticEmbeddingScorer is a provider-agnostic SPI hook for future semantic quality scoring.
// TODO: bind provider adapters in a future milestone.
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
