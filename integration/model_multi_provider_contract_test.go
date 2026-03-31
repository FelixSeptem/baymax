package integration

import (
	"context"
	"errors"
	"iter"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/FelixSeptem/baymax/core/runner"
	"github.com/FelixSeptem/baymax/core/types"
	"github.com/FelixSeptem/baymax/integration/fakes"
	anthropicmodel "github.com/FelixSeptem/baymax/model/anthropic"
	geminimodel "github.com/FelixSeptem/baymax/model/gemini"
	openaimodel "github.com/FelixSeptem/baymax/model/openai"
	providererror "github.com/FelixSeptem/baymax/model/providererror"
	runtimeconfig "github.com/FelixSeptem/baymax/runtime/config"
	"github.com/FelixSeptem/baymax/tool/local"
	anthropic "github.com/anthropics/anthropic-sdk-go"
	"google.golang.org/genai"
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

func TestModelProviderStreamingContract(t *testing.T) {
	openaiStep := 0
	anthropicStep := 0
	geminiStep := 0
	cases := map[string]types.ModelClient{
		"openai": openaimodel.NewClient(openaimodel.Config{
			StreamFn: func(ctx context.Context, req types.ModelRequest, onEvent func(types.ModelEvent) error) error {
				openaiStep++
				if openaiStep == 1 {
					if err := onEvent(types.ModelEvent{Type: types.ModelEventTypeToolCall, ToolCall: &types.ToolCall{
						CallID: "call-openai", Name: "local.weather", Args: map[string]any{"city": "shanghai"},
					}}); err != nil {
						return err
					}
					return onEvent(types.ModelEvent{Type: types.ModelEventTypeResponseCompleted})
				}
				if err := onEvent(types.ModelEvent{Type: types.ModelEventTypeOutputTextDelta, TextDelta: "he"}); err != nil {
					return err
				}
				if err := onEvent(types.ModelEvent{Type: types.ModelEventTypeOutputTextDelta, TextDelta: "llo"}); err != nil {
					return err
				}
				return onEvent(types.ModelEvent{Type: types.ModelEventTypeResponseCompleted})
			},
			DiscoverFn: func(ctx context.Context, model string) (types.ProviderCapabilities, error) {
				return supportedToolStreamingCapabilities("openai", model), nil
			},
		}),
		"anthropic": anthropicmodel.NewClient(anthropicmodel.Config{
			StreamFn: func(ctx context.Context, input string) anthropicmodel.Stream {
				anthropicStep++
				if anthropicStep == 1 {
					return &fakeAnthropicContractStream{events: []anthropic.MessageStreamEventUnion{
						{Type: "content_block_start", Index: 1, ContentBlock: anthropic.ContentBlockStartEventContentBlockUnion{Type: "tool_use", ID: "call-anthropic", Name: "local.weather"}},
						{Type: "content_block_delta", Index: 1, Delta: anthropic.MessageStreamEventUnionDelta{Type: "input_json_delta", PartialJSON: `{"city":"shanghai"}`}},
						{Type: "content_block_stop", Index: 1},
						{Type: "message_stop"},
					}}
				}
				return &fakeAnthropicContractStream{events: []anthropic.MessageStreamEventUnion{
					{Type: "content_block_delta", Index: 0, Delta: anthropic.MessageStreamEventUnionDelta{Type: "text_delta", Text: "he"}},
					{Type: "content_block_delta", Index: 0, Delta: anthropic.MessageStreamEventUnionDelta{Type: "text_delta", Text: "llo"}},
					{Type: "message_stop"},
				}}
			},
			DiscoverFn: func(ctx context.Context, model string) (types.ProviderCapabilities, error) {
				return supportedToolStreamingCapabilities("anthropic", model), nil
			},
		}),
		"gemini": mustGeminiClient(t, geminimodel.Config{
			StreamFn: func(ctx context.Context, input string) iter.Seq2[*genai.GenerateContentResponse, error] {
				geminiStep++
				if geminiStep == 1 {
					return seqGeminiChunks([]*genai.GenerateContentResponse{
						{Candidates: []*genai.Candidate{{Content: &genai.Content{Parts: []*genai.Part{{
							FunctionCall: &genai.FunctionCall{ID: "call-gemini", Name: "local.weather", Args: map[string]any{"city": "shanghai"}},
						}}}}}},
					}, nil)
				}
				return seqGeminiChunks([]*genai.GenerateContentResponse{
					{Candidates: []*genai.Candidate{{Content: &genai.Content{Parts: []*genai.Part{{Text: "he"}}}}}},
					{Candidates: []*genai.Candidate{{Content: &genai.Content{Parts: []*genai.Part{{Text: "llo"}}}}}},
				}, nil)
			},
			DiscoverFn: func(ctx context.Context, model string) (types.ProviderCapabilities, error) {
				return supportedToolStreamingCapabilities("gemini", model), nil
			},
		}),
	}

	for name, model := range cases {
		t.Run(name, func(t *testing.T) {
			reg := local.NewRegistry()
			if _, err := reg.Register(&fakes.Tool{
				NameValue:   "weather",
				SchemaValue: map[string]any{"type": "object"},
				InvokeFn: func(ctx context.Context, args map[string]any) (types.ToolResult, error) {
					return types.ToolResult{Content: "ok"}, nil
				},
			}); err != nil {
				t.Fatalf("register tool: %v", err)
			}
			collector := &eventCollector{}
			eng := runner.New(model, runner.WithLocalRegistry(reg))
			res, err := eng.Stream(context.Background(), types.RunRequest{Input: "hello"}, collector)
			if err != nil {
				t.Fatalf("Stream error: %v", err)
			}
			if res.FinalAnswer != "hello" {
				t.Fatalf("final answer=%q, want hello", res.FinalAnswer)
			}
			toolEvents := 0
			for _, ev := range collector.events {
				if ev.Type != "model.delta" {
					continue
				}
				if eventType, _ := ev.Payload["event_type"].(string); eventType == types.ModelEventTypeToolCall {
					toolEvents++
				}
			}
			if toolEvents != 1 {
				t.Fatalf("tool events=%d, want 1", toolEvents)
			}
		})
	}
}

func TestModelProviderStreamingFailFastClassification(t *testing.T) {
	cases := map[string]types.ModelClient{
		"openai-timeout": openaimodel.NewClient(openaimodel.Config{
			StreamFn: func(ctx context.Context, req types.ModelRequest, onEvent func(types.ModelEvent) error) error {
				return providererror.FromError(errors.New("request timeout"))
			},
			DiscoverFn: func(ctx context.Context, model string) (types.ProviderCapabilities, error) {
				return supportedToolStreamingCapabilities("openai", model), nil
			},
		}),
		"anthropic-timeout": anthropicmodel.NewClient(anthropicmodel.Config{
			StreamFn: func(ctx context.Context, input string) anthropicmodel.Stream {
				return &fakeAnthropicContractStream{err: errors.New("request timeout")}
			},
			DiscoverFn: func(ctx context.Context, model string) (types.ProviderCapabilities, error) {
				return supportedToolStreamingCapabilities("anthropic", model), nil
			},
		}),
		"gemini-timeout": mustGeminiClient(t, geminimodel.Config{
			StreamFn: func(ctx context.Context, input string) iter.Seq2[*genai.GenerateContentResponse, error] {
				return seqGeminiChunks(nil, errors.New("request timeout"))
			},
			DiscoverFn: func(ctx context.Context, model string) (types.ProviderCapabilities, error) {
				return supportedToolStreamingCapabilities("gemini", model), nil
			},
		}),
	}

	for name, model := range cases {
		t.Run(name, func(t *testing.T) {
			eng := runner.New(model)
			res, err := eng.Stream(context.Background(), types.RunRequest{Input: "hello"}, nil)
			if err == nil {
				t.Fatal("expected stream error")
			}
			if res.Error == nil || res.Error.Class != types.ErrPolicyTimeout {
				t.Fatalf("error=%+v, want %q", res.Error, types.ErrPolicyTimeout)
			}
		})
	}
}

func TestModelProviderStreamingMidStepFailureDoesNotSwitchProvider(t *testing.T) {
	cfgPath := filepath.Join(t.TempDir(), "runtime-stream-midstep-fallback.yaml")
	cfg := `
provider_fallback:
  enabled: true
  providers: [openai, anthropic]
`
	if err := os.WriteFile(cfgPath, []byte(strings.TrimSpace(cfg)), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	mgr, err := runtimeconfig.NewManager(runtimeconfig.ManagerOptions{FilePath: cfgPath, EnvPrefix: "BAYMAX"})
	if err != nil {
		t.Fatalf("new manager: %v", err)
	}
	t.Cleanup(func() { _ = mgr.Close() })

	anthropicCalled := 0
	openaiClient := openaimodel.NewClient(openaimodel.Config{
		StreamFn: func(ctx context.Context, req types.ModelRequest, onEvent func(types.ModelEvent) error) error {
			if err := onEvent(types.ModelEvent{Type: types.ModelEventTypeOutputTextDelta, TextDelta: "partial"}); err != nil {
				return err
			}
			return providererror.FromError(errors.New("stream transport exploded"))
		},
		DiscoverFn: func(ctx context.Context, model string) (types.ProviderCapabilities, error) {
			return supportedToolStreamingCapabilities("openai", model), nil
		},
	})
	anthropicClient := anthropicmodel.NewClient(anthropicmodel.Config{
		StreamFn: func(ctx context.Context, input string) anthropicmodel.Stream {
			anthropicCalled++
			return &fakeAnthropicContractStream{events: []anthropic.MessageStreamEventUnion{
				{Type: "content_block_delta", Index: 0, Delta: anthropic.MessageStreamEventUnionDelta{Type: "text_delta", Text: "fallback"}},
				{Type: "message_stop"},
			}}
		},
		DiscoverFn: func(ctx context.Context, model string) (types.ProviderCapabilities, error) {
			return supportedToolStreamingCapabilities("anthropic", model), nil
		},
	})

	eng := runner.New(
		openaiClient,
		runner.WithProviderModels("openai", map[string]types.ModelClient{
			"openai":    openaiClient,
			"anthropic": anthropicClient,
		}),
		runner.WithRuntimeManager(mgr),
	)
	res, streamErr := eng.Stream(context.Background(), types.RunRequest{Input: "hello"}, nil)
	if streamErr == nil {
		t.Fatal("expected mid-step stream failure")
	}
	if res.Error == nil || res.Error.Class != types.ErrModel {
		t.Fatalf("error=%#v, want ErrModel", res.Error)
	}
	if anthropicCalled != 0 {
		t.Fatalf("fallback provider should not be called mid-step, anthropicCalled=%d", anthropicCalled)
	}
}

func TestModelProviderCapabilityFallbackContract(t *testing.T) {
	cfgPath := filepath.Join(t.TempDir(), "runtime.yaml")
	cfg := `
provider_fallback:
  enabled: true
  providers: [openai, anthropic]
`
	if err := os.WriteFile(cfgPath, []byte(strings.TrimSpace(cfg)), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	mgr, err := runtimeconfig.NewManager(runtimeconfig.ManagerOptions{FilePath: cfgPath, EnvPrefix: "BAYMAX"})
	if err != nil {
		t.Fatalf("new manager: %v", err)
	}
	t.Cleanup(func() { _ = mgr.Close() })

	openaiClient := openaimodel.NewClient(openaimodel.Config{
		GenerateFn: func(ctx context.Context, req types.ModelRequest) (types.ModelResponse, error) {
			return types.ModelResponse{FinalAnswer: "should-not-reach"}, nil
		},
		DiscoverFn: func(ctx context.Context, model string) (types.ProviderCapabilities, error) {
			return types.ProviderCapabilities{
				Provider: "openai",
				Support: map[types.ModelCapability]types.CapabilitySupport{
					types.ModelCapabilityToolCall:  types.CapabilitySupportUnsupported,
					types.ModelCapabilityStreaming: types.CapabilitySupportSupported,
				},
			}, nil
		},
	})
	anthropicClient := anthropicmodel.NewClient(anthropicmodel.Config{
		GenerateFn: func(ctx context.Context, input string) (types.ModelResponse, error) {
			return types.ModelResponse{FinalAnswer: "ok-fallback"}, nil
		},
		DiscoverFn: func(ctx context.Context, model string) (types.ProviderCapabilities, error) {
			return types.ProviderCapabilities{
				Provider: "anthropic",
				Support: map[types.ModelCapability]types.CapabilitySupport{
					types.ModelCapabilityToolCall:  types.CapabilitySupportSupported,
					types.ModelCapabilityStreaming: types.CapabilitySupportSupported,
				},
			}, nil
		},
	})

	eng := runner.New(openaiClient,
		runner.WithProviderModels("openai", map[string]types.ModelClient{
			"openai":    openaiClient,
			"anthropic": anthropicClient,
		}),
		runner.WithRuntimeManager(mgr),
	)
	res, runErr := eng.Run(context.Background(), types.RunRequest{
		Input: "hello",
		Capabilities: types.CapabilityRequirements{
			Required: []types.ModelCapability{types.ModelCapabilityToolCall},
		},
	}, nil)
	if runErr != nil {
		t.Fatalf("Run error: %v", runErr)
	}
	if res.FinalAnswer != "ok-fallback" {
		t.Fatalf("final answer=%q, want ok-fallback", res.FinalAnswer)
	}
}

func TestModelProviderCapabilityFallbackFailFastContract(t *testing.T) {
	cfgPath := filepath.Join(t.TempDir(), "runtime.yaml")
	cfg := `
provider_fallback:
  enabled: true
  providers: [openai, anthropic]
`
	if err := os.WriteFile(cfgPath, []byte(strings.TrimSpace(cfg)), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	mgr, err := runtimeconfig.NewManager(runtimeconfig.ManagerOptions{FilePath: cfgPath, EnvPrefix: "BAYMAX"})
	if err != nil {
		t.Fatalf("new manager: %v", err)
	}
	t.Cleanup(func() { _ = mgr.Close() })

	unsupportedCaps := map[types.ModelCapability]types.CapabilitySupport{
		types.ModelCapabilityToolCall:  types.CapabilitySupportUnsupported,
		types.ModelCapabilityStreaming: types.CapabilitySupportSupported,
	}
	openaiClient := openaimodel.NewClient(openaimodel.Config{
		DiscoverFn: func(ctx context.Context, model string) (types.ProviderCapabilities, error) {
			return types.ProviderCapabilities{Provider: "openai", Support: unsupportedCaps}, nil
		},
	})
	anthropicClient := anthropicmodel.NewClient(anthropicmodel.Config{
		DiscoverFn: func(ctx context.Context, model string) (types.ProviderCapabilities, error) {
			return types.ProviderCapabilities{Provider: "anthropic", Support: unsupportedCaps}, nil
		},
	})
	eng := runner.New(openaiClient,
		runner.WithProviderModels("openai", map[string]types.ModelClient{
			"openai":    openaiClient,
			"anthropic": anthropicClient,
		}),
		runner.WithRuntimeManager(mgr),
	)
	res, runErr := eng.Run(context.Background(), types.RunRequest{
		Input: "hello",
		Capabilities: types.CapabilityRequirements{
			Required: []types.ModelCapability{types.ModelCapabilityToolCall},
		},
	}, nil)
	if runErr == nil {
		t.Fatal("expected fail-fast error")
	}
	if res.Error == nil || res.Error.Class != types.ErrModel {
		t.Fatalf("error=%+v, want %q", res.Error, types.ErrModel)
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

type fakeAnthropicContractStream struct {
	events []anthropic.MessageStreamEventUnion
	err    error
	index  int
}

func (s *fakeAnthropicContractStream) Next() bool {
	if s.index >= len(s.events) {
		return false
	}
	s.index++
	return true
}

func (s *fakeAnthropicContractStream) Current() anthropic.MessageStreamEventUnion {
	return s.events[s.index-1]
}

func (s *fakeAnthropicContractStream) Err() error   { return s.err }
func (s *fakeAnthropicContractStream) Close() error { return nil }

func seqGeminiChunks(chunks []*genai.GenerateContentResponse, tailErr error) iter.Seq2[*genai.GenerateContentResponse, error] {
	return func(yield func(*genai.GenerateContentResponse, error) bool) {
		for _, chunk := range chunks {
			if !yield(chunk, nil) {
				return
			}
		}
		if tailErr != nil {
			_ = yield(nil, tailErr)
		}
	}
}

func supportedToolStreamingCapabilities(provider string, model string) types.ProviderCapabilities {
	return types.ProviderCapabilities{
		Provider: provider,
		Model:    model,
		Support: map[types.ModelCapability]types.CapabilitySupport{
			types.ModelCapabilityStreaming: types.CapabilitySupportSupported,
			types.ModelCapabilityToolCall:  types.CapabilitySupportSupported,
		},
		Source: "integration-test",
	}
}
