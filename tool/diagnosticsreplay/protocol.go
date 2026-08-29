package diagnosticsreplay

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

const (
	AgentRuntimeProtocolFixtureV1                     = "agent_runtime_protocol.v1"
	AgentRuntimeProtocolCheckpointProvenanceFixtureV1 = "agent_runtime_protocol_checkpoint_provenance.v1"
	ReasonCodeProtocolDrift                           = "agent_runtime_protocol_drift"
	ReasonCodeProtocolSchema                          = "agent_runtime_protocol_schema_mismatch"
	ReasonCodeProtocolLineage                         = "agent_runtime_protocol_lineage_drift"
	ReasonCodeProtocolProfileDrift                    = "agent_runtime_protocol_profile_drift"
	ReasonCodeProtocolCapabilityDrift                 = "agent_runtime_protocol_capability_drift"
	ReasonCodeProtocolContextDrift                    = "agent_runtime_protocol_context_limit_drift"
	ReasonCodeProtocolAdmissionDrift                  = "agent_runtime_protocol_admission_drift"
	ReasonCodeProtocolCorrelationDrift                = "agent_runtime_protocol_correlation_drift"
	ReasonCodeProtocolStreamBindingDrift              = "agent_runtime_protocol_stream_binding_drift"
	ReasonCodeCheckpointLineageDrift                  = "checkpoint_lineage_drift"
	ReasonCodeCheckpointSchemaDrift                   = "checkpoint_schema_drift"
	ReasonCodeCheckpointBranchDrift                   = "checkpoint_branch_drift"
	ReasonCodeCheckpointReplayDrift                   = "checkpoint_replay_drift"
	ReasonCodeWorkspaceProvenanceDrift                = "workspace_provenance_drift"
	ReasonCodeWorkspaceIntegrityDrift                 = "workspace_integrity_drift"
	ReasonCodeRunStreamProvenanceParityDrift          = "run_stream_provenance_parity_drift"
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
	RunID                string                                   `json:"run_id"`
	SessionID            string                                   `json:"session_id,omitempty"`
	State                string                                   `json:"state"`
	StepIDs              []string                                 `json:"step_ids,omitempty"`
	EventIDs             []string                                 `json:"event_ids,omitempty"`
	ArtifactIDs          []string                                 `json:"artifact_ids,omitempty"`
	CheckpointID         string                                   `json:"checkpoint_id,omitempty"`
	EventSequence        []int64                                  `json:"event_sequence,omitempty"`
	Descriptor           *ProtocolDescriptorObservation           `json:"descriptor,omitempty"`
	Context              *ProtocolContextObservation              `json:"context,omitempty"`
	Admission            *ProtocolAdmissionObservation            `json:"admission,omitempty"`
	StreamBinding        *ProtocolStreamBindingObservation        `json:"stream_binding,omitempty"`
	CheckpointProvenance *ProtocolCheckpointProvenanceObservation `json:"checkpoint_provenance,omitempty"`
}

type ProtocolCheckpointProvenanceObservation struct {
	SchemaVersion      string `json:"schema_version,omitempty"`
	Relation           string `json:"relation,omitempty"`
	ParentCheckpointID string `json:"parent_checkpoint_id,omitempty"`
	BranchID           string `json:"branch_id,omitempty"`
	HistoryIndex       int    `json:"history_index,omitempty"`
	RestoreSource      string `json:"restore_source,omitempty"`
	ReplayKey          string `json:"replay_key,omitempty"`
	WorkspaceID        string `json:"workspace_id,omitempty"`
	ChangeSetID        string `json:"change_set_id,omitempty"`
	BeforeIntegrity    string `json:"before_integrity,omitempty"`
	AfterIntegrity     string `json:"after_integrity,omitempty"`
}

type ProtocolStreamBindingObservation struct {
	SubscriptionID   string `json:"subscription_id,omitempty"`
	Phase            string `json:"phase,omitempty"`
	ReasonCode       string `json:"reason_code,omitempty"`
	CursorMode       string `json:"cursor_mode,omitempty"`
	SequenceBoundary int64  `json:"sequence_boundary,omitempty"`
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
	if strings.TrimSpace(fixture.Version) != AgentRuntimeProtocolFixtureV1 && strings.TrimSpace(fixture.Version) != AgentRuntimeProtocolCheckpointProvenanceFixtureV1 {
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
	fixture.Version = strings.TrimSpace(fixture.Version)
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
	out := ProtocolReplayOutput{Version: fixture.Version, Cases: make([]ProtocolNormalizedOutput, 0, len(cases))}
	for _, tc := range cases {
		for label, obs := range map[string]ProtocolObservation{"expected": tc.Expected, "run": tc.Run, "stream": tc.Stream} {
			if err := validateProtocolObservation(obs, tc.Name, label); err != nil {
				return ProtocolReplayOutput{}, err
			}
		}
		if !protocolCheckpointProvenanceEqual(tc.Run.CheckpointProvenance, tc.Stream.CheckpointProvenance) {
			return ProtocolReplayOutput{}, &ValidationError{Code: ReasonCodeRunStreamProvenanceParityDrift, Message: fmt.Sprintf("case %q run/stream provenance parity drift", tc.Name)}
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
		case !protocolStreamBindingEqual(expected.StreamBinding, actual.StreamBinding):
			code = ReasonCodeProtocolStreamBindingDrift
		case !protocolCheckpointProvenanceEqual(expected.CheckpointProvenance, actual.CheckpointProvenance):
			code = checkpointProvenanceDriftCode(expected.CheckpointProvenance, actual.CheckpointProvenance)
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
	if obs.StreamBinding != nil {
		if strings.TrimSpace(obs.StreamBinding.SubscriptionID) == "" || strings.TrimSpace(obs.StreamBinding.Phase) == "" || strings.TrimSpace(obs.StreamBinding.ReasonCode) == "" || strings.TrimSpace(obs.StreamBinding.CursorMode) == "" {
			return &ValidationError{Code: ReasonCodeProtocolSchema, Message: fmt.Sprintf("case %q %s stream_binding fields are required", name, label)}
		}
		if obs.StreamBinding.SequenceBoundary < 0 {
			return &ValidationError{Code: ReasonCodeProtocolSchema, Message: fmt.Sprintf("case %q %s stream_binding sequence boundary must not be negative", name, label)}
		}
	}
	if obs.CheckpointProvenance != nil {
		p := obs.CheckpointProvenance
		if strings.TrimSpace(p.Relation) == "" || strings.TrimSpace(p.RestoreSource) == "" || strings.TrimSpace(p.WorkspaceID) == "" || strings.TrimSpace(p.ChangeSetID) == "" {
			return &ValidationError{Code: ReasonCodeProtocolSchema, Message: fmt.Sprintf("case %q %s checkpoint_provenance fields are required", name, label)}
		}
		if p.HistoryIndex < 0 {
			return &ValidationError{Code: ReasonCodeCheckpointLineageDrift, Message: fmt.Sprintf("case %q %s checkpoint history index must not be negative", name, label)}
		}
	}
	return nil
}

func protocolObservationEqual(a, b ProtocolObservation) bool {
	return a.RunID == b.RunID && a.SessionID == b.SessionID && a.State == b.State && stringSliceEqual(a.StepIDs, b.StepIDs) && stringSliceEqual(a.EventIDs, b.EventIDs) && stringSliceEqual(a.ArtifactIDs, b.ArtifactIDs) && a.CheckpointID == b.CheckpointID && int64SliceEqual(a.EventSequence, b.EventSequence) && protocolDescriptorEqual(a.Descriptor, b.Descriptor) && protocolContextEqual(a.Context, b.Context) && protocolAdmissionEqual(a.Admission, b.Admission) && protocolStreamBindingEqual(a.StreamBinding, b.StreamBinding) && protocolCheckpointProvenanceEqual(a.CheckpointProvenance, b.CheckpointProvenance)
}

func protocolCheckpointProvenanceEqual(a, b *ProtocolCheckpointProvenanceObservation) bool {
	if a == nil || b == nil {
		return a == b
	}
	return a.SchemaVersion == b.SchemaVersion && a.Relation == b.Relation && a.ParentCheckpointID == b.ParentCheckpointID && a.BranchID == b.BranchID && a.HistoryIndex == b.HistoryIndex && a.RestoreSource == b.RestoreSource && a.ReplayKey == b.ReplayKey && a.WorkspaceID == b.WorkspaceID && a.ChangeSetID == b.ChangeSetID && a.BeforeIntegrity == b.BeforeIntegrity && a.AfterIntegrity == b.AfterIntegrity
}

func checkpointProvenanceDriftCode(expected, actual *ProtocolCheckpointProvenanceObservation) string {
	if expected == nil || actual == nil {
		return ReasonCodeWorkspaceProvenanceDrift
	}
	switch {
	case expected.SchemaVersion != actual.SchemaVersion:
		return ReasonCodeCheckpointSchemaDrift
	case expected.Relation != actual.Relation || expected.ParentCheckpointID != actual.ParentCheckpointID || expected.HistoryIndex != actual.HistoryIndex || expected.RestoreSource != actual.RestoreSource:
		return ReasonCodeCheckpointLineageDrift
	case expected.BranchID != actual.BranchID:
		return ReasonCodeCheckpointBranchDrift
	case expected.ReplayKey != actual.ReplayKey:
		return ReasonCodeCheckpointReplayDrift
	case expected.BeforeIntegrity != actual.BeforeIntegrity || expected.AfterIntegrity != actual.AfterIntegrity:
		return ReasonCodeWorkspaceIntegrityDrift
	case expected.WorkspaceID != actual.WorkspaceID || expected.ChangeSetID != actual.ChangeSetID:
		return ReasonCodeWorkspaceProvenanceDrift
	default:
		return ReasonCodeWorkspaceProvenanceDrift
	}
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

func protocolStreamBindingEqual(a, b *ProtocolStreamBindingObservation) bool {
	if a == nil || b == nil {
		return a == b
	}
	return a.SubscriptionID == b.SubscriptionID && a.Phase == b.Phase && a.ReasonCode == b.ReasonCode && a.CursorMode == b.CursorMode && a.SequenceBoundary == b.SequenceBoundary
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
