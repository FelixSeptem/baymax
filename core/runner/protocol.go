package runner

import "github.com/FelixSeptem/baymax/core/types"

// ProtocolDescriptorForRuntime exposes an opt-in Runner descriptor. Runner
// remains the owner of execution, authorization, and concurrent admission.
func ProtocolDescriptorForRuntime(runtimeID, profileVersion string, capabilities []types.ProtocolCapability, actions []types.ProtocolAction) (types.ProtocolDescriptor, error) {
	return types.ProtocolDescriptorForSource(types.ProtocolSourceRunner, runtimeID, profileVersion, capabilities, actions)
}

// RealtimeProtocolDescriptorForRuntime exposes the Realtime source descriptor
// without changing interrupt/resume ownership or event ordering semantics.
func RealtimeProtocolDescriptorForRuntime(runtimeID, profileVersion string, capabilities []types.ProtocolCapability, actions []types.ProtocolAction) (types.ProtocolDescriptor, error) {
	return types.ProtocolDescriptorForSource(types.ProtocolSourceRealtime, runtimeID, profileVersion, capabilities, actions)
}

// ProjectRealtimeEventStreamBinding delegates to the transport-neutral
// protocol projection while keeping Realtime as the source owner.
func ProjectRealtimeEventStreamBinding(subscription types.EventStreamSubscription, outcome types.EventStreamBindingOutcome, history, live []types.RealtimeEventEnvelope) (types.EventStreamBindingProjection, error) {
	return types.ProjectEventStreamBinding(subscription, outcome, history, live)
}

// ProjectRealtimeEventStreamTerminalRecovery composes the source-owned
// Realtime binding with the terminal recovery projection. It remains pure:
// Realtime owns history/cursors and the caller supplies terminal facts.
func ProjectRealtimeEventStreamTerminalRecovery(subscription types.EventStreamSubscription, outcome types.EventStreamBindingOutcome, history, live []types.RealtimeEventEnvelope, observation types.EventStreamTerminalRecoveryObservation) (types.EventStreamTerminalRecovery, error) {
	binding, err := ProjectRealtimeEventStreamBinding(subscription, outcome, history, live)
	if err != nil {
		return types.EventStreamTerminalRecovery{}, err
	}
	return types.ProjectEventStreamTerminalRecovery(binding, observation)
}
