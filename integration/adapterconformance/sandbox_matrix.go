package adapterconformance

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	adaptermanifest "github.com/FelixSeptem/baymax/adapter/manifest"
	adapterprofile "github.com/FelixSeptem/baymax/adapter/profile"
	runtimeconfig "github.com/FelixSeptem/baymax/runtime/config"
)

const (
	SandboxDriftBackendProfile           = "sandbox_backend_profile_drift"
	SandboxDriftCapabilityClaim          = "sandbox_capability_claim_drift"
	SandboxDriftSessionLifecycle         = "sandbox_session_lifecycle_drift"
	SandboxDriftReasonTaxonomy           = "sandbox_reason_taxonomy_drift"
	SandboxDriftManifestCompat           = "sandbox_manifest_compat_drift"
	SandboxDriftSessionModeCompat        = "sandbox_session_mode_drift"
	SandboxDriftEgressPolicyDecision     = "sandbox_egress_policy_decision_drift"
	SandboxDriftEgressSelectorPrecedence = "sandbox_egress_selector_precedence_drift"
	SandboxDriftAllowlistActivation      = "adapter_allowlist_activation_drift"
	SandboxDriftAllowlistTaxonomy        = "adapter_allowlist_taxonomy_drift"
	SandboxSkipBackendUnavailable        = "sandbox_backend_unavailable_for_host"
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

type SandboxEgressPolicyCase struct {
	CaseID               string
	Backend              string
	ProfileID            string
	NamespaceTool        string
	Host                 string
	DefaultAction        string
	OnViolation          string
	ByTool               map[string]string
	Allowlist            []string
	ExpectedAction       string
	ExpectedPolicySource string
}

type SandboxEgressPolicyResult struct {
	Action       string
	PolicySource string
	ReasonCode   string
}

type AdapterAllowlistActivationCase struct {
	CaseID               string
	Backend              string
	ProfileID            string
	Manifest             adaptermanifest.Manifest
	ActivationContext    adaptermanifest.ActivationContext
	ExpectedContractCode string
}

type AdapterAllowlistActivationResult struct {
	Accepted     bool
	ContractCode string
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

func MainstreamSandboxEgressPolicyMatrix() []SandboxEgressPolicyCase {
	backends := MainstreamSandboxBackendMatrix()
	out := make([]SandboxEgressPolicyCase, 0, len(backends)*4)
	for i := range backends {
		row := backends[i]
		out = append(out,
			SandboxEgressPolicyCase{
				CaseID:               "sandbox-egress-deny-matrix",
				Backend:              row.Backend,
				ProfileID:            row.ProfileID,
				NamespaceTool:        "local+shell",
				Host:                 "api.example.com",
				DefaultAction:        runtimeconfig.SecuritySandboxEgressActionDeny,
				OnViolation:          runtimeconfig.SecuritySandboxEgressOnViolationDeny,
				ExpectedAction:       runtimeconfig.SecuritySandboxEgressActionDeny,
				ExpectedPolicySource: "default_action",
			},
			SandboxEgressPolicyCase{
				CaseID:               "sandbox-egress-allow-matrix",
				Backend:              row.Backend,
				ProfileID:            row.ProfileID,
				NamespaceTool:        "local+shell",
				Host:                 "api.example.com",
				DefaultAction:        runtimeconfig.SecuritySandboxEgressActionDeny,
				OnViolation:          runtimeconfig.SecuritySandboxEgressOnViolationDeny,
				Allowlist:            []string{"api.example.com"},
				ExpectedAction:       runtimeconfig.SecuritySandboxEgressActionAllow,
				ExpectedPolicySource: "allowlist",
			},
			SandboxEgressPolicyCase{
				CaseID:               "sandbox-egress-allow-and-record-matrix",
				Backend:              row.Backend,
				ProfileID:            row.ProfileID,
				NamespaceTool:        "local+shell",
				Host:                 "api.example.com",
				DefaultAction:        runtimeconfig.SecuritySandboxEgressActionDeny,
				OnViolation:          runtimeconfig.SecuritySandboxEgressOnViolationAllowAndRecord,
				ExpectedAction:       runtimeconfig.SecuritySandboxEgressActionAllowAndRecord,
				ExpectedPolicySource: "on_violation",
			},
			SandboxEgressPolicyCase{
				CaseID:               "sandbox-egress-selector-override-precedence",
				Backend:              row.Backend,
				ProfileID:            row.ProfileID,
				NamespaceTool:        "local+shell",
				Host:                 "api.example.com",
				DefaultAction:        runtimeconfig.SecuritySandboxEgressActionAllow,
				OnViolation:          runtimeconfig.SecuritySandboxEgressOnViolationDeny,
				ByTool:               map[string]string{"local+shell": runtimeconfig.SecuritySandboxEgressActionDeny},
				Allowlist:            []string{"api.example.com"},
				ExpectedAction:       runtimeconfig.SecuritySandboxEgressActionDeny,
				ExpectedPolicySource: "by_tool",
			},
		)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].CaseID != out[j].CaseID {
			return out[i].CaseID < out[j].CaseID
		}
		if out[i].Backend != out[j].Backend {
			return out[i].Backend < out[j].Backend
		}
		return out[i].ProfileID < out[j].ProfileID
	})
	return out
}

func EvaluateSandboxEgressPolicyCase(in SandboxEgressPolicyCase) (SandboxEgressPolicyResult, error) {
	defaultAction := strings.ToLower(strings.TrimSpace(in.DefaultAction))
	onViolation := strings.ToLower(strings.TrimSpace(in.OnViolation))
	if !isSandboxEgressAction(defaultAction) {
		return SandboxEgressPolicyResult{}, fmt.Errorf("sandbox default_action must be deny|allow|allow_and_record, got %q", in.DefaultAction)
	}
	if !isSandboxEgressOnViolation(onViolation) {
		return SandboxEgressPolicyResult{}, fmt.Errorf("sandbox on_violation must be deny|allow_and_record, got %q", in.OnViolation)
	}
	namespaceTool := strings.ToLower(strings.TrimSpace(in.NamespaceTool))
	host := strings.ToLower(strings.TrimSpace(in.Host))
	byTool := normalizeSandboxEgressPolicyMap(in.ByTool)
	allowlist := normalizeSandboxAllowlistPatterns(in.Allowlist)

	action := defaultAction
	policySource := "default_action"
	if selectorAction, ok := byTool[namespaceTool]; ok {
		action = selectorAction
		policySource = "by_tool"
	} else if host != "" && matchesSandboxAllowlist(host, allowlist) {
		action = runtimeconfig.SecuritySandboxEgressActionAllow
		policySource = "allowlist"
	}
	if policySource == "default_action" &&
		action == runtimeconfig.SecuritySandboxEgressActionDeny &&
		onViolation == runtimeconfig.SecuritySandboxEgressOnViolationAllowAndRecord {
		action = runtimeconfig.SecuritySandboxEgressActionAllowAndRecord
		policySource = "on_violation"
	}
	reasonCode := ""
	switch action {
	case runtimeconfig.SecuritySandboxEgressActionDeny:
		reasonCode = "sandbox.egress_deny"
	case runtimeconfig.SecuritySandboxEgressActionAllowAndRecord:
		reasonCode = "sandbox.egress_allow_and_record"
	}
	return SandboxEgressPolicyResult{
		Action:       action,
		PolicySource: policySource,
		ReasonCode:   reasonCode,
	}, nil
}

func MainstreamAdapterAllowlistActivationMatrix() []AdapterAllowlistActivationCase {
	backends := MainstreamSandboxBackendMatrix()
	out := make([]AdapterAllowlistActivationCase, 0, len(backends)*4)
	for i := range backends {
		row := backends[i]
		identity := adaptermanifest.AllowlistID{
			AdapterID:       "adapter." + strings.ReplaceAll(row.Backend, "_", "."),
			Publisher:       "acme",
			Version:         "1.0.0",
			SignatureStatus: "valid",
		}
		manifest := sandboxAllowlistManifestForMatrix(row, identity)
		baseContext := adaptermanifest.ActivationContext{
			HostOS:            row.HostOS,
			HostArch:          row.HostArch,
			RequestedSession:  firstSandboxSessionMode(row.SessionModesSupported),
			SupportedBackends: []string{row.Backend},
		}
		out = append(out,
			AdapterAllowlistActivationCase{
				CaseID:    "adapter-allowlist-missing-entry-enforce",
				Backend:   row.Backend,
				ProfileID: row.ProfileID,
				Manifest:  manifest,
				ActivationContext: withAllowlistPolicy(baseContext, adaptermanifest.AllowlistPolicy{
					Enabled:            true,
					EnforcementMode:    "enforce",
					OnUnknownSignature: "deny",
					Entries: []adaptermanifest.AllowlistID{
						{
							AdapterID:       "adapter.other",
							Publisher:       "acme",
							Version:         "1.0.0",
							SignatureStatus: "valid",
						},
					},
				}),
				ExpectedContractCode: adaptermanifest.CodeAllowlistMissingEntry,
			},
			AdapterAllowlistActivationCase{
				CaseID:    "adapter-allowlist-signature-invalid-enforce",
				Backend:   row.Backend,
				ProfileID: row.ProfileID,
				Manifest:  manifest,
				ActivationContext: withAllowlistPolicy(baseContext, adaptermanifest.AllowlistPolicy{
					Enabled:            true,
					EnforcementMode:    "enforce",
					OnUnknownSignature: "deny",
					Entries: []adaptermanifest.AllowlistID{
						{
							AdapterID:       identity.AdapterID,
							Publisher:       identity.Publisher,
							Version:         identity.Version,
							SignatureStatus: "invalid",
						},
					},
				}),
				ExpectedContractCode: adaptermanifest.CodeAllowlistSignatureInvalid,
			},
			AdapterAllowlistActivationCase{
				CaseID:    "adapter-allowlist-allowed-path-enforce",
				Backend:   row.Backend,
				ProfileID: row.ProfileID,
				Manifest:  manifest,
				ActivationContext: withAllowlistPolicy(baseContext, adaptermanifest.AllowlistPolicy{
					Enabled:            true,
					EnforcementMode:    "enforce",
					OnUnknownSignature: "deny",
					Entries: []adaptermanifest.AllowlistID{
						{
							AdapterID:       identity.AdapterID,
							Publisher:       identity.Publisher,
							Version:         identity.Version,
							SignatureStatus: "valid",
						},
					},
				}),
				ExpectedContractCode: "",
			},
			AdapterAllowlistActivationCase{
				CaseID:    "adapter-allowlist-policy-conflict",
				Backend:   row.Backend,
				ProfileID: row.ProfileID,
				Manifest:  manifest,
				ActivationContext: withAllowlistPolicy(baseContext, adaptermanifest.AllowlistPolicy{
					Enabled:            true,
					EnforcementMode:    "strict",
					OnUnknownSignature: "deny",
					Entries: []adaptermanifest.AllowlistID{
						{
							AdapterID:       identity.AdapterID,
							Publisher:       identity.Publisher,
							Version:         identity.Version,
							SignatureStatus: "valid",
						},
					},
				}),
				ExpectedContractCode: adaptermanifest.CodeAllowlistPolicyConflict,
			},
		)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].CaseID != out[j].CaseID {
			return out[i].CaseID < out[j].CaseID
		}
		if out[i].Backend != out[j].Backend {
			return out[i].Backend < out[j].Backend
		}
		return out[i].ProfileID < out[j].ProfileID
	})
	return out
}

func EvaluateAdapterAllowlistActivationCase(in AdapterAllowlistActivationCase) (AdapterAllowlistActivationResult, error) {
	_, err := adaptermanifest.ActivateWithRequestAndProfileWindowWithContext(
		in.Manifest,
		"0.26.0-rc.2",
		[]string{"tool.invoke.required_input"},
		adaptermanifest.CapabilityRequest{
			Required: []string{"tool.invoke.required_input"},
		},
		adapterprofile.DefaultWindow(),
		in.ActivationContext,
	)
	if err == nil {
		return AdapterAllowlistActivationResult{Accepted: true}, nil
	}
	ce := &adaptermanifest.ContractError{}
	if !errors.As(err, &ce) {
		return AdapterAllowlistActivationResult{}, err
	}
	return AdapterAllowlistActivationResult{
		Accepted:     false,
		ContractCode: strings.TrimSpace(ce.Code),
	}, nil
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
		SandboxDriftAllowlistActivation,
		SandboxDriftAllowlistTaxonomy,
		SandboxDriftBackendProfile,
		SandboxDriftCapabilityClaim,
		SandboxDriftEgressPolicyDecision,
		SandboxDriftEgressSelectorPrecedence,
		SandboxDriftManifestCompat,
		SandboxDriftSessionLifecycle,
		SandboxDriftSessionModeCompat,
		SandboxDriftReasonTaxonomy,
	}
}

func classifySandboxEgressMatrixDrift(expect SandboxEgressPolicyCase, got SandboxEgressPolicyResult) string {
	if expect.ExpectedAction != got.Action || expect.ExpectedPolicySource != got.PolicySource {
		if expect.CaseID == "sandbox-egress-selector-override-precedence" {
			return SandboxDriftEgressSelectorPrecedence
		}
		return SandboxDriftEgressPolicyDecision
	}
	if got.Action == runtimeconfig.SecuritySandboxEgressActionDeny && got.ReasonCode != "sandbox.egress_deny" {
		return SandboxDriftReasonTaxonomy
	}
	if got.Action == runtimeconfig.SecuritySandboxEgressActionAllowAndRecord && got.ReasonCode != "sandbox.egress_allow_and_record" {
		return SandboxDriftReasonTaxonomy
	}
	if got.Action == runtimeconfig.SecuritySandboxEgressActionAllow && strings.TrimSpace(got.ReasonCode) != "" {
		return SandboxDriftReasonTaxonomy
	}
	return ""
}

func classifyAllowlistMatrixDrift(expectCode string, got AdapterAllowlistActivationResult) string {
	expected := strings.TrimSpace(expectCode)
	if expected == "" {
		if got.Accepted {
			return ""
		}
		if strings.HasPrefix(strings.TrimSpace(got.ContractCode), "adapter-manifest.allowlist-") {
			return SandboxDriftAllowlistActivation
		}
		return SandboxDriftAllowlistTaxonomy
	}
	if got.Accepted {
		return SandboxDriftAllowlistActivation
	}
	if strings.TrimSpace(got.ContractCode) != expected {
		if strings.HasPrefix(strings.TrimSpace(got.ContractCode), "adapter-manifest.allowlist-") {
			return SandboxDriftAllowlistActivation
		}
		return SandboxDriftAllowlistTaxonomy
	}
	return ""
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

func isSandboxEgressAction(action string) bool {
	switch strings.ToLower(strings.TrimSpace(action)) {
	case runtimeconfig.SecuritySandboxEgressActionDeny,
		runtimeconfig.SecuritySandboxEgressActionAllow,
		runtimeconfig.SecuritySandboxEgressActionAllowAndRecord:
		return true
	default:
		return false
	}
}

func isSandboxEgressOnViolation(action string) bool {
	switch strings.ToLower(strings.TrimSpace(action)) {
	case runtimeconfig.SecuritySandboxEgressOnViolationDeny,
		runtimeconfig.SecuritySandboxEgressOnViolationAllowAndRecord:
		return true
	default:
		return false
	}
}

func normalizeSandboxEgressPolicyMap(in map[string]string) map[string]string {
	if len(in) == 0 {
		return map[string]string{}
	}
	out := make(map[string]string, len(in))
	for rawKey, rawValue := range in {
		key := strings.ToLower(strings.TrimSpace(rawKey))
		value := strings.ToLower(strings.TrimSpace(rawValue))
		if key == "" || !isSandboxEgressAction(value) {
			continue
		}
		out[key] = value
	}
	return out
}

func normalizeSandboxAllowlistPatterns(in []string) []string {
	if len(in) == 0 {
		return nil
	}
	out := make([]string, 0, len(in))
	seen := map[string]struct{}{}
	for _, raw := range in {
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
	return out
}

func matchesSandboxAllowlist(host string, patterns []string) bool {
	target := strings.ToLower(strings.TrimSpace(host))
	if target == "" {
		return false
	}
	for _, raw := range patterns {
		pattern := strings.ToLower(strings.TrimSpace(raw))
		if pattern == "" {
			continue
		}
		if pattern == target {
			return true
		}
		if strings.HasPrefix(pattern, "*.") {
			suffix := strings.TrimPrefix(pattern, "*")
			if suffix != "" && strings.HasSuffix(target, suffix) {
				return true
			}
		}
	}
	return false
}

func sandboxAllowlistManifestForMatrix(row SandboxBackendSuite, identity adaptermanifest.AllowlistID) adaptermanifest.Manifest {
	return adaptermanifest.Manifest{
		Type:                   "tool",
		Name:                   "allowlist-tool-" + strings.ReplaceAll(row.Backend, "_", "-"),
		Version:                "0.1.0",
		ContractProfileVersion: adapterprofile.CurrentProfile,
		BaymaxCompat:           ">=0.26.0-rc.1 <0.27.0",
		Capabilities: adaptermanifest.Capabilities{
			Required: []string{"tool.invoke.required_input"},
			Optional: []string{},
		},
		ConformanceProfile:    "tool-invoke-fail-fast",
		SandboxBackend:        row.Backend,
		SandboxProfileID:      row.ProfileID,
		HostOS:                row.HostOS,
		HostArch:              row.HostArch,
		SessionModesSupported: append([]string(nil), row.SessionModesSupported...),
		Allowlist:             &identity,
	}
}

func withAllowlistPolicy(base adaptermanifest.ActivationContext, policy adaptermanifest.AllowlistPolicy) adaptermanifest.ActivationContext {
	out := base
	out.Allowlist = policy
	return out
}

func firstSandboxSessionMode(items []string) string {
	for _, raw := range items {
		mode := strings.ToLower(strings.TrimSpace(raw))
		if mode != "" {
			return mode
		}
	}
	return adaptermanifest.SandboxSessionModePerCall
}
