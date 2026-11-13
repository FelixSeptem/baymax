package sandboxexternal

import (
	"github.com/FelixSeptem/baymax/core/types"
	runtimeconfig "github.com/FelixSeptem/baymax/runtime/config"
)

// NewAnthropicExecutor builds the anthropics/sandbox-runtime adapter, mapped to the
// oci_runtime canonical profile (Firecracker micro-VM). Point BaseURL at a local
// sandbox-runtime server (docker) or the managed API.
func NewAnthropicExecutor(cfg Config) (types.SandboxExecutor, error) {
	caps := append(egressCaps(),
		runtimeconfig.SecuritySandboxCapabilityMemoryLimit,
		runtimeconfig.SecuritySandboxCapabilityPIDLimit,
		runtimeconfig.SecuritySandboxCapabilityOOMSignal,
	)
	return newCloud(cfg, caps, perCallPerSessionModes(), "/v1/exec"), nil
}
