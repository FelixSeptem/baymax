package types

import (
	"fmt"
	"strings"
	"time"
	"unicode"
)

// RuntimeInputProtocolVersionV1 identifies the first steering/follow-up input profile.
const RuntimeInputProtocolVersionV1 = "runtime_input.v1"

// RuntimeInputMaxPayloadBytes bounds content carried by a single input envelope.
const RuntimeInputMaxPayloadBytes = 16 * 1024

type RuntimeInputKind string

const (
	RuntimeInputKindSteering RuntimeInputKind = "steering"
	RuntimeInputKindFollowUp RuntimeInputKind = "follow_up"
)

type RuntimeInputApplyBoundary string

const (
	RuntimeInputApplyBoundaryUnspecified  RuntimeInputApplyBoundary = ""
	RuntimeInputApplyBoundaryNextDecision RuntimeInputApplyBoundary = "next_decision"
	RuntimeInputApplyBoundaryIdleTerminal RuntimeInputApplyBoundary = "idle_terminal"
)

// RuntimeInputEnvelope is the transport-neutral source admission request.
// Payload is intentionally opaque to the protocol and is bounded before source mutation.
type RuntimeInputEnvelope struct {
	Version           string                    `json:"version"`
	InputID           string                    `json:"input_id"`
	Kind              RuntimeInputKind          `json:"kind"`
	Time              time.Time                 `json:"time"`
	SessionID         string                    `json:"session_id"`
	RunID             string                    `json:"run_id"`
	CausationID       string                    `json:"causation_id,omitempty"`
	SourceCorrelation string                    `json:"source_correlation,omitempty"`
	ProfileVersion    string                    `json:"profile_version,omitempty"`
	ApplyBoundary     RuntimeInputApplyBoundary `json:"apply_boundary,omitempty"`
	Payload           string                    `json:"payload"`
}

func (in RuntimeInputEnvelope) Validate() error {
	if in.Version != RuntimeInputProtocolVersionV1 {
		return fmt.Errorf("%s: %s", RuntimeInputReasonUnsupportedVersion, in.Version)
	}
	if err := validateRuntimeInputID(in.InputID, "input_id"); err != nil {
		return err
	}
	if in.Kind != RuntimeInputKindSteering && in.Kind != RuntimeInputKindFollowUp {
		return fmt.Errorf("%s: %q", RuntimeInputReasonUnsupportedKind, in.Kind)
	}
	if in.Time.IsZero() {
		return fmt.Errorf("%s: timestamp is required", RuntimeInputReasonInvalidEnvelope)
	}
	for name, value := range map[string]string{
		"session_id":         in.SessionID,
		"run_id":             in.RunID,
		"causation_id":       in.CausationID,
		"source_correlation": in.SourceCorrelation,
		"profile_version":    in.ProfileVersion,
	} {
		if name == "causation_id" || name == "source_correlation" || name == "profile_version" {
			if strings.TrimSpace(value) == "" {
				continue
			}
		}
		if err := validateRuntimeInputID(value, name); err != nil {
			return err
		}
	}
	if strings.TrimSpace(in.Payload) == "" {
		return fmt.Errorf("%s: payload required", RuntimeInputReasonInvalidEnvelope)
	}
	if len([]byte(in.Payload)) > RuntimeInputMaxPayloadBytes {
		return fmt.Errorf("%s: payload too large", RuntimeInputReasonPayloadTooLarge)
	}
	for _, r := range in.Payload {
		if unicode.IsControl(r) {
			return fmt.Errorf("%s: payload contains control character", RuntimeInputReasonInvalidEnvelope)
		}
	}
	switch in.ApplyBoundary {
	case RuntimeInputApplyBoundaryUnspecified, RuntimeInputApplyBoundaryNextDecision, RuntimeInputApplyBoundaryIdleTerminal:
	default:
		return fmt.Errorf("%s: invalid apply boundary %q", RuntimeInputReasonInvalidEnvelope, in.ApplyBoundary)
	}
	return nil
}

func (in RuntimeInputEnvelope) NormalizedIdentity() string {
	return strings.Join([]string{strings.TrimSpace(in.SessionID), strings.TrimSpace(in.RunID), strings.TrimSpace(in.InputID)}, "/")
}

type RuntimeInputAdmissionStatus string

const (
	RuntimeInputAdmissionStatusAccepted  RuntimeInputAdmissionStatus = "accepted"
	RuntimeInputAdmissionStatusRejected  RuntimeInputAdmissionStatus = "rejected"
	RuntimeInputAdmissionStatusDuplicate RuntimeInputAdmissionStatus = "duplicate"
)

const (
	RuntimeInputReasonAccepted           = "runtime_input.accepted"
	RuntimeInputReasonRejected           = "runtime_input.rejected"
	RuntimeInputReasonDuplicate          = "runtime_input.duplicate"
	RuntimeInputReasonStale              = "runtime_input.stale"
	RuntimeInputReasonTerminal           = "runtime_input.terminal"
	RuntimeInputReasonBackpressure       = "runtime_input.backpressure"
	RuntimeInputReasonDisconnected       = "runtime_input.disconnected"
	RuntimeInputReasonNotApplied         = "runtime_input.not_applied"
	RuntimeInputReasonInvalidEnvelope    = "runtime_input.invalid_envelope"
	RuntimeInputReasonUnsupportedVersion = "runtime_input.unsupported_version"
	RuntimeInputReasonUnsupportedKind    = "runtime_input.unsupported_kind"
	RuntimeInputReasonPayloadTooLarge    = "runtime_input.payload_too_large"
)

func IsValidRuntimeInputAdmissionStatus(status RuntimeInputAdmissionStatus) bool {
	switch status {
	case RuntimeInputAdmissionStatusAccepted, RuntimeInputAdmissionStatusRejected, RuntimeInputAdmissionStatusDuplicate:
		return true
	default:
		return false
	}
}

func IsValidRuntimeInputReason(reason string) bool {
	switch reason {
	case RuntimeInputReasonAccepted, RuntimeInputReasonRejected, RuntimeInputReasonDuplicate, RuntimeInputReasonStale, RuntimeInputReasonTerminal, RuntimeInputReasonBackpressure, RuntimeInputReasonDisconnected, RuntimeInputReasonNotApplied, RuntimeInputReasonInvalidEnvelope, RuntimeInputReasonUnsupportedVersion, RuntimeInputReasonUnsupportedKind, RuntimeInputReasonPayloadTooLarge:
		return true
	default:
		return false
	}
}

type RuntimeInputAdmission struct {
	Version           string                      `json:"version"`
	InputID           string                      `json:"input_id"`
	Kind              RuntimeInputKind            `json:"kind"`
	SessionID         string                      `json:"session_id"`
	RunID             string                      `json:"run_id"`
	CausationID       string                      `json:"causation_id,omitempty"`
	SourceCorrelation string                      `json:"source_correlation,omitempty"`
	Status            RuntimeInputAdmissionStatus `json:"status"`
	ReasonCode        string                      `json:"reason_code"`
}

func NormalizeRuntimeInputAdmission(in RuntimeInputEnvelope, status RuntimeInputAdmissionStatus, reason string) (RuntimeInputAdmission, error) {
	if err := in.Validate(); err != nil {
		return RuntimeInputAdmission{}, err
	}
	if !IsValidRuntimeInputAdmissionStatus(status) {
		return RuntimeInputAdmission{}, fmt.Errorf("%s: invalid status %q", RuntimeInputReasonInvalidEnvelope, status)
	}
	if strings.TrimSpace(reason) == "" {
		switch status {
		case RuntimeInputAdmissionStatusAccepted:
			reason = RuntimeInputReasonAccepted
		case RuntimeInputAdmissionStatusDuplicate:
			reason = RuntimeInputReasonDuplicate
		default:
			reason = RuntimeInputReasonInvalidEnvelope
		}
	}
	return RuntimeInputAdmission{Version: in.Version, InputID: in.InputID, Kind: in.Kind, SessionID: in.SessionID, RunID: in.RunID, CausationID: in.CausationID, SourceCorrelation: in.SourceCorrelation, Status: status, ReasonCode: reason}, nil
}

func validateRuntimeInputID(value, name string) error {
	s := strings.TrimSpace(value)
	if s == "" {
		return fmt.Errorf("%s: %s required", RuntimeInputReasonInvalidEnvelope, name)
	}
	if len(s) > 256 {
		return fmt.Errorf("%s: %s too long", RuntimeInputReasonInvalidEnvelope, name)
	}
	for _, r := range s {
		if unicode.IsSpace(r) || unicode.IsControl(r) {
			return fmt.Errorf("%s: invalid %s", RuntimeInputReasonInvalidEnvelope, name)
		}
	}
	return nil
}
