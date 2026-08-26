package scheduler

import "github.com/FelixSeptem/baymax/core/types"

// ProtocolDescriptorForRuntime provides an opt-in Scheduler descriptor; it
// does not expose or implement a second queue.
func ProtocolDescriptorForRuntime(runtimeID, profileVersion string, capabilities []types.ProtocolCapability, actions []types.ProtocolAction) (types.ProtocolDescriptor, error) {
	return types.ProtocolDescriptorForSource(types.ProtocolSourceScheduler, runtimeID, profileVersion, capabilities, actions)
}
