package sandboxconformance

import (
	"fmt"
	"strings"

	"github.com/FelixSeptem/baymax/core/types"
	runtimeconfig "github.com/FelixSeptem/baymax/runtime/config"
)

// ExternalBackendMatrix covers mainstream external sandbox runtimes registered as
// canonical backend variants. Each scenario maps an external backend to its
// canonical profile and required probe capabilities, so the same frozen
// contract (capability negotiation, session modes, fallback) applies across
// e2b, open-sandbox and anthropics/sandbox-runtime.
var ExternalBackendMatrix = []Scenario{
	{
		ID:          "e2b-oci-runtime",
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
		ID:          "open-sandbox-oci-runtime",
		Backend:     runtimeconfig.SecuritySandboxBackendOCIRuntime,
		SessionMode: types.SandboxSessionModePerCall,
		RequiredCapabilities: []string{
			runtimeconfig.SecuritySandboxCapabilityStdoutStderrCapture,
			runtimeconfig.SecuritySandboxCapabilitySessionPerCall,
		},
		ProbeCapabilities: []string{
			runtimeconfig.SecuritySandboxCapabilityStdoutStderrCapture,
			runtimeconfig.SecuritySandboxCapabilitySessionPerCall,
			runtimeconfig.SecuritySandboxCapabilityReadonlyRoot,
			runtimeconfig.SecuritySandboxCapabilityMemoryLimit,
		},
		SupportedModes: []string{
			runtimeconfig.SecuritySandboxSessionModePerCall,
			runtimeconfig.SecuritySandboxSessionModePerSession,
		},
		FallbackAction: runtimeconfig.SecuritySandboxFallbackDeny,
	},
	{
		ID:          "anthropic-sandbox-runtime-oci-runtime",
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
			runtimeconfig.SecuritySandboxCapabilityOOMSignal,
		},
		SupportedModes: []string{
			runtimeconfig.SecuritySandboxSessionModePerCall,
			runtimeconfig.SecuritySandboxSessionModePerSession,
		},
		FallbackAction: runtimeconfig.SecuritySandboxFallbackDeny,
	},
}

// ValidateExternalBackendMatrix checks each external scenario without requiring
// full coverage of all four canonical backends (they are variants of existing
// ones). It reuses the same structural invariants as ValidateMinimumMatrix.
func ValidateExternalBackendMatrix(matrix []Scenario) error {
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
	return nil
}
