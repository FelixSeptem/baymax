// Package catalog provides a bounded, host-injected provider/model admission
// catalog. It intentionally contains metadata and redacted credential evidence
// only; provider protocol details remain in model/<provider> packages.
package catalog

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/FelixSeptem/baymax/adapter/capability"
)

const (
	StatusReady    = "ready"
	StatusDegraded = "degraded"
	StatusBlocked  = "blocked"

	CredentialAvailable  = "available"
	CredentialMissing    = "missing"
	CredentialInvalid    = "invalid"
	CredentialUnverified = "unverified"

	ReasonUnknownModel         = "provider.catalog.unknown_model"
	ReasonDuplicateDescriptor  = "provider.catalog.duplicate_descriptor"
	ReasonInvalidDescriptor    = "provider.catalog.invalid_descriptor"
	ReasonInvalidCapability    = "provider.catalog.invalid_capability"
	ReasonInvalidFallback      = "provider.catalog.invalid_fallback"
	ReasonCredentialMissing    = "provider.credential.missing"
	ReasonCredentialInvalid    = "provider.credential.invalid"
	ReasonCredentialUnverified = "provider.credential.unverified"
	ReasonOptionalFallback     = "provider.catalog.optional_fallback"
)

var capabilityPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9._:/-]{0,63}$`)

type Identity struct {
	Provider string `json:"provider"`
	Model    string `json:"model"`
}

type Descriptor struct {
	Provider      string    `json:"provider"`
	Model         string    `json:"model"`
	ContextWindow int       `json:"context_window"`
	Capabilities  []string  `json:"capabilities,omitempty"`
	Fallback      *Identity `json:"fallback,omitempty"`
}

type Input struct {
	Version     string       `json:"version"`
	Descriptors []Descriptor `json:"descriptors"`
}

type Catalog struct {
	version     string
	descriptors map[string]Descriptor
}

type CredentialEvidence struct {
	Provider string `json:"provider"`
	Status   string `json:"status"`
	Reason   string `json:"reason,omitempty"`
}

type Request struct {
	Identity Identity
	Required []string
	Optional []string
	Strategy string
}

type Admission struct {
	Status       string
	Catalog      string
	Selected     Identity
	Fallback     *Identity
	Capabilities capability.Outcome
	Credential   CredentialEvidence
	Reasons      []string
}

func New(input Input) (Catalog, error) {
	version := strings.TrimSpace(input.Version)
	if version == "" {
		return Catalog{}, reasonError(ReasonInvalidDescriptor, "catalog version is required")
	}
	out := Catalog{version: version, descriptors: make(map[string]Descriptor, len(input.Descriptors))}
	for _, raw := range input.Descriptors {
		descriptor, err := normalizeDescriptor(raw)
		if err != nil {
			return Catalog{}, err
		}
		key := identityKey(Identity{Provider: descriptor.Provider, Model: descriptor.Model})
		if _, exists := out.descriptors[key]; exists {
			return Catalog{}, reasonError(ReasonDuplicateDescriptor, key)
		}
		out.descriptors[key] = descriptor
	}
	for key, descriptor := range out.descriptors {
		if descriptor.Fallback == nil {
			continue
		}
		fallbackKey := identityKey(*descriptor.Fallback)
		if fallbackKey == key {
			return Catalog{}, reasonError(ReasonInvalidFallback, key)
		}
		if _, exists := out.descriptors[fallbackKey]; !exists {
			return Catalog{}, reasonError(ReasonInvalidFallback, fallbackKey)
		}
		seen := map[string]struct{}{key: {}}
		current := fallbackKey
		for {
			if _, exists := seen[current]; exists {
				return Catalog{}, reasonError(ReasonInvalidFallback, current)
			}
			seen[current] = struct{}{}
			next, exists := out.descriptors[current]
			if !exists || next.Fallback == nil {
				break
			}
			current = identityKey(*next.Fallback)
		}
	}
	return out, nil
}

func (c Catalog) Version() string { return c.version }

func (c Catalog) Lookup(identity Identity) (Descriptor, bool) {
	descriptor, ok := c.descriptors[identityKey(identity)]
	if !ok {
		return Descriptor{}, false
	}
	return cloneDescriptor(descriptor), true
}

func Evaluate(c Catalog, request Request, credentials map[string]CredentialEvidence, strict bool) (Admission, error) {
	identity := normalizeIdentity(request.Identity)
	descriptor, ok := c.Lookup(identity)
	if !ok {
		return Admission{Status: StatusBlocked, Catalog: c.version, Selected: identity, Reasons: []string{ReasonUnknownModel}}, nil
	}

	negotiated, err := capability.Negotiate(strategyOrDefault(request.Strategy), capability.Set{Required: descriptor.Capabilities}, capability.Request{Required: request.Required, Optional: request.Optional})
	if err != nil {
		return Admission{}, err
	}
	evidence := normalizeEvidence(credentials[identity.Provider], identity.Provider)
	admission := Admission{
		Status:       StatusReady,
		Catalog:      c.version,
		Selected:     identity,
		Capabilities: negotiated,
		Credential:   evidence,
	}
	if !negotiated.Accepted {
		admission.Status = StatusBlocked
		admission.Reasons = append(admission.Reasons, negotiated.Reasons...)
		return admission, nil
	}
	if evidence.Status == CredentialMissing {
		admission.Status = StatusBlocked
		admission.Reasons = append(admission.Reasons, ReasonCredentialMissing)
		return admission, nil
	}
	if evidence.Status == CredentialInvalid {
		admission.Status = StatusBlocked
		admission.Reasons = append(admission.Reasons, ReasonCredentialInvalid)
		return admission, nil
	}
	if evidence.Status == CredentialUnverified {
		admission.Reasons = append(admission.Reasons, ReasonCredentialUnverified)
		if strict {
			admission.Status = StatusBlocked
			return admission, nil
		}
		admission.Status = StatusDegraded
	}
	if negotiated.Downgraded && descriptor.Fallback != nil {
		fallback, fallbackAdmission, fallbackErr := evaluateFallback(c, *descriptor.Fallback, request, credentials, strict)
		if fallbackErr != nil {
			return Admission{}, fallbackErr
		}
		if fallbackAdmission.Status != StatusBlocked {
			admission.Fallback = &fallback
			admission.Status = StatusDegraded
			admission.Reasons = append(admission.Reasons, negotiated.Reasons...)
			admission.Reasons = append(admission.Reasons, ReasonOptionalFallback)
		}
	}
	if len(admission.Reasons) == 0 && negotiated.Downgraded {
		admission.Status = StatusDegraded
		admission.Reasons = append(admission.Reasons, negotiated.Reasons...)
	}
	return admission, nil
}

func evaluateFallback(c Catalog, fallback Identity, request Request, credentials map[string]CredentialEvidence, strict bool) (Identity, Admission, error) {
	fallbackRequest := request
	fallbackRequest.Identity = fallback
	fallbackAdmission, err := Evaluate(c, fallbackRequest, credentials, strict)
	return fallback, fallbackAdmission, err
}

func normalizeDescriptor(raw Descriptor) (Descriptor, error) {
	descriptor := Descriptor{Provider: strings.TrimSpace(strings.ToLower(raw.Provider)), Model: strings.TrimSpace(strings.ToLower(raw.Model)), ContextWindow: raw.ContextWindow, Capabilities: normalizeCapabilities(raw.Capabilities)}
	if descriptor.Provider == "" || descriptor.Model == "" || descriptor.ContextWindow <= 0 {
		return Descriptor{}, reasonError(ReasonInvalidDescriptor, "provider, model, and positive context window are required")
	}
	for _, item := range descriptor.Capabilities {
		if !capabilityPattern.MatchString(item) {
			return Descriptor{}, reasonError(ReasonInvalidCapability, item)
		}
	}
	if raw.Fallback != nil {
		fallback := normalizeIdentity(*raw.Fallback)
		if fallback.Provider == "" || fallback.Model == "" {
			return Descriptor{}, reasonError(ReasonInvalidFallback, "fallback identity is required")
		}
		descriptor.Fallback = &fallback
	}
	return descriptor, nil
}

func normalizeEvidence(evidence CredentialEvidence, provider string) CredentialEvidence {
	evidence.Provider = normalizeIdentity(Identity{Provider: provider}).Provider
	evidence.Status = strings.ToLower(strings.TrimSpace(evidence.Status))
	evidence.Reason = boundedReason(evidence.Reason)
	if evidence.Status == "" {
		evidence.Status = CredentialMissing
		evidence.Reason = ReasonCredentialMissing
	} else if evidence.Status != CredentialAvailable && evidence.Status != CredentialMissing && evidence.Status != CredentialInvalid && evidence.Status != CredentialUnverified {
		evidence.Status = CredentialInvalid
		if evidence.Reason == "" {
			evidence.Reason = ReasonCredentialInvalid
		}
	}
	return evidence
}

func normalizeIdentity(identity Identity) Identity {
	return Identity{Provider: strings.ToLower(strings.TrimSpace(identity.Provider)), Model: strings.ToLower(strings.TrimSpace(identity.Model))}
}

func normalizeCapabilities(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		normalized := strings.ToLower(strings.TrimSpace(value))
		if normalized == "" {
			continue
		}
		if _, exists := seen[normalized]; exists {
			continue
		}
		seen[normalized] = struct{}{}
		result = append(result, normalized)
	}
	sort.Strings(result)
	return result
}

func cloneDescriptor(in Descriptor) Descriptor {
	out := in
	out.Capabilities = append([]string(nil), in.Capabilities...)
	if in.Fallback != nil {
		fallback := *in.Fallback
		out.Fallback = &fallback
	}
	return out
}

func identityKey(identity Identity) string {
	normalized := normalizeIdentity(identity)
	return normalized.Provider + "\x00" + normalized.Model
}

func strategyOrDefault(strategy string) string {
	if strings.TrimSpace(strategy) == "" {
		return capability.StrategyFailFast
	}
	return strategy
}

func boundedReason(reason string) string {
	reason = strings.TrimSpace(reason)
	if len(reason) > 128 {
		return reason[:128]
	}
	return reason
}

type validationError struct {
	code    string
	message string
}

func (e validationError) Error() string {
	return fmt.Sprintf("[%s] catalog validation failed: %s", e.code, e.message)
}

func reasonError(code, message string) error {
	return validationError{code: code, message: message}
}

func HasReason(err error, reason string) bool {
	return err != nil && strings.Contains(err.Error(), "["+reason+"]")
}
