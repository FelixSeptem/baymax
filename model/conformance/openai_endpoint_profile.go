package conformance

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/url"
	"sort"
	"strings"
)

const FixtureVersionOpenAIEndpointProfileV1 = "openai-compatible-endpoint-profile-conformance.v1"

const (
	OpenAIEndpointProfileVerdictConformant = "conformant"
	OpenAIEndpointProfileVerdictBlocked    = "blocked"
)

const (
	ReasonOpenAIEndpointProfileSchemaDrift     = "openai_endpoint_profile_schema_drift"
	ReasonOpenAIEndpointProfileReferenceDrift  = "openai_endpoint_profile_reference_drift"
	ReasonOpenAIEndpointProfilePrivacyDrift    = "openai_endpoint_profile_privacy_drift"
	ReasonOpenAIEndpointProfileOverflowDrift   = "openai_endpoint_profile_overflow_drift"
	ReasonOpenAIEndpointProfileRoleDrift       = "openai_endpoint_profile_role_drift"
	ReasonOpenAIEndpointProfileToolResultDrift = "openai_endpoint_profile_tool_result_drift"
	ReasonOpenAIEndpointProfileStreamDrift     = "openai_endpoint_profile_stream_drift"
	ReasonOpenAIEndpointProfileCacheDrift      = "openai_endpoint_profile_cache_drift"
	ReasonOpenAIEndpointProfileParityDrift     = "openai_endpoint_profile_parity_drift"
	ReasonOpenAIEndpointProfileCapabilityDrift = "openai_endpoint_profile_capability_drift"
)

const (
	OpenAIEndpointCapabilityStructuredInput  = "structured_input"
	OpenAIEndpointCapabilityNativeToolResult = "native_tool_result"
	OpenAIEndpointCapabilityStreaming        = "streaming"
	OpenAIEndpointCapabilityCacheUsage       = "cache_usage"
	OpenAIEndpointCapabilityCountTokens      = "count_tokens"
)

const (
	maxOpenAIEndpointProfileCases  = 32
	maxOpenAIEndpointProfileText   = 512
	maxOpenAIEndpointProfileRoles  = 16
	maxOpenAIEndpointProfileEvents = 32
)

type OpenAIEndpointProfileFixture struct {
	Version string                             `json:"version"`
	Cases   []OpenAIEndpointProfileFixtureCase `json:"cases"`
}

type OpenAIEndpointProfileFixtureCase struct {
	Name                             string            `json:"name"`
	ProfileID                        string            `json:"profile_id"`
	Endpoint                         string            `json:"endpoint"`
	Model                            string            `json:"model"`
	APIShape                         string            `json:"api_shape"`
	ProjectionCase                   string            `json:"projection_case"`
	ProjectionDigest                 string            `json:"projection_digest"`
	Capabilities                     map[string]string `json:"capabilities"`
	Roles                            []string          `json:"roles"`
	ToolResultNative                 bool              `json:"tool_result_native"`
	ToolResultCorrelated             bool              `json:"tool_result_correlated"`
	StreamEvents                     []string          `json:"stream_events"`
	ProviderSwitchAfterSemanticEvent bool              `json:"provider_switch_after_semantic_event"`
	TerminalCacheUsage               bool              `json:"terminal_cache_usage"`
	RunStreamParity                  bool              `json:"run_stream_parity"`
}

type OpenAIEndpointProfileReplayCase struct {
	Name           string   `json:"name"`
	ProfileID      string   `json:"profile_id"`
	ProjectionCase string   `json:"projection_case"`
	Verdict        string   `json:"verdict"`
	Reasons        []string `json:"reasons,omitempty"`
	Digest         string   `json:"digest"`
	ReplayDigest   string   `json:"replay_digest"`
	Idempotent     bool     `json:"idempotent"`
}

type OpenAIEndpointProfileReplayResult struct {
	Version string                            `json:"version"`
	Cases   []OpenAIEndpointProfileReplayCase `json:"cases"`
}

func ParseOpenAIEndpointProfileFixtureJSON(raw []byte) (OpenAIEndpointProfileFixture, error) {
	if len(raw) > 1<<20 {
		return OpenAIEndpointProfileFixture{}, openAIEndpointProfileError(ReasonOpenAIEndpointProfileOverflowDrift, "fixture exceeds 1 MiB")
	}
	var fixture OpenAIEndpointProfileFixture
	if err := json.Unmarshal(raw, &fixture); err != nil {
		return OpenAIEndpointProfileFixture{}, openAIEndpointProfileError(ReasonOpenAIEndpointProfileSchemaDrift, "%s", err.Error())
	}
	if err := ValidateOpenAIEndpointProfileFixture(fixture); err != nil {
		return OpenAIEndpointProfileFixture{}, err
	}
	return fixture, nil
}

func ValidateOpenAIEndpointProfileFixture(fixture OpenAIEndpointProfileFixture) error {
	if fixture.Version != FixtureVersionOpenAIEndpointProfileV1 {
		return openAIEndpointProfileError(ReasonOpenAIEndpointProfileSchemaDrift, "unsupported fixture version %q", fixture.Version)
	}
	if len(fixture.Cases) == 0 || len(fixture.Cases) > maxOpenAIEndpointProfileCases {
		return openAIEndpointProfileError(ReasonOpenAIEndpointProfileOverflowDrift, "cases must contain between one and %d items", maxOpenAIEndpointProfileCases)
	}
	seen := map[string]struct{}{}
	for i := range fixture.Cases {
		if err := ValidateOpenAIEndpointProfileCase(fixture.Cases[i]); err != nil {
			return openAIEndpointProfileError(ReasonOpenAIEndpointProfileSchemaDrift, "cases[%d]: %v", i, err)
		}
		if _, ok := seen[fixture.Cases[i].Name]; ok {
			return openAIEndpointProfileError(ReasonOpenAIEndpointProfileSchemaDrift, "duplicate case %q", fixture.Cases[i].Name)
		}
		seen[fixture.Cases[i].Name] = struct{}{}
	}
	return nil
}

func ValidateOpenAIEndpointProfileCase(in OpenAIEndpointProfileFixtureCase) error {
	for name, value := range map[string]string{
		"name": in.Name, "profile_id": in.ProfileID, "endpoint": in.Endpoint, "model": in.Model,
		"api_shape": in.APIShape, "projection_case": in.ProjectionCase, "projection_digest": in.ProjectionDigest,
	} {
		if err := validateOpenAIEndpointProfileString(name, value); err != nil {
			return err
		}
	}
	if in.APIShape != "responses.v1" {
		return openAIEndpointProfileError(ReasonOpenAIEndpointProfileSchemaDrift, "unsupported API shape %q", in.APIShape)
	}
	parsed, err := url.Parse(in.Endpoint)
	if err != nil || (parsed.Scheme != "https" && parsed.Scheme != "http") || parsed.Host == "" || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
		if parsed != nil && parsed.User != nil {
			return openAIEndpointProfileError(ReasonOpenAIEndpointProfilePrivacyDrift, "endpoint contains userinfo")
		}
		return openAIEndpointProfileError(ReasonOpenAIEndpointProfileSchemaDrift, "endpoint must be an http(s) URL without query or fragment")
	}
	if len(in.Roles) > maxOpenAIEndpointProfileRoles || len(in.StreamEvents) > maxOpenAIEndpointProfileEvents {
		return openAIEndpointProfileError(ReasonOpenAIEndpointProfileOverflowDrift, "roles or stream_events exceed bounds")
	}
	for _, role := range in.Roles {
		if err := validateOpenAIEndpointProfileString("role", role); err != nil {
			return err
		}
	}
	for _, event := range in.StreamEvents {
		if err := validateOpenAIEndpointProfileString("stream_event", event); err != nil {
			return err
		}
	}
	for capability, support := range in.Capabilities {
		if !validOpenAIEndpointCapability(capability) || !validOpenAIEndpointSupport(support) {
			return openAIEndpointProfileError(ReasonOpenAIEndpointProfileCapabilityDrift, "invalid capability declaration %q=%q", capability, support)
		}
	}
	if !validOpenAIProjectionDigest(in.ProjectionDigest) {
		return openAIEndpointProfileError(ReasonOpenAIEndpointProfileReferenceDrift, "projection_digest must be sha256 hex")
	}
	return nil
}

func ReplayOpenAIEndpointProfileFixture(fixture OpenAIEndpointProfileFixture, projectionDigests map[string]string) (OpenAIEndpointProfileReplayResult, error) {
	if err := ValidateOpenAIEndpointProfileFixture(fixture); err != nil {
		return OpenAIEndpointProfileReplayResult{}, err
	}
	result := OpenAIEndpointProfileReplayResult{Version: fixture.Version, Cases: make([]OpenAIEndpointProfileReplayCase, 0, len(fixture.Cases))}
	for _, input := range fixture.Cases {
		if projectionDigests != nil {
			key := input.ProfileID
			_ = key
			projectionKey := "openai\x00" + input.ProjectionCase
			if expected, ok := projectionDigests[projectionKey]; !ok || !strings.EqualFold(expected, input.ProjectionDigest) {
				return OpenAIEndpointProfileReplayResult{}, openAIEndpointProfileError(ReasonOpenAIEndpointProfileReferenceDrift, "%s: projection reference mismatch", input.Name)
			}
		}
		first := replayOpenAIEndpointProfileCase(input)
		second := replayOpenAIEndpointProfileCase(input)
		if first.Digest != second.Digest || first.Verdict != second.Verdict || !sameOpenAIEndpointReasons(first.Reasons, second.Reasons) {
			return OpenAIEndpointProfileReplayResult{}, openAIEndpointProfileError(ReasonOpenAIEndpointProfileSchemaDrift, "%s: replay is not deterministic", input.Name)
		}
		result.Cases = append(result.Cases, first)
	}
	return result, nil
}

func replayOpenAIEndpointProfileCase(input OpenAIEndpointProfileFixtureCase) OpenAIEndpointProfileReplayCase {
	reasons := make([]string, 0, 4)
	if input.Capabilities[OpenAIEndpointCapabilityStructuredInput] == "supported" && len(input.Roles) == 0 {
		reasons = append(reasons, ReasonOpenAIEndpointProfileRoleDrift)
	}
	if input.Capabilities[OpenAIEndpointCapabilityNativeToolResult] == "supported" && (!input.ToolResultNative || !input.ToolResultCorrelated) {
		reasons = append(reasons, ReasonOpenAIEndpointProfileToolResultDrift)
	}
	if input.Capabilities[OpenAIEndpointCapabilityStreaming] == "supported" && len(input.StreamEvents) == 0 {
		reasons = append(reasons, ReasonOpenAIEndpointProfileStreamDrift)
	}
	if input.Capabilities[OpenAIEndpointCapabilityCacheUsage] == "supported" && !input.TerminalCacheUsage {
		reasons = append(reasons, ReasonOpenAIEndpointProfileCacheDrift)
	}
	if !input.RunStreamParity {
		reasons = append(reasons, ReasonOpenAIEndpointProfileParityDrift)
	}
	if input.ProviderSwitchAfterSemanticEvent {
		reasons = append(reasons, ReasonOpenAIEndpointProfileStreamDrift)
	}
	sort.Strings(reasons)
	verdict := OpenAIEndpointProfileVerdictConformant
	if len(reasons) > 0 {
		verdict = OpenAIEndpointProfileVerdictBlocked
	}
	digest := digestOpenAIEndpointProfileCase(input)
	return OpenAIEndpointProfileReplayCase{
		Name: input.Name, ProfileID: input.ProfileID, ProjectionCase: input.ProjectionCase,
		Verdict: verdict, Reasons: reasons, Digest: digest, ReplayDigest: digest, Idempotent: true,
	}
}

func digestOpenAIEndpointProfileCase(input OpenAIEndpointProfileFixtureCase) string {
	capabilities := make([]string, 0, len(input.Capabilities))
	for key, value := range input.Capabilities {
		capabilities = append(capabilities, key+"="+value)
	}
	sort.Strings(capabilities)
	semantic := struct {
		Name, ProfileID, Endpoint, Model, APIShape, ProjectionCase, ProjectionDigest string
		Capabilities                                                                 []string `json:"capabilities"`
		Roles                                                                        []string `json:"roles"`
		ToolResultNative, ToolResultCorrelated                                       bool
		StreamEvents                                                                 []string `json:"stream_events"`
		ProviderSwitchAfterSemanticEvent                                             bool     `json:"provider_switch_after_semantic_event"`
		TerminalCacheUsage, RunStreamParity                                          bool
	}{input.Name, input.ProfileID, input.Endpoint, input.Model, input.APIShape, input.ProjectionCase, input.ProjectionDigest, capabilities, input.Roles, input.ToolResultNative, input.ToolResultCorrelated, input.StreamEvents, input.ProviderSwitchAfterSemanticEvent, input.TerminalCacheUsage, input.RunStreamParity}
	raw, _ := json.Marshal(semantic)
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}

func validateOpenAIEndpointProfileString(name, value string) error {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return openAIEndpointProfileError(ReasonOpenAIEndpointProfileSchemaDrift, "%s is required", name)
	}
	if len(trimmed) > maxOpenAIEndpointProfileText {
		return openAIEndpointProfileError(ReasonOpenAIEndpointProfileOverflowDrift, "%s exceeds %d bytes", name, maxOpenAIEndpointProfileText)
	}
	lower := strings.ToLower(trimmed)
	for _, marker := range []string{"bearer ", "password", "secret", "credential", "raw_prompt", "reasoning", "raw_payload"} {
		if strings.Contains(lower, marker) {
			return openAIEndpointProfileError(ReasonOpenAIEndpointProfilePrivacyDrift, "%s contains sensitive material", name)
		}
	}
	return nil
}

func validOpenAIEndpointCapability(value string) bool {
	switch value {
	case OpenAIEndpointCapabilityStructuredInput, OpenAIEndpointCapabilityNativeToolResult, OpenAIEndpointCapabilityStreaming, OpenAIEndpointCapabilityCacheUsage, OpenAIEndpointCapabilityCountTokens:
		return true
	default:
		return false
	}
}

func validOpenAIEndpointSupport(value string) bool {
	return value == "supported" || value == "unsupported" || value == "unknown"
}

func validOpenAIProjectionDigest(value string) bool {
	if len(value) != 64 {
		return false
	}
	for _, char := range value {
		isHex := char >= '0' && char <= '9' || char >= 'a' && char <= 'f' || char >= 'A' && char <= 'F'
		if !isHex {
			return false
		}
	}
	return true
}

func sameOpenAIEndpointReasons(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}
	return true
}

func openAIEndpointProfileError(code, format string, args ...any) error {
	return fmt.Errorf("%s: %s", code, fmt.Sprintf(format, args...))
}
