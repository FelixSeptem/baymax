package integration

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/FelixSeptem/baymax/core/runner"
	"github.com/FelixSeptem/baymax/core/types"
	"github.com/FelixSeptem/baymax/integration/fakes"
)

type streamGoldenEvent struct {
	Type      string `json:"type"`
	EventType string `json:"event_type,omitempty"`
	Delta     string `json:"delta,omitempty"`
	HasTool   bool   `json:"has_tool_call,omitempty"`
}

func TestStreamingEventSequenceGolden(t *testing.T) {
	model := fakes.NewModel(nil)
	model.SetStream([]types.ModelEvent{
		{Type: types.ModelEventTypeOutputTextDelta, TextDelta: "hello "},
		{
			Type: types.ModelEventTypeToolCall,
			ToolCall: &types.ToolCall{
				CallID: "call-1",
				Name:   "local.weather",
				Args:   map[string]any{"city": "shanghai"},
			},
		},
		{Type: types.ModelEventTypeOutputTextDelta, TextDelta: "world"},
	}, nil)

	collector := &eventCollector{}
	eng := runner.New(model)
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
