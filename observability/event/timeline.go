package event

import (
	"strings"
	"time"

	"github.com/FelixSeptem/baymax/core/types"
)

// ParseActionTimeline converts a generic event payload into the normalized timeline DTO.
func ParseActionTimeline(ev types.Event) (types.ActionTimelineEvent, bool) {
	if ev.Type != types.EventTypeActionTimeline {
		return types.ActionTimelineEvent{}, false
	}
	phase := normalizeActionPhase(payloadString(ev.Payload, "phase"))
	status := normalizeActionStatus(payloadString(ev.Payload, "status"))
	seq := payloadInt64(ev.Payload, "sequence")
	if seq <= 0 {
		return types.ActionTimelineEvent{}, false
	}
	if ev.Time.IsZero() {
		ev.Time = time.Now()
	}
	out := types.ActionTimelineEvent{
		RunID:     strings.TrimSpace(ev.RunID),
		Iteration: ev.Iteration,
		Phase:     phase,
		Status:    status,
		Reason:    strings.TrimSpace(payloadString(ev.Payload, "reason")),
		Sequence:  seq,
		Time:      ev.Time,
	}
	if out.RunID == "" || out.Phase == "" || out.Status == "" {
		return types.ActionTimelineEvent{}, false
	}
	return out, true
}

func normalizeActionPhase(v string) types.ActionPhase {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case string(types.ActionPhaseRun):
		return types.ActionPhaseRun
	case string(types.ActionPhaseContextAssembler):
		return types.ActionPhaseContextAssembler
	case string(types.ActionPhaseModel):
		return types.ActionPhaseModel
	case string(types.ActionPhaseTool):
		return types.ActionPhaseTool
	case string(types.ActionPhaseMCP):
		return types.ActionPhaseMCP
	case string(types.ActionPhaseSkill):
		return types.ActionPhaseSkill
	default:
		return ""
	}
}

func normalizeActionStatus(v string) types.ActionStatus {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case string(types.ActionStatusPending):
		return types.ActionStatusPending
	case string(types.ActionStatusRunning):
		return types.ActionStatusRunning
	case string(types.ActionStatusSucceeded):
		return types.ActionStatusSucceeded
	case string(types.ActionStatusFailed):
		return types.ActionStatusFailed
	case string(types.ActionStatusSkipped):
		return types.ActionStatusSkipped
	case string(types.ActionStatusCanceled):
		return types.ActionStatusCanceled
	default:
		return ""
	}
}
