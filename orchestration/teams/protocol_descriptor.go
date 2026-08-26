package teams

import "github.com/FelixSeptem/baymax/core/types"

// ProtocolDescriptorForRuntime provides an opt-in Teams descriptor.
func ProtocolDescriptorForRuntime(runtimeID, profileVersion string, capabilities []types.ProtocolCapability, actions []types.ProtocolAction) (types.ProtocolDescriptor, error) {
	return types.ProtocolDescriptorForSource(types.ProtocolSourceTeams, runtimeID, profileVersion, capabilities, actions)
}
