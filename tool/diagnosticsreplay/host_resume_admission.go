package diagnosticsreplay

import (
	"bytes"
	"encoding/json"
	"io"
	"unicode"
	"unicode/utf8"
)

const HostResumeAdmissionReplayVersion = "host_resume_admission.v1"
const ReasonHostResumeReplayDrift = "host_resume.replay_drift"

// HostResumeAdmissionFixture records bounded synthetic facts only. Replay has
// no verifier, runner, store, provider, tool, execution or effect handle.
type HostResumeAdmissionFixture struct {
	Version string                          `json:"version"`
	Cases   []HostResumeAdmissionReplayCase `json:"cases"`
}

type HostResumeAdmissionReplayCase struct {
	CaseID          string `json:"case_id"`
	ParityGroup     string `json:"parity_group,omitempty"`
	Mode            string `json:"mode"`
	RunID           string `json:"run_id"`
	SessionID       string `json:"session_id"`
	CheckpointID    string `json:"checkpoint_id"`
	Decision        string `json:"decision"`
	Status          string `json:"status"`
	SourceReason    string `json:"source_reason,omitempty"`
	BindingMatches  bool   `json:"binding_matches"`
	AuthorityValid  bool   `json:"authority_valid"`
	RevisionMatches bool   `json:"revision_matches"`
	Resolutions     int    `json:"resolutions"`
	TerminalEvents  int    `json:"terminal_events"`
	TerminalState   string `json:"terminal_state,omitempty"`
	ProviderCalls   int    `json:"provider_calls"`
	ToolCalls       int    `json:"tool_calls"`
	Effects         int    `json:"effects"`
	IterationBefore int    `json:"iteration_before"`
	IterationAfter  int    `json:"iteration_after"`
	ExpectedReason  string `json:"expected_reason,omitempty"`
}

type HostResumeAdmissionReplayResult struct {
	CaseID string `json:"case_id"`
	Reason string `json:"reason,omitempty"`
}

func hostResumeReplayError() error {
	return &ValidationError{Code: ReasonHostResumeReplayDrift, Message: "host resume admission conformance failed"}
}

func ReplayHostResumeAdmissionFixtureJSON(raw []byte) ([]HostResumeAdmissionReplayResult, error) {
	if len(raw) == 0 || len(raw) > 256*1024 || !utf8.Valid(raw) {
		return nil, hostResumeReplayError()
	}
	d := json.NewDecoder(bytes.NewReader(raw))
	d.DisallowUnknownFields()
	var f HostResumeAdmissionFixture
	if d.Decode(&f) != nil || d.Decode(new(any)) != io.EOF || f.Version != HostResumeAdmissionReplayVersion || len(f.Cases) == 0 || len(f.Cases) > 256 {
		return nil, hostResumeReplayError()
	}
	seen := map[string]bool{}
	parity := map[string]HostResumeAdmissionReplayCase{}
	modes := map[string]map[string]bool{}
	results := make([]HostResumeAdmissionReplayResult, 0, len(f.Cases))
	for _, x := range f.Cases {
		for _, s := range []string{x.CaseID, x.RunID, x.SessionID, x.CheckpointID} {
			if !hostResumeReplayRef(s) {
				return nil, hostResumeReplayError()
			}
		}
		if seen[x.CaseID] || (x.Mode != "run" && x.Mode != "stream") || (x.Decision != "confirm" && x.Decision != "deny" && x.Decision != "timeout") || x.IterationBefore < 0 || x.IterationAfter < x.IterationBefore {
			return nil, hostResumeReplayError()
		}
		seen[x.CaseID] = true
		for _, n := range []int{x.Resolutions, x.TerminalEvents, x.ProviderCalls, x.ToolCalls, x.Effects} {
			if n < 0 || n > 1024 {
				return nil, hostResumeReplayError()
			}
		}
		reason, ok := hostResumeReplayReason(x)
		if !ok || reason != x.ExpectedReason {
			return nil, hostResumeReplayError()
		}
		if x.ParityGroup != "" {
			if !hostResumeReplayRef(x.ParityGroup) {
				return nil, hostResumeReplayError()
			}
			// Correlation references may differ across recordings; execution
			// semantics, counts and classification must remain identical.
			normal := x
			normal.CaseID = ""
			normal.Mode = ""
			normal.RunID = ""
			normal.SessionID = ""
			normal.CheckpointID = ""
			if prior, ok := parity[x.ParityGroup]; ok && prior != normal {
				return nil, hostResumeReplayError()
			}
			parity[x.ParityGroup] = normal
			if modes[x.ParityGroup] == nil {
				modes[x.ParityGroup] = map[string]bool{}
			}
			if modes[x.ParityGroup][x.Mode] {
				return nil, hostResumeReplayError()
			}
			modes[x.ParityGroup][x.Mode] = true
		}
		results = append(results, HostResumeAdmissionReplayResult{CaseID: x.CaseID, Reason: reason})
	}
	for _, group := range modes {
		if !group["run"] || !group["stream"] {
			return nil, hostResumeReplayError()
		}
	}
	return results, nil
}

func hostResumeReplayRef(s string) bool {
	if s == "" || len(s) > 256 || !utf8.ValidString(s) {
		return false
	}
	for _, r := range s {
		if unicode.IsSpace(r) || unicode.IsControl(r) {
			return false
		}
	}
	return true
}

func hostResumeReplayReason(x HostResumeAdmissionReplayCase) (string, bool) {
	reason := x.SourceReason
	switch {
	case reason != "":
		switch reason {
		case "source.stale", "source.expired", "source.terminal", "source.conflict", "source.unknown":
		default:
			return "", false
		}
	case !x.BindingMatches:
		reason = "host_resume.binding_mismatch"
	case !x.AuthorityValid:
		reason = "host_resume.authority_denied"
	case x.Decision == "confirm" && !x.RevisionMatches:
		reason = "host_resume.revision_drift"
	}
	switch x.Status {
	case "accepted":
		if reason != "" || x.Resolutions != 1 || x.TerminalEvents != 1 {
			return "", false
		}
		if x.Decision == "confirm" {
			if x.TerminalState != "completed" || x.ProviderCalls != 1 || x.ToolCalls != 0 || x.Effects > 1 || x.IterationAfter != x.IterationBefore+1 {
				return "", false
			}
		} else {
			if x.TerminalState != "canceled" || x.ProviderCalls != 0 || x.ToolCalls != 0 || x.Effects != 0 || x.IterationAfter != x.IterationBefore {
				return "", false
			}
		}
	case "rejected", "duplicate", "in_progress":
		if x.Resolutions != 0 || x.TerminalEvents != 0 || x.ProviderCalls != 0 || x.ToolCalls != 0 || x.Effects != 0 || x.IterationBefore != x.IterationAfter || x.TerminalState != "" {
			return "", false
		}
		if x.Status == "rejected" {
			if reason == "" {
				return "", false
			}
		} else {
			if x.SourceReason != "" {
				return "", false
			}
			reason = ""
			if x.Status == "in_progress" {
				reason = "source.in_progress"
			}
		}
	default:
		return "", false
	}
	return reason, true
}
