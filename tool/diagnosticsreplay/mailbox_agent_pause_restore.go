package diagnosticsreplay

import (
	"bytes"
	"encoding/json"
	"io"
	"strings"
	"unicode/utf8"
)

const MailboxAgentPauseRestoreReplayVersion = "mailbox_agent_pause_restore.v1"

type MailboxAgentPauseRestoreFixture struct {
	Version string                               `json:"version"`
	Cases   []MailboxAgentPauseRestoreReplayCase `json:"cases"`
}

type MailboxAgentPauseRestoreReplayCase struct {
	CaseID                string `json:"case_id"`
	PauseReason           string `json:"pause_reason"`
	FreshProcessRestore   bool   `json:"fresh_process_restore"`
	CompletedEffectsRerun bool   `json:"completed_effects_rerun"`
	DuplicateResumeCount  int    `json:"duplicate_resume_count"`
	DeploymentScope       string `json:"deployment_scope"`
	TerminalOutcome       string `json:"terminal_outcome"`
	UnsupportedReason     string `json:"unsupported_reason,omitempty"`
}

type MailboxAgentPauseRestoreReplayResult struct {
	CaseID string `json:"case_id"`
	Reason string `json:"reason"`
}

func ReplayMailboxAgentPauseRestoreFixtureJSON(raw []byte) ([]MailboxAgentPauseRestoreReplayResult, error) {
	if len(raw) == 0 || len(raw) > 64*1024 || !utf8.Valid(raw) {
		return nil, mailboxPlanReplayError("mailbox_agent_pause_restore.replay_drift")
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	var fixture MailboxAgentPauseRestoreFixture
	if err := decoder.Decode(&fixture); err != nil {
		return nil, mailboxPlanReplayError("mailbox_agent_pause_restore.replay_drift")
	}
	if err := decoder.Decode(new(any)); err != io.EOF || fixture.Version != MailboxAgentPauseRestoreReplayVersion || len(fixture.Cases) == 0 || len(fixture.Cases) > 16 {
		return nil, mailboxPlanReplayError("mailbox_agent_pause_restore.replay_drift")
	}
	seen := map[string]struct{}{}
	results := make([]MailboxAgentPauseRestoreReplayResult, 0, len(fixture.Cases))
	for _, item := range fixture.Cases {
		if strings.TrimSpace(item.CaseID) == "" || len(item.CaseID) > 128 || item.DuplicateResumeCount < 1 || len(item.DeploymentScope) > 128 || len(item.TerminalOutcome) > 128 {
			return nil, mailboxPlanReplayError("mailbox_agent_pause_restore.replay_drift")
		}
		if _, ok := seen[item.CaseID]; ok {
			return nil, mailboxPlanReplayError("mailbox_agent_pause_restore.replay_drift")
		}
		seen[item.CaseID] = struct{}{}
		if item.UnsupportedReason == "" {
			if item.PauseReason != "input_required" && item.PauseReason != "approval_required" && item.PauseReason != "budget_exhausted" {
				return nil, mailboxPlanReplayError("mailbox_agent_pause_restore.replay_drift")
			}
			if !item.FreshProcessRestore || item.CompletedEffectsRerun || item.DeploymentScope != "same_host_single_writer_sequential_reopen" || item.TerminalOutcome != "acked" {
				return nil, mailboxPlanReplayError("mailbox_agent_pause_restore.replay_drift")
			}
			results = append(results, MailboxAgentPauseRestoreReplayResult{CaseID: item.CaseID})
			continue
		}
		if item.UnsupportedReason != "mailbox_handler_safe_point_unsupported" || item.PauseReason != "input_required" || item.FreshProcessRestore || item.CompletedEffectsRerun || item.TerminalOutcome != "unsupported" {
			return nil, mailboxPlanReplayError("mailbox_agent_pause_restore.replay_drift")
		}
		results = append(results, MailboxAgentPauseRestoreReplayResult{CaseID: item.CaseID, Reason: item.UnsupportedReason})
	}
	return results, nil
}
