package event

import (
	"context"
	"strings"

	"github.com/FelixSeptem/baymax/core/types"
	runtimeconfig "github.com/FelixSeptem/baymax/runtime/config"
	runtimediag "github.com/FelixSeptem/baymax/runtime/diagnostics"
)

type RuntimeRecorder struct {
	manager *runtimeconfig.Manager
}

func NewRuntimeRecorder(manager *runtimeconfig.Manager) *RuntimeRecorder {
	return &RuntimeRecorder{manager: manager}
}

func (r *RuntimeRecorder) OnEvent(_ context.Context, ev types.Event) {
	if r == nil || r.manager == nil {
		return
	}
	switch ev.Type {
	case "run.finished":
		errorClass := payloadString(ev.Payload, "error_class")
		r.manager.RecordRun(runtimediag.RunRecord{
			Time:       ev.Time,
			RunID:      ev.RunID,
			Iterations: ev.Iteration,
			LatencyMs:  payloadInt64(ev.Payload, "latency_ms"),
			ErrorClass: errorClass,
		})
	case "skill.warning":
		r.manager.RecordSkill(runtimediag.SkillRecord{
			Time:       ev.Time,
			RunID:      ev.RunID,
			SkillName:  payloadString(ev.Payload, "name"),
			Action:     "compile",
			Status:     "failed",
			ErrorClass: string(types.ErrSkill),
			Payload:    ev.Payload,
		})
	case "skill.loaded":
		r.manager.RecordSkill(runtimediag.SkillRecord{
			Time:      ev.Time,
			RunID:     ev.RunID,
			SkillName: payloadString(ev.Payload, "name"),
			Action:    "compile",
			Status:    "success",
			Payload:   ev.Payload,
		})
	}
}

func payloadString(m map[string]any, key string) string {
	if len(m) == 0 {
		return ""
	}
	raw, ok := m[key]
	if !ok {
		return ""
	}
	s, _ := raw.(string)
	return strings.TrimSpace(s)
}

func payloadInt64(m map[string]any, key string) int64 {
	if len(m) == 0 {
		return 0
	}
	raw, ok := m[key]
	if !ok {
		return 0
	}
	switch tv := raw.(type) {
	case int64:
		return tv
	case int:
		return int64(tv)
	case float64:
		return int64(tv)
	default:
		return 0
	}
}

var _ types.EventHandler = (*RuntimeRecorder)(nil)
