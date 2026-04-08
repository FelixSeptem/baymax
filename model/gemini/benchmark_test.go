package gemini

import (
	"testing"

	"google.golang.org/genai"
)

func benchmarkGeminiStreamChunk() *genai.GenerateContentResponse {
	return &genai.GenerateContentResponse{
		Candidates: []*genai.Candidate{
			{
				Content: &genai.Content{
					Parts: []*genai.Part{
						{Text: "hello"},
						{
							FunctionCall: &genai.FunctionCall{
								ID:   "call-1",
								Name: "local.weather",
								Args: map[string]any{"city": "shanghai"},
							},
						},
					},
				},
			},
		},
	}
}

func benchmarkGeminiDecodeResponse() *genai.GenerateContentResponse {
	return &genai.GenerateContentResponse{
		Candidates: []*genai.Candidate{
			{
				Content: &genai.Content{
					Parts: []*genai.Part{
						{Text: "hello"},
						{Text: "world"},
					},
				},
			},
		},
		UsageMetadata: &genai.GenerateContentResponseUsageMetadata{
			PromptTokenCount:     64,
			CandidatesTokenCount: 32,
			TotalTokenCount:      96,
		},
	}
}

func BenchmarkProviderStreamEventMapGemini(b *testing.B) {
	chunk := benchmarkGeminiStreamChunk()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		toolSeq := 0
		_ = mapStreamChunk(chunk, &toolSeq)
	}
}

func BenchmarkProviderResponseDecodeGemini(b *testing.B) {
	resp := benchmarkGeminiDecodeResponse()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = decodeGenerateResponse(resp)
	}
}
