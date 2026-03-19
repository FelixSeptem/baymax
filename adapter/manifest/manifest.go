package manifest

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"
)

const (
	CodeMissingFile               = "adapter-manifest.missing-file"
	CodeInvalidJSON               = "adapter-manifest.invalid-json"
	CodeMissingField              = "adapter-manifest.missing-field"
	CodeInvalidField              = "adapter-manifest.invalid-field"
	CodeInvalidCompatExpression   = "adapter-manifest.invalid-compat-expression"
	CodeCompatibilityMismatch     = "adapter-manifest.compatibility-mismatch"
	CodeRequiredCapabilityMissing = "adapter-manifest.required-capability-missing"
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
	Type               string       `json:"type"`
	Name               string       `json:"name"`
	Version            string       `json:"version"`
	BaymaxCompat       string       `json:"baymax_compat"`
	Capabilities       Capabilities `json:"capabilities"`
	ConformanceProfile string       `json:"conformance_profile"`
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

type ActivationResult struct {
	OptionalDowngrades []OptionalDowngrade
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
	return nil
}

func Activate(manifest Manifest, runtimeVersion string, availableCapabilities []string) (ActivationResult, error) {
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

	available := make(map[string]struct{}, len(availableCapabilities))
	for _, capability := range availableCapabilities {
		key := normalizeCapability(capability)
		if key == "" {
			continue
		}
		available[key] = struct{}{}
	}

	for _, required := range normalized.Capabilities.Required {
		key := normalizeCapability(required)
		if _, ok := available[key]; ok {
			continue
		}
		return ActivationResult{}, &ContractError{
			Code:    CodeRequiredCapabilityMissing,
			Field:   "capabilities.required",
			Message: "required capability unavailable: " + key,
		}
	}

	downgrades := make([]OptionalDowngrade, 0)
	for _, optional := range normalized.Capabilities.Optional {
		key := normalizeCapability(optional)
		if _, ok := available[key]; ok {
			continue
		}
		downgrades = append(downgrades, OptionalDowngrade{
			Capability: key,
			ReasonCode: "adapter.manifest.capability.optional_missing." + reasonSegment(key),
		})
	}
	sort.Slice(downgrades, func(i, j int) bool {
		return downgrades[i].Capability < downgrades[j].Capability
	})
	return ActivationResult{OptionalDowngrades: downgrades}, nil
}

func normalize(manifest Manifest) Manifest {
	manifest.Type = strings.ToLower(strings.TrimSpace(manifest.Type))
	manifest.Name = strings.ToLower(strings.TrimSpace(manifest.Name))
	manifest.Version = strings.TrimSpace(manifest.Version)
	manifest.BaymaxCompat = strings.TrimSpace(manifest.BaymaxCompat)
	manifest.ConformanceProfile = strings.ToLower(strings.TrimSpace(manifest.ConformanceProfile))
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
