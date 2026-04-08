package openai

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/FelixSeptem/baymax/core/types"
	providererror "github.com/FelixSeptem/baymax/model/providererror"
	"github.com/FelixSeptem/baymax/model/toolcontract"
	openai "github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
	"github.com/openai/openai-go/responses"
)

type Config struct {
	APIKey     string
	BaseURL    string
	Model      string
	GenerateFn func(context.Context, types.ModelRequest) (types.ModelResponse, error)
	StreamFn   func(context.Context, types.ModelRequest, func(types.ModelEvent) error) error
	DiscoverFn func(context.Context, string) (types.ProviderCapabilities, error)
}

type Client struct {
	sdk         openai.Client
	model       string
	generateFn  func(context.Context, types.ModelRequest) (types.ModelResponse, error)
	streamFn    func(context.Context, types.ModelRequest, func(types.ModelEvent) error) error
	discoverFn  func(context.Context, string) (types.ProviderCapabilities, error)
	newResponse func(context.Context, responses.ResponseNewParams) (*responses.Response, error)
	newStream   func(context.Context, responses.ResponseNewParams) responseStream
}

type responseStream interface {
	Next() bool
	Current() responses.ResponseStreamEventUnion
	Err() error
	Close() error
}

type toolCallState struct {
	itemID    string
	callID    string
	name      string
	arguments string
	argsReady bool
	emitted   bool
}

type streamState struct {
	toolCalls map[string]*toolCallState
}

const maxToolCallArgsDecodeBufferCap = 64 * 1024

var openAIToolArgsDecodeBufferPool = sync.Pool{
	New: func() any {
		buf := make([]byte, 0, 256)
		return &buf
	},
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
	sdkClient := openai.NewClient(opts...)
	client := &Client{
		sdk:        sdkClient,
		model:      model,
		generateFn: cfg.GenerateFn,
		streamFn:   cfg.StreamFn,
		discoverFn: cfg.DiscoverFn,
	}
	client.newResponse = func(ctx context.Context, params responses.ResponseNewParams) (*responses.Response, error) {
		return client.sdk.Responses.New(ctx, params)
	}
	client.newStream = func(ctx context.Context, params responses.ResponseNewParams) responseStream {
		return client.sdk.Responses.NewStreaming(ctx, params)
	}
	if client.discoverFn == nil {
		client.discoverFn = client.discoverWithSDK
	}
	return client
}

func (c *Client) ProviderName() string {
	return "openai"
}

func (c *Client) Generate(ctx context.Context, req types.ModelRequest) (types.ModelResponse, error) {
	normalizedReq, err := toolcontract.WithCanonicalInput(req)
	if err != nil {
		return types.ModelResponse{}, err
	}
	req = normalizedReq
	if c.generateFn != nil {
		return c.generateFn(ctx, req)
	}
	input := strings.TrimSpace(req.Input)
	if input == "" {
		return types.ModelResponse{}, errors.New("model input is empty")
	}

	resp, err := c.newResponse(ctx, responses.ResponseNewParams{
		Model: c.model,
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
	normalizedReq, err := toolcontract.WithCanonicalInput(req)
	if err != nil {
		return err
	}
	req = normalizedReq
	if c.streamFn != nil {
		return c.streamFn(ctx, req, onEvent)
	}
	input := strings.TrimSpace(req.Input)
	if input == "" {
		return errors.New("model input is empty")
	}

	stream := c.newStream(ctx, responses.ResponseNewParams{
		Model: c.model,
		Input: responses.ResponseNewParamsInputUnion{OfString: openai.String(input)},
	})
	if stream == nil {
		return errors.New("openai stream is nil")
	}
	defer func() {
		_ = stream.Close()
	}()

	state := streamState{toolCalls: map[string]*toolCallState{}}
	for stream.Next() {
		mapped, err := mapStreamEvent(stream.Current(), &state)
		if err != nil {
			var classified *providererror.Classified
			if errors.As(err, &classified) {
				return classified
			}
			if onEvent != nil {
				_ = onEvent(types.ModelEvent{
					Type: types.ModelEventTypeResponseError,
					Meta: openAIErrorMeta(err.Error()),
				})
			}
			return err
		}
		if onEvent == nil {
			continue
		}
		for _, ev := range mapped {
			if err := onEvent(ev); err != nil {
				return err
			}
		}
	}
	if err := stream.Err(); err != nil {
		if onEvent != nil {
			_ = onEvent(types.ModelEvent{
				Type: types.ModelEventTypeResponseError,
				Meta: openAIErrorMeta(err.Error()),
			})
		}
		return err
	}
	return ctx.Err()
}

func (c *Client) DiscoverCapabilities(ctx context.Context, req types.ModelRequest) (types.ProviderCapabilities, error) {
	model := strings.TrimSpace(req.Model)
	if model == "" {
		model = c.model
	}
	return c.discoverFn(ctx, model)
}

func (c *Client) CountTokens(ctx context.Context, req types.ModelRequest) (int, error) {
	_ = ctx
	_ = req
	return 0, errors.New("openai official sdk does not provide token count api in this adapter")
}

func (c *Client) discoverWithSDK(ctx context.Context, model string) (types.ProviderCapabilities, error) {
	info, err := c.sdk.Models.Get(ctx, model)
	if err != nil {
		return types.ProviderCapabilities{}, err
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
	case strings.Contains(raw, `"supports_tool_calls":true`), strings.Contains(raw, `"tool_calls":true`), strings.Contains(raw, `"function_calling":true`):
		out.Support[types.ModelCapabilityToolCall] = types.CapabilitySupportSupported
	case strings.Contains(raw, `"supports_tool_calls":false`), strings.Contains(raw, `"tool_calls":false`), strings.Contains(raw, `"function_calling":false`):
		out.Support[types.ModelCapabilityToolCall] = types.CapabilitySupportUnsupported
	}
	return out, nil
}

func mapStreamEvent(ev responses.ResponseStreamEventUnion, state *streamState) ([]types.ModelEvent, error) {
	switch ev.Type {
	case "error":
		msg := strings.TrimSpace(ev.Message)
		if msg == "" {
			msg = "openai stream error event"
		}
		return nil, errors.New(msg)
	case "response.failed":
		return nil, errors.New("openai response failed")
	case "response.incomplete":
		return nil, errors.New("openai response incomplete")
	}

	events := make([]types.ModelEvent, 0, 2)
	meta := openAIStreamMeta(ev)

	switch ev.Type {
	case "response.output_text.delta":
		delta := ev.Delta.OfString
		events = append(events, types.ModelEvent{Type: ev.Type, TextDelta: delta, Meta: meta})
	case "response.output_text.done":
		meta["text"] = ev.Text
		events = append(events, types.ModelEvent{Type: ev.Type, Meta: meta})
	case "response.function_call_arguments.delta":
		delta := ev.Delta.OfString
		call := ensureToolCall(state, ev.ItemID)
		call.arguments += delta
		events = append(events, types.ModelEvent{Type: ev.Type, Meta: meta})
	case "response.function_call_arguments.done":
		call := ensureToolCall(state, ev.ItemID)
		call.arguments = ev.Arguments
		call.argsReady = true
		events = append(events, types.ModelEvent{Type: ev.Type, Meta: meta})
		if toolEvent, err := maybeEmitToolCall(state, ev.ItemID); err != nil {
			return nil, err
		} else if toolEvent != nil {
			events = append(events, *toolEvent)
		}
	case "response.output_item.added", "response.output_item.done":
		updateToolCallFromItem(state, ev.ItemID, ev.Item)
		if toolEvent, err := maybeEmitToolCall(state, ev.ItemID); err != nil {
			return nil, err
		} else if toolEvent != nil {
			events = append(events, *toolEvent)
		}
		events = append(events, types.ModelEvent{Type: ev.Type, Meta: meta})
	default:
		events = append(events, types.ModelEvent{Type: ev.Type, Meta: meta})
	}

	return events, nil
}

func ensureToolCall(state *streamState, itemID string) *toolCallState {
	call, ok := state.toolCalls[itemID]
	if ok {
		return call
	}
	call = &toolCallState{itemID: itemID}
	state.toolCalls[itemID] = call
	return call
}

func updateToolCallFromItem(state *streamState, itemID string, item responses.ResponseOutputItemUnion) {
	if item.Type != "function_call" {
		return
	}
	call := ensureToolCall(state, itemID)
	if item.CallID != "" {
		call.callID = item.CallID
	}
	if item.Name != "" {
		call.name = item.Name
	}
	if item.Arguments != "" {
		call.arguments = item.Arguments
		call.argsReady = true
	}
}

func maybeEmitToolCall(state *streamState, itemID string) (*types.ModelEvent, error) {
	call, ok := state.toolCalls[itemID]
	if !ok || call.emitted {
		return nil, nil
	}
	if call.callID == "" || call.name == "" || !call.argsReady {
		return nil, nil
	}
	raw := strings.TrimSpace(call.arguments)
	if raw == "" {
		raw = "{}"
	}
	args, err := decodeOpenAIToolCallArgs(raw)
	if err != nil {
		return nil, &providererror.Classified{
			Class:     types.ErrModel,
			Reason:    "request_invalid",
			Retryable: false,
			Cause:     fmt.Errorf("invalid tool call arguments for %s: %w", call.callID, err),
		}
	}
	call.emitted = true
	toolCall := types.ToolCall{
		CallID: call.callID,
		Name:   call.name,
		Args:   args,
	}
	return &types.ModelEvent{
		Type:     "tool_call",
		ToolCall: &toolCall,
		Meta:     openAIToolCallMeta(itemID, toolCall.CallID, toolCall.Name),
	}, nil
}

func openAIErrorMeta(message string) map[string]any {
	meta := make(map[string]any, 2)
	meta["provider"] = "openai"
	meta["error"] = message
	return meta
}

func openAIStreamMeta(ev responses.ResponseStreamEventUnion) map[string]any {
	size := 3
	if ev.ItemID != "" {
		size++
	}
	meta := make(map[string]any, size)
	meta["openai_event_type"] = ev.Type
	meta["sequence_number"] = ev.SequenceNumber
	meta["output_index"] = ev.OutputIndex
	if ev.ItemID != "" {
		meta["item_id"] = ev.ItemID
	}
	return meta
}

func openAIToolCallMeta(itemID, callID, name string) map[string]any {
	meta := make(map[string]any, 4)
	meta["provider"] = "openai"
	meta["item_id"] = itemID
	meta["tool_call_id"] = callID
	meta["tool_name"] = name
	return meta
}

func decodeOpenAIToolCallArgs(raw string) (map[string]any, error) {
	if raw == "{}" {
		return map[string]any{}, nil
	}
	bufPtr := openAIToolArgsDecodeBufferPool.Get().(*[]byte)
	buf := (*bufPtr)[:0]
	buf = append(buf, raw...)

	var args map[string]any
	err := json.Unmarshal(buf, &args)

	if cap(buf) > maxToolCallArgsDecodeBufferCap {
		*bufPtr = make([]byte, 0, 256)
	} else {
		*bufPtr = buf[:0]
	}
	openAIToolArgsDecodeBufferPool.Put(bufPtr)

	if err != nil {
		return nil, err
	}
	return args, nil
}

var _ types.ModelClient = (*Client)(nil)
var _ types.ModelCapabilityDiscovery = (*Client)(nil)
