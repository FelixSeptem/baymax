package config

import (
	"strings"

	"github.com/FelixSeptem/baymax/extension"
)

// ExtensionAdmissionOutcome is the deterministic readiness projection for an extension.
type ExtensionAdmissionOutcome string

const (
	ExtensionAdmissionReady    ExtensionAdmissionOutcome = "ready"
	ExtensionAdmissionDegraded ExtensionAdmissionOutcome = "degraded"
	ExtensionAdmissionBlocked  ExtensionAdmissionOutcome = "blocked"
)

// ExtensionAdmissionDecision describes manifest and runtime readiness admission.
// Activated is set only after both admission stages allow and the activation callback succeeds.
type ExtensionAdmissionDecision struct {
	Outcome             ExtensionAdmissionOutcome
	Activated           bool
	ReasonCode          string
	ReadinessStatus     ReadinessStatus
	ReadinessReasonCode string
	CapabilityDowngrade bool
}

// ActivateExtension performs manifest-first admission, then delegates to the existing
// runtime readiness/policy gate. The callback is never invoked on a deny, keeping denied
// activation side-effect free.
func ActivateExtension(
	mgr *Manager,
	descriptor extension.Descriptor,
	input extension.AdmissionInput,
	activate func() error,
) (ExtensionAdmissionDecision, error) {
	if mgr == nil {
		return ExtensionAdmissionDecision{Outcome: ExtensionAdmissionBlocked, ReasonCode: ReadinessAdmissionCodeManagerNotReady}, nil
	}
	validated, err := extension.ValidateDescriptor(descriptor)
	if err != nil {
		return ExtensionAdmissionDecision{
			Outcome:    ExtensionAdmissionBlocked,
			ReasonCode: extension.ErrorCode(err),
		}, nil
	}
	manifestDecision, err := extension.Admit(validated, input)
	if err != nil || manifestDecision.Outcome != extension.AdmissionAllow {
		reason := extension.ErrorCode(err)
		if reason == "" {
			reason = extension.ReasonCompatibilityMismatch
		}
		return ExtensionAdmissionDecision{
			Outcome:    ExtensionAdmissionBlocked,
			ReasonCode: reason,
		}, nil
	}

	readinessDecision := mgr.EvaluateReadinessAdmission()
	capabilityResult, _ := extension.NegotiateCapabilities(validated, input.AvailableCapabilities, true)
	decision := ExtensionAdmissionDecision{
		ReadinessStatus:     readinessDecision.ReadinessStatus,
		ReadinessReasonCode: strings.TrimSpace(readinessDecision.ReasonCode),
		CapabilityDowngrade: capabilityResult.Downgraded,
	}
	if readinessDecision.Outcome == ReadinessAdmissionOutcomeDeny {
		decision.Outcome = ExtensionAdmissionBlocked
		decision.ReasonCode = strings.TrimSpace(readinessDecision.ReasonCode)
		if decision.ReasonCode == "" {
			decision.ReasonCode = ReadinessAdmissionCodeBlocked
		}
		return decision, nil
	}
	if readinessDecision.ReadinessStatus == ReadinessStatusDegraded {
		decision.Outcome = ExtensionAdmissionDegraded
	} else {
		decision.Outcome = ExtensionAdmissionReady
	}
	if activate != nil {
		if err := activate(); err != nil {
			return decision, err
		}
	}
	decision.Activated = true
	return decision, nil
}

// EvaluateExtensionAdmission projects the same admission contract without invoking
// extension code. It is used by Run and Stream paths to assert semantic parity.
func EvaluateExtensionAdmission(
	mgr *Manager,
	descriptor extension.Descriptor,
	input extension.AdmissionInput,
) (ExtensionAdmissionDecision, error) {
	return ActivateExtension(mgr, descriptor, input, nil)
}
