package guard

import (
	"errors"
	"strings"
	"testing"

	"github.com/FelixSeptem/baymax/core/types"
)

func TestGuardFailFastOnSchemaViolation(t *testing.T) {
	g := New(true)
	_, err := g.Apply(types.ContextAssembleRequest{
		RunID:         "",
		PrefixVersion: "ca1",
	}, "hash", "")
	if !errors.Is(err, ErrGuardViolation) {
		t.Fatalf("err = %v, want ErrGuardViolation", err)
	}
}

func TestGuardSanitizeSensitiveText(t *testing.T) {
	g := New(true)
	out, err := g.Apply(types.ContextAssembleRequest{
		RunID:         "run-1",
		PrefixVersion: "ca1",
		Input:         "token=abc123",
		Messages: []types.Message{
			{Role: "user", Content: "api_key: sk-xxxx"},
		},
	}, "hash", "")
	if err != nil {
		t.Fatalf("Apply failed: %v", err)
	}
	if strings.Contains(out.Input, "abc123") {
		t.Fatalf("input not sanitized: %q", out.Input)
	}
	if strings.Contains(out.Messages[0].Content, "sk-xxxx") {
		t.Fatalf("message not sanitized: %q", out.Messages[0].Content)
	}
}
