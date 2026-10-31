# Provider Prompt Cache

## Pattern

Explicit, request-scoped provider-native cache controls.

## Build a request

```go
request := types.ModelRequest{
	Messages: []types.Message{
		{Role: "system", Content: "stable instructions"},
		{Role: "user", Content: "current question"},
	},
	PromptCache: &types.PromptCacheDirective{
		Provider: "openai",
		OpenAI:   &types.OpenAIPromptCacheDirective{Key: "example-only-cache-key"},
	},
}
```

Use exactly one provider variant matching the selected adapter:

```go
// Anthropic marks the selected zero-based interpreted message boundary.
types.PromptCacheDirective{
	Provider:  "anthropic",
	Anthropic: &types.AnthropicPromptCacheDirective{BoundaryMessageIndex: 0, TTL: "5m"},
}

// Gemini references a cached-content resource created by the host.
types.PromptCacheDirective{
	Provider: "gemini",
	Gemini:   &types.GeminiPromptCacheDirective{CachedContentRef: "cachedContents/example"},
}
```

## Contract

- Omit `PromptCache` to retain the existing request shape.
- A provider mismatch, invalid variant, unsupported TTL, invalid boundary, or out-of-bound value fails before provider invocation.
- The OpenAI key and Gemini resource reference are opaque caller-owned values. Do not place secrets, user identifiers, or raw prompt material in them; Baymax never writes them to diagnostics, fixtures, or normalized projections.
- Anthropic boundaries index the interpreted message sequence, including a synthesized final input when present.
- Baymax does not create, look up, update, or delete cached resources. A directive does not prove a cache hit or lower cost.
- OpenAI and Gemini CountTokens reject a supplied cache directive with `prompt_cache_unsupported`; Anthropic preserves the supported breakpoint in its token-count request.

Run provider shape and privacy checks with `pwsh -File scripts/check-provider-request-projection-contract.ps1` or `bash scripts/check-provider-request-projection-contract.sh`.
