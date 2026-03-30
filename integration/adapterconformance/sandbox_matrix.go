package adapterconformance

import (
	"fmt"
	"sort"
	"strings"

	adaptermanifest "github.com/FelixSeptem/baymax/adapter/manifest"
)

const (
	SandboxDriftBackendProfile    = "sandbox_backend_profile_drift"
	SandboxDriftCapabilityClaim   = "sandbox_capability_claim_drift"
	SandboxDriftSessionLifecycle  = "sandbox_session_lifecycle_drift"
	SandboxDriftReasonTaxonomy    = "sandbox_reason_taxonomy_drift"
	SandboxDriftManifestCompat    = "sandbox_manifest_compat_drift"
	SandboxDriftSessionModeCompat = "sandbox_session_mode_drift"
	SandboxSkipBackendUnavailable = "sandbox_backend_unavailable_for_host"
)

type SandboxBackendSuite struct {
	ProfileID             string
	Backend               string
	HostOS                string
	HostArch              string
	RequiredCapabilities  []string
	OptionalCapabilities  []string
	SessionModesSupported []string
}

type SandboxBackendSuiteResult struct {
	ProfileID string
	Backend   string
	Executed  bool
	SkipClass string
	SkipNote  string
}

type SandboxCapabilityNegotiationResult struct {
	Accepted           bool
	DriftClass         string
	MissingRequired    []string
	DowngradedOptional []string
}

type SandboxSessionLifecycleHarness struct {
	counter        int
	perSession     map[string]string
	closedTerminal map[string]bool
}

func MainstreamSandboxBackendMatrix() []SandboxBackendSuite {
	backends := adaptermanifest.SupportedSandboxBackends()
	out := make([]SandboxBackendSuite, 0, len(backends))
	for _, backend := range backends {
		profile, ok := adaptermanifest.SandboxProfileByID(backend)
		if !ok {
			continue
		}
		out = append(out, SandboxBackendSuite{
			ProfileID: profile.ID,
			Backend:   profile.Backend,
			HostOS:    profile.HostOS,
			HostArch:  profile.HostArch,
			RequiredCapabilities: []string{
				"sandbox.adapter.backend_profile_resolved",
				"sandbox.adapter.reason_taxonomy_stable",
			},
			OptionalCapabilities: []string{
				"sandbox.adapter.lifecycle.crash_reconnect",
			},
			SessionModesSupported: append([]string(nil), profile.SessionModesSupported...),
		})
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].Backend < out[j].Backend
	})
	return out
}

func EvaluateMainstreamSandboxBackendMatrix(hostOS string) []SandboxBackendSuiteResult {
	normalizedHostOS := strings.ToLower(strings.TrimSpace(hostOS))
	matrix := MainstreamSandboxBackendMatrix()
	out := make([]SandboxBackendSuiteResult, 0, len(matrix))
	for i := range matrix {
		row := matrix[i]
		result := SandboxBackendSuiteResult{
			ProfileID: row.ProfileID,
			Backend:   row.Backend,
		}
		if normalizedHostOS == row.HostOS {
			result.Executed = true
		} else {
			result.Executed = false
			result.SkipClass = SandboxSkipBackendUnavailable
			result.SkipNote = fmt.Sprintf("host_os=%s requires backend host_os=%s", normalizedHostOS, row.HostOS)
		}
		out = append(out, result)
	}
	return out
}

func EvaluateSandboxCapabilityNegotiation(required, optional, supported []string) SandboxCapabilityNegotiationResult {
	requiredNorm := normalizeSandboxCapabilityList(required)
	optionalNorm := normalizeSandboxCapabilityList(optional)
	supportedSet := map[string]struct{}{}
	for _, item := range normalizeSandboxCapabilityList(supported) {
		supportedSet[item] = struct{}{}
	}
	missingRequired := make([]string, 0)
	for _, capability := range requiredNorm {
		if _, ok := supportedSet[capability]; ok {
			continue
		}
		missingRequired = append(missingRequired, capability)
	}
	if len(missingRequired) > 0 {
		return SandboxCapabilityNegotiationResult{
			Accepted:        false,
			DriftClass:      SandboxDriftCapabilityClaim,
			MissingRequired: missingRequired,
		}
	}
	downgradedOptional := make([]string, 0)
	for _, capability := range optionalNorm {
		if _, ok := supportedSet[capability]; ok {
			continue
		}
		downgradedOptional = append(downgradedOptional, capability)
	}
	return SandboxCapabilityNegotiationResult{
		Accepted:           true,
		DriftClass:         "",
		MissingRequired:    nil,
		DowngradedOptional: downgradedOptional,
	}
}

func NewSandboxSessionLifecycleHarness() *SandboxSessionLifecycleHarness {
	return &SandboxSessionLifecycleHarness{
		perSession:     map[string]string{},
		closedTerminal: map[string]bool{},
	}
}

func (h *SandboxSessionLifecycleHarness) Open(mode, key string) string {
	if h == nil {
		return ""
	}
	normalizedMode := strings.ToLower(strings.TrimSpace(mode))
	normalizedKey := strings.TrimSpace(key)
	if normalizedMode == adaptermanifest.SandboxSessionModePerSession {
		if token := strings.TrimSpace(h.perSession[normalizedKey]); token != "" {
			return token
		}
	}
	h.counter++
	token := fmt.Sprintf("sandbox-session-%03d", h.counter)
	if normalizedMode == adaptermanifest.SandboxSessionModePerSession {
		h.perSession[normalizedKey] = token
	}
	return token
}

func (h *SandboxSessionLifecycleHarness) Crash(normalizedKey string) {
	if h == nil {
		return
	}
	delete(h.perSession, strings.TrimSpace(normalizedKey))
}

func (h *SandboxSessionLifecycleHarness) Close(normalizedKey string) bool {
	if h == nil {
		return false
	}
	key := strings.TrimSpace(normalizedKey)
	if h.closedTerminal[key] {
		return false
	}
	h.closedTerminal[key] = true
	delete(h.perSession, key)
	return true
}

func CanonicalSandboxDriftClasses() []string {
	return []string{
		SandboxDriftBackendProfile,
		SandboxDriftCapabilityClaim,
		SandboxDriftManifestCompat,
		SandboxDriftSessionLifecycle,
		SandboxDriftSessionModeCompat,
		SandboxDriftReasonTaxonomy,
	}
}

func normalizeSandboxCapabilityList(items []string) []string {
	if len(items) == 0 {
		return nil
	}
	out := make([]string, 0, len(items))
	seen := map[string]struct{}{}
	for _, raw := range items {
		item := strings.ToLower(strings.TrimSpace(raw))
		if item == "" {
			continue
		}
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		out = append(out, item)
	}
	sort.Strings(out)
	return out
}
