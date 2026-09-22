package toolcontract

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/FelixSeptem/baymax/core/types"
	providererror "github.com/FelixSeptem/baymax/model/providererror"
)

const FeedbackHeader = "[tool_result_feedback.v1]"
const MaxCanonicalFeedbackBytes = 64 * 1024

// InterpretedRequest is the SDK-neutral, source-preserving request view used
// by provider-owned request builders. It deliberately contains core request
// types only: provider SDK request shapes remain private to model/<provider>.
type InterpretedRequest struct {
	Messages    []types.Message
	ToolResults []types.ToolCallOutcome
}

type feedbackEnvelopeItem struct {
	ToolCallID string             `json:"tool_call_id"`
	ToolName   string             `json:"tool_name"`
	Content    string             `json:"content,omitempty"`
	Structured map[string]any     `json:"structured,omitempty"`
	Error      *feedbackErrorItem `json:"error,omitempty"`
}

type feedbackErrorItem struct {
	Class   string         `json:"class,omitempty"`
	Message string         `json:"message"`
	Details map[string]any `json:"details,omitempty"`
}

// InterpretRequest normalizes source request facts without projecting them to
// a provider protocol. Non-empty messages retain their source order; a
// non-empty Input is appended once as the final user message. Valid tool
// results retain their call/name association for native provider mapping.
func InterpretRequest(req types.ModelRequest) (InterpretedRequest, error) {
	interpreted := InterpretedRequest{
		Messages:    make([]types.Message, 0, len(req.Messages)+1),
		ToolResults: make([]types.ToolCallOutcome, 0, len(req.ToolResult)),
	}
	for _, message := range req.Messages {
		content := strings.TrimSpace(message.Content)
		if content == "" {
			continue
		}
		interpreted.Messages = append(interpreted.Messages, types.Message{
			Role:    strings.TrimSpace(message.Role),
			Content: content,
		})
	}
	if input := strings.TrimSpace(req.Input); input != "" {
		interpreted.Messages = append(interpreted.Messages, types.Message{Role: "user", Content: input})
	}

	items := make([]feedbackEnvelopeItem, 0, len(req.ToolResult))
	for _, outcome := range req.ToolResult {
		callID := strings.TrimSpace(outcome.CallID)
		name := strings.TrimSpace(outcome.Name)
		if callID == "" || name == "" {
			return InterpretedRequest{}, invalidFeedbackError(callID, name)
		}
		outcome.CallID = callID
		outcome.Name = name
		outcome.Result.Structured = cloneAnyMap(outcome.Result.Structured)
		if outcome.Result.Error != nil {
			cloned := *outcome.Result.Error
			cloned.Details = cloneAnyMap(cloned.Details)
			outcome.Result.Error = &cloned
		}
		interpreted.ToolResults = append(interpreted.ToolResults, outcome)
		items = append(items, feedbackEnvelopeFromOutcome(outcome))
	}
	if err := validateFeedbackSize(items); err != nil {
		return InterpretedRequest{}, err
	}
	return interpreted, nil
}

func invalidFeedbackError(callID, name string) error {
	return &providererror.Classified{
		Class:     types.ErrModel,
		Reason:    "feedback_invalid",
		Retryable: false,
		Cause: fmt.Errorf(
			"tool result feedback requires non-empty call_id and tool_name, got call_id=%q tool_name=%q",
			callID,
			name,
		),
	}
}

func feedbackEnvelopeFromOutcome(outcome types.ToolCallOutcome) feedbackEnvelopeItem {
	item := feedbackEnvelopeItem{
		ToolCallID: outcome.CallID,
		ToolName:   outcome.Name,
	}
	if strings.TrimSpace(outcome.Result.Content) != "" {
		item.Content = outcome.Result.Content
	}
	if len(outcome.Result.Structured) > 0 {
		item.Structured = cloneAnyMap(outcome.Result.Structured)
	}
	if outcome.Result.Error != nil {
		item.Error = &feedbackErrorItem{
			Class:   string(outcome.Result.Error.Class),
			Message: strings.TrimSpace(outcome.Result.Error.Message),
			Details: cloneAnyMap(outcome.Result.Error.Details),
		}
	}
	return item
}

func validateFeedbackSize(items []feedbackEnvelopeItem) error {
	blob, err := json.Marshal(items)
	if err != nil {
		return &providererror.Classified{
			Class:     types.ErrModel,
			Reason:    "feedback_invalid",
			Retryable: false,
			Cause:     fmt.Errorf("marshal canonical tool result feedback: %w", err),
		}
	}
	if len(blob) > MaxCanonicalFeedbackBytes {
		return &providererror.Classified{Class: types.ErrModel, Reason: "overflow", Retryable: false, Cause: fmt.Errorf("tool result feedback exceeds %d bytes", MaxCanonicalFeedbackBytes)}
	}
	return nil
}

func WithCanonicalInput(req types.ModelRequest) (types.ModelRequest, error) {
	input, err := CanonicalInput(req)
	if err != nil {
		return req, err
	}
	out := req
	out.Input = input
	return out, nil
}

func CanonicalInput(req types.ModelRequest) (string, error) {
	base := strings.TrimSpace(req.Input)
	if base == "" && len(req.Messages) > 0 {
		base = strings.TrimSpace(req.Messages[len(req.Messages)-1].Content)
	}
	if len(req.ToolResult) == 0 {
		return base, nil
	}

	interpreted, err := InterpretRequest(req)
	if err != nil {
		return "", err
	}
	items := make([]feedbackEnvelopeItem, 0, len(interpreted.ToolResults))
	for _, outcome := range interpreted.ToolResults {
		items = append(items, feedbackEnvelopeFromOutcome(outcome))
	}
	blob, err := json.Marshal(items)
	if err != nil {
		return "", err
	}
	if base == "" {
		return FeedbackHeader + "\n" + string(blob), nil
	}
	return base + "\n\n" + FeedbackHeader + "\n" + string(blob), nil
}

func cloneAnyMap(in map[string]any) map[string]any {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]any, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}
