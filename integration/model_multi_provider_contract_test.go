package integration

import (
	"context"
	"errors"
	"testing"

	"github.com/FelixSeptem/baymax/core/runner"
	"github.com/FelixSeptem/baymax/core/types"
	anthropicmodel "github.com/FelixSeptem/baymax/model/anthropic"
	geminimodel "github.com/FelixSeptem/baymax/model/gemini"
	openaimodel "github.com/FelixSeptem/baymax/model/openai"
	providererror "github.com/FelixSeptem/baymax/model/providererror"
)

func TestModelProviderContractRunSuccess(t *testing.T) {
	cases := map[string]types.ModelClient{
		"openai": openaimodel.NewClient(openaimodel.Config{
			GenerateFn: func(ctx context.Context, req types.ModelRequest) (types.ModelResponse, error) {
				return types.ModelResponse{FinalAnswer: "ok-openai"}, nil
			},
		}),
		"anthropic": anthropicmodel.NewClient(anthropicmodel.Config{
			GenerateFn: func(ctx context.Context, input string) (types.ModelResponse, error) {
				return types.ModelResponse{FinalAnswer: "ok-anthropic"}, nil
			},
		}),
	}
	geminiClient, err := geminimodel.NewClient(context.Background(), geminimodel.Config{
		GenerateFn: func(ctx context.Context, input string) (types.ModelResponse, error) {
			return types.ModelResponse{FinalAnswer: "ok-gemini"}, nil
		},
	})
	if err != nil {
		t.Fatalf("new gemini client: %v", err)
	}
	cases["gemini"] = geminiClient

	for name, model := range cases {
		t.Run(name, func(t *testing.T) {
			eng := runner.New(model)
			res, runErr := eng.Run(context.Background(), types.RunRequest{Input: "hello"}, nil)
			if runErr != nil {
				t.Fatalf("Run error: %v", runErr)
			}
			if res.FinalAnswer == "" {
				t.Fatalf("empty final answer for provider %s", name)
			}
			if res.Error != nil {
				t.Fatalf("unexpected classified error: %+v", res.Error)
			}
		})
	}
}

func TestModelProviderContractErrorClassification(t *testing.T) {
	cases := map[string]types.ModelClient{
		"openai-timeout": openaimodel.NewClient(openaimodel.Config{
			GenerateFn: func(ctx context.Context, req types.ModelRequest) (types.ModelResponse, error) {
				return types.ModelResponse{}, providererror.FromError(errors.New("request timeout"))
			},
		}),
		"anthropic-rate-limit": anthropicmodel.NewClient(anthropicmodel.Config{
			GenerateFn: func(ctx context.Context, input string) (types.ModelResponse, error) {
				return types.ModelResponse{}, providererror.FromError(errors.New("429 rate limit"))
			},
		}),
		"gemini-auth": mustGeminiClient(t, geminimodel.Config{
			GenerateFn: func(ctx context.Context, input string) (types.ModelResponse, error) {
				return types.ModelResponse{}, providererror.FromError(errors.New("401 unauthorized"))
			},
		}),
	}
	for name, model := range cases {
		t.Run(name, func(t *testing.T) {
			eng := runner.New(model)
			res, runErr := eng.Run(context.Background(), types.RunRequest{Input: "hello"}, nil)
			if runErr == nil {
				t.Fatal("expected run error")
			}
			if res.Error == nil {
				t.Fatal("expected classified error in run result")
			}
			if name == "openai-timeout" && res.Error.Class != types.ErrPolicyTimeout {
				t.Fatalf("class=%q, want %q", res.Error.Class, types.ErrPolicyTimeout)
			}
			if name != "openai-timeout" && res.Error.Class != types.ErrModel {
				t.Fatalf("class=%q, want %q", res.Error.Class, types.ErrModel)
			}
		})
	}
}

func mustGeminiClient(t *testing.T, cfg geminimodel.Config) types.ModelClient {
	t.Helper()
	c, err := geminimodel.NewClient(context.Background(), cfg)
	if err != nil {
		t.Fatalf("new gemini client: %v", err)
	}
	return c
}
