package openai

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
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
	sdk         openai.Client
	model       string
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
		sdk:   sdkClient,
		model: model,
	}
	client.newResponse = func(ctx context.Context, params responses.ResponseNewParams) (*responses.Response, error) {
		return client.sdk.Responses.New(ctx, params)
	}
	client.newStream = func(ctx context.Context, params responses.ResponseNewParams) responseStream {
		return client.sdk.Responses.NewStreaming(ctx, params)
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
	input := strings.TrimSpace(req.Input)
	if input == "" && len(req.Messages) > 0 {
		input = req.Messages[len(req.Messages)-1].Content
	}
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
		return err
	}
	return ctx.Err()
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
	meta := map[string]any{
		"openai_event_type": ev.Type,
		"sequence_number":   ev.SequenceNumber,
		"output_index":      ev.OutputIndex,
	}
	if ev.ItemID != "" {
		meta["item_id"] = ev.ItemID
	}

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
	var args map[string]any
	if err := json.Unmarshal([]byte(raw), &args); err != nil {
		return nil, fmt.Errorf("invalid tool call arguments for %s: %w", call.callID, err)
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
		Meta: map[string]any{
			"item_id": itemID,
		},
	}, nil
}

var _ types.ModelClient = (*Client)(nil)
