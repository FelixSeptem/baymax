package diagnostics

import (
	"crypto/sha1"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"reflect"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/FelixSeptem/baymax/runtime/security/redaction"
)

type CallRecord struct {
	Time           time.Time `json:"time"`
	Component      string    `json:"component"`
	Transport      string    `json:"transport,omitempty"`
	Profile        string    `json:"profile,omitempty"`
	RunID          string    `json:"run_id,omitempty"`
	CallID         string    `json:"call_id,omitempty"`
	Name           string    `json:"name,omitempty"`
	Action         string    `json:"action,omitempty"`
	LatencyMs      int64     `json:"latency_ms"`
	RetryCount     int       `json:"retry_count"`
	ReconnectCount int       `json:"reconnect_count"`
	ErrorClass     string    `json:"error_class,omitempty"`
}

type RuntimePolicyDecisionPathEntry struct {
	Stage    string `json:"stage"`
	Code     string `json:"code,omitempty"`
	Source   string `json:"source,omitempty"`
	Decision string `json:"decision,omitempty"`
}

type RunRecord struct {
	Time                                        time.Time                         `json:"time"`
	RunID                                       string                            `json:"run_id"`
	Status                                      string                            `json:"status,omitempty"`
	Iterations                                  int                               `json:"iterations"`
	ToolCalls                                   int                               `json:"tool_calls"`
	LatencyMs                                   int64                             `json:"latency_ms"`
	ErrorClass                                  string                            `json:"error_class,omitempty"`
	PolicyKind                                  string                            `json:"policy_kind,omitempty"`
	NamespaceTool                               string                            `json:"namespace_tool,omitempty"`
	FilterStage                                 string                            `json:"filter_stage,omitempty"`
	Decision                                    string                            `json:"decision,omitempty"`
	ReasonCode                                  string                            `json:"reason_code,omitempty"`
	Severity                                    string                            `json:"severity,omitempty"`
	AlertDispatchStatus                         string                            `json:"alert_dispatch_status,omitempty"`
	AlertDispatchFailureReason                  string                            `json:"alert_dispatch_failure_reason,omitempty"`
	AlertDeliveryMode                           string                            `json:"alert_delivery_mode,omitempty"`
	AlertRetryCount                             int                               `json:"alert_retry_count,omitempty"`
	AlertQueueDropped                           bool                              `json:"alert_queue_dropped,omitempty"`
	AlertQueueDropCount                         int                               `json:"alert_queue_drop_count,omitempty"`
	AlertCircuitState                           string                            `json:"alert_circuit_state,omitempty"`
	AlertCircuitOpenReason                      string                            `json:"alert_circuit_open_reason,omitempty"`
	ModelProvider                               string                            `json:"model_provider,omitempty"`
	FallbackUsed                                bool                              `json:"fallback_used,omitempty"`
	FallbackInitial                             string                            `json:"fallback_initial,omitempty"`
	FallbackPath                                string                            `json:"fallback_path,omitempty"`
	RequiredCapabilities                        string                            `json:"required_capabilities,omitempty"`
	FallbackReason                              string                            `json:"fallback_reason,omitempty"`
	MemoryMode                                  string                            `json:"memory_mode,omitempty"`
	MemoryProvider                              string                            `json:"memory_provider,omitempty"`
	MemoryProfile                               string                            `json:"memory_profile,omitempty"`
	MemoryContractVersion                       string                            `json:"memory_contract_version,omitempty"`
	MemoryQueryTotal                            int                               `json:"memory_query_total,omitempty"`
	MemoryUpsertTotal                           int                               `json:"memory_upsert_total,omitempty"`
	MemoryDeleteTotal                           int                               `json:"memory_delete_total,omitempty"`
	MemoryErrorTotal                            int                               `json:"memory_error_total,omitempty"`
	MemoryFallbackTotal                         int                               `json:"memory_fallback_total,omitempty"`
	MemoryFallbackReasonCode                    string                            `json:"memory_fallback_reason_code,omitempty"`
	MemoryLatencyMsP95                          int64                             `json:"memory_latency_ms_p95,omitempty"`
	MemoryScopeSelected                         string                            `json:"memory_scope_selected,omitempty"`
	MemoryBudgetUsed                            int                               `json:"memory_budget_used,omitempty"`
	MemoryHits                                  int                               `json:"memory_hits,omitempty"`
	MemoryRerankStats                           map[string]int                    `json:"memory_rerank_stats,omitempty"`
	MemoryLifecycleAction                       string                            `json:"memory_lifecycle_action,omitempty"`
	ObservabilityExportProfile                  string                            `json:"observability_export_profile,omitempty"`
	ObservabilityExportStatus                   string                            `json:"observability_export_status,omitempty"`
	ObservabilityExportErrorTotal               int                               `json:"observability_export_error_total,omitempty"`
	ObservabilityExportDropTotal                int                               `json:"observability_export_drop_total,omitempty"`
	ObservabilityExportQueueDepthPeak           int                               `json:"observability_export_queue_depth_peak,omitempty"`
	DiagnosticsBundleTotal                      int                               `json:"diagnostics_bundle_total,omitempty"`
	DiagnosticsBundleLastStatus                 string                            `json:"diagnostics_bundle_last_status,omitempty"`
	DiagnosticsBundleLastReasonCode             string                            `json:"diagnostics_bundle_last_reason_code,omitempty"`
	DiagnosticsBundleLastSchemaVersion          string                            `json:"diagnostics_bundle_last_schema_version,omitempty"`
	SandboxMode                                 string                            `json:"sandbox_mode,omitempty"`
	SandboxBackend                              string                            `json:"sandbox_backend,omitempty"`
	SandboxProfile                              string                            `json:"sandbox_profile,omitempty"`
	SandboxSessionMode                          string                            `json:"sandbox_session_mode,omitempty"`
	SandboxRequiredCapabilities                 []string                          `json:"sandbox_required_capabilities,omitempty"`
	SandboxDecision                             string                            `json:"sandbox_decision,omitempty"`
	SandboxReasonCode                           string                            `json:"sandbox_reason_code,omitempty"`
	SandboxEgressAction                         string                            `json:"sandbox_egress_action,omitempty"`
	SandboxEgressViolationTotal                 int                               `json:"sandbox_egress_violation_total,omitempty"`
	SandboxEgressPolicySource                   string                            `json:"sandbox_egress_policy_source,omitempty"`
	SandboxFallbackUsed                         bool                              `json:"sandbox_fallback_used,omitempty"`
	SandboxFallbackReason                       string                            `json:"sandbox_fallback_reason,omitempty"`
	SandboxTimeoutTotal                         int                               `json:"sandbox_timeout_total,omitempty"`
	SandboxLaunchFailedTotal                    int                               `json:"sandbox_launch_failed_total,omitempty"`
	SandboxCapabilityMismatchTotal              int                               `json:"sandbox_capability_mismatch_total,omitempty"`
	SandboxQueueWaitMsP95                       int64                             `json:"sandbox_queue_wait_ms_p95,omitempty"`
	SandboxExecLatencyMsP95                     int64                             `json:"sandbox_exec_latency_ms_p95,omitempty"`
	SandboxExitCodeLast                         int                               `json:"sandbox_exit_code_last,omitempty"`
	SandboxOOMTotal                             int                               `json:"sandbox_oom_total,omitempty"`
	SandboxResourceCPUMsTotal                   int64                             `json:"sandbox_resource_cpu_ms_total,omitempty"`
	SandboxResourceMemoryPeakBytesP95           int64                             `json:"sandbox_resource_memory_peak_bytes_p95,omitempty"`
	SandboxRolloutPhase                         string                            `json:"sandbox_rollout_phase,omitempty"`
	SandboxRolloutEffectiveRatio                float64                           `json:"sandbox_rollout_effective_ratio,omitempty"`
	SandboxHealthBudgetStatus                   string                            `json:"sandbox_health_budget_status,omitempty"`
	SandboxHealthBudgetBreachTotal              int                               `json:"sandbox_health_budget_breach_total,omitempty"`
	SandboxFreezeState                          bool                              `json:"sandbox_freeze_state,omitempty"`
	SandboxFreezeReasonCode                     string                            `json:"sandbox_freeze_reason_code,omitempty"`
	SandboxCapacityAction                       string                            `json:"sandbox_capacity_action,omitempty"`
	SandboxCapacityQueueDepth                   int                               `json:"sandbox_capacity_queue_depth,omitempty"`
	SandboxCapacityInflight                     int                               `json:"sandbox_capacity_inflight,omitempty"`
	PrefixHash                                  string                            `json:"prefix_hash,omitempty"`
	AssembleLatencyMs                           int64                             `json:"assemble_latency_ms,omitempty"`
	AssembleStatus                              string                            `json:"assemble_status,omitempty"`
	GuardViolation                              string                            `json:"guard_violation,omitempty"`
	AssembleStageStatus                         string                            `json:"assemble_stage_status,omitempty"`
	Stage2SkipReason                            string                            `json:"stage2_skip_reason,omitempty"`
	Stage2RouterMode                            string                            `json:"stage2_router_mode,omitempty"`
	Stage2RouterDecision                        string                            `json:"stage2_router_decision,omitempty"`
	Stage2RouterReason                          string                            `json:"stage2_router_reason,omitempty"`
	Stage2RouterLatencyMs                       int64                             `json:"stage2_router_latency_ms,omitempty"`
	Stage2RouterError                           string                            `json:"stage2_router_error,omitempty"`
	Stage1LatencyMs                             int64                             `json:"stage1_latency_ms,omitempty"`
	Stage2LatencyMs                             int64                             `json:"stage2_latency_ms,omitempty"`
	Stage2Provider                              string                            `json:"stage2_provider,omitempty"`
	Stage2Profile                               string                            `json:"stage2_profile,omitempty"`
	Stage2TemplateProfile                       string                            `json:"stage2_template_profile,omitempty"`
	Stage2TemplateResolutionSource              string                            `json:"stage2_template_resolution_source,omitempty"`
	Stage2HintApplied                           bool                              `json:"stage2_hint_applied,omitempty"`
	Stage2HintMismatchReason                    string                            `json:"stage2_hint_mismatch_reason,omitempty"`
	Stage2HitCount                              int                               `json:"stage2_hit_count,omitempty"`
	Stage2Source                                string                            `json:"stage2_source,omitempty"`
	Stage2Reason                                string                            `json:"stage2_reason,omitempty"`
	Stage2ReasonCode                            string                            `json:"stage2_reason_code,omitempty"`
	Stage2ErrorLayer                            string                            `json:"stage2_error_layer,omitempty"`
	CA3PressureZone                             string                            `json:"ca3_pressure_zone,omitempty"`
	CA3PressureReason                           string                            `json:"ca3_pressure_reason,omitempty"`
	CA3PressureTrigger                          string                            `json:"ca3_pressure_trigger,omitempty"`
	CA3ZoneResidencyMs                          map[string]int64                  `json:"ca3_zone_residency_ms,omitempty"`
	CA3TriggerCounts                            map[string]int                    `json:"ca3_trigger_counts,omitempty"`
	CA3CompressionRatio                         float64                           `json:"ca3_compression_ratio,omitempty"`
	CA3SpillCount                               int                               `json:"ca3_spill_count,omitempty"`
	CA3SwapBackCount                            int                               `json:"ca3_swap_back_count,omitempty"`
	CA3CompactionMode                           string                            `json:"ca3_compaction_mode,omitempty"`
	CA3CompactionFallback                       bool                              `json:"ca3_compaction_fallback,omitempty"`
	CA3CompactionFallbackReason                 string                            `json:"ca3_compaction_fallback_reason,omitempty"`
	CA3CompactionQualityScore                   float64                           `json:"ca3_compaction_quality_score,omitempty"`
	CA3CompactionQualityReason                  string                            `json:"ca3_compaction_quality_reason,omitempty"`
	CA3CompactionEmbeddingProvider              string                            `json:"ca3_compaction_embedding_provider,omitempty"`
	CA3CompactionEmbeddingSimilarity            float64                           `json:"ca3_compaction_embedding_similarity,omitempty"`
	CA3CompactionEmbeddingContribution          float64                           `json:"ca3_compaction_embedding_contribution,omitempty"`
	CA3CompactionEmbeddingStatus                string                            `json:"ca3_compaction_embedding_status,omitempty"`
	CA3CompactionEmbeddingFallbackReason        string                            `json:"ca3_compaction_embedding_fallback_reason,omitempty"`
	CA3CompactionRerankerUsed                   bool                              `json:"ca3_compaction_reranker_used,omitempty"`
	CA3CompactionRerankerProvider               string                            `json:"ca3_compaction_reranker_provider,omitempty"`
	CA3CompactionRerankerModel                  string                            `json:"ca3_compaction_reranker_model,omitempty"`
	CA3CompactionRerankerThresholdSource        string                            `json:"ca3_compaction_reranker_threshold_source,omitempty"`
	CA3CompactionRerankerThresholdHit           bool                              `json:"ca3_compaction_reranker_threshold_hit,omitempty"`
	CA3CompactionRerankerFallbackReason         string                            `json:"ca3_compaction_reranker_fallback_reason,omitempty"`
	CA3CompactionRerankerProfileVersion         string                            `json:"ca3_compaction_reranker_profile_version,omitempty"`
	CA3CompactionRerankerRolloutHit             bool                              `json:"ca3_compaction_reranker_rollout_hit,omitempty"`
	CA3CompactionRerankerThresholdDrift         float64                           `json:"ca3_compaction_reranker_threshold_drift,omitempty"`
	CA3RetainedEvidence                         int                               `json:"ca3_compaction_retained_evidence_count,omitempty"`
	ContextRefDiscoverCount                     int                               `json:"context_ref_discover_count,omitempty"`
	ContextRefResolveCount                      int                               `json:"context_ref_resolve_count,omitempty"`
	ContextEditEstimatedSavedTokens             int                               `json:"context_edit_estimated_saved_tokens,omitempty"`
	ContextEditGateDecision                     string                            `json:"context_edit_gate_decision,omitempty"`
	ContextSwapbackRelevanceScore               float64                           `json:"context_swapback_relevance_score,omitempty"`
	ContextLifecycleTierStats                   map[string]int                    `json:"context_lifecycle_tier_stats,omitempty"`
	ContextRecapSource                          string                            `json:"context_recap_source,omitempty"`
	RecapStatus                                 string                            `json:"recap_status,omitempty"`
	TeamID                                      string                            `json:"team_id,omitempty"`
	TeamStrategy                                string                            `json:"team_strategy,omitempty"`
	TeamTaskTotal                               int                               `json:"team_task_total,omitempty"`
	TeamTaskFailed                              int                               `json:"team_task_failed,omitempty"`
	TeamTaskCanceled                            int                               `json:"team_task_canceled,omitempty"`
	TeamRemoteTaskTotal                         int                               `json:"team_remote_task_total,omitempty"`
	TeamRemoteTaskFailed                        int                               `json:"team_remote_task_failed,omitempty"`
	WorkflowID                                  string                            `json:"workflow_id,omitempty"`
	WorkflowStatus                              string                            `json:"workflow_status,omitempty"`
	WorkflowStepTotal                           int                               `json:"workflow_step_total,omitempty"`
	WorkflowStepFailed                          int                               `json:"workflow_step_failed,omitempty"`
	WorkflowRemoteStepTotal                     int                               `json:"workflow_remote_step_total,omitempty"`
	WorkflowRemoteStepFailed                    int                               `json:"workflow_remote_step_failed,omitempty"`
	WorkflowSubgraphExpansionTotal              int                               `json:"workflow_subgraph_expansion_total,omitempty"`
	WorkflowConditionTemplateTotal              int                               `json:"workflow_condition_template_total,omitempty"`
	WorkflowGraphCompileFailed                  bool                              `json:"workflow_graph_compile_failed,omitempty"`
	WorkflowResumeCount                         int                               `json:"workflow_resume_count,omitempty"`
	TaskID                                      string                            `json:"task_id,omitempty"`
	A2ATaskTotal                                int                               `json:"a2a_task_total,omitempty"`
	A2ATaskFailed                               int                               `json:"a2a_task_failed,omitempty"`
	PeerID                                      string                            `json:"peer_id,omitempty"`
	A2AErrorLayer                               string                            `json:"a2a_error_layer,omitempty"`
	A2ADeliveryMode                             string                            `json:"a2a_delivery_mode,omitempty"`
	A2ADeliveryFallbackUsed                     bool                              `json:"a2a_delivery_fallback_used,omitempty"`
	A2ADeliveryFallbackReason                   string                            `json:"a2a_delivery_fallback_reason,omitempty"`
	A2AVersionLocal                             string                            `json:"a2a_version_local,omitempty"`
	A2AVersionPeer                              string                            `json:"a2a_version_peer,omitempty"`
	A2AVersionNegotiationResult                 string                            `json:"a2a_version_negotiation_result,omitempty"`
	A2AAsyncReportTotal                         int                               `json:"a2a_async_report_total,omitempty"`
	A2AAsyncReportFailed                        int                               `json:"a2a_async_report_failed,omitempty"`
	A2AAsyncReportRetryTotal                    int                               `json:"a2a_async_report_retry_total,omitempty"`
	A2AAsyncReportDedupTotal                    int                               `json:"a2a_async_report_dedup_total,omitempty"`
	AsyncAwaitTotal                             int                               `json:"async_await_total,omitempty"`
	AsyncTimeoutTotal                           int                               `json:"async_timeout_total,omitempty"`
	AsyncLateReportTotal                        int                               `json:"async_late_report_total,omitempty"`
	AsyncReportDedupTotal                       int                               `json:"async_report_dedup_total,omitempty"`
	AsyncReconcilePollTotal                     int                               `json:"async_reconcile_poll_total,omitempty"`
	AsyncReconcileTerminalByPollTotal           int                               `json:"async_reconcile_terminal_by_poll_total,omitempty"`
	AsyncReconcileErrorTotal                    int                               `json:"async_reconcile_error_total,omitempty"`
	AsyncTerminalConflictTotal                  int                               `json:"async_terminal_conflict_total,omitempty"`
	ComposerManaged                             bool                              `json:"composer_managed,omitempty"`
	SchedulerBackend                            string                            `json:"scheduler_backend,omitempty"`
	SchedulerQoSMode                            string                            `json:"scheduler_qos_mode,omitempty"`
	SchedulerBackendFallback                    bool                              `json:"scheduler_backend_fallback,omitempty"`
	SchedulerBackendFallbackReason              string                            `json:"scheduler_backend_fallback_reason,omitempty"`
	SchedulerQueueTotal                         int                               `json:"scheduler_queue_total,omitempty"`
	SchedulerClaimTotal                         int                               `json:"scheduler_claim_total,omitempty"`
	SchedulerReclaimTotal                       int                               `json:"scheduler_reclaim_total,omitempty"`
	SchedulerPriorityClaimTotal                 int                               `json:"scheduler_priority_claim_total,omitempty"`
	SchedulerFairnessYieldTotal                 int                               `json:"scheduler_fairness_yield_total,omitempty"`
	SchedulerRetryBackoffTotal                  int                               `json:"scheduler_retry_backoff_total,omitempty"`
	SchedulerDeadLetterTotal                    int                               `json:"scheduler_dead_letter_total,omitempty"`
	SchedulerDelayedTaskTotal                   int                               `json:"scheduler_delayed_task_total,omitempty"`
	SchedulerDelayedClaimTotal                  int                               `json:"scheduler_delayed_claim_total,omitempty"`
	SchedulerDelayedWaitMsP95                   int64                             `json:"scheduler_delayed_wait_ms_p95,omitempty"`
	TaskBoardManualControlTotal                 int                               `json:"task_board_manual_control_total,omitempty"`
	TaskBoardManualControlSuccessTotal          int                               `json:"task_board_manual_control_success_total,omitempty"`
	TaskBoardManualControlRejectedTotal         int                               `json:"task_board_manual_control_rejected_total,omitempty"`
	TaskBoardManualControlDedupTotal            int                               `json:"task_board_manual_control_idempotent_dedup_total,omitempty"`
	TaskBoardManualControlByAction              map[string]int                    `json:"task_board_manual_control_by_action,omitempty"`
	TaskBoardManualControlByReason              map[string]int                    `json:"task_board_manual_control_by_reason,omitempty"`
	SubagentChildTotal                          int                               `json:"subagent_child_total,omitempty"`
	SubagentChildFailed                         int                               `json:"subagent_child_failed,omitempty"`
	SubagentBudgetRejectTotal                   int                               `json:"subagent_budget_reject_total,omitempty"`
	CollabHandoffTotal                          int                               `json:"collab_handoff_total,omitempty"`
	CollabDelegationTotal                       int                               `json:"collab_delegation_total,omitempty"`
	CollabAggregationTotal                      int                               `json:"collab_aggregation_total,omitempty"`
	CollabAggregationStrategy                   string                            `json:"collab_aggregation_strategy,omitempty"`
	CollabFailFastTotal                         int                               `json:"collab_fail_fast_total,omitempty"`
	CollabRetryTotal                            int                               `json:"collab_retry_total,omitempty"`
	CollabRetrySuccessTotal                     int                               `json:"collab_retry_success_total,omitempty"`
	CollabRetryExhaustedTotal                   int                               `json:"collab_retry_exhausted_total,omitempty"`
	RecoveryEnabled                             bool                              `json:"recovery_enabled,omitempty"`
	RecoveryResumeBoundary                      string                            `json:"recovery_resume_boundary,omitempty"`
	RecoveryInflightPolicy                      string                            `json:"recovery_inflight_policy,omitempty"`
	RecoveryRecovered                           bool                              `json:"recovery_recovered,omitempty"`
	RecoveryReplayTotal                         int                               `json:"recovery_replay_total,omitempty"`
	RecoveryTimeoutReentryTotal                 int                               `json:"recovery_timeout_reentry_total,omitempty"`
	RecoveryTimeoutReentryExhaustedTotal        int                               `json:"recovery_timeout_reentry_exhausted_total,omitempty"`
	RecoveryConflict                            bool                              `json:"recovery_conflict,omitempty"`
	RecoveryConflictCode                        string                            `json:"recovery_conflict_code,omitempty"`
	StateSnapshotVersion                        string                            `json:"state_snapshot_version,omitempty"`
	StateRestoreAction                          string                            `json:"state_restore_action,omitempty"`
	StateRestoreConflictCode                    string                            `json:"state_restore_conflict_code,omitempty"`
	StateRestoreSource                          string                            `json:"state_restore_source,omitempty"`
	RecoveryFallbackUsed                        bool                              `json:"recovery_fallback_used,omitempty"`
	RecoveryFallbackReason                      string                            `json:"recovery_fallback_reason,omitempty"`
	RuntimeReadinessStatus                      string                            `json:"runtime_readiness_status,omitempty"`
	RuntimeReadinessFindingTotal                int                               `json:"runtime_readiness_finding_total,omitempty"`
	RuntimeReadinessBlockingTotal               int                               `json:"runtime_readiness_blocking_total,omitempty"`
	RuntimeReadinessDegradedTotal               int                               `json:"runtime_readiness_degraded_total,omitempty"`
	RuntimePrimaryDomain                        string                            `json:"runtime_primary_domain,omitempty"`
	RuntimePrimaryCode                          string                            `json:"runtime_primary_code,omitempty"`
	RuntimePrimarySource                        string                            `json:"runtime_primary_source,omitempty"`
	RuntimePrimaryConflictTotal                 int                               `json:"runtime_primary_conflict_total,omitempty"`
	RuntimeSecondaryReasonCodes                 []string                          `json:"runtime_secondary_reason_codes,omitempty"`
	RuntimeSecondaryReasonCount                 int                               `json:"runtime_secondary_reason_count,omitempty"`
	RuntimeArbitrationRuleVersion               string                            `json:"runtime_arbitration_rule_version,omitempty"`
	RuntimeArbitrationRuleRequestedVersion      string                            `json:"runtime_arbitration_rule_requested_version,omitempty"`
	RuntimeArbitrationRuleEffectiveVersion      string                            `json:"runtime_arbitration_rule_effective_version,omitempty"`
	RuntimeArbitrationRuleVersionSource         string                            `json:"runtime_arbitration_rule_version_source,omitempty"`
	RuntimeArbitrationRulePolicyAction          string                            `json:"runtime_arbitration_rule_policy_action,omitempty"`
	RuntimeArbitrationRuleUnsupportedTotal      int                               `json:"runtime_arbitration_rule_unsupported_total,omitempty"`
	RuntimeArbitrationRuleMismatchTotal         int                               `json:"runtime_arbitration_rule_mismatch_total,omitempty"`
	RuntimeRemediationHintCode                  string                            `json:"runtime_remediation_hint_code,omitempty"`
	RuntimeRemediationHintDomain                string                            `json:"runtime_remediation_hint_domain,omitempty"`
	RuntimeReadinessPrimaryCode                 string                            `json:"runtime_readiness_primary_code,omitempty"`
	RuntimeReadinessAdmissionTotal              int                               `json:"runtime_readiness_admission_total,omitempty"`
	RuntimeReadinessAdmissionBlockedTotal       int                               `json:"runtime_readiness_admission_blocked_total,omitempty"`
	RuntimeReadinessAdmissionDegradedAllowTotal int                               `json:"runtime_readiness_admission_degraded_allow_total,omitempty"`
	RuntimeReadinessAdmissionBypassTotal        int                               `json:"runtime_readiness_admission_bypass_total,omitempty"`
	RuntimeReadinessAdmissionMode               string                            `json:"runtime_readiness_admission_mode,omitempty"`
	RuntimeReadinessAdmissionPrimaryCode        string                            `json:"runtime_readiness_admission_primary_code,omitempty"`
	TraceExportStatus                           string                            `json:"trace_export_status,omitempty"`
	TraceSchemaVersion                          string                            `json:"trace_schema_version,omitempty"`
	EvalSuiteID                                 string                            `json:"eval_suite_id,omitempty"`
	EvalSummary                                 map[string]any                    `json:"eval_summary,omitempty"`
	EvalExecutionMode                           string                            `json:"eval_execution_mode,omitempty"`
	EvalJobID                                   string                            `json:"eval_job_id,omitempty"`
	EvalShardTotal                              int                               `json:"eval_shard_total,omitempty"`
	EvalResumeCount                             int                               `json:"eval_resume_count,omitempty"`
	BudgetSnapshot                              map[string]any                    `json:"budget_snapshot,omitempty"`
	BudgetDecision                              string                            `json:"budget_decision,omitempty"`
	DegradeAction                               string                            `json:"degrade_action,omitempty"`
	PolicyPrecedenceVersion                     string                            `json:"policy_precedence_version,omitempty"`
	WinnerStage                                 string                            `json:"winner_stage,omitempty"`
	DenySource                                  string                            `json:"deny_source,omitempty"`
	TieBreakReason                              string                            `json:"tie_break_reason,omitempty"`
	PolicyDecisionPath                          []RuntimePolicyDecisionPathEntry  `json:"policy_decision_path,omitempty"`
	AdapterAllowlistDecision                    string                            `json:"adapter_allowlist_decision,omitempty"`
	AdapterAllowlistBlockTotal                  int                               `json:"adapter_allowlist_block_total,omitempty"`
	AdapterAllowlistPrimaryCode                 string                            `json:"adapter_allowlist_primary_code,omitempty"`
	AdapterHealthStatus                         string                            `json:"adapter_health_status,omitempty"`
	AdapterHealthProbeTotal                     int                               `json:"adapter_health_probe_total,omitempty"`
	AdapterHealthDegradedTotal                  int                               `json:"adapter_health_degraded_total,omitempty"`
	AdapterHealthUnavailableTotal               int                               `json:"adapter_health_unavailable_total,omitempty"`
	AdapterHealthPrimaryCode                    string                            `json:"adapter_health_primary_code,omitempty"`
	AdapterHealthBackoffAppliedTotal            int                               `json:"adapter_health_backoff_applied_total,omitempty"`
	AdapterHealthCircuitOpenTotal               int                               `json:"adapter_health_circuit_open_total,omitempty"`
	AdapterHealthCircuitHalfOpenTotal           int                               `json:"adapter_health_circuit_half_open_total,omitempty"`
	AdapterHealthCircuitRecoverTotal            int                               `json:"adapter_health_circuit_recover_total,omitempty"`
	AdapterHealthCircuitState                   string                            `json:"adapter_health_circuit_state,omitempty"`
	AdapterHealthGovernancePrimaryCode          string                            `json:"adapter_health_governance_primary_code,omitempty"`
	EffectiveOperationProfile                   string                            `json:"effective_operation_profile,omitempty"`
	TimeoutResolutionSource                     string                            `json:"timeout_resolution_source,omitempty"`
	TimeoutResolutionTrace                      string                            `json:"timeout_resolution_trace,omitempty"`
	TimeoutParentBudgetClampTotal               int                               `json:"timeout_parent_budget_clamp_total,omitempty"`
	TimeoutParentBudgetRejectTotal              int                               `json:"timeout_parent_budget_reject_total,omitempty"`
	GateChecks                                  int                               `json:"gate_checks,omitempty"`
	GateDeniedCount                             int                               `json:"gate_denied_count,omitempty"`
	GateTimeoutCount                            int                               `json:"gate_timeout_count,omitempty"`
	GateRuleHitCount                            int                               `json:"gate_rule_hit_count,omitempty"`
	GateRuleLastID                              string                            `json:"gate_rule_last_id,omitempty"`
	AwaitCount                                  int                               `json:"await_count,omitempty"`
	ResumeCount                                 int                               `json:"resume_count,omitempty"`
	CancelByUserCount                           int                               `json:"cancel_by_user_count,omitempty"`
	CancelPropagated                            int                               `json:"cancel_propagated_count,omitempty"`
	BackpressureDrop                            int                               `json:"backpressure_drop_count,omitempty"`
	BackpressureDropByPhase                     map[string]int                    `json:"backpressure_drop_count_by_phase,omitempty"`
	InflightPeak                                int                               `json:"inflight_peak,omitempty"`
	ReactEnabled                                bool                              `json:"react_enabled,omitempty"`
	ReactIterationTotal                         int                               `json:"react_iteration_total,omitempty"`
	ReactToolCallTotal                          int                               `json:"react_tool_call_total,omitempty"`
	ReactToolCallBudgetHitTotal                 int                               `json:"react_tool_call_budget_hit_total,omitempty"`
	ReactIterationBudgetHitTotal                int                               `json:"react_iteration_budget_hit_total,omitempty"`
	ReactTerminationReason                      string                            `json:"react_termination_reason,omitempty"`
	ReactStreamDispatchEnabled                  bool                              `json:"react_stream_dispatch_enabled,omitempty"`
	ReactPlanID                                 string                            `json:"react_plan_id,omitempty"`
	ReactPlanVersion                            int                               `json:"react_plan_version,omitempty"`
	ReactPlanChangeTotal                        int                               `json:"react_plan_change_total,omitempty"`
	ReactPlanLastAction                         string                            `json:"react_plan_last_action,omitempty"`
	ReactPlanChangeReason                       string                            `json:"react_plan_change_reason,omitempty"`
	ReactPlanRecoverCount                       int                               `json:"react_plan_recover_count,omitempty"`
	ReactPlanHookStatus                         string                            `json:"react_plan_hook_status,omitempty"`
	RealtimeProtocolVersion                     string                            `json:"realtime_protocol_version,omitempty"`
	RealtimeEventSeqMax                         int64                             `json:"realtime_event_seq_max,omitempty"`
	RealtimeInterruptTotal                      int                               `json:"realtime_interrupt_total,omitempty"`
	RealtimeResumeTotal                         int                               `json:"realtime_resume_total,omitempty"`
	RealtimeResumeSource                        string                            `json:"realtime_resume_source,omitempty"`
	RealtimeIdempotencyDedupTotal               int                               `json:"realtime_idempotency_dedup_total,omitempty"`
	RealtimeLastErrorCode                       string                            `json:"realtime_last_error_code,omitempty"`
	HooksEnabled                                bool                              `json:"hooks_enabled,omitempty"`
	HooksFailMode                               string                            `json:"hooks_fail_mode,omitempty"`
	HooksPhases                                 []string                          `json:"hooks_phases,omitempty"`
	ToolMiddlewareEnabled                       bool                              `json:"tool_middleware_enabled,omitempty"`
	ToolMiddlewareFailMode                      string                            `json:"tool_middleware_fail_mode,omitempty"`
	SkillDiscoveryMode                          string                            `json:"skill_discovery_mode,omitempty"`
	SkillDiscoveryRoots                         []string                          `json:"skill_discovery_roots,omitempty"`
	SkillPreprocessEnabled                      bool                              `json:"skill_preprocess_enabled,omitempty"`
	SkillPreprocessPhase                        string                            `json:"skill_preprocess_phase,omitempty"`
	SkillPreprocessFailMode                     string                            `json:"skill_preprocess_fail_mode,omitempty"`
	SkillPreprocessStatus                       string                            `json:"skill_preprocess_status,omitempty"`
	SkillPreprocessReasonCode                   string                            `json:"skill_preprocess_reason_code,omitempty"`
	SkillPreprocessSpecCount                    int                               `json:"skill_preprocess_spec_count,omitempty"`
	SkillBundlePromptMode                       string                            `json:"skill_bundle_prompt_mode,omitempty"`
	SkillBundleWhitelistMode                    string                            `json:"skill_bundle_whitelist_mode,omitempty"`
	SkillBundleConflictPolicy                   string                            `json:"skill_bundle_conflict_policy,omitempty"`
	SkillBundlePromptTotal                      int                               `json:"skill_bundle_prompt_total,omitempty"`
	SkillBundleWhitelistTotal                   int                               `json:"skill_bundle_whitelist_total,omitempty"`
	SkillBundleWhitelistRejectedTotal           int                               `json:"skill_bundle_whitelist_rejected_total,omitempty"`
	DiagnosticsCardinalityBudgetHitTotal        int                               `json:"diagnostics_cardinality_budget_hit_total,omitempty"`
	DiagnosticsCardinalityTruncatedTotal        int                               `json:"diagnostics_cardinality_truncated_total,omitempty"`
	DiagnosticsCardinalityFailFastRejectTotal   int                               `json:"diagnostics_cardinality_fail_fast_reject_total,omitempty"`
	DiagnosticsCardinalityOverflowPolicy        string                            `json:"diagnostics_cardinality_overflow_policy,omitempty"`
	DiagnosticsCardinalityTruncatedFieldSummary string                            `json:"diagnostics_cardinality_truncated_field_summary,omitempty"`
	TimelinePhases                              map[string]TimelinePhaseAggregate `json:"timeline_phases,omitempty"`
}

type MailboxRecord struct {
	Time                  time.Time `json:"time"`
	MessageID             string    `json:"message_id"`
	IdempotencyKey        string    `json:"idempotency_key,omitempty"`
	CorrelationID         string    `json:"correlation_id,omitempty"`
	Kind                  string    `json:"kind,omitempty"`
	State                 string    `json:"state,omitempty"`
	FromAgent             string    `json:"from_agent,omitempty"`
	ToAgent               string    `json:"to_agent,omitempty"`
	RunID                 string    `json:"run_id,omitempty"`
	TaskID                string    `json:"task_id,omitempty"`
	WorkflowID            string    `json:"workflow_id,omitempty"`
	TeamID                string    `json:"team_id,omitempty"`
	Attempt               int       `json:"attempt,omitempty"`
	ConsumerID            string    `json:"consumer_id,omitempty"`
	ReasonCode            string    `json:"reason_code,omitempty"`
	Backend               string    `json:"backend,omitempty"`
	ConfiguredBackend     string    `json:"configured_backend,omitempty"`
	BackendFallback       bool      `json:"backend_fallback,omitempty"`
	BackendFallbackReason string    `json:"backend_fallback_reason,omitempty"`
	PublishPath           string    `json:"publish_path,omitempty"`
	Reclaimed             bool      `json:"reclaimed,omitempty"`
	PanicRecovered        bool      `json:"panic_recovered,omitempty"`
}

type MailboxQueryTimeRange struct {
	Start time.Time `json:"start,omitempty"`
	End   time.Time `json:"end,omitempty"`
}

type MailboxQuerySort struct {
	Field string `json:"field,omitempty"`
	Order string `json:"order,omitempty"`
}

type MailboxQueryRequest struct {
	MessageID      string                 `json:"message_id,omitempty"`
	IdempotencyKey string                 `json:"idempotency_key,omitempty"`
	CorrelationID  string                 `json:"correlation_id,omitempty"`
	Kind           string                 `json:"kind,omitempty"`
	State          string                 `json:"state,omitempty"`
	RunID          string                 `json:"run_id,omitempty"`
	TaskID         string                 `json:"task_id,omitempty"`
	WorkflowID     string                 `json:"workflow_id,omitempty"`
	TeamID         string                 `json:"team_id,omitempty"`
	TimeRange      *MailboxQueryTimeRange `json:"time_range,omitempty"`
	PageSize       *int                   `json:"page_size,omitempty"`
	Sort           MailboxQuerySort       `json:"sort,omitempty"`
	Cursor         string                 `json:"cursor,omitempty"`
}

type MailboxQueryResult struct {
	Items      []MailboxRecord `json:"items"`
	NextCursor string          `json:"next_cursor,omitempty"`
	PageSize   int             `json:"page_size"`
	SortField  string          `json:"sort_field"`
	SortOrder  string          `json:"sort_order"`
}

type MailboxAggregateRequest struct {
	Kind       string                 `json:"kind,omitempty"`
	State      string                 `json:"state,omitempty"`
	RunID      string                 `json:"run_id,omitempty"`
	TaskID     string                 `json:"task_id,omitempty"`
	WorkflowID string                 `json:"workflow_id,omitempty"`
	TeamID     string                 `json:"team_id,omitempty"`
	TimeRange  *MailboxQueryTimeRange `json:"time_range,omitempty"`
}

type MailboxAggregate struct {
	TotalRecords     int            `json:"total_records"`
	TotalMessages    int            `json:"total_messages"`
	ByKind           map[string]int `json:"by_kind,omitempty"`
	ByState          map[string]int `json:"by_state,omitempty"`
	RetryTotal       int            `json:"retry_total,omitempty"`
	DeadLetterTotal  int            `json:"dead_letter_total,omitempty"`
	ExpiredTotal     int            `json:"expired_total,omitempty"`
	ReasonCodeTotals map[string]int `json:"reason_code_totals,omitempty"`
}

type TimelinePhaseAggregate struct {
	CountTotal    int   `json:"count_total,omitempty"`
	FailedTotal   int   `json:"failed_total,omitempty"`
	CanceledTotal int   `json:"canceled_total,omitempty"`
	SkippedTotal  int   `json:"skipped_total,omitempty"`
	LatencyMs     int64 `json:"latency_ms,omitempty"`
	LatencyP95Ms  int64 `json:"latency_p95_ms,omitempty"`
}

type TimelineTrendMode string

const (
	TimelineTrendModeLastNRuns  TimelineTrendMode = "last_n_runs"
	TimelineTrendModeTimeWindow TimelineTrendMode = "time_window"
)

type TimelineTrendQuery struct {
	Mode       TimelineTrendMode
	LastNRuns  int
	TimeWindow time.Duration
}

type TimelineTrendRecord struct {
	Phase         string    `json:"phase"`
	Status        string    `json:"status"`
	CountTotal    int       `json:"count_total"`
	FailedTotal   int       `json:"failed_total"`
	CanceledTotal int       `json:"canceled_total"`
	SkippedTotal  int       `json:"skipped_total"`
	LatencyAvgMs  int64     `json:"latency_avg_ms"`
	LatencyP95Ms  int64     `json:"latency_p95_ms"`
	WindowStart   time.Time `json:"window_start"`
	WindowEnd     time.Time `json:"window_end"`
}

type SkillRecord struct {
	Time       time.Time      `json:"time"`
	RunID      string         `json:"run_id,omitempty"`
	SkillName  string         `json:"skill_name,omitempty"`
	Action     string         `json:"action"`
	Status     string         `json:"status"`
	LatencyMs  int64          `json:"latency_ms,omitempty"`
	ErrorClass string         `json:"error_class,omitempty"`
	Payload    map[string]any `json:"payload,omitempty"`
}

type ReloadRecord struct {
	Time    time.Time `json:"time"`
	Success bool      `json:"success"`
	Error   string    `json:"error,omitempty"`
}

type Store struct {
	mu sync.RWMutex

	maxCallRecords  int
	maxRunRecords   int
	maxReloadErrors int
	maxSkillRecords int

	calls   []CallRecord
	runs    []RunRecord
	mailbox []MailboxRecord
	reloads []ReloadRecord
	skills  []SkillRecord
	runKeys map[string]int
	mbxKeys map[string]int
	sklKeys map[string]int

	runTimesStrictAscending bool

	timelineStates map[string]*timelineRunState
	trendConfig    TimelineTrendConfig
	ca2TrendConfig CA2ExternalTrendConfig
	cardinality    CardinalityConfig
}

type timelineRunState struct {
	seen           map[string]struct{}
	runningSince   map[string]time.Time
	phaseLatencyMs map[string][]int64
	phases         map[string]TimelinePhaseAggregate
	buckets        map[string]timelineTrendBucket
}

type timelineTrendBucket struct {
	CountTotal    int
	FailedTotal   int
	CanceledTotal int
	SkippedTotal  int
	LatencyTotal  int64
	Latencies     []int64
}

type TimelineTrendConfig struct {
	Enabled    bool
	LastNRuns  int
	TimeWindow time.Duration
}

type CA2ExternalTrendConfig struct {
	Enabled    bool
	Window     time.Duration
	Thresholds CA2ExternalThresholds
}

type CA2ExternalThresholds struct {
	P95LatencyMs int64
	ErrorRate    float64
	HitRate      float64
}

type CardinalityConfig struct {
	Enabled        bool
	MaxMapEntries  int
	MaxListEntries int
	MaxStringBytes int
	OverflowPolicy string
}

type CA2ExternalTrendQuery struct {
	Window time.Duration
}

type CA2ExternalTrendRecord struct {
	Provider               string         `json:"provider"`
	WindowStart            time.Time      `json:"window_start"`
	WindowEnd              time.Time      `json:"window_end"`
	P95LatencyMs           int64          `json:"p95_latency_ms"`
	ErrorRate              float64        `json:"error_rate"`
	HitRate                float64        `json:"hit_rate"`
	ThresholdHits          []string       `json:"threshold_hits,omitempty"`
	ErrorLayerDistribution map[string]int `json:"error_layer_distribution,omitempty"`
}

const (
	DefaultUnifiedQueryPageSize = 50
	MaxUnifiedQueryPageSize     = 200
	DefaultMailboxQueryPageSize = 50
	MaxMailboxQueryPageSize     = 200

	DefaultCardinalityEnabled        = true
	DefaultCardinalityMaxMapEntries  = 64
	DefaultCardinalityMaxListEntries = 64
	DefaultCardinalityMaxStringBytes = 2048

	CardinalityOverflowTruncateAndRecord = "truncate_and_record"
	CardinalityOverflowFailFast          = "fail_fast"
	maxCardinalitySummaryFields          = 16
)

type UnifiedQueryTimeRange struct {
	Start time.Time `json:"start,omitempty"`
	End   time.Time `json:"end,omitempty"`
}

type UnifiedQuerySort struct {
	Field string `json:"field,omitempty"`
	Order string `json:"order,omitempty"`
}

type UnifiedRunQueryRequest struct {
	RunID      string                 `json:"run_id,omitempty"`
	TeamID     string                 `json:"team_id,omitempty"`
	WorkflowID string                 `json:"workflow_id,omitempty"`
	TaskID     string                 `json:"task_id,omitempty"`
	Status     string                 `json:"status,omitempty"`
	TimeRange  *UnifiedQueryTimeRange `json:"time_range,omitempty"`
	PageSize   *int                   `json:"page_size,omitempty"`
	Sort       UnifiedQuerySort       `json:"sort,omitempty"`
	Cursor     string                 `json:"cursor,omitempty"`
}

type UnifiedRunQueryResult struct {
	Items      []RunRecord `json:"items"`
	NextCursor string      `json:"next_cursor,omitempty"`
	PageSize   int         `json:"page_size"`
	SortField  string      `json:"sort_field"`
	SortOrder  string      `json:"sort_order"`
}

type normalizedUnifiedRunQuery struct {
	RunID      string
	TeamID     string
	WorkflowID string
	TaskID     string
	Status     string
	TimeRange  *UnifiedQueryTimeRange
	PageSize   int
	SortField  string
	SortOrder  string
	Cursor     string
}

type unifiedRunQueryCursor struct {
	Offset    int    `json:"offset"`
	QueryHash string `json:"query_hash"`
}

type normalizedMailboxQuery struct {
	MessageID      string
	IdempotencyKey string
	CorrelationID  string
	Kind           string
	State          string
	RunID          string
	TaskID         string
	WorkflowID     string
	TeamID         string
	TimeRange      *MailboxQueryTimeRange
	PageSize       int
	SortField      string
	SortOrder      string
	Cursor         string
}

type mailboxQueryCursor struct {
	Offset    int    `json:"offset"`
	QueryHash string `json:"query_hash"`
}

func NewStore(maxCalls, maxRuns, maxReloads, maxSkills int, trend TimelineTrendConfig, ca2 CA2ExternalTrendConfig) *Store {
	if maxCalls <= 0 {
		maxCalls = 200
	}
	if maxRuns <= 0 {
		maxRuns = 200
	}
	if maxReloads <= 0 {
		maxReloads = 100
	}
	if maxSkills <= 0 {
		maxSkills = 200
	}
	if trend.LastNRuns <= 0 {
		trend.LastNRuns = 100
	}
	if trend.TimeWindow <= 0 {
		trend.TimeWindow = 15 * time.Minute
	}
	if ca2.Window <= 0 {
		ca2.Window = 15 * time.Minute
	}
	if ca2.Thresholds.P95LatencyMs <= 0 {
		ca2.Thresholds.P95LatencyMs = 1500
	}
	if ca2.Thresholds.ErrorRate < 0 || ca2.Thresholds.ErrorRate > 1 {
		ca2.Thresholds.ErrorRate = 0.1
	}
	if ca2.Thresholds.HitRate < 0 || ca2.Thresholds.HitRate > 1 {
		ca2.Thresholds.HitRate = 0.2
	}
	return &Store{
		maxCallRecords:          maxCalls,
		maxRunRecords:           maxRuns,
		maxReloadErrors:         maxReloads,
		maxSkillRecords:         maxSkills,
		calls:                   make([]CallRecord, 0, maxCalls),
		runs:                    make([]RunRecord, 0, maxRuns),
		mailbox:                 make([]MailboxRecord, 0, maxRuns),
		reloads:                 make([]ReloadRecord, 0, maxReloads),
		skills:                  make([]SkillRecord, 0, maxSkills),
		runKeys:                 make(map[string]int, maxRuns),
		mbxKeys:                 make(map[string]int, maxRuns),
		sklKeys:                 make(map[string]int, maxSkills),
		runTimesStrictAscending: true,
		timelineStates:          make(map[string]*timelineRunState, maxRuns),
		trendConfig:             trend,
		ca2TrendConfig:          ca2,
		cardinality:             normalizeCardinalityConfig(CardinalityConfig{}),
	}
}

func (d *Store) Resize(maxCalls, maxRuns, maxReloads, maxSkills int) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if maxCalls > 0 {
		d.maxCallRecords = maxCalls
		d.calls = trimTail(d.calls, d.maxCallRecords)
	}
	if maxRuns > 0 {
		d.maxRunRecords = maxRuns
		d.runs = trimTail(d.runs, d.maxRunRecords)
		d.mailbox = trimTail(d.mailbox, d.maxRunRecords)
		d.rebuildRunKeys()
		d.rebuildMailboxKeys()
		d.pruneTimelineStates()
	}
	if maxReloads > 0 {
		d.maxReloadErrors = maxReloads
		d.reloads = trimTail(d.reloads, d.maxReloadErrors)
	}
	if maxSkills > 0 {
		d.maxSkillRecords = maxSkills
		d.skills = trimTail(d.skills, d.maxSkillRecords)
		d.rebuildSkillKeys()
	}
}

func (d *Store) SetTrendConfig(cfg TimelineTrendConfig) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if cfg.LastNRuns <= 0 {
		cfg.LastNRuns = 100
	}
	if cfg.TimeWindow <= 0 {
		cfg.TimeWindow = 15 * time.Minute
	}
	d.trendConfig = cfg
}

func (d *Store) SetCA2ExternalTrendConfig(cfg CA2ExternalTrendConfig) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if cfg.Window <= 0 {
		cfg.Window = 15 * time.Minute
	}
	if cfg.Thresholds.P95LatencyMs <= 0 {
		cfg.Thresholds.P95LatencyMs = 1500
	}
	if cfg.Thresholds.ErrorRate < 0 || cfg.Thresholds.ErrorRate > 1 {
		cfg.Thresholds.ErrorRate = 0.1
	}
	if cfg.Thresholds.HitRate < 0 || cfg.Thresholds.HitRate > 1 {
		cfg.Thresholds.HitRate = 0.2
	}
	d.ca2TrendConfig = cfg
}

func (d *Store) SetCardinalityConfig(cfg CardinalityConfig) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.cardinality = normalizeCardinalityConfig(cfg)
}

func (d *Store) AddCall(rec CallRecord) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.calls = append(d.calls, rec)
	d.calls = trimTail(d.calls, d.maxCallRecords)
}

func (d *Store) AddRun(rec RunRecord) {
	d.mu.Lock()
	defer d.mu.Unlock()
	rec.Status = normalizeRunStatus(rec.Status, rec.ErrorClass)
	rec.TaskBoardManualControlByAction = cloneIntMap(rec.TaskBoardManualControlByAction)
	rec.TaskBoardManualControlByReason = cloneIntMap(rec.TaskBoardManualControlByReason)
	rec.MemoryRerankStats = cloneIntMap(rec.MemoryRerankStats)
	rec.RuntimeSecondaryReasonCodes = cloneStringSlice(rec.RuntimeSecondaryReasonCodes)
	rec.PolicyDecisionPath = cloneRuntimePolicyDecisionPath(rec.PolicyDecisionPath)
	rec.SandboxRequiredCapabilities = cloneStringSlice(rec.SandboxRequiredCapabilities)
	rec.HooksPhases = cloneStringSlice(rec.HooksPhases)
	rec.SkillDiscoveryRoots = cloneStringSlice(rec.SkillDiscoveryRoots)
	rec.BudgetSnapshot = cloneAnyMap(rec.BudgetSnapshot)
	rec.EvalSummary = cloneAnyMap(rec.EvalSummary)
	if len(rec.TimelinePhases) == 0 {
		rec.TimelinePhases = d.timelinePhasesForRun(rec.RunID)
	}
	rec = applyCardinalityGovernance(rec, d.cardinality)
	key := RunIdempotencyKey(rec)
	if idx, ok := d.runKeys[key]; ok && idx >= 0 && idx < len(d.runs) {
		if d.runTimesStrictAscending {
			if idx > 0 && !rec.Time.After(d.runs[idx-1].Time) {
				d.runTimesStrictAscending = false
			}
			if idx+1 < len(d.runs) && !d.runs[idx+1].Time.After(rec.Time) {
				d.runTimesStrictAscending = false
			}
		}
		d.runs[idx] = rec
		return
	}
	if d.runTimesStrictAscending && len(d.runs) > 0 {
		last := d.runs[len(d.runs)-1]
		if !rec.Time.After(last.Time) {
			d.runTimesStrictAscending = false
		}
	}
	d.runs = append(d.runs, rec)
	d.runs = trimTail(d.runs, d.maxRunRecords)
	d.rebuildRunKeys()
	d.pruneTimelineStates()
}

func (d *Store) AddMailbox(rec MailboxRecord) {
	d.mu.Lock()
	defer d.mu.Unlock()
	rec = normalizeMailboxRecord(rec)
	key := MailboxIdempotencyKey(rec)
	if idx, ok := d.mbxKeys[key]; ok && idx >= 0 && idx < len(d.mailbox) {
		d.mailbox[idx] = rec
		return
	}
	d.mailbox = append(d.mailbox, rec)
	d.mailbox = trimTail(d.mailbox, d.maxRunRecords)
	d.rebuildMailboxKeys()
}

func (d *Store) AddTimelineEvent(runID, phase, status string, sequence int64, ts time.Time) {
	d.mu.Lock()
	defer d.mu.Unlock()
	runID = strings.TrimSpace(runID)
	phase = strings.TrimSpace(phase)
	status = strings.ToLower(strings.TrimSpace(status))
	if runID == "" || phase == "" || sequence <= 0 {
		return
	}
	state := d.timelineStates[runID]
	if state == nil {
		state = &timelineRunState{
			seen:           map[string]struct{}{},
			runningSince:   map[string]time.Time{},
			phaseLatencyMs: map[string][]int64{},
			phases:         map[string]TimelinePhaseAggregate{},
			buckets:        map[string]timelineTrendBucket{},
		}
		d.timelineStates[runID] = state
	}
	key := fmt.Sprintf("%d:%s:%s", sequence, phase, status)
	if _, ok := state.seen[key]; ok {
		return
	}
	state.seen[key] = struct{}{}
	if ts.IsZero() {
		ts = time.Now()
	}
	switch status {
	case "running":
		state.runningSince[phase] = ts
	case "succeeded", "failed", "canceled", "skipped":
		agg := state.phases[phase]
		agg.CountTotal++
		switch status {
		case "failed":
			agg.FailedTotal++
		case "canceled":
			agg.CanceledTotal++
		case "skipped":
			agg.SkippedTotal++
		}
		if startedAt, ok := state.runningSince[phase]; ok && !startedAt.IsZero() {
			lat := ts.Sub(startedAt).Milliseconds()
			if lat < 0 {
				lat = 0
			}
			agg.LatencyMs += lat
			phaseSamples := state.phaseLatencyMs[phase]
			phaseSamples = append(phaseSamples, lat)
			state.phaseLatencyMs[phase] = phaseSamples
			agg.LatencyP95Ms = percentileP95(phaseSamples)
			delete(state.runningSince, phase)
		}
		state.phases[phase] = agg
		state.recordBucket(phase, status, state.phaseLatencyMs[phase])
	}
}

func (s *timelineRunState) recordBucket(phase, status string, phaseSamples []int64) {
	if s == nil {
		return
	}
	key := trendBucketKey(phase, status)
	b := s.buckets[key]
	b.CountTotal++
	switch status {
	case "failed":
		b.FailedTotal++
	case "canceled":
		b.CanceledTotal++
	case "skipped":
		b.SkippedTotal++
	}
	if len(phaseSamples) > 0 {
		lat := phaseSamples[len(phaseSamples)-1]
		b.LatencyTotal += lat
		b.Latencies = append(b.Latencies, lat)
	}
	s.buckets[key] = b
}

func (d *Store) AddReload(rec ReloadRecord) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.reloads = append(d.reloads, rec)
	d.reloads = trimTail(d.reloads, d.maxReloadErrors)
}

func (d *Store) AddSkill(rec SkillRecord) {
	d.mu.Lock()
	defer d.mu.Unlock()
	rec.Status = normalizeSkillStatus(rec.Status)
	key := SkillIdempotencyKey(rec)
	if idx, ok := d.sklKeys[key]; ok && idx >= 0 && idx < len(d.skills) {
		d.skills[idx] = rec
		return
	}
	d.skills = append(d.skills, rec)
	d.skills = trimTail(d.skills, d.maxSkillRecords)
	d.rebuildSkillKeys()
}

type cardinalityGovernanceStats struct {
	budgetHitFields map[string]struct{}
	truncatedFields map[string]struct{}
}

func applyCardinalityGovernance(rec RunRecord, cfg CardinalityConfig) RunRecord {
	cfg = normalizeCardinalityConfig(cfg)
	rec.DiagnosticsCardinalityBudgetHitTotal = 0
	rec.DiagnosticsCardinalityTruncatedTotal = 0
	rec.DiagnosticsCardinalityFailFastRejectTotal = 0
	rec.DiagnosticsCardinalityOverflowPolicy = cfg.OverflowPolicy
	rec.DiagnosticsCardinalityTruncatedFieldSummary = ""
	if !cfg.Enabled {
		return rec
	}
	stats := cardinalityGovernanceStats{
		budgetHitFields: map[string]struct{}{},
		truncatedFields: map[string]struct{}{},
	}
	rv := reflect.ValueOf(&rec).Elem()
	rt := rv.Type()
	for i := 0; i < rv.NumField(); i++ {
		field := rv.Field(i)
		structField := rt.Field(i)
		jsonName := jsonFieldName(structField)
		if jsonName == "" || strings.HasPrefix(jsonName, "diagnostics_cardinality_") {
			continue
		}
		switch field.Kind() {
		case reflect.String:
			if isCardinalityStringExemptField(jsonName) {
				continue
			}
			value := field.String()
			if len([]byte(value)) <= cfg.MaxStringBytes {
				continue
			}
			recordCardinalityBudgetHit(&stats, jsonName)
			if cfg.OverflowPolicy == CardinalityOverflowTruncateAndRecord {
				field.SetString(truncateUTF8ByBytes(value, cfg.MaxStringBytes))
				recordCardinalityTruncated(&stats, jsonName)
				continue
			}
			field.SetString("")
		case reflect.Map:
			if field.IsNil() {
				continue
			}
			keys := field.MapKeys()
			if len(keys) <= cfg.MaxMapEntries {
				continue
			}
			recordCardinalityBudgetHit(&stats, jsonName)
			if cfg.OverflowPolicy == CardinalityOverflowFailFast {
				field.Set(reflect.Zero(field.Type()))
				continue
			}
			sort.Slice(keys, func(i, j int) bool {
				return keys[i].String() < keys[j].String()
			})
			limit := cfg.MaxMapEntries
			if limit > len(keys) {
				limit = len(keys)
			}
			out := reflect.MakeMapWithSize(field.Type(), limit)
			for idx := 0; idx < limit; idx++ {
				key := keys[idx]
				out.SetMapIndex(key, field.MapIndex(key))
			}
			field.Set(out)
			recordCardinalityTruncated(&stats, jsonName)
		case reflect.Slice:
			if field.IsNil() || field.Len() <= cfg.MaxListEntries {
				continue
			}
			recordCardinalityBudgetHit(&stats, jsonName)
			if cfg.OverflowPolicy == CardinalityOverflowFailFast {
				field.Set(reflect.Zero(field.Type()))
				continue
			}
			limit := cfg.MaxListEntries
			if limit > field.Len() {
				limit = field.Len()
			}
			out := reflect.MakeSlice(field.Type(), limit, limit)
			reflect.Copy(out, field.Slice(0, limit))
			field.Set(out)
			recordCardinalityTruncated(&stats, jsonName)
		}
	}

	rec.DiagnosticsCardinalityBudgetHitTotal = len(stats.budgetHitFields)
	rec.DiagnosticsCardinalityTruncatedTotal = len(stats.truncatedFields)
	rec.DiagnosticsCardinalityOverflowPolicy = cfg.OverflowPolicy
	if cfg.OverflowPolicy == CardinalityOverflowFailFast && rec.DiagnosticsCardinalityBudgetHitTotal > 0 {
		rec.DiagnosticsCardinalityFailFastRejectTotal = 1
		rec.DiagnosticsCardinalityTruncatedFieldSummary = boundedCardinalitySummary(stats.budgetHitFields)
		return rec
	}
	rec.DiagnosticsCardinalityTruncatedFieldSummary = boundedCardinalitySummary(stats.truncatedFields)
	return rec
}

func governCardinalityValue(
	value any,
	path string,
	cfg CardinalityConfig,
	stats *cardinalityGovernanceStats,
	enforceMapBudget bool,
) (any, bool) {
	switch typed := value.(type) {
	case map[string]any:
		keys := make([]string, 0, len(typed))
		for key := range typed {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		overflow := enforceMapBudget && len(keys) > cfg.MaxMapEntries
		if overflow {
			recordCardinalityBudgetHit(stats, path)
		}

		limit := len(keys)
		if overflow {
			limit = cfg.MaxMapEntries
		}
		out := make(map[string]any, limit)
		for i := 0; i < limit; i++ {
			key := keys[i]
			nextPath := key
			if path != "" {
				nextPath = path + "." + key
			}
			nextValue, _ := governCardinalityValue(typed[key], nextPath, cfg, stats, true)
			out[key] = nextValue
		}
		if overflow {
			if cfg.OverflowPolicy == CardinalityOverflowTruncateAndRecord {
				recordCardinalityTruncated(stats, path)
				return out, true
			}
			return nil, true
		}
		return out, false
	case []any:
		overflow := len(typed) > cfg.MaxListEntries
		if overflow {
			recordCardinalityBudgetHit(stats, path)
		}
		limit := len(typed)
		if overflow {
			limit = cfg.MaxListEntries
		}
		out := make([]any, 0, limit)
		for i := 0; i < limit; i++ {
			nextPath := path
			if nextPath == "" {
				nextPath = "list"
			}
			nextValue, _ := governCardinalityValue(typed[i], nextPath, cfg, stats, true)
			out = append(out, nextValue)
		}
		if overflow {
			if cfg.OverflowPolicy == CardinalityOverflowTruncateAndRecord {
				recordCardinalityTruncated(stats, path)
				return out, true
			}
			return nil, true
		}
		return out, false
	case string:
		if len([]byte(typed)) <= cfg.MaxStringBytes {
			return typed, false
		}
		recordCardinalityBudgetHit(stats, path)
		if cfg.OverflowPolicy == CardinalityOverflowTruncateAndRecord {
			recordCardinalityTruncated(stats, path)
			return truncateUTF8ByBytes(typed, cfg.MaxStringBytes), true
		}
		return "", true
	default:
		return value, false
	}
}

func normalizeCardinalityConfig(cfg CardinalityConfig) CardinalityConfig {
	if !cfg.Enabled && cfg.MaxMapEntries == 0 && cfg.MaxListEntries == 0 && cfg.MaxStringBytes == 0 && strings.TrimSpace(cfg.OverflowPolicy) == "" {
		cfg.Enabled = DefaultCardinalityEnabled
	}
	if cfg.MaxMapEntries <= 0 {
		cfg.MaxMapEntries = DefaultCardinalityMaxMapEntries
	}
	if cfg.MaxListEntries <= 0 {
		cfg.MaxListEntries = DefaultCardinalityMaxListEntries
	}
	if cfg.MaxStringBytes <= 0 {
		cfg.MaxStringBytes = DefaultCardinalityMaxStringBytes
	}
	policy := strings.ToLower(strings.TrimSpace(cfg.OverflowPolicy))
	switch policy {
	case CardinalityOverflowTruncateAndRecord, CardinalityOverflowFailFast:
	default:
		policy = CardinalityOverflowTruncateAndRecord
	}
	cfg.OverflowPolicy = policy
	return cfg
}

func recordCardinalityBudgetHit(stats *cardinalityGovernanceStats, path string) {
	if stats == nil {
		return
	}
	if stats.budgetHitFields == nil {
		stats.budgetHitFields = map[string]struct{}{}
	}
	stats.budgetHitFields[cardinalitySummaryField(path)] = struct{}{}
}

func recordCardinalityTruncated(stats *cardinalityGovernanceStats, path string) {
	if stats == nil {
		return
	}
	if stats.truncatedFields == nil {
		stats.truncatedFields = map[string]struct{}{}
	}
	stats.truncatedFields[cardinalitySummaryField(path)] = struct{}{}
}

func cardinalitySummaryField(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return "_root"
	}
	if idx := strings.Index(path, "."); idx > 0 {
		return path[:idx]
	}
	return path
}

func boundedCardinalitySummary(fields map[string]struct{}) string {
	if len(fields) == 0 {
		return ""
	}
	items := make([]string, 0, len(fields))
	for field := range fields {
		items = append(items, field)
	}
	sort.Strings(items)
	if len(items) > maxCardinalitySummaryFields {
		items = items[:maxCardinalitySummaryFields]
	}
	return strings.Join(items, ",")
}

func jsonFieldName(field reflect.StructField) string {
	tag := strings.TrimSpace(field.Tag.Get("json"))
	if tag == "" {
		return ""
	}
	name := strings.SplitN(tag, ",", 2)[0]
	if name == "-" {
		return ""
	}
	return strings.TrimSpace(name)
}

func isCardinalityStringExemptField(name string) bool {
	name = strings.ToLower(strings.TrimSpace(name))
	if name == "" {
		return true
	}
	switch name {
	case "status", "error_class":
		return true
	}
	return strings.HasSuffix(name, "_id")
}

func truncateUTF8ByBytes(in string, maxBytes int) string {
	if maxBytes <= 0 {
		return ""
	}
	raw := []byte(in)
	if len(raw) <= maxBytes {
		return in
	}
	end := maxBytes
	for end > 0 && !utf8.Valid(raw[:end]) {
		end--
	}
	if end <= 0 {
		return ""
	}
	return string(raw[:end])
}

func (d *Store) RecentCalls(n int) []CallRecord {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return tailCopy(d.calls, n)
}

func (d *Store) RecentRuns(n int) []RunRecord {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return tailCopy(d.runs, n)
}

func (d *Store) RecentMailbox(n int) []MailboxRecord {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return tailCopy(d.mailbox, n)
}

func (d *Store) QueryMailbox(req MailboxQueryRequest) (MailboxQueryResult, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	q, err := normalizeMailboxQuery(req)
	if err != nil {
		return MailboxQueryResult{}, err
	}
	queryHash := mailboxQueryHash(q)
	start, err := decodeMailboxCursor(q.Cursor, queryHash)
	if err != nil {
		return MailboxQueryResult{}, err
	}

	filtered := make([]MailboxRecord, 0, len(d.mailbox))
	for i := range d.mailbox {
		if matchesMailboxQuery(d.mailbox[i], q) {
			filtered = append(filtered, d.mailbox[i])
		}
	}
	sortMailboxQuery(filtered, q.SortOrder)

	if start > len(filtered) {
		return MailboxQueryResult{}, fmt.Errorf("invalid query cursor")
	}
	end := start + q.PageSize
	if end > len(filtered) {
		end = len(filtered)
	}
	items := append([]MailboxRecord(nil), filtered[start:end]...)
	nextCursor := ""
	if end < len(filtered) {
		nextCursor, err = encodeMailboxCursor(mailboxQueryCursor{
			Offset:    end,
			QueryHash: queryHash,
		})
		if err != nil {
			return MailboxQueryResult{}, err
		}
	}
	return MailboxQueryResult{
		Items:      items,
		NextCursor: nextCursor,
		PageSize:   q.PageSize,
		SortField:  q.SortField,
		SortOrder:  q.SortOrder,
	}, nil
}

func (d *Store) MailboxAggregates(req MailboxAggregateRequest) MailboxAggregate {
	d.mu.RLock()
	defer d.mu.RUnlock()

	q, err := normalizeMailboxQuery(MailboxQueryRequest{
		Kind:       req.Kind,
		State:      req.State,
		RunID:      req.RunID,
		TaskID:     req.TaskID,
		WorkflowID: req.WorkflowID,
		TeamID:     req.TeamID,
		TimeRange:  req.TimeRange,
	})
	if err != nil {
		return MailboxAggregate{
			ByKind:           map[string]int{},
			ByState:          map[string]int{},
			ReasonCodeTotals: map[string]int{},
		}
	}

	filtered := make([]MailboxRecord, 0, len(d.mailbox))
	for i := range d.mailbox {
		if matchesMailboxQuery(d.mailbox[i], q) {
			filtered = append(filtered, d.mailbox[i])
		}
	}
	latestByMessage := map[string]MailboxRecord{}
	for i := range filtered {
		rec := filtered[i]
		key := strings.TrimSpace(rec.MessageID)
		if key == "" {
			key = MailboxIdempotencyKey(rec)
		}
		existing, ok := latestByMessage[key]
		if !ok || rec.Time.After(existing.Time) || (rec.Time.Equal(existing.Time) && rec.Attempt > existing.Attempt) {
			latestByMessage[key] = rec
		}
	}

	out := MailboxAggregate{
		TotalRecords:     len(filtered),
		TotalMessages:    len(latestByMessage),
		ByKind:           map[string]int{},
		ByState:          map[string]int{},
		ReasonCodeTotals: map[string]int{},
	}
	for _, rec := range latestByMessage {
		if rec.Kind != "" {
			out.ByKind[rec.Kind]++
		}
		if rec.State != "" {
			out.ByState[rec.State]++
		}
		if rec.Attempt > 1 {
			out.RetryTotal += rec.Attempt - 1
		}
		switch rec.State {
		case "dead_letter":
			out.DeadLetterTotal++
		case "expired":
			out.ExpiredTotal++
		}
		if rec.ReasonCode != "" {
			out.ReasonCodeTotals[rec.ReasonCode]++
		}
	}
	return out
}

func (d *Store) QueryRuns(req UnifiedRunQueryRequest) (UnifiedRunQueryResult, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	q, err := normalizeUnifiedRunQuery(req)
	if err != nil {
		return UnifiedRunQueryResult{}, err
	}
	queryHash := unifiedRunQueryHash(q)
	start, err := decodeUnifiedRunCursor(q.Cursor, queryHash)
	if err != nil {
		return UnifiedRunQueryResult{}, err
	}

	if d.runTimesStrictAscending && q.SortField == "time" {
		return queryRunsFastTimeSorted(d.runs, q, start, queryHash)
	}

	filtered := make([]RunRecord, 0, len(d.runs))
	for i := range d.runs {
		if matchesUnifiedRunQuery(d.runs[i], q) {
			filtered = append(filtered, d.runs[i])
		}
	}
	sortUnifiedRunQuery(filtered, q.SortOrder)

	if start > len(filtered) {
		return UnifiedRunQueryResult{}, fmt.Errorf("invalid query cursor")
	}
	end := start + q.PageSize
	if end > len(filtered) {
		end = len(filtered)
	}
	items := append([]RunRecord(nil), filtered[start:end]...)
	nextCursor := ""
	if end < len(filtered) {
		nextCursor, err = encodeUnifiedRunCursor(unifiedRunQueryCursor{
			Offset:    end,
			QueryHash: queryHash,
		})
		if err != nil {
			return UnifiedRunQueryResult{}, err
		}
	}
	return UnifiedRunQueryResult{
		Items:      items,
		NextCursor: nextCursor,
		PageSize:   q.PageSize,
		SortField:  q.SortField,
		SortOrder:  q.SortOrder,
	}, nil
}

func queryRunsFastTimeSorted(runs []RunRecord, q normalizedUnifiedRunQuery, start int, queryHash string) (UnifiedRunQueryResult, error) {
	items := make([]RunRecord, 0, q.PageSize)
	matched := 0
	hasMore := false

	if q.SortOrder == "asc" {
		for i := 0; i < len(runs); i++ {
			rec := runs[i]
			if !matchesUnifiedRunQuery(rec, q) {
				continue
			}
			if matched < start {
				matched++
				continue
			}
			if len(items) < q.PageSize {
				items = append(items, rec)
				matched++
				continue
			}
			hasMore = true
			break
		}
	} else {
		for i := len(runs) - 1; i >= 0; i-- {
			rec := runs[i]
			if !matchesUnifiedRunQuery(rec, q) {
				continue
			}
			if matched < start {
				matched++
				continue
			}
			if len(items) < q.PageSize {
				items = append(items, rec)
				matched++
				continue
			}
			hasMore = true
			break
		}
	}

	if len(items) == 0 && matched < start {
		return UnifiedRunQueryResult{}, fmt.Errorf("invalid query cursor")
	}

	nextCursor := ""
	if hasMore {
		end := start + len(items)
		encoded, err := encodeUnifiedRunCursor(unifiedRunQueryCursor{
			Offset:    end,
			QueryHash: queryHash,
		})
		if err != nil {
			return UnifiedRunQueryResult{}, err
		}
		nextCursor = encoded
	}

	return UnifiedRunQueryResult{
		Items:      items,
		NextCursor: nextCursor,
		PageSize:   q.PageSize,
		SortField:  q.SortField,
		SortOrder:  q.SortOrder,
	}, nil
}

func (d *Store) TimelineTrends(query TimelineTrendQuery) []TimelineTrendRecord {
	d.mu.RLock()
	defer d.mu.RUnlock()
	if !d.trendConfig.Enabled {
		return []TimelineTrendRecord{}
	}
	selected, start, end := d.selectTrendRuns(query)
	if len(selected) == 0 {
		return []TimelineTrendRecord{}
	}
	type aggState struct {
		countTotal    int
		failedTotal   int
		canceledTotal int
		skippedTotal  int
		latencyTotal  int64
		latencies     []int64
	}
	agg := map[string]*aggState{}
	for _, rec := range selected {
		runID := strings.TrimSpace(rec.RunID)
		if runID == "" {
			continue
		}
		state := d.timelineStates[runID]
		if state == nil {
			continue
		}
		for bucketKey, bucket := range state.buckets {
			if bucket.CountTotal == 0 {
				continue
			}
			s := agg[bucketKey]
			if s == nil {
				s = &aggState{}
				agg[bucketKey] = s
			}
			s.countTotal += bucket.CountTotal
			s.failedTotal += bucket.FailedTotal
			s.canceledTotal += bucket.CanceledTotal
			s.skippedTotal += bucket.SkippedTotal
			s.latencyTotal += bucket.LatencyTotal
			if len(bucket.Latencies) > 0 {
				s.latencies = append(s.latencies, bucket.Latencies...)
			}
		}
	}
	if len(agg) == 0 {
		return []TimelineTrendRecord{}
	}
	out := make([]TimelineTrendRecord, 0, len(agg))
	for key, s := range agg {
		phase, status := splitTrendBucketKey(key)
		latAvg := int64(0)
		if s.countTotal > 0 {
			latAvg = s.latencyTotal / int64(s.countTotal)
		}
		out = append(out, TimelineTrendRecord{
			Phase:         phase,
			Status:        status,
			CountTotal:    s.countTotal,
			FailedTotal:   s.failedTotal,
			CanceledTotal: s.canceledTotal,
			SkippedTotal:  s.skippedTotal,
			LatencyAvgMs:  latAvg,
			LatencyP95Ms:  percentileP95(s.latencies),
			WindowStart:   start,
			WindowEnd:     end,
		})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Phase == out[j].Phase {
			return out[i].Status < out[j].Status
		}
		return out[i].Phase < out[j].Phase
	})
	return out
}

func (d *Store) CA2ExternalTrends(query CA2ExternalTrendQuery) []CA2ExternalTrendRecord {
	d.mu.RLock()
	defer d.mu.RUnlock()
	if !d.ca2TrendConfig.Enabled {
		return []CA2ExternalTrendRecord{}
	}
	selected, start, end := d.selectCA2Runs(query)
	if len(selected) == 0 {
		return []CA2ExternalTrendRecord{}
	}
	type agg struct {
		total      int
		hits       int
		errors     int
		latencies  []int64
		layerCount map[string]int
	}
	byProvider := map[string]*agg{}
	for i := range selected {
		provider := strings.ToLower(strings.TrimSpace(selected[i].Stage2Provider))
		if provider == "" {
			continue
		}
		item := byProvider[provider]
		if item == nil {
			item = &agg{layerCount: map[string]int{}}
			byProvider[provider] = item
		}
		item.total++
		if selected[i].Stage2LatencyMs > 0 {
			item.latencies = append(item.latencies, selected[i].Stage2LatencyMs)
		}
		if selected[i].Stage2HitCount > 0 {
			item.hits++
		}
		if isCA2ExternalError(selected[i]) {
			item.errors++
			layer := strings.ToLower(strings.TrimSpace(selected[i].Stage2ErrorLayer))
			if layer == "" {
				layer = "unknown"
			}
			item.layerCount[layer]++
		}
	}
	if len(byProvider) == 0 {
		return []CA2ExternalTrendRecord{}
	}
	out := make([]CA2ExternalTrendRecord, 0, len(byProvider))
	for provider, item := range byProvider {
		if item.total == 0 {
			continue
		}
		errorRate := float64(item.errors) / float64(item.total)
		hitRate := float64(item.hits) / float64(item.total)
		p95 := percentileP95(item.latencies)
		thresholdHits := make([]string, 0, 3)
		if p95 > d.ca2TrendConfig.Thresholds.P95LatencyMs {
			thresholdHits = append(thresholdHits, "p95_latency_ms")
		}
		if errorRate > d.ca2TrendConfig.Thresholds.ErrorRate {
			thresholdHits = append(thresholdHits, "error_rate")
		}
		if hitRate < d.ca2TrendConfig.Thresholds.HitRate {
			thresholdHits = append(thresholdHits, "hit_rate")
		}
		sort.Strings(thresholdHits)
		out = append(out, CA2ExternalTrendRecord{
			Provider:               provider,
			WindowStart:            start,
			WindowEnd:              end,
			P95LatencyMs:           p95,
			ErrorRate:              errorRate,
			HitRate:                hitRate,
			ThresholdHits:          thresholdHits,
			ErrorLayerDistribution: item.layerCount,
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Provider < out[j].Provider })
	return out
}

func (d *Store) RecentReloads(n int) []ReloadRecord {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return tailCopy(d.reloads, n)
}

func (d *Store) RecentSkills(n int) []SkillRecord {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return tailCopy(d.skills, n)
}

func SanitizeMap(in map[string]any) map[string]any {
	return redaction.New(true, redaction.DefaultKeywords()).SanitizeMap(in)
}

func RunIdempotencyKey(rec RunRecord) string {
	status := normalizeRunStatus(rec.Status, rec.ErrorClass)
	if strings.TrimSpace(rec.RunID) != "" {
		return fmt.Sprintf("run:%s:%s", strings.TrimSpace(rec.RunID), status)
	}
	return fmt.Sprintf(
		"run:anon:%d:%d:%d:%s:%s",
		rec.Iterations,
		rec.ToolCalls,
		rec.LatencyMs,
		status,
		strings.TrimSpace(rec.ErrorClass),
	)
}

func SkillIdempotencyKey(rec SkillRecord) string {
	return fmt.Sprintf(
		"skill:%s:%s:%s:%s:%s:%s",
		strings.TrimSpace(rec.RunID),
		strings.TrimSpace(rec.SkillName),
		strings.TrimSpace(rec.Action),
		normalizeSkillStatus(rec.Status),
		strings.TrimSpace(rec.ErrorClass),
		payloadDigest(rec.Payload),
	)
}

func MailboxIdempotencyKey(rec MailboxRecord) string {
	messageID := strings.TrimSpace(rec.MessageID)
	if messageID == "" {
		messageID = strings.TrimSpace(rec.IdempotencyKey)
	}
	if messageID == "" {
		messageID = strings.TrimSpace(rec.CorrelationID)
	}
	return fmt.Sprintf(
		"mailbox:%s:%s:%d:%s:%t:%t",
		messageID,
		strings.ToLower(strings.TrimSpace(rec.State)),
		rec.Attempt,
		strings.TrimSpace(rec.ReasonCode),
		rec.Reclaimed,
		rec.PanicRecovered,
	)
}

func normalizeRunStatus(status, errorClass string) string {
	s := strings.ToLower(strings.TrimSpace(status))
	switch s {
	case "success", "failed":
		return s
	}
	if strings.TrimSpace(errorClass) != "" {
		return "failed"
	}
	return "success"
}

func normalizeSkillStatus(status string) string {
	s := strings.ToLower(strings.TrimSpace(status))
	switch s {
	case "success", "failed", "warning":
		return s
	default:
		return "warning"
	}
}

func normalizeMailboxRecord(rec MailboxRecord) MailboxRecord {
	rec.MessageID = strings.TrimSpace(rec.MessageID)
	rec.IdempotencyKey = strings.TrimSpace(rec.IdempotencyKey)
	rec.CorrelationID = strings.TrimSpace(rec.CorrelationID)
	rec.Kind = strings.ToLower(strings.TrimSpace(rec.Kind))
	rec.State = strings.ToLower(strings.TrimSpace(rec.State))
	rec.FromAgent = strings.TrimSpace(rec.FromAgent)
	rec.ToAgent = strings.TrimSpace(rec.ToAgent)
	rec.RunID = strings.TrimSpace(rec.RunID)
	rec.TaskID = strings.TrimSpace(rec.TaskID)
	rec.WorkflowID = strings.TrimSpace(rec.WorkflowID)
	rec.TeamID = strings.TrimSpace(rec.TeamID)
	rec.ConsumerID = strings.TrimSpace(rec.ConsumerID)
	rec.ReasonCode = strings.TrimSpace(rec.ReasonCode)
	rec.Backend = strings.ToLower(strings.TrimSpace(rec.Backend))
	rec.ConfiguredBackend = strings.ToLower(strings.TrimSpace(rec.ConfiguredBackend))
	rec.BackendFallbackReason = strings.TrimSpace(rec.BackendFallbackReason)
	rec.PublishPath = strings.ToLower(strings.TrimSpace(rec.PublishPath))
	if rec.Attempt < 0 {
		rec.Attempt = 0
	}
	if rec.Time.IsZero() {
		rec.Time = time.Now().UTC()
	} else {
		rec.Time = rec.Time.UTC()
	}
	return rec
}

func normalizeUnifiedRunQuery(req UnifiedRunQueryRequest) (normalizedUnifiedRunQuery, error) {
	pageSize := DefaultUnifiedQueryPageSize
	if req.PageSize != nil {
		if *req.PageSize <= 0 || *req.PageSize > MaxUnifiedQueryPageSize {
			return normalizedUnifiedRunQuery{}, fmt.Errorf("page_size must be within [1,%d]", MaxUnifiedQueryPageSize)
		}
		pageSize = *req.PageSize
	}
	sortField := strings.ToLower(strings.TrimSpace(req.Sort.Field))
	if sortField == "" {
		sortField = "time"
	}
	if sortField != "time" {
		return normalizedUnifiedRunQuery{}, fmt.Errorf("unsupported sort.field %q", req.Sort.Field)
	}
	sortOrder := strings.ToLower(strings.TrimSpace(req.Sort.Order))
	if sortOrder == "" {
		sortOrder = "desc"
	}
	if sortOrder != "asc" && sortOrder != "desc" {
		return normalizedUnifiedRunQuery{}, fmt.Errorf("unsupported sort.order %q", req.Sort.Order)
	}
	status := strings.ToLower(strings.TrimSpace(req.Status))
	if status != "" && status != "success" && status != "failed" {
		return normalizedUnifiedRunQuery{}, fmt.Errorf("unsupported status filter %q", req.Status)
	}
	var tr *UnifiedQueryTimeRange
	if req.TimeRange != nil {
		start := req.TimeRange.Start
		end := req.TimeRange.End
		if !start.IsZero() && !end.IsZero() && start.After(end) {
			return normalizedUnifiedRunQuery{}, fmt.Errorf("time_range.start must be <= time_range.end")
		}
		tr = &UnifiedQueryTimeRange{Start: start, End: end}
	}
	return normalizedUnifiedRunQuery{
		RunID:      strings.TrimSpace(req.RunID),
		TeamID:     strings.TrimSpace(req.TeamID),
		WorkflowID: strings.TrimSpace(req.WorkflowID),
		TaskID:     strings.TrimSpace(req.TaskID),
		Status:     status,
		TimeRange:  tr,
		PageSize:   pageSize,
		SortField:  sortField,
		SortOrder:  sortOrder,
		Cursor:     strings.TrimSpace(req.Cursor),
	}, nil
}

func unifiedRunQueryHash(q normalizedUnifiedRunQuery) string {
	start := int64(0)
	end := int64(0)
	if q.TimeRange != nil {
		if !q.TimeRange.Start.IsZero() {
			start = q.TimeRange.Start.UnixNano()
		}
		if !q.TimeRange.End.IsZero() {
			end = q.TimeRange.End.UnixNano()
		}
	}
	raw := strings.Join([]string{
		q.RunID,
		q.TeamID,
		q.WorkflowID,
		q.TaskID,
		q.Status,
		fmt.Sprintf("%d", start),
		fmt.Sprintf("%d", end),
		q.SortField,
		q.SortOrder,
		fmt.Sprintf("%d", q.PageSize),
	}, "|")
	sum := sha1.Sum([]byte(raw))
	return hex.EncodeToString(sum[:])
}

func encodeUnifiedRunCursor(c unifiedRunQueryCursor) (string, error) {
	raw, err := json.Marshal(c)
	if err != nil {
		return "", fmt.Errorf("encode query cursor: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(raw), nil
}

func decodeUnifiedRunCursor(cursor, expectedHash string) (int, error) {
	if strings.TrimSpace(cursor) == "" {
		return 0, nil
	}
	raw, err := base64.RawURLEncoding.DecodeString(strings.TrimSpace(cursor))
	if err != nil {
		return 0, fmt.Errorf("invalid query cursor")
	}
	var decoded unifiedRunQueryCursor
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return 0, fmt.Errorf("invalid query cursor")
	}
	if decoded.Offset < 0 || strings.TrimSpace(decoded.QueryHash) == "" {
		return 0, fmt.Errorf("invalid query cursor")
	}
	if strings.TrimSpace(decoded.QueryHash) != strings.TrimSpace(expectedHash) {
		return 0, fmt.Errorf("invalid query cursor")
	}
	return decoded.Offset, nil
}

func matchesUnifiedRunQuery(rec RunRecord, q normalizedUnifiedRunQuery) bool {
	if q.RunID != "" && strings.TrimSpace(rec.RunID) != q.RunID {
		return false
	}
	if q.TeamID != "" && strings.TrimSpace(rec.TeamID) != q.TeamID {
		return false
	}
	if q.WorkflowID != "" && strings.TrimSpace(rec.WorkflowID) != q.WorkflowID {
		return false
	}
	if q.TaskID != "" && strings.TrimSpace(rec.TaskID) != q.TaskID {
		return false
	}
	if q.Status != "" && strings.ToLower(strings.TrimSpace(rec.Status)) != q.Status {
		return false
	}
	if q.TimeRange != nil {
		if !q.TimeRange.Start.IsZero() {
			if rec.Time.IsZero() || rec.Time.Before(q.TimeRange.Start) {
				return false
			}
		}
		if !q.TimeRange.End.IsZero() {
			if rec.Time.IsZero() || rec.Time.After(q.TimeRange.End) {
				return false
			}
		}
	}
	return true
}

func sortUnifiedRunQuery(items []RunRecord, order string) {
	desc := strings.TrimSpace(strings.ToLower(order)) != "asc"
	sort.SliceStable(items, func(i, j int) bool {
		left := items[i]
		right := items[j]
		if left.Time.Equal(right.Time) {
			if strings.TrimSpace(left.RunID) == strings.TrimSpace(right.RunID) {
				return strings.TrimSpace(left.Status) < strings.TrimSpace(right.Status)
			}
			return strings.TrimSpace(left.RunID) < strings.TrimSpace(right.RunID)
		}
		if desc {
			return left.Time.After(right.Time)
		}
		return left.Time.Before(right.Time)
	})
}

func normalizeMailboxQuery(req MailboxQueryRequest) (normalizedMailboxQuery, error) {
	pageSize := DefaultMailboxQueryPageSize
	if req.PageSize != nil {
		if *req.PageSize <= 0 || *req.PageSize > MaxMailboxQueryPageSize {
			return normalizedMailboxQuery{}, fmt.Errorf("page_size must be within [1,%d]", MaxMailboxQueryPageSize)
		}
		pageSize = *req.PageSize
	}
	sortField := strings.ToLower(strings.TrimSpace(req.Sort.Field))
	if sortField == "" {
		sortField = "time"
	}
	if sortField != "time" {
		return normalizedMailboxQuery{}, fmt.Errorf("unsupported sort.field %q", req.Sort.Field)
	}
	sortOrder := strings.ToLower(strings.TrimSpace(req.Sort.Order))
	if sortOrder == "" {
		sortOrder = "desc"
	}
	if sortOrder != "asc" && sortOrder != "desc" {
		return normalizedMailboxQuery{}, fmt.Errorf("unsupported sort.order %q", req.Sort.Order)
	}
	kind := strings.ToLower(strings.TrimSpace(req.Kind))
	if kind != "" {
		switch kind {
		case "command", "event", "result":
		default:
			return normalizedMailboxQuery{}, fmt.Errorf("unsupported kind filter %q", req.Kind)
		}
	}
	state := strings.ToLower(strings.TrimSpace(req.State))
	if state != "" {
		switch state {
		case "queued", "in_flight", "acked", "nacked", "dead_letter", "expired":
		default:
			return normalizedMailboxQuery{}, fmt.Errorf("unsupported state filter %q", req.State)
		}
	}
	var tr *MailboxQueryTimeRange
	if req.TimeRange != nil {
		start := req.TimeRange.Start
		end := req.TimeRange.End
		if !start.IsZero() && !end.IsZero() && start.After(end) {
			return normalizedMailboxQuery{}, fmt.Errorf("time_range.start must be <= time_range.end")
		}
		tr = &MailboxQueryTimeRange{Start: start, End: end}
	}
	return normalizedMailboxQuery{
		MessageID:      strings.TrimSpace(req.MessageID),
		IdempotencyKey: strings.TrimSpace(req.IdempotencyKey),
		CorrelationID:  strings.TrimSpace(req.CorrelationID),
		Kind:           kind,
		State:          state,
		RunID:          strings.TrimSpace(req.RunID),
		TaskID:         strings.TrimSpace(req.TaskID),
		WorkflowID:     strings.TrimSpace(req.WorkflowID),
		TeamID:         strings.TrimSpace(req.TeamID),
		TimeRange:      tr,
		PageSize:       pageSize,
		SortField:      sortField,
		SortOrder:      sortOrder,
		Cursor:         strings.TrimSpace(req.Cursor),
	}, nil
}

func mailboxQueryHash(q normalizedMailboxQuery) string {
	start := int64(0)
	end := int64(0)
	if q.TimeRange != nil {
		if !q.TimeRange.Start.IsZero() {
			start = q.TimeRange.Start.UnixNano()
		}
		if !q.TimeRange.End.IsZero() {
			end = q.TimeRange.End.UnixNano()
		}
	}
	raw := strings.Join([]string{
		q.MessageID,
		q.IdempotencyKey,
		q.CorrelationID,
		q.Kind,
		q.State,
		q.RunID,
		q.TaskID,
		q.WorkflowID,
		q.TeamID,
		fmt.Sprintf("%d", start),
		fmt.Sprintf("%d", end),
		q.SortField,
		q.SortOrder,
		fmt.Sprintf("%d", q.PageSize),
	}, "|")
	sum := sha1.Sum([]byte(raw))
	return hex.EncodeToString(sum[:])
}

func encodeMailboxCursor(c mailboxQueryCursor) (string, error) {
	raw, err := json.Marshal(c)
	if err != nil {
		return "", fmt.Errorf("encode query cursor: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(raw), nil
}

func decodeMailboxCursor(cursor, expectedHash string) (int, error) {
	if strings.TrimSpace(cursor) == "" {
		return 0, nil
	}
	raw, err := base64.RawURLEncoding.DecodeString(strings.TrimSpace(cursor))
	if err != nil {
		return 0, fmt.Errorf("invalid query cursor")
	}
	var decoded mailboxQueryCursor
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return 0, fmt.Errorf("invalid query cursor")
	}
	if decoded.Offset < 0 || strings.TrimSpace(decoded.QueryHash) == "" {
		return 0, fmt.Errorf("invalid query cursor")
	}
	if strings.TrimSpace(decoded.QueryHash) != strings.TrimSpace(expectedHash) {
		return 0, fmt.Errorf("invalid query cursor")
	}
	return decoded.Offset, nil
}

func matchesMailboxQuery(rec MailboxRecord, q normalizedMailboxQuery) bool {
	if q.MessageID != "" && strings.TrimSpace(rec.MessageID) != q.MessageID {
		return false
	}
	if q.IdempotencyKey != "" && strings.TrimSpace(rec.IdempotencyKey) != q.IdempotencyKey {
		return false
	}
	if q.CorrelationID != "" && strings.TrimSpace(rec.CorrelationID) != q.CorrelationID {
		return false
	}
	if q.Kind != "" && strings.ToLower(strings.TrimSpace(rec.Kind)) != q.Kind {
		return false
	}
	if q.State != "" && strings.ToLower(strings.TrimSpace(rec.State)) != q.State {
		return false
	}
	if q.RunID != "" && strings.TrimSpace(rec.RunID) != q.RunID {
		return false
	}
	if q.TaskID != "" && strings.TrimSpace(rec.TaskID) != q.TaskID {
		return false
	}
	if q.WorkflowID != "" && strings.TrimSpace(rec.WorkflowID) != q.WorkflowID {
		return false
	}
	if q.TeamID != "" && strings.TrimSpace(rec.TeamID) != q.TeamID {
		return false
	}
	if q.TimeRange != nil {
		if !q.TimeRange.Start.IsZero() {
			if rec.Time.IsZero() || rec.Time.Before(q.TimeRange.Start) {
				return false
			}
		}
		if !q.TimeRange.End.IsZero() {
			if rec.Time.IsZero() || rec.Time.After(q.TimeRange.End) {
				return false
			}
		}
	}
	return true
}

func sortMailboxQuery(items []MailboxRecord, order string) {
	desc := strings.TrimSpace(strings.ToLower(order)) != "asc"
	sort.SliceStable(items, func(i, j int) bool {
		left := items[i]
		right := items[j]
		if left.Time.Equal(right.Time) {
			return strings.TrimSpace(left.MessageID) < strings.TrimSpace(right.MessageID)
		}
		if desc {
			return left.Time.After(right.Time)
		}
		return left.Time.Before(right.Time)
	})
}

func payloadDigest(payload map[string]any) string {
	if len(payload) == 0 {
		return ""
	}
	raw, err := json.Marshal(normalizePayloadForKey(payload))
	if err != nil {
		return "marshal_error"
	}
	sum := sha1.Sum(raw)
	return hex.EncodeToString(sum[:])
}

func normalizePayloadForKey(in map[string]any) map[string]any {
	out := make(map[string]any, len(in))
	for k, v := range in {
		lk := strings.ToLower(strings.TrimSpace(k))
		if lk == "latency_ms" || lk == "time" || lk == "timestamp" {
			continue
		}
		switch tv := v.(type) {
		case map[string]any:
			out[k] = normalizePayloadForKey(tv)
		case []any:
			out[k] = normalizeSliceForKey(tv)
		default:
			out[k] = v
		}
	}
	return out
}

func normalizeSliceForKey(in []any) []any {
	out := make([]any, 0, len(in))
	for _, v := range in {
		switch tv := v.(type) {
		case map[string]any:
			out = append(out, normalizePayloadForKey(tv))
		case []any:
			out = append(out, normalizeSliceForKey(tv))
		default:
			out = append(out, v)
		}
	}
	return out
}

func (d *Store) rebuildRunKeys() {
	d.runKeys = make(map[string]int, len(d.runs))
	for i := range d.runs {
		d.runKeys[RunIdempotencyKey(d.runs[i])] = i
	}
}

func (d *Store) rebuildMailboxKeys() {
	d.mbxKeys = make(map[string]int, len(d.mailbox))
	for i := range d.mailbox {
		d.mbxKeys[MailboxIdempotencyKey(d.mailbox[i])] = i
	}
}

func (d *Store) timelinePhasesForRun(runID string) map[string]TimelinePhaseAggregate {
	runID = strings.TrimSpace(runID)
	if runID == "" {
		return nil
	}
	state := d.timelineStates[runID]
	if state == nil || len(state.phases) == 0 {
		return nil
	}
	out := make(map[string]TimelinePhaseAggregate, len(state.phases))
	for phase, agg := range state.phases {
		out[phase] = agg
	}
	return out
}

func (d *Store) selectTrendRuns(query TimelineTrendQuery) ([]RunRecord, time.Time, time.Time) {
	if len(d.runs) == 0 {
		return nil, time.Time{}, time.Time{}
	}
	mode := query.Mode
	if mode == "" {
		mode = TimelineTrendModeLastNRuns
	}
	switch mode {
	case TimelineTrendModeTimeWindow:
		window := query.TimeWindow
		if window <= 0 {
			window = d.trendConfig.TimeWindow
		}
		if window <= 0 {
			return nil, time.Time{}, time.Time{}
		}
		end := d.runs[len(d.runs)-1].Time
		if end.IsZero() {
			end = time.Now()
		}
		start := end.Add(-window)
		selected := make([]RunRecord, 0, len(d.runs))
		for i := range d.runs {
			ts := d.runs[i].Time
			if ts.IsZero() {
				continue
			}
			if ts.Before(start) || ts.After(end) {
				continue
			}
			selected = append(selected, d.runs[i])
		}
		return selected, start, end
	default:
		n := query.LastNRuns
		if n <= 0 {
			n = d.trendConfig.LastNRuns
		}
		selected := tailCopy(d.runs, n)
		if len(selected) == 0 {
			return nil, time.Time{}, time.Time{}
		}
		start := selected[0].Time
		end := selected[len(selected)-1].Time
		return selected, start, end
	}
}

func (d *Store) selectCA2Runs(query CA2ExternalTrendQuery) ([]RunRecord, time.Time, time.Time) {
	if len(d.runs) == 0 {
		return nil, time.Time{}, time.Time{}
	}
	window := query.Window
	if window <= 0 {
		window = d.ca2TrendConfig.Window
	}
	if window <= 0 {
		return nil, time.Time{}, time.Time{}
	}
	end := d.runs[len(d.runs)-1].Time
	if end.IsZero() {
		end = time.Now()
	}
	start := end.Add(-window)
	selected := make([]RunRecord, 0, len(d.runs))
	for i := range d.runs {
		rec := d.runs[i]
		ts := rec.Time
		if ts.IsZero() {
			continue
		}
		if ts.Before(start) || ts.After(end) {
			continue
		}
		if strings.TrimSpace(rec.Stage2Provider) == "" {
			continue
		}
		selected = append(selected, rec)
	}
	return selected, start, end
}

func trendBucketKey(phase, status string) string {
	return strings.TrimSpace(phase) + "|" + strings.ToLower(strings.TrimSpace(status))
}

func splitTrendBucketKey(key string) (string, string) {
	parts := strings.SplitN(key, "|", 2)
	if len(parts) != 2 {
		return key, ""
	}
	return parts[0], parts[1]
}

func isCA2ExternalError(rec RunRecord) bool {
	if strings.TrimSpace(rec.Stage2ErrorLayer) != "" {
		return true
	}
	code := strings.ToLower(strings.TrimSpace(rec.Stage2ReasonCode))
	return code != "" && code != "ok"
}

func (d *Store) pruneTimelineStates() {
	if len(d.timelineStates) == 0 {
		return
	}
	keep := make(map[string]struct{}, len(d.runs))
	for i := range d.runs {
		runID := strings.TrimSpace(d.runs[i].RunID)
		if runID == "" {
			continue
		}
		keep[runID] = struct{}{}
	}
	for runID := range d.timelineStates {
		if _, ok := keep[runID]; ok {
			continue
		}
		delete(d.timelineStates, runID)
	}
}

func percentileP95(samples []int64) int64 {
	if len(samples) == 0 {
		return 0
	}
	if len(samples) == 1 {
		return samples[0]
	}
	cp := make([]int64, len(samples))
	copy(cp, samples)
	sort.Slice(cp, func(i, j int) bool { return cp[i] < cp[j] })
	idx := int(math.Ceil(0.95*float64(len(cp)))) - 1
	if idx < 0 {
		idx = 0
	}
	if idx >= len(cp) {
		idx = len(cp) - 1
	}
	return cp[idx]
}

func (d *Store) rebuildSkillKeys() {
	d.sklKeys = make(map[string]int, len(d.skills))
	for i := range d.skills {
		d.sklKeys[SkillIdempotencyKey(d.skills[i])] = i
	}
}

func trimTail[T any](src []T, n int) []T {
	if n <= 0 || len(src) <= n {
		return src
	}
	dst := make([]T, n)
	copy(dst, src[len(src)-n:])
	return dst
}

func tailCopy[T any](src []T, n int) []T {
	if n <= 0 || n > len(src) {
		n = len(src)
	}
	dst := make([]T, n)
	copy(dst, src[len(src)-n:])
	return dst
}

func cloneIntMap(in map[string]int) map[string]int {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]int, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}

func cloneAnyMap(in map[string]any) map[string]any {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]any, len(in))
	for key, value := range in {
		if child, ok := value.(map[string]any); ok {
			out[key] = cloneAnyMap(child)
			continue
		}
		out[key] = value
	}
	return out
}

func cloneStringSlice(in []string) []string {
	if len(in) == 0 {
		return nil
	}
	out := make([]string, 0, len(in))
	for i := range in {
		item := strings.TrimSpace(in[i])
		if item == "" {
			continue
		}
		out = append(out, item)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func cloneRuntimePolicyDecisionPath(in []RuntimePolicyDecisionPathEntry) []RuntimePolicyDecisionPathEntry {
	if len(in) == 0 {
		return nil
	}
	out := make([]RuntimePolicyDecisionPathEntry, 0, len(in))
	for i := range in {
		item := in[i]
		item.Stage = strings.ToLower(strings.TrimSpace(item.Stage))
		item.Code = strings.TrimSpace(item.Code)
		item.Source = strings.ToLower(strings.TrimSpace(item.Source))
		item.Decision = strings.ToLower(strings.TrimSpace(item.Decision))
		if item.Stage == "" {
			continue
		}
		out = append(out, item)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}
