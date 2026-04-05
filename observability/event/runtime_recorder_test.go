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
			"sandbox_mode":                                     "enforce",
			"sandbox_backend":                                  "windows_job",
			"sandbox_profile":                                  "default",
			"sandbox_session_mode":                             "per_call",
			"sandbox_required_capabilities":                    []any{"stdout_stderr_capture", "oom_signal"},
			"sandbox_decision":                                 "sandbox",
			"sandbox_reason_code":                              "sandbox.timeout",
			"sandbox_fallback_used":                            true,
			"sandbox_fallback_reason":                          "sandbox.fallback_allow_and_record",
			"sandbox_timeout_total":                            1,
			"sandbox_launch_failed_total":                      2,
			"sandbox_capability_mismatch_total":                3,
			"sandbox_queue_wait_ms_p95":                        int64(9),
			"sandbox_exec_latency_ms_p95":                      int64(11),
			"sandbox_exit_code_last":                           137,
			"sandbox_oom_total":                                4,
			"sandbox_resource_cpu_ms_total":                    int64(321),
			"sandbox_resource_memory_peak_bytes_p95":           int64(2048),
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
			"runtime_secondary_reason_codes":                   []any{"runtime.timeout.exhausted", "runtime.timeout.parent_budget_clamped"},
			"runtime_secondary_reason_count":                   2,
			"runtime_arbitration_rule_version":                 "a49.v1",
			"runtime_remediation_hint_code":                    "timeout.adjust_parent_budget",
			"runtime_remediation_hint_domain":                  "timeout",
			"runtime_readiness_primary_code":                   "scheduler.backend.fallback",
			"runtime_readiness_admission_total":                1,
			"runtime_readiness_admission_blocked_total":        0,
			"runtime_readiness_admission_degraded_allow_total": 1,
			"runtime_readiness_admission_bypass_total":         0,
			"runtime_readiness_admission_mode":                 "fail_fast",
			"runtime_readiness_admission_primary_code":         "scheduler.backend.fallback",
			"policy_precedence_version":                        "policy_stack.v1",
			"winner_stage":                                     "sandbox_action",
			"deny_source":                                      "sandbox_action",
			"tie_break_reason":                                 "lexical_code_then_source_order",
			"policy_decision_path": []any{
				map[string]any{
					"stage":    "sandbox_action",
					"code":     "runtime.readiness.admission.sandbox_capacity_deny",
					"source":   "sandbox_action",
					"decision": "deny",
				},
				map[string]any{
					"stage":    "readiness_admission",
					"code":     "runtime.readiness.admission.blocked",
					"source":   "readiness_admission",
					"decision": "deny",
				},
			},
			"adapter_health_status":                  "unavailable",
			"adapter_health_probe_total":             3,
			"adapter_health_degraded_total":          1,
			"adapter_health_unavailable_total":       2,
			"adapter_health_primary_code":            "adapter.health.required_unavailable",
			"adapter_health_backoff_applied_total":   4,
			"adapter_health_circuit_open_total":      2,
			"adapter_health_circuit_half_open_total": 1,
			"adapter_health_circuit_recover_total":   1,
			"adapter_health_circuit_state":           "open",
			"adapter_health_governance_primary_code": "adapter.health.circuit_open",
			"gate_checks":                            4,
			"gate_denied_count":                      2,
			"gate_timeout_count":                     1,
			"gate_rule_hit_count":                    2,
			"gate_rule_last_id":                      "allow-echoloop",
			"await_count":                            2,
			"resume_count":                           1,
			"cancel_by_user_count":                   1,
			"cancel_propagated_count":                3,
			"backpressure_drop_count":                0,
			"inflight_peak":                          8,
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
	if items[0].SandboxMode != "enforce" ||
		items[0].SandboxBackend != "windows_job" ||
		items[0].SandboxProfile != "default" ||
		items[0].SandboxSessionMode != "per_call" ||
		len(items[0].SandboxRequiredCapabilities) != 2 ||
		items[0].SandboxRequiredCapabilities[0] != "stdout_stderr_capture" ||
		items[0].SandboxRequiredCapabilities[1] != "oom_signal" ||
		items[0].SandboxDecision != "sandbox" ||
		items[0].SandboxReasonCode != "sandbox.timeout" ||
		!items[0].SandboxFallbackUsed ||
		items[0].SandboxFallbackReason != "sandbox.fallback_allow_and_record" ||
		items[0].SandboxTimeoutTotal != 1 ||
		items[0].SandboxLaunchFailedTotal != 2 ||
		items[0].SandboxCapabilityMismatchTotal != 3 ||
		items[0].SandboxQueueWaitMsP95 != 9 ||
		items[0].SandboxExecLatencyMsP95 != 11 ||
		items[0].SandboxExitCodeLast != 137 ||
		items[0].SandboxOOMTotal != 4 ||
		items[0].SandboxResourceCPUMsTotal != 321 ||
		items[0].SandboxResourceMemoryPeakBytesP95 != 2048 {
		t.Fatalf("sandbox fields mismatch: %#v", items[0])
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
		len(items[0].RuntimeSecondaryReasonCodes) != 2 ||
		items[0].RuntimeSecondaryReasonCodes[0] != "runtime.timeout.exhausted" ||
		items[0].RuntimeSecondaryReasonCodes[1] != "runtime.timeout.parent_budget_clamped" ||
		items[0].RuntimeSecondaryReasonCount != 2 ||
		items[0].RuntimeArbitrationRuleVersion != "a49.v1" ||
		items[0].RuntimeRemediationHintCode != "timeout.adjust_parent_budget" ||
		items[0].RuntimeRemediationHintDomain != "timeout" ||
		items[0].RuntimeReadinessPrimaryCode != "scheduler.backend.fallback" ||
		items[0].RuntimeReadinessAdmissionTotal != 1 ||
		items[0].RuntimeReadinessAdmissionBlockedTotal != 0 ||
		items[0].RuntimeReadinessAdmissionDegradedAllowTotal != 1 ||
		items[0].RuntimeReadinessAdmissionBypassTotal != 0 ||
		items[0].RuntimeReadinessAdmissionMode != "fail_fast" ||
		items[0].RuntimeReadinessAdmissionPrimaryCode != "scheduler.backend.fallback" ||
		items[0].PolicyPrecedenceVersion != "policy_stack.v1" ||
		items[0].WinnerStage != "sandbox_action" ||
		items[0].DenySource != "sandbox_action" ||
		items[0].TieBreakReason != "lexical_code_then_source_order" ||
		len(items[0].PolicyDecisionPath) != 2 ||
		items[0].PolicyDecisionPath[0].Stage != "sandbox_action" ||
		items[0].PolicyDecisionPath[1].Stage != "readiness_admission" ||
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

func TestRuntimeRecorderRecordsReactAdditiveFieldsAndReplayIdempotent(t *testing.T) {
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
	ev := types.Event{
		Version: types.EventSchemaVersionV1,
		Type:    "run.finished",
		RunID:   "run-a56-react-recorder",
		Time:    time.Now(),
		Payload: map[string]any{
			"status":                           "failed",
			"error_class":                      "ErrIterationLimit",
			"react_enabled":                    true,
			"react_iteration_total":            3,
			"react_tool_call_total":            5,
			"react_tool_call_budget_hit_total": 1,
			"react_iteration_budget_hit_total": 0,
			"react_termination_reason":         "react.tool_call_limit_exceeded",
			"react_stream_dispatch_enabled":    true,
		},
	}
	rec.OnEvent(context.Background(), ev)
	rec.OnEvent(context.Background(), ev)

	items := mgr.RecentRuns(1)
	if len(items) != 1 {
		t.Fatalf("run records len = %d, want 1", len(items))
	}
	got := items[0]
	if !got.ReactEnabled ||
		got.ReactIterationTotal != 3 ||
		got.ReactToolCallTotal != 5 ||
		got.ReactToolCallBudgetHitTotal != 1 ||
		got.ReactIterationBudgetHitTotal != 0 ||
		got.ReactTerminationReason != "react.tool_call_limit_exceeded" ||
		!got.ReactStreamDispatchEnabled {
		t.Fatalf("react additive fields mismatch: %#v", got)
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
		len(got.RuntimeSecondaryReasonCodes) != 0 ||
		got.RuntimeSecondaryReasonCount != 0 ||
		got.RuntimeArbitrationRuleVersion != "" ||
		got.RuntimeRemediationHintCode != "" ||
		got.RuntimeRemediationHintDomain != "" ||
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

func TestRuntimeRecorderA49ParserCompatibilityAdditiveNullableDefault(t *testing.T) {
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
		RunID:   "run-a49-compat",
		Time:    time.Now(),
		Payload: map[string]any{
			"status":           "success",
			"latency_ms":       int64(49),
			"a49_future_field": "ignore_me",
		},
	})

	items := mgr.RecentRuns(1)
	if len(items) != 1 {
		t.Fatalf("run records len = %d, want 1", len(items))
	}
	got := items[0]
	if got.Status != "success" || got.LatencyMs != 49 {
		t.Fatalf("existing run fields should stay unchanged: %#v", got)
	}
	if len(got.RuntimeSecondaryReasonCodes) != 0 ||
		got.RuntimeSecondaryReasonCount != 0 ||
		got.RuntimeArbitrationRuleVersion != "" ||
		got.RuntimeRemediationHintCode != "" ||
		got.RuntimeRemediationHintDomain != "" {
		t.Fatalf("missing A49 additive fields must resolve to documented defaults: %#v", got)
	}
}

func TestRuntimeRecorderA50ParserCompatibilityAdditiveNullableDefault(t *testing.T) {
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
		RunID:   "run-a50-compat",
		Time:    time.Now(),
		Payload: map[string]any{
			"status":           "success",
			"latency_ms":       int64(50),
			"a50_future_field": "ignore_me",
		},
	})

	items := mgr.RecentRuns(1)
	if len(items) != 1 {
		t.Fatalf("run records len = %d, want 1", len(items))
	}
	got := items[0]
	if got.Status != "success" || got.LatencyMs != 50 {
		t.Fatalf("existing run fields should stay unchanged: %#v", got)
	}
	if got.RuntimeArbitrationRuleRequestedVersion != "" ||
		got.RuntimeArbitrationRuleEffectiveVersion != "" ||
		got.RuntimeArbitrationRuleVersionSource != "" ||
		got.RuntimeArbitrationRulePolicyAction != "" ||
		got.RuntimeArbitrationRuleUnsupportedTotal != 0 ||
		got.RuntimeArbitrationRuleMismatchTotal != 0 {
		t.Fatalf("missing A50 additive fields must resolve to documented defaults: %#v", got)
	}
}

func TestRuntimeRecorderA51ParserCompatibilityAdditiveNullableDefault(t *testing.T) {
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
		RunID:   "run-a51-compat",
		Time:    time.Now(),
		Payload: map[string]any{
			"status":           "success",
			"latency_ms":       int64(51),
			"a51_future_field": "ignore_me",
		},
	})

	items := mgr.RecentRuns(1)
	if len(items) != 1 {
		t.Fatalf("run records len = %d, want 1", len(items))
	}
	got := items[0]
	if got.Status != "success" || got.LatencyMs != 51 {
		t.Fatalf("existing run fields should stay unchanged: %#v", got)
	}
	if got.SandboxMode != "" ||
		got.SandboxBackend != "" ||
		got.SandboxProfile != "" ||
		got.SandboxSessionMode != "" ||
		len(got.SandboxRequiredCapabilities) != 0 ||
		got.SandboxDecision != "" ||
		got.SandboxReasonCode != "" ||
		got.SandboxFallbackUsed ||
		got.SandboxFallbackReason != "" ||
		got.SandboxTimeoutTotal != 0 ||
		got.SandboxLaunchFailedTotal != 0 ||
		got.SandboxCapabilityMismatchTotal != 0 ||
		got.SandboxQueueWaitMsP95 != 0 ||
		got.SandboxExecLatencyMsP95 != 0 ||
		got.SandboxExitCodeLast != 0 ||
		got.SandboxOOMTotal != 0 ||
		got.SandboxResourceCPUMsTotal != 0 ||
		got.SandboxResourceMemoryPeakBytesP95 != 0 {
		t.Fatalf("missing A51 additive fields must resolve to documented defaults: %#v", got)
	}
}

func TestRuntimeRecorderA52ParserCompatibilityAdditiveNullableDefault(t *testing.T) {
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
		RunID:   "run-a52-compat",
		Time:    time.Now(),
		Payload: map[string]any{
			"status":           "success",
			"latency_ms":       int64(52),
			"a52_future_field": "ignore_me",
		},
	})

	items := mgr.RecentRuns(1)
	if len(items) != 1 {
		t.Fatalf("run records len = %d, want 1", len(items))
	}
	got := items[0]
	if got.Status != "success" || got.LatencyMs != 52 {
		t.Fatalf("existing run fields should stay unchanged: %#v", got)
	}
	if got.SandboxRolloutPhase != "" ||
		got.SandboxRolloutEffectiveRatio != 0 ||
		got.SandboxHealthBudgetStatus != "" ||
		got.SandboxHealthBudgetBreachTotal != 0 ||
		got.SandboxFreezeState ||
		got.SandboxFreezeReasonCode != "" ||
		got.SandboxCapacityAction != "" ||
		got.SandboxCapacityQueueDepth != 0 ||
		got.SandboxCapacityInflight != 0 {
		t.Fatalf("missing A52 additive fields must resolve to documented defaults: %#v", got)
	}
}

func TestRuntimeRecorderParsesA52RolloutGovernanceFields(t *testing.T) {
	mgr, err := runtimeconfig.NewManager(runtimeconfig.ManagerOptions{EnvPrefix: "BAYMAX"})
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}
	t.Cleanup(func() { _ = mgr.Close() })

	rec := NewRuntimeRecorder(mgr)
	rec.OnEvent(context.Background(), types.Event{
		Version: types.EventSchemaVersionV1,
		Type:    "run.finished",
		RunID:   "run-a52-rollout-governance",
		Time:    time.Now(),
		Payload: map[string]any{
			"status":                             "failed",
			"sandbox_rollout_phase":              "frozen",
			"sandbox_rollout_effective_ratio":    0.25,
			"sandbox_health_budget_status":       "breached",
			"sandbox_health_budget_breach_total": 3,
			"sandbox_freeze_state":               true,
			"sandbox_freeze_reason_code":         "sandbox.rollout.health_budget_breached",
			"sandbox_capacity_action":            "deny",
			"sandbox_capacity_queue_depth":       17,
			"sandbox_capacity_inflight":          8,
		},
	})

	items := mgr.RecentRuns(1)
	if len(items) != 1 {
		t.Fatalf("run records len = %d, want 1", len(items))
	}
	got := items[0]
	if got.SandboxRolloutPhase != "frozen" ||
		got.SandboxRolloutEffectiveRatio != 0.25 ||
		got.SandboxHealthBudgetStatus != "breached" ||
		got.SandboxHealthBudgetBreachTotal != 3 ||
		!got.SandboxFreezeState ||
		got.SandboxFreezeReasonCode != "sandbox.rollout.health_budget_breached" ||
		got.SandboxCapacityAction != "deny" ||
		got.SandboxCapacityQueueDepth != 17 ||
		got.SandboxCapacityInflight != 8 {
		t.Fatalf("A52 rollout governance fields parse mismatch: %#v", got)
	}
}

func TestRuntimeRecorderParsesA54MemoryDiagnosticsFields(t *testing.T) {
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
		RunID:   "run-a54-memory",
		Time:    time.Now(),
		Payload: map[string]any{
			"status":                      "success",
			"memory_mode":                 "external_spi",
			"memory_provider":             "mem0",
			"memory_profile":              "mem0",
			"memory_contract_version":     "memory.v1",
			"memory_query_total":          3,
			"memory_upsert_total":         1,
			"memory_delete_total":         0,
			"memory_error_total":          1,
			"memory_fallback_total":       1,
			"memory_fallback_reason_code": "memory.fallback.used",
			"memory_latency_ms_p95":       int64(27),
		},
	})

	items := mgr.RecentRuns(1)
	if len(items) != 1 {
		t.Fatalf("run records len = %d, want 1", len(items))
	}
	got := items[0]
	if got.MemoryMode != "external_spi" ||
		got.MemoryProvider != "mem0" ||
		got.MemoryProfile != "mem0" ||
		got.MemoryContractVersion != "memory.v1" ||
		got.MemoryQueryTotal != 3 ||
		got.MemoryUpsertTotal != 1 ||
		got.MemoryDeleteTotal != 0 ||
		got.MemoryErrorTotal != 1 ||
		got.MemoryFallbackTotal != 1 ||
		got.MemoryFallbackReasonCode != "memory.fallback.used" ||
		got.MemoryLatencyMsP95 != 27 {
		t.Fatalf("A54 memory field parse mismatch: %#v", got)
	}
}

func TestRuntimeRecorderA54ParserCompatibilityAdditiveNullableDefault(t *testing.T) {
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
		RunID:   "run-a54-compat",
		Time:    time.Now(),
		Payload: map[string]any{
			"status":           "success",
			"latency_ms":       int64(54),
			"a54_future_field": "ignore_me",
		},
	})

	items := mgr.RecentRuns(1)
	if len(items) != 1 {
		t.Fatalf("run records len = %d, want 1", len(items))
	}
	got := items[0]
	if got.Status != "success" || got.LatencyMs != 54 {
		t.Fatalf("existing run fields should stay unchanged: %#v", got)
	}
	if got.MemoryMode != "" ||
		got.MemoryProvider != "" ||
		got.MemoryProfile != "" ||
		got.MemoryContractVersion != "" ||
		got.MemoryQueryTotal != 0 ||
		got.MemoryUpsertTotal != 0 ||
		got.MemoryDeleteTotal != 0 ||
		got.MemoryErrorTotal != 0 ||
		got.MemoryFallbackTotal != 0 ||
		got.MemoryFallbackReasonCode != "" ||
		got.MemoryLatencyMsP95 != 0 {
		t.Fatalf("missing A54 additive fields must resolve to documented defaults: %#v", got)
	}
}

func TestRuntimeRecorderParsesA55ObservabilityDiagnosticsFields(t *testing.T) {
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
		RunID:   "run-a55-observability",
		Time:    time.Now(),
		Payload: map[string]any{
			"status":                                 "failed",
			"observability_export_profile":           "otlp",
			"observability_export_status":            "degraded",
			"observability_export_error_total":       2,
			"observability_export_drop_total":        1,
			"observability_export_queue_depth_peak":  8,
			"diagnostics_bundle_total":               1,
			"diagnostics_bundle_last_status":         "failed",
			"diagnostics_bundle_last_reason_code":    "diagnostics.bundle.output_unavailable",
			"diagnostics_bundle_last_schema_version": "bundle.v1",
		},
	})

	items := mgr.RecentRuns(1)
	if len(items) != 1 {
		t.Fatalf("run records len = %d, want 1", len(items))
	}
	got := items[0]
	if got.ObservabilityExportProfile != "otlp" ||
		got.ObservabilityExportStatus != "degraded" ||
		got.ObservabilityExportErrorTotal != 2 ||
		got.ObservabilityExportDropTotal != 1 ||
		got.ObservabilityExportQueueDepthPeak != 8 ||
		got.DiagnosticsBundleTotal != 1 ||
		got.DiagnosticsBundleLastStatus != "failed" ||
		got.DiagnosticsBundleLastReasonCode != "diagnostics.bundle.output_unavailable" ||
		got.DiagnosticsBundleLastSchemaVersion != "bundle.v1" {
		t.Fatalf("A55 observability diagnostics parse mismatch: %#v", got)
	}
}

func TestRuntimeRecorderA55ParserCompatibilityAdditiveNullableDefault(t *testing.T) {
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
		RunID:   "run-a55-compat",
		Payload: map[string]any{
			"phase":    "run",
			"status":   "running",
			"sequence": int64(1),
		},
	})
	rec.OnEvent(context.Background(), types.Event{
		Version: types.EventSchemaVersionV1,
		Type:    "run.finished",
		RunID:   "run-a55-compat",
		Time:    time.Now(),
		Payload: map[string]any{
			"status":           "success",
			"latency_ms":       int64(55),
			"a55_future_field": "ignore_me",
		},
	})

	items := mgr.RecentRuns(1)
	if len(items) != 1 {
		t.Fatalf("run records len = %d, want 1", len(items))
	}
	got := items[0]
	if got.Status != "success" || got.LatencyMs != 55 {
		t.Fatalf("existing run fields should stay unchanged: %#v", got)
	}
	if got.ObservabilityExportProfile != runtimeconfig.RuntimeObservabilityExportProfileNone ||
		got.ObservabilityExportStatus != RuntimeExportStatusDisabled ||
		got.ObservabilityExportErrorTotal != 0 ||
		got.ObservabilityExportDropTotal != 0 ||
		got.ObservabilityExportQueueDepthPeak != 0 ||
		got.DiagnosticsBundleTotal != 0 ||
		got.DiagnosticsBundleLastStatus != "" ||
		got.DiagnosticsBundleLastReasonCode != "" ||
		got.DiagnosticsBundleLastSchemaVersion != "" {
		t.Fatalf("missing A55 additive fields must resolve to documented defaults: %#v", got)
	}
}

func TestRuntimeRecorderAutoGeneratesA55DiagnosticsBundleSuccess(t *testing.T) {
	bundleDir := filepath.ToSlash(filepath.Join(t.TempDir(), "bundles"))
	cfgPath := filepath.Join(t.TempDir(), "runtime-a55-bundle-success.yaml")
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
runtime:
  diagnostics:
    bundle:
      enabled: true
      output_dir: ` + bundleDir + `
      max_size_mb: 8
      include_sections:
        - timeline
        - diagnostics
        - effective_config
        - replay_hints
        - gate_fingerprint
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
		RunID:   "run-a55-bundle-success",
		Time:    time.Now(),
		Payload: map[string]any{
			"status":        "success",
			"client_secret": "raw-secret",
		},
	})

	items := mgr.RecentRuns(1)
	if len(items) != 1 {
		t.Fatalf("run records len = %d, want 1", len(items))
	}
	got := items[0]
	if got.DiagnosticsBundleTotal != 1 ||
		got.DiagnosticsBundleLastStatus != runtimeconfig.RuntimeDiagnosticsBundleStatusSuccess ||
		got.DiagnosticsBundleLastReasonCode != "" ||
		got.DiagnosticsBundleLastSchemaVersion != runtimeconfig.RuntimeDiagnosticsBundleSchemaVersionV1 {
		t.Fatalf("auto bundle generation summary mismatch: %#v", got)
	}
	entries, err := os.ReadDir(bundleDir)
	if err != nil {
		t.Fatalf("read bundle output dir failed: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("bundle directory count = %d, want 1", len(entries))
	}
	manifestPath := filepath.Join(bundleDir, entries[0].Name(), "manifest.json")
	if _, err := os.Stat(manifestPath); err != nil {
		t.Fatalf("manifest file missing: %v", err)
	}
}

func TestRuntimeRecorderAutoGeneratesA55DiagnosticsBundleFailureReason(t *testing.T) {
	tmp := t.TempDir()
	blocked := filepath.Join(tmp, "blocked")
	if err := os.WriteFile(blocked, []byte("x"), 0o600); err != nil {
		t.Fatalf("write blocked marker: %v", err)
	}
	bundleDir := filepath.ToSlash(filepath.Join(blocked, "bundles"))
	cfgPath := filepath.Join(tmp, "runtime-a55-bundle-failure.yaml")
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
runtime:
  diagnostics:
    bundle:
      enabled: true
      output_dir: ` + bundleDir + `
      max_size_mb: 8
      include_sections:
        - timeline
        - diagnostics
        - effective_config
        - replay_hints
        - gate_fingerprint
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
		RunID:   "run-a55-bundle-failure",
		Time:    time.Now(),
		Payload: map[string]any{
			"status": "failed",
		},
	})

	items := mgr.RecentRuns(1)
	if len(items) != 1 {
		t.Fatalf("run records len = %d, want 1", len(items))
	}
	got := items[0]
	if got.DiagnosticsBundleTotal != 1 ||
		got.DiagnosticsBundleLastStatus != runtimeconfig.RuntimeDiagnosticsBundleStatusFailed ||
		got.DiagnosticsBundleLastReasonCode != runtimeconfig.RuntimeDiagnosticsBundleReasonOutputUnavailable ||
		got.DiagnosticsBundleLastSchemaVersion != runtimeconfig.RuntimeDiagnosticsBundleSchemaVersionV1 {
		t.Fatalf("auto bundle failure mapping mismatch: %#v", got)
	}
}

func TestRuntimeRecorderNormalizesA55ObservabilityAndBundleCardinalitySensitiveFields(t *testing.T) {
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
		RunID:   "run-a55-normalize",
		Time:    time.Now(),
		Payload: map[string]any{
			"status":                                 "success",
			"observability_export_profile":           "otlp-east-2.custom-tenant",
			"observability_export_status":            "partially_failed",
			"diagnostics_bundle_last_status":         "FAILED",
			"diagnostics_bundle_last_reason_code":    "diagnostics.bundle.backend_42_private_code",
			"diagnostics_bundle_last_schema_version": "bundle.v999",
		},
	})

	items := mgr.RecentRuns(1)
	if len(items) != 1 {
		t.Fatalf("run records len = %d, want 1", len(items))
	}
	got := items[0]
	if got.ObservabilityExportProfile != runtimeconfig.RuntimeObservabilityExportProfileNone ||
		got.ObservabilityExportStatus != RuntimeExportStatusDisabled ||
		got.DiagnosticsBundleLastStatus != runtimeconfig.RuntimeDiagnosticsBundleStatusFailed ||
		got.DiagnosticsBundleLastReasonCode != runtimeconfig.RuntimeDiagnosticsBundleReasonUnknown ||
		got.DiagnosticsBundleLastSchemaVersion != runtimeconfig.RuntimeDiagnosticsBundleSchemaVersionV1 {
		t.Fatalf("A55 normalization mismatch: %#v", got)
	}
}

func TestRuntimeRecorderParsesA50ArbitrationVersionGovernanceFields(t *testing.T) {
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
		RunID:   "run-a50-governance",
		Time:    time.Now(),
		Payload: map[string]any{
			"status": "failed",
			"runtime_arbitration_rule_requested_version": "a77.v9",
			"runtime_arbitration_rule_effective_version": "",
			"runtime_arbitration_rule_version_source":    "requested",
			"runtime_arbitration_rule_policy_action":     "fail_fast_unsupported_version",
			"runtime_arbitration_rule_unsupported_total": 1,
			"runtime_arbitration_rule_mismatch_total":    0,
		},
	})

	items := mgr.RecentRuns(1)
	if len(items) != 1 {
		t.Fatalf("run records len = %d, want 1", len(items))
	}
	got := items[0]
	if got.RuntimeArbitrationRuleRequestedVersion != "a77.v9" ||
		got.RuntimeArbitrationRuleEffectiveVersion != "" ||
		got.RuntimeArbitrationRuleVersionSource != "requested" ||
		got.RuntimeArbitrationRulePolicyAction != "fail_fast_unsupported_version" ||
		got.RuntimeArbitrationRuleUnsupportedTotal != 1 ||
		got.RuntimeArbitrationRuleMismatchTotal != 0 {
		t.Fatalf("A50 governance parsing mismatch: %#v", got)
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

func TestRuntimeRecorderParsesA57AdditiveFields(t *testing.T) {
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
		RunID:   "run-a57-recorder",
		Time:    time.Now(),
		Payload: map[string]any{
			"status":                         "failed",
			"sandbox_egress_action":          "deny",
			"sandbox_egress_violation_total": 2,
			"sandbox_egress_policy_source":   "by_tool",
			"adapter_allowlist_decision":     "deny",
			"adapter_allowlist_block_total":  1,
			"adapter_allowlist_primary_code": "adapter.allowlist.missing_entry",
		},
	})

	items := mgr.RecentRuns(1)
	if len(items) != 1 {
		t.Fatalf("run records len = %d, want 1", len(items))
	}
	got := items[0]
	if got.SandboxEgressAction != "deny" ||
		got.SandboxEgressViolationTotal != 2 ||
		got.SandboxEgressPolicySource != "by_tool" ||
		got.AdapterAllowlistDecision != "deny" ||
		got.AdapterAllowlistBlockTotal != 1 ||
		got.AdapterAllowlistPrimaryCode != "adapter.allowlist.missing_entry" {
		t.Fatalf("A57 additive field parse mismatch: %#v", got)
	}
}

func TestRuntimeRecorderParsesA58AdditiveFields(t *testing.T) {
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
		RunID:   "run-a58-recorder",
		Time:    time.Now(),
		Payload: map[string]any{
			"status":                    "failed",
			"policy_precedence_version": "policy_stack.v1",
			"winner_stage":              "sandbox_action",
			"deny_source":               "sandbox_action",
			"tie_break_reason":          "lexical_code_then_source_order",
			"policy_decision_path": []any{
				map[string]any{
					"stage":    "sandbox_action",
					"code":     "runtime.readiness.admission.sandbox_capacity_deny",
					"source":   "sandbox_action",
					"decision": "deny",
				},
				map[string]any{
					"stage":    "readiness_admission",
					"code":     "runtime.readiness.admission.blocked",
					"source":   "readiness_admission",
					"decision": "deny",
				},
			},
		},
	})

	items := mgr.RecentRuns(1)
	if len(items) != 1 {
		t.Fatalf("run records len = %d, want 1", len(items))
	}
	got := items[0]
	if got.PolicyPrecedenceVersion != "policy_stack.v1" ||
		got.WinnerStage != "sandbox_action" ||
		got.DenySource != "sandbox_action" ||
		got.TieBreakReason != "lexical_code_then_source_order" ||
		len(got.PolicyDecisionPath) != 2 ||
		got.PolicyDecisionPath[0].Stage != "sandbox_action" ||
		got.PolicyDecisionPath[1].Stage != "readiness_admission" {
		t.Fatalf("A58 additive field parse mismatch: %#v", got)
	}
}

func TestRuntimeRecorderParsesA59MemoryGovernanceAdditiveFields(t *testing.T) {
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
		RunID:   "run-a59-recorder",
		Time:    time.Now(),
		Payload: map[string]any{
			"status":                  "success",
			"memory_scope_selected":   "session",
			"memory_budget_used":      3,
			"memory_hits":             3,
			"memory_rerank_stats":     map[string]any{"input_total": float64(4), "output_total": float64(3)},
			"memory_lifecycle_action": "ttl_expired",
		},
	})

	items := mgr.RecentRuns(1)
	if len(items) != 1 {
		t.Fatalf("run records len = %d, want 1", len(items))
	}
	got := items[0]
	if got.MemoryScopeSelected != "session" ||
		got.MemoryBudgetUsed != 3 ||
		got.MemoryHits != 3 ||
		got.MemoryRerankStats["input_total"] != 4 ||
		got.MemoryRerankStats["output_total"] != 3 ||
		got.MemoryLifecycleAction != "ttl_expired" {
		t.Fatalf("A59 additive field parse mismatch: %#v", got)
	}
}

func TestRuntimeRecorderA59ParserCompatibilityAdditiveNullableDefault(t *testing.T) {
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
		RunID:   "run-a59-compat",
		Time:    time.Now(),
		Payload: map[string]any{
			"status":           "success",
			"latency_ms":       int64(11),
			"a59_future_field": "ignore_me",
		},
	})

	items := mgr.RecentRuns(1)
	if len(items) != 1 {
		t.Fatalf("run records len = %d, want 1", len(items))
	}
	got := items[0]
	if got.Status != "success" || got.LatencyMs != 11 {
		t.Fatalf("existing run fields should stay unchanged: %#v", got)
	}
	if got.MemoryScopeSelected != "" ||
		got.MemoryBudgetUsed != 0 ||
		got.MemoryHits != 0 ||
		len(got.MemoryRerankStats) != 0 ||
		got.MemoryLifecycleAction != "" {
		t.Fatalf("missing A59 additive fields must resolve to documented defaults: %#v", got)
	}
}

func TestRuntimeRecorderParsesA60BudgetAdmissionAdditiveFields(t *testing.T) {
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
		RunID:   "run-a60-recorder",
		Time:    time.Now(),
		Payload: map[string]any{
			"status":          "success",
			"budget_decision": "degrade",
			"degrade_action":  "trim_memory_context",
			"budget_snapshot": map[string]any{
				"version": "budget_admission.v1",
				"cost_estimate": map[string]any{
					"token":   0.2,
					"tool":    0.1,
					"sandbox": 0.08,
					"memory":  0.05,
					"total":   0.43,
				},
				"latency_estimate": map[string]any{
					"token_ms":   180,
					"tool_ms":    120,
					"sandbox_ms": 100,
					"memory_ms":  60,
					"total_ms":   460,
				},
			},
		},
	})

	items := mgr.RecentRuns(1)
	if len(items) != 1 {
		t.Fatalf("run records len = %d, want 1", len(items))
	}
	got := items[0]
	if got.BudgetDecision != "degrade" ||
		got.DegradeAction != "trim_memory_context" ||
		got.BudgetSnapshot["version"] != "budget_admission.v1" {
		t.Fatalf("A60 additive field parse mismatch: %#v", got)
	}
}

func TestRuntimeRecorderA60ParserCompatibilityAdditiveNullableDefault(t *testing.T) {
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
		RunID:   "run-a60-compat",
		Time:    time.Now(),
		Payload: map[string]any{
			"status":           "success",
			"latency_ms":       int64(13),
			"a60_future_field": "ignore_me",
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
	if got.BudgetDecision != "" ||
		got.DegradeAction != "" ||
		len(got.BudgetSnapshot) != 0 {
		t.Fatalf("missing A60 additive fields must resolve to documented defaults: %#v", got)
	}
}

func TestRuntimeRecorderParsesA61TracingEvalAdditiveFields(t *testing.T) {
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
		RunID:   "run-a61-recorder",
		Time:    time.Now(),
		Payload: map[string]any{
			"status":               "success",
			"trace_export_status":  "degraded",
			"trace_schema_version": "otel_semconv.v1",
			"eval_suite_id":        "agent_eval.v1",
			"eval_summary": map[string]any{
				"task_success": map[string]any{"pass_rate": 0.94},
			},
			"eval_execution_mode": "distributed",
			"eval_job_id":         "eval-job-recorder-a61",
			"eval_shard_total":    8,
			"eval_resume_count":   2,
		},
	})

	items := mgr.RecentRuns(1)
	if len(items) != 1 {
		t.Fatalf("run records len = %d, want 1", len(items))
	}
	got := items[0]
	if got.TraceExportStatus != "degraded" ||
		got.TraceSchemaVersion != "otel_semconv.v1" ||
		got.EvalSuiteID != "agent_eval.v1" ||
		got.EvalExecutionMode != "distributed" ||
		got.EvalJobID != "eval-job-recorder-a61" ||
		got.EvalShardTotal != 8 ||
		got.EvalResumeCount != 2 {
		t.Fatalf("A61 additive field parse mismatch: %#v", got)
	}
	if summary, ok := got.EvalSummary["task_success"].(map[string]any); !ok || summary["pass_rate"] != 0.94 {
		t.Fatalf("A61 eval_summary parse mismatch: %#v", got.EvalSummary)
	}
}

func TestRuntimeRecorderA61ParserCompatibilityAdditiveNullableDefault(t *testing.T) {
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
		RunID:   "run-a61-compat",
		Time:    time.Now(),
		Payload: map[string]any{
			"status":           "success",
			"latency_ms":       int64(17),
			"a61_future_field": "ignore_me",
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
	if got.TraceExportStatus != "" ||
		got.TraceSchemaVersion != "" ||
		got.EvalSuiteID != "" ||
		len(got.EvalSummary) != 0 ||
		got.EvalExecutionMode != "" ||
		got.EvalJobID != "" ||
		got.EvalShardTotal != 0 ||
		got.EvalResumeCount != 0 {
		t.Fatalf("missing A61 additive fields must resolve to documented defaults: %#v", got)
	}
}

func TestRuntimeRecorderParsesA65HooksMiddlewareSkillAdditiveFields(t *testing.T) {
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
		RunID:   "run-a65-recorder",
		Time:    time.Now(),
		Payload: map[string]any{
			"status":                                "success",
			"hooks_enabled":                         true,
			"hooks_fail_mode":                       "degrade",
			"hooks_phases":                          []any{"before_reasoning", "after_reasoning"},
			"tool_middleware_enabled":               true,
			"tool_middleware_fail_mode":             "fail_fast",
			"skill_discovery_mode":                  "hybrid",
			"skill_discovery_roots":                 []any{"./skills", "./agents"},
			"skill_preprocess_enabled":              true,
			"skill_preprocess_phase":                "before_run_stream",
			"skill_preprocess_fail_mode":            "degrade",
			"skill_preprocess_status":               "success",
			"skill_preprocess_reason_code":          "skill_preprocess_failed",
			"skill_preprocess_spec_count":           3,
			"skill_bundle_prompt_mode":              "append",
			"skill_bundle_whitelist_mode":           "merge",
			"skill_bundle_conflict_policy":          "first_win",
			"skill_bundle_prompt_total":             2,
			"skill_bundle_whitelist_total":          4,
			"skill_bundle_whitelist_rejected_total": 1,
		},
	})

	items := mgr.RecentRuns(1)
	if len(items) != 1 {
		t.Fatalf("run records len = %d, want 1", len(items))
	}
	got := items[0]
	if !got.HooksEnabled ||
		got.HooksFailMode != "degrade" ||
		len(got.HooksPhases) != 2 ||
		got.HooksPhases[0] != "before_reasoning" ||
		got.HooksPhases[1] != "after_reasoning" ||
		!got.ToolMiddlewareEnabled ||
		got.ToolMiddlewareFailMode != "fail_fast" ||
		got.SkillDiscoveryMode != "hybrid" ||
		len(got.SkillDiscoveryRoots) != 2 ||
		got.SkillDiscoveryRoots[0] != "./skills" ||
		got.SkillDiscoveryRoots[1] != "./agents" ||
		!got.SkillPreprocessEnabled ||
		got.SkillPreprocessPhase != "before_run_stream" ||
		got.SkillPreprocessFailMode != "degrade" ||
		got.SkillPreprocessStatus != "success" ||
		got.SkillPreprocessReasonCode != "skill_preprocess_failed" ||
		got.SkillPreprocessSpecCount != 3 ||
		got.SkillBundlePromptMode != "append" ||
		got.SkillBundleWhitelistMode != "merge" ||
		got.SkillBundleConflictPolicy != "first_win" ||
		got.SkillBundlePromptTotal != 2 ||
		got.SkillBundleWhitelistTotal != 4 ||
		got.SkillBundleWhitelistRejectedTotal != 1 {
		t.Fatalf("A65 additive field parse mismatch: %#v", got)
	}
}

func TestRuntimeRecorderA65ParserCompatibilityAdditiveNullableDefault(t *testing.T) {
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
		RunID:   "run-a65-compat",
		Time:    time.Now(),
		Payload: map[string]any{
			"status":           "success",
			"latency_ms":       int64(21),
			"a65_future_field": "ignore_me",
		},
	})

	items := mgr.RecentRuns(1)
	if len(items) != 1 {
		t.Fatalf("run records len = %d, want 1", len(items))
	}
	got := items[0]
	if got.Status != "success" || got.LatencyMs != 21 {
		t.Fatalf("existing run fields should stay unchanged: %#v", got)
	}
	if got.HooksEnabled ||
		got.HooksFailMode != "" ||
		len(got.HooksPhases) != 0 ||
		got.ToolMiddlewareEnabled ||
		got.ToolMiddlewareFailMode != "" ||
		got.SkillDiscoveryMode != "" ||
		len(got.SkillDiscoveryRoots) != 0 ||
		got.SkillPreprocessEnabled ||
		got.SkillPreprocessPhase != "" ||
		got.SkillPreprocessFailMode != "" ||
		got.SkillPreprocessStatus != "" ||
		got.SkillPreprocessReasonCode != "" ||
		got.SkillPreprocessSpecCount != 0 ||
		got.SkillBundlePromptMode != "" ||
		got.SkillBundleWhitelistMode != "" ||
		got.SkillBundleConflictPolicy != "" ||
		got.SkillBundlePromptTotal != 0 ||
		got.SkillBundleWhitelistTotal != 0 ||
		got.SkillBundleWhitelistRejectedTotal != 0 {
		t.Fatalf("missing A65 additive fields must resolve to documented defaults: %#v", got)
	}
}

func TestRuntimeRecorderParsesA67PlanNotebookAdditiveFields(t *testing.T) {
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
		RunID:   "run-a67-recorder",
		Time:    time.Now(),
		Payload: map[string]any{
			"status":                   "success",
			"react_plan_id":            "run-a67-recorder",
			"react_plan_version":       4,
			"react_plan_change_total":  4,
			"react_plan_last_action":   "ReViSe",
			"react_plan_change_reason": "react_iteration_boundary",
			"react_plan_recover_count": 1,
			"react_plan_hook_status":   "DEGRADED",
		},
	})

	items := mgr.RecentRuns(1)
	if len(items) != 1 {
		t.Fatalf("run records len = %d, want 1", len(items))
	}
	got := items[0]
	if got.ReactPlanID != "run-a67-recorder" ||
		got.ReactPlanVersion != 4 ||
		got.ReactPlanChangeTotal != 4 ||
		got.ReactPlanLastAction != "revise" ||
		got.ReactPlanChangeReason != "react_iteration_boundary" ||
		got.ReactPlanRecoverCount != 1 ||
		got.ReactPlanHookStatus != "degraded" {
		t.Fatalf("A67 additive field parse mismatch: %#v", got)
	}
}

func TestRuntimeRecorderA67ParserCompatibilityAdditiveNullableDefault(t *testing.T) {
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
		RunID:   "run-a67-compat",
		Time:    time.Now(),
		Payload: map[string]any{
			"status":           "success",
			"latency_ms":       int64(31),
			"a67_future_field": "ignore_me",
		},
	})

	items := mgr.RecentRuns(1)
	if len(items) != 1 {
		t.Fatalf("run records len = %d, want 1", len(items))
	}
	got := items[0]
	if got.Status != "success" || got.LatencyMs != 31 {
		t.Fatalf("existing run fields should stay unchanged: %#v", got)
	}
	if got.ReactPlanID != "" ||
		got.ReactPlanVersion != 0 ||
		got.ReactPlanChangeTotal != 0 ||
		got.ReactPlanLastAction != "" ||
		got.ReactPlanChangeReason != "" ||
		got.ReactPlanRecoverCount != 0 ||
		got.ReactPlanHookStatus != "" {
		t.Fatalf("missing A67 additive fields must resolve to documented defaults: %#v", got)
	}
}

func TestRuntimeRecorderParsesA67ContextJITOrganizationAdditiveFields(t *testing.T) {
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
		RunID:   "run-a67-ctx-recorder",
		Time:    time.Now(),
		Payload: map[string]any{
			"status":                              "success",
			"context_ref_discover_count":          7,
			"context_ref_resolve_count":           5,
			"context_edit_estimated_saved_tokens": 88,
			"context_edit_gate_decision":          "allow.threshold_met",
			"context_swapback_relevance_score":    0.77,
			"context_lifecycle_tier_stats":        map[string]any{"hot": 2, "warm": 3, "cold": 1},
			"context_recap_source":                "task_aware.stage_actions.v1",
		},
	})

	items := mgr.RecentRuns(1)
	if len(items) != 1 {
		t.Fatalf("run records len = %d, want 1", len(items))
	}
	got := items[0]
	if got.ContextRefDiscoverCount != 7 ||
		got.ContextRefResolveCount != 5 ||
		got.ContextEditEstimatedSavedTokens != 88 ||
		got.ContextEditGateDecision != "allow.threshold_met" ||
		got.ContextSwapbackRelevanceScore != 0.77 ||
		got.ContextLifecycleTierStats["warm"] != 3 ||
		got.ContextRecapSource != "task_aware.stage_actions.v1" {
		t.Fatalf("A67-CTX additive field parse mismatch: %#v", got)
	}
}

func TestRuntimeRecorderA67ContextJITParserCompatibilityAdditiveNullableDefault(t *testing.T) {
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
		RunID:   "run-a67-ctx-compat",
		Time:    time.Now(),
		Payload: map[string]any{
			"status":               "success",
			"latency_ms":           int64(31),
			"a67_ctx_future_field": "ignore_me",
		},
	})

	items := mgr.RecentRuns(1)
	if len(items) != 1 {
		t.Fatalf("run records len = %d, want 1", len(items))
	}
	got := items[0]
	if got.Status != "success" || got.LatencyMs != 31 {
		t.Fatalf("existing run fields should stay unchanged: %#v", got)
	}
	if got.ContextRefDiscoverCount != 0 ||
		got.ContextRefResolveCount != 0 ||
		got.ContextEditEstimatedSavedTokens != 0 ||
		got.ContextEditGateDecision != "" ||
		got.ContextSwapbackRelevanceScore != 0 ||
		len(got.ContextLifecycleTierStats) != 0 ||
		got.ContextRecapSource != "" {
		t.Fatalf("missing A67-CTX additive fields must resolve to documented defaults: %#v", got)
	}
}

func TestRuntimeRecorderParsesA68RealtimeAdditiveFields(t *testing.T) {
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
		RunID:   "run-a68-recorder",
		Time:    time.Now(),
		Payload: map[string]any{
			"status":                           "success",
			"realtime_protocol_version":        "realtime_event_protocol.v1",
			"realtime_event_seq_max":           int64(18),
			"realtime_interrupt_total":         2,
			"realtime_resume_total":            1,
			"realtime_resume_source":           "cursor",
			"realtime_idempotency_dedup_total": 3,
			"realtime_last_error_code":         "realtime.sequence_gap",
		},
	})

	items := mgr.RecentRuns(1)
	if len(items) != 1 {
		t.Fatalf("run records len = %d, want 1", len(items))
	}
	got := items[0]
	if got.RealtimeProtocolVersion != "realtime_event_protocol.v1" ||
		got.RealtimeEventSeqMax != 18 ||
		got.RealtimeInterruptTotal != 2 ||
		got.RealtimeResumeTotal != 1 ||
		got.RealtimeResumeSource != "cursor" ||
		got.RealtimeIdempotencyDedupTotal != 3 ||
		got.RealtimeLastErrorCode != "realtime.sequence_gap" {
		t.Fatalf("A68 additive field parse mismatch: %#v", got)
	}
}

func TestRuntimeRecorderA68ParserCompatibilityAdditiveNullableDefault(t *testing.T) {
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
		RunID:   "run-a68-compat",
		Time:    time.Now(),
		Payload: map[string]any{
			"status":           "success",
			"latency_ms":       int64(31),
			"a68_future_field": "ignore_me",
		},
	})

	items := mgr.RecentRuns(1)
	if len(items) != 1 {
		t.Fatalf("run records len = %d, want 1", len(items))
	}
	got := items[0]
	if got.Status != "success" || got.LatencyMs != 31 {
		t.Fatalf("existing run fields should stay unchanged: %#v", got)
	}
	if got.RealtimeProtocolVersion != "" ||
		got.RealtimeEventSeqMax != 0 ||
		got.RealtimeInterruptTotal != 0 ||
		got.RealtimeResumeTotal != 0 ||
		got.RealtimeResumeSource != "" ||
		got.RealtimeIdempotencyDedupTotal != 0 ||
		got.RealtimeLastErrorCode != "" {
		t.Fatalf("missing A68 additive fields must resolve to documented defaults: %#v", got)
	}
}

func TestRuntimeRecorderA65ReasonTaxonomyDriftGuardCanonicalFallback(t *testing.T) {
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
		RunID:   "run-a65-reason-guard",
		Time:    time.Now(),
		Payload: map[string]any{
			"status":                       "failed",
			"skill_preprocess_status":      "failed",
			"skill_preprocess_reason_code": "skill.preprocess.custom_alias",
		},
	})

	items := mgr.RecentRuns(1)
	if len(items) != 1 {
		t.Fatalf("run records len = %d, want 1", len(items))
	}
	if items[0].SkillPreprocessReasonCode != "skill_preprocess_failed" {
		t.Fatalf("reason taxonomy drift should fallback to canonical code, got %#v", items[0].SkillPreprocessReasonCode)
	}
}

func TestRuntimeRecorderParsesA66SnapshotRestoreAdditiveFields(t *testing.T) {
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
		RunID:   "run-a66-recorder",
		Time:    time.Now(),
		Payload: map[string]any{
			"status":                      "failed",
			"state_snapshot_version":      "state_session_snapshot.v1",
			"state_restore_action":        "compatible_exact_restore",
			"state_restore_conflict_code": "state_snapshot_compat_window_exceeded",
			"state_restore_source":        "Composer",
		},
	})

	items := mgr.RecentRuns(1)
	if len(items) != 1 {
		t.Fatalf("run records len = %d, want 1", len(items))
	}
	got := items[0]
	if got.StateSnapshotVersion != "state_session_snapshot.v1" ||
		got.StateRestoreAction != "compatible_exact_restore" ||
		got.StateRestoreConflictCode != "state_snapshot_compat_window_exceeded" ||
		got.StateRestoreSource != "composer" {
		t.Fatalf("A66 additive field parse mismatch: %#v", got)
	}
}

func TestRuntimeRecorderA66ParserCompatibilityAdditiveNullableDefault(t *testing.T) {
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
		RunID:   "run-a66-compat",
		Time:    time.Now(),
		Payload: map[string]any{
			"status":           "success",
			"latency_ms":       int64(25),
			"a66_future_field": "ignore_me",
		},
	})

	items := mgr.RecentRuns(1)
	if len(items) != 1 {
		t.Fatalf("run records len = %d, want 1", len(items))
	}
	got := items[0]
	if got.Status != "success" || got.LatencyMs != 25 {
		t.Fatalf("existing run fields should stay unchanged: %#v", got)
	}
	if got.StateSnapshotVersion != "" ||
		got.StateRestoreAction != "" ||
		got.StateRestoreConflictCode != "" ||
		got.StateRestoreSource != "" {
		t.Fatalf("missing A66 additive fields must resolve to documented defaults: %#v", got)
	}
}

func TestRuntimeRecorderA66RestoreTaxonomyDriftGuardCanonicalFallback(t *testing.T) {
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
		RunID:   "run-a66-taxonomy-guard",
		Time:    time.Now(),
		Payload: map[string]any{
			"status":                      "failed",
			"state_restore_action":        "compat.restore.alias",
			"state_restore_conflict_code": "snapshot.custom.conflict.alias",
			"state_restore_source":        "Composer",
		},
	})

	items := mgr.RecentRuns(1)
	if len(items) != 1 {
		t.Fatalf("run records len = %d, want 1", len(items))
	}
	got := items[0]
	if got.StateRestoreAction != "compatible_bounded_restore" {
		t.Fatalf("restore action taxonomy drift should fallback to canonical action, got %#v", got.StateRestoreAction)
	}
	if got.StateRestoreConflictCode != "state_snapshot_invalid_payload" {
		t.Fatalf("restore conflict taxonomy drift should fallback to canonical conflict code, got %#v", got.StateRestoreConflictCode)
	}
	if got.StateRestoreSource != "composer" {
		t.Fatalf("state restore source should normalize to lower-case, got %#v", got.StateRestoreSource)
	}
}

func TestRuntimeRecorderA58ParserCompatibilityAdditiveNullableDefault(t *testing.T) {
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
		RunID:   "run-a58-compat",
		Time:    time.Now(),
		Payload: map[string]any{
			"status":           "success",
			"latency_ms":       int64(9),
			"a58_future_field": "ignore_me",
		},
	})

	items := mgr.RecentRuns(1)
	if len(items) != 1 {
		t.Fatalf("run records len = %d, want 1", len(items))
	}
	got := items[0]
	if got.Status != "success" || got.LatencyMs != 9 {
		t.Fatalf("existing run fields should stay unchanged: %#v", got)
	}
	if got.PolicyPrecedenceVersion != "" ||
		got.WinnerStage != "" ||
		got.DenySource != "" ||
		got.TieBreakReason != "" ||
		len(got.PolicyDecisionPath) != 0 {
		t.Fatalf("missing A58 additive fields must resolve to documented defaults: %#v", got)
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
