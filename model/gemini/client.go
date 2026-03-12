package gemini

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"iter"
	"slices"
	"strings"
	"time"

	"github.com/FelixSeptem/baymax/core/types"
	providererror "github.com/FelixSeptem/baymax/model/providererror"
	"github.com/tidwall/gjson"
	"google.golang.org/genai"
)

type Config struct {
	APIKey     string
	Model      string
	GenerateFn func(ctx context.Context, input string) (types.ModelResponse, error)
	StreamFn   func(ctx context.Context, input string) iter.Seq2[*genai.GenerateContentResponse, error]
	DiscoverFn func(ctx context.Context, model string) (types.ProviderCapabilities, error)
}

type Client struct {
	model    string
	sdk      *genai.Client
	generate func(ctx context.Context, input string) (types.ModelResponse, error)
	stream   func(ctx context.Context, input string) iter.Seq2[*genai.GenerateContentResponse, error]
	discover func(ctx context.Context, model string) (types.ProviderCapabilities, error)
}

func NewClient(ctx context.Context, cfg Config) (*Client, error) {
	model := cfg.Model
	if model == "" {
		model = "gemini-2.5-flash"
	}
	if cfg.GenerateFn != nil || cfg.StreamFn != nil {
		client := &Client{model: model}
		if cfg.GenerateFn != nil {
			client.generate = cfg.GenerateFn
		} else {
			client.generate = func(ctx context.Context, input string) (types.ModelResponse, error) {
				_ = ctx
				_ = input
				return types.ModelResponse{}, errors.New("gemini generate is not configured")
			}
		}
		if cfg.StreamFn != nil {
			client.stream = cfg.StreamFn
		} else {
			client.stream = func(ctx context.Context, input string) iter.Seq2[*genai.GenerateContentResponse, error] {
				_ = ctx
				_ = input
				return func(yield func(*genai.GenerateContentResponse, error) bool) {
					_ = yield(nil, errors.New("gemini stream is not configured"))
				}
			}
		}
		if cfg.DiscoverFn != nil {
			client.discover = cfg.DiscoverFn
		} else {
			client.discover = func(ctx context.Context, model string) (types.ProviderCapabilities, error) {
				_ = ctx
				_ = model
				return types.ProviderCapabilities{
					Provider:  "gemini",
					Model:     client.model,
					Source:    "sdk.unavailable",
					CheckedAt: time.Now(),
					Support: map[types.ModelCapability]types.CapabilitySupport{
						types.ModelCapabilityStreaming: types.CapabilitySupportSupported,
						types.ModelCapabilityToolCall:  types.CapabilitySupportUnknown,
					},
				}, nil
			}
		}
		return client, nil
	}

	sdk, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:  cfg.APIKey,
		Backend: genai.BackendGeminiAPI,
	})
	if err != nil {
		return nil, providererror.FromError(err)
	}
	client := &Client{
		model:    model,
		sdk:      sdk,
		discover: cfg.DiscoverFn,
	}
	client.generate = client.generateWithSDK
	if cfg.GenerateFn != nil {
		client.generate = cfg.GenerateFn
	}
	client.stream = client.streamWithSDK
	if cfg.StreamFn != nil {
		client.stream = cfg.StreamFn
	}
	if client.discover == nil {
		client.discover = client.discoverWithSDK
	}
	return client, nil
}

func (c *Client) ProviderName() string {
	return "gemini"
}

func (c *Client) Generate(ctx context.Context, req types.ModelRequest) (types.ModelResponse, error) {
	input := strings.TrimSpace(req.Input)
	if input == "" && len(req.Messages) > 0 {
		input = req.Messages[len(req.Messages)-1].Content
	}
	if input == "" {
		return types.ModelResponse{}, errors.New("model input is empty")
	}
	return c.generate(ctx, input)
}

func (c *Client) Stream(ctx context.Context, req types.ModelRequest, onEvent func(types.ModelEvent) error) error {
	input := strings.TrimSpace(req.Input)
	if input == "" && len(req.Messages) > 0 {
		input = req.Messages[len(req.Messages)-1].Content
	}
	if input == "" {
		return errors.New("model input is empty")
	}

	toolSeq := 0
	for chunk, err := range c.stream(ctx, input) {
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			if onEvent != nil {
				_ = onEvent(types.ModelEvent{Type: types.ModelEventTypeResponseError, Meta: map[string]any{"provider": "gemini", "error": err.Error()}})
			}
			return providererror.FromError(err)
		}
		mapped := mapStreamChunk(chunk, &toolSeq)
		if onEvent == nil {
			continue
		}
		for _, ev := range mapped {
			if err := onEvent(ev); err != nil {
				return err
			}
		}
	}
	if onEvent != nil {
		if err := onEvent(types.ModelEvent{Type: types.ModelEventTypeResponseCompleted, Meta: map[string]any{"provider": "gemini"}}); err != nil {
			return err
		}
	}
	return ctx.Err()
}

func (c *Client) DiscoverCapabilities(ctx context.Context, req types.ModelRequest) (types.ProviderCapabilities, error) {
	model := strings.TrimSpace(req.Model)
	if model == "" {
		model = c.model
	}
	return c.discover(ctx, model)
}

func (c *Client) discoverWithSDK(ctx context.Context, model string) (types.ProviderCapabilities, error) {
	m, err := c.sdk.Models.Get(ctx, model, nil)
	if err != nil {
		return types.ProviderCapabilities{}, providererror.FromError(err)
	}
	out := types.ProviderCapabilities{
		Provider:  c.ProviderName(),
		Model:     model,
		Source:    "sdk.models.get",
		CheckedAt: time.Now(),
		Support: map[types.ModelCapability]types.CapabilitySupport{
			types.ModelCapabilityStreaming: types.CapabilitySupportSupported,
			types.ModelCapabilityToolCall:  types.CapabilitySupportUnknown,
		},
	}
	for _, action := range m.SupportedActions {
		act := strings.ToLower(strings.TrimSpace(action))
		if strings.Contains(act, "stream") {
			out.Support[types.ModelCapabilityStreaming] = types.CapabilitySupportSupported
		}
		if strings.Contains(act, "tool") || strings.Contains(act, "function") {
			out.Support[types.ModelCapabilityToolCall] = types.CapabilitySupportSupported
		}
	}
	if slices.Contains(m.SupportedActions, "generateContent") && out.Support[types.ModelCapabilityStreaming] == types.CapabilitySupportUnknown {
		out.Support[types.ModelCapabilityStreaming] = types.CapabilitySupportSupported
	}
	return out, nil
}

func mapStreamChunk(resp *genai.GenerateContentResponse, toolSeq *int) []types.ModelEvent {
	if resp == nil {
		return nil
	}
	events := make([]types.ModelEvent, 0, 2)
	for _, candidate := range resp.Candidates {
		if candidate == nil || candidate.Content == nil {
			continue
		}
		for _, part := range candidate.Content.Parts {
			if part == nil {
				continue
			}
			if strings.TrimSpace(part.Text) != "" {
				events = append(events, types.ModelEvent{
					Type:      types.ModelEventTypeOutputTextDelta,
					TextDelta: part.Text,
					Meta:      map[string]any{"provider": "gemini"},
				})
			}
			if part.FunctionCall != nil && part.FunctionCall.Name != "" {
				(*toolSeq)++
				callID := strings.TrimSpace(part.FunctionCall.ID)
				if callID == "" {
					callID = fmt.Sprintf("gemini-tool-%d", *toolSeq)
				}
				args := part.FunctionCall.Args
				if args == nil {
					args = map[string]any{}
				}
				events = append(events, types.ModelEvent{
					Type: types.ModelEventTypeToolCall,
					ToolCall: &types.ToolCall{
						CallID: callID,
						Name:   part.FunctionCall.Name,
						Args:   args,
					},
					Meta: map[string]any{"provider": "gemini"},
				})
			}
		}
	}
	return events
}

func (c *Client) streamWithSDK(ctx context.Context, input string) iter.Seq2[*genai.GenerateContentResponse, error] {
	return c.sdk.Models.GenerateContentStream(ctx, c.model, genai.Text(input), nil)
}

func (c *Client) generateWithSDK(ctx context.Context, input string) (types.ModelResponse, error) {
	resp, err := c.sdk.Models.GenerateContent(ctx, c.model, genai.Text(input), nil)
	if err != nil {
		return types.ModelResponse{}, providererror.FromError(err)
	}
	return decodeGenerateResponse(resp), nil
}

func decodeGenerateResponse(resp any) types.ModelResponse {
	raw, _ := json.Marshal(resp)
	text := strings.TrimSpace(gjson.GetBytes(raw, "text").String())
	if text == "" {
		text = decodeCandidateText(raw)
	}
	in := int(gjson.GetBytes(raw, "usage_metadata.prompt_token_count").Int())
	out := int(gjson.GetBytes(raw, "usage_metadata.candidates_token_count").Int())
	total := in + out
	if candidate := int(gjson.GetBytes(raw, "usage_metadata.total_token_count").Int()); candidate > 0 {
		total = candidate
	}
	return types.ModelResponse{
		FinalAnswer: text,
		Usage: types.TokenUsage{
			InputTokens:  in,
			OutputTokens: out,
			TotalTokens:  total,
		},
	}
}

func decodeCandidateText(raw []byte) string {
	var builder strings.Builder
	gjson.GetBytes(raw, "candidates").ForEach(func(_, candidate gjson.Result) bool {
		parts := candidate.Get("content.parts")
		parts.ForEach(func(_, part gjson.Result) bool {
			text := strings.TrimSpace(part.Get("text").String())
			if text == "" {
				return true
			}
			if builder.Len() > 0 {
				builder.WriteString("\n")
			}
			builder.WriteString(text)
			return true
		})
		return true
	})
	return builder.String()
}

var _ types.ModelClient = (*Client)(nil)
var _ types.ModelCapabilityDiscovery = (*Client)(nil)
