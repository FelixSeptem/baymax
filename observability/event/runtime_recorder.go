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
	case types.EventTypeActionTimeline:
		if timeline, ok := ParseActionTimeline(types.Event{
			Version:   ev.Version,
			Type:      ev.Type,
			RunID:     ev.RunID,
			Iteration: ev.Iteration,
			Time:      ev.Time,
			Payload:   payload,
		}); ok {
			r.manager.RecordRunTimelineEvent(
				timeline.RunID,
				string(timeline.Phase),
				string(timeline.Status),
				timeline.Sequence,
				timeline.Time,
			)
		}
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
			Time:                                 ev.Time,
			RunID:                                ev.RunID,
			Status:                               status,
			Iterations:                           ev.Iteration,
			ToolCalls:                            payloadInt(payload, "tool_calls"),
			LatencyMs:                            payloadInt64(payload, "latency_ms"),
			ErrorClass:                           errorClass,
			PolicyKind:                           payloadString(payload, "policy_kind"),
			NamespaceTool:                        payloadString(payload, "namespace_tool"),
			FilterStage:                          payloadString(payload, "filter_stage"),
			Decision:                             payloadString(payload, "decision"),
			ReasonCode:                           payloadString(payload, "reason_code"),
			Severity:                             payloadString(payload, "severity"),
			AlertDispatchStatus:                  payloadString(payload, "alert_dispatch_status"),
			AlertDispatchFailureReason:           payloadString(payload, "alert_dispatch_failure_reason"),
			AlertDeliveryMode:                    payloadString(payload, "alert_delivery_mode"),
			AlertRetryCount:                      payloadInt(payload, "alert_retry_count"),
			AlertQueueDropped:                    payloadBool(payload, "alert_queue_dropped"),
			AlertQueueDropCount:                  payloadInt(payload, "alert_queue_drop_count"),
			AlertCircuitState:                    payloadString(payload, "alert_circuit_state"),
			AlertCircuitOpenReason:               payloadString(payload, "alert_circuit_open_reason"),
			ModelProvider:                        payloadString(payload, "model_provider"),
			FallbackUsed:                         payloadBool(payload, "fallback_used"),
			FallbackInitial:                      payloadString(payload, "fallback_initial"),
			FallbackPath:                         payloadString(payload, "fallback_path"),
			RequiredCapabilities:                 payloadString(payload, "required_capabilities"),
			FallbackReason:                       payloadString(payload, "fallback_reason"),
			PrefixHash:                           payloadString(payload, "prefix_hash"),
			AssembleLatencyMs:                    payloadInt64(payload, "assemble_latency_ms"),
			AssembleStatus:                       payloadString(payload, "assemble_status"),
			GuardViolation:                       payloadString(payload, "guard_violation"),
			AssembleStageStatus:                  payloadString(payload, "assemble_stage_status"),
			Stage2SkipReason:                     payloadString(payload, "stage2_skip_reason"),
			Stage2RouterMode:                     payloadString(payload, "stage2_router_mode"),
			Stage2RouterDecision:                 payloadString(payload, "stage2_router_decision"),
			Stage2RouterReason:                   payloadString(payload, "stage2_router_reason"),
			Stage2RouterLatencyMs:                payloadInt64(payload, "stage2_router_latency_ms"),
			Stage2RouterError:                    payloadString(payload, "stage2_router_error"),
			Stage1LatencyMs:                      payloadInt64(payload, "stage1_latency_ms"),
			Stage2LatencyMs:                      payloadInt64(payload, "stage2_latency_ms"),
			Stage2Provider:                       payloadString(payload, "stage2_provider"),
			Stage2Profile:                        payloadString(payload, "stage2_profile"),
			Stage2TemplateProfile:                payloadString(payload, "stage2_template_profile"),
			Stage2TemplateResolutionSource:       payloadString(payload, "stage2_template_resolution_source"),
			Stage2HintApplied:                    payloadBool(payload, "stage2_hint_applied"),
			Stage2HintMismatchReason:             payloadString(payload, "stage2_hint_mismatch_reason"),
			Stage2HitCount:                       payloadInt(payload, "stage2_hit_count"),
			Stage2Source:                         payloadString(payload, "stage2_source"),
			Stage2Reason:                         payloadString(payload, "stage2_reason"),
			Stage2ReasonCode:                     payloadString(payload, "stage2_reason_code"),
			Stage2ErrorLayer:                     payloadString(payload, "stage2_error_layer"),
			CA3PressureZone:                      payloadString(payload, "ca3_pressure_zone"),
			CA3PressureReason:                    payloadString(payload, "ca3_pressure_reason"),
			CA3PressureTrigger:                   payloadString(payload, "ca3_pressure_trigger"),
			CA3ZoneResidencyMs:                   payloadInt64Map(payload, "ca3_zone_residency_ms"),
			CA3TriggerCounts:                     payloadIntMap(payload, "ca3_trigger_counts"),
			CA3CompressionRatio:                  payloadFloat64(payload, "ca3_compression_ratio"),
			CA3SpillCount:                        payloadInt(payload, "ca3_spill_count"),
			CA3SwapBackCount:                     payloadInt(payload, "ca3_swap_back_count"),
			CA3CompactionMode:                    payloadString(payload, "ca3_compaction_mode"),
			CA3CompactionFallback:                payloadBool(payload, "ca3_compaction_fallback"),
			CA3CompactionFallbackReason:          payloadString(payload, "ca3_compaction_fallback_reason"),
			CA3CompactionQualityScore:            payloadFloat64(payload, "ca3_compaction_quality_score"),
			CA3CompactionQualityReason:           payloadString(payload, "ca3_compaction_quality_reason"),
			CA3CompactionEmbeddingProvider:       payloadString(payload, "ca3_compaction_embedding_provider"),
			CA3CompactionEmbeddingSimilarity:     payloadFloat64(payload, "ca3_compaction_embedding_similarity"),
			CA3CompactionEmbeddingContribution:   payloadFloat64(payload, "ca3_compaction_embedding_contribution"),
			CA3CompactionEmbeddingStatus:         payloadString(payload, "ca3_compaction_embedding_status"),
			CA3CompactionEmbeddingFallbackReason: payloadString(payload, "ca3_compaction_embedding_fallback_reason"),
			CA3CompactionRerankerUsed:            payloadBool(payload, "ca3_compaction_reranker_used"),
			CA3CompactionRerankerProvider:        payloadString(payload, "ca3_compaction_reranker_provider"),
			CA3CompactionRerankerModel:           payloadString(payload, "ca3_compaction_reranker_model"),
			CA3CompactionRerankerThresholdSource: payloadString(payload, "ca3_compaction_reranker_threshold_source"),
			CA3CompactionRerankerThresholdHit:    payloadBool(payload, "ca3_compaction_reranker_threshold_hit"),
			CA3CompactionRerankerFallbackReason:  payloadString(payload, "ca3_compaction_reranker_fallback_reason"),
			CA3CompactionRerankerProfileVersion:  payloadString(payload, "ca3_compaction_reranker_profile_version"),
			CA3CompactionRerankerRolloutHit:      payloadBool(payload, "ca3_compaction_reranker_rollout_hit"),
			CA3CompactionRerankerThresholdDrift:  payloadFloat64(payload, "ca3_compaction_reranker_threshold_drift"),
			CA3RetainedEvidence:                  payloadInt(payload, "ca3_compaction_retained_evidence_count"),
			RecapStatus:                          payloadString(payload, "recap_status"),
			TeamID:                               payloadString(payload, "team_id"),
			TeamStrategy:                         payloadString(payload, "team_strategy"),
			TeamTaskTotal:                        payloadInt(payload, "team_task_total"),
			TeamTaskFailed:                       payloadInt(payload, "team_task_failed"),
			TeamTaskCanceled:                     payloadInt(payload, "team_task_canceled"),
			WorkflowID:                           payloadString(payload, "workflow_id"),
			WorkflowStatus:                       payloadString(payload, "workflow_status"),
			WorkflowStepTotal:                    payloadInt(payload, "workflow_step_total"),
			WorkflowStepFailed:                   payloadInt(payload, "workflow_step_failed"),
			WorkflowResumeCount:                  payloadInt(payload, "workflow_resume_count"),
			GateChecks:                           payloadInt(payload, "gate_checks"),
			GateDeniedCount:                      payloadInt(payload, "gate_denied_count"),
			GateTimeoutCount:                     payloadInt(payload, "gate_timeout_count"),
			GateRuleHitCount:                     payloadInt(payload, "gate_rule_hit_count"),
			GateRuleLastID:                       payloadString(payload, "gate_rule_last_id"),
			AwaitCount:                           payloadInt(payload, "await_count"),
			ResumeCount:                          payloadInt(payload, "resume_count"),
			CancelByUserCount:                    payloadInt(payload, "cancel_by_user_count"),
			CancelPropagated:                     payloadInt(payload, "cancel_propagated_count"),
			BackpressureDrop:                     payloadInt(payload, "backpressure_drop_count"),
			BackpressureDropByPhase:              payloadIntMap(payload, "backpressure_drop_count_by_phase"),
			InflightPeak:                         payloadInt(payload, "inflight_peak"),
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

func payloadFloat64(m map[string]any, key string) float64 {
	if len(m) == 0 {
		return 0
	}
	raw, ok := m[key]
	if !ok {
		return 0
	}
	switch tv := raw.(type) {
	case float64:
		return tv
	case float32:
		return float64(tv)
	case int:
		return float64(tv)
	case int64:
		return float64(tv)
	default:
		return 0
	}
}

func payloadInt64Map(m map[string]any, key string) map[string]int64 {
	if len(m) == 0 {
		return nil
	}
	raw, ok := m[key]
	if !ok {
		return nil
	}
	out := map[string]int64{}
	switch src := raw.(type) {
	case map[string]any:
		for k, v := range src {
			switch tv := v.(type) {
			case int64:
				out[k] = tv
			case int:
				out[k] = int64(tv)
			case float64:
				out[k] = int64(tv)
			}
		}
	case map[string]int64:
		for k, v := range src {
			out[k] = v
		}
	case map[string]int:
		for k, v := range src {
			out[k] = int64(v)
		}
	default:
		return nil
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func payloadIntMap(m map[string]any, key string) map[string]int {
	in := payloadInt64Map(m, key)
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]int, len(in))
	for k, v := range in {
		out[k] = int(v)
	}
	return out
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
