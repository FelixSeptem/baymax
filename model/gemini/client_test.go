package gemini

import (
	"context"
	"errors"
	"testing"

	"github.com/FelixSeptem/baymax/core/types"
	providererror "github.com/FelixSeptem/baymax/model/providererror"
)

func TestGenerateUsesConfiguredGenerateFn(t *testing.T) {
	c := &Client{
		model: "gemini-2.5-flash",
		generate: func(ctx context.Context, input string) (types.ModelResponse, error) {
			if input != "hello" {
				t.Fatalf("input = %q, want hello", input)
			}
			return types.ModelResponse{
				FinalAnswer: "ok",
				Usage:       types.TokenUsage{InputTokens: 2, OutputTokens: 3, TotalTokens: 5},
			}, nil
		},
	}
	got, err := c.Generate(context.Background(), types.ModelRequest{Input: "hello"})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}
	if got.FinalAnswer != "ok" || got.Usage.TotalTokens != 5 {
		t.Fatalf("unexpected response: %+v", got)
	}
}

func TestGenerateClassifiesTimeoutErrors(t *testing.T) {
	c := &Client{
		model: "gemini-2.5-flash",
		generate: func(ctx context.Context, input string) (types.ModelResponse, error) {
			return types.ModelResponse{}, providererror.FromError(errors.New("request timeout"))
		},
	}
	_, err := c.Generate(context.Background(), types.ModelRequest{Input: "x"})
	if err == nil {
		t.Fatal("expected error")
	}
	var classified *providererror.Classified
	if !errors.As(err, &classified) {
		t.Fatalf("expected classified provider error, got %T", err)
	}
	if classified.Class != types.ErrPolicyTimeout {
		t.Fatalf("class = %q, want %q", classified.Class, types.ErrPolicyTimeout)
	}
}

func TestStreamReturnsNotImplementedInM1(t *testing.T) {
	c := &Client{model: "gemini-2.5-flash"}
	err := c.Stream(context.Background(), types.ModelRequest{Input: "x"}, nil)
	if err == nil {
		t.Fatal("expected stream error")
	}
}
