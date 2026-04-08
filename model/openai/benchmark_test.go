package openai

import (
	"testing"

	"github.com/openai/openai-go/responses"
)

func BenchmarkProviderStreamEventMapOpenAI(b *testing.B) {
	streamEvents := []responses.ResponseStreamEventUnion{
		{
			Type:   "response.output_item.done",
			ItemID: "item-1",
			Item: responses.ResponseOutputItemUnion{
				Type:   "function_call",
				CallID: "call-1",
				Name:   "local.weather",
			},
		},
		{
			Type:   "response.function_call_arguments.delta",
			ItemID: "item-1",
			Delta: responses.ResponseStreamEventUnionDelta{
				OfString: `{"city":"`,
			},
		},
		{
			Type:      "response.function_call_arguments.done",
			ItemID:    "item-1",
			Arguments: `{"city":"shanghai"}`,
		},
		{
			Type: "response.output_text.delta",
			Delta: responses.ResponseStreamEventUnionDelta{
				OfString: "hello",
			},
		},
		{
			Type: "response.output_text.done",
			Text: "hello",
		},
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		state := &streamState{toolCalls: map[string]*toolCallState{}}
		for j := range streamEvents {
			if _, err := mapStreamEvent(streamEvents[j], state); err != nil {
				b.Fatalf("mapStreamEvent failed: %v", err)
			}
		}
	}
}
