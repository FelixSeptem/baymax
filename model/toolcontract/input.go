package toolcontract

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/FelixSeptem/baymax/core/types"
	providererror "github.com/FelixSeptem/baymax/model/providererror"
)

const FeedbackHeader = "[tool_result_feedback.v1]"

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

	items := make([]feedbackEnvelopeItem, 0, len(req.ToolResult))
	for i := range req.ToolResult {
		outcome := req.ToolResult[i]
		callID := strings.TrimSpace(outcome.CallID)
		name := strings.TrimSpace(outcome.Name)
		if callID == "" || name == "" {
			return "", &providererror.Classified{
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
		item := feedbackEnvelopeItem{
			ToolCallID: callID,
			ToolName:   name,
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
		items = append(items, item)
	}
	blob, err := json.Marshal(items)
	if err != nil {
		return "", &providererror.Classified{
			Class:     types.ErrModel,
			Reason:    "feedback_invalid",
			Retryable: false,
			Cause:     fmt.Errorf("marshal canonical tool result feedback: %w", err),
		}
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
