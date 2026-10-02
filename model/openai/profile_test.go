package openai

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/FelixSeptem/baymax/core/types"
	"github.com/openai/openai-go/responses"
)

func validCompatibleProfile() EndpointProfile {
	return EndpointProfile{
		Version:  EndpointProfileVersion,
		ID:       "qwen-compatible",
		Endpoint: "https://example.test/v1",
		Model:    "qwen-max",
		APIShape: ResponsesAPIShapeVersion,
		Capabilities: map[EndpointCapability]types.CapabilitySupport{
			EndpointCapabilityStructuredInput:  types.CapabilitySupportSupported,
			EndpointCapabilityNativeToolResult: types.CapabilitySupportSupported,
			EndpointCapabilityStreaming:        types.CapabilitySupportSupported,
			EndpointCapabilityCacheUsage:       types.CapabilitySupportUnknown,
			EndpointCapabilityCountTokens:      types.CapabilitySupportUnsupported,
		},
	}
}

func TestEndpointProfileValidateAcceptsBoundedCompatibleProfile(t *testing.T) {
	profile := validCompatibleProfile()
	if err := profile.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	if got := profile.Capability(EndpointCapabilityCacheUsage); got != types.CapabilitySupportUnknown {
		t.Fatalf("cache capability = %q, want unknown", got)
	}
}

func TestEndpointProfileValidateRejectsMalformedAndSensitiveProfiles(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(*EndpointProfile)
		want   string
	}{
		{name: "version", mutate: func(p *EndpointProfile) { p.Version = "v0" }, want: ReasonProfileUnsupportedVersion},
		{name: "endpoint scheme", mutate: func(p *EndpointProfile) { p.Endpoint = "ftp://example.test/v1" }, want: ReasonProfileInvalidEndpoint},
		{name: "endpoint userinfo", mutate: func(p *EndpointProfile) { p.Endpoint = "https://key:secret@example.test/v1" }, want: ReasonProfilePrivacyViolation},
		{name: "model sensitive", mutate: func(p *EndpointProfile) { p.Model = "secret-model" }, want: ReasonProfilePrivacyViolation},
		{name: "api shape", mutate: func(p *EndpointProfile) { p.APIShape = "chat-completions.v1" }, want: ReasonProfileUnsupportedAPIShape},
		{name: "capability", mutate: func(p *EndpointProfile) { p.Capabilities[EndpointCapabilityStreaming] = "maybe" }, want: ReasonProfileInvalidCapability},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			profile := validCompatibleProfile()
			tc.mutate(&profile)
			err := profile.Validate()
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("Validate() error = %v, want code %q", err, tc.want)
			}
		})
	}
}

func TestOpenAIRequestRejectsCustomBaseURLWithoutExplicitProfile(t *testing.T) {
	called := false
	client := NewClient(Config{
		BaseURL: "https://example.test/v1",
		Model:   "qwen-max",
		GenerateFn: func(context.Context, types.ModelRequest) (types.ModelResponse, error) {
			called = true
			return types.ModelResponse{FinalAnswer: "unexpected"}, nil
		},
	})
	_, err := client.Generate(context.Background(), types.ModelRequest{Input: "hello"})
	if err == nil || !strings.Contains(err.Error(), ReasonProfileRequired) {
		t.Fatalf("Generate() error = %v, want %q", err, ReasonProfileRequired)
	}
	if called {
		t.Fatal("GenerateFn was called despite missing compatibility profile")
	}
}

func TestOpenAIRequestRejectsProfileEndpointMismatchBeforeProviderCall(t *testing.T) {
	called := false
	profile := validCompatibleProfile()
	client := NewClient(Config{
		BaseURL: "https://other.test/v1",
		Model:   profile.Model,
		Profile: &profile,
		GenerateFn: func(context.Context, types.ModelRequest) (types.ModelResponse, error) {
			called = true
			return types.ModelResponse{}, nil
		},
	})
	_, err := client.Generate(context.Background(), types.ModelRequest{Input: "hello"})
	if err == nil || !strings.Contains(err.Error(), ReasonProfileEndpointMismatch) {
		t.Fatalf("Generate() error = %v, want %q", err, ReasonProfileEndpointMismatch)
	}
	if called {
		t.Fatal("GenerateFn was called despite endpoint mismatch")
	}
}

func TestProfileUnknownCacheCapabilityDoesNotFabricateUsage(t *testing.T) {
	profile := validCompatibleProfile()
	var response responses.Response
	if err := json.Unmarshal([]byte(`{"output":[],"usage":{"input_tokens":30,"input_tokens_details":{"cached_tokens":12},"output_tokens":2,"output_tokens_details":{},"total_tokens":32}}`), &response); err != nil {
		t.Fatal(err)
	}
	client := NewClient(Config{BaseURL: profile.Endpoint, Model: profile.Model, Profile: &profile})
	client.newResponse = func(context.Context, responses.ResponseNewParams) (*responses.Response, error) { return &response, nil }
	got, err := client.Generate(context.Background(), types.ModelRequest{Input: "hello"})
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if got.CacheUsage.Available || got.CacheUsage.ReadTokens != 0 || got.CacheUsage.TotalTokens != 0 {
		t.Fatalf("cache usage = %+v, want unavailable zero projection", got.CacheUsage)
	}
}

func TestProfileMissingStructuredCapabilityBlocksRequest(t *testing.T) {
	profile := validCompatibleProfile()
	delete(profile.Capabilities, EndpointCapabilityStructuredInput)
	called := false
	client := NewClient(Config{
		BaseURL: profile.Endpoint,
		Model:   profile.Model,
		Profile: &profile,
		GenerateFn: func(context.Context, types.ModelRequest) (types.ModelResponse, error) {
			called = true
			return types.ModelResponse{}, nil
		},
	})
	_, err := client.Generate(context.Background(), types.ModelRequest{Input: "hello"})
	if err == nil || !strings.Contains(err.Error(), ReasonProfileCapabilityUnsupported) {
		t.Fatalf("Generate() error = %v, want %q", err, ReasonProfileCapabilityUnsupported)
	}
	if called {
		t.Fatal("GenerateFn was called despite unknown structured-input capability")
	}
}
