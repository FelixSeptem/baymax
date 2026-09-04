package diagnosticsreplay

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/FelixSeptem/baymax/core/types"
)

const SessionHistoryReplayFixtureV1 = types.SessionHistoryCheckpointReplayVersionV1

const (
	ReasonCodeSessionHistoryGap            = types.SessionHistoryReasonGap
	ReasonCodeSessionHistoryConflict       = types.SessionHistoryReasonConflict
	ReasonCodeSessionBranchDrift           = "session.branch_drift"
	ReasonCodeSessionCheckpointAssociation = types.SessionHistoryReasonCheckpointAssociation
	ReasonCodeSessionReplaySideEffect      = types.SessionHistoryReasonReplaySideEffect
	ReasonCodeSessionReplayConflict        = types.SessionHistoryReasonReplayConflict
	ReasonCodeRunStreamHistoryReplayParity = "run_stream_history_replay_parity_drift"
)

type SessionHistoryFixture struct {
	Version string                      `json:"version"`
	Cases   []SessionHistoryFixtureCase `json:"cases"`
}

type SessionHistoryFixtureCase struct {
	Name        string                    `json:"name"`
	Run         SessionHistoryObservation `json:"run"`
	Stream      SessionHistoryObservation `json:"stream"`
	Expected    SessionHistoryObservation `json:"expected"`
	Idempotency types.ReplayOperation     `json:"idempotency"`
}

type SessionHistoryObservation struct {
	SessionID        string                       `json:"session_id"`
	RunID            string                       `json:"run_id"`
	History          types.SessionHistoryBoundary `json:"history"`
	Branch           *types.BranchProjection      `json:"branch,omitempty"`
	Restore          *types.RestoreOperation      `json:"restore,omitempty"`
	ReplaySideEffect bool                         `json:"replay_side_effect,omitempty"`
	NormalizedDigest string                       `json:"normalized_digest,omitempty"`
}

type SessionHistoryReplayOutput struct {
	Version string                           `json:"version"`
	Cases   []SessionHistoryNormalizedOutput `json:"cases"`
}

type SessionHistoryNormalizedOutput struct {
	Name             string `json:"name"`
	SessionID        string `json:"session_id"`
	RunID            string `json:"run_id"`
	HistoryDigest    string `json:"history_digest"`
	BranchID         string `json:"branch_id,omitempty"`
	RestoreOperation string `json:"restore_operation,omitempty"`
}

func ParseSessionHistoryFixtureJSON(raw []byte) (SessionHistoryFixture, error) {
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	var fixture SessionHistoryFixture
	if err := dec.Decode(&fixture); err != nil {
		return SessionHistoryFixture{}, &ValidationError{Code: ReasonCodeInvalidJSON, Message: err.Error()}
	}
	if strings.TrimSpace(fixture.Version) != SessionHistoryReplayFixtureV1 {
		return SessionHistoryFixture{}, &ValidationError{Code: ReasonCodeProtocolSchema, Message: fmt.Sprintf("unsupported fixture version %q", fixture.Version)}
	}
	if len(fixture.Cases) == 0 {
		return SessionHistoryFixture{}, &ValidationError{Code: ReasonCodeProtocolSchema, Message: "cases must not be empty"}
	}
	seen := map[string]struct{}{}
	for i := range fixture.Cases {
		fixture.Cases[i].Name = strings.TrimSpace(fixture.Cases[i].Name)
		if fixture.Cases[i].Name == "" {
			return SessionHistoryFixture{}, &ValidationError{Code: ReasonCodeProtocolSchema, Message: fmt.Sprintf("cases[%d].name is required", i)}
		}
		if _, ok := seen[fixture.Cases[i].Name]; ok {
			return SessionHistoryFixture{}, &ValidationError{Code: ReasonCodeProtocolSchema, Message: fmt.Sprintf("duplicate case %q", fixture.Cases[i].Name)}
		}
		seen[fixture.Cases[i].Name] = struct{}{}
	}
	return fixture, nil
}

func EvaluateSessionHistoryFixtureJSON(raw []byte) (SessionHistoryReplayOutput, error) {
	fixture, err := ParseSessionHistoryFixtureJSON(raw)
	if err != nil {
		return SessionHistoryReplayOutput{}, err
	}
	return EvaluateSessionHistoryFixture(fixture)
}

func EvaluateSessionHistoryFixture(fixture SessionHistoryFixture) (SessionHistoryReplayOutput, error) {
	out := SessionHistoryReplayOutput{Version: fixture.Version}
	for _, tc := range fixture.Cases {
		for label, obs := range map[string]SessionHistoryObservation{"run": tc.Run, "stream": tc.Stream, "expected": tc.Expected} {
			if err := validateSessionHistoryObservation(tc.Name, label, obs); err != nil {
				return SessionHistoryReplayOutput{}, err
			}
		}
		if tc.Run.SessionID != tc.Stream.SessionID || tc.Run.History.NormalizedDigest() != tc.Stream.History.NormalizedDigest() || branchID(tc.Run.Branch) != branchID(tc.Stream.Branch) {
			return SessionHistoryReplayOutput{}, &ValidationError{Code: ReasonCodeRunStreamHistoryReplayParity, Message: fmt.Sprintf("case %q run/stream history replay parity drift", tc.Name)}
		}
		if tc.Expected.SessionID != tc.Run.SessionID || tc.Expected.History.NormalizedDigest() != tc.Run.History.NormalizedDigest() || branchID(tc.Expected.Branch) != branchID(tc.Run.Branch) {
			return SessionHistoryReplayOutput{}, &ValidationError{Code: ReasonCodeRunStreamHistoryReplayParity, Message: fmt.Sprintf("case %q expected history replay drift", tc.Name)}
		}
		if !tc.Idempotency.SideEffectFree {
			return SessionHistoryReplayOutput{}, &ValidationError{Code: ReasonCodeSessionReplaySideEffect, Message: fmt.Sprintf("case %q replay must be side-effect-free", tc.Name)}
		}
		out.Cases = append(out.Cases, SessionHistoryNormalizedOutput{
			Name: tc.Name, SessionID: tc.Expected.SessionID, RunID: tc.Expected.RunID,
			HistoryDigest: tc.Expected.History.NormalizedDigest(), BranchID: branchID(tc.Expected.Branch), RestoreOperation: restoreID(tc.Expected.Restore),
		})
	}
	return out, nil
}

func validateSessionHistoryObservation(name, label string, obs SessionHistoryObservation) error {
	if strings.TrimSpace(obs.SessionID) == "" || strings.TrimSpace(obs.RunID) == "" {
		return &ValidationError{Code: ReasonCodeProtocolSchema, Message: fmt.Sprintf("case %q %s session_id/run_id required", name, label)}
	}
	if obs.History.SessionID != obs.SessionID {
		return &ValidationError{Code: ReasonCodeSessionCheckpointAssociation, Message: fmt.Sprintf("case %q %s history session mismatch", name, label)}
	}
	if err := obs.History.Validate(types.HistoryValidationLimits{MaxEntries: 256, MaxSerializedBytes: 1 << 20}); err != nil {
		code := ReasonCodeSessionHistoryGap
		if strings.Contains(err.Error(), types.SessionHistoryReasonConflict) {
			code = ReasonCodeSessionHistoryConflict
		}
		return &ValidationError{Code: code, Message: fmt.Sprintf("case %q %s: %v", name, label, err)}
	}
	if obs.Branch != nil {
		if err := obs.Branch.Validate(); err != nil {
			return &ValidationError{Code: ReasonCodeSessionBranchDrift, Message: fmt.Sprintf("case %q %s: %v", name, label, err)}
		}
		if obs.Branch.SessionID != obs.SessionID || obs.Branch.RunID != obs.RunID {
			return &ValidationError{Code: ReasonCodeSessionCheckpointAssociation, Message: fmt.Sprintf("case %q %s branch association mismatch", name, label)}
		}
	}
	if obs.Restore != nil {
		if err := obs.Restore.Validate(); err != nil {
			return &ValidationError{Code: ReasonCodeSessionReplayConflict, Message: fmt.Sprintf("case %q %s: %v", name, label, err)}
		}
		if obs.Restore.SessionID != "" && obs.Restore.SessionID != obs.SessionID {
			return &ValidationError{Code: ReasonCodeSessionCheckpointAssociation, Message: fmt.Sprintf("case %q %s restore session mismatch", name, label)}
		}
	}
	if obs.ReplaySideEffect {
		return &ValidationError{Code: ReasonCodeSessionReplaySideEffect, Message: fmt.Sprintf("case %q %s side effect requested", name, label)}
	}
	return nil
}

func branchID(branch *types.BranchProjection) string {
	if branch == nil {
		return ""
	}
	return branch.BranchID
}
func restoreID(restore *types.RestoreOperation) string {
	if restore == nil {
		return ""
	}
	return restore.OperationID
}
