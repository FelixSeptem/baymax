package anthropic

import (
	"fmt"

	"github.com/FelixSeptem/baymax/model/conformance"
	"github.com/anthropics/anthropic-sdk-go"
)

func projectMessageCacheUsage(usage anthropic.Usage) (conformance.CacheUsageProjection, error) {
	if !usage.JSON.CacheReadInputTokens.Valid() && !usage.JSON.CacheCreationInputTokens.Valid() {
		return conformance.CacheUsageProjection{}, fmt.Errorf("anthropic cache usage is unavailable")
	}
	return projectCacheUsageCounts(usage.CacheReadInputTokens, usage.CacheCreationInputTokens)
}

func projectMessageDeltaCacheUsage(usage anthropic.MessageDeltaUsage) (conformance.CacheUsageProjection, error) {
	if !usage.JSON.CacheReadInputTokens.Valid() && !usage.JSON.CacheCreationInputTokens.Valid() {
		return conformance.CacheUsageProjection{}, fmt.Errorf("anthropic cache usage is unavailable")
	}
	return projectCacheUsageCounts(usage.CacheReadInputTokens, usage.CacheCreationInputTokens)
}

func projectCacheUsageCounts(readTokens, writeTokens int64) (conformance.CacheUsageProjection, error) {
	if readTokens < 0 || writeTokens < 0 {
		return conformance.CacheUsageProjection{}, fmt.Errorf("anthropic cache usage must not be negative")
	}
	projection := conformance.CacheUsageProjection{
		Available:     true,
		ReadTokens:    readTokens,
		WriteTokens:   writeTokens,
		TotalTokens:   readTokens + writeTokens,
		SourceKind:    conformance.CacheUsageSourceAnthropic,
		SourceVersion: "v1",
	}
	if err := conformance.ValidateCacheUsageProjection(projection); err != nil {
		return conformance.CacheUsageProjection{}, err
	}
	return projection, nil
}
