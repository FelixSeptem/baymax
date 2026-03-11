package openai

import (
	"context"
	"errors"
	"strings"

	"github.com/FelixSeptem/baymax/core/types"
	openai "github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
	"github.com/openai/openai-go/responses"
)

type Config struct {
	APIKey  string
	BaseURL string
	Model   string
}

type Client struct {
	sdk   openai.Client
	model string
}

func NewClient(cfg Config) *Client {
	opts := make([]option.RequestOption, 0, 2)
	if cfg.APIKey != "" {
		opts = append(opts, option.WithAPIKey(cfg.APIKey))
	}
	if cfg.BaseURL != "" {
		opts = append(opts, option.WithBaseURL(cfg.BaseURL))
	}
	model := cfg.Model
	if model == "" {
		model = "gpt-4.1-mini"
	}
	return &Client{sdk: openai.NewClient(opts...), model: model}
}

func (c *Client) Generate(ctx context.Context, req types.ModelRequest) (types.ModelResponse, error) {
	input := strings.TrimSpace(req.Input)
	if input == "" && len(req.Messages) > 0 {
		input = req.Messages[len(req.Messages)-1].Content
	}
	if input == "" {
		return types.ModelResponse{}, errors.New("model input is empty")
	}

	resp, err := c.sdk.Responses.New(ctx, responses.ResponseNewParams{
		Model: responses.ResponsesModel(c.model),
		Input: responses.ResponseNewParamsInputUnion{OfString: openai.String(input)},
	})
	if err != nil {
		return types.ModelResponse{}, err
	}

	return types.ModelResponse{
		FinalAnswer: resp.OutputText(),
		Usage: types.TokenUsage{
			InputTokens:  int(resp.Usage.InputTokens),
			OutputTokens: int(resp.Usage.OutputTokens),
			TotalTokens:  int(resp.Usage.TotalTokens),
		},
	}, nil
}

func (c *Client) Stream(ctx context.Context, req types.ModelRequest, onEvent func(types.ModelEvent) error) error {
	res, err := c.Generate(ctx, req)
	if err != nil {
		return err
	}
	if res.FinalAnswer != "" && onEvent != nil {
		if err := onEvent(types.ModelEvent{Type: "final_answer", TextDelta: res.FinalAnswer}); err != nil {
			return err
		}
	}
	return nil
}

var _ types.ModelClient = (*Client)(nil)
