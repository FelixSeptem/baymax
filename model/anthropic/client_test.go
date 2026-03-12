package anthropic

import (
	"context"
	"errors"
	"testing"

	"github.com/FelixSeptem/baymax/core/types"
	providererror "github.com/FelixSeptem/baymax/model/providererror"
)

func TestGenerateUsesConfiguredGenerateFn(t *testing.T) {
	c := NewClient(Config{
		GenerateFn: func(ctx context.Context, input string) (types.ModelResponse, error) {
			if input != "hello" {
				t.Fatalf("input = %q, want hello", input)
			}
			return types.ModelResponse{
				FinalAnswer: "ok",
				Usage:       types.TokenUsage{InputTokens: 1, OutputTokens: 1, TotalTokens: 2},
			}, nil
		},
	})
	got, err := c.Generate(context.Background(), types.ModelRequest{Input: "hello"})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}
	if got.FinalAnswer != "ok" || got.Usage.TotalTokens != 2 {
		t.Fatalf("unexpected response: %+v", got)
	}
}

func TestGenerateClassifiesProviderErrors(t *testing.T) {
	c := NewClient(Config{
		GenerateFn: func(ctx context.Context, input string) (types.ModelResponse, error) {
			return types.ModelResponse{}, providererror.FromError(errors.New("429 rate limit"))
		},
	})
	_, err := c.Generate(context.Background(), types.ModelRequest{Input: "x"})
	if err == nil {
		t.Fatal("expected error")
	}
	var classified *providererror.Classified
	if !errors.As(err, &classified) {
		t.Fatalf("expected classified provider error, got %T", err)
	}
	if classified.Reason != "rate_limit" {
		t.Fatalf("reason = %q, want rate_limit", classified.Reason)
	}
}

func TestStreamReturnsNotImplementedInM1(t *testing.T) {
	c := NewClient(Config{})
	err := c.Stream(context.Background(), types.ModelRequest{Input: "x"}, nil)
	if err == nil {
		t.Fatal("expected stream error")
	}
}
