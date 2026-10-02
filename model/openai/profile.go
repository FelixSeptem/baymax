package openai

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/FelixSeptem/baymax/core/types"
)

const (
	EndpointProfileVersion   = "openai-compatible-endpoint-profile.v1"
	ResponsesAPIShapeVersion = "responses.v1"
)

type EndpointCapability string

const (
	EndpointCapabilityStructuredInput  EndpointCapability = "structured_input"
	EndpointCapabilityNativeToolResult EndpointCapability = "native_tool_result"
	EndpointCapabilityStreaming        EndpointCapability = "streaming"
	EndpointCapabilityCacheUsage       EndpointCapability = "cache_usage"
	EndpointCapabilityCountTokens      EndpointCapability = "count_tokens"
)

const (
	ReasonProfileRequired              = "openai_endpoint_profile_required"
	ReasonProfileUnsupportedVersion    = "openai_endpoint_profile_unsupported_version"
	ReasonProfileUnsupportedAPIShape   = "openai_endpoint_profile_unsupported_api_shape"
	ReasonProfileInvalidEndpoint       = "openai_endpoint_profile_invalid_endpoint"
	ReasonProfilePrivacyViolation      = "openai_endpoint_profile_privacy_violation"
	ReasonProfileOverflow              = "openai_endpoint_profile_overflow"
	ReasonProfileInvalidCapability     = "openai_endpoint_profile_invalid_capability"
	ReasonProfileEndpointMismatch      = "openai_endpoint_profile_endpoint_mismatch"
	ReasonProfileModelMismatch         = "openai_endpoint_profile_model_mismatch"
	ReasonProfileCapabilityUnsupported = "openai_endpoint_profile_capability_unsupported"
)

const (
	maxEndpointProfileID       = 128
	maxEndpointProfileEndpoint = 512
	maxEndpointProfileModel    = 256
)

// EndpointProfile is an explicit, host-owned declaration that a BaseURL may
// reuse the OpenAI Responses projection. It is not a discovery result.
type EndpointProfile struct {
	Version      string                                         `json:"version"`
	ID           string                                         `json:"id"`
	Endpoint     string                                         `json:"endpoint"`
	Model        string                                         `json:"model"`
	APIShape     string                                         `json:"api_shape"`
	Capabilities map[EndpointCapability]types.CapabilitySupport `json:"capabilities"`
}

func (p EndpointProfile) Validate() error {
	if p.Version != EndpointProfileVersion {
		return profileError(ReasonProfileUnsupportedVersion, "unsupported profile version %q", p.Version)
	}
	if err := validateProfileString("id", p.ID, maxEndpointProfileID); err != nil {
		return err
	}
	if err := validateProfileString("endpoint", p.Endpoint, maxEndpointProfileEndpoint); err != nil {
		return err
	}
	if err := validateProfileString("model", p.Model, maxEndpointProfileModel); err != nil {
		return err
	}
	if p.APIShape != ResponsesAPIShapeVersion {
		return profileError(ReasonProfileUnsupportedAPIShape, "unsupported API shape %q", p.APIShape)
	}
	u, err := url.Parse(strings.TrimSpace(p.Endpoint))
	if err != nil || u.Scheme != "https" && u.Scheme != "http" || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" {
		return profileError(ReasonProfileInvalidEndpoint, "endpoint must be an http(s) URL without userinfo, query, or fragment")
	}
	for capability, support := range p.Capabilities {
		if !validEndpointCapability(capability) || support != types.CapabilitySupportSupported && support != types.CapabilitySupportUnsupported && support != types.CapabilitySupportUnknown {
			return profileError(ReasonProfileInvalidCapability, "capability %q has unsupported declaration %q", capability, support)
		}
	}
	return nil
}

func (p EndpointProfile) Capability(capability EndpointCapability) types.CapabilitySupport {
	if support, ok := p.Capabilities[capability]; ok {
		return support
	}
	return types.CapabilitySupportUnknown
}

func validateProfileString(name, value string, limit int) error {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return profileError(ReasonProfileInvalidEndpoint, "%s is required", name)
	}
	if len(trimmed) > limit {
		return profileError(ReasonProfileOverflow, "%s exceeds %d bytes", name, limit)
	}
	lower := strings.ToLower(trimmed)
	for _, marker := range []string{"bearer ", "password", "secret", "credential", "raw_prompt", "reasoning"} {
		if strings.Contains(lower, marker) {
			return profileError(ReasonProfilePrivacyViolation, "%s contains sensitive material", name)
		}
	}
	return nil
}

func validEndpointCapability(value EndpointCapability) bool {
	switch value {
	case EndpointCapabilityStructuredInput, EndpointCapabilityNativeToolResult, EndpointCapabilityStreaming, EndpointCapabilityCacheUsage, EndpointCapabilityCountTokens:
		return true
	default:
		return false
	}
}

func profileError(code, format string, args ...any) error {
	return fmt.Errorf("%s: %s", code, fmt.Sprintf(format, args...))
}

func normalizeProfileEndpoint(value string) string {
	return strings.TrimRight(strings.ToLower(strings.TrimSpace(value)), "/")
}
