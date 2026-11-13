package sandboxexternal

import (
	"github.com/FelixSeptem/baymax/core/types"
	runtimeconfig "github.com/FelixSeptem/baymax/runtime/config"
)

// NewE2BExecutor builds the e2b adapter, mapped to the oci_runtime canonical profile.
// Point BaseURL at a local e2b dev server (docker) or the e2b cloud API.
func NewE2BExecutor(cfg Config) (types.SandboxExecutor, error) {
	caps := append(egressCaps(),
		runtimeconfig.SecuritySandboxCapabilityMemoryLimit,
		runtimeconfig.SecuritySandboxCapabilityPIDLimit,
	)
	return newCloud(cfg, caps, perCallPerSessionModes(), "/v1/exec"), nil
}
