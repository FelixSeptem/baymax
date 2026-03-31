package anthropic

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/FelixSeptem/baymax/core/types"
	providererror "github.com/FelixSeptem/baymax/model/providererror"
	"github.com/FelixSeptem/baymax/model/toolcontract"
	anthropic "github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	"github.com/anthropics/anthropic-sdk-go/packages/ssestream"
	"github.com/tidwall/gjson"
)

type Config struct {
	APIKey     string
	BaseURL    string
	Model      string
	MaxTokens  int64
	GenerateFn func(ctx context.Context, input string) (types.ModelResponse, error)
	StreamFn   func(ctx context.Context, input string) Stream
	DiscoverFn func(ctx context.Context, model string) (types.ProviderCapabilities, error)
}

type Client struct {
	model     string
	maxToken  int64
	sdk       anthropic.Client
	generate  func(ctx context.Context, input string) (types.ModelResponse, error)
	newStream func(ctx context.Context, input string) Stream
	discover  func(ctx context.Context, model string) (types.ProviderCapabilities, error)
}

type Stream interface {
	Next() bool
	Current() anthropic.MessageStreamEventUnion
	Err() error
	Close() error
}

type toolCallState struct {
	id       string
	name     string
	inputRaw string
	emitted  bool
}

type streamState struct {
	toolByIndex map[int64]*toolCallState
	toolSeq     int
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
		discover: cfg.DiscoverFn,
	}
	client.generate = client.generateWithSDK
	if cfg.GenerateFn != nil {
		client.generate = cfg.GenerateFn
	}
	client.newStream = func(ctx context.Context, input string) Stream {
		return client.sdk.Messages.NewStreaming(ctx, anthropic.MessageNewParams{
			Model:     anthropic.Model(client.model),
			MaxTokens: client.maxToken,
			Messages: []anthropic.MessageParam{
				anthropic.NewUserMessage(anthropic.NewTextBlock(input)),
			},
		})
	}
	if cfg.StreamFn != nil {
		client.newStream = cfg.StreamFn
	}
	if client.discover == nil {
		client.discover = client.discoverWithSDK
	}
	return client
}

func (c *Client) ProviderName() string {
	return "anthropic"
}

func (c *Client) Generate(ctx context.Context, req types.ModelRequest) (types.ModelResponse, error) {
	input, err := toolcontract.CanonicalInput(req)
	if err != nil {
		return types.ModelResponse{}, err
	}
	if input == "" {
		return types.ModelResponse{}, errors.New("model input is empty")
	}
	return c.generate(ctx, input)
}

func (c *Client) Stream(ctx context.Context, req types.ModelRequest, onEvent func(types.ModelEvent) error) error {
	input, err := toolcontract.CanonicalInput(req)
	if err != nil {
		return err
	}
	if input == "" {
		return errors.New("model input is empty")
	}
	stream := c.newStream(ctx, input)
	if stream == nil {
		return errors.New("anthropic stream is nil")
	}
	defer func() { _ = stream.Close() }()

	state := streamState{toolByIndex: map[int64]*toolCallState{}}
	completed := false
	for stream.Next() {
		events, err := mapStreamEvent(stream.Current(), &state)
		if err != nil {
			classified := providererror.FromError(err)
			if onEvent != nil {
				_ = onEvent(types.ModelEvent{Type: types.ModelEventTypeResponseError, Meta: map[string]any{"provider": "anthropic", "error": err.Error()}})
			}
			return classified
		}
		if onEvent == nil {
			continue
		}
		for _, ev := range events {
			if ev.Type == types.ModelEventTypeResponseCompleted {
				completed = true
			}
			if err := onEvent(ev); err != nil {
				return err
			}
		}
	}
	if err := stream.Err(); err != nil {
		if onEvent != nil {
			_ = onEvent(types.ModelEvent{Type: types.ModelEventTypeResponseError, Meta: map[string]any{"provider": "anthropic", "error": err.Error()}})
		}
		return providererror.FromError(err)
	}
	if onEvent != nil && !completed {
		if err := onEvent(types.ModelEvent{Type: types.ModelEventTypeResponseCompleted, Meta: map[string]any{"provider": "anthropic"}}); err != nil {
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

func (c *Client) CountTokens(ctx context.Context, req types.ModelRequest) (int, error) {
	model := strings.TrimSpace(req.Model)
	if model == "" {
		model = c.model
	}
	systemBlocks := make([]anthropic.TextBlockParam, 0, len(req.Messages))
	msgs := make([]anthropic.MessageParam, 0, len(req.Messages))
	for _, m := range req.Messages {
		role := strings.ToLower(strings.TrimSpace(m.Role))
		content := strings.TrimSpace(m.Content)
		if content == "" {
			continue
		}
		switch role {
		case "system":
			systemBlocks = append(systemBlocks, anthropic.TextBlockParam{Text: content})
		case "assistant":
			msgs = append(msgs, anthropic.NewAssistantMessage(anthropic.NewTextBlock(content)))
		default:
			msgs = append(msgs, anthropic.NewUserMessage(anthropic.NewTextBlock(content)))
		}
	}
	if input := strings.TrimSpace(req.Input); input != "" {
		msgs = append(msgs, anthropic.NewUserMessage(anthropic.NewTextBlock(input)))
	}
	if len(msgs) == 0 {
		return 0, errors.New("model input is empty")
	}
	resp, err := c.sdk.Messages.CountTokens(ctx, anthropic.MessageCountTokensParams{
		Model: anthropic.Model(model),
		System: anthropic.MessageCountTokensParamsSystemUnion{
			OfTextBlockArray: systemBlocks,
		},
		Messages: msgs,
	})
	if err != nil {
		return 0, providererror.FromError(err)
	}
	return int(resp.InputTokens), nil
}

func (c *Client) discoverWithSDK(ctx context.Context, model string) (types.ProviderCapabilities, error) {
	info, err := c.sdk.Models.Get(ctx, model, anthropic.ModelGetParams{})
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
	raw := info.RawJSON()
	switch {
	case strings.Contains(raw, `"tool_use":true`), strings.Contains(raw, `"supports_tool_use":true`), strings.Contains(raw, `"function_calling":true`):
		out.Support[types.ModelCapabilityToolCall] = types.CapabilitySupportSupported
	case strings.Contains(raw, `"tool_use":false`), strings.Contains(raw, `"supports_tool_use":false`), strings.Contains(raw, `"function_calling":false`):
		out.Support[types.ModelCapabilityToolCall] = types.CapabilitySupportUnsupported
	}
	return out, nil
}

func mapStreamEvent(ev anthropic.MessageStreamEventUnion, state *streamState) ([]types.ModelEvent, error) {
	events := make([]types.ModelEvent, 0, 2)
	meta := map[string]any{"provider": "anthropic", "event_type": ev.Type, "index": ev.Index}
	switch ev.Type {
	case "content_block_delta":
		delta := ev.Delta
		switch delta.Type {
		case "text_delta":
			if strings.TrimSpace(delta.Text) != "" {
				events = append(events, types.ModelEvent{Type: types.ModelEventTypeOutputTextDelta, TextDelta: delta.Text, Meta: meta})
			}
		case "input_json_delta":
			call := ensureToolState(state, ev.Index)
			call.inputRaw += delta.PartialJSON
		}
	case "content_block_start":
		block := ev.ContentBlock
		if block.Type != "tool_use" {
			break
		}
		call := ensureToolState(state, ev.Index)
		if block.ID != "" {
			call.id = block.ID
		}
		if block.Name != "" {
			call.name = block.Name
		}
		if block.Input != nil {
			raw, err := json.Marshal(block.Input)
			if err != nil {
				return nil, fmt.Errorf("marshal anthropic tool input: %w", err)
			}
			call.inputRaw = string(raw)
			if toolEvent, err := maybeEmitToolCall(call, ev.Index, state); err != nil {
				return nil, err
			} else if toolEvent != nil {
				events = append(events, *toolEvent)
			}
		}
	case "content_block_stop":
		call := state.toolByIndex[ev.Index]
		if call == nil {
			break
		}
		if toolEvent, err := maybeEmitToolCall(call, ev.Index, state); err != nil {
			return nil, err
		} else if toolEvent != nil {
			events = append(events, *toolEvent)
		}
	case "message_stop":
		events = append(events, types.ModelEvent{Type: types.ModelEventTypeResponseCompleted, Meta: meta})
	}
	return events, nil
}

func ensureToolState(state *streamState, index int64) *toolCallState {
	if call := state.toolByIndex[index]; call != nil {
		return call
	}
	state.toolSeq++
	call := &toolCallState{id: fmt.Sprintf("anthropic-tool-%d", state.toolSeq)}
	state.toolByIndex[index] = call
	return call
}

func maybeEmitToolCall(call *toolCallState, index int64, state *streamState) (*types.ModelEvent, error) {
	_ = index
	_ = state
	if call == nil || call.emitted {
		return nil, nil
	}
	if call.name == "" {
		return nil, nil
	}
	raw := strings.TrimSpace(call.inputRaw)
	if raw == "" {
		raw = "{}"
	}
	args := map[string]any{}
	if err := json.Unmarshal([]byte(raw), &args); err != nil {
		return nil, &providererror.Classified{
			Class:     types.ErrModel,
			Reason:    "request_invalid",
			Retryable: false,
			Cause:     fmt.Errorf("invalid anthropic tool arguments for %s: %w", call.id, err),
		}
	}
	call.emitted = true
	return &types.ModelEvent{
		Type: types.ModelEventTypeToolCall,
		ToolCall: &types.ToolCall{
			CallID: call.id,
			Name:   call.name,
			Args:   args,
		},
		Meta: map[string]any{
			"provider":     "anthropic",
			"tool_call_id": call.id,
			"tool_name":    call.name,
		},
	}, nil
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
var _ types.ModelCapabilityDiscovery = (*Client)(nil)

var _ Stream = (*ssestream.Stream[anthropic.MessageStreamEventUnion])(nil)
