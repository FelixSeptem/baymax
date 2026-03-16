package assembler

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"

	openai "github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
	"google.golang.org/genai"

	runtimeconfig "github.com/FelixSeptem/baymax/runtime/config"
)

type embeddingVectorAdapter interface {
	Embed(ctx context.Context, text string) ([]float64, error)
}

type semanticEmbeddingAdapterScorer struct {
	selector string
	provider string
	model    string
	adapter  embeddingVectorAdapter
}

func (s *semanticEmbeddingAdapterScorer) Score(ctx context.Context, req SemanticEmbeddingScoreRequest) (float64, error) {
	if s == nil || s.adapter == nil {
		return 0, errors.New("embedding adapter not configured")
	}
	sourceVec, err := s.adapter.Embed(ctx, req.Source)
	if err != nil {
		return 0, fmt.Errorf("embed source: %w", err)
	}
	summaryVec, err := s.adapter.Embed(ctx, req.Summary)
	if err != nil {
		return 0, fmt.Errorf("embed summary: %w", err)
	}
	score, err := cosineSimilarity(sourceVec, summaryVec)
	if err != nil {
		return 0, err
	}
	return score, nil
}

type openAIEmbeddingAdapter struct {
	client openai.Client
	model  string
}

func (a *openAIEmbeddingAdapter) Embed(ctx context.Context, text string) ([]float64, error) {
	resp, err := a.client.Embeddings.New(ctx, openai.EmbeddingNewParams{
		Input: openai.EmbeddingNewParamsInputUnion{
			OfString: openai.String(strings.TrimSpace(text)),
		},
		Model: a.model,
	})
	if err != nil {
		return nil, err
	}
	if resp == nil || len(resp.Data) == 0 || len(resp.Data[0].Embedding) == 0 {
		return nil, errors.New("openai embedding response is empty")
	}
	return resp.Data[0].Embedding, nil
}

type geminiEmbeddingAdapter struct {
	client *genai.Client
	model  string
}

func (a *geminiEmbeddingAdapter) Embed(ctx context.Context, text string) ([]float64, error) {
	resp, err := a.client.Models.EmbedContent(ctx, a.model, genai.Text(strings.TrimSpace(text)), &genai.EmbedContentConfig{})
	if err != nil {
		return nil, err
	}
	if resp == nil || len(resp.Embeddings) == 0 || resp.Embeddings[0] == nil || len(resp.Embeddings[0].Values) == 0 {
		return nil, errors.New("gemini embedding response is empty")
	}
	out := make([]float64, 0, len(resp.Embeddings[0].Values))
	for _, v := range resp.Embeddings[0].Values {
		out = append(out, float64(v))
	}
	return out, nil
}

type anthropicEmbeddingAdapter struct{}

func (a *anthropicEmbeddingAdapter) Embed(ctx context.Context, text string) ([]float64, error) {
	_ = ctx
	_ = text
	return nil, errors.New("anthropic embedding API is not available in current SDK adapter")
}

func buildEmbeddingScorer(cfg runtimeconfig.ContextAssemblerCA3CompactionEmbeddingConfig) (SemanticEmbeddingScorer, error) {
	if !cfg.Enabled {
		return nil, nil
	}
	provider := strings.ToLower(strings.TrimSpace(cfg.Provider))
	model := strings.TrimSpace(cfg.Model)
	if provider == "" || model == "" {
		return nil, errors.New("embedding provider/model is required")
	}
	auth := resolveEmbeddingAuth(cfg, provider)
	var adapter embeddingVectorAdapter
	switch provider {
	case "openai":
		opts := make([]option.RequestOption, 0, 2)
		if strings.TrimSpace(auth.APIKey) != "" {
			opts = append(opts, option.WithAPIKey(auth.APIKey))
		}
		if strings.TrimSpace(auth.BaseURL) != "" {
			opts = append(opts, option.WithBaseURL(auth.BaseURL))
		}
		adapter = &openAIEmbeddingAdapter{
			client: openai.NewClient(opts...),
			model:  model,
		}
	case "gemini":
		client, err := genai.NewClient(context.Background(), &genai.ClientConfig{
			APIKey:  strings.TrimSpace(auth.APIKey),
			Backend: genai.BackendGeminiAPI,
		})
		if err != nil {
			return nil, fmt.Errorf("create gemini embedding client: %w", err)
		}
		adapter = &geminiEmbeddingAdapter{
			client: client,
			model:  model,
		}
	case "anthropic":
		adapter = &anthropicEmbeddingAdapter{}
	default:
		return nil, fmt.Errorf("unsupported embedding provider: %s", provider)
	}
	return &semanticEmbeddingAdapterScorer{
		selector: strings.TrimSpace(cfg.Selector),
		provider: provider,
		model:    model,
		adapter:  adapter,
	}, nil
}

func resolveEmbeddingAuth(cfg runtimeconfig.ContextAssemblerCA3CompactionEmbeddingConfig, provider string) runtimeconfig.ContextAssemblerCA3EmbeddingAuthConfig {
	base := cfg.Auth
	switch provider {
	case "openai":
		if strings.TrimSpace(cfg.ProviderAuth.OpenAI.APIKey) != "" {
			base.APIKey = strings.TrimSpace(cfg.ProviderAuth.OpenAI.APIKey)
		}
		if strings.TrimSpace(cfg.ProviderAuth.OpenAI.BaseURL) != "" {
			base.BaseURL = strings.TrimSpace(cfg.ProviderAuth.OpenAI.BaseURL)
		}
	case "gemini":
		if strings.TrimSpace(cfg.ProviderAuth.Gemini.APIKey) != "" {
			base.APIKey = strings.TrimSpace(cfg.ProviderAuth.Gemini.APIKey)
		}
		if strings.TrimSpace(cfg.ProviderAuth.Gemini.BaseURL) != "" {
			base.BaseURL = strings.TrimSpace(cfg.ProviderAuth.Gemini.BaseURL)
		}
	case "anthropic":
		if strings.TrimSpace(cfg.ProviderAuth.Anthropic.APIKey) != "" {
			base.APIKey = strings.TrimSpace(cfg.ProviderAuth.Anthropic.APIKey)
		}
		if strings.TrimSpace(cfg.ProviderAuth.Anthropic.BaseURL) != "" {
			base.BaseURL = strings.TrimSpace(cfg.ProviderAuth.Anthropic.BaseURL)
		}
	}
	return base
}

func cosineSimilarity(left []float64, right []float64) (float64, error) {
	if len(left) == 0 || len(right) == 0 {
		return 0, errors.New("embedding vector is empty")
	}
	if len(left) != len(right) {
		return 0, fmt.Errorf("embedding dimension mismatch: %d != %d", len(left), len(right))
	}
	dot := 0.0
	normLeft := 0.0
	normRight := 0.0
	for i := range left {
		dot += left[i] * right[i]
		normLeft += left[i] * left[i]
		normRight += right[i] * right[i]
	}
	if normLeft <= 0 || normRight <= 0 {
		return 0, errors.New("embedding vector norm is zero")
	}
	score := dot / (math.Sqrt(normLeft) * math.Sqrt(normRight))
	if score < -1 {
		score = -1
	}
	if score > 1 {
		score = 1
	}
	// Normalize cosine similarity from [-1,1] to [0,1] for quality blending.
	return (score + 1) / 2, nil
}
