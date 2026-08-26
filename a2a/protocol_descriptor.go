package a2a

import "github.com/FelixSeptem/baymax/core/types"

// ProtocolDescriptorForRuntime provides an opt-in A2A descriptor.
func ProtocolDescriptorForRuntime(runtimeID, profileVersion string, capabilities []types.ProtocolCapability, actions []types.ProtocolAction) (types.ProtocolDescriptor, error) {
	return types.ProtocolDescriptorForSource(types.ProtocolSourceA2A, runtimeID, profileVersion, capabilities, actions)
}
