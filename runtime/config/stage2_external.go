package config

import (
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strings"
)

type PrecheckSeverity string

const (
	PrecheckSeverityWarning PrecheckSeverity = "warning"
	PrecheckSeverityError   PrecheckSeverity = "error"
)

const (
	Stage2TemplateResolutionProfileDefaultsOnly         = "profile_defaults_only"
	Stage2TemplateResolutionProfileDefaultsWithOverride = "profile_defaults_then_explicit_overrides"
	Stage2TemplateResolutionExplicitOnly                = "explicit_only"
)

var hintCapabilityPattern = regexp.MustCompile(`^[a-z0-9._/\-]+$`)

type ExternalPrecheckFinding struct {
	Severity PrecheckSeverity `json:"severity"`
	Code     string           `json:"code"`
	Field    string           `json:"field,omitempty"`
	Message  string           `json:"message"`
}

type ExternalPrecheckResult struct {
	Normalized ContextAssemblerCA2ExternalConfig `json:"normalized"`
	Findings   []ExternalPrecheckFinding         `json:"findings,omitempty"`
}

func (r ExternalPrecheckResult) FirstError() error {
	for _, finding := range r.Findings {
		if finding.Severity != PrecheckSeverityError {
			continue
		}
		if finding.Field != "" {
			return fmt.Errorf("%s: %s", finding.Field, finding.Message)
		}
		return errors.New(finding.Message)
	}
	return nil
}

func (r ExternalPrecheckResult) HasWarnings() bool {
	for _, finding := range r.Findings {
		if finding.Severity == PrecheckSeverityWarning {
			return true
		}
	}
	return false
}

func SupportedStage2ExternalProfiles() []string {
	return []string{
		ContextStage2ExternalProfileHTTPGeneric,
		ContextStage2ExternalProfileRAGFlowLike,
		ContextStage2ExternalProfileGraphRAGLike,
		ContextStage2ExternalProfileElasticsearchLike,
		ContextStage2ExternalProfileExplicitOnly,
	}
}

func SupportedStage2TemplatePackProfiles() []string {
	return []string{
		ContextStage2ExternalProfileRAGFlowLike,
		ContextStage2ExternalProfileGraphRAGLike,
		ContextStage2ExternalProfileElasticsearchLike,
	}
}

func applyStage2ExternalProfile(in ContextAssemblerCA2ExternalConfig) ContextAssemblerCA2ExternalConfig {
	base := DefaultConfig().ContextAssembler.CA2.Stage2.External
	profile := strings.ToLower(strings.TrimSpace(in.Profile))
	if profile == "" {
		profile = ContextStage2ExternalProfileHTTPGeneric
	}
	out := base
	switch profile {
	case ContextStage2ExternalProfileHTTPGeneric:
		// keep base defaults
	case ContextStage2ExternalProfileExplicitOnly:
		// keep base defaults; explicit fields still override after profile application.
	case ContextStage2ExternalProfileRAGFlowLike:
		out.Mapping.Request.QueryField = "question"
		out.Mapping.Request.MaxItemsField = "top_k"
		out.Mapping.Response.ChunksField = "data.chunks"
		out.Mapping.Response.SourceField = "data.source"
		out.Mapping.Response.ReasonField = "data.reason"
		out.Mapping.Response.ErrorField = "error"
		out.Mapping.Response.ErrorMessageField = "error.message"
	case ContextStage2ExternalProfileGraphRAGLike:
		out.Mapping.Request.QueryField = "query.text"
		out.Mapping.Request.SessionIDField = "query.session_id"
		out.Mapping.Request.RunIDField = "query.run_id"
		out.Mapping.Request.MaxItemsField = "query.top_k"
		out.Mapping.Response.ChunksField = "result.chunks"
		out.Mapping.Response.SourceField = "result.source"
		out.Mapping.Response.ReasonField = "result.reason"
		out.Mapping.Response.ErrorField = "error"
		out.Mapping.Response.ErrorMessageField = "error.message"
	case ContextStage2ExternalProfileElasticsearchLike:
		out.Mapping.Request.QueryField = "query"
		out.Mapping.Request.MaxItemsField = "size"
		out.Mapping.Response.ChunksField = "hits.chunks"
		out.Mapping.Response.SourceField = "meta.source"
		out.Mapping.Response.ReasonField = "meta.reason"
		out.Mapping.Response.ErrorField = "error"
		out.Mapping.Response.ErrorMessageField = "error.message"
	default:
		// Preserve user input for invalid profile and let precheck return explicit error.
	}
	out.Profile = profile
	out.TemplateResolutionSource = resolveStage2TemplateResolutionSource(profile, false)
	return out
}

func mergeExternalOverrides(base, override ContextAssemblerCA2ExternalConfig) ContextAssemblerCA2ExternalConfig {
	if strings.TrimSpace(override.Profile) != "" {
		base.Profile = strings.ToLower(strings.TrimSpace(override.Profile))
	}
	if strings.TrimSpace(override.TemplateResolutionSource) != "" {
		base.TemplateResolutionSource = strings.ToLower(strings.TrimSpace(override.TemplateResolutionSource))
	}
	if strings.TrimSpace(override.Endpoint) != "" {
		base.Endpoint = strings.TrimSpace(override.Endpoint)
	}
	if strings.TrimSpace(override.Method) != "" {
		base.Method = strings.ToUpper(strings.TrimSpace(override.Method))
	}
	if strings.TrimSpace(override.Auth.BearerToken) != "" {
		base.Auth.BearerToken = strings.TrimSpace(override.Auth.BearerToken)
	}
	if strings.TrimSpace(override.Auth.HeaderName) != "" {
		base.Auth.HeaderName = strings.TrimSpace(override.Auth.HeaderName)
	}
	if len(override.Headers) > 0 {
		base.Headers = normalizeStringMap(override.Headers)
	}
	if strings.TrimSpace(override.Mapping.Request.Mode) != "" {
		base.Mapping.Request.Mode = strings.ToLower(strings.TrimSpace(override.Mapping.Request.Mode))
	}
	if strings.TrimSpace(override.Mapping.Request.MethodName) != "" {
		base.Mapping.Request.MethodName = strings.TrimSpace(override.Mapping.Request.MethodName)
	}
	if strings.TrimSpace(override.Mapping.Request.JSONRPCVersion) != "" {
		base.Mapping.Request.JSONRPCVersion = strings.TrimSpace(override.Mapping.Request.JSONRPCVersion)
	}
	if strings.TrimSpace(override.Mapping.Request.QueryField) != "" {
		base.Mapping.Request.QueryField = strings.TrimSpace(override.Mapping.Request.QueryField)
	}
	if strings.TrimSpace(override.Mapping.Request.SessionIDField) != "" {
		base.Mapping.Request.SessionIDField = strings.TrimSpace(override.Mapping.Request.SessionIDField)
	}
	if strings.TrimSpace(override.Mapping.Request.RunIDField) != "" {
		base.Mapping.Request.RunIDField = strings.TrimSpace(override.Mapping.Request.RunIDField)
	}
	if strings.TrimSpace(override.Mapping.Request.MaxItemsField) != "" {
		base.Mapping.Request.MaxItemsField = strings.TrimSpace(override.Mapping.Request.MaxItemsField)
	}
	if strings.TrimSpace(override.Mapping.Response.ChunksField) != "" {
		base.Mapping.Response.ChunksField = strings.TrimSpace(override.Mapping.Response.ChunksField)
	}
	if strings.TrimSpace(override.Mapping.Response.SourceField) != "" {
		base.Mapping.Response.SourceField = strings.TrimSpace(override.Mapping.Response.SourceField)
	}
	if strings.TrimSpace(override.Mapping.Response.ReasonField) != "" {
		base.Mapping.Response.ReasonField = strings.TrimSpace(override.Mapping.Response.ReasonField)
	}
	if strings.TrimSpace(override.Mapping.Response.ErrorField) != "" {
		base.Mapping.Response.ErrorField = strings.TrimSpace(override.Mapping.Response.ErrorField)
	}
	if strings.TrimSpace(override.Mapping.Response.ErrorMessageField) != "" {
		base.Mapping.Response.ErrorMessageField = strings.TrimSpace(override.Mapping.Response.ErrorMessageField)
	}
	base.Hints.Enabled = override.Hints.Enabled
	if len(override.Hints.Capabilities) > 0 {
		base.Hints.Capabilities = normalizeHintCapabilities(override.Hints.Capabilities)
	}
	return base
}

func PrecheckStage2External(provider string, cfg ContextAssemblerCA2ExternalConfig) ExternalPrecheckResult {
	out := mergeExternalOverrides(applyStage2ExternalProfile(ContextAssemblerCA2ExternalConfig{Profile: cfg.Profile}), cfg)
	findings := make([]ExternalPrecheckFinding, 0, 8)
	providerName := strings.ToLower(strings.TrimSpace(provider))

	if !slices.Contains(SupportedStage2ExternalProfiles(), out.Profile) {
		findings = append(findings, ExternalPrecheckFinding{
			Severity: PrecheckSeverityError,
			Code:     "invalid_profile",
			Field:    "context_assembler.ca2.stage2.external.profile",
			Message:  fmt.Sprintf("unsupported profile %q", cfg.Profile),
		})
	}
	if strings.TrimSpace(out.TemplateResolutionSource) == "" {
		out.TemplateResolutionSource = resolveStage2TemplateResolutionSource(out.Profile, true)
	}
	if providerName != ContextStage2ProviderFile && strings.TrimSpace(out.Endpoint) == "" {
		findings = append(findings, ExternalPrecheckFinding{
			Severity: PrecheckSeverityError,
			Code:     "missing_endpoint",
			Field:    "context_assembler.ca2.stage2.external.endpoint",
			Message:  "context_assembler.ca2.stage2.external.endpoint is required for non-file providers",
		})
	}
	method := strings.ToUpper(strings.TrimSpace(out.Method))
	switch method {
	case "", "POST", "PUT":
	default:
		findings = append(findings, ExternalPrecheckFinding{
			Severity: PrecheckSeverityError,
			Code:     "invalid_http_method",
			Field:    "context_assembler.ca2.stage2.external.method",
			Message:  fmt.Sprintf("context_assembler.ca2.stage2.external.method must be one of [POST,PUT], got %q", out.Method),
		})
	}

	mode := strings.ToLower(strings.TrimSpace(out.Mapping.Request.Mode))
	switch mode {
	case "", "plain", "jsonrpc2":
	default:
		findings = append(findings, ExternalPrecheckFinding{
			Severity: PrecheckSeverityError,
			Code:     "invalid_request_mode",
			Field:    "context_assembler.ca2.stage2.external.mapping.request.mode",
			Message:  fmt.Sprintf("context_assembler.ca2.stage2.external.mapping.request.mode must be one of [plain,jsonrpc2], got %q", out.Mapping.Request.Mode),
		})
	}
	if strings.TrimSpace(out.Mapping.Request.QueryField) == "" {
		findings = append(findings, ExternalPrecheckFinding{
			Severity: PrecheckSeverityError,
			Code:     "missing_query_field",
			Field:    "context_assembler.ca2.stage2.external.mapping.request.query_field",
			Message:  "context_assembler.ca2.stage2.external.mapping.request.query_field is required",
		})
	}
	if strings.TrimSpace(out.Mapping.Response.ChunksField) == "" {
		findings = append(findings, ExternalPrecheckFinding{
			Severity: PrecheckSeverityError,
			Code:     "missing_chunks_field",
			Field:    "context_assembler.ca2.stage2.external.mapping.response.chunks_field",
			Message:  "context_assembler.ca2.stage2.external.mapping.response.chunks_field is required",
		})
	}
	if mode == "jsonrpc2" && strings.TrimSpace(out.Mapping.Request.MethodName) == "" {
		findings = append(findings, ExternalPrecheckFinding{
			Severity: PrecheckSeverityError,
			Code:     "missing_method_name",
			Field:    "context_assembler.ca2.stage2.external.mapping.request.method_name",
			Message:  "context_assembler.ca2.stage2.external.mapping.request.method_name is required when mode=jsonrpc2",
		})
	}
	if strings.EqualFold(strings.TrimSpace(out.Mapping.Request.QueryField), strings.TrimSpace(out.Mapping.Response.ChunksField)) &&
		strings.TrimSpace(out.Mapping.Request.QueryField) != "" {
		findings = append(findings, ExternalPrecheckFinding{
			Severity: PrecheckSeverityError,
			Code:     "mapping_field_conflict",
			Field:    "context_assembler.ca2.stage2.external.mapping",
			Message:  "request.query_field and response.chunks_field must not point to the same path",
		})
	}
	if strings.TrimSpace(out.Auth.BearerToken) == "" {
		findings = append(findings, ExternalPrecheckFinding{
			Severity: PrecheckSeverityWarning,
			Code:     "missing_auth_token",
			Field:    "context_assembler.ca2.stage2.external.auth.bearer_token",
			Message:  "external retriever bearer token is empty; ensure upstream endpoint accepts anonymous requests",
		})
	}
	out.Hints.Capabilities = normalizeHintCapabilities(out.Hints.Capabilities)
	if out.Hints.Enabled {
		if len(out.Hints.Capabilities) == 0 {
			findings = append(findings, ExternalPrecheckFinding{
				Severity: PrecheckSeverityError,
				Code:     "missing_hint_capabilities",
				Field:    "context_assembler.ca2.stage2.external.hints.capabilities",
				Message:  "context_assembler.ca2.stage2.external.hints.capabilities is required when hints.enabled=true",
			})
		}
		for i, capability := range out.Hints.Capabilities {
			if !hintCapabilityPattern.MatchString(capability) {
				findings = append(findings, ExternalPrecheckFinding{
					Severity: PrecheckSeverityError,
					Code:     "invalid_hint_capability",
					Field:    fmt.Sprintf("context_assembler.ca2.stage2.external.hints.capabilities[%d]", i),
					Message:  fmt.Sprintf("invalid hint capability %q: allowed charset is [a-z0-9._/-]", capability),
				})
			}
		}
	}

	return ExternalPrecheckResult{
		Normalized: out,
		Findings:   findings,
	}
}

func normalizeHintCapabilities(in []string) []string {
	if len(in) == 0 {
		return []string{}
	}
	out := make([]string, 0, len(in))
	seen := map[string]struct{}{}
	for _, item := range in {
		for _, chunk := range strings.Split(item, ",") {
			capability := strings.ToLower(strings.TrimSpace(chunk))
			if capability == "" {
				continue
			}
			if _, ok := seen[capability]; ok {
				continue
			}
			seen[capability] = struct{}{}
			out = append(out, capability)
		}
	}
	return out
}

func resolveStage2TemplateResolutionSource(profile string, explicitOverrides bool) string {
	normalizedProfile := strings.ToLower(strings.TrimSpace(profile))
	if normalizedProfile == "" || normalizedProfile == ContextStage2ExternalProfileExplicitOnly {
		return Stage2TemplateResolutionExplicitOnly
	}
	if explicitOverrides {
		return Stage2TemplateResolutionProfileDefaultsWithOverride
	}
	return Stage2TemplateResolutionProfileDefaultsOnly
}
