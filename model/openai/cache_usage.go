package openai

import (
	"fmt"

	"github.com/FelixSeptem/baymax/model/conformance"
	"github.com/openai/openai-go/responses"
)

func projectResponseCacheUsage(usage responses.ResponseUsage) (conformance.CacheUsageProjection, error) {
	projection := conformance.CacheUsageProjection{}
	if !usage.JSON.InputTokensDetails.Valid() || !usage.InputTokensDetails.JSON.CachedTokens.Valid() {
		return projection, fmt.Errorf("openai cached input token usage is unavailable")
	}
	if usage.InputTokensDetails.CachedTokens < 0 {
		return projection, fmt.Errorf("openai cached input token usage must not be negative")
	}
	projection = conformance.CacheUsageProjection{
		Available:     true,
		ReadTokens:    usage.InputTokensDetails.CachedTokens,
		TotalTokens:   usage.InputTokensDetails.CachedTokens,
		SourceKind:    conformance.CacheUsageSourceOpenAIResponses,
		SourceVersion: "v1",
	}
	if err := conformance.ValidateCacheUsageProjection(projection); err != nil {
		return conformance.CacheUsageProjection{}, err
	}
	return projection, nil
}
