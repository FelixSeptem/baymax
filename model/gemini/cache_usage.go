package gemini

import (
	"encoding/json"
	"fmt"

	"github.com/FelixSeptem/baymax/model/conformance"
	"google.golang.org/genai"
)

func projectGenerateContentCacheUsage(resp *genai.GenerateContentResponse) (conformance.CacheUsageProjection, error) {
	if resp == nil || resp.UsageMetadata == nil || resp.UsageMetadata.CachedContentTokenCount <= 0 {
		return conformance.CacheUsageProjection{}, fmt.Errorf("gemini cached content token usage is unavailable")
	}
	count := int64(resp.UsageMetadata.CachedContentTokenCount)
	projection := conformance.CacheUsageProjection{
		Available:     true,
		ReadTokens:    count,
		TotalTokens:   count,
		SourceKind:    conformance.CacheUsageSourceGemini,
		SourceVersion: "v1",
	}
	if err := conformance.ValidateCacheUsageProjection(projection); err != nil {
		return conformance.CacheUsageProjection{}, err
	}
	return projection, nil
}

func projectGenerateContentCacheUsageBytes(raw []byte) (conformance.CacheUsageProjection, error) {
	var resp genai.GenerateContentResponse
	if err := json.Unmarshal(raw, &resp); err != nil {
		return conformance.CacheUsageProjection{}, fmt.Errorf("decode gemini cache usage: %w", err)
	}
	return projectGenerateContentCacheUsage(&resp)
}
