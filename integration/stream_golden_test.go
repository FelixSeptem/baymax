package integration

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/FelixSeptem/baymax/core/runner"
	"github.com/FelixSeptem/baymax/core/types"
	"github.com/FelixSeptem/baymax/tool/local"
)

type streamGoldenModel struct {
	mu    sync.Mutex
	steps int
}

func (m *streamGoldenModel) Generate(ctx context.Context, req types.ModelRequest) (types.ModelResponse, error) {
	_ = ctx
	_ = req
	return types.ModelResponse{}, nil
}

func (m *streamGoldenModel) Stream(
	ctx context.Context,
	req types.ModelRequest,
	onEvent func(types.ModelEvent) error,
) error {
	_ = ctx
	_ = req
	m.mu.Lock()
	m.steps++
	step := m.steps
	m.mu.Unlock()

	if step == 1 {
		if err := onEvent(types.ModelEvent{Type: types.ModelEventTypeOutputTextDelta, TextDelta: "hello "}); err != nil {
			return err
		}
		if err := onEvent(types.ModelEvent{
			Type: types.ModelEventTypeToolCall,
			ToolCall: &types.ToolCall{
				CallID: "call-1",
				Name:   "local.weather",
				Args:   map[string]any{"city": "shanghai"},
			},
		}); err != nil {
			return err
		}
		return onEvent(types.ModelEvent{Type: types.ModelEventTypeOutputTextDelta, TextDelta: "world"})
	}
	return nil
}

func (m *streamGoldenModel) ProviderName() string {
	return "stream-golden"
}

func (m *streamGoldenModel) DiscoverCapabilities(
	ctx context.Context,
	req types.ModelRequest,
) (types.ProviderCapabilities, error) {
	_ = ctx
	return types.ProviderCapabilities{
		Provider: "stream-golden",
		Model:    req.Model,
		Support: map[types.ModelCapability]types.CapabilitySupport{
			types.ModelCapabilityStreaming: types.CapabilitySupportSupported,
			types.ModelCapabilityToolCall:  types.CapabilitySupportSupported,
		},
		Source:    "integration-test",
		CheckedAt: time.Now(),
	}, nil
}

type streamGoldenEvent struct {
	Type      string `json:"type"`
	EventType string `json:"event_type,omitempty"`
	Delta     string `json:"delta,omitempty"`
	HasTool   bool   `json:"has_tool_call,omitempty"`
}

func TestStreamingEventSequenceGolden(t *testing.T) {
	model := &streamGoldenModel{}

	collector := &eventCollector{}
	reg := local.NewRegistry()
	if _, err := reg.Register(streamGoldenTool{}); err != nil {
		t.Fatalf("register tool: %v", err)
	}
	eng := runner.New(model, runner.WithLocalRegistry(reg))
	if _, err := eng.Stream(context.Background(), types.RunRequest{Input: "hello"}, collector); err != nil {
		t.Fatalf("Stream failed: %v", err)
	}

	seq := make([]streamGoldenEvent, 0, len(collector.events))
	for _, ev := range nonTimelineEvents(collector.events) {
		item := streamGoldenEvent{Type: ev.Type}
		if ev.Type == "model.delta" && ev.Payload != nil {
			if v, ok := ev.Payload["event_type"].(string); ok {
				item.EventType = v
			}
			if v, ok := ev.Payload["delta"].(string); ok {
				item.Delta = v
			}
			_, item.HasTool = ev.Payload["tool_call"]
		}
		seq = append(seq, item)
	}

	got, err := json.MarshalIndent(seq, "", "  ")
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	goldenPath := filepath.Join("testdata", "stream_events.golden.json")
	want, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("read golden: %v", err)
	}
	if strings.TrimSpace(string(got)) != strings.TrimSpace(string(want)) {
		t.Fatalf("golden mismatch\n--- got ---\n%s\n--- want ---\n%s", string(got), string(want))
	}
}

type streamGoldenTool struct{}

func (streamGoldenTool) Name() string {
	return "weather"
}

func (streamGoldenTool) Description() string {
	return "stream golden tool"
}

func (streamGoldenTool) JSONSchema() map[string]any {
	return map[string]any{"type": "object"}
}

func (streamGoldenTool) Invoke(ctx context.Context, args map[string]any) (types.ToolResult, error) {
	_ = ctx
	_ = args
	return types.ToolResult{Content: "ok"}, nil
}
