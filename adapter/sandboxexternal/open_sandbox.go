package sandboxexternal

import (
	"github.com/FelixSeptem/baymax/core/types"
	runtimeconfig "github.com/FelixSeptem/baymax/runtime/config"
)

// NewOpenSandboxExecutor builds the open-sandbox.ai adapter, mapped to the
// oci_runtime canonical profile. Point BaseURL at a local open-sandbox server
// (docker) or the managed API.
func NewOpenSandboxExecutor(cfg Config) (types.SandboxExecutor, error) {
	caps := append(egressCaps(),
		runtimeconfig.SecuritySandboxCapabilityReadonlyRoot,
		runtimeconfig.SecuritySandboxCapabilityMemoryLimit,
	)
	return newCloud(cfg, caps, perCallPerSessionModes(), "/v1/exec"), nil
}
