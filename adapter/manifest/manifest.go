package manifest

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"regexp"
	"runtime"
	"sort"
	"strings"

	adaptercap "github.com/FelixSeptem/baymax/adapter/capability"
	adapterprofile "github.com/FelixSeptem/baymax/adapter/profile"
)

const (
	CodeMissingFile                = "adapter-manifest.missing-file"
	CodeInvalidJSON                = "adapter-manifest.invalid-json"
	CodeMissingField               = "adapter-manifest.missing-field"
	CodeInvalidField               = "adapter-manifest.invalid-field"
	CodeInvalidCompatExpression    = "adapter-manifest.invalid-compat-expression"
	CodeCompatibilityMismatch      = "adapter-manifest.compatibility-mismatch"
	CodeRequiredCapabilityMissing  = "adapter-manifest.required-capability-missing"
	CodeInvalidNegotiationConfig   = "adapter-manifest.invalid-negotiation-config"
	CodeUnknownContractProfile     = "adapter-manifest.unknown-contract-profile-version"
	CodeContractProfileOutOfWindow = "adapter-manifest.contract-profile-out-of-window"
	CodeSandboxProfileUnknown      = "adapter-manifest.sandbox-profile-unknown"
	CodeSandboxBackendUnsupported  = "adapter-manifest.sandbox-backend-unsupported"
	CodeSandboxHostMismatch        = "adapter-manifest.sandbox-host-mismatch"
	CodeSandboxSessionUnsupported  = "adapter-manifest.sandbox-session-mode-unsupported"
)

var (
	namePattern               = regexp.MustCompile(`^[a-z][a-z0-9-]*$`)
	conformanceProfilePattern = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)
)

type Capabilities struct {
	Required []string `json:"required"`
	Optional []string `json:"optional"`
}

type Manifest struct {
	Type                   string       `json:"type"`
	Name                   string       `json:"name"`
	Version                string       `json:"version"`
	ContractProfileVersion string       `json:"contract_profile_version"`
	BaymaxCompat           string       `json:"baymax_compat"`
	Capabilities           Capabilities `json:"capabilities"`
	Negotiation            Negotiation  `json:"negotiation,omitempty"`
	ConformanceProfile     string       `json:"conformance_profile"`
	SandboxBackend         string       `json:"sandbox_backend,omitempty"`
	SandboxProfileID       string       `json:"sandbox_profile_id,omitempty"`
	HostOS                 string       `json:"host_os,omitempty"`
	HostArch               string       `json:"host_arch,omitempty"`
	SessionModesSupported  []string     `json:"session_modes_supported,omitempty"`
}

type Negotiation struct {
	DefaultStrategy      string `json:"default_strategy,omitempty"`
	AllowRequestOverride bool   `json:"allow_request_override,omitempty"`
}

type ContractError struct {
	Code    string
	Field   string
	Message string
}

func (e *ContractError) Error() string {
	if e == nil {
		return ""
	}
	if strings.TrimSpace(e.Field) != "" {
		return fmt.Sprintf("[%s] %s: %s", e.Code, e.Field, e.Message)
	}
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

type OptionalDowngrade struct {
	Capability string
	ReasonCode string
}

type CapabilityRequest struct {
	Required         []string
	Optional         []string
	StrategyOverride string
}

type ActivationContext struct {
	HostOS            string
	HostArch          string
	RequestedSession  string
	SupportedBackends []string
}

type ActivationResult struct {
	OptionalDowngrades     []OptionalDowngrade
	ContractProfileVersion string
	StrategyApplied        string
	StrategyOverride       bool
	MissingRequired        []string
	MissingOptional        []string
	ReasonCodes            []string
	Diagnostics            adaptercap.Diagnostics
}

func LoadFile(path string) (Manifest, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Manifest{}, &ContractError{
				Code:    CodeMissingFile,
				Field:   "manifest",
				Message: "manifest file does not exist",
			}
		}
		return Manifest{}, err
	}
	return Parse(raw)
}

func Parse(raw []byte) (Manifest, error) {
	var manifest Manifest
	if err := json.Unmarshal(raw, &manifest); err != nil {
		return Manifest{}, &ContractError{
			Code:    CodeInvalidJSON,
			Field:   "manifest",
			Message: "manifest JSON decode failed",
		}
	}
	if err := Validate(manifest); err != nil {
		return Manifest{}, err
	}
	return normalize(manifest), nil
}

func Validate(manifest Manifest) error {
	normalized := normalize(manifest)

	orderedChecks := []struct {
		value string
		field string
	}{
		{value: normalized.Type, field: "type"},
		{value: normalized.Name, field: "name"},
		{value: normalized.Version, field: "version"},
		{value: normalized.ContractProfileVersion, field: "contract_profile_version"},
		{value: normalized.BaymaxCompat, field: "baymax_compat"},
		{value: normalized.ConformanceProfile, field: "conformance_profile"},
	}
	for _, check := range orderedChecks {
		if check.value == "" {
			return &ContractError{
				Code:    CodeMissingField,
				Field:   check.field,
				Message: "required field is missing",
			}
		}
	}
	if len(normalized.Capabilities.Required) == 0 {
		return &ContractError{
			Code:    CodeMissingField,
			Field:   "capabilities.required",
			Message: "required field is missing",
		}
	}
	if normalized.Capabilities.Optional == nil {
		return &ContractError{
			Code:    CodeMissingField,
			Field:   "capabilities.optional",
			Message: "required field is missing",
		}
	}

	switch normalized.Type {
	case "mcp", "model", "tool":
	default:
		return &ContractError{
			Code:    CodeInvalidField,
			Field:   "type",
			Message: "type must be one of: mcp|model|tool",
		}
	}
	if !namePattern.MatchString(normalized.Name) {
		return &ContractError{
			Code:    CodeInvalidField,
			Field:   "name",
			Message: "name must match ^[a-z][a-z0-9-]*$",
		}
	}
	if _, err := parseSemver(normalized.Version); err != nil {
		return &ContractError{
			Code:    CodeInvalidField,
			Field:   "version",
			Message: "version must be valid semver",
		}
	}
	if !conformanceProfilePattern.MatchString(normalized.ConformanceProfile) {
		return &ContractError{
			Code:    CodeInvalidField,
			Field:   "conformance_profile",
			Message: "conformance_profile must match ^[a-z0-9]+(-[a-z0-9]+)*$",
		}
	}
	if hasEmptyCapability(normalized.Capabilities.Required) {
		return &ContractError{
			Code:    CodeInvalidField,
			Field:   "capabilities.required",
			Message: "capability names must not be empty",
		}
	}
	if hasEmptyCapability(normalized.Capabilities.Optional) {
		return &ContractError{
			Code:    CodeInvalidField,
			Field:   "capabilities.optional",
			Message: "capability names must not be empty",
		}
	}
	if _, err := parseSemverRange(normalized.BaymaxCompat); err != nil {
		return &ContractError{
			Code:    CodeInvalidCompatExpression,
			Field:   "baymax_compat",
			Message: "invalid semver range expression",
		}
	}
	if _, err := adapterprofile.Parse(normalized.ContractProfileVersion); err != nil {
		return mapProfileError(err)
	}
	if strings.TrimSpace(normalized.Negotiation.DefaultStrategy) != "" && !adaptercap.IsStrategy(normalized.Negotiation.DefaultStrategy) {
		return &ContractError{
			Code:    CodeInvalidNegotiationConfig,
			Field:   "negotiation.default_strategy",
			Message: "default_strategy must be one of [fail_fast,best_effort]",
		}
	}
	return validateSandboxMetadata(normalized)
}

func Activate(manifest Manifest, runtimeVersion string, availableCapabilities []string) (ActivationResult, error) {
	return ActivateWithRequest(manifest, runtimeVersion, availableCapabilities, CapabilityRequest{
		Required: append([]string(nil), manifest.Capabilities.Required...),
		Optional: append([]string(nil), manifest.Capabilities.Optional...),
	})
}

func ActivateWithRequest(manifest Manifest, runtimeVersion string, availableCapabilities []string, request CapabilityRequest) (ActivationResult, error) {
	return ActivateWithRequestAndProfileWindow(manifest, runtimeVersion, availableCapabilities, request, adapterprofile.DefaultWindow())
}

func ActivateWithRequestAndProfileWindow(manifest Manifest, runtimeVersion string, availableCapabilities []string, request CapabilityRequest, profileWindow adapterprofile.Window) (ActivationResult, error) {
	return ActivateWithRequestAndProfileWindowWithContext(manifest, runtimeVersion, availableCapabilities, request, profileWindow, ActivationContext{})
}

func ActivateWithRequestAndProfileWindowWithContext(manifest Manifest, runtimeVersion string, availableCapabilities []string, request CapabilityRequest, profileWindow adapterprofile.Window, activationCtx ActivationContext) (ActivationResult, error) {
	normalized := normalize(manifest)
	if err := Validate(normalized); err != nil {
		return ActivationResult{}, err
	}
	ok, err := evaluateSemverRange(normalized.BaymaxCompat, runtimeVersion)
	if err != nil {
		return ActivationResult{}, &ContractError{
			Code:    CodeInvalidCompatExpression,
			Field:   "baymax_compat",
			Message: "invalid semver range expression",
		}
	}
	if !ok {
		return ActivationResult{}, &ContractError{
			Code:    CodeCompatibilityMismatch,
			Field:   "baymax_compat",
			Message: "runtime version does not satisfy baymax_compat",
		}
	}
	profileVersion, err := adapterprofile.ValidateCompatibility(normalized.ContractProfileVersion, profileWindow)
	if err != nil {
		return ActivationResult{}, mapProfileError(err)
	}
	if err := validateSandboxActivationCompatibility(normalized, activationCtx); err != nil {
		return ActivationResult{}, err
	}

	available := make(map[string]struct{}, len(availableCapabilities))
	for _, capability := range availableCapabilities {
		key := normalizeCapability(capability)
		if key == "" {
			continue
		}
		available[key] = struct{}{}
	}
	supportedSet := adaptercap.Set{
		Required: filterAvailableCapabilities(normalized.Capabilities.Required, available),
		Optional: filterAvailableCapabilities(normalized.Capabilities.Optional, available),
	}

	defaultStrategy := strings.TrimSpace(normalized.Negotiation.DefaultStrategy)
	if defaultStrategy == "" {
		defaultStrategy = adaptercap.StrategyFailFast
	}
	reqStrategy := strings.TrimSpace(request.StrategyOverride)
	if reqStrategy != "" && !normalized.Negotiation.AllowRequestOverride {
		return ActivationResult{}, &ContractError{
			Code:    CodeInvalidNegotiationConfig,
			Field:   "negotiation.allow_request_override",
			Message: "request strategy override is not allowed by manifest negotiation policy",
		}
	}

	outcome, err := adaptercap.Negotiate(defaultStrategy, adaptercap.Set{
		Required: append([]string(nil), supportedSet.Required...),
		Optional: append([]string(nil), supportedSet.Optional...),
	}, adaptercap.Request{
		Required:         append([]string(nil), request.Required...),
		Optional:         append([]string(nil), request.Optional...),
		StrategyOverride: reqStrategy,
	})
	if err != nil {
		return ActivationResult{}, err
	}

	if !outcome.Accepted {
		return ActivationResult{}, &ContractError{
			Code:    CodeRequiredCapabilityMissing,
			Field:   "capabilities.required",
			Message: "required capability unavailable: " + strings.Join(outcome.MissingRequired, ","),
		}
	}

	downgrades := make([]OptionalDowngrade, 0, len(outcome.DowngradedOptional))
	for _, missingOptional := range outcome.DowngradedOptional {
		downgrades = append(downgrades, OptionalDowngrade{
			Capability: missingOptional,
			ReasonCode: "adapter.manifest.capability.optional_missing." + reasonSegment(missingOptional),
		})
	}

	sort.Slice(downgrades, func(i, j int) bool {
		return downgrades[i].Capability < downgrades[j].Capability
	})
	return ActivationResult{
		OptionalDowngrades:     downgrades,
		ContractProfileVersion: profileVersion.String(),
		StrategyApplied:        outcome.AppliedStrategy,
		StrategyOverride:       outcome.StrategyOverrideApplied,
		MissingRequired:        append([]string(nil), outcome.MissingRequired...),
		MissingOptional:        append([]string(nil), outcome.MissingOptional...),
		ReasonCodes:            append([]string(nil), outcome.Reasons...),
		Diagnostics:            outcome.Diagnostics,
	}, nil
}

func normalize(manifest Manifest) Manifest {
	manifest.Type = strings.ToLower(strings.TrimSpace(manifest.Type))
	manifest.Name = strings.ToLower(strings.TrimSpace(manifest.Name))
	manifest.Version = strings.TrimSpace(manifest.Version)
	manifest.ContractProfileVersion = strings.ToLower(strings.TrimSpace(manifest.ContractProfileVersion))
	manifest.BaymaxCompat = strings.TrimSpace(manifest.BaymaxCompat)
	manifest.Negotiation.DefaultStrategy = strings.ToLower(strings.TrimSpace(manifest.Negotiation.DefaultStrategy))
	manifest.ConformanceProfile = strings.ToLower(strings.TrimSpace(manifest.ConformanceProfile))
	manifest.SandboxBackend = strings.ToLower(strings.TrimSpace(manifest.SandboxBackend))
	manifest.SandboxProfileID = strings.ToLower(strings.TrimSpace(manifest.SandboxProfileID))
	manifest.HostOS = strings.ToLower(strings.TrimSpace(manifest.HostOS))
	manifest.HostArch = strings.ToLower(strings.TrimSpace(manifest.HostArch))
	manifest.SessionModesSupported = normalizeSessionModes(manifest.SessionModesSupported)
	manifest.Capabilities.Required = normalizeCapabilities(manifest.Capabilities.Required)
	manifest.Capabilities.Optional = normalizeCapabilities(manifest.Capabilities.Optional)
	return manifest
}

func normalizeCapabilities(raw []string) []string {
	if raw == nil {
		return nil
	}
	out := make([]string, 0, len(raw))
	seen := map[string]struct{}{}
	for _, capability := range raw {
		key := normalizeCapability(capability)
		if key == "" {
			out = append(out, "")
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, key)
	}
	sort.Strings(out)
	return out
}

func normalizeCapability(in string) string {
	return strings.ToLower(strings.TrimSpace(in))
}

func hasEmptyCapability(items []string) bool {
	for _, item := range items {
		if strings.TrimSpace(item) == "" {
			return true
		}
	}
	return false
}

func filterAvailableCapabilities(items []string, available map[string]struct{}) []string {
	if len(items) == 0 || len(available) == 0 {
		return nil
	}
	out := make([]string, 0, len(items))
	for _, item := range items {
		key := normalizeCapability(item)
		if key == "" {
			continue
		}
		if _, ok := available[key]; !ok {
			continue
		}
		out = append(out, key)
	}
	return out
}

func reasonSegment(capability string) string {
	raw := strings.ToLower(strings.TrimSpace(capability))
	if raw == "" {
		return "unknown"
	}
	builder := strings.Builder{}
	lastUnderscore := false
	for _, ch := range raw {
		valid := (ch >= 'a' && ch <= 'z') || (ch >= '0' && ch <= '9')
		if valid {
			builder.WriteRune(ch)
			lastUnderscore = false
			continue
		}
		if !lastUnderscore {
			builder.WriteRune('_')
			lastUnderscore = true
		}
	}
	out := strings.Trim(builder.String(), "_")
	if out == "" {
		return "unknown"
	}
	return out
}

func validateSandboxMetadata(manifest Manifest) error {
	if !isSandboxManifest(manifest) {
		return nil
	}

	requiredChecks := []struct {
		field string
		value string
	}{
		{field: "sandbox_backend", value: manifest.SandboxBackend},
		{field: "sandbox_profile_id", value: manifest.SandboxProfileID},
		{field: "host_os", value: manifest.HostOS},
		{field: "host_arch", value: manifest.HostArch},
	}
	for _, check := range requiredChecks {
		if strings.TrimSpace(check.value) == "" {
			return &ContractError{
				Code:    CodeMissingField,
				Field:   check.field,
				Message: "required field is missing",
			}
		}
	}
	if len(manifest.SessionModesSupported) == 0 {
		return &ContractError{
			Code:    CodeMissingField,
			Field:   "session_modes_supported",
			Message: "required field is missing",
		}
	}
	if !IsSupportedSandboxBackend(manifest.SandboxBackend) {
		return &ContractError{
			Code:    CodeInvalidField,
			Field:   "sandbox_backend",
			Message: "sandbox_backend must be one of: linux_nsjail|linux_bwrap|oci_runtime|windows_job",
		}
	}
	profile, ok := SandboxProfileByID(manifest.SandboxProfileID)
	if !ok {
		return &ContractError{
			Code:    CodeSandboxProfileUnknown,
			Field:   "sandbox_profile_id",
			Message: "sandbox_profile_id is not recognized in profile-pack",
		}
	}
	if profile.Backend != manifest.SandboxBackend {
		return &ContractError{
			Code:    CodeInvalidField,
			Field:   "sandbox_backend",
			Message: "sandbox_backend does not match profile-pack backend mapping",
		}
	}
	if profile.HostOS != manifest.HostOS {
		return &ContractError{
			Code:    CodeInvalidField,
			Field:   "host_os",
			Message: "host_os does not match profile-pack host constraint",
		}
	}
	if profile.HostArch != manifest.HostArch {
		return &ContractError{
			Code:    CodeInvalidField,
			Field:   "host_arch",
			Message: "host_arch does not match profile-pack host constraint",
		}
	}
	supportedSessionModes := toStringSet(profile.SessionModesSupported)
	for _, mode := range manifest.SessionModesSupported {
		if _, ok := supportedSessionModes[mode]; ok {
			continue
		}
		return &ContractError{
			Code:    CodeInvalidField,
			Field:   "session_modes_supported",
			Message: "session_modes_supported contains mode not covered by profile-pack",
		}
	}
	return nil
}

func isSandboxManifest(manifest Manifest) bool {
	return strings.TrimSpace(manifest.SandboxBackend) != "" ||
		strings.TrimSpace(manifest.SandboxProfileID) != "" ||
		strings.TrimSpace(manifest.HostOS) != "" ||
		strings.TrimSpace(manifest.HostArch) != "" ||
		len(manifest.SessionModesSupported) > 0
}

func normalizeSessionModes(items []string) []string {
	if items == nil {
		return nil
	}
	out := make([]string, 0, len(items))
	seen := map[string]struct{}{}
	for _, raw := range items {
		mode := strings.ToLower(strings.TrimSpace(raw))
		if mode == "" {
			out = append(out, "")
			continue
		}
		switch mode {
		case SandboxSessionModePerCall, SandboxSessionModePerSession:
		default:
			out = append(out, mode)
			continue
		}
		if _, ok := seen[mode]; ok {
			continue
		}
		seen[mode] = struct{}{}
		out = append(out, mode)
	}
	sort.Strings(out)
	return out
}

func validateSandboxActivationCompatibility(manifest Manifest, in ActivationContext) error {
	if !isSandboxManifest(manifest) {
		return nil
	}
	profile, ok := SandboxProfileByID(manifest.SandboxProfileID)
	if !ok {
		return &ContractError{
			Code:    CodeSandboxProfileUnknown,
			Field:   "sandbox_profile_id",
			Message: "sandbox_profile_id is not recognized in profile-pack",
		}
	}
	ctx := normalizeActivationContext(in)
	if len(ctx.SupportedBackends) > 0 {
		backends := toStringSet(ctx.SupportedBackends)
		if _, ok := backends[manifest.SandboxBackend]; !ok {
			return &ContractError{
				Code:    CodeSandboxBackendUnsupported,
				Field:   "sandbox_backend",
				Message: "runtime does not support manifest sandbox_backend",
			}
		}
	}
	if manifest.HostOS != "" && manifest.HostOS != ctx.HostOS {
		return &ContractError{
			Code:    CodeSandboxHostMismatch,
			Field:   "host_os",
			Message: "host_os does not match current host",
		}
	}
	if manifest.HostArch != "" && manifest.HostArch != ctx.HostArch {
		return &ContractError{
			Code:    CodeSandboxHostMismatch,
			Field:   "host_arch",
			Message: "host_arch does not match current host",
		}
	}
	requested := strings.TrimSpace(ctx.RequestedSession)
	if requested != "" {
		allowed := toStringSet(manifest.SessionModesSupported)
		if _, ok := allowed[requested]; !ok {
			return &ContractError{
				Code:    CodeSandboxSessionUnsupported,
				Field:   "session_modes_supported",
				Message: "requested session mode is unsupported by manifest",
			}
		}
		if _, ok := toStringSet(profile.SessionModesSupported)[requested]; !ok {
			return &ContractError{
				Code:    CodeSandboxSessionUnsupported,
				Field:   "session_modes_supported",
				Message: "requested session mode is unsupported by profile-pack",
			}
		}
	}
	return nil
}

func normalizeActivationContext(in ActivationContext) ActivationContext {
	out := in
	out.HostOS = strings.ToLower(strings.TrimSpace(out.HostOS))
	out.HostArch = strings.ToLower(strings.TrimSpace(out.HostArch))
	if out.HostOS == "" {
		out.HostOS = strings.ToLower(strings.TrimSpace(runtime.GOOS))
	}
	if out.HostArch == "" {
		out.HostArch = strings.ToLower(strings.TrimSpace(runtime.GOARCH))
	}
	out.RequestedSession = strings.ToLower(strings.TrimSpace(out.RequestedSession))
	out.SupportedBackends = normalizeCapabilities(out.SupportedBackends)
	return out
}

func toStringSet(items []string) map[string]struct{} {
	out := map[string]struct{}{}
	for _, raw := range items {
		item := strings.ToLower(strings.TrimSpace(raw))
		if item == "" {
			continue
		}
		out[item] = struct{}{}
	}
	return out
}

func mapProfileError(err error) error {
	if err == nil {
		return nil
	}
	pe := &adapterprofile.Error{}
	if !errors.As(err, &pe) {
		return err
	}
	switch pe.Code {
	case adapterprofile.CodeUnknownProfileVersion:
		return &ContractError{
			Code:    CodeUnknownContractProfile,
			Field:   "contract_profile_version",
			Message: pe.Message,
		}
	case adapterprofile.CodeProfileOutOfWindow:
		return &ContractError{
			Code:    CodeContractProfileOutOfWindow,
			Field:   "contract_profile_version",
			Message: pe.Message,
		}
	default:
		return &ContractError{
			Code:    CodeUnknownContractProfile,
			Field:   "contract_profile_version",
			Message: pe.Message,
		}
	}
}
