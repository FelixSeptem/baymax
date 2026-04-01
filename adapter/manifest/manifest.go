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
	memorypkg "github.com/FelixSeptem/baymax/memory"
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
	CodeMemoryModeMismatch         = "adapter-manifest.memory-mode-mismatch"
	CodeMemoryProfileMismatch      = "adapter-manifest.memory-profile-mismatch"
	CodeMemoryContractMismatch     = "adapter-manifest.memory-contract-mismatch"
	CodeMemoryRequiredOpMissing    = "adapter-manifest.memory-required-operation-missing"
	CodeMemoryContextMissing       = "adapter-manifest.memory-context-missing"
	CodeAllowlistMissingEntry      = "adapter-manifest.allowlist-missing-entry"
	CodeAllowlistSignatureInvalid  = "adapter-manifest.allowlist-signature-invalid"
	CodeAllowlistPolicyConflict    = "adapter-manifest.allowlist-policy-conflict"
)

var (
	namePattern               = regexp.MustCompile(`^[a-z][a-z0-9-]*$`)
	conformanceProfilePattern = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)
	memoryIdentityPattern     = regexp.MustCompile(`^[a-z][a-z0-9_-]*$`)
	memoryContractPattern     = regexp.MustCompile(`^memory\.v[0-9]+$`)
	memoryOperationPattern    = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)
	allowlistIdentityPattern  = regexp.MustCompile(`^[a-z][a-z0-9._-]*$`)
)

const (
	allowlistSignatureStatusValid   = "valid"
	allowlistSignatureStatusInvalid = "invalid"
	allowlistSignatureStatusUnknown = "unknown"

	allowlistEnforcementModeObserve = "observe"
	allowlistEnforcementModeEnforce = "enforce"

	allowlistUnknownSignatureDeny           = "deny"
	allowlistUnknownSignatureAllowAndRecord = "allow_and_record"
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
	Memory                 *Memory      `json:"memory,omitempty"`
	Allowlist              *AllowlistID `json:"allowlist,omitempty"`
}

type AllowlistID struct {
	AdapterID       string `json:"adapter_id"`
	Publisher       string `json:"publisher"`
	Version         string `json:"version"`
	SignatureStatus string `json:"signature_status"`
}

type Memory struct {
	Provider        string           `json:"provider,omitempty"`
	Profile         string           `json:"profile,omitempty"`
	ContractVersion string           `json:"contract_version,omitempty"`
	Operations      MemoryOperations `json:"operations,omitempty"`
	Fallback        MemoryFallback   `json:"fallback,omitempty"`
}

type MemoryOperations struct {
	Required []string `json:"required"`
	Optional []string `json:"optional"`
}

type MemoryFallback struct {
	Supported *bool `json:"supported,omitempty"`
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
	MemoryMode        string
	MemoryProvider    string
	MemoryProfile     string
	MemoryContract    string
	MemoryOperations  []string
	Allowlist         AllowlistPolicy
}

type AllowlistPolicy struct {
	Enabled            bool
	EnforcementMode    string
	OnUnknownSignature string
	Entries            []AllowlistID
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
	if err := validateSandboxMetadata(normalized); err != nil {
		return err
	}
	if err := validateAllowlistManifest(normalized); err != nil {
		return err
	}
	return validateMemoryManifest(normalized)
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
	if err := validateAllowlistActivationCompatibility(normalized, activationCtx); err != nil {
		return ActivationResult{}, err
	}
	memoryOptionalDowngrades, err := validateMemoryActivationCompatibility(normalized, activationCtx)
	if err != nil {
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
	downgrades = append(downgrades, memoryOptionalDowngrades...)
	sort.Slice(downgrades, func(i, j int) bool {
		if downgrades[i].Capability == downgrades[j].Capability {
			return downgrades[i].ReasonCode < downgrades[j].ReasonCode
		}
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
	manifest.Memory = normalizeMemory(manifest.Memory)
	manifest.Allowlist = normalizeAllowlistID(manifest.Allowlist)
	return manifest
}

func normalizeAllowlistID(in *AllowlistID) *AllowlistID {
	if in == nil {
		return nil
	}
	out := *in
	out.AdapterID = strings.ToLower(strings.TrimSpace(out.AdapterID))
	out.Publisher = strings.ToLower(strings.TrimSpace(out.Publisher))
	out.Version = strings.TrimSpace(out.Version)
	out.SignatureStatus = strings.ToLower(strings.TrimSpace(out.SignatureStatus))
	return &out
}

func normalizeMemory(in *Memory) *Memory {
	if in == nil {
		return nil
	}
	out := *in
	out.Provider = strings.ToLower(strings.TrimSpace(out.Provider))
	out.Profile = strings.ToLower(strings.TrimSpace(out.Profile))
	out.ContractVersion = strings.ToLower(strings.TrimSpace(out.ContractVersion))
	out.Operations.Required = normalizeMemoryOperations(out.Operations.Required)
	out.Operations.Optional = normalizeMemoryOperations(out.Operations.Optional)
	if in.Fallback.Supported != nil {
		v := *in.Fallback.Supported
		out.Fallback.Supported = &v
	}
	return &out
}

func normalizeMemoryOperations(raw []string) []string {
	if raw == nil {
		return nil
	}
	out := make([]string, 0, len(raw))
	seen := map[string]struct{}{}
	for _, item := range raw {
		normalized := strings.ToLower(strings.TrimSpace(item))
		if normalized == "" {
			out = append(out, "")
			continue
		}
		if _, ok := seen[normalized]; ok {
			continue
		}
		seen[normalized] = struct{}{}
		out = append(out, normalized)
	}
	sort.Strings(out)
	return out
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

func validateMemoryManifest(manifest Manifest) error {
	if manifest.Memory == nil {
		return nil
	}
	mem := manifest.Memory
	requiredChecks := []struct {
		field string
		value string
	}{
		{field: "memory.provider", value: mem.Provider},
		{field: "memory.profile", value: mem.Profile},
		{field: "memory.contract_version", value: mem.ContractVersion},
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
	if mem.Operations.Required == nil {
		return &ContractError{
			Code:    CodeMissingField,
			Field:   "memory.operations.required",
			Message: "required field is missing",
		}
	}
	if mem.Operations.Optional == nil {
		return &ContractError{
			Code:    CodeMissingField,
			Field:   "memory.operations.optional",
			Message: "required field is missing",
		}
	}
	if mem.Fallback.Supported == nil {
		return &ContractError{
			Code:    CodeMissingField,
			Field:   "memory.fallback.supported",
			Message: "required field is missing",
		}
	}
	if len(mem.Operations.Required) == 0 {
		return &ContractError{
			Code:    CodeMissingField,
			Field:   "memory.operations.required",
			Message: "required field is missing",
		}
	}
	if !memoryIdentityPattern.MatchString(mem.Provider) {
		return &ContractError{
			Code:    CodeInvalidField,
			Field:   "memory.provider",
			Message: "memory.provider must match ^[a-z][a-z0-9_-]*$",
		}
	}
	if !memoryIdentityPattern.MatchString(mem.Profile) {
		return &ContractError{
			Code:    CodeInvalidField,
			Field:   "memory.profile",
			Message: "memory.profile must match ^[a-z][a-z0-9_-]*$",
		}
	}
	if !memoryContractPattern.MatchString(mem.ContractVersion) {
		return &ContractError{
			Code:    CodeInvalidField,
			Field:   "memory.contract_version",
			Message: "memory.contract_version must match ^memory\\.v[0-9]+$",
		}
	}
	if err := validateMemoryOperationField("memory.operations.required", mem.Operations.Required); err != nil {
		return err
	}
	if err := validateMemoryOperationField("memory.operations.optional", mem.Operations.Optional); err != nil {
		return err
	}
	requiredSet := toStringSet(mem.Operations.Required)
	for _, op := range []string{memorypkg.OperationQuery, memorypkg.OperationUpsert, memorypkg.OperationDelete} {
		if _, ok := requiredSet[op]; ok {
			continue
		}
		return &ContractError{
			Code:    CodeInvalidField,
			Field:   "memory.operations.required",
			Message: "memory.operations.required must include query|upsert|delete",
		}
	}
	for _, optional := range mem.Operations.Optional {
		if _, ok := requiredSet[optional]; ok {
			return &ContractError{
				Code:    CodeInvalidField,
				Field:   "memory.operations.optional",
				Message: "memory.operations.optional must not overlap required operations",
			}
		}
	}
	return nil
}

func validateMemoryOperationField(field string, ops []string) error {
	if hasEmptyCapability(ops) {
		return &ContractError{
			Code:    CodeInvalidField,
			Field:   field,
			Message: "memory operation identifiers must not be empty",
		}
	}
	for _, op := range ops {
		if memoryOperationPattern.MatchString(op) {
			continue
		}
		return &ContractError{
			Code:    CodeInvalidField,
			Field:   field,
			Message: "memory operation identifier is invalid",
		}
	}
	return nil
}

func validateMemoryActivationCompatibility(manifest Manifest, in ActivationContext) ([]OptionalDowngrade, error) {
	if manifest.Memory == nil {
		return nil, nil
	}
	ctx := normalizeActivationContext(in)
	if strings.TrimSpace(ctx.MemoryMode) == "" {
		return nil, &ContractError{
			Code:    CodeMemoryContextMissing,
			Field:   "memory_mode",
			Message: "runtime memory mode is required for memory manifest activation",
		}
	}
	if strings.TrimSpace(ctx.MemoryProvider) == "" {
		return nil, &ContractError{
			Code:    CodeMemoryContextMissing,
			Field:   "memory_provider",
			Message: "runtime memory provider is required for memory manifest activation",
		}
	}
	if strings.TrimSpace(ctx.MemoryProfile) == "" {
		return nil, &ContractError{
			Code:    CodeMemoryContextMissing,
			Field:   "memory_profile",
			Message: "runtime memory profile is required for memory manifest activation",
		}
	}
	if strings.TrimSpace(ctx.MemoryContract) == "" {
		return nil, &ContractError{
			Code:    CodeMemoryContextMissing,
			Field:   "memory_contract_version",
			Message: "runtime memory contract version is required for memory manifest activation",
		}
	}

	expectedMode := memorypkg.ModeExternalSPI
	if manifest.Memory.Provider == memorypkg.ModeBuiltinFilesystem {
		expectedMode = memorypkg.ModeBuiltinFilesystem
	}
	if ctx.MemoryMode != expectedMode {
		return nil, &ContractError{
			Code:    CodeMemoryModeMismatch,
			Field:   "memory.mode",
			Message: fmt.Sprintf("runtime memory mode %q mismatches manifest expectation %q", ctx.MemoryMode, expectedMode),
		}
	}
	if ctx.MemoryProvider != manifest.Memory.Provider {
		return nil, &ContractError{
			Code:    CodeMemoryProfileMismatch,
			Field:   "memory.provider",
			Message: fmt.Sprintf("runtime memory provider %q mismatches manifest provider %q", ctx.MemoryProvider, manifest.Memory.Provider),
		}
	}
	if ctx.MemoryProfile != manifest.Memory.Profile {
		return nil, &ContractError{
			Code:    CodeMemoryProfileMismatch,
			Field:   "memory.profile",
			Message: fmt.Sprintf("runtime memory profile %q mismatches manifest profile %q", ctx.MemoryProfile, manifest.Memory.Profile),
		}
	}
	if !memoryContractCompatible(manifest.Memory.ContractVersion, ctx.MemoryContract) {
		return nil, &ContractError{
			Code:    CodeMemoryContractMismatch,
			Field:   "memory.contract_version",
			Message: fmt.Sprintf("runtime memory contract_version %q mismatches manifest contract_version %q", ctx.MemoryContract, manifest.Memory.ContractVersion),
		}
	}

	supported := toStringSet(ctx.MemoryOperations)
	missingRequired := make([]string, 0)
	for _, op := range manifest.Memory.Operations.Required {
		if _, ok := supported[op]; ok {
			continue
		}
		missingRequired = append(missingRequired, op)
	}
	if len(missingRequired) > 0 {
		sort.Strings(missingRequired)
		return nil, &ContractError{
			Code:    CodeMemoryRequiredOpMissing,
			Field:   "memory.operations.required",
			Message: "required memory operation unavailable: " + strings.Join(missingRequired, ","),
		}
	}

	downgrades := make([]OptionalDowngrade, 0)
	for _, op := range manifest.Memory.Operations.Optional {
		if _, ok := supported[op]; ok {
			continue
		}
		downgrades = append(downgrades, OptionalDowngrade{
			Capability: "memory." + op,
			ReasonCode: "adapter.manifest.memory.operation.optional_missing." + reasonSegment(op),
		})
	}
	sort.Slice(downgrades, func(i, j int) bool {
		return downgrades[i].Capability < downgrades[j].Capability
	})
	return downgrades, nil
}

func memoryContractCompatible(manifestContract, runtimeContract string) bool {
	manifestVersion := strings.TrimSpace(manifestContract)
	runtimeVersion := strings.TrimSpace(runtimeContract)
	if manifestVersion == "" || runtimeVersion == "" {
		return false
	}
	return manifestVersion == runtimeVersion
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

func validateAllowlistManifest(manifest Manifest) error {
	if manifest.Allowlist == nil {
		return nil
	}
	identity := manifest.Allowlist
	checks := []struct {
		field string
		value string
	}{
		{field: "allowlist.adapter_id", value: identity.AdapterID},
		{field: "allowlist.publisher", value: identity.Publisher},
		{field: "allowlist.version", value: identity.Version},
		{field: "allowlist.signature_status", value: identity.SignatureStatus},
	}
	for _, check := range checks {
		if strings.TrimSpace(check.value) == "" {
			return &ContractError{
				Code:    CodeMissingField,
				Field:   check.field,
				Message: "required field is missing",
			}
		}
	}
	if !allowlistIdentityPattern.MatchString(identity.AdapterID) {
		return &ContractError{
			Code:    CodeInvalidField,
			Field:   "allowlist.adapter_id",
			Message: "allowlist.adapter_id must match ^[a-z][a-z0-9._-]*$",
		}
	}
	if !allowlistIdentityPattern.MatchString(identity.Publisher) {
		return &ContractError{
			Code:    CodeInvalidField,
			Field:   "allowlist.publisher",
			Message: "allowlist.publisher must match ^[a-z][a-z0-9._-]*$",
		}
	}
	if !isSupportedAllowlistSignatureStatus(identity.SignatureStatus) {
		return &ContractError{
			Code:    CodeInvalidField,
			Field:   "allowlist.signature_status",
			Message: "allowlist.signature_status must be one of: valid|invalid|unknown",
		}
	}
	return nil
}

func validateAllowlistActivationCompatibility(manifest Manifest, in ActivationContext) error {
	ctx := normalizeActivationContext(in)
	if !ctx.Allowlist.Enabled {
		return nil
	}
	if manifest.Allowlist == nil {
		return &ContractError{
			Code:    CodeAllowlistMissingEntry,
			Field:   "allowlist.adapter_id",
			Message: "manifest allowlist identity metadata is missing",
		}
	}
	mode := strings.TrimSpace(ctx.Allowlist.EnforcementMode)
	if mode == "" {
		mode = allowlistEnforcementModeEnforce
	}
	switch mode {
	case allowlistEnforcementModeObserve, allowlistEnforcementModeEnforce:
	default:
		return &ContractError{
			Code:    CodeAllowlistPolicyConflict,
			Field:   "allowlist.enforcement_mode",
			Message: "allowlist enforcement_mode must be one of: observe|enforce",
		}
	}
	unknownSignatureAction := strings.TrimSpace(ctx.Allowlist.OnUnknownSignature)
	if unknownSignatureAction == "" {
		unknownSignatureAction = allowlistUnknownSignatureDeny
	}
	switch unknownSignatureAction {
	case allowlistUnknownSignatureDeny, allowlistUnknownSignatureAllowAndRecord:
	default:
		return &ContractError{
			Code:    CodeAllowlistPolicyConflict,
			Field:   "allowlist.on_unknown_signature",
			Message: "allowlist on_unknown_signature must be one of: deny|allow_and_record",
		}
	}

	identity := *manifest.Allowlist
	if mode == allowlistEnforcementModeEnforce {
		if identity.SignatureStatus == allowlistSignatureStatusInvalid {
			return &ContractError{
				Code:    CodeAllowlistSignatureInvalid,
				Field:   "allowlist.signature_status",
				Message: "manifest signature_status is invalid under enforce mode",
			}
		}
		if identity.SignatureStatus == allowlistSignatureStatusUnknown && unknownSignatureAction == allowlistUnknownSignatureDeny {
			return &ContractError{
				Code:    CodeAllowlistSignatureInvalid,
				Field:   "allowlist.signature_status",
				Message: "manifest signature_status is unknown and on_unknown_signature=deny",
			}
		}
	}

	match, ok := findAllowlistEntry(identity, ctx.Allowlist.Entries)
	if !ok {
		return &ContractError{
			Code:    CodeAllowlistMissingEntry,
			Field:   "allowlist.entries",
			Message: "manifest allowlist identity is not in effective allowlist entries",
		}
	}
	if mode == allowlistEnforcementModeEnforce {
		if match.SignatureStatus == allowlistSignatureStatusInvalid {
			return &ContractError{
				Code:    CodeAllowlistSignatureInvalid,
				Field:   "allowlist.signature_status",
				Message: "allowlist entry signature_status is invalid under enforce mode",
			}
		}
		if match.SignatureStatus == allowlistSignatureStatusUnknown && unknownSignatureAction == allowlistUnknownSignatureDeny {
			return &ContractError{
				Code:    CodeAllowlistSignatureInvalid,
				Field:   "allowlist.signature_status",
				Message: "allowlist entry signature_status is unknown and on_unknown_signature=deny",
			}
		}
	}
	return nil
}

func isSupportedAllowlistSignatureStatus(status string) bool {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case allowlistSignatureStatusValid, allowlistSignatureStatusInvalid, allowlistSignatureStatusUnknown:
		return true
	default:
		return false
	}
}

func findAllowlistEntry(identity AllowlistID, entries []AllowlistID) (AllowlistID, bool) {
	for i := range entries {
		entry := entries[i]
		if identity.AdapterID == entry.AdapterID &&
			identity.Publisher == entry.Publisher &&
			identity.Version == entry.Version {
			return entry, true
		}
	}
	return AllowlistID{}, false
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
	out.MemoryMode = strings.ToLower(strings.TrimSpace(out.MemoryMode))
	out.MemoryProvider = strings.ToLower(strings.TrimSpace(out.MemoryProvider))
	out.MemoryProfile = strings.ToLower(strings.TrimSpace(out.MemoryProfile))
	out.MemoryContract = strings.ToLower(strings.TrimSpace(out.MemoryContract))
	out.MemoryOperations = normalizeMemoryOperations(out.MemoryOperations)
	out.Allowlist = normalizeAllowlistPolicy(out.Allowlist)
	return out
}

func normalizeAllowlistPolicy(in AllowlistPolicy) AllowlistPolicy {
	out := in
	out.EnforcementMode = strings.ToLower(strings.TrimSpace(out.EnforcementMode))
	out.OnUnknownSignature = strings.ToLower(strings.TrimSpace(out.OnUnknownSignature))
	if len(out.Entries) == 0 {
		out.Entries = []AllowlistID{}
		return out
	}
	normalizedEntries := make([]AllowlistID, 0, len(out.Entries))
	for i := range out.Entries {
		entry := out.Entries[i]
		entry.AdapterID = strings.ToLower(strings.TrimSpace(entry.AdapterID))
		entry.Publisher = strings.ToLower(strings.TrimSpace(entry.Publisher))
		entry.Version = strings.TrimSpace(entry.Version)
		entry.SignatureStatus = strings.ToLower(strings.TrimSpace(entry.SignatureStatus))
		normalizedEntries = append(normalizedEntries, entry)
	}
	out.Entries = normalizedEntries
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
