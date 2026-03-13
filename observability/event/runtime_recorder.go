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
	payload := ev.Payload
	if len(payload) > 0 {
		payload = r.manager.RedactPayload(payload)
	}
	switch ev.Type {
	case "run.finished":
		errorClass := payloadString(payload, "error_class")
		status := payloadString(payload, "status")
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
			ToolCalls:            payloadInt(payload, "tool_calls"),
			LatencyMs:            payloadInt64(payload, "latency_ms"),
			ErrorClass:           errorClass,
			ModelProvider:        payloadString(payload, "model_provider"),
			FallbackUsed:         payloadBool(payload, "fallback_used"),
			FallbackInitial:      payloadString(payload, "fallback_initial"),
			FallbackPath:         payloadString(payload, "fallback_path"),
			RequiredCapabilities: payloadString(payload, "required_capabilities"),
			FallbackReason:       payloadString(payload, "fallback_reason"),
			PrefixHash:           payloadString(payload, "prefix_hash"),
			AssembleLatencyMs:    payloadInt64(payload, "assemble_latency_ms"),
			AssembleStatus:       payloadString(payload, "assemble_status"),
			GuardViolation:       payloadString(payload, "guard_violation"),
			AssembleStageStatus:  payloadString(payload, "assemble_stage_status"),
			Stage2SkipReason:     payloadString(payload, "stage2_skip_reason"),
			Stage1LatencyMs:      payloadInt64(payload, "stage1_latency_ms"),
			Stage2LatencyMs:      payloadInt64(payload, "stage2_latency_ms"),
			Stage2Provider:       payloadString(payload, "stage2_provider"),
			Stage2HitCount:       payloadInt(payload, "stage2_hit_count"),
			Stage2Source:         payloadString(payload, "stage2_source"),
			Stage2Reason:         payloadString(payload, "stage2_reason"),
			RecapStatus:          payloadString(payload, "recap_status"),
		})
	case "skill.discovered":
		r.manager.RecordSkill(runtimediag.SkillRecord{
			Time:      eventTime(ev.Time),
			RunID:     ev.RunID,
			Action:    "discover",
			Status:    "success",
			LatencyMs: payloadInt64(payload, "latency_ms"),
			Payload:   payload,
		})
	case "skill.warning":
		action := payloadString(payload, "action")
		if action == "" {
			action = "compile"
		}
		errorClass := payloadString(payload, "error_class")
		if errorClass == "" {
			errorClass = string(types.ErrSkill)
		}
		r.manager.RecordSkill(runtimediag.SkillRecord{
			Time:       eventTime(ev.Time),
			RunID:      ev.RunID,
			SkillName:  payloadString(payload, "name"),
			Action:     action,
			Status:     payloadOrDefault(payload, "status", "warning"),
			LatencyMs:  payloadInt64(payload, "latency_ms"),
			ErrorClass: errorClass,
			Payload:    payload,
		})
	case "skill.loaded":
		action := payloadString(payload, "action")
		if action == "" {
			action = "compile"
		}
		r.manager.RecordSkill(runtimediag.SkillRecord{
			Time:      eventTime(ev.Time),
			RunID:     ev.RunID,
			SkillName: payloadString(payload, "name"),
			Action:    action,
			Status:    payloadOrDefault(payload, "status", "success"),
			LatencyMs: payloadInt64(payload, "latency_ms"),
			Payload:   payload,
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
