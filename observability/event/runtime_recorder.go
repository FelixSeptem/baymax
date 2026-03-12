package event

import (
	"context"
	"strings"
	"time"

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
		status := payloadString(ev.Payload, "status")
		if status == "" {
			if errorClass != "" {
				status = "failed"
			} else {
				status = "success"
			}
		}
		r.manager.RecordRun(runtimediag.RunRecord{
			Time:                 ev.Time,
			RunID:                ev.RunID,
			Status:               status,
			Iterations:           ev.Iteration,
			ToolCalls:            payloadInt(ev.Payload, "tool_calls"),
			LatencyMs:            payloadInt64(ev.Payload, "latency_ms"),
			ErrorClass:           errorClass,
			ModelProvider:        payloadString(ev.Payload, "model_provider"),
			FallbackUsed:         payloadBool(ev.Payload, "fallback_used"),
			FallbackInitial:      payloadString(ev.Payload, "fallback_initial"),
			FallbackPath:         payloadString(ev.Payload, "fallback_path"),
			RequiredCapabilities: payloadString(ev.Payload, "required_capabilities"),
			FallbackReason:       payloadString(ev.Payload, "fallback_reason"),
			PrefixHash:           payloadString(ev.Payload, "prefix_hash"),
			AssembleLatencyMs:    payloadInt64(ev.Payload, "assemble_latency_ms"),
			AssembleStatus:       payloadString(ev.Payload, "assemble_status"),
			GuardViolation:       payloadString(ev.Payload, "guard_violation"),
			AssembleStageStatus:  payloadString(ev.Payload, "assemble_stage_status"),
			Stage2SkipReason:     payloadString(ev.Payload, "stage2_skip_reason"),
			Stage1LatencyMs:      payloadInt64(ev.Payload, "stage1_latency_ms"),
			Stage2LatencyMs:      payloadInt64(ev.Payload, "stage2_latency_ms"),
			Stage2Provider:       payloadString(ev.Payload, "stage2_provider"),
			RecapStatus:          payloadString(ev.Payload, "recap_status"),
		})
	case "skill.discovered":
		r.manager.RecordSkill(runtimediag.SkillRecord{
			Time:      eventTime(ev.Time),
			RunID:     ev.RunID,
			Action:    "discover",
			Status:    "success",
			LatencyMs: payloadInt64(ev.Payload, "latency_ms"),
			Payload:   ev.Payload,
		})
	case "skill.warning":
		action := payloadString(ev.Payload, "action")
		if action == "" {
			action = "compile"
		}
		errorClass := payloadString(ev.Payload, "error_class")
		if errorClass == "" {
			errorClass = string(types.ErrSkill)
		}
		r.manager.RecordSkill(runtimediag.SkillRecord{
			Time:       eventTime(ev.Time),
			RunID:      ev.RunID,
			SkillName:  payloadString(ev.Payload, "name"),
			Action:     action,
			Status:     payloadOrDefault(ev.Payload, "status", "warning"),
			LatencyMs:  payloadInt64(ev.Payload, "latency_ms"),
			ErrorClass: errorClass,
			Payload:    ev.Payload,
		})
	case "skill.loaded":
		action := payloadString(ev.Payload, "action")
		if action == "" {
			action = "compile"
		}
		r.manager.RecordSkill(runtimediag.SkillRecord{
			Time:      eventTime(ev.Time),
			RunID:     ev.RunID,
			SkillName: payloadString(ev.Payload, "name"),
			Action:    action,
			Status:    payloadOrDefault(ev.Payload, "status", "success"),
			LatencyMs: payloadInt64(ev.Payload, "latency_ms"),
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

func payloadInt(m map[string]any, key string) int {
	return int(payloadInt64(m, key))
}

func payloadOrDefault(m map[string]any, key, fallback string) string {
	v := payloadString(m, key)
	if v == "" {
		return fallback
	}
	return v
}

func payloadBool(m map[string]any, key string) bool {
	if len(m) == 0 {
		return false
	}
	raw, ok := m[key]
	if !ok {
		return false
	}
	v, _ := raw.(bool)
	return v
}

func eventTime(ts time.Time) time.Time {
	if ts.IsZero() {
		return time.Now()
	}
	return ts
}

var _ types.EventHandler = (*RuntimeRecorder)(nil)
