package fixture

import (
	"context"
	"testing"
)

func TestFixtureMcpAdapterClientEcho(t *testing.T) {
	client := NewFixtureMcpAdapterClient()
	defer func() { _ = client.Close() }()

	res, err := client.CallTool(context.Background(), "echo", map[string]any{"input": "hello"})
	if err != nil {
		t.Fatalf("call tool: %v", err)
	}
	if res.Content != "tool=echo input=hello" {
		t.Fatalf("unexpected content: %s", res.Content)
	}
}
