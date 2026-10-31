package types

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestModelResponseCacheUsageRoundTrip(t *testing.T) {
	want := ModelResponse{
		FinalAnswer: "cache-aware",
		CacheUsage: CacheUsageProjection{
			Available: true, ReadTokens: 7, TotalTokens: 7,
			SourceKind: "openai_responses", SourceVersion: "v1",
		},
	}
	raw, err := json.Marshal(want)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	var got ModelResponse
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if got.CacheUsage != want.CacheUsage {
		t.Fatalf("cache usage = %+v, want %+v", got.CacheUsage, want.CacheUsage)
	}
}

func TestModelRequestPromptCacheDirectiveRoundTripAndLegacyDefault(t *testing.T) {
	legacy, err := json.Marshal(ModelRequest{Input: "hello"})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(legacy), "prompt_cache") {
		t.Fatalf("legacy request unexpectedly contains cache directive: %s", legacy)
	}

	want := ModelRequest{
		Input: "hello",
		PromptCache: &PromptCacheDirective{
			Provider: "openai",
			OpenAI:   &OpenAIPromptCacheDirective{Key: "opaque-key"},
		},
	}
	raw, err := json.Marshal(want)
	if err != nil {
		t.Fatal(err)
	}
	var got ModelRequest
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatal(err)
	}
	if got.PromptCache == nil || got.PromptCache.Provider != "openai" || got.PromptCache.OpenAI == nil || got.PromptCache.OpenAI.Key != "opaque-key" {
		t.Fatalf("prompt cache directive round trip = %+v", got.PromptCache)
	}
}

func TestPromptCacheDirectiveValidatesProviderVariantAndBounds(t *testing.T) {
	cases := []struct {
		name      string
		provider  string
		count     int
		directive PromptCacheDirective
		wantErr   bool
	}{
		{name: "openai valid", provider: "openai", directive: PromptCacheDirective{Provider: "openai", OpenAI: &OpenAIPromptCacheDirective{Key: "key"}}},
		{name: "anthropic valid", provider: "anthropic", count: 2, directive: PromptCacheDirective{Provider: "anthropic", Anthropic: &AnthropicPromptCacheDirective{BoundaryMessageIndex: 1, TTL: "5m"}}},
		{name: "gemini valid", provider: "gemini", directive: PromptCacheDirective{Provider: "gemini", Gemini: &GeminiPromptCacheDirective{CachedContentRef: "cachedContents/abc"}}},
		{name: "adapter mismatch", provider: "gemini", directive: PromptCacheDirective{Provider: "openai", OpenAI: &OpenAIPromptCacheDirective{Key: "key"}}, wantErr: true},
		{name: "conflicting variants", provider: "openai", directive: PromptCacheDirective{Provider: "openai", OpenAI: &OpenAIPromptCacheDirective{Key: "key"}, Gemini: &GeminiPromptCacheDirective{CachedContentRef: "ref"}}, wantErr: true},
		{name: "empty key", provider: "openai", directive: PromptCacheDirective{Provider: "openai", OpenAI: &OpenAIPromptCacheDirective{}}, wantErr: true},
		{name: "oversized key", provider: "openai", directive: PromptCacheDirective{Provider: "openai", OpenAI: &OpenAIPromptCacheDirective{Key: strings.Repeat("k", 65)}}, wantErr: true},
		{name: "invalid ttl", provider: "anthropic", count: 2, directive: PromptCacheDirective{Provider: "anthropic", Anthropic: &AnthropicPromptCacheDirective{BoundaryMessageIndex: 0, TTL: "2h"}}, wantErr: true},
		{name: "boundary out of range", provider: "anthropic", count: 1, directive: PromptCacheDirective{Provider: "anthropic", Anthropic: &AnthropicPromptCacheDirective{BoundaryMessageIndex: 1, TTL: "1h"}}, wantErr: true},
		{name: "empty gemini ref", provider: "gemini", directive: PromptCacheDirective{Provider: "gemini", Gemini: &GeminiPromptCacheDirective{}}, wantErr: true},
		{name: "oversized gemini ref", provider: "gemini", directive: PromptCacheDirective{Provider: "gemini", Gemini: &GeminiPromptCacheDirective{CachedContentRef: strings.Repeat("r", 513)}}, wantErr: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.directive.ValidateForProvider(tc.provider, tc.count)
			if (err != nil) != tc.wantErr {
				t.Fatalf("ValidateForProvider() error = %v, wantErr %v", err, tc.wantErr)
			}
		})
	}
}

func TestDefaultLoopPolicy(t *testing.T) {
	p := DefaultLoopPolicy()
	if p.MaxIterations != 12 {
		t.Fatalf("MaxIterations = %d, want 12", p.MaxIterations)
	}
	if p.MaxToolCallsPerIteration != 8 {
		t.Fatalf("MaxToolCallsPerIteration = %d, want 8", p.MaxToolCallsPerIteration)
	}
	if p.ToolCallLimit != 64 {
		t.Fatalf("ToolCallLimit = %d, want 64", p.ToolCallLimit)
	}
	if p.StepTimeout != 60*time.Second {
		t.Fatalf("StepTimeout = %s, want 60s", p.StepTimeout)
	}
	if p.ModelRetry != 2 {
		t.Fatalf("ModelRetry = %d, want 2", p.ModelRetry)
	}
	if p.ToolRetry != 1 {
		t.Fatalf("ToolRetry = %d, want 1", p.ToolRetry)
	}
	if p.ContinueOnToolError {
		t.Fatal("ContinueOnToolError = true, want false")
	}
}

func TestRunResultJSONRoundTrip(t *testing.T) {
	want := RunResult{
		RunID:       "run-1",
		FinalAnswer: "done",
		Iterations:  2,
		TokenUsage: TokenUsage{
			InputTokens:  10,
			OutputTokens: 20,
			TotalTokens:  30,
		},
		LatencyMs: 1234,
		Warnings:  []string{"warn"},
		Error: &ClassifiedError{
			Class:   ErrTool,
			Message: "tool failed",
		},
	}
	data, err := json.Marshal(want)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	var got RunResult
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if got.RunID != want.RunID || got.FinalAnswer != want.FinalAnswer || got.Iterations != want.Iterations {
		t.Fatalf("unexpected round-trip result: %#v", got)
	}
	if got.Error == nil || got.Error.Class != ErrTool {
		t.Fatalf("unexpected error round-trip: %#v", got.Error)
	}
}

func TestModelResponseJSONRoundTrip(t *testing.T) {
	want := ModelResponse{
		FinalAnswer: "next",
		ToolCalls: []ToolCall{
			{CallID: "c1", Name: "local.search", Args: map[string]any{"q": "golang"}},
		},
		ClarificationRequest: &ClarificationRequest{
			RequestID:      "clarify-1",
			Questions:      []string{"which repo?"},
			ContextSummary: "missing target scope",
			Timeout:        5 * time.Second,
		},
		Usage: TokenUsage{InputTokens: 3, OutputTokens: 7, TotalTokens: 10},
	}
	data, err := json.Marshal(want)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	var got ModelResponse
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if len(got.ToolCalls) != 1 || got.ToolCalls[0].Name != "local.search" {
		t.Fatalf("unexpected tool calls: %#v", got.ToolCalls)
	}
	if got.ClarificationRequest == nil || got.ClarificationRequest.RequestID != "clarify-1" {
		t.Fatalf("unexpected clarification request: %#v", got.ClarificationRequest)
	}
	if got.Usage.TotalTokens != 10 {
		t.Fatalf("unexpected usage: %#v", got.Usage)
	}
}

func TestToolResultJSONRoundTrip(t *testing.T) {
	want := ToolResult{
		Content:    "ok",
		Structured: map[string]any{"count": float64(1)},
		Error: &ClassifiedError{
			Class:     ErrPolicyTimeout,
			Message:   "timeout",
			Retryable: true,
		},
	}
	data, err := json.Marshal(want)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	var got ToolResult
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if got.Error == nil || got.Error.Class != ErrPolicyTimeout {
		t.Fatalf("unexpected error: %#v", got.Error)
	}
}

func TestCapabilityRequirementsNormalized(t *testing.T) {
	req := CapabilityRequirements{
		Required: []ModelCapability{
			ModelCapabilityStreaming,
			"",
			ModelCapabilityToolCall,
			ModelCapabilityStreaming,
		},
	}
	got := req.Normalized()
	if len(got) != 2 {
		t.Fatalf("normalized len = %d, want 2", len(got))
	}
	if got[0] != ModelCapabilityStreaming || got[1] != ModelCapabilityToolCall {
		t.Fatalf("normalized = %#v", got)
	}
}

func TestProviderCapabilitiesMissing(t *testing.T) {
	caps := ProviderCapabilities{
		Support: map[ModelCapability]CapabilitySupport{
			ModelCapabilityStreaming: CapabilitySupportSupported,
			ModelCapabilityToolCall:  CapabilitySupportUnknown,
		},
	}
	missing := caps.Missing([]ModelCapability{ModelCapabilityStreaming, ModelCapabilityToolCall})
	if len(missing) != 1 || missing[0] != ModelCapabilityToolCall {
		t.Fatalf("missing = %#v, want [tool_call]", missing)
	}
}

func TestNormalizeSandboxExecSpecDefaultsAndSanitization(t *testing.T) {
	spec, err := NormalizeSandboxExecSpec(SandboxExecSpec{
		NamespaceTool: " Local+Shell ",
		Command:       " powershell ",
		Workdir:       ".",
		Env: map[string]string{
			"":        "skip",
			" EMPTY ": "   ",
			" PATH ":  " C:\\Windows ",
		},
		Mounts: []SandboxMount{
			{Source: "C:\\b", Target: "/b", ReadOnly: true},
			{Source: " C:\\a ", Target: " /a "},
		},
	})
	if err != nil {
		t.Fatalf("NormalizeSandboxExecSpec failed: %v", err)
	}
	if spec.NamespaceTool != "local+shell" {
		t.Fatalf("namespace_tool=%q, want local+shell", spec.NamespaceTool)
	}
	if spec.Command != "powershell" {
		t.Fatalf("command=%q, want powershell", spec.Command)
	}
	if spec.SessionMode != SandboxSessionModePerCall {
		t.Fatalf("session_mode=%q, want %q", spec.SessionMode, SandboxSessionModePerCall)
	}
	if !filepath.IsAbs(spec.Workdir) {
		t.Fatalf("workdir must be absolute, got %q", spec.Workdir)
	}
	if len(spec.Env) != 1 || spec.Env["PATH"] != "C:\\Windows" {
		t.Fatalf("env sanitization mismatch: %#v", spec.Env)
	}
	if len(spec.Mounts) != 2 {
		t.Fatalf("mounts len=%d, want 2", len(spec.Mounts))
	}
	if spec.Mounts[0].Target != "/a" || spec.Mounts[0].Source != "C:\\a" {
		t.Fatalf("mounts[0] should be normalized and sorted, got %#v", spec.Mounts[0])
	}
	if spec.Mounts[1].Target != "/b" {
		t.Fatalf("mounts[1] target=%q, want /b", spec.Mounts[1].Target)
	}
}

func TestNormalizeSandboxExecSpecRejectsInvalidSpec(t *testing.T) {
	_, err := NormalizeSandboxExecSpec(SandboxExecSpec{
		Command: "",
	})
	if err == nil || !strings.Contains(err.Error(), "sandbox exec command is required") {
		t.Fatalf("expected required command error, got %v", err)
	}

	_, err = NormalizeSandboxExecSpec(SandboxExecSpec{
		Command: "cmd",
		Mounts: []SandboxMount{
			{Source: "C:\\ok", Target: ""},
		},
	})
	if err == nil || !strings.Contains(err.Error(), "source/target must not be empty") {
		t.Fatalf("expected invalid mount error, got %v", err)
	}
}
