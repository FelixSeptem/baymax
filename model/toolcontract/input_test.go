package toolcontract

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/FelixSeptem/baymax/core/types"
	providererror "github.com/FelixSeptem/baymax/model/providererror"
)

func TestInterpretRequestPreservesOrderedMessagesFinalInputAndToolCorrelation(t *testing.T) {
	interpreted, err := InterpretRequest(types.ModelRequest{
		Messages: []types.Message{
			{Role: "system", Content: "  system rule  "},
			{Role: "user", Content: "user question"},
			{Role: "assistant", Content: "assistant history"},
			{Role: "user", Content: " \t "},
		},
		Input: "  final instruction  ",
		ToolResult: []types.ToolCallOutcome{{
			CallID: " call-1 ",
			Name:   " local.read ",
			Result: types.ToolResult{Content: "file body"},
		}},
	})
	if err != nil {
		t.Fatalf("InterpretRequest error: %v", err)
	}

	wantMessages := []types.Message{
		{Role: "system", Content: "system rule"},
		{Role: "user", Content: "user question"},
		{Role: "assistant", Content: "assistant history"},
		{Role: "user", Content: "final instruction"},
	}
	if !reflect.DeepEqual(interpreted.Messages, wantMessages) {
		t.Fatalf("messages = %#v, want %#v", interpreted.Messages, wantMessages)
	}
	if len(interpreted.ToolResults) != 1 {
		t.Fatalf("tool result count = %d, want 1", len(interpreted.ToolResults))
	}
	gotResult := interpreted.ToolResults[0]
	if gotResult.CallID != "call-1" || gotResult.Name != "local.read" || gotResult.Result.Content != "file body" {
		t.Fatalf("tool result = %#v, want trimmed native correlation", gotResult)
	}
}

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

func TestCanonicalInputRejectsOversizedFeedback(t *testing.T) {
	_, err := CanonicalInput(types.ModelRequest{ToolResult: []types.ToolCallOutcome{{
		CallID: "call-1", Name: "local.echo", Result: types.ToolResult{Content: strings.Repeat("x", MaxCanonicalFeedbackBytes)},
	}}})
	if err == nil {
		t.Fatal("expected overflow error")
	}
	var classified *providererror.Classified
	if !errors.As(err, &classified) || classified.Reason != "overflow" {
		t.Fatalf("error=%v, want overflow classification", err)
	}
}
