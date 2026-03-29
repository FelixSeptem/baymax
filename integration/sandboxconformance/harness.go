package sandboxconformance

import (
	"fmt"
	"sort"
	"strings"
	"sync"

	"github.com/FelixSeptem/baymax/core/types"
	runtimeconfig "github.com/FelixSeptem/baymax/runtime/config"
)

const (
	ReasonCapabilityMismatch     = "sandbox.capability_mismatch"
	ReasonSessionModeUnsupported = "sandbox.session_mode_unsupported"
	ReasonFallbackAllowAndRecord = "sandbox.fallback_allow_and_record"
	ReasonLaunchFailed           = "sandbox.launch_failed"
)

type Scenario struct {
	ID                   string
	Backend              string
	SessionMode          types.SandboxSessionMode
	RequiredCapabilities []string
	ProbeCapabilities    []string
	SupportedModes       []string
	FallbackAction       string
}

var MinimumMatrix = []Scenario{
	{
		ID:          "linux-nsjail-canonical",
		Backend:     runtimeconfig.SecuritySandboxBackendLinuxNSJail,
		SessionMode: types.SandboxSessionModePerCall,
		RequiredCapabilities: []string{
			runtimeconfig.SecuritySandboxCapabilityStdoutStderrCapture,
			runtimeconfig.SecuritySandboxCapabilitySessionPerCall,
		},
		ProbeCapabilities: []string{
			runtimeconfig.SecuritySandboxCapabilityStdoutStderrCapture,
			runtimeconfig.SecuritySandboxCapabilitySessionPerCall,
			runtimeconfig.SecuritySandboxCapabilityNetworkOff,
		},
		SupportedModes: []string{
			runtimeconfig.SecuritySandboxSessionModePerCall,
			runtimeconfig.SecuritySandboxSessionModePerSession,
		},
		FallbackAction: runtimeconfig.SecuritySandboxFallbackDeny,
	},
	{
		ID:          "linux-bwrap-canonical",
		Backend:     runtimeconfig.SecuritySandboxBackendLinuxBwrap,
		SessionMode: types.SandboxSessionModePerCall,
		RequiredCapabilities: []string{
			runtimeconfig.SecuritySandboxCapabilityStdoutStderrCapture,
			runtimeconfig.SecuritySandboxCapabilitySessionPerCall,
		},
		ProbeCapabilities: []string{
			runtimeconfig.SecuritySandboxCapabilityStdoutStderrCapture,
			runtimeconfig.SecuritySandboxCapabilitySessionPerCall,
			runtimeconfig.SecuritySandboxCapabilityReadonlyRoot,
		},
		SupportedModes: []string{
			runtimeconfig.SecuritySandboxSessionModePerCall,
		},
		FallbackAction: runtimeconfig.SecuritySandboxFallbackAllowAndRecord,
	},
	{
		ID:          "oci-runtime-canonical",
		Backend:     runtimeconfig.SecuritySandboxBackendOCIRuntime,
		SessionMode: types.SandboxSessionModePerCall,
		RequiredCapabilities: []string{
			runtimeconfig.SecuritySandboxCapabilityStdoutStderrCapture,
			runtimeconfig.SecuritySandboxCapabilitySessionPerCall,
		},
		ProbeCapabilities: []string{
			runtimeconfig.SecuritySandboxCapabilityStdoutStderrCapture,
			runtimeconfig.SecuritySandboxCapabilitySessionPerCall,
			runtimeconfig.SecuritySandboxCapabilityMemoryLimit,
			runtimeconfig.SecuritySandboxCapabilityPIDLimit,
		},
		SupportedModes: []string{
			runtimeconfig.SecuritySandboxSessionModePerCall,
			runtimeconfig.SecuritySandboxSessionModePerSession,
		},
		FallbackAction: runtimeconfig.SecuritySandboxFallbackDeny,
	},
	{
		ID:          "windows-job-canonical",
		Backend:     runtimeconfig.SecuritySandboxBackendWindowsJob,
		SessionMode: types.SandboxSessionModePerCall,
		RequiredCapabilities: []string{
			runtimeconfig.SecuritySandboxCapabilityStdoutStderrCapture,
			runtimeconfig.SecuritySandboxCapabilitySessionPerCall,
		},
		ProbeCapabilities: []string{
			runtimeconfig.SecuritySandboxCapabilityStdoutStderrCapture,
			runtimeconfig.SecuritySandboxCapabilitySessionPerCall,
		},
		SupportedModes: []string{
			runtimeconfig.SecuritySandboxSessionModePerCall,
		},
		FallbackAction: runtimeconfig.SecuritySandboxFallbackDeny,
	},
}

type ConformanceError struct {
	ReasonCode string
	Message    string
}

func (e *ConformanceError) Error() string {
	if e == nil {
		return ""
	}
	return e.Message
}

func ValidateMinimumMatrix(matrix []Scenario) error {
	if len(matrix) == 0 {
		return fmt.Errorf("empty sandbox conformance matrix")
	}
	requiredBackends := map[string]struct{}{
		runtimeconfig.SecuritySandboxBackendLinuxNSJail: {},
		runtimeconfig.SecuritySandboxBackendLinuxBwrap:  {},
		runtimeconfig.SecuritySandboxBackendOCIRuntime:  {},
		runtimeconfig.SecuritySandboxBackendWindowsJob:  {},
	}
	coveredBackends := map[string]struct{}{}
	ids := map[string]struct{}{}
	for i := range matrix {
		row := matrix[i]
		id := strings.TrimSpace(row.ID)
		if id == "" {
			return fmt.Errorf("scenario[%d] id must not be empty", i)
		}
		if _, dup := ids[id]; dup {
			return fmt.Errorf("duplicate scenario id: %s", id)
		}
		ids[id] = struct{}{}
		if _, ok := requiredBackends[row.Backend]; !ok {
			return fmt.Errorf("scenario[%d] has unsupported backend %q", i, row.Backend)
		}
		coveredBackends[row.Backend] = struct{}{}
		if len(row.RequiredCapabilities) == 0 {
			return fmt.Errorf("scenario[%s] must include required capabilities", row.ID)
		}
		if len(row.ProbeCapabilities) == 0 {
			return fmt.Errorf("scenario[%s] must include probe capabilities", row.ID)
		}
		if len(row.SupportedModes) == 0 {
			return fmt.Errorf("scenario[%s] must include supported session modes", row.ID)
		}
		if row.SessionMode != types.SandboxSessionModePerCall && row.SessionMode != types.SandboxSessionModePerSession {
			return fmt.Errorf("scenario[%s] has unsupported session mode %q", row.ID, row.SessionMode)
		}
		if row.FallbackAction != runtimeconfig.SecuritySandboxFallbackAllowAndRecord &&
			row.FallbackAction != runtimeconfig.SecuritySandboxFallbackDeny {
			return fmt.Errorf("scenario[%s] has unsupported fallback action %q", row.ID, row.FallbackAction)
		}
	}
	for backend := range requiredBackends {
		if _, ok := coveredBackends[backend]; !ok {
			return fmt.Errorf("missing required backend coverage: %s", backend)
		}
	}
	return nil
}

func EvaluateCapabilityNegotiation(probe types.SandboxCapabilityProbe, required []string, mode types.SandboxSessionMode) error {
	normalizedRequired := normalizeKeywords(required)
	missing := make([]string, 0)
	for _, capability := range normalizedRequired {
		if !probe.Supports(capability) {
			missing = append(missing, capability)
		}
	}
	if len(missing) > 0 {
		sort.Strings(missing)
		return &ConformanceError{
			ReasonCode: ReasonCapabilityMismatch,
			Message:    fmt.Sprintf("%s: missing required capabilities [%s]", ReasonCapabilityMismatch, strings.Join(missing, ",")),
		}
	}
	if mode != "" && !probe.SupportsSessionMode(mode) {
		return &ConformanceError{
			ReasonCode: ReasonSessionModeUnsupported,
			Message:    fmt.Sprintf("%s: probe does not support %s", ReasonSessionModeUnsupported, mode),
		}
	}
	return nil
}

func CanonicalizeExecResult(in types.SandboxExecResult) types.SandboxExecResult {
	out := in
	if len(out.ViolationCodes) == 0 {
		return out
	}
	normalized := make([]string, 0, len(out.ViolationCodes))
	seen := map[string]struct{}{}
	for _, code := range out.ViolationCodes {
		key := strings.ToLower(strings.TrimSpace(code))
		if key == "" {
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		normalized = append(normalized, key)
	}
	sort.Strings(normalized)
	out.ViolationCodes = normalized
	return out
}

type SessionLifecycleHarness struct {
	mu         sync.Mutex
	seq        int
	perSession map[string]string
}

func NewSessionLifecycleHarness() *SessionLifecycleHarness {
	return &SessionLifecycleHarness{
		perSession: map[string]string{},
	}
}

func (h *SessionLifecycleHarness) Acquire(mode types.SandboxSessionMode, sessionID string) string {
	h.mu.Lock()
	defer h.mu.Unlock()

	if mode == types.SandboxSessionModePerSession {
		id := strings.TrimSpace(sessionID)
		if id != "" {
			if token, ok := h.perSession[id]; ok {
				return token
			}
			h.seq++
			token := fmt.Sprintf("session-%03d", h.seq)
			h.perSession[id] = token
			return token
		}
	}
	h.seq++
	return fmt.Sprintf("call-%03d", h.seq)
}

func (h *SessionLifecycleHarness) Close(sessionID string) bool {
	h.mu.Lock()
	defer h.mu.Unlock()

	id := strings.TrimSpace(sessionID)
	if id == "" {
		return false
	}
	if _, ok := h.perSession[id]; !ok {
		return false
	}
	delete(h.perSession, id)
	return true
}

type FallbackOutcome struct {
	Decision   string
	ReasonCode string
}

func ResolveLaunchFailureFallback(action string) (FallbackOutcome, error) {
	normalized := strings.ToLower(strings.TrimSpace(action))
	if normalized == "" {
		normalized = runtimeconfig.SecuritySandboxFallbackDeny
	}
	switch normalized {
	case runtimeconfig.SecuritySandboxFallbackAllowAndRecord:
		return FallbackOutcome{
			Decision:   runtimeconfig.SecuritySandboxActionHost,
			ReasonCode: ReasonFallbackAllowAndRecord,
		}, nil
	case runtimeconfig.SecuritySandboxFallbackDeny:
		return FallbackOutcome{
			Decision:   runtimeconfig.SecuritySandboxActionDeny,
			ReasonCode: ReasonLaunchFailed,
		}, nil
	default:
		return FallbackOutcome{}, fmt.Errorf("unsupported fallback action: %s", action)
	}
}

func normalizeKeywords(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	out := make([]string, 0, len(values))
	seen := map[string]struct{}{}
	for _, value := range values {
		key := strings.ToLower(strings.TrimSpace(value))
		if key == "" {
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
