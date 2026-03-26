package event

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/FelixSeptem/baymax/core/types"
	runtimeconfig "github.com/FelixSeptem/baymax/runtime/config"
)

func TestRuntimeRecorderRecordsSkillEvents(t *testing.T) {
	cfgPath := filepath.Join(t.TempDir(), "runtime.yaml")
	cfg := `
mcp:
  active_profile: default
  profiles:
    default:
      call_timeout: 2s
      retry: 0
      backoff: 10ms
      queue_size: 16
      backpressure: block
      read_pool_size: 2
      write_pool_size: 1
`
	if err := os.WriteFile(cfgPath, []byte(strings.TrimSpace(cfg)), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	mgr, err := runtimeconfig.NewManager(runtimeconfig.ManagerOptions{FilePath: cfgPath, EnvPrefix: "BAYMAX"})
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}
	t.Cleanup(func() { _ = mgr.Close() })

	rec := NewRuntimeRecorder(mgr)
	rec.OnEvent(context.Background(), types.Event{
		Version: types.EventSchemaVersionV1,
		Type:    "skill.loaded",
		Time:    time.Now(),
		RunID:   "run-1",
		Payload: map[string]any{"name": "skill-a"},
	})
	items := mgr.RecentSkills(1)
	if len(items) != 1 {
		t.Fatalf("skills len = %d, want 1", len(items))
	}
	if items[0].SkillName != "skill-a" || items[0].Status != "success" {
		t.Fatalf("unexpected skill record: %#v", items[0])
	}
}

func TestRuntimeRecorderRecordsRunFinishedAndDedup(t *testing.T) {
	cfgPath := filepath.Join(t.TempDir(), "runtime.yaml")
	cfg := `
mcp:
  active_profile: default
  profiles:
    default:
      call_timeout: 2s
      retry: 0
      backoff: 10ms
      queue_size: 16
      backpressure: block
      read_pool_size: 2
      write_pool_size: 1
`
	if err := os.WriteFile(cfgPath, []byte(strings.TrimSpace(cfg)), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	mgr, err := runtimeconfig.NewManager(runtimeconfig.ManagerOptions{FilePath: cfgPath, EnvPrefix: "BAYMAX"})
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}
	t.Cleanup(func() { _ = mgr.Close() })

	rec := NewRuntimeRecorder(mgr)
	rec.OnEvent(context.Background(), types.Event{
		Version: types.EventSchemaVersionV1,
		Type:    types.EventTypeActionTimeline,
		Time:    time.Now(),
		RunID:   "run-1",
		Payload: map[string]any{
			"phase":    "model",
			"status":   "running",
			"sequence": int64(1),
		},
	})
	rec.OnEvent(context.Background(), types.Event{
		Version: types.EventSchemaVersionV1,
		Type:    types.EventTypeActionTimeline,
		Time:    time.Now().Add(10 * time.Millisecond),
		RunID:   "run-1",
		Payload: map[string]any{
			"phase":    "model",
			"status":   "failed",
			"sequence": int64(2),
		},
	})
	rec.OnEvent(context.Background(), types.Event{
		Version: types.EventSchemaVersionV1,
		Type:    types.EventTypeActionTimeline,
		Time:    time.Now().Add(10 * time.Millisecond),
		RunID:   "run-1",
		Payload: map[string]any{
			"phase":    "model",
			"status":   "failed",
			"sequence": int64(2),
		},
	})
	ev := types.Event{
		Version:   types.EventSchemaVersionV1,
		Type:      "run.finished",
		Time:      time.Now(),
		RunID:     "run-1",
		Iteration: 2,
		Payload: map[string]any{
			"status":                                           "failed",
			"latency_ms":                                       int64(120),
			"tool_calls":                                       3,
			"error_class":                                      "ErrTool",
			"policy_kind":                                      "permission",
			"namespace_tool":                                   "local+shell",
			"filter_stage":                                     "input",
			"decision":                                         "deny",
			"reason_code":                                      "security.permission_denied",
			"severity":                                         "high",
			"alert_dispatch_status":                            "failed",
			"alert_dispatch_failure_reason":                    "alert.callback_error",
			"alert_delivery_mode":                              "sync",
			"alert_retry_count":                                2,
			"alert_queue_dropped":                              true,
			"alert_queue_drop_count":                           1,
			"alert_circuit_state":                              "open",
			"alert_circuit_open_reason":                        "alert.callback_error",
			"prefix_hash":                                      "abc123",
			"assemble_latency_ms":                              int64(8),
			"assemble_status":                                  "success",
			"guard_violation":                                  "",
			"assemble_stage_status":                            "stage1_only",
			"stage2_skip_reason":                               "routing.threshold.not_met",
			"stage2_router_mode":                               "agentic",
			"stage2_router_decision":                           "skip_stage2",
			"stage2_router_reason":                             "agentic.fallback.agentic.callback_missing|routing.threshold.not_met",
			"stage2_router_latency_ms":                         int64(7),
			"stage2_router_error":                              "agentic.callback_missing",
			"stage1_latency_ms":                                int64(3),
			"stage2_latency_ms":                                int64(0),
			"stage2_provider":                                  "file",
			"stage2_profile":                                   "http_generic",
			"stage2_template_profile":                          "ragflow_like",
			"stage2_template_resolution_source":                "profile_defaults_then_explicit_overrides",
			"stage2_hint_applied":                              false,
			"stage2_hint_mismatch_reason":                      "hint.unsupported",
			"stage2_hit_count":                                 2,
			"stage2_source":                                    "http",
			"stage2_reason":                                    "ok",
			"stage2_reason_code":                               "ok",
			"stage2_error_layer":                               "",
			"ca3_pressure_zone":                                "warning",
			"ca3_pressure_reason":                              "usage_percent_trigger",
			"ca3_pressure_trigger":                             "warning",
			"ca3_zone_residency_ms":                            map[string]any{"safe": float64(12), "warning": float64(8)},
			"ca3_trigger_counts":                               map[string]any{"warning": float64(2)},
			"ca3_compression_ratio":                            0.42,
			"ca3_spill_count":                                  1,
			"ca3_swap_back_count":                              1,
			"ca3_compaction_mode":                              "semantic",
			"ca3_compaction_fallback":                          true,
			"ca3_compaction_fallback_reason":                   "quality_below_threshold",
			"ca3_compaction_quality_score":                     0.66,
			"ca3_compaction_quality_reason":                    "coverage_low",
			"ca3_compaction_embedding_provider":                "openai",
			"ca3_compaction_embedding_similarity":              0.81,
			"ca3_compaction_embedding_contribution":            0.24,
			"ca3_compaction_embedding_status":                  "used",
			"ca3_compaction_reranker_used":                     true,
			"ca3_compaction_reranker_provider":                 "openai",
			"ca3_compaction_reranker_model":                    "text-embedding-3-small",
			"ca3_compaction_reranker_threshold_source":         "provider_model_profile",
			"ca3_compaction_reranker_threshold_hit":            true,
			"ca3_compaction_reranker_profile_version":          "e5-canary-v1",
			"ca3_compaction_reranker_rollout_hit":              true,
			"ca3_compaction_reranker_threshold_drift":          0.12,
			"ca3_compaction_retained_evidence_count":           2,
			"recap_status":                                     "appended",
			"team_id":                                          "team-alpha",
			"team_strategy":                                    "parallel",
			"team_task_total":                                  5,
			"team_task_failed":                                 1,
			"team_task_canceled":                               1,
			"team_remote_task_total":                           3,
			"team_remote_task_failed":                          1,
			"workflow_id":                                      "wf-alpha",
			"workflow_status":                                  "failed",
			"workflow_step_total":                              7,
			"workflow_step_failed":                             2,
			"workflow_remote_step_total":                       4,
			"workflow_remote_step_failed":                      2,
			"workflow_subgraph_expansion_total":                3,
			"workflow_condition_template_total":                2,
			"workflow_graph_compile_failed":                    false,
			"workflow_resume_count":                            1,
			"task_id":                                          "task-observed-1",
			"a2a_task_total":                                   4,
			"a2a_task_failed":                                  1,
			"peer_id":                                          "peer-a2a-1",
			"a2a_error_layer":                                  "transport",
			"a2a_delivery_mode":                                "sse",
			"a2a_delivery_fallback_used":                       true,
			"a2a_delivery_fallback_reason":                     "a2a.delivery_unsupported",
			"a2a_version_local":                                "a2a.v1.2",
			"a2a_version_peer":                                 "a2a.v1.0",
			"a2a_version_negotiation_result":                   "compatible",
			"a2a_async_report_total":                           3,
			"a2a_async_report_failed":                          1,
			"a2a_async_report_retry_total":                     2,
			"a2a_async_report_dedup_total":                     1,
			"async_await_total":                                2,
			"async_timeout_total":                              1,
			"async_late_report_total":                          1,
			"async_report_dedup_total":                         1,
			"composer_managed":                                 true,
			"scheduler_backend":                                "file",
			"scheduler_qos_mode":                               "priority",
			"scheduler_backend_fallback":                       true,
			"scheduler_backend_fallback_reason":                "scheduler.backend.file_init_failed",
			"scheduler_queue_total":                            3,
			"scheduler_claim_total":                            4,
			"scheduler_reclaim_total":                          1,
			"scheduler_priority_claim_total":                   3,
			"scheduler_fairness_yield_total":                   1,
			"scheduler_retry_backoff_total":                    2,
			"scheduler_dead_letter_total":                      1,
			"scheduler_delayed_task_total":                     2,
			"scheduler_delayed_claim_total":                    2,
			"scheduler_delayed_wait_ms_p95":                    int64(180),
			"subagent_child_total":                             2,
			"subagent_child_failed":                            1,
			"subagent_budget_reject_total":                     1,
			"effective_operation_profile":                      "interactive",
			"timeout_resolution_source":                        "request",
			"timeout_resolution_trace":                         `{"version":"v1","selected_source":"request"}`,
			"timeout_parent_budget_clamp_total":                1,
			"timeout_parent_budget_reject_total":               0,
			"collab_handoff_total":                             1,
			"collab_delegation_total":                          2,
			"collab_aggregation_total":                         2,
			"collab_aggregation_strategy":                      "all_settled",
			"collab_fail_fast_total":                           1,
			"collab_retry_total":                               3,
			"collab_retry_success_total":                       1,
			"collab_retry_exhausted_total":                     1,
			"recovery_enabled":                                 true,
			"recovery_resume_boundary":                         "next_attempt_only",
			"recovery_inflight_policy":                         "no_rewind",
			"recovery_recovered":                               true,
			"recovery_replay_total":                            2,
			"recovery_timeout_reentry_total":                   1,
			"recovery_timeout_reentry_exhausted_total":         1,
			"recovery_conflict":                                false,
			"recovery_conflict_code":                           "",
			"recovery_fallback_used":                           true,
			"recovery_fallback_reason":                         "recovery.backend.file_init_failed",
			"runtime_readiness_status":                         "degraded",
			"runtime_readiness_finding_total":                  2,
			"runtime_readiness_blocking_total":                 0,
			"runtime_readiness_degraded_total":                 2,
			"runtime_primary_domain":                           "timeout",
			"runtime_primary_code":                             "runtime.timeout.parent_budget_rejected",
			"runtime_primary_source":                           "timeout.resolution.request",
			"runtime_primary_conflict_total":                   1,
			"runtime_readiness_primary_code":                   "scheduler.backend.fallback",
			"runtime_readiness_admission_total":                1,
			"runtime_readiness_admission_blocked_total":        0,
			"runtime_readiness_admission_degraded_allow_total": 1,
			"runtime_readiness_admission_bypass_total":         0,
			"runtime_readiness_admission_mode":                 "fail_fast",
			"runtime_readiness_admission_primary_code":         "scheduler.backend.fallback",
			"adapter_health_status":                            "unavailable",
			"adapter_health_probe_total":                       3,
			"adapter_health_degraded_total":                    1,
			"adapter_health_unavailable_total":                 2,
			"adapter_health_primary_code":                      "adapter.health.required_unavailable",
			"adapter_health_backoff_applied_total":             4,
			"adapter_health_circuit_open_total":                2,
			"adapter_health_circuit_half_open_total":           1,
			"adapter_health_circuit_recover_total":             1,
			"adapter_health_circuit_state":                     "open",
			"adapter_health_governance_primary_code":           "adapter.health.circuit_open",
			"gate_checks":                                      4,
			"gate_denied_count":                                2,
			"gate_timeout_count":                               1,
			"gate_rule_hit_count":                              2,
			"gate_rule_last_id":                                "allow-echoloop",
			"await_count":                                      2,
			"resume_count":                                     1,
			"cancel_by_user_count":                             1,
			"cancel_propagated_count":                          3,
			"backpressure_drop_count":                          0,
			"inflight_peak":                                    8,
		},
	}
	rec.OnEvent(context.Background(), ev)
	rec.OnEvent(context.Background(), ev)

	items := mgr.RecentRuns(10)
	if len(items) != 1 {
		t.Fatalf("run records len = %d, want 1", len(items))
	}
	if items[0].Status != "failed" || items[0].ErrorClass != "ErrTool" || items[0].ToolCalls != 3 {
		t.Fatalf("unexpected run record: %#v", items[0])
	}
	if items[0].PolicyKind != "permission" || items[0].NamespaceTool != "local+shell" || items[0].Decision != "deny" {
		t.Fatalf("security fields mismatch: %#v", items[0])
	}
	if items[0].FilterStage != "input" || items[0].ReasonCode != "security.permission_denied" {
		t.Fatalf("security filter/reason mismatch: %#v", items[0])
	}
	if items[0].Severity != "high" || items[0].AlertDispatchStatus != "failed" || items[0].AlertDispatchFailureReason != "alert.callback_error" {
		t.Fatalf("security severity/alert fields mismatch: %#v", items[0])
	}
	if items[0].AlertDeliveryMode != "sync" || items[0].AlertRetryCount != 2 || !items[0].AlertQueueDropped || items[0].AlertQueueDropCount != 1 {
		t.Fatalf("security delivery fields mismatch: %#v", items[0])
	}
	if items[0].AlertCircuitState != "open" || items[0].AlertCircuitOpenReason != "alert.callback_error" {
		t.Fatalf("security circuit fields mismatch: %#v", items[0])
	}
	if items[0].PrefixHash != "abc123" || items[0].AssembleLatencyMs != 8 || items[0].AssembleStatus != "success" {
		t.Fatalf("assembler fields mismatch: %#v", items[0])
	}
	if items[0].AssembleStageStatus != "stage1_only" || items[0].Stage2SkipReason == "" || items[0].RecapStatus != "appended" {
		t.Fatalf("ca2 fields mismatch: %#v", items[0])
	}
	if items[0].Stage2RouterMode != "agentic" || items[0].Stage2RouterDecision != "skip_stage2" {
		t.Fatalf("ca2 router mode/decision mismatch: %#v", items[0])
	}
	if items[0].Stage2RouterLatencyMs != 7 || items[0].Stage2RouterError != "agentic.callback_missing" {
		t.Fatalf("ca2 router latency/error mismatch: %#v", items[0])
	}
	if !strings.Contains(items[0].Stage2RouterReason, "agentic.fallback.agentic.callback_missing") {
		t.Fatalf("ca2 router reason mismatch: %#v", items[0])
	}
	if items[0].Stage2HitCount != 2 || items[0].Stage2Source != "http" || items[0].Stage2Reason != "ok" {
		t.Fatalf("ca2 retrieval fields mismatch: %#v", items[0])
	}
	if items[0].Stage2ReasonCode != "ok" || items[0].Stage2Profile != "http_generic" {
		t.Fatalf("ca2 retrieval extended fields mismatch: %#v", items[0])
	}
	if items[0].Stage2TemplateProfile != "ragflow_like" ||
		items[0].Stage2TemplateResolutionSource != "profile_defaults_then_explicit_overrides" ||
		items[0].Stage2HintMismatchReason != "hint.unsupported" {
		t.Fatalf("ca2 hint/template fields mismatch: %#v", items[0])
	}
	if items[0].CA3PressureZone != "warning" || items[0].CA3PressureReason == "" {
		t.Fatalf("ca3 fields mismatch: %#v", items[0])
	}
	if items[0].CA3PressureTrigger != "warning" {
		t.Fatalf("ca3 trigger mismatch: %#v", items[0])
	}
	if items[0].CA3CompressionRatio == 0 || items[0].CA3SpillCount != 1 || items[0].CA3SwapBackCount != 1 {
		t.Fatalf("ca3 metrics mismatch: %#v", items[0])
	}
	if items[0].CA3CompactionMode != "semantic" || !items[0].CA3CompactionFallback || items[0].CA3RetainedEvidence != 2 {
		t.Fatalf("ca3 compaction metrics mismatch: %#v", items[0])
	}
	if items[0].CA3CompactionFallbackReason != "quality_below_threshold" {
		t.Fatalf("ca3 compaction fallback reason mismatch: %#v", items[0])
	}
	if items[0].CA3CompactionQualityScore != 0.66 || items[0].CA3CompactionQualityReason != "coverage_low" {
		t.Fatalf("ca3 compaction quality metrics mismatch: %#v", items[0])
	}
	if items[0].CA3CompactionEmbeddingProvider != "openai" || items[0].CA3CompactionEmbeddingStatus != "used" {
		t.Fatalf("ca3 compaction embedding provider/status mismatch: %#v", items[0])
	}
	if items[0].CA3CompactionEmbeddingSimilarity <= 0 || items[0].CA3CompactionEmbeddingContribution <= 0 {
		t.Fatalf("ca3 compaction embedding metrics mismatch: %#v", items[0])
	}
	if !items[0].CA3CompactionRerankerUsed || items[0].CA3CompactionRerankerProvider != "openai" {
		t.Fatalf("ca3 compaction reranker provider/used mismatch: %#v", items[0])
	}
	if items[0].CA3CompactionRerankerModel != "text-embedding-3-small" {
		t.Fatalf("ca3 compaction reranker model mismatch: %#v", items[0])
	}
	if items[0].CA3CompactionRerankerThresholdSource != "provider_model_profile" || !items[0].CA3CompactionRerankerThresholdHit {
		t.Fatalf("ca3 compaction reranker threshold fields mismatch: %#v", items[0])
	}
	if items[0].CA3CompactionRerankerProfileVersion != "e5-canary-v1" || !items[0].CA3CompactionRerankerRolloutHit {
		t.Fatalf("ca3 compaction reranker governance fields mismatch: %#v", items[0])
	}
	if items[0].CA3CompactionRerankerThresholdDrift <= 0 {
		t.Fatalf("ca3 compaction reranker threshold drift mismatch: %#v", items[0])
	}
	if items[0].TeamID != "team-alpha" || items[0].TeamStrategy != "parallel" {
		t.Fatalf("teams summary id/strategy mismatch: %#v", items[0])
	}
	if items[0].TeamTaskTotal != 5 || items[0].TeamTaskFailed != 1 || items[0].TeamTaskCanceled != 1 {
		t.Fatalf("teams summary counters mismatch: %#v", items[0])
	}
	if items[0].TeamRemoteTaskTotal != 3 || items[0].TeamRemoteTaskFailed != 1 {
		t.Fatalf("teams remote summary counters mismatch: %#v", items[0])
	}
	if items[0].WorkflowID != "wf-alpha" || items[0].WorkflowStatus != "failed" {
		t.Fatalf("workflow summary id/status mismatch: %#v", items[0])
	}
	if items[0].WorkflowStepTotal != 7 || items[0].WorkflowStepFailed != 2 || items[0].WorkflowResumeCount != 1 {
		t.Fatalf("workflow summary counters mismatch: %#v", items[0])
	}
	if items[0].WorkflowRemoteStepTotal != 4 || items[0].WorkflowRemoteStepFailed != 2 {
		t.Fatalf("workflow remote summary counters mismatch: %#v", items[0])
	}
	if items[0].WorkflowSubgraphExpansionTotal != 3 || items[0].WorkflowConditionTemplateTotal != 2 || items[0].WorkflowGraphCompileFailed {
		t.Fatalf("workflow graph compile summary mismatch: %#v", items[0])
	}
	if items[0].TaskID != "task-observed-1" {
		t.Fatalf("task_id mismatch: %#v", items[0])
	}
	if items[0].A2ATaskTotal != 4 || items[0].A2ATaskFailed != 1 {
		t.Fatalf("a2a summary counters mismatch: %#v", items[0])
	}
	if items[0].PeerID != "peer-a2a-1" || items[0].A2AErrorLayer != "transport" {
		t.Fatalf("a2a summary fields mismatch: %#v", items[0])
	}
	if items[0].A2ADeliveryMode != "sse" || !items[0].A2ADeliveryFallbackUsed || items[0].A2ADeliveryFallbackReason != "a2a.delivery_unsupported" {
		t.Fatalf("a2a delivery fields mismatch: %#v", items[0])
	}
	if items[0].A2AVersionLocal != "a2a.v1.2" || items[0].A2AVersionPeer != "a2a.v1.0" || items[0].A2AVersionNegotiationResult != "compatible" {
		t.Fatalf("a2a version fields mismatch: %#v", items[0])
	}
	if items[0].A2AAsyncReportTotal != 3 ||
		items[0].A2AAsyncReportFailed != 1 ||
		items[0].A2AAsyncReportRetryTotal != 2 ||
		items[0].A2AAsyncReportDedupTotal != 1 {
		t.Fatalf("a2a async report fields mismatch: %#v", items[0])
	}
	if items[0].AsyncAwaitTotal != 2 ||
		items[0].AsyncTimeoutTotal != 1 ||
		items[0].AsyncLateReportTotal != 1 ||
		items[0].AsyncReportDedupTotal != 1 {
		t.Fatalf("a31 async-await fields mismatch: %#v", items[0])
	}
	if !items[0].ComposerManaged {
		t.Fatalf("composer marker mismatch: %#v", items[0])
	}
	if items[0].SchedulerBackend != "file" || !items[0].SchedulerBackendFallback || items[0].SchedulerBackendFallbackReason != "scheduler.backend.file_init_failed" {
		t.Fatalf("scheduler fallback markers mismatch: %#v", items[0])
	}
	if items[0].SchedulerQueueTotal != 3 || items[0].SchedulerClaimTotal != 4 || items[0].SchedulerReclaimTotal != 1 {
		t.Fatalf("scheduler fields mismatch: %#v", items[0])
	}
	if items[0].SchedulerQoSMode != "priority" ||
		items[0].SchedulerPriorityClaimTotal != 3 ||
		items[0].SchedulerFairnessYieldTotal != 1 ||
		items[0].SchedulerRetryBackoffTotal != 2 ||
		items[0].SchedulerDeadLetterTotal != 1 {
		t.Fatalf("scheduler qos fields mismatch: %#v", items[0])
	}
	if items[0].SchedulerDelayedTaskTotal != 2 ||
		items[0].SchedulerDelayedClaimTotal != 2 ||
		items[0].SchedulerDelayedWaitMsP95 != 180 {
		t.Fatalf("scheduler delayed fields mismatch: %#v", items[0])
	}
	if items[0].SubagentChildTotal != 2 || items[0].SubagentChildFailed != 1 || items[0].SubagentBudgetRejectTotal != 1 {
		t.Fatalf("subagent fields mismatch: %#v", items[0])
	}
	if items[0].EffectiveOperationProfile != "interactive" ||
		items[0].TimeoutResolutionSource != "request" ||
		items[0].TimeoutResolutionTrace == "" ||
		items[0].TimeoutParentBudgetClampTotal != 1 ||
		items[0].TimeoutParentBudgetRejectTotal != 0 {
		t.Fatalf("timeout resolution fields mismatch: %#v", items[0])
	}
	if items[0].CollabHandoffTotal != 1 ||
		items[0].CollabDelegationTotal != 2 ||
		items[0].CollabAggregationTotal != 2 ||
		items[0].CollabAggregationStrategy != "all_settled" ||
		items[0].CollabFailFastTotal != 1 ||
		items[0].CollabRetryTotal != 3 ||
		items[0].CollabRetrySuccessTotal != 1 ||
		items[0].CollabRetryExhaustedTotal != 1 {
		t.Fatalf("collab fields mismatch: %#v", items[0])
	}
	if !items[0].RecoveryEnabled || !items[0].RecoveryRecovered || items[0].RecoveryReplayTotal != 2 {
		t.Fatalf("recovery summary fields mismatch: %#v", items[0])
	}
	if items[0].RecoveryResumeBoundary != "next_attempt_only" ||
		items[0].RecoveryInflightPolicy != "no_rewind" ||
		items[0].RecoveryTimeoutReentryTotal != 1 ||
		items[0].RecoveryTimeoutReentryExhaustedTotal != 1 {
		t.Fatalf("recovery boundary summary fields mismatch: %#v", items[0])
	}
	if !items[0].RecoveryFallbackUsed || items[0].RecoveryFallbackReason != "recovery.backend.file_init_failed" {
		t.Fatalf("recovery fallback fields mismatch: %#v", items[0])
	}
	if items[0].RuntimeReadinessStatus != "degraded" ||
		items[0].RuntimeReadinessFindingTotal != 2 ||
		items[0].RuntimeReadinessBlockingTotal != 0 ||
		items[0].RuntimeReadinessDegradedTotal != 2 ||
		items[0].RuntimePrimaryDomain != "timeout" ||
		items[0].RuntimePrimaryCode != "runtime.timeout.parent_budget_rejected" ||
		items[0].RuntimePrimarySource != "timeout.resolution.request" ||
		items[0].RuntimePrimaryConflictTotal != 1 ||
		items[0].RuntimeReadinessPrimaryCode != "scheduler.backend.fallback" ||
		items[0].RuntimeReadinessAdmissionTotal != 1 ||
		items[0].RuntimeReadinessAdmissionBlockedTotal != 0 ||
		items[0].RuntimeReadinessAdmissionDegradedAllowTotal != 1 ||
		items[0].RuntimeReadinessAdmissionBypassTotal != 0 ||
		items[0].RuntimeReadinessAdmissionMode != "fail_fast" ||
		items[0].RuntimeReadinessAdmissionPrimaryCode != "scheduler.backend.fallback" ||
		items[0].AdapterHealthStatus != "unavailable" ||
		items[0].AdapterHealthProbeTotal != 3 ||
		items[0].AdapterHealthDegradedTotal != 1 ||
		items[0].AdapterHealthUnavailableTotal != 2 ||
		items[0].AdapterHealthPrimaryCode != "adapter.health.required_unavailable" ||
		items[0].AdapterHealthBackoffAppliedTotal != 4 ||
		items[0].AdapterHealthCircuitOpenTotal != 2 ||
		items[0].AdapterHealthCircuitHalfOpenTotal != 1 ||
		items[0].AdapterHealthCircuitRecoverTotal != 1 ||
		items[0].AdapterHealthCircuitState != "open" ||
		items[0].AdapterHealthGovernancePrimaryCode != "adapter.health.circuit_open" {
		t.Fatalf("runtime readiness fields mismatch: %#v", items[0])
	}
	if items[0].GateChecks != 4 || items[0].GateDeniedCount != 2 || items[0].GateTimeoutCount != 1 {
		t.Fatalf("action gate metrics mismatch: %#v", items[0])
	}
	if items[0].GateRuleHitCount != 2 || items[0].GateRuleLastID != "allow-echoloop" {
		t.Fatalf("action gate rule metrics mismatch: %#v", items[0])
	}
	if items[0].AwaitCount != 2 || items[0].ResumeCount != 1 || items[0].CancelByUserCount != 1 {
		t.Fatalf("clarification metrics mismatch: %#v", items[0])
	}
	if items[0].CancelPropagated != 3 || items[0].BackpressureDrop != 0 || items[0].InflightPeak != 8 {
		t.Fatalf("concurrency metrics mismatch: %#v", items[0])
	}
	modelAgg, ok := items[0].TimelinePhases["model"]
	if !ok {
		t.Fatalf("timeline model aggregate missing: %#v", items[0].TimelinePhases)
	}
	if modelAgg.CountTotal != 1 || modelAgg.FailedTotal != 1 {
		t.Fatalf("timeline model aggregate mismatch: %#v", modelAgg)
	}
}

func TestRuntimeRecorderA14ParserCompatibilityAdditiveNullableDefault(t *testing.T) {
	cfgPath := filepath.Join(t.TempDir(), "runtime.yaml")
	cfg := `
mcp:
  active_profile: default
  profiles:
    default:
      call_timeout: 2s
      retry: 0
      backoff: 10ms
      queue_size: 16
      backpressure: block
      read_pool_size: 2
      write_pool_size: 1
`
	if err := os.WriteFile(cfgPath, []byte(strings.TrimSpace(cfg)), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	mgr, err := runtimeconfig.NewManager(runtimeconfig.ManagerOptions{FilePath: cfgPath, EnvPrefix: "BAYMAX"})
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}
	t.Cleanup(func() { _ = mgr.Close() })

	rec := NewRuntimeRecorder(mgr)
	rec.OnEvent(context.Background(), types.Event{
		Version: types.EventSchemaVersionV1,
		Type:    "run.finished",
		RunID:   "run-a14-compat",
		Time:    time.Now(),
		Payload: map[string]any{
			"status":                "success",
			"latency_ms":            int64(11),
			"tool_calls":            1,
			"team_task_total":       2,
			"a14_future_field":      123,
			"a14_future_nested_map": map[string]any{"k": "v"},
		},
	})

	items := mgr.RecentRuns(1)
	if len(items) != 1 {
		t.Fatalf("run records len = %d, want 1", len(items))
	}
	got := items[0]
	if got.Status != "success" || got.ToolCalls != 1 || got.LatencyMs != 11 {
		t.Fatalf("existing run fields should stay unchanged: %#v", got)
	}
	if got.TeamTaskTotal != 2 {
		t.Fatalf("pre-existing additive field semantics changed: %#v", got)
	}
	if got.A2AAsyncReportTotal != 0 ||
		got.A2AAsyncReportFailed != 0 ||
		got.A2AAsyncReportRetryTotal != 0 ||
		got.A2AAsyncReportDedupTotal != 0 ||
		got.AsyncAwaitTotal != 0 ||
		got.AsyncTimeoutTotal != 0 ||
		got.AsyncLateReportTotal != 0 ||
		got.AsyncReportDedupTotal != 0 ||
		got.SchedulerDelayedTaskTotal != 0 ||
		got.SchedulerDelayedClaimTotal != 0 ||
		got.SchedulerDelayedWaitMsP95 != 0 ||
		got.WorkflowSubgraphExpansionTotal != 0 ||
		got.WorkflowConditionTemplateTotal != 0 ||
		got.WorkflowGraphCompileFailed {
		t.Fatalf("missing A12/A13 additive fields must resolve to documented defaults: %#v", got)
	}
}

func TestRuntimeRecorderA16ParserCompatibilityAdditiveNullableDefault(t *testing.T) {
	cfgPath := filepath.Join(t.TempDir(), "runtime.yaml")
	cfg := `
mcp:
  active_profile: default
  profiles:
    default:
      call_timeout: 2s
      retry: 0
      backoff: 10ms
      queue_size: 16
      backpressure: block
      read_pool_size: 2
      write_pool_size: 1
`
	if err := os.WriteFile(cfgPath, []byte(strings.TrimSpace(cfg)), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	mgr, err := runtimeconfig.NewManager(runtimeconfig.ManagerOptions{FilePath: cfgPath, EnvPrefix: "BAYMAX"})
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}
	t.Cleanup(func() { _ = mgr.Close() })

	rec := NewRuntimeRecorder(mgr)
	rec.OnEvent(context.Background(), types.Event{
		Version: types.EventSchemaVersionV1,
		Type:    "run.finished",
		RunID:   "run-a16-compat",
		Time:    time.Now(),
		Payload: map[string]any{
			"status":           "success",
			"latency_ms":       int64(12),
			"a16_future_field": "ignore_me",
		},
	})

	items := mgr.RecentRuns(1)
	if len(items) != 1 {
		t.Fatalf("run records len = %d, want 1", len(items))
	}
	got := items[0]
	if got.Status != "success" || got.LatencyMs != 12 {
		t.Fatalf("existing run fields should stay unchanged: %#v", got)
	}
	if got.CollabHandoffTotal != 0 ||
		got.CollabDelegationTotal != 0 ||
		got.CollabAggregationTotal != 0 ||
		got.CollabAggregationStrategy != "" ||
		got.CollabFailFastTotal != 0 ||
		got.CollabRetryTotal != 0 ||
		got.CollabRetrySuccessTotal != 0 ||
		got.CollabRetryExhaustedTotal != 0 {
		t.Fatalf("missing A16 additive fields must resolve to documented defaults: %#v", got)
	}
}

func TestRuntimeRecorderA17ParserCompatibilityAdditiveNullableDefault(t *testing.T) {
	cfgPath := filepath.Join(t.TempDir(), "runtime.yaml")
	cfg := `
mcp:
  active_profile: default
  profiles:
    default:
      call_timeout: 2s
      retry: 0
      backoff: 10ms
      queue_size: 16
      backpressure: block
      read_pool_size: 2
      write_pool_size: 1
`
	if err := os.WriteFile(cfgPath, []byte(strings.TrimSpace(cfg)), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	mgr, err := runtimeconfig.NewManager(runtimeconfig.ManagerOptions{FilePath: cfgPath, EnvPrefix: "BAYMAX"})
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}
	t.Cleanup(func() { _ = mgr.Close() })

	rec := NewRuntimeRecorder(mgr)
	rec.OnEvent(context.Background(), types.Event{
		Version: types.EventSchemaVersionV1,
		Type:    "run.finished",
		RunID:   "run-a17-compat",
		Time:    time.Now(),
		Payload: map[string]any{
			"status":           "success",
			"latency_ms":       int64(13),
			"a17_future_field": "ignore_me",
		},
	})

	items := mgr.RecentRuns(1)
	if len(items) != 1 {
		t.Fatalf("run records len = %d, want 1", len(items))
	}
	got := items[0]
	if got.Status != "success" || got.LatencyMs != 13 {
		t.Fatalf("existing run fields should stay unchanged: %#v", got)
	}
	if got.RecoveryResumeBoundary != "" ||
		got.RecoveryInflightPolicy != "" ||
		got.RecoveryTimeoutReentryTotal != 0 ||
		got.RecoveryTimeoutReentryExhaustedTotal != 0 {
		t.Fatalf("missing A17 additive fields must resolve to documented defaults: %#v", got)
	}
}

func TestRuntimeRecorderA40ParserCompatibilityAdditiveNullableDefault(t *testing.T) {
	cfgPath := filepath.Join(t.TempDir(), "runtime.yaml")
	cfg := `
mcp:
  active_profile: default
  profiles:
    default:
      call_timeout: 2s
      retry: 0
      backoff: 10ms
      queue_size: 16
      backpressure: block
      read_pool_size: 2
      write_pool_size: 1
`
	if err := os.WriteFile(cfgPath, []byte(strings.TrimSpace(cfg)), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	mgr, err := runtimeconfig.NewManager(runtimeconfig.ManagerOptions{FilePath: cfgPath, EnvPrefix: "BAYMAX"})
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}
	t.Cleanup(func() { _ = mgr.Close() })

	rec := NewRuntimeRecorder(mgr)
	rec.OnEvent(context.Background(), types.Event{
		Version: types.EventSchemaVersionV1,
		Type:    "run.finished",
		RunID:   "run-a40-compat",
		Time:    time.Now(),
		Payload: map[string]any{
			"status":           "success",
			"latency_ms":       int64(14),
			"a40_future_field": "ignore_me",
		},
	})

	items := mgr.RecentRuns(1)
	if len(items) != 1 {
		t.Fatalf("run records len = %d, want 1", len(items))
	}
	got := items[0]
	if got.Status != "success" || got.LatencyMs != 14 {
		t.Fatalf("existing run fields should stay unchanged: %#v", got)
	}
	if got.RuntimeReadinessStatus != "" ||
		got.RuntimeReadinessFindingTotal != 0 ||
		got.RuntimeReadinessBlockingTotal != 0 ||
		got.RuntimeReadinessDegradedTotal != 0 ||
		got.RuntimePrimaryDomain != "" ||
		got.RuntimePrimaryCode != "" ||
		got.RuntimePrimarySource != "" ||
		got.RuntimePrimaryConflictTotal != 0 ||
		got.RuntimeReadinessPrimaryCode != "" ||
		got.RuntimeReadinessAdmissionTotal != 0 ||
		got.RuntimeReadinessAdmissionBlockedTotal != 0 ||
		got.RuntimeReadinessAdmissionDegradedAllowTotal != 0 ||
		got.RuntimeReadinessAdmissionBypassTotal != 0 ||
		got.RuntimeReadinessAdmissionMode != "" ||
		got.RuntimeReadinessAdmissionPrimaryCode != "" ||
		got.AdapterHealthStatus != "" ||
		got.AdapterHealthProbeTotal != 0 ||
		got.AdapterHealthDegradedTotal != 0 ||
		got.AdapterHealthUnavailableTotal != 0 ||
		got.AdapterHealthPrimaryCode != "" ||
		got.AdapterHealthBackoffAppliedTotal != 0 ||
		got.AdapterHealthCircuitOpenTotal != 0 ||
		got.AdapterHealthCircuitHalfOpenTotal != 0 ||
		got.AdapterHealthCircuitRecoverTotal != 0 ||
		got.AdapterHealthCircuitState != "" ||
		got.AdapterHealthGovernancePrimaryCode != "" {
		t.Fatalf("missing A40 additive fields must resolve to documented defaults: %#v", got)
	}
}

func TestRuntimeRecorderA41ParserCompatibilityAdditiveNullableDefault(t *testing.T) {
	cfgPath := filepath.Join(t.TempDir(), "runtime.yaml")
	cfg := `
mcp:
  active_profile: default
  profiles:
    default:
      call_timeout: 2s
      retry: 0
      backoff: 10ms
      queue_size: 16
      backpressure: block
      read_pool_size: 2
      write_pool_size: 1
`
	if err := os.WriteFile(cfgPath, []byte(strings.TrimSpace(cfg)), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	mgr, err := runtimeconfig.NewManager(runtimeconfig.ManagerOptions{FilePath: cfgPath, EnvPrefix: "BAYMAX"})
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}
	t.Cleanup(func() { _ = mgr.Close() })

	rec := NewRuntimeRecorder(mgr)
	rec.OnEvent(context.Background(), types.Event{
		Version: types.EventSchemaVersionV1,
		Type:    "run.finished",
		RunID:   "run-a41-compat",
		Time:    time.Now(),
		Payload: map[string]any{
			"status":           "success",
			"latency_ms":       int64(15),
			"a41_future_field": "ignore_me",
		},
	})

	items := mgr.RecentRuns(1)
	if len(items) != 1 {
		t.Fatalf("run records len = %d, want 1", len(items))
	}
	got := items[0]
	if got.Status != "success" || got.LatencyMs != 15 {
		t.Fatalf("existing run fields should stay unchanged: %#v", got)
	}
	if got.EffectiveOperationProfile != "" ||
		got.TimeoutResolutionSource != "" ||
		got.TimeoutResolutionTrace != "" ||
		got.TimeoutParentBudgetClampTotal != 0 ||
		got.TimeoutParentBudgetRejectTotal != 0 {
		t.Fatalf("missing A41 additive fields must resolve to documented defaults: %#v", got)
	}
}

func TestRuntimeRecorderA45ParserCompatibilityAdditiveNullableDefault(t *testing.T) {
	cfgPath := filepath.Join(t.TempDir(), "runtime.yaml")
	cfg := `
mcp:
  active_profile: default
  profiles:
    default:
      call_timeout: 2s
      retry: 0
      backoff: 10ms
      queue_size: 16
      backpressure: block
      read_pool_size: 2
      write_pool_size: 1
`
	if err := os.WriteFile(cfgPath, []byte(strings.TrimSpace(cfg)), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	mgr, err := runtimeconfig.NewManager(runtimeconfig.ManagerOptions{FilePath: cfgPath, EnvPrefix: "BAYMAX"})
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}
	t.Cleanup(func() { _ = mgr.Close() })

	rec := NewRuntimeRecorder(mgr)
	rec.OnEvent(context.Background(), types.Event{
		Version: types.EventSchemaVersionV1,
		Type:    "run.finished",
		RunID:   "run-a45-compat",
		Time:    time.Now(),
		Payload: map[string]any{
			"status":           "success",
			"latency_ms":       int64(16),
			"a45_future_field": "ignore_me",
		},
	})

	items := mgr.RecentRuns(1)
	if len(items) != 1 {
		t.Fatalf("run records len = %d, want 1", len(items))
	}
	got := items[0]
	if got.Status != "success" || got.LatencyMs != 16 {
		t.Fatalf("existing run fields should stay unchanged: %#v", got)
	}
	if got.DiagnosticsCardinalityBudgetHitTotal != 0 ||
		got.DiagnosticsCardinalityTruncatedTotal != 0 ||
		got.DiagnosticsCardinalityFailFastRejectTotal != 0 ||
		got.DiagnosticsCardinalityOverflowPolicy != "truncate_and_record" ||
		got.DiagnosticsCardinalityTruncatedFieldSummary != "" {
		t.Fatalf("missing A45 additive fields must resolve to documented defaults: %#v", got)
	}
}

func TestRuntimeRecorderA46ParserCompatibilityAdditiveNullableDefault(t *testing.T) {
	cfgPath := filepath.Join(t.TempDir(), "runtime.yaml")
	cfg := `
mcp:
  active_profile: default
  profiles:
    default:
      call_timeout: 2s
      retry: 0
      backoff: 10ms
      queue_size: 16
      backpressure: block
      read_pool_size: 2
      write_pool_size: 1
`
	if err := os.WriteFile(cfgPath, []byte(strings.TrimSpace(cfg)), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	mgr, err := runtimeconfig.NewManager(runtimeconfig.ManagerOptions{FilePath: cfgPath, EnvPrefix: "BAYMAX"})
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}
	t.Cleanup(func() { _ = mgr.Close() })

	rec := NewRuntimeRecorder(mgr)
	rec.OnEvent(context.Background(), types.Event{
		Version: types.EventSchemaVersionV1,
		Type:    "run.finished",
		RunID:   "run-a46-compat",
		Time:    time.Now(),
		Payload: map[string]any{
			"status":           "success",
			"latency_ms":       int64(17),
			"a46_future_field": "ignore_me",
		},
	})

	items := mgr.RecentRuns(1)
	if len(items) != 1 {
		t.Fatalf("run records len = %d, want 1", len(items))
	}
	got := items[0]
	if got.Status != "success" || got.LatencyMs != 17 {
		t.Fatalf("existing run fields should stay unchanged: %#v", got)
	}
	if got.AdapterHealthBackoffAppliedTotal != 0 ||
		got.AdapterHealthCircuitOpenTotal != 0 ||
		got.AdapterHealthCircuitHalfOpenTotal != 0 ||
		got.AdapterHealthCircuitRecoverTotal != 0 ||
		got.AdapterHealthCircuitState != "" ||
		got.AdapterHealthGovernancePrimaryCode != "" {
		t.Fatalf("missing A46 additive fields must resolve to documented defaults: %#v", got)
	}
}

func TestRuntimeRecorderRedactsSensitivePayload(t *testing.T) {
	cfgPath := filepath.Join(t.TempDir(), "runtime.yaml")
	cfg := `
mcp:
  active_profile: default
  profiles:
    default:
      call_timeout: 2s
      retry: 0
      backoff: 10ms
      queue_size: 16
      backpressure: block
      read_pool_size: 2
      write_pool_size: 1
security:
  redaction:
    enabled: true
    strategy: keyword
    keywords: [token]
`
	if err := os.WriteFile(cfgPath, []byte(strings.TrimSpace(cfg)), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	mgr, err := runtimeconfig.NewManager(runtimeconfig.ManagerOptions{FilePath: cfgPath, EnvPrefix: "BAYMAX"})
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}
	t.Cleanup(func() { _ = mgr.Close() })

	rec := NewRuntimeRecorder(mgr)
	rec.OnEvent(context.Background(), types.Event{
		Version: types.EventSchemaVersionV1,
		Type:    "skill.loaded",
		Time:    time.Now(),
		RunID:   "run-1",
		Payload: map[string]any{"name": "skill-a", "access_token": "secret"},
	})
	items := mgr.RecentSkills(1)
	if len(items) != 1 {
		t.Fatalf("skills len = %d, want 1", len(items))
	}
	if items[0].Payload["access_token"] != "***" {
		t.Fatalf("access_token should be masked, got %#v", items[0].Payload["access_token"])
	}
}

func TestRuntimeRecorderIgnoresActionTimelineForRunAggregation(t *testing.T) {
	cfgPath := filepath.Join(t.TempDir(), "runtime.yaml")
	cfg := `
mcp:
  active_profile: default
  profiles:
    default:
      call_timeout: 2s
      retry: 0
      backoff: 10ms
      queue_size: 16
      backpressure: block
      read_pool_size: 2
      write_pool_size: 1
`
	if err := os.WriteFile(cfgPath, []byte(strings.TrimSpace(cfg)), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	mgr, err := runtimeconfig.NewManager(runtimeconfig.ManagerOptions{FilePath: cfgPath, EnvPrefix: "BAYMAX"})
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}
	t.Cleanup(func() { _ = mgr.Close() })

	rec := NewRuntimeRecorder(mgr)
	rec.OnEvent(context.Background(), types.Event{
		Version: types.EventSchemaVersionV1,
		Type:    types.EventTypeActionTimeline,
		RunID:   "run-1",
		Time:    time.Now(),
		Payload: map[string]any{
			"phase":    "run",
			"status":   "running",
			"sequence": int64(1),
		},
	})
	rec.OnEvent(context.Background(), types.Event{
		Version: types.EventSchemaVersionV1,
		Type:    types.EventTypeActionTimeline,
		RunID:   "run-1",
		Time:    time.Now().Add(15 * time.Millisecond),
		Payload: map[string]any{
			"phase":    "run",
			"status":   "success", // invalid status should be ignored
			"sequence": int64(2),
		},
	})
	rec.OnEvent(context.Background(), types.Event{
		Version: types.EventSchemaVersionV1,
		Type:    "run.finished",
		RunID:   "run-1",
		Time:    time.Now(),
		Payload: map[string]any{
			"status":     "success",
			"latency_ms": int64(9),
		},
	})

	items := mgr.RecentRuns(5)
	if len(items) != 1 {
		t.Fatalf("run records len = %d, want 1", len(items))
	}
	if items[0].RunID != "run-1" || items[0].Status != "success" {
		t.Fatalf("unexpected run record: %#v", items[0])
	}
	runAgg := items[0].TimelinePhases["run"]
	if runAgg.CountTotal != 0 {
		t.Fatalf("invalid timeline status should not be aggregated: %#v", runAgg)
	}
}

func TestRuntimeRecorderTracksCancelPropagationAcrossMCPAndSkillPhases(t *testing.T) {
	cfgPath := filepath.Join(t.TempDir(), "runtime.yaml")
	cfg := `
mcp:
  active_profile: default
  profiles:
    default:
      call_timeout: 2s
      retry: 0
      backoff: 10ms
      queue_size: 16
      backpressure: block
      read_pool_size: 2
      write_pool_size: 1
`
	if err := os.WriteFile(cfgPath, []byte(strings.TrimSpace(cfg)), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	mgr, err := runtimeconfig.NewManager(runtimeconfig.ManagerOptions{FilePath: cfgPath, EnvPrefix: "BAYMAX"})
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}
	t.Cleanup(func() { _ = mgr.Close() })

	rec := NewRuntimeRecorder(mgr)
	now := time.Now()
	for _, phase := range []string{"mcp", "skill"} {
		rec.OnEvent(context.Background(), types.Event{
			Version: types.EventSchemaVersionV1,
			Type:    types.EventTypeActionTimeline,
			RunID:   "run-phase-cancel",
			Time:    now,
			Payload: map[string]any{
				"phase":    phase,
				"status":   "running",
				"sequence": int64(len(phase)),
			},
		})
		rec.OnEvent(context.Background(), types.Event{
			Version: types.EventSchemaVersionV1,
			Type:    types.EventTypeActionTimeline,
			RunID:   "run-phase-cancel",
			Time:    now.Add(10 * time.Millisecond),
			Payload: map[string]any{
				"phase":    phase,
				"status":   "canceled",
				"reason":   "cancel.propagated",
				"sequence": int64(len(phase) + 100),
			},
		})
	}
	rec.OnEvent(context.Background(), types.Event{
		Version: types.EventSchemaVersionV1,
		Type:    "run.finished",
		RunID:   "run-phase-cancel",
		Time:    now.Add(20 * time.Millisecond),
		Payload: map[string]any{
			"status":                  "failed",
			"error_class":             "ErrPolicyTimeout",
			"cancel_propagated_count": 2,
		},
	})

	items := mgr.RecentRuns(1)
	if len(items) != 1 {
		t.Fatalf("run records len = %d, want 1", len(items))
	}
	if items[0].CancelPropagated != 2 {
		t.Fatalf("cancel_propagated_count = %d, want 2", items[0].CancelPropagated)
	}
	mcpAgg := items[0].TimelinePhases["mcp"]
	skillAgg := items[0].TimelinePhases["skill"]
	if mcpAgg.CanceledTotal != 1 || skillAgg.CanceledTotal != 1 {
		t.Fatalf("mcp/skill cancel aggregates mismatch: mcp=%#v skill=%#v", mcpAgg, skillAgg)
	}
}

func TestRuntimeRecorderAcceptsSchedulerNamespaceTimelineEvents(t *testing.T) {
	cfgPath := filepath.Join(t.TempDir(), "runtime.yaml")
	cfg := `
mcp:
  active_profile: default
  profiles:
    default:
      call_timeout: 2s
      retry: 0
      backoff: 10ms
      queue_size: 16
      backpressure: block
      read_pool_size: 2
      write_pool_size: 1
`
	if err := os.WriteFile(cfgPath, []byte(strings.TrimSpace(cfg)), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	mgr, err := runtimeconfig.NewManager(runtimeconfig.ManagerOptions{FilePath: cfgPath, EnvPrefix: "BAYMAX"})
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}
	t.Cleanup(func() { _ = mgr.Close() })

	rec := NewRuntimeRecorder(mgr)
	now := time.Now()
	rec.OnEvent(context.Background(), types.Event{
		Version: types.EventSchemaVersionV1,
		Type:    types.EventTypeActionTimeline,
		RunID:   "run-scheduler-namespace",
		Time:    now,
		Payload: map[string]any{
			"phase":      "run",
			"status":     "running",
			"reason":     "scheduler.claim",
			"task_id":    "task-1",
			"attempt_id": "attempt-1",
			"sequence":   int64(1),
		},
	})
	rec.OnEvent(context.Background(), types.Event{
		Version: types.EventSchemaVersionV1,
		Type:    types.EventTypeActionTimeline,
		RunID:   "run-scheduler-namespace",
		Time:    now.Add(10 * time.Millisecond),
		Payload: map[string]any{
			"phase":      "run",
			"status":     "succeeded",
			"reason":     "subagent.join",
			"task_id":    "task-1",
			"attempt_id": "attempt-1",
			"sequence":   int64(2),
		},
	})
	rec.OnEvent(context.Background(), types.Event{
		Version: types.EventSchemaVersionV1,
		Type:    "run.finished",
		RunID:   "run-scheduler-namespace",
		Time:    now.Add(20 * time.Millisecond),
		Payload: map[string]any{
			"status": "success",
		},
	})

	items := mgr.RecentRuns(1)
	if len(items) != 1 {
		t.Fatalf("run records len = %d, want 1", len(items))
	}
	agg := items[0].TimelinePhases["run"]
	if agg.CountTotal != 1 {
		t.Fatalf("timeline aggregate count_total = %d, want 1", agg.CountTotal)
	}
}
