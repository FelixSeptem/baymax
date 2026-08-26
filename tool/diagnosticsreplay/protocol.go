package diagnosticsreplay

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

const (
	AgentRuntimeProtocolFixtureV1      = "agent_runtime_protocol.v1"
	ReasonCodeProtocolDrift            = "agent_runtime_protocol_drift"
	ReasonCodeProtocolSchema           = "agent_runtime_protocol_schema_mismatch"
	ReasonCodeProtocolLineage          = "agent_runtime_protocol_lineage_drift"
	ReasonCodeProtocolProfileDrift     = "agent_runtime_protocol_profile_drift"
	ReasonCodeProtocolCapabilityDrift  = "agent_runtime_protocol_capability_drift"
	ReasonCodeProtocolContextDrift     = "agent_runtime_protocol_context_limit_drift"
	ReasonCodeProtocolAdmissionDrift   = "agent_runtime_protocol_admission_drift"
	ReasonCodeProtocolCorrelationDrift = "agent_runtime_protocol_correlation_drift"
)

type ProtocolFixture struct {
	Version string                `json:"version"`
	Cases   []ProtocolFixtureCase `json:"cases"`
}

type ProtocolFixtureCase struct {
	Name        string              `json:"name"`
	Run         ProtocolObservation `json:"run"`
	Stream      ProtocolObservation `json:"stream"`
	Expected    ProtocolObservation `json:"expected"`
	Idempotency ProtocolIdempotency `json:"idempotency"`
}

type ProtocolObservation struct {
	RunID         string                         `json:"run_id"`
	SessionID     string                         `json:"session_id,omitempty"`
	State         string                         `json:"state"`
	StepIDs       []string                       `json:"step_ids,omitempty"`
	EventIDs      []string                       `json:"event_ids,omitempty"`
	ArtifactIDs   []string                       `json:"artifact_ids,omitempty"`
	CheckpointID  string                         `json:"checkpoint_id,omitempty"`
	EventSequence []int64                        `json:"event_sequence,omitempty"`
	Descriptor    *ProtocolDescriptorObservation `json:"descriptor,omitempty"`
	Context       *ProtocolContextObservation    `json:"context,omitempty"`
	Admission     *ProtocolAdmissionObservation  `json:"admission,omitempty"`
}

type ProtocolDescriptorObservation struct {
	ProfileVersion     string   `json:"profile_version,omitempty"`
	CapabilityDecision string   `json:"capability_decision,omitempty"`
	CapabilityReason   string   `json:"capability_reason,omitempty"`
	SupportedActions   []string `json:"supported_actions,omitempty"`
}

type ProtocolContextObservation struct {
	Scope           string   `json:"scope,omitempty"`
	MetadataKeys    []string `json:"metadata_keys,omitempty"`
	SerializedBytes int      `json:"serialized_bytes,omitempty"`
}

type ProtocolAdmissionObservation struct {
	Policy            string   `json:"policy,omitempty"`
	Decision          string   `json:"decision,omitempty"`
	ReasonCode        string   `json:"reason_code,omitempty"`
	ConflictingRunIDs []string `json:"conflicting_run_ids,omitempty"`
	BranchRunID       string   `json:"branch_run_id,omitempty"`
}

type ProtocolIdempotency struct {
	FirstLogicalIngestTotal  int `json:"first_logical_ingest_total"`
	ReplayLogicalIngestTotal int `json:"replay_logical_ingest_total"`
}

type ProtocolReplayOutput struct {
	Version string                     `json:"version"`
	Cases   []ProtocolNormalizedOutput `json:"cases"`
}

type ProtocolNormalizedOutput struct {
	Name        string              `json:"name"`
	Canonical   ProtocolObservation `json:"canonical"`
	Idempotency ProtocolIdempotency `json:"idempotency"`
}

func ParseProtocolFixtureJSON(raw []byte) (ProtocolFixture, error) {
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	var fixture ProtocolFixture
	if err := dec.Decode(&fixture); err != nil {
		return ProtocolFixture{}, &ValidationError{Code: ReasonCodeInvalidJSON, Message: err.Error()}
	}
	if strings.TrimSpace(fixture.Version) != AgentRuntimeProtocolFixtureV1 {
		return ProtocolFixture{}, &ValidationError{Code: ReasonCodeProtocolSchema, Message: fmt.Sprintf("unsupported fixture version %q", fixture.Version)}
	}
	if len(fixture.Cases) == 0 {
		return ProtocolFixture{}, &ValidationError{Code: ReasonCodeProtocolSchema, Message: "cases must not be empty"}
	}
	seen := map[string]struct{}{}
	for i := range fixture.Cases {
		name := strings.TrimSpace(fixture.Cases[i].Name)
		if name == "" {
			return ProtocolFixture{}, &ValidationError{Code: ReasonCodeProtocolSchema, Message: fmt.Sprintf("cases[%d].name is required", i)}
		}
		if _, ok := seen[name]; ok {
			return ProtocolFixture{}, &ValidationError{Code: ReasonCodeProtocolSchema, Message: fmt.Sprintf("duplicate case %q", name)}
		}
		seen[name] = struct{}{}
		fixture.Cases[i].Name = name
	}
	fixture.Version = AgentRuntimeProtocolFixtureV1
	return fixture, nil
}

func EvaluateProtocolFixtureJSON(raw []byte) (ProtocolReplayOutput, error) {
	fixture, err := ParseProtocolFixtureJSON(raw)
	if err != nil {
		return ProtocolReplayOutput{}, err
	}
	return EvaluateProtocolFixture(fixture)
}

func EvaluateProtocolFixture(fixture ProtocolFixture) (ProtocolReplayOutput, error) {
	cases := append([]ProtocolFixtureCase(nil), fixture.Cases...)
	sort.Slice(cases, func(i, j int) bool { return cases[i].Name < cases[j].Name })
	out := ProtocolReplayOutput{Version: AgentRuntimeProtocolFixtureV1, Cases: make([]ProtocolNormalizedOutput, 0, len(cases))}
	for _, tc := range cases {
		for label, obs := range map[string]ProtocolObservation{"expected": tc.Expected, "run": tc.Run, "stream": tc.Stream} {
			if err := validateProtocolObservation(obs, tc.Name, label); err != nil {
				return ProtocolReplayOutput{}, err
			}
		}
		if err := protocolObservationDrift(tc.Name, tc.Expected, tc.Run); err != nil {
			return ProtocolReplayOutput{}, err
		}
		if err := protocolObservationDrift(tc.Name, tc.Expected, tc.Stream); err != nil {
			return ProtocolReplayOutput{}, err
		}
		if tc.Idempotency.FirstLogicalIngestTotal <= 0 || tc.Idempotency.FirstLogicalIngestTotal != tc.Idempotency.ReplayLogicalIngestTotal {
			return ProtocolReplayOutput{}, &ValidationError{Code: ReasonCodeProtocolDrift, Message: fmt.Sprintf("case %q idempotency drift", tc.Name)}
		}
		out.Cases = append(out.Cases, ProtocolNormalizedOutput{Name: tc.Name, Canonical: tc.Expected, Idempotency: tc.Idempotency})
	}
	return out, nil
}

func protocolObservationDrift(name string, expected, actual ProtocolObservation) error {
	if !protocolObservationEqual(expected, actual) {
		code := ReasonCodeProtocolDrift
		switch {
		case !protocolDescriptorEqual(expected.Descriptor, actual.Descriptor):
			code = ReasonCodeProtocolCapabilityDrift
			if expected.Descriptor != nil && actual.Descriptor != nil && expected.Descriptor.ProfileVersion != actual.Descriptor.ProfileVersion {
				code = ReasonCodeProtocolProfileDrift
			}
		case !protocolContextEqual(expected.Context, actual.Context):
			code = ReasonCodeProtocolContextDrift
		case !protocolAdmissionEqual(expected.Admission, actual.Admission):
			code = ReasonCodeProtocolAdmissionDrift
		case expected.RunID != actual.RunID || expected.SessionID != actual.SessionID:
			code = ReasonCodeProtocolCorrelationDrift
		}
		return &ValidationError{Code: code, Message: fmt.Sprintf("case %q run/stream mapping drift", name)}
	}
	return nil
}

func validateProtocolObservation(obs ProtocolObservation, name, label string) error {
	if strings.TrimSpace(obs.RunID) == "" || strings.TrimSpace(obs.State) == "" {
		return &ValidationError{Code: ReasonCodeProtocolSchema, Message: fmt.Sprintf("case %q %s run_id/state required", name, label)}
	}
	if len(obs.EventIDs) != len(obs.EventSequence) {
		return &ValidationError{Code: ReasonCodeProtocolSchema, Message: fmt.Sprintf("case %q %s event ids/sequence length mismatch", name, label)}
	}
	if len(obs.ArtifactIDs) > 0 && strings.TrimSpace(obs.CheckpointID) == "" {
		return &ValidationError{Code: ReasonCodeProtocolLineage, Message: fmt.Sprintf("case %q %s artifact lineage requires checkpoint", name, label)}
	}
	return nil
}

func protocolObservationEqual(a, b ProtocolObservation) bool {
	return a.RunID == b.RunID && a.SessionID == b.SessionID && a.State == b.State && stringSliceEqual(a.StepIDs, b.StepIDs) && stringSliceEqual(a.EventIDs, b.EventIDs) && stringSliceEqual(a.ArtifactIDs, b.ArtifactIDs) && a.CheckpointID == b.CheckpointID && int64SliceEqual(a.EventSequence, b.EventSequence) && protocolDescriptorEqual(a.Descriptor, b.Descriptor) && protocolContextEqual(a.Context, b.Context) && protocolAdmissionEqual(a.Admission, b.Admission)
}

func protocolDescriptorEqual(a, b *ProtocolDescriptorObservation) bool {
	if a == nil || b == nil {
		return a == b
	}
	return a.ProfileVersion == b.ProfileVersion && a.CapabilityDecision == b.CapabilityDecision && a.CapabilityReason == b.CapabilityReason && stringSliceEqual(a.SupportedActions, b.SupportedActions)
}

func protocolContextEqual(a, b *ProtocolContextObservation) bool {
	if a == nil || b == nil {
		return a == b
	}
	return a.Scope == b.Scope && a.SerializedBytes == b.SerializedBytes && stringSliceEqual(a.MetadataKeys, b.MetadataKeys)
}

func protocolAdmissionEqual(a, b *ProtocolAdmissionObservation) bool {
	if a == nil || b == nil {
		return a == b
	}
	return a.Policy == b.Policy && a.Decision == b.Decision && a.ReasonCode == b.ReasonCode && a.BranchRunID == b.BranchRunID && stringSliceEqual(a.ConflictingRunIDs, b.ConflictingRunIDs)
}

func stringSliceEqual(a, b []string) bool { return strings.Join(a, "\x00") == strings.Join(b, "\x00") }
func int64SliceEqual(a, b []int64) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
