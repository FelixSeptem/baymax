package types

import (
	"context"
	"fmt"
	"strings"
	"time"
	"unicode"
)

// HostProtocolVersionV1 identifies the first embedded host contract profile.
const HostProtocolVersionV1 = "embedded_host_protocol.v1"

type HostEnvelopeKind string

const (
	HostEnvelopeKindCommand         HostEnvelopeKind = "command"
	HostEnvelopeKindCommandResponse HostEnvelopeKind = "command_response"
	HostEnvelopeKindRuntimeEvent    HostEnvelopeKind = "runtime_event"
	HostEnvelopeKindHostRequest     HostEnvelopeKind = "host_request"
	HostEnvelopeKindHostResponse    HostEnvelopeKind = "host_response"
)

type HostCommandKind string

const (
	HostCommandKindRunStart          HostCommandKind = "run.start"
	HostCommandKindAction            HostCommandKind = "run.action"
	HostCommandKindRunAction         HostCommandKind = "run.action"
	HostCommandKindRealtimeInterrupt HostCommandKind = "realtime.interrupt"
	HostCommandKindRealtimeResume    HostCommandKind = "realtime.resume"
	HostCommandKindHITLRespond       HostCommandKind = "hitl.respond"
	HostCommandKindEventsSubscribe   HostCommandKind = "events.subscribe"
	HostCommandKindRunGet            HostCommandKind = "run.get"
)

type HostAdmissionStatus string

const (
	HostAdmissionStatusAccepted  HostAdmissionStatus = "accepted"
	HostAdmissionStatusRejected  HostAdmissionStatus = "rejected"
	HostAdmissionStatusDuplicate HostAdmissionStatus = "duplicate"
)

// HostRunExecutionMode selects the source-owned Run or Stream execution path.
type HostRunExecutionMode string

const (
	HostRunExecutionModeRun    HostRunExecutionMode = "run"
	HostRunExecutionModeStream HostRunExecutionMode = "stream"
)

// HostRunExecution is an admitted source execution. The source owns its
// lifecycle; Release abandons an execution that cannot be activated.
type HostRunExecution interface {
	Execute(EventHandler) (RunResult, error)
	Release()
}

// HostRunStartAdmission separates source admission from execution activation.
type HostRunStartAdmission struct {
	Status     HostAdmissionStatus
	ReasonCode string
	RunID      string
	Execution  HostRunExecution
}

// HostRunStarter admits a Run before any business execution begins.
type HostRunStarter interface {
	AdmitHostRun(context.Context, RunRequest, HostRunExecutionMode) (HostRunStartAdmission, error)
}

const (
	HostReasonInvalidEnvelope    = "host.invalid_envelope"
	HostReasonUnsupportedVersion = "host.unsupported_version"
	HostReasonUnknownKind        = "host.unknown_kind"
	HostReasonUnauthorized       = "host.unauthorized"
	HostReasonAlreadyTerminal    = "host.already_terminal"
	HostReasonDuplicate          = "host.duplicate"
)

// HostCommandEnvelope is the transport-neutral command DTO.
type HostCommandEnvelope struct {
	Version           string          `json:"version"`
	MessageID         string          `json:"message_id"`
	Kind              HostCommandKind `json:"kind"`
	Time              time.Time       `json:"time"`
	RequestID         string          `json:"request_id"`
	SessionID         string          `json:"session_id,omitempty"`
	RunID             string          `json:"run_id,omitempty"`
	CausationID       string          `json:"causation_id,omitempty"`
	SourceCorrelation string          `json:"source_correlation,omitempty"`
	Payload           map[string]any  `json:"payload,omitempty"`
}

func (c HostCommandEnvelope) Validate() error {
	if c.Version != HostProtocolVersionV1 {
		return fmt.Errorf("%s: %s", HostReasonUnsupportedVersion, c.Version)
	}
	if err := validateHostID(c.MessageID, "message_id"); err != nil {
		return err
	}
	if c.Time.IsZero() {
		return fmt.Errorf("%s: timestamp is required", HostReasonInvalidEnvelope)
	}
	if !isHostCommandKind(c.Kind) {
		return fmt.Errorf("%s: %q", HostReasonUnknownKind, c.Kind)
	}
	if err := validateHostID(c.RequestID, "request_id"); err != nil {
		return err
	}
	switch c.Kind {
	case HostCommandKindEventsSubscribe, HostCommandKindRunGet:
		if strings.TrimSpace(c.SessionID) == "" && strings.TrimSpace(c.RunID) == "" {
			return fmt.Errorf("%s: session_id or run_id required", HostReasonInvalidEnvelope)
		}
	case HostCommandKindRunStart:
		if strings.TrimSpace(c.SessionID) == "" {
			return fmt.Errorf("%s: session_id required", HostReasonInvalidEnvelope)
		}
	default:
		if strings.TrimSpace(c.SessionID) == "" {
			return fmt.Errorf("%s: session_id required", HostReasonInvalidEnvelope)
		}
		if strings.TrimSpace(c.RunID) == "" {
			return fmt.Errorf("%s: run_id required", HostReasonInvalidEnvelope)
		}
	}
	for name, value := range map[string]string{"session_id": c.SessionID, "run_id": c.RunID, "causation_id": c.CausationID, "source_correlation": c.SourceCorrelation} {
		if strings.TrimSpace(value) != "" {
			if err := validateHostID(value, name); err != nil {
				return err
			}
		}
	}
	return nil
}

type HostCommandResponse struct {
	Version           string              `json:"version"`
	MessageID         string              `json:"message_id"`
	Kind              HostEnvelopeKind    `json:"kind"`
	Time              time.Time           `json:"time"`
	RequestID         string              `json:"request_id"`
	SessionID         string              `json:"session_id,omitempty"`
	RunID             string              `json:"run_id,omitempty"`
	Status            HostAdmissionStatus `json:"status"`
	ReasonCode        string              `json:"reason_code,omitempty"`
	SourceCorrelation string              `json:"source_correlation,omitempty"`
	Terminal          *TerminalOutcome    `json:"terminal,omitempty"`
}

func (r HostCommandResponse) Validate() error {
	if r.Version != HostProtocolVersionV1 {
		return fmt.Errorf("%s: %s", HostReasonUnsupportedVersion, r.Version)
	}
	if err := validateHostID(r.MessageID, "message_id"); err != nil {
		return err
	}
	if r.Kind != HostEnvelopeKindCommandResponse {
		return fmt.Errorf("%s: invalid envelope kind", HostReasonInvalidEnvelope)
	}
	if r.Time.IsZero() {
		return fmt.Errorf("%s: timestamp is required", HostReasonInvalidEnvelope)
	}
	return validateHostID(r.RequestID, "request_id")
}

type HostRuntimeEventEnvelope struct {
	Version           string           `json:"version"`
	MessageID         string           `json:"message_id"`
	Kind              HostEnvelopeKind `json:"kind"`
	Time              time.Time        `json:"time"`
	Event             EventEnvelope    `json:"event"`
	RequestID         string           `json:"request_id,omitempty"`
	CausationID       string           `json:"causation_id,omitempty"`
	SourceCorrelation string           `json:"source_correlation,omitempty"`
}

func (e HostRuntimeEventEnvelope) Validate() error {
	if e.Version != HostProtocolVersionV1 {
		return fmt.Errorf("%s: %s", HostReasonUnsupportedVersion, e.Version)
	}
	if err := validateHostID(e.MessageID, "message_id"); err != nil {
		return err
	}
	if e.Kind != HostEnvelopeKindRuntimeEvent {
		return fmt.Errorf("%s: invalid envelope kind", HostReasonInvalidEnvelope)
	}
	if e.Time.IsZero() {
		return fmt.Errorf("%s: timestamp is required", HostReasonInvalidEnvelope)
	}
	return e.Event.ValidateProtocolReference()
}

type HostRequestEnvelope struct {
	Version     string           `json:"version"`
	MessageID   string           `json:"message_id"`
	Kind        HostEnvelopeKind `json:"kind"`
	Time        time.Time        `json:"time"`
	RequestID   string           `json:"request_id"`
	SessionID   string           `json:"session_id,omitempty"`
	RunID       string           `json:"run_id,omitempty"`
	RequestType string           `json:"request_type"`
	Payload     map[string]any   `json:"payload,omitempty"`
}

func (r HostRequestEnvelope) Validate() error {
	if r.Version != HostProtocolVersionV1 {
		return fmt.Errorf("%s: %s", HostReasonUnsupportedVersion, r.Version)
	}
	if err := validateHostID(r.MessageID, "message_id"); err != nil {
		return err
	}
	if r.Kind != HostEnvelopeKindHostRequest {
		return fmt.Errorf("%s: invalid envelope kind", HostReasonInvalidEnvelope)
	}
	if r.Time.IsZero() {
		return fmt.Errorf("%s: timestamp is required", HostReasonInvalidEnvelope)
	}
	return validateHostID(r.RequestID, "request_id")
}

type HostResponseEnvelope struct {
	Version   string           `json:"version"`
	MessageID string           `json:"message_id"`
	Kind      HostEnvelopeKind `json:"kind"`
	Time      time.Time        `json:"time"`
	RequestID string           `json:"request_id"`
	SessionID string           `json:"session_id,omitempty"`
	RunID     string           `json:"run_id,omitempty"`
	Payload   map[string]any   `json:"payload,omitempty"`
	Accepted  bool             `json:"accepted"`
}

func (r HostResponseEnvelope) Validate() error {
	if r.Version != HostProtocolVersionV1 {
		return fmt.Errorf("%s: %s", HostReasonUnsupportedVersion, r.Version)
	}
	if err := validateHostID(r.MessageID, "message_id"); err != nil {
		return err
	}
	if r.Kind != HostEnvelopeKindHostResponse {
		return fmt.Errorf("%s: invalid envelope kind", HostReasonInvalidEnvelope)
	}
	if r.Time.IsZero() {
		return fmt.Errorf("%s: timestamp is required", HostReasonInvalidEnvelope)
	}
	return validateHostID(r.RequestID, "request_id")
}

type HostPendingOutcome string

const (
	HostPendingCompleted    HostPendingOutcome = "completed"
	HostPendingRejected     HostPendingOutcome = "rejected"
	HostPendingTimedOut     HostPendingOutcome = "timed_out"
	HostPendingDisconnected HostPendingOutcome = "disconnected"
)

type HostPendingResult struct {
	Outcome    HostPendingOutcome `json:"outcome"`
	ReasonCode string             `json:"reason_code,omitempty"`
}

func NormalizeHostCommandAdmission(c HostCommandEnvelope, status HostAdmissionStatus, reason string) (HostCommandResponse, error) {
	if err := c.Validate(); err != nil {
		return HostCommandResponse{}, err
	}
	if status != HostAdmissionStatusAccepted && status != HostAdmissionStatusRejected && status != HostAdmissionStatusDuplicate {
		return HostCommandResponse{}, fmt.Errorf("%s: invalid status %q", HostReasonInvalidEnvelope, status)
	}
	if status != HostAdmissionStatusAccepted && strings.TrimSpace(reason) == "" {
		reason = HostReasonInvalidEnvelope
	}
	if status == HostAdmissionStatusDuplicate && reason == "" {
		reason = HostReasonDuplicate
	}
	return HostCommandResponse{Version: c.Version, MessageID: c.MessageID, Kind: HostEnvelopeKindCommandResponse, Time: time.Now().UTC(), RequestID: c.RequestID, SessionID: c.SessionID, RunID: c.RunID, Status: status, ReasonCode: reason, SourceCorrelation: c.SourceCorrelation}, nil
}

type HostSourceOwnership struct {
	Descriptor       ProtocolDescriptor
	SessionID, RunID string
	Active           bool
}

func ValidateHostSourceOwnership(o HostSourceOwnership) error {
	if err := o.Descriptor.Validate(); err != nil {
		return err
	}
	if strings.TrimSpace(o.SessionID) == "" || strings.TrimSpace(o.RunID) == "" {
		return fmt.Errorf("%s: source correlation required", HostReasonInvalidEnvelope)
	}
	if !o.Active {
		return fmt.Errorf("%s: run inactive", HostReasonAlreadyTerminal)
	}
	return nil
}

type HostAuthorization struct {
	Descriptor                                                                       ProtocolDescriptor
	Action                                                                           ProtocolAction
	Ready, PolicyAllowed, SandboxAllowed, DurableBindingValid, TerminalRecoveryValid bool
}

func AuthorizeHostCommand(a HostAuthorization) error {
	if err := a.Descriptor.Validate(); err != nil {
		return err
	}
	if a.Action != "" {
		if err := ValidateProtocolAction(a.Descriptor, a.Action); err != nil {
			return err
		}
	}
	if !a.Ready || !a.PolicyAllowed || !a.SandboxAllowed || !a.DurableBindingValid || !a.TerminalRecoveryValid {
		return fmt.Errorf("%s", HostReasonUnauthorized)
	}
	return nil
}

func isHostCommandKind(k HostCommandKind) bool {
	switch k {
	case HostCommandKindRunStart, HostCommandKindAction, HostCommandKindRealtimeInterrupt, HostCommandKindRealtimeResume, HostCommandKindHITLRespond, HostCommandKindEventsSubscribe, HostCommandKindRunGet:
		return true
	}
	return false
}
func validateHostID(v, name string) error {
	s := strings.TrimSpace(v)
	if s == "" {
		return fmt.Errorf("%s: %s required", HostReasonInvalidEnvelope, name)
	}
	if len(s) > 256 {
		return fmt.Errorf("%s: %s too long", HostReasonInvalidEnvelope, name)
	}
	for _, r := range s {
		if unicode.IsSpace(r) || unicode.IsControl(r) {
			return fmt.Errorf("%s: invalid %s", HostReasonInvalidEnvelope, name)
		}
	}
	return nil
}
