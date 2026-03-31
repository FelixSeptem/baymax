package toolcontract

import (
	"errors"
	"strings"
	"testing"

	"github.com/FelixSeptem/baymax/core/types"
	providererror "github.com/FelixSeptem/baymax/model/providererror"
)

func TestCanonicalInputWithoutToolFeedbackUsesBaseInput(t *testing.T) {
	input, err := CanonicalInput(types.ModelRequest{
		Input: "hello",
	})
	if err != nil {
		t.Fatalf("CanonicalInput error: %v", err)
	}
	if input != "hello" {
		t.Fatalf("input=%q, want hello", input)
	}
}

func TestCanonicalInputAppendsToolFeedbackEnvelope(t *testing.T) {
	input, err := CanonicalInput(types.ModelRequest{
		Input: "hello",
		ToolResult: []types.ToolCallOutcome{
			{
				CallID: "call-1",
				Name:   "local.echo",
				Result: types.ToolResult{
					Content: "ok",
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("CanonicalInput error: %v", err)
	}
	if !strings.Contains(input, "hello") ||
		!strings.Contains(input, FeedbackHeader) ||
		!strings.Contains(input, `"tool_call_id":"call-1"`) ||
		!strings.Contains(input, `"tool_name":"local.echo"`) {
		t.Fatalf("canonical input missing expected feedback envelope: %q", input)
	}
}

func TestCanonicalInputRejectsInvalidFeedbackShape(t *testing.T) {
	_, err := CanonicalInput(types.ModelRequest{
		Input: "hello",
		ToolResult: []types.ToolCallOutcome{
			{
				CallID: "",
				Name:   "local.echo",
				Result: types.ToolResult{Content: "ok"},
			},
		},
	})
	if err == nil {
		t.Fatal("expected feedback_invalid error")
	}
	var classified *providererror.Classified
	if !errors.As(err, &classified) {
		t.Fatalf("expected provider classified error, got %T", err)
	}
	if classified.Reason != "feedback_invalid" {
		t.Fatalf("reason=%q, want feedback_invalid", classified.Reason)
	}
}
