package gemini

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"github.com/FelixSeptem/baymax/core/types"
	providererror "github.com/FelixSeptem/baymax/model/providererror"
	"github.com/tidwall/gjson"
	"google.golang.org/genai"
)

type Config struct {
	APIKey     string
	Model      string
	GenerateFn func(ctx context.Context, input string) (types.ModelResponse, error)
}

type Client struct {
	model    string
	sdk      *genai.Client
	generate func(ctx context.Context, input string) (types.ModelResponse, error)
}

func NewClient(ctx context.Context, cfg Config) (*Client, error) {
	model := cfg.Model
	if model == "" {
		model = "gemini-2.5-flash"
	}
	if cfg.GenerateFn != nil {
		return &Client{
			model:    model,
			generate: cfg.GenerateFn,
		}, nil
	}
	sdk, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:  cfg.APIKey,
		Backend: genai.BackendGeminiAPI,
	})
	if err != nil {
		return nil, providererror.FromError(err)
	}
	client := &Client{
		model: model,
		sdk:   sdk,
	}
	client.generate = client.generateWithSDK
	if cfg.GenerateFn != nil {
		client.generate = cfg.GenerateFn
	}
	return client, nil
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
	_ = ctx
	_ = req
	_ = onEvent
	return errors.New("gemini stream is not implemented in M1; TODO(r3-m2): add streaming alignment")
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
