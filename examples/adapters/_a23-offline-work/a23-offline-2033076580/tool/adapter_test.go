package offline

import (
	"context"
	"testing"
)

func TestOfflineToolAdapterInvokeFailFast(t *testing.T) {
	tool := OfflineToolAdapter{}
	res, err := tool.Invoke(context.Background(), map[string]any{"input": "hello"})
	if err != nil {
		t.Fatalf("invoke: %v", err)
	}
	if res.Content != "echo=hello" {
		t.Fatalf("unexpected content: %s", res.Content)
	}
	if _, err := tool.Invoke(context.Background(), map[string]any{}); err == nil {
		t.Fatal("expected fail-fast for missing required input")
	}
}
