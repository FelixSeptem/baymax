package composer

import "github.com/FelixSeptem/baymax/core/types"

// ProtocolDescriptorForRuntime provides an opt-in Composer descriptor while
// keeping scheduling and admission semantics source-owned.
func ProtocolDescriptorForRuntime(runtimeID, profileVersion string, capabilities []types.ProtocolCapability, actions []types.ProtocolAction) (types.ProtocolDescriptor, error) {
	return types.ProtocolDescriptorForSource(types.ProtocolSourceComposer, runtimeID, profileVersion, capabilities, actions)
}
