package anthropic

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"github.com/FelixSeptem/baymax/core/types"
	providererror "github.com/FelixSeptem/baymax/model/providererror"
	anthropic "github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	"github.com/tidwall/gjson"
)

type Config struct {
	APIKey     string
	BaseURL    string
	Model      string
	MaxTokens  int64
	GenerateFn func(ctx context.Context, input string) (types.ModelResponse, error)
}

type Client struct {
	model    string
	maxToken int64
	sdk      anthropic.Client
	generate func(ctx context.Context, input string) (types.ModelResponse, error)
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
		model = "claude-3-5-sonnet-latest"
	}
	maxToken := cfg.MaxTokens
	if maxToken <= 0 {
		maxToken = 1024
	}

	client := &Client{
		model:    model,
		maxToken: maxToken,
		sdk:      anthropic.NewClient(opts...),
	}
	client.generate = client.generateWithSDK
	if cfg.GenerateFn != nil {
		client.generate = cfg.GenerateFn
	}
	return client
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
	_ = req
	_ = onEvent
	return errors.New("anthropic stream is not implemented in M1; TODO(r3-m2): add streaming alignment")
}

func (c *Client) generateWithSDK(ctx context.Context, input string) (types.ModelResponse, error) {
	msg, err := c.sdk.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     anthropic.Model(c.model),
		MaxTokens: c.maxToken,
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(input)),
		},
	})
	if err != nil {
		var apiErr *anthropic.Error
		if errors.As(err, &apiErr) {
			return types.ModelResponse{}, providererror.FromStatusCode(err, apiErr.StatusCode)
		}
		return types.ModelResponse{}, providererror.FromError(err)
	}
	return decodeMessage(msg), nil
}

func decodeMessage(msg any) types.ModelResponse {
	raw, _ := json.Marshal(msg)
	text := decodeFirstText(raw)
	in := int(gjson.GetBytes(raw, "usage.input_tokens").Int())
	out := int(gjson.GetBytes(raw, "usage.output_tokens").Int())
	total := in + out
	if candidate := int(gjson.GetBytes(raw, "usage.total_tokens").Int()); candidate > 0 {
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

func decodeFirstText(raw []byte) string {
	var builder strings.Builder
	gjson.GetBytes(raw, "content").ForEach(func(_, value gjson.Result) bool {
		text := value.Get("text").String()
		if strings.TrimSpace(text) == "" {
			return true
		}
		if builder.Len() > 0 {
			builder.WriteString("\n")
		}
		builder.WriteString(text)
		return true
	})
	return builder.String()
}

var _ types.ModelClient = (*Client)(nil)
