package workflow

import "github.com/FelixSeptem/baymax/core/types"

// ProtocolDescriptorForRuntime provides an opt-in workflow descriptor.
func ProtocolDescriptorForRuntime(runtimeID, profileVersion string, capabilities []types.ProtocolCapability, actions []types.ProtocolAction) (types.ProtocolDescriptor, error) {
	return types.ProtocolDescriptorForSource(types.ProtocolSourceWorkflow, runtimeID, profileVersion, capabilities, actions)
}
