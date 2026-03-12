package reliability

import (
	"github.com/FelixSeptem/baymax/core/types"
	mcpprofile "github.com/FelixSeptem/baymax/mcp/profile"
	runtimeconfig "github.com/FelixSeptem/baymax/runtime/config"
)

func ResolveStartupPolicy(profile mcpprofile.Name, runtimeMgr *runtimeconfig.Manager, runtimePolicy *types.MCPRuntimePolicy) (types.MCPRuntimePolicy, error) {
	if runtimeMgr == nil {
		return mcpprofile.Resolve(profile, runtimePolicy), nil
	}
	return runtimeMgr.ResolvePolicy(profile, runtimePolicy)
}

func ResolveRuntimePolicy(profile mcpprofile.Name, runtimeMgr *runtimeconfig.Manager, explicitRuntimePolicy *types.MCPRuntimePolicy) (types.MCPRuntimePolicy, error) {
	if profile == "" {
		profile = mcpprofile.Default
	}
	if runtimeMgr == nil {
		return mcpprofile.Resolve(profile, explicitRuntimePolicy), nil
	}
	return runtimeMgr.ResolvePolicy(profile, explicitRuntimePolicy)
}
