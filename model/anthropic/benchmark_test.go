package anthropic

import (
	"testing"

	anthropic "github.com/anthropics/anthropic-sdk-go"
)

func BenchmarkProviderStreamEventMapAnthropic(b *testing.B) {
	streamEvents := []anthropic.MessageStreamEventUnion{
		{Type: "content_block_start", Index: 1, ContentBlock: anthropic.ContentBlockStartEventContentBlockUnion{Type: "tool_use", ID: "call-1", Name: "local.weather"}},
		{Type: "content_block_delta", Index: 1, Delta: anthropic.MessageStreamEventUnionDelta{Type: "input_json_delta", PartialJSON: `{"city":"shan`}},
		{Type: "content_block_delta", Index: 1, Delta: anthropic.MessageStreamEventUnionDelta{Type: "input_json_delta", PartialJSON: `ghai"}`}},
		{Type: "content_block_stop", Index: 1},
		{Type: "content_block_delta", Index: 0, Delta: anthropic.MessageStreamEventUnionDelta{Type: "text_delta", Text: "hello"}},
		{Type: "message_stop", Index: 0},
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		state := &streamState{toolByIndex: map[int64]*toolCallState{}}
		for j := range streamEvents {
			if _, err := mapStreamEvent(streamEvents[j], state); err != nil {
				b.Fatalf("mapStreamEvent failed: %v", err)
			}
		}
	}
}

func BenchmarkProviderResponseDecodeAnthropic(b *testing.B) {
	msg := anthropic.Message{
		Content: []anthropic.ContentBlockUnion{
			{Type: "text", Text: "hello"},
			{Type: "text", Text: "world"},
		},
		Usage: anthropic.Usage{
			InputTokens:  64,
			OutputTokens: 32,
		},
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = decodeMessage(msg)
	}
}
